package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/patryk-plonka/devops-ai-agents/internal/azure"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

const first = "11111111-1111-1111-1111-111111111111"
const second = "22222222-2222-2222-2222-222222222222"
const tenant = "33333333-3333-3333-3333-333333333333"

func cached(id, name string) azure.Account {
	a := azure.Account{ID: id, Name: name, TenantID: tenant, Environment: "AzureCloud"}
	a.User.Type = "user"
	return a
}

func testApp(t *testing.T) (App, *bytes.Buffer, *bytes.Buffer, *int) {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module "+runstore.Module+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	out, diagnostics := new(bytes.Buffer), new(bytes.Buffer)
	calls := new(int)
	return App{Directory: root, Out: out, Err: diagnostics, Discover: func(context.Context) ([]azure.Account, error) {
		*calls++
		return []azure.Account{cached(first, "Client")}, nil
	}, Validate: func(context.Context, azure.Account) error { *calls++; return nil },
		Inventory:      func(context.Context, string, func(health.Page) error) error { return nil },
		ResourceHealth: func(context.Context, string, func(health.Page) error) error { return nil }}, out, diagnostics, calls
}

func snapshot(t *testing.T, root string) health.Run {
	t.Helper()
	paths, err := filepath.Glob(filepath.Join(root, "runs", "*", "run.json"))
	if err != nil || len(paths) != 1 {
		t.Fatalf("snapshots %v (%v)", paths, err)
	}
	b, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	var r health.Run
	if err := json.Unmarshal(b, &r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestHelpAndUsageHaveNoSideEffects(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"health", "--help"}, {"health", "check", "--help"}, {}, {"health", "check"}, {"health", "check", "--subscription", "Client", "extra"}, {"health", "check", "--unknown", "Bearer canary"}} {
		a, out, diag, calls := testApp(t)
		code := a.Run(context.Background(), args)
		want := 2
		if len(args) > 0 && args[len(args)-1] == "--help" {
			want = 0
		}
		if code != want || *calls != 0 {
			t.Fatalf("%v: code %d, calls %d", args, code, *calls)
		}
		if _, err := os.Stat(filepath.Join(a.Directory, "runs")); !os.IsNotExist(err) {
			t.Fatal("usage created storage")
		}
		if strings.Contains(out.String()+diag.String(), "canary") {
			t.Fatal("untrusted flag value leaked")
		}
	}
}

func TestCollectedScopeRemainsExplicitlyIncomplete(t *testing.T) {
	a, out, diag, calls := testApp(t)
	a.Validate = func(_ context.Context, account azure.Account) error {
		*calls++
		if account.ID != first {
			t.Fatal("wrong scope")
		}
		if snapshot(t, a.Directory).Outcome != "in_progress" {
			t.Fatal("Azure accessed before initial snapshot")
		}
		return nil
	}
	code := a.Run(context.Background(), []string{"health", "check", "--subscription", "cLiEnT"})
	r := snapshot(t, a.Directory)
	if code != 0 || out.Len() == 0 || *calls != 2 || r.Outcome != "completed" || r.Coverage != "incomplete" || r.Scope.SubscriptionID != first || r.Revision != 6 {
		t.Fatalf("code %d run %+v output %s", code, r, diag)
	}
	if !strings.Contains(diag.String(), "overall coverage: incomplete") || !strings.Contains(diag.String(), "Report saved:") {
		t.Fatal("misleading preliminary result")
	}
	for name, source := range r.Sources {
		if name == "inventory" || name == "resource_health" {
			if source.Status != "completed" {
				t.Fatal("collector incomplete")
			}
			continue
		}
		if source.Status != "not_checked" {
			t.Fatal("uncollected source marked checked")
		}
	}
}

func TestScopeFailureCategoriesAndPrivacy(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		discoverErr, validateErr error
		requested                string
		code                     int
		category                 string
	}{
		{"missing", nil, nil, "Absent", 2, "subscription_not_in_session"},
		{"expired", &azure.Failure{Category: "login_required", Message: "Run az login externally.", Code: 1}, nil, "Client", 1, "login_required"},
		{"denied", nil, &azure.Failure{Category: "subscription_access_denied", Message: "Live access denied.", Code: 1}, "Client", 1, "subscription_access_denied"},
		{"raw external error", nil, errors.New("Bearer secret-canary"), "Client", 1, "execution_failed"},
		{"secret input", nil, nil, "Bearer secret-canary", 2, "subscription_not_in_session"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, out, diag, _ := testApp(t)
			if tc.discoverErr != nil {
				a.Discover = func(context.Context) ([]azure.Account, error) { return nil, tc.discoverErr }
			}
			a.Validate = func(context.Context, azure.Account) error { return tc.validateErr }
			code := a.Run(context.Background(), []string{"health", "check", "--subscription", tc.requested})
			r := snapshot(t, a.Directory)
			if code != tc.code || r.Outcome != "failed" || r.Diagnostics[0] != tc.category || out.Len() == 0 {
				t.Fatalf("code %d run %+v", code, r)
			}
			b, _ := json.Marshal(r)
			if strings.Contains(string(b)+diag.String(), "secret-canary") {
				t.Fatal("secret persisted or printed")
			}
			if tc.category == "subscription_not_in_session" && r.Scope != nil {
				t.Fatal("unresolved scope claimed resolved")
			}
		})
	}
}

func TestAmbiguityRequiresTerminalAndExplicitSelection(t *testing.T) {
	for _, tc := range []struct {
		name, input              string
		terminal, noninteractive bool
		code                     int
		id                       string
	}{
		{"pipe", "1\n", false, false, 2, ""}, {"noninteractive", "1\n", true, true, 2, ""},
		{"select second", "2\n", true, false, 0, second}, {"EOF", "1", true, false, 2, ""},
		{"blank", "\n", true, false, 2, ""}, {"invalid", "99\n", true, false, 2, ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, _, diag, _ := testApp(t)
			a.Terminal = tc.terminal
			a.In = strings.NewReader(tc.input)
			a.Discover = func(context.Context) ([]azure.Account, error) {
				return []azure.Account{cached(second, "Client"), cached(first, "Client"), cached(first, "Client")}, nil
			}
			var validated string
			a.Validate = func(_ context.Context, account azure.Account) error { validated = account.ID; return nil }
			args := []string{"health", "check", "--subscription", "Client"}
			if tc.noninteractive {
				args = append(args, "--non-interactive")
			}
			if code := a.Run(context.Background(), args); code != tc.code || validated != tc.id {
				t.Fatalf("code %d validated %s: %s", code, validated, diag)
			}
			if tc.id != "" && strings.Index(diag.String(), first) > strings.Index(diag.String(), second) {
				t.Fatal("unstable selection order")
			}
		})
	}
}

func TestStorageFailurePreventsAzure(t *testing.T) {
	a, _, diag, calls := testApp(t)
	if err := os.WriteFile(filepath.Join(a.Directory, "runs"), []byte("obstruction"), 0600); err != nil {
		t.Fatal(err)
	}
	if code := a.Run(context.Background(), []string{"health", "check", "--subscription", "Client"}); code != 1 || *calls != 0 || strings.Contains(diag.String(), "record saved") {
		t.Fatalf("storage failure: code %d calls %d", code, *calls)
	}
}

type failingFinalStore struct{ Storage }

func (s failingFinalStore) Save(*health.Run) error { return errors.New("secret-canary") }
func TestFailedFinalWriteDoesNotClaimSavedOutput(t *testing.T) {
	a, out, diag, _ := testApp(t)
	a.NewStore = func(root string, r *health.Run) (Storage, error) {
		s, err := runstore.New(root, r)
		return failingFinalStore{s}, err
	}
	code := a.Run(context.Background(), []string{"health", "check", "--subscription", "Client"})
	if code != 1 || out.Len() != 0 || strings.Contains(diag.String(), "record saved") || strings.Contains(diag.String(), "secret-canary") {
		t.Fatalf("false saved message: %s", diag)
	}
	if snapshot(t, a.Directory).Revision != 1 {
		t.Fatal("checkpoint lost")
	}
}

func TestMissingRepositoryPreventsAzure(t *testing.T) {
	a, _, _, calls := testApp(t)
	a.Directory = t.TempDir()
	if code := a.Run(context.Background(), []string{"health", "check", "--subscription", "Client"}); code != 2 || *calls != 0 {
		t.Fatal("missing repository accepted")
	}
}

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/patryk-plonka/devops-ai-agents/internal/azure"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

func TestDraftUsageIsOffline(t *testing.T) {
	for _, tc := range []struct {
		args []string
		code int
	}{
		{[]string{"--help"}, 0},
		{nil, 2},
		{[]string{"--report", "report.md", "--finding", "finding-" + strings.Repeat("a", 64)}, 2},
		{[]string{"--report", "report.md", "--type", "unsupported", "--finding", "finding-" + strings.Repeat("a", 64)}, 2},
		{[]string{"--report", "report.md", "--type", "ticket", "--finding", "Bearer private-canary"}, 2},
		{[]string{"--report", "report.md", "--type", "ticket"}, 2},
	} {
		app, out, diag, _ := testApp(t)
		prohibitDraftCollection(t, &app)
		app.Directory = filepath.Join(app.Directory, "nonexistent")
		if got := app.Run(context.Background(), append([]string{"health", "draft"}, tc.args...)); got != tc.code {
			t.Errorf("got exit %d, want %d\n%s", got, tc.code, diag)
		}
		if tc.code == 0 && !strings.Contains(out.String(), "health draft --report") {
			t.Error("draft help missing")
		}
		if strings.Contains(out.String()+diag.String(), "private-canary") {
			t.Error("flag value leaked")
		}
	}
}

func TestDraftCLISelectionAndReuse(t *testing.T) {
	app, out, diag, run, path := draftCLIExample(t)
	jsonPath := filepath.Join(filepath.Dir(path), "run.json")
	originalReport, originalJSON := readDraftCLIFile(t, path), readDraftCLIFile(t, jsonPath)
	// Use the IDs printed next to the two Fired alerts in the published S-04 pair.
	_, alerts, ok := strings.Cut(string(originalReport), "\n## Azure Monitor\n")
	if !ok {
		t.Fatal("S-04 fixture is missing the Monitor section")
	}
	monitorSection, _, _ := strings.Cut(alerts, "\n## ")
	ids := reportedFindingIDs(monitorSection)
	if len(ids) != 2 {
		t.Fatalf("S-04 fixture Monitor finding IDs: %v", ids)
	}
	first, second := ids[1], ids[0]
	dir := filepath.Join(filepath.Dir(path), "drafts")
	ticket1, ticket2 := filepath.Join(dir, "ticket-"+first+".md"), filepath.Join(dir, "ticket-"+second+".md")
	relative, err := filepath.Rel(app.Directory, path)
	if err != nil {
		t.Fatal(err)
	}

	for _, id := range []string{"finding-" + strings.Repeat("a", 64), "finding-" + strings.Repeat("A", 64)} {
		if code := app.Run(context.Background(), draftArgs(relative, "ticket", first, id)); code != 2 || out.Len() != 0 {
			t.Fatalf("invalid mixed selection returned %d, stdout %q", code, out)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Fatal("invalid mixed selection created the drafts directory")
		}
	}
	if code := app.Run(context.Background(), draftArgs(relative, "ticket", first)); code != 0 {
		t.Fatalf("first draft exit %d\n%s", code, diag)
	}
	if !strings.Contains(string(readDraftCLIFile(t, ticket1)), "current Resource Health evidence is missing") {
		t.Fatal("Fired finding lost its missing Resource Health check")
	}
	edited := []byte("operator-edited working copy\n")
	if err := os.WriteFile(ticket1, edited, 0600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	diag.Reset()
	if code := app.Run(context.Background(), draftArgs(path, "ticket", first, second, first)); code != 0 {
		t.Fatalf("mixed request exit %d\n%s", code, diag)
	}
	if got := strings.Split(strings.TrimSpace(out.String()), "\n"); !reflect.DeepEqual(got, []string{ticket1, ticket2}) {
		t.Errorf("incorrect delivery order or duplication: %v", got)
	}
	if !strings.Contains(diag.String(), "Draft already exists: "+ticket1) || !strings.Contains(diag.String(), "Draft created: "+ticket2) {
		t.Errorf("missing status for reuse and creation:\n%s", diag)
	}

	out.Reset()
	if code := app.Run(context.Background(), draftArgs(path, "recommendation", first)); code != 0 {
		t.Fatalf("recommendation exit %d\n%s", code, diag)
	}
	recommendation := filepath.Join(dir, "recommendation-"+first+".md")
	if strings.TrimSpace(out.String()) != recommendation || !bytes.HasPrefix(readDraftCLIFile(t, recommendation), []byte("# Recommendation:")) {
		t.Fatal("recommendation was not delivered independently")
	}
	if !bytes.Equal(readDraftCLIFile(t, ticket1), edited) {
		t.Fatal("ticket working copy changed")
	}
	body := string(readDraftCLIFile(t, recommendation))
	trace := fmt.Sprintf("Run ID: `%s`\n- Revision: %d\n- Finding ID: `%s`", run.ID, run.Revision, first)
	if !strings.Contains(body, trace) {
		t.Fatal("draft did not carry the S-04 run, revision and finding")
	}
	if !bytes.Equal(readDraftCLIFile(t, path), originalReport) || !bytes.Equal(readDraftCLIFile(t, jsonPath), originalJSON) {
		t.Fatal("drafting changed the S-04 report or snapshot")
	}
}

func TestDraftCLIFailureAndInterruptionRetainPaths(t *testing.T) {
	for _, scenario := range []string{"stdout fails", "cancel after first path", "second destination obstructed", "directory obstructed", "mismatched revision"} {
		t.Run(scenario, func(t *testing.T) {
			app, out, diag, run, reportPath := draftCLIExample(t)
			first, second := run.Findings[0].ID, run.Findings[1].ID
			dir := filepath.Join(filepath.Dir(reportPath), "drafts")
			path1, path2 := filepath.Join(dir, "ticket-"+first+".md"), filepath.Join(dir, "ticket-"+second+".md")
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			wantCode, retained := 1, true
			switch scenario {
			case "stdout fails":
				app.Out = draftOutputFunc(func([]byte) (int, error) { return 0, io.ErrClosedPipe })
			case "cancel after first path":
				wantCode = 130
				app.Out = draftOutputFunc(func(p []byte) (int, error) {
					cancel()
					return out.Write(p)
				})
			case "second destination obstructed":
				if err := os.MkdirAll(path2, 0700); err != nil {
					t.Fatal(err)
				}
			case "directory obstructed":
				retained = false
				if err := os.WriteFile(dir, []byte("unrelated data"), 0600); err != nil {
					t.Fatal(err)
				}
			case "mismatched revision":
				wantCode, retained = 2, false
				snapshot := filepath.Join(filepath.Dir(reportPath), "run.json")
				run.Revision++
				data, err := json.Marshal(run)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(snapshot, data, 0600); err != nil {
					t.Fatal(err)
				}
			}

			if code := app.Run(ctx, draftArgs(reportPath, "ticket", first, second)); code != wantCode {
				t.Fatalf("exit %d, expected %d\n%s", code, wantCode, diag)
			}
			if !retained {
				if out.Len() != 0 || strings.Contains(diag.String(), "Draft created:") {
					t.Fatal("reported a draft during a pre-publication failure")
				}
				return
			}
			if !strings.Contains(diag.String(), "Draft created: "+path1) || !strings.Contains(diag.String(), "Pending selections:\n"+second) {
				t.Fatalf("expected retained path and pending selection:\n%s", diag)
			}
			body := readDraftCLIFile(t, path1)
			if !bytes.Contains(body, []byte(first)) || !bytes.HasSuffix(body, []byte("\n")) {
				t.Fatal("published draft is incomplete")
			}
			if scenario == "second destination obstructed" {
				if err := os.Remove(path2); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := os.Stat(path2); !os.IsNotExist(err) {
				t.Fatal("pending draft exists")
			}

			app.Out = out
			out.Reset()
			diag.Reset()
			if code := app.Run(context.Background(), draftArgs(reportPath, "ticket", first, second)); code != 0 {
				t.Fatalf("retry exit %d\n%s", code, diag)
			}
			if !bytes.Equal(readDraftCLIFile(t, path1), body) || !strings.Contains(diag.String(), "Draft already exists: "+path1) {
				t.Fatal("retry replaced or misreported the retained draft")
			}
			if !strings.Contains(diag.String(), "Draft created: "+path2) || !bytes.Contains(readDraftCLIFile(t, path2), []byte(second)) {
				t.Fatal("retry did not complete the pending selection")
			}
		})
	}
}

func draftCLIExample(t *testing.T) (App, *bytes.Buffer, *bytes.Buffer, health.Run, string) {
	t.Helper()
	app, out, diag, _ := testApp(t)
	prohibitDraftCollection(t, &app)
	fixture := filepath.Join("..", "report", "testdata", "assessment")
	data := readDraftCLIFile(t, filepath.Join(fixture, "run.json"))
	var run health.Run
	if err := json.Unmarshal(data, &run); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(app.Directory, "runs", run.ID)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "report.md")
	for name, body := range map[string][]byte{
		filepath.Join(dir, "run.json"): data,
		path:                           readDraftCLIFile(t, filepath.Join(fixture, "report.md")),
	} {
		if err := os.WriteFile(name, body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return app, out, diag, run, path
}

func prohibitDraftCollection(t *testing.T, app *App) {
	t.Helper()
	app.NewStore = func(string, *health.Run) (Storage, error) {
		t.Fatal("draft attempted to allocate a run")
		return nil, nil
	}
	app.Discover = func(context.Context) ([]azure.Account, error) {
		t.Fatal("draft attempted Azure discovery")
		return nil, nil
	}
	app.Validate = func(context.Context, azure.Account) error {
		t.Fatal("draft attempted Azure validation")
		return nil
	}
	collect := func(context.Context, string, func(health.Page) error) error {
		t.Fatal("draft attempted collection")
		return nil
	}
	app.Inventory, app.ResourceHealth, app.ServiceHealth, app.AzureMonitor, app.Advisor = collect, collect, collect, collect, collect
}

func draftArgs(path, kind string, ids ...string) []string {
	args := []string{"health", "draft", "--report", path, "--type", kind}
	for _, id := range ids {
		args = append(args, "--finding", id)
	}
	return args
}

func readDraftCLIFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

type draftOutputFunc func([]byte) (int, error)

func (f draftOutputFunc) Write(p []byte) (int, error) { return f(p) }

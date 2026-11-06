package cli

import (
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

const cliVM = "/subscriptions/" + first + "/resourceGroups/g/providers/Microsoft.Compute/virtualMachines/vm"

func TestCLICollectionOutcomesAndEvidence(t *testing.T) {
	for _, tc := range []struct {
		name                        string
		inventoryFails, healthFails bool
		code                        int
		outcome                     string
	}{
		{"completed", false, false, 0, "completed"},
		{"inventory denied", true, false, 3, "partial"},
		{"health denied", false, true, 3, "partial"},
		{"both denied", true, true, 1, "failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, out, diag, _ := testApp(t)
			sequence := []string{}
			a.Inventory = func(_ context.Context, id string, emit func(health.Page) error) error {
				sequence = append(sequence, "inventory")
				if id != first {
					t.Fatal("wrong scope")
				}
				if tc.inventoryFails {
					return &azure.Failure{Category: "access_denied", Message: "raw-canary", Code: 1}
				}
				return emit(health.Page{Resources: []health.ResourceInput{{ID: cliVM, Name: "vm", Tags: map[string]*string{"password": stringPointer("secret-canary")}}}})
			}
			a.ResourceHealth = func(_ context.Context, id string, emit func(health.Page) error) error {
				sequence = append(sequence, "resource_health")
				if tc.healthFails {
					return errors.New("Bearer secret-canary")
				}
				state := "Unavailable"
				return emit(health.Page{Health: []health.HealthInput{{StatusID: cliVM + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current", PropertiesPresent: true, State: &state}}})
			}
			code := a.Run(context.Background(), []string{"health", "check", "--subscription", first})
			r := snapshot(t, a.Directory)
			if code != tc.code || r.Outcome != tc.outcome || out.Len() == 0 || r.Coverage != "incomplete" || len(sequence) != 2 {
				t.Fatalf("code %d run %+v diag %s", code, r, diag)
			}
			if !tc.healthFails && (len(r.Findings) != 1 || len(r.Findings[0].EvidenceIDs) != 1) {
				t.Fatal("finding lost")
			}
			b, _ := json.Marshal(r)
			if strings.Contains(string(b)+diag.String(), "canary") {
				t.Fatal("unfiltered artifact/diagnostic")
			}
			if !strings.Contains(diag.String(), "overall coverage: incomplete") || !strings.Contains(diag.String(), "Report saved:") {
				t.Fatal("interim behavior unclear")
			}
			reports, _ := filepath.Glob(filepath.Join(a.Directory, "runs", "*", "report.md"))
			if len(reports) != 1 {
				t.Fatal("missing report")
			}
		})
	}
}
func stringPointer(s string) *string { return &s }

type failAfterStore struct {
	Storage
	saves int
}

func (s *failAfterStore) Save(r *health.Run) error {
	s.saves++
	if s.saves == 2 {
		return errors.New("secret-canary")
	}
	return s.Storage.Save(r)
}
func TestCLIPageWriteFailureStopsCollectionAndIdentifiesCheckpoint(t *testing.T) {
	a, out, diag, _ := testApp(t)
	calls := 0
	a.NewStore = func(root string, r *health.Run) (Storage, error) {
		s, err := runstore.New(root, r)
		return &failAfterStore{Storage: s}, err
	}
	a.Inventory = func(_ context.Context, _ string, emit func(health.Page) error) error {
		calls++
		return emit(health.Page{Resources: []health.ResourceInput{{ID: cliVM}}})
	}
	a.ResourceHealth = func(context.Context, string, func(health.Page) error) error {
		t.Fatal("read after storage failure")
		return nil
	}
	code := a.Run(context.Background(), []string{"health", "check", "--subscription", first})
	if code != 1 || out.Len() != 0 || calls != 1 || strings.Contains(diag.String(), "canary") || !strings.Contains(diag.String(), "Last saved snapshot:") || !strings.Contains(diag.String(), "revision 2") {
		t.Fatalf("wrong failure %d %s", code, diag)
	}
	paths, _ := filepath.Glob(filepath.Join(a.Directory, "runs", "*", "run.json"))
	b, err := os.ReadFile(paths[0])
	if err != nil {
		t.Fatal(err)
	}
	var saved health.Run
	_ = json.Unmarshal(b, &saved)
	if saved.Revision != 2 || len(saved.Evidence) != 0 {
		t.Fatal("prior checkpoint overwritten")
	}
}

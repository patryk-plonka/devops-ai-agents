package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

const subscription = "11111111-1111-1111-1111-111111111111"
const vm = "/subscriptions/" + subscription + "/resourceGroups/g/providers/Microsoft.Compute/virtualMachines/vm"

type failure string

func (f failure) Error() string      { return "Bearer error-body-canary" }
func (f failure) Diagnostic() string { return string(f) }
func TestIndependentCollectionAndCommittedEvidence(t *testing.T) {
	for _, tc := range []struct {
		name, inventoryFailure, healthFailure string
		inventoryPage, healthPage             bool
		code                                  int
		outcome                               string
	}{
		{name: "both completed", inventoryPage: true, healthPage: true, code: 0, outcome: "completed"},
		{name: "inventory denied", inventoryFailure: "access_denied", healthPage: true, code: 3, outcome: "partial"},
		{name: "health denied", healthFailure: "access_denied", inventoryPage: true, code: 3, outcome: "partial"},
		{name: "later inventory timeout", inventoryFailure: "timeout", inventoryPage: true, healthPage: true, code: 3, outcome: "partial"},
		{name: "both fail", inventoryFailure: "login_required", healthFailure: "access_denied", code: 1, outcome: "failed"},
		{name: "partial useful health", inventoryFailure: "access_denied", healthFailure: "throttled", healthPage: true, code: 3, outcome: "partial"},
		{name: "empty completed collector remains useful", healthFailure: "unsupported", code: 3, outcome: "partial"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := health.NewRun("test", "test")
			r.Scope = &health.Scope{SubscriptionID: subscription}
			store, err := runstore.New(t.TempDir(), &r)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			order := []string{}
			read := func() health.Run {
				b, e := os.ReadFile(store.Path())
				if e != nil {
					t.Fatal(e)
				}
				var result health.Run
				if json.Unmarshal(b, &result) != nil {
					t.Fatal("bad checkpoint")
				}
				return result
			}
			makeCollector := func(source, reason string, page bool) health.Collector {
				return func(_ context.Context, id string, emit func(health.Page) error) error {
					order = append(order, source)
					if id != subscription {
						t.Fatal("wrong scope")
					}
					if page {
						p := health.Page{}
						if source == "inventory" {
							p.Resources = []health.ResourceInput{{ID: vm, Name: "vm"}}
						} else {
							state := "Unavailable"
							p.Health = []health.HealthInput{{StatusID: vm + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current", PropertiesPresent: true, State: &state}}
						}
						if err := emit(p); err != nil {
							return err
						}
						saved := read()
						if len(saved.Evidence) == 0 || saved.Sources[source].Pages != 1 || saved.Revision != r.Revision {
							t.Fatal("next page before committed evidence")
						}
					}
					if reason != "" {
						return failure(reason)
					}
					return nil
				}
			}
			code, err := health.Check(context.Background(), &r, []health.NamedCollector{{Name: "inventory", Collect: makeCollector("inventory", tc.inventoryFailure, tc.inventoryPage)}, {Name: "resource_health", Collect: makeCollector("resource_health", tc.healthFailure, tc.healthPage)}}, store.Save)
			if err != nil || code != tc.code || r.Outcome != tc.outcome || len(order) != 2 || order[0] != "inventory" || order[1] != "resource_health" {
				t.Fatalf("code %d err %v run %+v order %v", code, err, r, order)
			}
			saved := read()
			wantEvidence := 0
			if tc.inventoryPage {
				wantEvidence++
			}
			if tc.healthPage {
				wantEvidence++
			}
			if len(saved.Evidence) != wantEvidence || r.Coverage != "incomplete" {
				t.Fatal("evidence lost or invented coverage")
			}
			for _, name := range []string{"advisor", "service_health", "azure_monitor"} {
				s := saved.Sources[name]
				if s.Status != "not_checked" || s.Reason != "not_attempted" {
					t.Fatal("invented source coverage")
				}
			}
			if tc.inventoryFailure != "" && saved.Sources["inventory"].Reason != tc.inventoryFailure {
				t.Fatal("lost inventory failure")
			}
			if tc.healthFailure != "" && saved.Sources["resource_health"].Reason != tc.healthFailure {
				t.Fatal("lost health failure")
			}
		})
	}
}
func TestCheckpointFailureStopsBothCollectors(t *testing.T) {
	r := health.NewRun("test", "test")
	r.Scope = &health.Scope{SubscriptionID: subscription}
	store, err := runstore.New(t.TempDir(), &r)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	calls, saves := 0, 0
	collect := func(_ context.Context, _ string, emit func(health.Page) error) error {
		calls++
		return emit(health.Page{Resources: []health.ResourceInput{{ID: vm}}})
	}
	save := func(run *health.Run) error {
		saves++
		if saves == 2 {
			return errors.New("storage-canary")
		}
		return store.Save(run)
	}
	code, err := health.Check(context.Background(), &r, []health.NamedCollector{{Name: "inventory", Collect: collect}, {Name: "resource_health", Collect: collect}}, save)
	if code != 1 || err == nil || calls != 1 || r.Revision != 2 {
		t.Fatalf("code %d err %v calls %d revision %d", code, err, calls, r.Revision)
	}
	b, _ := os.ReadFile(store.Path())
	var saved health.Run
	_ = json.Unmarshal(b, &saved)
	if len(saved.Evidence) != 0 || saved.Revision != 2 {
		t.Fatal("failed checkpoint overwrote committed record")
	}
}
func TestCancellationPreventsNextSourceAndRetainsPage(t *testing.T) {
	r := health.NewRun("test", "test")
	r.Scope = &health.Scope{SubscriptionID: subscription}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	inventory := func(_ context.Context, _ string, emit func(health.Page) error) error {
		calls++
		if err := emit(health.Page{Resources: []health.ResourceInput{{ID: vm}}}); err != nil {
			return err
		}
		cancel()
		return ctx.Err()
	}
	other := func(context.Context, string, func(health.Page) error) error {
		t.Fatal("collection after cancellation")
		return nil
	}
	code, err := health.Check(ctx, &r, []health.NamedCollector{{Name: "inventory", Collect: inventory}, {Name: "resource_health", Collect: other}}, func(r *health.Run) error { r.Revision++; return nil })
	if err != nil || code != 130 || r.Outcome != "cancelled" || calls != 1 || len(r.Evidence) != 1 || r.Sources["resource_health"].Reason != "not_attempted" {
		t.Fatal("lost cancelled evidence/state")
	}
}

func TestInterruptionBetweenStageCheckpointAndCollectionLeavesStageNotAttempted(t *testing.T) {
	r := health.NewRun("synthetic", "test")
	r.Scope = &health.Scope{SubscriptionID: subscription}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	collector := func(context.Context, string, func(health.Page) error) error {
		t.Fatal("collector started after cancellation")
		return nil
	}
	code, err := health.Check(ctx, &r, []health.NamedCollector{{Name: "inventory", Collect: collector}, {Name: "resource_health", Collect: collector}}, func(*health.Run) error { cancel(); return nil })
	if err != nil || code != 130 || r.Outcome != "cancelled" || r.Sources["inventory"].Status != "not_checked" || r.Sources["inventory"].Reason != "not_attempted" {
		t.Fatalf("code %d err %v run %+v", code, err, r)
	}
}

func TestFourNamedCollectorsOutcomesAndCheckpointedProvenance(t *testing.T) {
	names := []string{"inventory", "resource_health", "service_health", "azure_monitor"}
	for mask := 0; mask < 16; mask++ {
		for _, retainPages := range []bool{false, true} {
			t.Run(fmt.Sprintf("failures=%04b/pages=%v", mask, retainPages), func(t *testing.T) {
				r := health.NewRun("test", "test")
				r.Scope = &health.Scope{SubscriptionID: subscription}
				store, err := runstore.New(t.TempDir(), &r)
				if err != nil {
					t.Fatal(err)
				}
				defer store.Close()
				sh, monitor := health.ServiceHealthQuery(r.StartedAt), health.MonitorQuery(r.StartedAt)
				r.Sources["service_health"] = health.Source{Status: "not_checked", Query: &sh}
				r.Sources["azure_monitor"] = health.Source{Status: "not_checked", Query: &monitor}
				read := func() health.Run {
					t.Helper()
					b, err := os.ReadFile(store.Path())
					if err != nil {
						t.Fatal(err)
					}
					var saved health.Run
					if err := json.Unmarshal(b, &saved); err != nil {
						t.Fatal(err)
					}
					if strings.Contains(string(b), "canary") {
						t.Fatal("raw error reached checkpoint")
					}
					return saved
				}
				order := []string{}
				collectors := []health.NamedCollector{}
				for i, name := range names {
					collectors = append(collectors, health.NamedCollector{Name: name, Collect: func(_ context.Context, id string, emit func(health.Page) error) error {
						order = append(order, name)
						saved := read()
						source := saved.Sources[name]
						if source.Status != "in_progress" || id != subscription {
							t.Fatal("collection preceded saved state or wrong subscription")
						}
						if name == "service_health" || name == "azure_monitor" {
							if source.Query == nil || source.Query.RequestedAt == nil || !source.Query.ReferenceAt.Equal(r.StartedAt) {
								t.Fatal("provenance not checkpointed before first read")
							}
						}
						if mask&(1<<i) == 0 || retainPages {
							page := health.Page{}
							switch name {
							case "inventory":
								page.Resources = []health.ResourceInput{{ID: vm}}
							case "resource_health":
								state := "Unavailable"
								page.Health = []health.HealthInput{{StatusID: vm + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current", PropertiesPresent: true, State: &state}}
							case "service_health":
								updated := r.StartedAt
								page.ServiceHealth = []health.ServiceHealthInput{{ID: "/providers/Microsoft.ResourceHealth/events/synthetic", PropertiesPresent: true, EventType: "ServiceIssue", Status: "Active", Level: "Bearer level-canary", LastUpdateTime: &updated}}
							case "azure_monitor":
								page.Monitor = []health.MonitorInput{{ID: "/subscriptions/" + subscription + "/providers/Microsoft.AlertsManagement/alerts/synthetic", PropertiesPresent: true, SignalType: "Log", Condition: "Fired", WorkflowState: "Bearer workflow-canary", TargetResourceID: "/subscriptions/22222222-2222-2222-2222-222222222222/providers/Foreign.Provider/targets/target-canary"}}
							}
							if err := emit(page); err != nil {
								return err
							}
							if read().Sources[name].Pages != 1 {
								t.Fatal("page not saved before next read")
							}
						}
						if mask&(1<<i) != 0 {
							if i%2 == 0 {
								return failure("login_required")
							}
							return failure("access_denied")
						}
						return nil
					}})
				}
				code, err := health.Check(context.Background(), &r, collectors, store.Save)
				wantCode, wantOutcome := 3, "partial"
				if mask == 0 {
					wantCode, wantOutcome = 0, "completed"
				}
				if mask == 15 && !retainPages {
					wantCode, wantOutcome = 1, "failed"
				}
				if err != nil || code != wantCode || r.Outcome != wantOutcome || !reflect.DeepEqual(order, names) || r.Coverage != "incomplete" {
					t.Fatalf("code %d err %v outcome %s order %v", code, err, r.Outcome, order)
				}
				saved := read()
				for i, name := range names {
					wantStatus := "completed"
					if mask&(1<<i) != 0 {
						wantStatus = "failed"
						if retainPages {
							wantStatus = "partial"
						}
					}
					if saved.Sources[name].Status != wantStatus {
						t.Fatalf("source %s: %+v", name, saved.Sources[name])
					}
				}
				if !reflect.DeepEqual(*saved.Sources["service_health"].Query.Cutoff, *sh.Cutoff) || saved.Sources["service_health"].Query.QueryStartTime != sh.QueryStartTime || saved.Sources["azure_monitor"].Query.TimeRange != "30d" {
					t.Fatal("transition lost immutable query policy")
				}
				if saved.Sources["advisor"].Status != "not_checked" {
					t.Fatal("Advisor coverage invented")
				}
			})
		}
	}
}

func TestOperationalEnumerationUnknownExcludedAndRejectedRemainDistinct(t *testing.T) {
	r := health.NewRun("test", "test")
	r.Scope = &health.Scope{SubscriptionID: subscription}
	reference := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	query := health.ServiceHealthQuery(reference)
	r.Sources["service_health"] = health.Source{Query: &query, Pages: 2, ExcludedRecords: 3, UnclassifiedRecords: 4}
	input := health.ServiceHealthInput{ID: "/providers/Microsoft.ResourceHealth/events/synthetic", PropertiesPresent: true, EventType: "FutureCategory", Status: "Active", LastUpdateTime: &reference}
	code, err := health.Check(context.Background(), &r, []health.NamedCollector{{Name: "service_health", Collect: func(_ context.Context, _ string, emit func(health.Page) error) error {
		return emit(health.Page{ServiceHealth: []health.ServiceHealthInput{input}})
	}}}, func(*health.Run) error { return nil })
	source := r.Sources["service_health"]
	if err != nil || code != 0 || source.Status != "completed" || source.Pages != 3 || source.RejectedRecords != 0 || source.ExcludedRecords != 3 || source.UnclassifiedRecords != 5 || source.Query.QueryStartTime != query.QueryStartTime || len(r.Findings) != 0 {
		t.Fatalf("source completion confused with unknown coverage: %+v", source)
	}
	// Rejected records make enumeration partial even without an HTTP failure.
	code, err = health.Check(context.Background(), &r, []health.NamedCollector{{Name: "service_health", Collect: func(_ context.Context, _ string, emit func(health.Page) error) error {
		return emit(health.Page{ServiceHealth: []health.ServiceHealthInput{{}}})
	}}}, func(*health.Run) error { return nil })
	if err != nil || code != 3 || r.Sources["service_health"].Status != "partial" || r.Sources["service_health"].RejectedRecords != 1 || r.Sources["service_health"].UnclassifiedRecords != 5 {
		t.Fatal("rejection accounting conflated")
	}
}

func TestNamedCollectorsStopOnOperationalCheckpointFailureOrCancellation(t *testing.T) {
	names := []string{"inventory", "resource_health", "service_health", "azure_monitor"}
	for _, stopSource := range []string{"service_health", "azure_monitor"} {
		for _, scenario := range []string{"stage failure", "page failure", "completion failure", "cancel after page", "cancel after stage"} {
			t.Run(stopSource+"/"+scenario, func(t *testing.T) {
				r := health.NewRun("test", "test")
				r.Scope = &health.Scope{SubscriptionID: subscription}
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				collectors := []health.NamedCollector{}
				calls := []string{}
				saves := 0
				inCallback := false
				for _, name := range names {
					collectors = append(collectors, health.NamedCollector{Name: name, Collect: func(_ context.Context, _ string, emit func(health.Page) error) error {
						calls = append(calls, name)
						if name == stopSource {
							inCallback = true
							err := emit(health.Page{})
							inCallback = false
							if scenario == "cancel after page" {
								cancel()
							}
							return err
						}
						return nil
					}})
				}
				code, err := health.Check(ctx, &r, collectors, func(run *health.Run) error {
					source := run.Sources[stopSource]
					if source.Status == "in_progress" || source.Status == "completed" {
						saves++
					}
					if source.Status == "in_progress" && source.Pages == 0 {
						if scenario == "stage failure" {
							return errors.New("storage-canary")
						}
						if scenario == "cancel after stage" {
							cancel()
						}
					}
					if inCallback && scenario == "page failure" {
						return errors.New("storage-canary")
					}
					if source.Status == "completed" && scenario == "completion failure" {
						return errors.New("storage-canary")
					}
					return nil
				})
				wantCode := 1
				if strings.HasPrefix(scenario, "cancel") {
					wantCode = 130
				}
				if code != wantCode || (err == nil) != (wantCode == 130) || saves == 0 {
					t.Fatalf("code %d err %v calls %v saves %d", code, err, calls, saves)
				}
				for _, name := range calls {
					if stopSource == "service_health" && name == "azure_monitor" {
						t.Fatal("later source read after stop")
					}
				}
				if scenario == "cancel after stage" && r.Sources[stopSource].Status != "not_checked" {
					t.Fatal("cancelled unstarted collector claimed attempted")
				}
				if stopSource == "service_health" && r.Sources["azure_monitor"].Reason != "not_attempted" {
					t.Fatal("supplied collector claimed unimplemented")
				}
			})
		}
	}
}

func TestStorageFailureCannotBeHiddenByCollector(t *testing.T) {
	r := health.NewRun("test", "test")
	r.Scope = &health.Scope{SubscriptionID: subscription}
	saves := 0
	storageErr := errors.New("storage-canary")
	code, err := health.Check(context.Background(), &r, []health.NamedCollector{{Name: "inventory", Collect: func(_ context.Context, _ string, emit func(health.Page) error) error {
		_ = emit(health.Page{Resources: []health.ResourceInput{{ID: vm}}})
		_ = emit(health.Page{Resources: []health.ResourceInput{{ID: vm + "-uncommitted"}}})
		return failure("login_required")
	}}}, func(*health.Run) error {
		saves++
		if saves == 2 {
			return storageErr
		}
		return nil
	})
	if code != 1 || !errors.Is(err, storageErr) || saves != 2 || len(r.Evidence) != 1 {
		t.Fatal("collector hid checkpoint failure or processed another page")
	}
}

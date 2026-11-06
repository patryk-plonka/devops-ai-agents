package health_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

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
			code, err := health.Check(context.Background(), &r, makeCollector("inventory", tc.inventoryFailure, tc.inventoryPage), makeCollector("resource_health", tc.healthFailure, tc.healthPage), store.Save)
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
				if s.Status != "not_checked" || s.Reason != "not_implemented_in_s01" {
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
	code, err := health.Check(context.Background(), &r, collect, collect, save)
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
	code, err := health.Check(ctx, &r, inventory, other, func(r *health.Run) error { r.Revision++; return nil })
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
	code, err := health.Check(ctx, &r, collector, collector, func(*health.Run) error { cancel(); return nil })
	if err != nil || code != 130 || r.Outcome != "cancelled" || r.Sources["inventory"].Status != "not_checked" || r.Sources["inventory"].Reason != "not_attempted" {
		t.Fatalf("code %d err %v run %+v", code, err, r)
	}
}

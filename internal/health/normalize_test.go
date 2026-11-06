package health

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

const testSubscription = "11111111-1111-1111-1111-111111111111"
const otherSubscription = "22222222-2222-2222-2222-222222222222"
const resourcePrefix = "/subscriptions/" + testSubscription + "/resourceGroups/Group/providers/Microsoft.Compute/virtualMachines/"

func ptr[T any](v T) *T { return &v }
func scopedRun() Run {
	r := NewRun("test", "test")
	r.Scope = &Scope{SubscriptionID: testSubscription}
	return r
}
func healthInput(name, state string) HealthInput {
	return HealthInput{StatusID: resourcePrefix + name + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current", PropertiesPresent: true, State: ptr(state)}
}
func hasGap(r Run, reason string) bool {
	for _, g := range r.Gaps {
		if g.Reason == reason {
			return true
		}
	}
	return false
}
func assertReferences(t *testing.T, r Run) {
	t.Helper()
	ids := map[string]bool{}
	for _, e := range r.Evidence {
		if e.ID == "" || ids[e.ID] {
			t.Fatal("invalid evidence identity")
		}
		ids[e.ID] = true
	}
	for _, f := range r.Findings {
		if len(f.EvidenceIDs) == 0 {
			t.Fatal("unsupported finding")
		}
		for _, id := range f.EvidenceIDs {
			if !ids[id] {
				t.Fatal("dangling finding evidence")
			}
		}
	}
	for _, g := range r.Gaps {
		for _, id := range g.EvidenceIDs {
			if !ids[id] {
				t.Fatal("dangling gap evidence")
			}
		}
	}
	for _, o := range r.Observations {
		for _, id := range o.EvidenceIDs {
			if !ids[id] {
				t.Fatal("dangling observation evidence")
			}
		}
	}
}

func TestResourceIDBoundaries(t *testing.T) {
	for _, tc := range []struct {
		id    string
		valid bool
	}{
		{resourcePrefix + "vm", true}, {strings.ToUpper(resourcePrefix + "vm"), true},
		{resourcePrefix + "vm/extensions/agent", true},
		{resourcePrefix + "vm/providers/Microsoft.Insights/diagnosticSettings/one", true},
		{"/subscriptions/" + testSubscription + "/providers/Microsoft.Authorization/policyAssignments/one", true},
		{strings.Replace(resourcePrefix, testSubscription, otherSubscription, 1) + "vm", false},
		{resourcePrefix + "", false}, {resourcePrefix + "..", false}, {resourcePrefix + "vm/../other", false},
		{resourcePrefix + "vm?secret=canary", false}, {resourcePrefix + "vm%2fother", false},
		{resourcePrefix + "vm\x1b[31m", false}, {"/subscriptions/" + testSubscription, false},
	} {
		if got := ResourceID(tc.id, testSubscription); got != tc.valid {
			t.Errorf("%q: %v", tc.id, got)
		}
	}
}

func TestNormalizeStatesAndTraceability(t *testing.T) {
	r := scopedRun()
	inputs := []HealthInput{}
	for _, state := range []string{"Available", "Unavailable", "Degraded", "Unknown", "brand-new-state"} {
		inputs = append(inputs, healthInput(state, state))
	}
	event := healthInput("event", "Available")
	event.State = nil
	event.EventID = "event-123"
	event.ReportedAt = ptr(time.Date(2026, 11, 6, 10, 0, 0, 0, time.FixedZone("test", 3600)))
	inputs = append(inputs, event, HealthInput{StatusID: resourcePrefix + "missing/proviDers/Microsoft.ResourceHealth/availabilityStatuses/current"})
	r.NormalizePage("resource_health", Page{Health: inputs})
	if len(r.Evidence) != 7 || len(r.Findings) != 3 || !hasGap(r, "source_unknown") || !hasGap(r, "undetermined_health_evidence") || !hasGap(r, "health_resource_absent_from_inventory") {
		t.Fatalf("run %+v", r)
	}
	states := map[string]string{}
	for _, o := range r.Observations {
		states[strings.Split(o.ResourceID, "/")[8]] = o.State
	}
	for _, state := range []string{"Available", "Unavailable", "Degraded", "Unknown"} {
		if states[state] != state {
			t.Fatalf("lost state %s: %v", state, states)
		}
	}
	if states["event"] != "undetermined" || states["missing"] != "undetermined" || states["brand-new-state"] != "undetermined" {
		t.Fatal("invented health state")
	}
	for _, e := range r.Evidence {
		if e.EventID != "" && (e.Kind != "event" || e.ReportedAt.Location() != time.UTC || e.CollectedAt.IsZero()) {
			t.Fatal("lost event timestamp/kind")
		}
	}
	if r.Coverage != "incomplete" {
		t.Fatal("invented overall assurance")
	}
	assertReferences(t, r)
}

func TestJoinDuplicatesConflictsAndCurrentEvidence(t *testing.T) {
	r := scopedRun()
	r.Sources["inventory"] = Source{Status: "completed"}
	r.Sources["resource_health"] = Source{Status: "completed"}
	r.NormalizePage("inventory", Page{Resources: []ResourceInput{{ID: resourcePrefix + "vm", Name: "vm", Tags: map[string]*string{"env": ptr("dev")}}, {ID: resourcePrefix + "missing", Name: "missing"}}})
	input := healthInput("vm", "Available")
	input.StatusID = strings.ToUpper(input.StatusID)
	r.NormalizePage("resource_health", Page{Health: []HealthInput{input, input}})
	if len(r.Evidence) != 3 || len(r.Observations) != 2 || !hasGap(r, "missing_health_evidence") || hasGap(r, "health_resource_absent_from_inventory") {
		t.Fatalf("join/duplicate failed %+v", r)
	}
	firstID := r.Evidence[2].ID
	r.NormalizePage("resource_health", Page{Health: []HealthInput{input}})
	if len(r.Evidence) != 3 || r.Evidence[2].ID != firstID {
		t.Fatal("unstable identity/duplicate")
	}
	r.NormalizePage("resource_health", Page{Health: []HealthInput{healthInput("vm", "Unavailable")}})
	if !hasGap(r, "conflicting_health_evidence") {
		t.Fatal("conflict hidden")
	}
	for _, o := range r.Observations {
		if strings.HasSuffix(strings.ToLower(o.ResourceID), "/vm") && (o.State != "undetermined" || len(o.EvidenceIDs) != 2) {
			t.Fatal("conflict became healthy")
		}
	}
	assertReferences(t, r)
	r = scopedRun()
	old := healthInput("vm", "Unavailable")
	old.ReportedAt = ptr(time.Date(2026, 11, 5, 0, 0, 0, 0, time.UTC))
	newer := healthInput("vm", "Available")
	newer.ReportedAt = ptr(old.ReportedAt.Add(time.Hour))
	r.NormalizePage("resource_health", Page{Health: []HealthInput{old, newer}})
	if r.Observations[0].State != "Available" || hasGap(r, "conflicting_health_evidence") || len(r.Evidence) != 2 {
		t.Fatal("unambiguous latest evidence lost")
	}
	newer.State = ptr("Degraded")
	r.NormalizePage("resource_health", Page{Health: []HealthInput{newer}})
	if r.Observations[0].State != "undetermined" || !hasGap(r, "conflicting_health_evidence") {
		t.Fatal("equal timestamps chose default")
	}
}

func TestMixedScopeAndSecretFilteringBeforeJSON(t *testing.T) {
	r := scopedRun()
	r.NormalizePage("inventory", Page{Resources: []ResourceInput{
		{ID: strings.Replace(resourcePrefix, testSubscription, otherSubscription, 1) + "rejected-payload-canary", Name: "rejected-payload-canary"},
		{ID: resourcePrefix + "vm", Name: "\x1b[31mvm\x1b[0m", Type: "Microsoft.Compute/virtualMachines", Tags: map[string]*string{"environment": ptr("<dev>|[x](url)"), "Client-Secret": ptr("tag-secret-canary"), "endpoint": ptr("https://user:value-secret-canary@example.com"), "null": nil}},
	}})
	valid := healthInput("vm", "Degraded")
	invalid := healthInput("foreign", "Available")
	invalid.StatusID = strings.Replace(invalid.StatusID, testSubscription, otherSubscription, 1)
	r.NormalizePage("resource_health", Page{Health: []HealthInput{valid, invalid, {StatusID: resourcePrefix + "vm/not-a-status", EventID: "rejected-payload-canary"}}})
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "secret-canary") || strings.Contains(string(b), "rejected-payload-canary") || strings.Contains(string(b), "\\u001b") {
		t.Fatalf("unsafe saved data: %s", b)
	}
	if len(r.Resources) != 1 || len(r.Evidence) != 2 || r.Redactions != 2 || r.Resources[0].Tags["environment"] != "<dev>|[x](url)" || r.Resources[0].Tags["Client-Secret"] != "[REDACTED]" || !hasGap(r, "invalid_records") {
		t.Fatalf("filter changed ordinary context %+v", r)
	}
	assertReferences(t, r)
}

func TestEmptyAndMissingHealthAreNotHealthy(t *testing.T) {
	r := scopedRun()
	r.Sources["inventory"] = Source{Status: "completed"}
	r.Sources["resource_health"] = Source{Status: "completed"}
	r.RebuildCoverage()
	if !hasGap(r, "no_resources_observed") || r.Coverage != "incomplete" || len(r.Findings) != 0 {
		t.Fatal("empty became healthy")
	}
	r.NormalizePage("inventory", Page{Resources: []ResourceInput{{ID: resourcePrefix + "vm"}}})
	if !hasGap(r, "missing_health_evidence") || r.Observations[0].State != "not_checked" {
		t.Fatal("absent became Available/unsupported")
	}
}

func TestStatusIDSuffixAndHistoricNames(t *testing.T) {
	for _, tc := range []struct {
		suffix string
		valid  bool
	}{
		{"/providers/Microsoft.ResourceHealth/availabilityStatuses/current", true},
		{"/providers/Microsoft.ResourceHealth/availabilityStatuses/2026-11-06T10%3A00%3A00Z", true},
		{"/providers/Foreign.Provider/availabilityStatuses/current", false},
		{"/providers/Microsoft.ResourceHealth/availabilityStatuses/current/extra", false},
		{"/providers/Microsoft.ResourceHealth/availabilityStatuses/..", false},
		{"/providers/Microsoft.ResourceHealth/availabilityStatuses/%2E%2E", false},
		{"/providers/Microsoft.ResourceHealth/availabilityStatuses/%2Fescape", false},
		{"/providers/Microsoft.ResourceHealth/availabilityStatuses/%0A", false},
		{"/providers/Microsoft.ResourceHealth/availabilityStatuses/%zz", false},
	} {
		_, got := statusResourceID(resourcePrefix+"vm"+tc.suffix, testSubscription)
		if got != tc.valid {
			t.Errorf("%s: %v", tc.suffix, got)
		}
	}
}

func TestEventMetadataWithoutIDRemainsEventEvidence(t *testing.T) {
	r := scopedRun()
	input := healthInput("event", "Available")
	input.State = nil
	input.EventPresent = true
	r.NormalizePage("resource_health", Page{Health: []HealthInput{input}})
	if len(r.Findings) != 1 || r.Findings[0].Category != "resource_health_event" || r.Observations[0].State != "undetermined" || r.Evidence[0].Kind != "event" {
		t.Fatal("event metadata lost or current state invented")
	}
	assertReferences(t, r)
}

func TestHistoricalEvidenceCannotDetermineOrConflictWithCurrentState(t *testing.T) {
	for _, tc := range []struct{ name, currentState, historicState string }{
		{"history only", "", "Available"},
		{"current issue with historical healthy", "Unavailable", "Available"},
		{"current healthy with historical issue", "Available", "Unavailable"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := scopedRun()
			history := healthInput("vm", tc.historicState)
			history.StatusID = strings.TrimSuffix(history.StatusID, "current") + "2026-11-05T10%3A00%3A00Z"
			history.ReportedAt = ptr(time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC))
			inputs := []HealthInput{history}
			if tc.currentState != "" {
				current := healthInput("vm", tc.currentState)
				current.ReportedAt = ptr(history.ReportedAt.Add(-time.Hour))
				inputs = append(inputs, current)
			}
			r.NormalizePage("resource_health", Page{Health: inputs})
			expected := tc.currentState
			if expected == "" {
				expected = "undetermined"
			}
			if len(r.Observations) != 1 || r.Observations[0].State != expected || hasGap(r, "conflicting_health_evidence") || !hasGap(r, "historical_health_evidence") {
				t.Fatalf("history altered current state: %+v", r.Observations)
			}
			if tc.currentState == "" && !hasGap(r, "missing_current_health_evidence") {
				t.Fatal("historical-only coverage hidden")
			}
			if tc.historicState == "Unavailable" && len(r.Findings) != 0 {
				t.Fatal("historical issue became current finding")
			}
			if len(r.Evidence) != len(inputs) || len(r.Observations[0].EvidenceIDs) != len(inputs) {
				t.Fatal("historical evidence lost")
			}
			for _, e := range r.Evidence {
				expectedRole := "current"
				if strings.Contains(e.StatusID, "2026-11-05T") {
					expectedRole = "historical"
				}
				if e.StatusRole != expectedRole {
					t.Fatal("source status role lost")
				}
			}
			assertReferences(t, r)
		})
	}
}

package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

const sub = "11111111-1111-1111-1111-111111111111"
const resource = "/subscriptions/" + sub + "/resourceGroups/g/providers/Microsoft.Compute/virtualMachines/"

func ptr(s string) *string { return &s }
func example() health.Run {
	r := health.NewRun("Synthetic subscription", "test")
	now := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	r.ID = "synthetic-run"
	r.Revision = 9
	r.StartedAt = now
	r.UpdatedAt = now
	r.EndedAt = &now
	r.Outcome = "completed"
	r.Scope = &health.Scope{SubscriptionID: sub, Name: "Synthetic subscription", TenantID: "22222222-2222-2222-2222-222222222222", Cloud: "AzureCloud"}
	for _, name := range []string{"inventory", "resource_health"} {
		r.Sources[name] = health.Source{Status: "completed"}
	}
	var inventory []health.ResourceInput
	var statuses []health.HealthInput
	for i, state := range []string{"Available", "Unavailable", "Degraded", "Unknown", "event", "missing", "historical", "conflict"} {
		id := fmt.Sprintf("%svm-%02d", resource, i)
		inventory = append(inventory, health.ResourceInput{ID: id, Name: fmt.Sprintf("vm-%02d", i), Type: "Microsoft.Compute/virtualMachines", Location: "westeurope", Tags: map[string]*string{"owner": ptr("consultant"), "password": ptr("credential-canary"), "display": ptr("<b>x</b>|[click](https://example.test)\n# heading\x1b[31m")}})
		if state == "missing" {
			continue
		}
		status := health.HealthInput{StatusID: id + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current", PropertiesPresent: true, State: ptr(state), ReportedAt: &now}
		if state == "historical" {
			status.StatusID = id + "/providers/Microsoft.ResourceHealth/availabilityStatuses/2026-11-05"
			status.State = ptr("Available")
		}
		if state == "event" {
			status.State = nil
			status.EventPresent = true
			status.EventID = "event-123"
			status.OccurredAt = &now
		}
		if state == "conflict" {
			status.State = ptr("Available")
			statuses = append(statuses, status)
			status.State = ptr("Unavailable")
		}
		statuses = append(statuses, status)
	}
	r.NormalizePage("inventory", health.Page{Resources: inventory, CollectedAt: now})
	r.NormalizePage("resource_health", health.Page{Health: statuses, CollectedAt: now})
	return r
}
func TestMarkdownEvidenceCoverageAndLiteralText(t *testing.T) {
	r := example()
	before, _ := json.Marshal(r)
	b, err := Markdown(r)
	if err != nil {
		t.Fatal(err)
	}
	text := html.UnescapeString(string(b))
	for _, want := range []string{"INCOMPLETE coverage", "Execution outcome: **completed**", "Revision: 9", r.ID, "Resource Health Available observations", "Unknown and not-checked resources", "source_unknown", "missing_health_evidence", "historical_health_evidence", "conflicting_health_evidence", "not_implemented_in_s01", "event-123", health.ResourceHealthAPI, health.InventoryAPI, "owner=consultant", "password=[REDACTED]", "source reported (UTC): 2026-11-06T10:00:00Z"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, bad := range []string{"credential-canary", "<b>", "[click](https://example.test)", "\x1b", "\n# heading", "|[click]"} {
		if bytes.Contains(b, []byte(bad)) {
			t.Errorf("active/unfiltered content %q", bad)
		}
	}
	for _, f := range r.Findings {
		for _, id := range f.EvidenceIDs {
			if !strings.Contains(text, "["+id+"](run.json)") {
				t.Errorf("missing finding reference %s", id)
			}
		}
	}
	for _, e := range r.Evidence {
		if !strings.Contains(text, "### "+e.ID) || !strings.Contains(text, "["+e.ID+"](run.json)") {
			t.Errorf("unresolvable evidence %s", e.ID)
		}
	}
	after, _ := json.Marshal(r)
	if !bytes.Equal(before, after) {
		t.Fatal("renderer mutated snapshot")
	}
	// Reordering serialized collections cannot change the published report.
	slices.Reverse(r.Evidence)
	slices.Reverse(r.Resources)
	slices.Reverse(r.Observations)
	slices.Reverse(r.Findings)
	slices.Reverse(r.Gaps)
	other, err := Markdown(r)
	if err != nil || !bytes.Equal(b, other) {
		t.Fatal("nondeterministic report")
	}
}
func TestReportDoesNotTruncateObservedResources(t *testing.T) {
	r := example()
	r.Evidence = nil
	r.RebuildCoverage()
	now := *r.EndedAt
	page := health.Page{CollectedAt: now}
	for i := range 350 {
		page.Resources = append(page.Resources, health.ResourceInput{ID: fmt.Sprintf("%sresource-%03d", resource, i), Name: fmt.Sprintf("resource-%03d", i), Tags: map[string]*string{"z": ptr("last"), "a": ptr("first")}})
	}
	r.NormalizePage("inventory", page)
	b, err := Markdown(r)
	if err != nil {
		t.Fatal(err)
	}
	text := html.UnescapeString(string(b))
	if !strings.Contains(text, "Observed inventory (350 resources)") || !strings.Contains(text, "a=first; z=last") {
		t.Fatal("missing inventory/tags")
	}
	for _, v := range r.Resources {
		if strings.Count(text, "| "+v.ID+" |") < 2 {
			t.Errorf("resource omitted from inventory or not-checked table: %s", v.ID)
		}
	}
}
func TestReportRejectsUnsupportedReferencesAndUnfinalizedRuns(t *testing.T) {
	for _, mutate := range []func(*health.Run){
		func(r *health.Run) { r.EndedAt = nil }, func(r *health.Run) { r.Outcome = "in_progress" }, func(r *health.Run) { r.Revision = 0 }, func(r *health.Run) { r.ID = "" },
		func(r *health.Run) { r.Findings[0].EvidenceIDs = nil }, func(r *health.Run) { r.Findings[0].EvidenceIDs = []string{"absent"} },
		func(r *health.Run) { r.Evidence = append(r.Evidence, r.Evidence[0]) }, func(r *health.Run) { r.Evidence[0].ID = "" },
		func(r *health.Run) { r.Observations[0].EvidenceIDs = []string{"absent"} }, func(r *health.Run) { r.Resources[0].EvidenceIDs = []string{"absent"} }, func(r *health.Run) { r.Gaps[0].EvidenceIDs = []string{"absent"} },
	} {
		r := example()
		mutate(&r)
		if _, err := Markdown(r); err == nil {
			t.Fatal("invalid report published")
		}
	}
}
func TestFailedAndCancelledReportsAreExplicit(t *testing.T) {
	for _, outcome := range []string{"failed", "cancelled"} {
		r := example()
		r.Scope = nil
		r.Evidence = nil
		r.Diagnostics = []string{"subscription_access_denied"}
		r.Outcome = outcome
		for _, name := range []string{"inventory", "resource_health"} {
			r.Sources[name] = health.Source{Status: "not_checked", Reason: "not_attempted"}
		}
		r.RebuildCoverage()
		b, err := Markdown(r)
		if err != nil {
			t.Fatal(err)
		}
		text := html.UnescapeString(string(b))
		for _, want := range []string{"Execution outcome: **" + outcome + "**", "Resolved scope: unresolved", "No evidence collected", "not_attempted", "subscription_access_denied", "INCOMPLETE"} {
			if !strings.Contains(text, want) {
				t.Errorf("missing %q", want)
			}
		}
	}
}
func TestSyntheticReportExample(t *testing.T) {
	b, err := Markdown(example())
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/mixed-report.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, want) {
		t.Fatal("synthetic example no longer matches rendered report")
	}
	data, err := os.ReadFile("testdata/run.json")
	if err != nil {
		t.Fatal(err)
	}
	var saved health.Run
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatal(err)
	}
	fromSnapshot, err := Markdown(saved)
	if err != nil || !bytes.Equal(fromSnapshot, want) {
		t.Fatal("example report differs from its sibling snapshot")
	}
}

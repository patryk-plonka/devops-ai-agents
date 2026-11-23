package health

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func selectionRun(t *testing.T) Run {
	t.Helper()
	older, newer, available := healthInput("outside", "Degraded"), healthInput("outside", "Degraded"), healthInput("outside", "Available")
	older.ReportedAt = ptr(operationalReference.Add(-3 * time.Hour))
	newer.ReportedAt = ptr(operationalReference.Add(-2 * time.Hour))
	available.ReportedAt = ptr(operationalReference.Add(-time.Hour))
	alert := monitorInput()
	alert.TargetResourceID = resourcePrefix + "vm"
	newAdvisor := advisorInput("vm")
	newAdvisor.TargetResourceID = ""
	postponed := advisorInput("postponed")
	postponed.Status = "Postponed"
	return classifiedRun(t, map[string][]Page{
		"inventory":       {assessmentInventory("vm", "unrelated-private-marker")},
		"resource_health": {{Health: []HealthInput{older, available, newer}}},
		"service_health":  {{ServiceHealth: []ServiceHealthInput{serviceInput()}}},
		"azure_monitor":   {{Monitor: []MonitorInput{alert}}},
		advisorSource:     {{Advisor: []AdvisorInput{newAdvisor, postponed}}},
	})
}

func selectCategory(t *testing.T, r Run, category string) Finding {
	t.Helper()
	for _, f := range r.Findings {
		if f.Category == category {
			return f
		}
	}
	t.Fatalf("missing finding category %s", category)
	return Finding{}
}

func TestResolveDraftSelectionOrderAndScope(t *testing.T) {
	r := selectionRun(t)
	alert := selectCategory(t, r, "monitor_alert")
	notice := selectCategory(t, r, "service_health_ServiceIssue")
	advisor := selectCategory(t, r, "advisor_recommendation")
	r.ID, r.Revision, r.Outcome = "reviewed-run", 7, "cancelled"
	r.RebuildCoverage()
	before := append([]Finding(nil), r.Findings...)
	result, err := r.ResolveDraftSelection([]string{notice.ID, alert.ID, notice.ID, advisor.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result.RunID != r.ID || result.Revision != r.Revision || result.Outcome != "cancelled" ||
		result.Coverage != "incomplete" || len(result.Findings) != 3 || !reflect.DeepEqual(before, r.Findings) {
		t.Fatalf("unexpected selection state: %+v", result)
	}
	service, fired, recommendation := result.Findings[0], result.Findings[1], result.Findings[2]
	if service.Finding.ID != notice.ID || service.Finding.ResourceID != "" || service.Collection.Query.Cutoff == nil ||
		!hasSelectionGap(service, "resource_impact_not_enumerated", true) {
		t.Fatalf("missing subscription notice context: %+v", service)
	}
	if fired.Finding.ID != alert.ID || fired.TargetName != "vm" || len(fired.TargetEvidenceIDs) != 1 ||
		fired.Collection.Query.TimeRange != "30d" || !hasSelectionGap(fired, "missing_health_evidence", false) {
		t.Fatalf("known alert lost its scope or missing proof: %+v", fired)
	}
	if recommendation.Finding.ID != advisor.ID || recommendation.Advisor == nil || recommendation.Advisor.Status != "New" ||
		recommendation.TargetName != "" || !hasSelectionGap(recommendation, "unknown_resource_attribution", false) {
		t.Fatalf("missing Advisor context: %+v", recommendation)
	}
	for _, item := range result.Findings {
		if !hasSelectionGap(item, "execution_not_finalized_successfully", true) {
			t.Fatalf("missing cancelled execution gap for %s", item.Source)
		}
		for _, g := range item.Gaps {
			if g.Broader && len(g.EvidenceIDs) > 0 {
				t.Fatalf("broader gap included unrelated trace references: %+v", g)
			}
		}
	}
}

func hasSelectionGap(item SelectedFinding, reason string, broader bool) bool {
	return slices.ContainsFunc(item.Gaps, func(g SelectionGap) bool {
		return g.Reason == reason && g.Broader == broader
	})
}

func TestResolveDraftSelectionKeepsRunCoverageAndSummarizesGroupGaps(t *testing.T) {
	input := map[string][]Page{
		"inventory":       {assessmentInventory("vm", "unrelated-private-marker")},
		"resource_health": {{Health: []HealthInput{healthInput("vm", "Available")}}},
		"azure_monitor":   {{Monitor: []MonitorInput{monitorInput()}}},
	}
	r := classifiedRun(t, input)
	alert := selectCategory(t, r, "monitor_alert")
	result, err := r.ResolveDraftSelection([]string{alert.ID})
	if err != nil {
		t.Fatal(err)
	}
	if result.Coverage != "incomplete" || slices.ContainsFunc(result.Findings[0].Gaps, func(g SelectionGap) bool { return g.Blocking }) {
		t.Fatalf("another resource's gap changed the selected target's proof: %+v", result)
	}

	groupSuggestion := advisorInput("vm")
	groupSuggestion.ID = advisorSubscription + advisorSuffix + "group"
	groupSuggestion.TargetResourceID = resourceGroupTarget(alert.ResourceID)
	input[advisorSource] = []Page{{Advisor: []AdvisorInput{groupSuggestion}}}
	r = classifiedRun(t, input)
	groupFinding := selectCategory(t, r, "advisor_recommendation")
	result, err = r.ResolveDraftSelection([]string{alert.ID, groupFinding.ID})
	if err != nil {
		t.Fatal(err)
	}
	if !hasSelectionGap(result.Findings[0], "advisor_group_target_only", true) ||
		!hasSelectionGap(result.Findings[1], "advisor_group_target_only", false) {
		t.Fatalf("group blocker was lost or attached to an unrelated finding: %+v", result)
	}
	for _, g := range result.Findings[0].Gaps {
		if g.Source == advisorSource && len(g.EvidenceIDs) != 0 {
			t.Fatalf("another finding's evidence leaked into group context: %+v", g)
		}
	}
}

func TestResolveDraftSelectionBindsEachHistoricalObservation(t *testing.T) {
	r := selectionRun(t)
	var history []Finding
	for _, f := range r.Findings {
		if f.Category == "resource_health_issue" {
			history = append(history, f)
		}
	}
	if len(history) != 2 {
		t.Fatal("expected two historical observations")
	}
	result, err := r.ResolveDraftSelection([]string{history[0].ID, history[1].ID})
	if err != nil {
		t.Fatal(err)
	}
	for i, f := range result.Findings {
		original := f.Supporting[0]
		now := f.CurrentHealth
		if original.ID != history[i].EvidenceIDs[0] || original.State != "Degraded" ||
			now.State != "Available" || len(now.Observations) != 1 || !now.Observations[0].ReportedAt.After(*original.ReportedAt) ||
			!now.Historical || now.Conflict || now.MissingSourceTime || !hasSelectionGap(f, "health_resource_absent_from_inventory", false) {
			t.Fatalf("selected observation and later state do not agree: %+v", f)
		}
	}
	if result.Findings[0].Supporting[0].ReportedAt.Equal(*result.Findings[1].Supporting[0].ReportedAt) {
		t.Fatal("historical findings resolve to the same source time")
	}

	a := slices.IndexFunc(r.Findings, func(f Finding) bool { return f.ID == history[0].ID })
	b := slices.IndexFunc(r.Findings, func(f Finding) bool { return f.ID == history[1].ID })
	r.Findings[a].EvidenceIDs, r.Findings[b].EvidenceIDs = r.Findings[b].EvidenceIDs, r.Findings[a].EvidenceIDs
	assertInvalidSelection(t, r, []string{selectCategory(t, r, "monitor_alert").ID, history[0].ID})
}

func TestResolveDraftSelectionKeepsResourceHealthConflict(t *testing.T) {
	for _, scenario := range []string{"tied source time", "missing source time"} {
		t.Run(scenario, func(t *testing.T) {
			degraded, available := healthInput("vm", "Degraded"), healthInput("vm", "Available")
			degraded.ReportedAt = ptr(operationalReference.Add(-time.Hour))
			if scenario == "tied source time" {
				available.ReportedAt = degraded.ReportedAt
			}
			r := classifiedRun(t, map[string][]Page{
				"resource_health": {{Health: []HealthInput{degraded}}, {Health: []HealthInput{available}, CollectedAt: operationalReference.Add(time.Minute)}},
			})
			result, err := r.ResolveDraftSelection([]string{selectCategory(t, r, "resource_health_issue").ID})
			if err != nil {
				t.Fatal(err)
			}
			f := result.Findings[0]
			now := f.CurrentHealth
			if now.State != "undetermined" || len(now.Observations) != 2 || !now.Conflict || now.Historical ||
				now.MissingSourceTime != (scenario == "missing source time") || !hasSelectionGap(f, "conflicting_health_evidence", false) {
				t.Fatalf("saved conflict lost: %+v", now)
			}
		})
	}
}

func TestResolveDraftSelectionKeepsSupportingRowsAndConflictContext(t *testing.T) {
	fired, cleared := monitorInput(), monitorInput()
	cleared.Condition = "Resolved"
	cleared.LastModifiedTime = fired.LastModifiedTime
	low, high := advisorInput("vm"), advisorInput("vm")
	low.Impact, high.Impact = "Low", "High"
	r := classifiedRun(t, map[string][]Page{
		"inventory":     {assessmentInventory("vm")},
		"azure_monitor": {{Monitor: []MonitorInput{fired, cleared}}},
		advisorSource:   {{Advisor: []AdvisorInput{low, high}}},
	})
	alert, adj := selectCategory(t, r, "monitor_alert"), selectCategory(t, r, "advisor_recommendation")
	result, err := r.ResolveDraftSelection([]string{alert.ID, adj.ID})
	if err != nil {
		t.Fatal(err)
	}
	f, a := result.Findings[0], result.Findings[1]
	if len(f.Supporting) != 1 || len(f.Context) != 1 || f.Context[0].Monitor.Condition != "Resolved" {
		t.Fatalf("conflicting monitor row was lost or included as support: %+v", f)
	}
	if len(a.Supporting) != 2 || a.Advisor == nil || a.Advisor.Status != "New" || a.Advisor.Impact != advisorUnknown ||
		!hasSelectionGap(a, "conflicting_advisor_evidence", false) {
		t.Fatalf("Advisor observations and consensus disagree: %+v", a)
	}
	for i := range r.AdvisorInstances {
		if r.AdvisorInstances[i].FindingID == adj.ID {
			r.AdvisorInstances[i].Impact = "High"
		}
	}
	assertInvalidSelection(t, r, []string{alert.ID, adj.ID})
}

func TestResolveDraftSelectionKeepsSupersededAdvisorContext(t *testing.T) {
	for _, scenario := range []string{"workflow", "target", "attribution"} {
		t.Run(scenario, func(t *testing.T) {
			current := advisorInput("vm")
			current.ID = advisorSubscription + advisorSuffix + "shared"
			previous := current
			previous.LastUpdated = ptr(current.LastUpdated.Add(-time.Hour))
			switch scenario {
			case "workflow":
				previous.Status = "Dismissed"
			case "target":
				previous.TargetResourceID = resourcePrefix + "elsewhere"
			case "attribution":
				previous.TargetResourceID = ""
			}
			unrelated := advisorInput("unrelated")
			unrelated.Status = "Dismissed"
			alert := monitorInput()
			r := classifiedRun(t, map[string][]Page{
				"inventory":       {assessmentInventory("vm")},
				"resource_health": {{Health: []HealthInput{healthInput("vm", "Available")}}},
				"azure_monitor":   {{Monitor: []MonitorInput{alert}}},
				advisorSource:     {{Advisor: []AdvisorInput{previous, unrelated}}, {Advisor: []AdvisorInput{current}}},
			})
			finding := selectCategory(t, r, "advisor_recommendation")
			result, err := r.ResolveDraftSelection([]string{finding.ID})
			if err != nil {
				t.Fatal(err)
			}
			item := result.Findings[0]
			if len(item.Supporting) != 1 || item.Advisor.Status != "New" ||
				!item.Supporting[0].Advisor.LastUpdated.Equal(*current.LastUpdated) ||
				len(item.Context) != 1 || item.Context[0].EntityID != previous.ID ||
				!item.Context[0].Advisor.LastUpdated.Equal(*previous.LastUpdated) {
				t.Fatalf("missing or misattributed Advisor history: %+v", item)
			}

			for i := range r.Evidence {
				if r.Evidence[i].ID == item.Context[0].ID {
					r.Evidence[i].ID = "ev-" + strings.Repeat("a", 63) + "/"
				}
			}
			r.RebuildCoverage()
			assertInvalidSelection(t, r, []string{selectCategory(t, r, "monitor_alert").ID, finding.ID})
		})
	}
}

func TestResolveDraftSelectionRejectsInvalidBatch(t *testing.T) {
	for _, scenario := range []string{
		"empty", "short", "uppercase", "nonhex", "separator", "newline", "unknown",
		"duplicate saved", "unsaved", "wrong target", "wrong category", "wrong source", "wrong entity", "duplicate evidence",
		"invalid support", "invalid deciding evidence", "invalid gap evidence", "evidence only", "group heading", "assessment only",
	} {
		t.Run(scenario, func(t *testing.T) {
			r := selectionRun(t)
			alert := selectCategory(t, r, "monitor_alert")
			history := selectCategory(t, r, "resource_health_issue")
			i := slices.IndexFunc(r.Findings, func(f Finding) bool { return f.ID == alert.ID })
			ids := []string{history.ID, alert.ID}
			switch scenario {
			case "empty":
				ids = nil
			case "short", "uppercase", "nonhex", "separator", "newline":
				suffix := strings.Repeat("a", 63)
				last := map[string]string{"short": "", "uppercase": "B", "nonhex": "g", "separator": "/", "newline": "b\n"}[scenario]
				ids[1] = "finding-" + suffix + last
				r.Findings[i].ID = ids[1]
			case "unknown":
				ids[1] = "finding-" + strings.Repeat("f", 64)
			case "duplicate saved":
				r.Findings = append(r.Findings, alert)
			case "unsaved":
				r.Findings = slices.Delete(r.Findings, i, i+1)
			case "wrong target":
				r.Findings[i].ResourceID = resourcePrefix + "unrelated-private-marker"
			case "wrong category":
				r.Findings[i].Category = "service_health_ServiceIssue"
			case "wrong source":
				r.Findings[i].Source = "service_health"
			case "wrong entity":
				r.Findings[i].EntityID += "-different"
			case "duplicate evidence":
				r.Findings[i].EvidenceIDs = []string{alert.EvidenceIDs[0], alert.EvidenceIDs[0]}
			case "invalid support":
				e := slices.IndexFunc(r.Evidence, func(e Evidence) bool { return e.ID == alert.EvidenceIDs[0] })
				r.Evidence[e].ID = "ev-" + strings.Repeat("a", 63) + "/"
				r.Findings[i].EvidenceIDs = []string{r.Evidence[e].ID}
			case "invalid deciding evidence":
				e := slices.IndexFunc(r.Evidence, func(e Evidence) bool { return e.State == "Available" })
				r.Evidence[e].ID = "ev-" + strings.Repeat("A", 64)
				r.RebuildCoverage()
				ids = []string{alert.ID, history.ID}
			case "invalid gap evidence":
				e := slices.IndexFunc(r.Evidence, func(e Evidence) bool { return e.Source == "inventory" && e.Resource.Name == "vm" })
				r.Evidence[e].ID = "ev-" + strings.Repeat("a", 63)
				r.Evidence[e].Resource.Name = ""
				r.RebuildCoverage()
			case "evidence only":
				ids[1] = r.Evidence[0].ID
			case "group heading":
				ids[1] = r.AdvisorInstances[0].TypeID
			case "assessment only":
				ids[1] = r.ResourceAssessments[0].ResourceID
			}
			assertInvalidSelection(t, r, ids)
		})
	}
}

func assertInvalidSelection(t *testing.T, r Run, ids []string) {
	t.Helper()
	got, err := r.ResolveDraftSelection(ids)
	if err == nil || len(got.Findings) != 0 {
		t.Fatalf("invalid selection returned %d resolved findings, error %v", len(got.Findings), err)
	}
}

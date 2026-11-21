package report

import (
	"bytes"
	"context"
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

func assessmentExample(t *testing.T, missingHealth, sharedIncident bool) health.Run {
	t.Helper()
	start := time.Date(2026, 11, 20, 8, 0, 0, 0, time.UTC)
	earlier, later := start.Add(-2*time.Hour), start.Add(-time.Hour)
	r := health.NewRun("Synthetic subscription", "test")
	r.ID, r.Revision, r.StartedAt = "synthetic-assessment-run", 17, start
	r.Scope = &health.Scope{SubscriptionID: sub, Name: "Synthetic subscription", Cloud: "AzureCloud"}
	status := func(name, state string, time *time.Time) health.HealthInput {
		return health.HealthInput{StatusID: resource + name + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current",
			PropertiesPresent: true, State: &state, ReportedAt: time}
	}
	alert := func(name string) health.MonitorInput {
		return health.MonitorInput{ID: "/subscriptions/" + sub + "/providers/Microsoft.AlertsManagement/alerts/" + name,
			PropertiesPresent: true, TargetResourceID: resource + name, SignalType: "Metric", Condition: "Fired", WorkflowState: "Acknowledged", Severity: "Sev2",
			LastModifiedTime: &later}
	}
	inventory := health.Page{Resources: []health.ResourceInput{
		{ID: resource + "eligible", Name: "vm | [link](https://bad.example) <script>"},
		{ID: resource + "finding", Name: "finding"},
		{ID: resource + "potential-gap", Name: "potential-gap"},
	}}
	current := health.Page{Health: []health.HealthInput{status("eligible", "Available", &later), status("finding", "Available", &later)}}
	if !missingHealth {
		current.Health = append(current.Health, status("potential-gap", "Available", &later))
	}
	service := health.Page{}
	if sharedIncident {
		service.ServiceHealth = []health.ServiceHealthInput{{ID: "/providers/Microsoft.ResourceHealth/events/shared",
			PropertiesPresent: true, EventType: "ServiceIssue", Status: "Active", LastUpdateTime: &later, ImpactStartTime: &earlier}}
	}
	advisor := health.Page{Advisor: []health.AdvisorInput{{ID: advisorPrefix + "postponed", TypeID: costType,
		Category: "Cost", Impact: "Low", Status: "Postponed", TargetResourceID: resource + "eligible", LastUpdated: &later}}}
	monitor := health.Page{Monitor: []health.MonitorInput{alert("finding"), alert("potential-gap")}}
	var collectors []health.NamedCollector
	for i, source := range []struct {
		name  string
		pages []health.Page
	}{
		{"inventory", []health.Page{inventory}},
		{"resource_health", []health.Page{{Health: []health.HealthInput{status("eligible", "Degraded", &earlier)}}, current}},
		{"service_health", []health.Page{service}},
		{"azure_monitor", []health.Page{monitor}},
		{"advisor", []health.Page{advisor}},
	} {
		collectors = append(collectors, health.NamedCollector{Name: source.name, Collect: func(_ context.Context, _ string, emit func(health.Page) error) error {
			for j, page := range source.pages {
				page.CollectedAt = start.Add(time.Duration(2*i+j+1) * time.Minute)
				if err := emit(page); err != nil {
					return err
				}
			}
			return nil
		}})
	}
	if code, err := health.Check(context.Background(), &r, collectors, func(*health.Run) error { return nil }); code != 0 || err != nil {
		t.Fatalf("synthetic collection failed: %d %v", code, err)
	}
	for _, item := range collectors {
		prepared := r.Sources[item.Name].FirstPageAt.Add(-time.Second)
		r.Sources[item.Name].Query.RequestedAt = &prepared
	}
	r.UpdatedAt = start.Add(12 * time.Minute)
	r.EndedAt = &r.UpdatedAt
	r.RebuildCoverage()
	return r
}

func TestSchema4AssessmentReport(t *testing.T) {
	for _, tc := range []struct {
		name, headline          string
		missingHealth, incident bool
	}{
		{"mixed", "Incomplete coverage · 1 resource OK, 1 with findings, 1 unknown", true, false},
		{"complete", "Complete coverage · 1 resource OK, 2 with findings, 0 unknown", false, false},
		{"shared incident", "Incomplete coverage · 0 resources OK, 0 with findings, 3 unknown", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := assessmentExample(t, tc.missingHealth, tc.incident)
			b, err := Markdown(r)
			if err != nil {
				t.Fatal(err)
			}
			text := html.UnescapeString(string(b))
			for _, want := range []string{"**" + tc.headline + ".**", "OK within the agreed checks", "vm | [link](https://bad.example) <script>",
				"Configured API timeRange: 30d", "Rolling 30-day lower cutoff (UTC): 2026-10-21T08:00:00Z", "Cached list scope",
				"Advisor Postponed:", "2026-11-20T08:05:00Z"} {
				if !strings.Contains(text, want) {
					t.Fatalf("report omitted %q", want)
				}
			}
			if strings.Contains(string(b), "<script>") || strings.Contains(string(b), "[link](https://bad.example)") {
				t.Fatal("source text rendered as markup")
			}
			_, sections, hasGaps := strings.Cut(text, "\n## Coverage gaps and failed checks\n")
			gaps, notes, hasNotes := strings.Cut(sections, "\n## Source limits and evidence notes\n")
			notes, _, _ = strings.Cut(notes, "\n## Execution diagnostics\n")
			if !hasGaps || !hasNotes || strings.Contains(gaps, "monitor_30_day_retention_and_stateless_limits") ||
				!strings.Contains(notes, "monitor_30_day_retention_and_stateless_limits") {
				t.Fatal("standing query limits were not distinguished from blocking gaps")
			}
			switch {
			case tc.missingHealth:
				if !strings.Contains(gaps, "missing_health_evidence") || strings.Contains(notes, "missing_health_evidence") {
					t.Fatal("missing health evidence should appear among the blocking gaps")
				}
			case tc.incident:
				if !strings.Contains(gaps, "service_health_impact_undetermined") || strings.Contains(notes, "service_health_impact_undetermined") {
					t.Fatal("unknown incident impact should appear among the blocking gaps")
				}
			default:
				if !strings.Contains(gaps, "No decision-blocking gaps recorded.") {
					t.Fatal("complete report did not state that there were no blocking gaps")
				}
			}
			row := func(name string) string {
				t.Helper()
				prefix := "| " + health.CanonicalID(resource+name) + " | "
				for line := range strings.SplitSeq(text, "\n") {
					if strings.HasPrefix(line, prefix+"OK | ") || strings.HasPrefix(line, prefix+"Unknown | ") || strings.HasPrefix(line, prefix+"Finding | ") {
						return line
					}
				}
				t.Fatalf("missing assessment row for %s", name)
				return ""
			}
			eligible := row("eligible")
			for _, name := range assessmentSourceNames {
				ref := "sources." + name
				if !strings.Contains(eligible, "["+ref+"](run.json)") || !r.HasAssessmentSourceProof(name) {
					t.Fatalf("resource check lacks resolvable source proof for %s", name)
				}
			}
			for _, e := range r.Evidence {
				if strings.EqualFold(e.ResourceID, resource+"eligible") && e.Source == "resource_health" {
					if strings.Contains(eligible, e.ID) != (e.State == "Available") {
						t.Fatalf("deciding cell cited wrong Resource Health version: %s", e.State)
					}
				}
			}
			for _, f := range r.Findings {
				switch f.Source {
				case "":
					if !strings.Contains(eligible, "Historical:") || !strings.Contains(eligible, "["+f.ID+"](run.json)") ||
						!strings.Contains(text, fmt.Sprintf("| %s | [%s](run.json) |", f.ID, f.EvidenceIDs[0])) {
						t.Fatal("older Resource Health finding is no longer traceable to original evidence")
					}
				case "azure_monitor":
					if strings.EqualFold(f.ResourceID, resource+"potential-gap") {
						line := row("potential-gap")
						if !strings.Contains(line, f.ID) || strings.Contains(line, "missing_health_evidence") != tc.missingHealth {
							t.Fatal("known alert and missing proof did not appear together")
						}
					}
				case "service_health":
					anchor := incidentAnchor(f.ID)
					if strings.Count(text, `id="`+anchor+`"`) != 1 || strings.Count(text, "](#"+anchor+")") != 3 {
						t.Fatal("subscription incident lacked one shared detail entry")
					}
				}
			}
		})
	}
}

func TestSchema4RejectsUnsupportedAssessmentReferences(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change func(*health.Run, *health.ResourceAssessment)
	}{
		{"missing source", func(r *health.Run, _ *health.ResourceAssessment) { delete(r.Sources, "azure_monitor") }},
		{"wrong scope", func(r *health.Run, _ *health.ResourceAssessment) {
			r.Sources["azure_monitor"].Query.SubscriptionID = "synthetic-other"
		}},
		{"wrong operation", func(r *health.Run, _ *health.ResourceAssessment) {
			r.Sources["azure_monitor"].Query.Operation = "Events.ListBySubscriptionId"
		}},
		{"wrong window", func(r *health.Run, _ *health.ResourceAssessment) { r.Sources["azure_monitor"].Query.TimeRange = "1h" }},
		{"prepared only", func(r *health.Run, _ *health.ResourceAssessment) {
			s := r.Sources["azure_monitor"]
			s.Pages, s.EnumerationComplete = 0, false
			r.Sources["azure_monitor"] = s
		}},
		{"duplicate reference", func(_ *health.Run, a *health.ResourceAssessment) { a.Checks[3].SourceRef = a.Checks[2].SourceRef }},
		{"wrong source evidence", func(_ *health.Run, a *health.ResourceAssessment) { a.Checks[3].EvidenceIDs = a.Checks[0].EvidenceIDs }},
		{"Monitor evidence for another resource", func(r *health.Run, a *health.ResourceAssessment) {
			i := slices.IndexFunc(r.Evidence, func(e health.Evidence) bool { return e.Source == "azure_monitor" })
			a.Checks[3].EvidenceIDs = []string{r.Evidence[i].ID}
		}},
		{"Postponed recommendation for another resource", func(r *health.Run, a *health.ResourceAssessment) {
			i := slices.IndexFunc(r.ResourceAssessments, func(a health.ResourceAssessment) bool { return a.Disposition == "finding" })
			other := &r.ResourceAssessments[i]
			other.Checks[4].EvidenceIDs = a.Checks[4].EvidenceIDs
			other.PostponedEvidenceIDs = a.PostponedEvidenceIDs
		}},
		{"current finding omitted", func(r *health.Run, _ *health.ResourceAssessment) {
			i := slices.IndexFunc(r.ResourceAssessments, func(a health.ResourceAssessment) bool { return a.Disposition == "finding" })
			r.ResourceAssessments[i].Disposition, r.ResourceAssessments[i].FindingIDs = "ok", nil
			r.AssessmentSummary.OK++
			r.AssessmentSummary.Finding--
		}},
		{"current finding moved to history", func(r *health.Run, _ *health.ResourceAssessment) {
			i := slices.IndexFunc(r.ResourceAssessments, func(a health.ResourceAssessment) bool { return a.Disposition == "finding" })
			other := &r.ResourceAssessments[i]
			other.Disposition, other.HistoricalFindingIDs, other.FindingIDs = "ok", other.FindingIDs, nil
			r.AssessmentSummary.OK++
			r.AssessmentSummary.Finding--
		}},
		{"unresolved history", func(_ *health.Run, a *health.ResourceAssessment) { a.HistoricalFindingIDs[0] = "missing" }},
		{"unresolved postponed", func(_ *health.Run, a *health.ResourceAssessment) { a.PostponedEvidenceIDs[0] = "missing" }},
		{"bad counts", func(r *health.Run, _ *health.ResourceAssessment) { r.AssessmentSummary.Unknown++ }},
		{"zero end time", func(r *health.Run, _ *health.ResourceAssessment) { r.EndedAt = &time.Time{} }},
		{"failed affirmative", func(r *health.Run, _ *health.ResourceAssessment) { r.Outcome, r.Coverage = "failed", "incomplete" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := assessmentExample(t, false, false)
			i := slices.IndexFunc(r.ResourceAssessments, func(a health.ResourceAssessment) bool { return a.Disposition == "ok" })
			if i < 0 {
				t.Fatal("test requires an eligible resource")
			}
			tc.change(&r, &r.ResourceAssessments[i])
			if b, err := Markdown(r); err == nil || len(b) != 0 {
				t.Fatal("invalid assessments were published")
			}
		})
	}
}

func TestSchema4ReportShowsLatestReadOfRepeatedMaintenance(t *testing.T) {
	r := assessmentExample(t, false, false)
	first := r.StartedAt.Add(6 * time.Minute)
	last := r.StartedAt.Add(8 * time.Minute)
	impactsAt := r.StartedAt.Add(7 * time.Minute)
	event := health.ServiceHealthInput{ID: "/providers/Microsoft.ResourceHealth/events/repeated-maintenance", PropertiesPresent: true,
		EventType: "PlannedMaintenance", Status: "Active", LastUpdateTime: &first, ImpactStartTime: &impactsAt}
	for _, collected := range []time.Time{first, last} {
		r.NormalizePage("service_health", health.Page{ServiceHealth: []health.ServiceHealthInput{event}, CollectedAt: collected})
	}
	b, err := Markdown(r)
	if err != nil {
		t.Fatal(err)
	}
	markdown := html.UnescapeString(string(b))
	for _, want := range []string{
		"Incomplete coverage · 0 resources OK, 0 with findings, 3 unknown",
		"Service Health Active PlannedMaintenance notice",
		"Collected (UTC): " + first.Format(time.RFC3339),
		"Last collected (UTC): " + last.Format(time.RFC3339),
		"Evidence collected (UTC): " + first.Format(time.RFC3339) + " to " + last.Format(time.RFC3339),
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("report omitted %q", want)
		}
	}
}

func TestSchema4RejectsInconsistentBlockingReasons(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*health.Run, *health.ResourceAssessment)
	}{
		{"missing subscription blocker", func(r *health.Run, a *health.ResourceAssessment) {
			a.Disposition, a.BlockingReasons = "ok", nil
			r.AssessmentSummary.OK++
			r.AssessmentSummary.Unknown--
		}},
		{"blocker missing from saved gaps", func(r *health.Run, _ *health.ResourceAssessment) {
			r.Gaps = slices.DeleteFunc(r.Gaps, func(g health.Gap) bool { return g.Blocking && g.Source == "service_health" })
		}},
		{"blocker changed to another resource", func(_ *health.Run, a *health.ResourceAssessment) {
			a.BlockingReasons[0].ResourceID = resource + "finding"
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := assessmentExample(t, false, true)
			i := slices.IndexFunc(r.ResourceAssessments, func(a health.ResourceAssessment) bool {
				return strings.EqualFold(a.ResourceID, resource+"eligible")
			})
			if i < 0 || len(r.ResourceAssessments[i].BlockingReasons) != 1 {
				t.Fatal("test requires one blocking subscription incident")
			}
			test.change(&r, &r.ResourceAssessments[i])
			if b, err := Markdown(r); err == nil || len(b) != 0 {
				t.Fatal("report was published with inconsistent blocking reasons")
			}
		})
	}
}

func TestSchema4RejectsNondecidingResourceHealthReferences(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*health.Run, *health.AssessmentCheck, string)
	}{
		{"superseded Available", func(_ *health.Run, check *health.AssessmentCheck, older string) {
			check.EvidenceIDs = []string{older}
		}},
		{"superseded Available with newer evidence omitted from observation", func(r *health.Run, check *health.AssessmentCheck, older string) {
			newer := check.EvidenceIDs[0]
			check.EvidenceIDs = []string{older}
			for i := range r.Observations {
				r.Observations[i].EvidenceIDs = slices.DeleteFunc(r.Observations[i].EvidenceIDs, func(id string) bool { return id == newer })
			}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			r := assessmentExample(t, false, false)
			older := r.StartedAt.Add(-3 * time.Hour)
			available := "Available"
			r.NormalizePage("resource_health", health.Page{
				CollectedAt: r.Sources["resource_health"].LastPageAt.Add(time.Second),
				Health: []health.HealthInput{{StatusID: resource + "eligible/providers/Microsoft.ResourceHealth/availabilityStatuses/current",
					PropertiesPresent: true, State: &available, ReportedAt: &older}},
			})
			if _, err := Markdown(r); err != nil {
				t.Fatalf("report rejected the deciding Available evidence: %v", err)
			}
			olderIndex := slices.IndexFunc(r.Evidence, func(e health.Evidence) bool {
				return e.Source == "resource_health" && e.ReportedAt != nil && e.ReportedAt.Equal(older)
			})
			assessment := slices.IndexFunc(r.ResourceAssessments, func(a health.ResourceAssessment) bool {
				return strings.EqualFold(a.ResourceID, resource+"eligible") && a.Disposition == "ok"
			})
			if olderIndex < 0 || assessment < 0 {
				t.Fatal("test requires the old Available evidence and an OK assessment")
			}
			checks := r.ResourceAssessments[assessment].Checks
			i := slices.IndexFunc(checks, func(c health.AssessmentCheck) bool { return c.SourceRef == "sources.resource_health" })
			if i < 0 || len(checks[i].EvidenceIDs) != 1 || checks[i].EvidenceIDs[0] == r.Evidence[olderIndex].ID {
				t.Fatal("test requires a newer deciding Available observation")
			}
			test.change(&r, &checks[i], r.Evidence[olderIndex].ID)
			if b, err := Markdown(r); err == nil || len(b) != 0 {
				t.Fatal("report published a superseded Resource Health citation as proof")
			}
		})
	}
}

func TestSchema4AcceptsAdvisorResourceGroupContext(t *testing.T) {
	r := assessmentExample(t, false, false)
	r.NormalizePage("advisor", health.Page{CollectedAt: r.UpdatedAt, Advisor: []health.AdvisorInput{{
		ID: advisorPrefix + "postponed-group", TypeID: costType, Category: "Cost", Impact: "Low", Status: "Postponed",
		TargetResourceID: "/subscriptions/" + sub + "/resourceGroups/G",
	}}})
	b, err := Markdown(r)
	if err != nil {
		t.Fatal(err)
	}
	i := slices.IndexFunc(r.Evidence, func(e health.Evidence) bool { return strings.HasSuffix(e.EntityID, "/postponed-group") })
	if i < 0 {
		t.Fatal("expected accepted group recommendation")
	}
	contextRows := 0
	for line := range strings.SplitSeq(string(b), "\n") {
		_, context, ok := strings.Cut(line, "Advisor Postponed: ")
		if ok && strings.Contains(context, "["+r.Evidence[i].ID+"](run.json)") {
			contextRows++
		}
	}
	if contextRows != len(r.Resources) || r.Coverage != "complete" {
		t.Fatal("group-scoped Postponed context did not appear on every matching assessment")
	}
}

func TestSyntheticAssessmentReportExample(t *testing.T) {
	r := assessmentExample(t, true, false)
	snapshot, err := os.ReadFile("testdata/assessment/run.json")
	if err != nil {
		t.Fatal(err)
	}
	expectedSnapshot, err := json.MarshalIndent(r, "", "  ")
	if err != nil || !bytes.Equal(append(expectedSnapshot, '\n'), snapshot) {
		t.Fatal("schema-4 fixture snapshot differs from synthetic example")
	}
	var saved health.Run
	if err := json.Unmarshal(snapshot, &saved); err != nil {
		t.Fatal(err)
	}
	rendered, err := Markdown(saved)
	if err != nil {
		t.Fatal(err)
	}
	markdown, err := os.ReadFile("testdata/assessment/report.md")
	if err != nil || !bytes.Equal(rendered, markdown) {
		t.Fatal("schema-4 report fixture differs from its sibling snapshot")
	}
}

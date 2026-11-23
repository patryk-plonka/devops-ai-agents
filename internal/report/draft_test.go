package report

import (
	"bytes"
	"html"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

const draftTenant = "33333333-3333-3333-3333-333333333333"

func draftExample() health.Run {
	r := health.NewRun("private-subscription-name", "test")
	r.Scope = &health.Scope{SubscriptionID: sub, TenantID: draftTenant, Cloud: "AzureCloud"}
	r.ID, r.Revision, r.Outcome = "20261123T150405.000000000Z-draft-fixture", 7, "cancelled"
	now := time.Date(2026, 11, 23, 15, 4, 5, 0, time.UTC)
	r.StartedAt, r.UpdatedAt, r.EndedAt = now, now, &now
	service, monitor, advisor := health.ServiceHealthQuery(now), health.MonitorQuery(now), health.AdvisorQuery(now)
	service.Operation, service.APIVersion, service.SubscriptionID = "Events.ListBySubscriptionId", health.ServiceHealthAPI, sub
	monitor.Operation, monitor.APIVersion, monitor.SubscriptionID = "Alerts.GetAll", health.MonitorAPI, sub
	monitor.MonitorCondition = "Fired"
	advisor.SubscriptionID = sub
	r.Sources["service_health"] = health.Source{Status: "completed", Query: &service}
	r.Sources["azure_monitor"] = health.Source{Status: "completed", Query: &monitor}
	r.Sources["advisor"] = health.Source{Status: "completed", Query: &advisor}
	r.Sources["resource_health"] = health.Source{Status: "partial", Reason: "interrupted"}
	return r
}

func TestDraftMarkdownSelectedFacts(t *testing.T) {
	r := draftExample()
	now := r.StartedAt
	before, later := now.Add(-3*time.Hour), now.Add(-time.Hour)
	workspace := "/subscriptions/" + sub + "/resourceGroups/g/providers/Microsoft.OperationalInsights/workspaces/target"
	cachedTarget := resource + "advisor-target"
	privateTag := "private-tag-marker"
	r.NormalizePage("inventory", health.Page{CollectedAt: now, Resources: []health.ResourceInput{
		{ID: workspace, Name: "workspace *literal*", Type: "Microsoft.OperationalInsights/workspaces", Tags: map[string]*string{"owner": &privateTag}},
		{ID: cachedTarget, Name: "saved advisor target", Type: "Microsoft.Compute/virtualMachines"},
		{ID: resource + "other", Name: "unrelated-marker", Type: "Microsoft.Compute/virtualMachines"},
	}})
	r.NormalizePage("resource_health", health.Page{CollectedAt: now, Health: []health.HealthInput{
		draftHealthInput(resource+"outside-inventory", "Degraded", &before),
		draftHealthInput(resource+"outside-inventory", "Available", &later),
	}})
	r.NormalizePage("service_health", health.Page{CollectedAt: now, ServiceHealth: []health.ServiceHealthInput{{
		ID: "/providers/Microsoft.ResourceHealth/events/notice-entity", PropertiesPresent: true,
		EventType: "ServiceIssue", Status: "Active", LastUpdateTime: &later, ImpactStartTime: &before,
		Impact: []health.ServiceImpactInput{{Service: "Virtual Machines", Regions: []health.ServiceRegionInput{
			{Name: "West Europe", Status: "Active", LastUpdateTime: &later, ImpactedSubscriptions: []string{sub}},
			{Name: "unverified-region-marker", Status: "Active"},
		}}},
	}}})
	suppressed := true
	r.NormalizePage("azure_monitor", health.Page{CollectedAt: now, Monitor: []health.MonitorInput{{
		ID: "/subscriptions/" + sub + "/providers/Microsoft.AlertsManagement/alerts/monitor-entity", PropertiesPresent: true,
		TargetResourceID: workspace, SignalType: "Log", Condition: "Fired", WorkflowState: "Acknowledged", IsSuppressed: &suppressed,
		LastModifiedTime: &later, StartTime: &before,
	}}})
	r.NormalizePage("advisor", health.Page{CollectedAt: now, Advisor: []health.AdvisorInput{{
		ID: cachedTarget + "/providers/Microsoft.Advisor/recommendations/advisor-entity", TargetResourceID: cachedTarget,
		Category: "Cost", Impact: "Medium", Status: "New", LastRefreshed: &later,
	}}})
	r.RebuildCoverage()

	cases := []struct {
		category string
		want     []string
	}{
		{"resource_health_issue", []string{"Affected resource: resource name unknown", "Degraded. Source reported 2026-11-23T12:04:05Z", "observation is historical", "Available. Source reported 2026-11-23T14:04:05Z", "collected 2026-11-23T15:04:05Z"}},
		{"service_health_ServiceIssue", []string{"category ServiceIssue, status Active", "impact start 2026-11-23T12:04:05Z", "fixed 30-day update window", "2026-10-24T15:04:05Z", "Virtual Machines", "West Europe", "did not enumerate affected resources"}},
		{"monitor_alert", []string{"Azure Monitor reported target", "workspace *literal*", "Microsoft.OperationalInsights/workspaces", "workspace observation only", "Signal Log, condition Fired, workflow Acknowledged, suppressed yes", "30-day Fired view", "Resource Health check did not complete"}},
		{"advisor_recommendation", []string{"workflow New, category Cost, Advisor impact Medium", "Advisor assessed target: resource name saved advisor target", "Source evaluation 2026-11-23T14:04:05Z", "cached recommendation list", "no date filter or refresh"}},
	}
	for _, tc := range cases {
		t.Run(tc.category, func(t *testing.T) {
			selection, item := draftSelection(t, r, tc.category)
			for _, kind := range []DraftType{Ticket, Recommendation} {
				got := draftBytes(t, kind, selection, item)
				want := append(slices.Clone(tc.want), "Run ID: `"+r.ID+"`", "Revision: 7",
					"Run outcome: cancelled. Overall coverage: incomplete.", "Finding ID: `"+item.Finding.ID+"`")
				if kind == Ticket {
					want = append(want, "# Ticket:", "Observed condition and impact", "Next verification step")
				} else {
					want = append(want, "# Recommendation:", "Reason for review", "Supporting facts", "Proposed review")
				}
				for _, e := range item.Supporting {
					want = append(want, e.ID)
				}
				if item.CurrentHealth != nil {
					want = append(want, item.CurrentHealth.Observations[0].ID)
				}
				assertDraftIncludes(t, html.UnescapeString(string(got)), want...)
				for _, bad := range []string{sub, draftTenant, "/subscriptions/", "private-subscription-name", privateTag,
					"unrelated-marker", "unverified-region-marker", "notice-entity", "monitor-entity", "advisor-entity", "workspace *literal*", "](run.json)"} {
					if bytes.Contains(got, []byte(bad)) {
						t.Errorf("%s exposed %q", kind, bad)
					}
				}
				for _, other := range r.Findings {
					if other.ID != item.Finding.ID && bytes.Contains(got, []byte(other.ID)) {
						t.Errorf("%s exposed an unselected finding", kind)
					}
				}
			}
		})
	}
}

func TestDraftMarkdownWithholdsSensitiveTargetLabels(t *testing.T) {
	for _, names := range [][]string{{"Resource " + resource + "private-ARM-marker"}, {"user Password=credential-marker"},
		{"site /private/tmp/local-marker"}, {"tenant " + draftTenant}, {"conflict-one", "conflict-two"}} {
		r := draftExample()
		target := resource + "verified"
		for _, name := range names {
			r.NormalizePage("inventory", health.Page{CollectedAt: r.StartedAt,
				Resources: []health.ResourceInput{{ID: target, Name: name, Type: "Microsoft.Compute/virtualMachines"}}})
		}
		r.NormalizePage("resource_health", health.Page{CollectedAt: r.StartedAt,
			Health: []health.HealthInput{draftHealthInput(target, "Degraded", nil)}})
		r.RebuildCoverage()
		selection, item := draftSelection(t, r, "resource_health_issue")
		got := html.UnescapeString(string(draftBytes(t, Ticket, selection, item)))
		assertDraftIncludes(t, got, "resource name unknown", "type Microsoft.Compute/virtualMachines", "Source reported unknown")
		for _, value := range names {
			if strings.Contains(got, value) {
				t.Errorf("unsafe or conflicting name %q", value)
			}
		}
	}
}

func TestDraftMarkdownUndeterminedHealth(t *testing.T) {
	for _, missing := range []bool{false, true} {
		r := draftExample()
		target := resource + "outside-inventory"
		available := draftHealthInput(target, "Available", &r.StartedAt)
		want := "The latest source observations disagree."
		if missing {
			available.ReportedAt = nil
			want = "at least one source timestamp is missing."
		}
		r.NormalizePage("resource_health", health.Page{CollectedAt: r.StartedAt, Health: []health.HealthInput{
			draftHealthInput(target, "Degraded", &r.StartedAt), available,
		}})
		r.RebuildCoverage()
		selection, item := draftSelection(t, r, "resource_health_issue")
		got := string(draftBytes(t, Ticket, selection, item))
		assertDraftIncludes(t, got, "Current state is unknown.", want, "Available.", "Degraded.")
		if strings.Contains(got, "later deciding evidence") {
			t.Fatal("unresolved conflict was described as historical recovery")
		}
	}
}

func TestDraftMarkdownQualifiesConflictingOperationalStates(t *testing.T) {
	for _, tc := range []struct {
		category, conflict, normal, unresolved string
		add                                    func(*health.Run, time.Time, bool)
	}{
		{
			"monitor_alert", "Azure Monitor alert with conflicting conditions", "Azure Monitor Fired alert", "alert condition unresolved",
			func(r *health.Run, otherTime time.Time, agrees bool) {
				input := health.MonitorInput{ID: "/subscriptions/" + sub + "/providers/Microsoft.AlertsManagement/alerts/tied",
					PropertiesPresent: true, TargetResourceID: resource + "vm", SignalType: "Metric", Condition: "Fired",
					WorkflowState: "New", LastModifiedTime: &r.StartedAt}
				other := input
				other.LastModifiedTime, other.WorkflowState = &otherTime, "Closed"
				if !agrees {
					other.Condition = "Resolved"
				}
				r.NormalizePage("azure_monitor", health.Page{CollectedAt: r.StartedAt, Monitor: []health.MonitorInput{input, other}})
			},
		},
		{
			"service_health_ServiceIssue", "Azure Service Health notice with conflicting statuses", "Azure Service Health active notice", "notice status unresolved",
			func(r *health.Run, otherTime time.Time, agrees bool) {
				input := health.ServiceHealthInput{ID: "/providers/Microsoft.ResourceHealth/events/tied", PropertiesPresent: true,
					EventType: "ServiceIssue", Status: "Active", LastUpdateTime: &r.StartedAt}
				other := input
				other.LastUpdateTime, other.Level = &otherTime, "Warning"
				if !agrees {
					other.Status = "Resolved"
				}
				r.NormalizePage("service_health", health.Page{CollectedAt: r.StartedAt, ServiceHealth: []health.ServiceHealthInput{input, other}})
			},
		},
	} {
		t.Run(tc.category, func(t *testing.T) {
			for _, scenario := range []struct {
				name   string
				offset time.Duration
				agrees bool
			}{
				{name: "opposite states at same time"},
				{name: "older opposite state", offset: -time.Hour},
				{name: "same state with different metadata", agrees: true},
			} {
				t.Run(scenario.name, func(t *testing.T) {
					r := draftExample()
					tc.add(&r, r.StartedAt.Add(scenario.offset), scenario.agrees)
					r.RebuildCoverage()
					selection, item := draftSelection(t, r, tc.category)
					wantConflict := scenario.offset == 0 && !scenario.agrees
					for _, kind := range []DraftType{Ticket, Recommendation} {
						body := string(draftBytes(t, kind, selection, item))
						if strings.Contains(body, tc.unresolved) != wantConflict {
							t.Fatalf("%s conflict wording does not reflect the deciding observations:\n%s", kind, body)
						}
						if kind == Ticket && wantConflict {
							assertDraftIncludes(t, body, "# Ticket: "+tc.conflict)
						}
						if kind == Ticket && !wantConflict {
							assertDraftIncludes(t, body, "# Ticket: "+tc.normal)
						}
						if !scenario.agrees {
							assertDraftIncludes(t, body, "Resolved", item.Supporting[0].ID, item.Context[0].ID)
						}
					}
				})
			}
		})
	}
}

func TestDraftMarkdownQualifiesPlannedMaintenanceTiming(t *testing.T) {
	for _, tc := range []struct {
		name          string
		collected     time.Duration
		wantScheduled bool
	}{
		{name: "future at observation", wantScheduled: true},
		{name: "started before collection", collected: 2 * time.Hour},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := draftExample()
			start := r.StartedAt.Add(time.Hour)
			r.NormalizePage("service_health", health.Page{CollectedAt: r.StartedAt.Add(tc.collected), ServiceHealth: []health.ServiceHealthInput{{
				ID: "/providers/Microsoft.ResourceHealth/events/maintenance", PropertiesPresent: true,
				EventType: "PlannedMaintenance", Status: "Active", LastUpdateTime: &r.StartedAt, ImpactStartTime: &start,
			}}})
			r.RebuildCoverage()
			selection, item := draftSelection(t, r, "service_health_PlannedMaintenance")
			for _, kind := range []DraftType{Ticket, Recommendation} {
				body := string(draftBytes(t, kind, selection, item))
				if strings.Contains(body, "the planned maintenance was scheduled to start later") != tc.wantScheduled {
					t.Errorf("%s did not reflect the timing at collection:\n%s", kind, body)
				}
				assertDraftIncludes(t, body, "Notice category PlannedMaintenance, status Active", "impact start "+start.Format(time.RFC3339Nano), item.Supporting[0].ID)
			}
		})
	}
}

func TestDraftMarkdownRespectsUnknownOperationalValues(t *testing.T) {
	r := draftExample()
	older := r.StartedAt.Add(-time.Hour)
	notice := health.ServiceHealthInput{ID: "/providers/Microsoft.ResourceHealth/events/notice", PropertiesPresent: true,
		EventType: "ServiceIssue", Status: "Active", LastUpdateTime: &r.StartedAt,
		Impact: []health.ServiceImpactInput{{Service: "Virtual Machines", Regions: []health.ServiceRegionInput{
			{Name: "West Europe", Status: "Resol\x00ved", ImpactedSubscriptions: []string{sub}},
		}}},
	}
	olderNotice := notice
	olderNotice.EventType, olderNotice.Status, olderNotice.LastUpdateTime = "ServiceIss\x00ue", "Resol\x00ved", &older
	olderNotice.Impact = nil
	r.NormalizePage("service_health", health.Page{CollectedAt: r.StartedAt, ServiceHealth: []health.ServiceHealthInput{notice, olderNotice}})

	alert := health.MonitorInput{ID: "/subscriptions/" + sub + "/providers/Microsoft.AlertsManagement/alerts/alert",
		PropertiesPresent: true, TargetResourceID: resource + "vm", SignalType: "Metric", Condition: "Fired",
		WorkflowState: "Ne\x00w", LastModifiedTime: &r.StartedAt}
	olderAlert := alert
	olderAlert.SignalType, olderAlert.Condition, olderAlert.WorkflowState = "Lo\x00g", "Resol\x00ved", "Closed"
	olderAlert.LastModifiedTime = &older
	r.NormalizePage("azure_monitor", health.Page{CollectedAt: r.StartedAt, Monitor: []health.MonitorInput{alert, olderAlert}})
	r.RebuildCoverage()

	for _, tc := range []struct {
		category   string
		want, omit []string
	}{
		{
			"service_health_ServiceIssue",
			[]string{"category ServiceIssue, status Active", "West Europe. Status unknown", "category unknown, status unknown"},
			[]string{"Status Resolved", "category ServiceIssue, status Resolved"},
		},
		{
			"monitor_alert",
			[]string{"Signal Metric, condition Fired, workflow unknown", "Signal unknown, condition unknown, workflow Closed"},
			[]string{"workflow New", "Signal Log", "condition Resolved"},
		},
	} {
		t.Run(tc.category, func(t *testing.T) {
			selection, item := draftSelection(t, r, tc.category)
			for _, kind := range []DraftType{Ticket, Recommendation} {
				body := string(draftBytes(t, kind, selection, item))
				assertDraftIncludes(t, body, tc.want...)
				for _, value := range tc.omit {
					if strings.Contains(body, value) {
						t.Errorf("%s presents an unknown value as %q", kind, value)
					}
				}
			}
		})
	}
}

func TestDraftMarkdownQualifiesUnverifiedQueries(t *testing.T) {
	r := draftExample()
	now := r.StartedAt
	cutoff := now.Add(-24 * time.Hour)
	r.Sources["service_health"].Query.Cutoff = &cutoff
	r.Sources["azure_monitor"].Query.SubscriptionID = draftTenant
	r.Sources["advisor"].Query.APIVersion = ""
	target := resource + "vm"
	r.NormalizePage("service_health", health.Page{CollectedAt: now, ServiceHealth: []health.ServiceHealthInput{{
		ID: "/providers/Microsoft.ResourceHealth/events/notice", PropertiesPresent: true,
		EventType: "ServiceIssue", Status: "Active", LastUpdateTime: &now,
	}}})
	r.NormalizePage("azure_monitor", health.Page{CollectedAt: now, Monitor: []health.MonitorInput{{
		ID:                "/subscriptions/" + sub + "/providers/Microsoft.AlertsManagement/alerts/alert",
		PropertiesPresent: true, TargetResourceID: target, SignalType: "Metric", Condition: "Fired",
	}}})
	r.NormalizePage("advisor", health.Page{CollectedAt: now, Advisor: []health.AdvisorInput{{
		ID: target + "/providers/Microsoft.Advisor/recommendations/recommendation", TargetResourceID: target,
		Category: "Cost", Impact: "Medium", Status: "New",
	}}})
	r.RebuildCoverage()
	markdown, err := Markdown(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateReviewedMarkdown(r, markdown); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		category, standard, verifiedWording string
	}{
		{"service_health_ServiceIssue", "standard query uses a fixed 30-day update window", "The query uses a fixed 30-day update window from run start"},
		{"monitor_alert", "standard view covers 30-day Fired alerts", "Query scope: 30-day Fired view"},
		{"advisor_recommendation", "standard query reads the cached recommendation list", "Scope: cached recommendation list for the checked subscription"},
	} {
		t.Run(tc.category, func(t *testing.T) {
			selection, item := draftSelection(t, r, tc.category)
			for _, kind := range []DraftType{Ticket, Recommendation} {
				body := string(draftBytes(t, kind, selection, item))
				assertDraftIncludes(t, body, "Saved query scope could not be verified", tc.standard, "unverified query policy")
				if tc.category == "service_health_ServiceIssue" {
					assertDraftIncludes(t, body, "Recorded query reference "+now.Format(time.RFC3339)+" and update cutoff "+cutoff.Format(time.RFC3339))
				}
				if strings.Contains(body, tc.verifiedWording) || strings.Contains(body, draftTenant) {
					t.Errorf("%s presents unverified or private query data:\n%s", kind, body)
				}
			}
		})
	}
}

func TestDraftMarkdownAdvisorAgreementAndContext(t *testing.T) {
	target := resource + "advisor-example"
	r := draftExample()
	before := r.StartedAt.Add(-time.Hour)
	low := health.AdvisorInput{ID: target + "/providers/Microsoft.Advisor/recommendations/synthetic", TargetResourceID: target,
		Category: "Cost", Impact: "Low", Status: "New", LastUpdated: r.EndedAt}
	high, previous := low, low
	high.Impact = "High"
	previous.Status, previous.LastUpdated = "Dismissed", &before
	input := []health.AdvisorInput{low, high, previous}
	expected := map[DraftType][]byte{}
	for _, reverse := range []bool{false, true} {
		r := draftExample()
		if reverse {
			slices.Reverse(input)
		}
		r.NormalizePage("advisor", health.Page{CollectedAt: r.StartedAt, Advisor: input})
		r.RebuildCoverage()
		selection, item := draftSelection(t, r, "advisor_recommendation")
		if len(item.Supporting) != 2 || len(item.Context) != 1 {
			t.Fatalf("wanted two supporting rows and the dismissed context, got %v", item.Finding.EvidenceIDs)
		}
		for _, kind := range []DraftType{Ticket, Recommendation} {
			got := draftBytes(t, kind, selection, item)
			assertDraftIncludes(t, string(got), "workflow New", "impact unknown because the observations disagree on impact",
				item.Supporting[0].ID, item.Supporting[1].ID, "Advisor workflow Dismissed", item.Context[0].ID)
			if reverse && !bytes.Equal(got, expected[kind]) {
				t.Errorf("%s changed when observation order reversed", kind)
			}
			expected[kind] = got
		}
	}
}

func draftHealthInput(target, state string, reported *time.Time) health.HealthInput {
	return health.HealthInput{StatusID: target + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current",
		PropertiesPresent: true, State: &state, ReportedAt: reported}
}

func draftSelection(t *testing.T, r health.Run, category string) (health.DraftSelection, health.SelectedFinding) {
	t.Helper()
	for _, f := range r.Findings {
		if f.Category == category {
			s, err := r.ResolveDraftSelection([]string{f.ID})
			if err != nil {
				t.Fatal(err)
			}
			return s, s.Findings[0]
		}
	}
	t.Fatalf("missing %s finding", category)
	return health.DraftSelection{}, health.SelectedFinding{}
}

func draftBytes(t *testing.T, kind DraftType, selection health.DraftSelection, item health.SelectedFinding) []byte {
	t.Helper()
	got, err := DraftMarkdown(kind, selection, item)
	if err != nil {
		t.Fatal(err)
	}
	return got
}

func assertDraftIncludes(t *testing.T, got string, want ...string) {
	t.Helper()
	for _, value := range want {
		if !strings.Contains(got, value) {
			t.Errorf("draft missing %q\n%s", value, got)
		}
	}
}

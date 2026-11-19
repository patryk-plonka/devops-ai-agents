package health

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"
)

var operationalReference = time.Date(2026, 11, 15, 14, 23, 45, 678, time.UTC)

const eventIdentity = "/providers/Microsoft.ResourceHealth/events/synthetic-event"
const alertIdentity = "/subscriptions/" + testSubscription + "/providers/Microsoft.AlertsManagement/alerts/synthetic-alert"

func operationalRun() Run {
	r := scopedRun()
	r.StartedAt = operationalReference
	sh, monitor := ServiceHealthQuery(r.StartedAt), MonitorQuery(r.StartedAt)
	r.Sources["service_health"] = Source{Status: "in_progress", Query: &sh}
	r.Sources["azure_monitor"] = Source{Status: "in_progress", Query: &monitor}
	return r
}
func serviceInput() ServiceHealthInput {
	return ServiceHealthInput{ID: eventIdentity, PropertiesPresent: true, EventType: "ServiceIssue", Status: "Active", EventSource: "ResourceHealth", LastUpdateTime: ptr(operationalReference.Add(-time.Hour)), Impact: []ServiceImpactInput{{Service: "Virtual Machines", Regions: []ServiceRegionInput{{Name: "West Europe", Status: "Active", ImpactedSubscriptions: []string{testSubscription}}}}}}
}
func monitorInput() MonitorInput {
	return MonitorInput{ID: alertIdentity, PropertiesPresent: true, TargetResourceID: resourcePrefix + "vm", SignalType: "Metric", Condition: "Fired", WorkflowState: "New", Severity: "Sev2", LastModifiedTime: ptr(operationalReference.Add(-time.Hour))}
}
func normalizeService(r *Run, inputs ...ServiceHealthInput) {
	r.NormalizePage("service_health", Page{ServiceHealth: inputs, CollectedAt: operationalReference})
}
func normalizeMonitor(r *Run, inputs ...MonitorInput) {
	r.NormalizePage("azure_monitor", Page{Monitor: inputs, CollectedAt: operationalReference})
}

func TestOperationalQueryPoliciesAndSchema(t *testing.T) {
	r := NewRun("synthetic", "test")
	if r.SchemaVersion != 3 {
		t.Fatal("new runs must use schema 3")
	}
	reference := operationalReference.In(time.FixedZone("test", 3600))
	q := ServiceHealthQuery(reference)
	if q.ReferenceAt.Location() != time.UTC || !q.ReferenceAt.Equal(reference) || q.Cutoff == nil || reference.Sub(*q.Cutoff) != 30*24*time.Hour || q.QueryStartTime != "10/16/2026" {
		t.Fatalf("wrong policy: %+v", q)
	}
	// Calendar date uses UTC even near midnight in the caller's zone.
	q = ServiceHealthQuery(time.Date(2026, 11, 15, 0, 1, 0, 0, time.FixedZone("east", 3600)))
	if q.QueryStartTime != "10/15/2026" {
		t.Fatal("local calendar date used")
	}
	m := MonitorQuery(reference)
	if m.TimeRange != "30d" || m.Cutoff != nil || m.QueryStartTime != "" {
		t.Fatal("undocumented Monitor local cutoff")
	}
}

func TestServiceHealthCategoriesStatesAndTimeBoundaries(t *testing.T) {
	for _, category := range []string{"ServiceIssue", "PlannedMaintenance", "HealthAdvisory", "SecurityAdvisory"} {
		for _, tc := range []struct {
			name, status, disposition, gap string
			update                         *time.Time
			findings                       int
		}{
			{"active", "Active", "active", "", ptr(operationalReference), 1},
			{"resolved", "Resolved", "inactive", "", ptr(operationalReference), 0},
			{"unknown", "FutureState", "unclassified", "unknown_state", ptr(operationalReference), 0},
			{"missing state", "", "unclassified", "unknown_state", ptr(operationalReference), 0},
			{"exact cutoff", "Active", "active", "", ptr(operationalReference.Add(-30 * 24 * time.Hour)), 1},
			{"one nanosecond before cutoff", "Active", "excluded", "", ptr(operationalReference.Add(-30*24*time.Hour - time.Nanosecond)), 0},
			{"query day extra portion", "Active", "excluded", "", ptr(time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC)), 0},
			{"missing time", "Active", "unclassified", "unknown_time_coverage", nil, 0},
			{"zero time", "Active", "unclassified", "unknown_time_coverage", ptr(time.Time{}), 0},
		} {
			t.Run(category+"/"+tc.name, func(t *testing.T) {
				r := operationalRun()
				input := serviceInput()
				input.EventType, input.Status, input.LastUpdateTime = category, tc.status, tc.update
				normalizeService(&r, input)
				if len(r.Evidence) != 1 || len(r.Findings) != tc.findings || r.Evidence[0].Disposition != tc.disposition || tc.gap != "" && !hasGap(r, tc.gap) {
					t.Fatalf("wrong classification: %+v", r)
				}
				if r.Evidence[0].ResourceID != "" || len(r.Observations) != 0 || len(r.Resources) != 0 || r.Evidence[0].ServiceHealth.Category != category || r.Evidence[0].ServiceHealth.EventSource != "ResourceHealth" {
					t.Fatal("event became resource fault or category/source lost")
				}
				if category == "SecurityAdvisory" && !hasGap(r, "sensitive_details_not_retrieved") {
					t.Fatal("sensitive detail limit absent")
				}
				if r.Sources["service_health"].RejectedRecords != 0 {
					t.Fatal("unknown domain fact rejected")
				}
				assertReferences(t, r)
			})
		}
	}
	for _, category := range []string{"Billing", "EmergingIssues", "RCA", "FutureCategory"} {
		r := operationalRun()
		input := serviceInput()
		input.EventType = category
		normalizeService(&r, input)
		source := r.Sources["service_health"]
		if len(r.Findings) != 0 || source.RejectedRecords != 0 {
			t.Fatal("excluded/unknown created finding or rejection")
		}
		if category == "FutureCategory" {
			if source.UnclassifiedRecords != 1 || !hasGap(r, "unknown_category") {
				t.Fatal("unknown category hidden")
			}
		} else if source.ExcludedRecords != 1 {
			t.Fatal("known exclusion conflated")
		}
	}
}

func TestServiceHealthAttributionIdentityAndSensitiveMetadata(t *testing.T) {
	for _, tc := range []struct {
		name, id string
		valid    bool
	}{
		{"global", eventIdentity, true}, {"scoped", "/subscriptions/" + testSubscription + eventIdentity, true},
		{"foreign", "/subscriptions/" + otherSubscription + eventIdentity, false},
		{"wrong provider", "/providers/Foreign.Provider/events/one", false},
		{"extra path", eventIdentity + "/child/one", false},
		{"resource event", resourcePrefix + "vm" + eventIdentity, false},
		{"unsafe", eventIdentity + "%2Fescape", false},
		{"empty", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := operationalRun()
			input := serviceInput()
			input.ID = tc.id
			normalizeService(&r, input)
			if (len(r.Evidence) == 1) != tc.valid || (r.Sources["service_health"].RejectedRecords == 1) == tc.valid {
				t.Fatal("identity boundary")
			}
		})
	}
	for _, tc := range []struct {
		name             string
		subscriptions    []string
		groups, regions  int
		attribution, gap string
	}{
		{"selected", []string{testSubscription}, 1, 1, "selected_subscription", ""},
		{"mixed", []string{otherSubscription, testSubscription}, 1, 1, "selected_subscription", ""},
		{"absent", nil, 1, 1, "unknown", "unknown_region_attribution"},
		{"service only", nil, 1, 0, "", "missing_region_attribution"},
		{"foreign", []string{otherSubscription}, 0, 0, "", "conflicting_subscription_attribution"},
		{"malformed", []string{"foreign-scope-canary"}, 0, 0, "", "conflicting_subscription_attribution"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := operationalRun()
			input := serviceInput()
			input.Sensitive = ptr(true)
			input.Impact[0].Regions[0].ImpactedSubscriptions = tc.subscriptions
			input.Impact[0].Regions[0].LastUpdateTime = ptr(operationalReference)
			if tc.name == "service only" {
				input.Impact[0].Regions = nil
			}
			normalizeService(&r, input)
			if len(r.Evidence) != 1 || len(r.Findings) != 1 || r.Sources["service_health"].RejectedRecords != 0 {
				t.Fatal("scope uncertainty removed the event or finding")
			}
			impact := r.Evidence[0].ServiceHealth.Impact
			if len(impact) != tc.groups {
				t.Fatalf("impact groups = %d, want %d", len(impact), tc.groups)
			}
			var regions []ServiceRegionFacts
			if len(impact) > 0 {
				if impact[0].Service != input.Impact[0].Service {
					t.Fatal("service label lost")
				}
				regions = impact[0].Regions
			}
			if len(regions) != tc.regions || len(regions) > 0 && (regions[0].Attribution != tc.attribution || regions[0].LastUpdateTime == nil) || tc.gap != "" && !hasGap(r, tc.gap) || !hasGap(r, "sensitive_details_not_retrieved") {
				t.Fatal("attribution/detail coverage lost")
			}
			b, _ := json.Marshal(r)
			if strings.Contains(string(b), otherSubscription) || strings.Contains(string(b), "foreign-scope-canary") {
				t.Fatal("foreign scopes persisted")
			}
		})
	}
	r := operationalRun()
	input := serviceInput()
	input.Impact = nil
	normalizeService(&r, input)
	if !hasGap(r, "missing_service_region_attribution") || len(r.Findings) != 1 {
		t.Fatal("missing attribution became rejection or hidden")
	}
	for _, category := range []string{"PlannedMaintenance", "HealthAdvisory", "SecurityAdvisory"} {
		r := operationalRun()
		input := serviceInput()
		input.EventType = category
		input.ImpactStartTime = ptr(operationalReference.Add(time.Hour))
		normalizeService(&r, input)
		if len(r.Findings) != 1 || !strings.Contains(r.Findings[0].Title, "future "+category+" notice") {
			t.Fatal("future notice treated as outage")
		}
	}
}

func TestMonitorConditionIndependentOfWorkflowSuppressionAndTime(t *testing.T) {
	for _, signal := range []string{"Metric", "Log"} {
		for _, workflow := range []string{"New", "Acknowledged", "Closed", "", "FutureWorkflow"} {
			for _, suppressed := range []*bool{nil, ptr(true), ptr(false)} {
				r := operationalRun()
				input := monitorInput()
				input.SignalType, input.WorkflowState, input.IsSuppressed = signal, workflow, suppressed
				// Monitor has an API window, not a local last-update cutoff.
				input.LastModifiedTime = ptr(operationalReference.Add(-100 * 24 * time.Hour))
				normalizeMonitor(&r, input)
				if len(r.Findings) != 1 || r.Evidence[0].Monitor.WorkflowState != workflow || !reflect.DeepEqual(r.Evidence[0].Monitor.IsSuppressed, suppressed) || r.Evidence[0].Disposition != "active" {
					t.Fatal("workflow/suppression/time suppressed Fired")
				}
				assertReferences(t, r)
			}
		}
	}
	for _, tc := range []struct{ signal, condition, disposition, gap string }{
		{"Metric", "Resolved", "inactive", ""}, {"Log", "FutureCondition", "unclassified", "unknown_condition"}, {"FutureSignal", "Fired", "unclassified", "unknown_signal_type"}, {"", "Fired", "unclassified", "unknown_signal_type"}, {"Activity Log", "Fired", "excluded", ""},
	} {
		r := operationalRun()
		input := monitorInput()
		input.SignalType, input.Condition = tc.signal, tc.condition
		normalizeMonitor(&r, input)
		if len(r.Findings) != 0 || r.Evidence[0].Disposition != tc.disposition || tc.gap != "" && !hasGap(r, tc.gap) || r.Sources["azure_monitor"].RejectedRecords != 0 {
			t.Fatal("condition/signal classification")
		}
	}
	r := operationalRun()
	input := monitorInput()
	input.Severity = "FutureSeverity"
	normalizeMonitor(&r, input)
	if len(r.Findings) != 1 || !hasGap(r, "unknown_severity") {
		t.Fatal("optional severity changed Fired condition or gap hidden")
	}
}

func TestMonitorOwnIdentityAndTargetAreIndependent(t *testing.T) {
	for _, tc := range []struct {
		name, identity, target string
		valid, targetValid     bool
		gap                    string
	}{
		{"selected", alertIdentity, resourcePrefix + "vm", true, true, ""},
		{"nested", resourcePrefix + "vm/providers/Microsoft.AlertsManagement/alerts/nested", resourcePrefix + "vm/extensions/agent", true, true, ""},
		{"foreign alert", strings.Replace(alertIdentity, testSubscription, otherSubscription, 1), resourcePrefix + "vm", false, false, ""},
		{"global alert", "/providers/Microsoft.AlertsManagement/alerts/one", resourcePrefix + "vm", false, false, ""},
		{"wrong provider", strings.Replace(alertIdentity, "Microsoft.AlertsManagement", "Foreign.Provider", 1), resourcePrefix + "vm", false, false, ""},
		{"missing target", alertIdentity, "", true, false, "unknown_resource_attribution"},
		{"unsafe target", alertIdentity, resourcePrefix + "foreign-target-canary?access_token=secret-canary", true, false, "unknown_resource_attribution"},
		{"foreign target", alertIdentity, strings.Replace(resourcePrefix, testSubscription, otherSubscription, 1) + "foreign-target-canary", true, false, "unknown_resource_attribution"},
		{"workspace", alertIdentity, "/subscriptions/" + testSubscription + "/resourceGroups/g/providers/Microsoft.OperationalInsights/workspaces/logs", true, true, "workspace_target_only"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := operationalRun()
			input := monitorInput()
			input.ID, input.TargetResourceID = tc.identity, tc.target
			normalizeMonitor(&r, input)
			if (len(r.Evidence) == 1) != tc.valid || (len(r.Findings) == 1) != tc.valid {
				t.Fatal("alert scope or unsafe target discarded valid metadata")
			}
			if tc.valid && ((r.Evidence[0].ResourceID != "") != tc.targetValid || tc.gap != "" && !hasGap(r, tc.gap)) {
				t.Fatal("unsafe/invented target attribution")
			}
			if len(r.Observations) != 0 || len(r.Resources) != 0 {
				t.Fatal("alert became Resource Health/inventory")
			}
			b, _ := json.Marshal(r)
			if strings.Contains(string(b), otherSubscription) || strings.Contains(string(b), "canary") {
				t.Fatal("unsafe scope persisted")
			}
			assertReferences(t, r)
		})
	}
}

func TestOperationalCurrentSelectionDuplicatesAndStableFindings(t *testing.T) {
	for _, source := range []string{"service_health", "azure_monitor"} {
		for _, scenario := range []string{"duplicate", "latest resolved", "latest fired", "tied conflict", "missing update", "same state different attribution", "duplicate reordered impact"} {
			t.Run(source+"/"+scenario, func(t *testing.T) {
				makePages := func() (Page, Page) {
					a, b := serviceInput(), serviceInput()
					x, y := monitorInput(), monitorInput()
					later := operationalReference
					switch scenario {
					case "latest resolved":
						b.Status = "Resolved"
						y.Condition = "Resolved"
						b.LastUpdateTime = &later
						y.LastModifiedTime = &later
					case "latest fired":
						a.Status = "Resolved"
						x.Condition = "Resolved"
						b.LastUpdateTime = &later
						y.LastModifiedTime = &later
					case "tied conflict":
						b.Status = "Resolved"
						y.Condition = "Resolved"
					case "missing update":
						b.Status = "Resolved"
						y.Condition = "Resolved"
						b.LastUpdateTime = nil
						y.LastModifiedTime = nil
					case "same state different attribution":
						b.Impact = nil
						y.TargetResourceID = ""
					case "duplicate reordered impact":
						a.Impact = append(a.Impact, ServiceImpactInput{Service: "Storage"})
						b.Impact = []ServiceImpactInput{a.Impact[1], a.Impact[0]}
					}
					return Page{ServiceHealth: []ServiceHealthInput{a}, Monitor: []MonitorInput{x}, CollectedAt: operationalReference}, Page{ServiceHealth: []ServiceHealthInput{b}, Monitor: []MonitorInput{y}, CollectedAt: operationalReference.Add(time.Minute)}
				}
				r := operationalRun()
				a, b := makePages()
				r.NormalizePage(source, a)
				baseline := operationalRun()
				baseline.NormalizePage(source, Page{ServiceHealth: []ServiceHealthInput{serviceInput()}, Monitor: []MonitorInput{monitorInput()}})
				first := baseline.Findings[0].ID
				r.NormalizePage(source, b)
				wantFindings := 1
				if scenario == "latest resolved" || scenario == "tied conflict" || scenario == "missing update" || scenario == "same state different attribution" {
					wantFindings = 0
				}
				count := 2
				if scenario == "duplicate" || scenario == "duplicate reordered impact" {
					count = 1
				}
				if len(r.Evidence) != count || len(r.Findings) != wantFindings {
					t.Fatalf("evidence=%d findings=%d gaps=%+v", len(r.Evidence), len(r.Findings), r.Gaps)
				}
				if wantFindings == 1 && r.Findings[0].ID != first {
					t.Fatal("finding identity changed with evidence update")
				}
				if strings.HasPrefix(scenario, "latest") && !hasGap(r, "superseded_operational_evidence") {
					t.Fatal("superseded evidence hidden")
				}
				if scenario == "tied conflict" || scenario == "missing update" || scenario == "same state different attribution" {
					if !hasGap(r, "conflicting_operational_evidence") {
						t.Fatal("conflict hidden")
					}
				}
				reversed := operationalRun()
				reversed.NormalizePage(source, b)
				reversed.NormalizePage(source, a)
				if !reflect.DeepEqual(r.Findings, reversed.Findings) || !reflect.DeepEqual(r.Gaps, reversed.Gaps) {
					t.Fatal("retrieval order chose current state")
				}
				assertReferences(t, r)
			})
		}
	}
}

func TestOperationalPrivacyDispatchAndEmptyResourceEnumeration(t *testing.T) {
	r := operationalRun()
	sh := serviceInput()
	sh.Level = "Bearer level-secret-canary"
	sh.Impact[0].Service = "\x1b[31m<service>|[label](url)\x1b[0m"
	sh.Impact[0].Regions[0].Name = "Password=region-secret-canary"
	m := monitorInput()
	m.WorkflowState = "https://user:credential-canary@example.com"
	m.Severity = "Bearer severity-secret-canary"
	r.NormalizePage("service_health", Page{ServiceHealth: []ServiceHealthInput{sh, {ID: eventIdentity, PropertiesPresent: false}}, Health: []HealthInput{healthInput("vm", "Unavailable")}})
	r.NormalizePage("azure_monitor", Page{Monitor: []MonitorInput{m, {ID: alertIdentity, PropertiesPresent: false}}, Health: []HealthInput{healthInput("vm", "Unavailable")}})
	r.NormalizePage("unrecognized_source", Page{Health: []HealthInput{healthInput("vm", "Unavailable")}})
	if len(r.Evidence) != 2 || len(r.Observations) != 0 || r.Sources["service_health"].RejectedRecords != 1 || r.Sources["azure_monitor"].RejectedRecords != 1 {
		t.Fatal("source dispatch or rejection isolation failed")
	}
	r.Sources["inventory"] = Source{Status: "completed"}
	r.Sources["resource_health"] = Source{Status: "completed"}
	r.RebuildCoverage()
	if !hasGap(r, "no_resources_observed") || r.Coverage != "incomplete" {
		t.Fatal("operational evidence hid empty resource enumeration")
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "canary") || strings.Contains(string(b), "\\u001b") || r.Redactions != 4 || r.Evidence[0].ServiceHealth.Impact[0].Service != "<service>|[label](url)" {
		t.Fatal("pre-checkpoint privacy/literal text failed")
	}
	var restored Run
	if err := json.Unmarshal(b, &restored); err != nil {
		t.Fatal(err)
	}
	restored.RebuildCoverage()
	if !reflect.DeepEqual(restored.Findings, r.Findings) || !reflect.DeepEqual(restored.Gaps, r.Gaps) || restored.Sources["service_health"].Query.QueryStartTime != "10/16/2026" {
		t.Fatal("schema 2 roundtrip lost facts, provenance or references")
	}
	assertReferences(t, restored)
}

func TestUnknownOutOfWindowAndIncompleteImpactFactsRemainExplicit(t *testing.T) {
	r := operationalRun()
	input := serviceInput()
	input.EventType = "FutureCategory"
	input.LastUpdateTime = ptr(operationalReference.Add(-31 * 24 * time.Hour))
	input.Impact = []ServiceImpactInput{{Regions: []ServiceRegionInput{{Status: "FutureRegionState", ImpactedSubscriptions: []string{testSubscription}}}}}
	normalizeService(&r, input)
	source := r.Sources["service_health"]
	if source.UnclassifiedRecords != 1 || source.ExcludedRecords != 0 || len(r.Findings) != 0 || !hasGap(r, "unknown_category") || !hasGap(r, "unknown_region_state") || !hasGap(r, "missing_region_attribution") || !hasGap(r, "missing_service_attribution") {
		t.Fatal("unknown or incomplete source facts hidden by time/scope filtering")
	}
}

func TestOperationalEnumMeaningIsIndependentOfTextFiltering(t *testing.T) {
	for _, transform := range []struct {
		name  string
		apply func(string) string
	}{
		{"newline", func(s string) string { return s[:1] + "\n" + s[1:] }},
		{"terminal sequence", func(s string) string { return "\x1b[31m" + s + "\x1b[0m" }},
		{"format character", func(s string) string { return s[:1] + "\u200b" + s[1:] }},
	} {
		for _, tc := range []struct {
			name, source, value, gap string
			service                  func(*ServiceHealthInput, string)
			monitor                  func(*MonitorInput, string)
			stored                   func(Evidence) string
			unclassified             bool
		}{
			{name: "Monitor Fired", source: "azure_monitor", value: "Fired", gap: "unknown_condition", unclassified: true,
				monitor: func(v *MonitorInput, s string) { v.Condition = s }, stored: func(e Evidence) string { return e.Monitor.Condition }},
			{name: "Monitor Resolved", source: "azure_monitor", value: "Resolved", gap: "unknown_condition", unclassified: true,
				monitor: func(v *MonitorInput, s string) { v.Condition = s }, stored: func(e Evidence) string { return e.Monitor.Condition }},
			{name: "Monitor Metric", source: "azure_monitor", value: "Metric", gap: "unknown_signal_type", unclassified: true,
				monitor: func(v *MonitorInput, s string) { v.SignalType = s }, stored: func(e Evidence) string { return e.Monitor.SignalType }},
			{name: "Monitor Log", source: "azure_monitor", value: "Log", gap: "unknown_signal_type", unclassified: true,
				monitor: func(v *MonitorInput, s string) { v.SignalType = s }, stored: func(e Evidence) string { return e.Monitor.SignalType }},
			{name: "Monitor excluded signal", source: "azure_monitor", value: "Activity Log", gap: "unknown_signal_type", unclassified: true,
				monitor: func(v *MonitorInput, s string) { v.SignalType = s }, stored: func(e Evidence) string { return e.Monitor.SignalType }},
			{name: "Monitor workflow", source: "azure_monitor", value: "Acknowledged", gap: "unknown_workflow_state",
				monitor: func(v *MonitorInput, s string) { v.WorkflowState = s }, stored: func(e Evidence) string { return e.Monitor.WorkflowState }},
			{name: "Monitor severity", source: "azure_monitor", value: "Sev2", gap: "unknown_severity",
				monitor: func(v *MonitorInput, s string) { v.Severity = s }, stored: func(e Evidence) string { return e.Monitor.Severity }},
			{name: "Service Health Active", source: "service_health", value: "Active", gap: "unknown_state", unclassified: true,
				service: func(v *ServiceHealthInput, s string) { v.Status = s }, stored: func(e Evidence) string { return e.ServiceHealth.Status }},
			{name: "Service Health Resolved", source: "service_health", value: "Resolved", gap: "unknown_state", unclassified: true,
				service: func(v *ServiceHealthInput, s string) { v.Status = s }, stored: func(e Evidence) string { return e.ServiceHealth.Status }},
			{name: "Service Health incident", source: "service_health", value: "ServiceIssue", gap: "unknown_category", unclassified: true,
				service: func(v *ServiceHealthInput, s string) { v.EventType = s }, stored: func(e Evidence) string { return e.ServiceHealth.Category }},
			{name: "Service Health maintenance", source: "service_health", value: "PlannedMaintenance", gap: "unknown_category", unclassified: true,
				service: func(v *ServiceHealthInput, s string) { v.EventType = s }, stored: func(e Evidence) string { return e.ServiceHealth.Category }},
			{name: "Service Health health advisory", source: "service_health", value: "HealthAdvisory", gap: "unknown_category", unclassified: true,
				service: func(v *ServiceHealthInput, s string) { v.EventType = s }, stored: func(e Evidence) string { return e.ServiceHealth.Category }},
			{name: "Service Health security advisory", source: "service_health", value: "SecurityAdvisory", gap: "unknown_category", unclassified: true,
				service: func(v *ServiceHealthInput, s string) { v.EventType = s }, stored: func(e Evidence) string { return e.ServiceHealth.Category }},
			{name: "Service Health excluded category", source: "service_health", value: "Billing", gap: "unknown_category", unclassified: true,
				service: func(v *ServiceHealthInput, s string) { v.EventType = s }, stored: func(e Evidence) string { return e.ServiceHealth.Category }},
			{name: "Service Health regional Active", source: "service_health", value: "Active", gap: "unknown_region_state",
				service: func(v *ServiceHealthInput, s string) { v.Impact[0].Regions[0].Status = s }, stored: func(e Evidence) string { return e.ServiceHealth.Impact[0].Regions[0].Status }},
			{name: "Service Health regional Resolved", source: "service_health", value: "Resolved", gap: "unknown_region_state",
				service: func(v *ServiceHealthInput, s string) { v.Impact[0].Regions[0].Status = s }, stored: func(e Evidence) string { return e.ServiceHealth.Impact[0].Regions[0].Status }},
		} {
			t.Run(transform.name+"/"+tc.name, func(t *testing.T) {
				r := operationalRun()
				if tc.service != nil {
					input := serviceInput()
					tc.service(&input, transform.apply(tc.value))
					normalizeService(&r, input)
				} else {
					input := monitorInput()
					tc.monitor(&input, transform.apply(tc.value))
					normalizeMonitor(&r, input)
				}
				if len(r.Evidence) != 1 {
					t.Fatal("unknown enum must remain evidence")
				}
				e := r.Evidence[0]
				wantDisposition, wantFindings, wantUnclassified := "active", 1, 0
				if tc.unclassified {
					wantDisposition, wantFindings, wantUnclassified = "unclassified", 0, 1
				}
				source := r.Sources[tc.source]
				if e.Disposition != wantDisposition || len(r.Findings) != wantFindings || source.UnclassifiedRecords != wantUnclassified || source.ExcludedRecords != 0 || source.RejectedRecords != 0 || !hasGap(r, tc.gap) {
					t.Fatalf("source enum changed meaning after filtering: disposition=%s want=%s findings=%d want=%d unclassified=%d want=%d excluded=%d rejected=%d gap=%s present=%v", e.Disposition, wantDisposition, len(r.Findings), wantFindings, source.UnclassifiedRecords, wantUnclassified, source.ExcludedRecords, source.RejectedRecords, tc.gap, hasGap(r, tc.gap))
				}
				if tc.stored(e) != tc.value {
					t.Fatal("persisted label was not filtered")
				}
				b, err := json.Marshal(r)
				if err != nil {
					t.Fatal(err)
				}
				for _, unsafe := range []string{"\\n", "\\u001b", "\u200b"} {
					if strings.Contains(string(b), unsafe) {
						t.Fatal("control/format characters reached serialized evidence")
					}
				}
				assertReferences(t, r)
			})
		}
	}
}

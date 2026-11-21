package health

import (
	"context"
	"encoding/json"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
)

func classifiedRun(t *testing.T, input map[string][]Page) Run {
	t.Helper()
	r := scopedRun()
	r.SchemaVersion = AssessmentSchemaVersion
	r.StartedAt = operationalReference
	collectors := make([]NamedCollector, 0, len(assessmentSources))
	for _, name := range assessmentSources {
		pages, configured := input[name]
		if !configured {
			pages = []Page{{}}
		}
		collectors = append(collectors, NamedCollector{Name: name, Collect: func(_ context.Context, _ string, emit func(Page) error) error {
			for _, page := range pages {
				if page.CollectedAt.IsZero() {
					page.CollectedAt = operationalReference
				}
				if err := emit(page); err != nil {
					return err
				}
			}
			return nil
		}})
	}
	code, err := Check(context.Background(), &r, collectors, func(checkpoint *Run) error {
		if checkpoint.Coverage != "incomplete" || checkpoint.AssessmentSummary.OK != 0 || checkpoint.EndedAt != nil {
			t.Fatalf("checkpoint affirmed coverage: %+v", checkpoint.AssessmentSummary)
		}
		return nil
	})
	if err != nil || code != 0 || r.AssessmentSummary.OK != 0 {
		t.Fatalf("unexpected unfinished run: %d %v %+v", code, err, r.AssessmentSummary)
	}
	r.EndedAt = ptr(operationalReference.Add(time.Minute))
	r.RebuildCoverage()
	return r
}

func assessmentInventory(names ...string) Page {
	var page Page
	for _, name := range names {
		page.Resources = append(page.Resources, ResourceInput{ID: resourcePrefix + name, Name: name, Location: "West Europe"})
	}
	return page
}

func assertAssessment(t *testing.T, r Run, name, disposition, blocker string) ResourceAssessment {
	t.Helper()
	for _, a := range r.ResourceAssessments {
		if !strings.EqualFold(a.ResourceID, resourcePrefix+name) {
			continue
		}
		if a.Disposition != disposition || blocker != "" && !slices.ContainsFunc(a.BlockingReasons, func(g Gap) bool { return g.Reason == blocker }) {
			t.Fatalf("assessment for %s: %+v", name, a)
		}
		return a
	}
	t.Fatalf("missing assessment for %s", name)
	return ResourceAssessment{}
}

func assertSummary(t *testing.T, r Run, coverage string, ok, findings, unknown int) {
	t.Helper()
	want := AssessmentSummary{OK: ok, Finding: findings, Unknown: unknown}
	if r.Coverage != coverage || r.AssessmentSummary == nil || *r.AssessmentSummary != want || len(r.ResourceAssessments) != ok+findings+unknown {
		t.Fatalf("got coverage %s, counts %+v and %d assessments, want %s %+v", r.Coverage, r.AssessmentSummary, len(r.ResourceAssessments), coverage, want)
	}
}

func TestAssessmentsRequireCompletedScopedQueries(t *testing.T) {
	input := map[string][]Page{
		"inventory":       {assessmentInventory("vm")},
		"resource_health": {{Health: []HealthInput{healthInput("vm", "Available")}}},
	}
	r := classifiedRun(t, input)
	assertSummary(t, r, "complete", 1, 0, 0)
	a := assertAssessment(t, r, "vm", "ok", "")
	for i, check := range a.Checks {
		source := assessmentSources[i]
		if check.SourceRef != "sources."+source || (i < 2 && len(check.EvidenceIDs) != 1) || (i >= 2 && len(check.EvidenceIDs) != 0) {
			t.Fatalf("unexpected check proof for %s: %+v", source, check)
		}
		s := r.Sources[source]
		if s.Pages != 1 || s.FirstPageAt == nil || s.LastPageAt == nil || s.Query == nil || s.Query.RequestedAt == nil ||
			s.Query.SubscriptionID != testSubscription || s.Query.Operation == "" || s.Query.APIVersion == "" {
			t.Fatalf("missing query provenance even for empty pages: %+v", s)
		}
	}
	var restored Run
	b, err := json.Marshal(r)
	if err != nil || json.Unmarshal(b, &restored) != nil {
		t.Fatal("cannot round-trip schema 4")
	}
	restored.RebuildCoverage()
	if !reflect.DeepEqual(r.ResourceAssessments, restored.ResourceAssessments) || !reflect.DeepEqual(r.Gaps, restored.Gaps) {
		t.Fatal("rebuilding the saved snapshot changed its assessment")
	}

	for _, source := range assessmentSources[2:] {
		t.Run(source, func(t *testing.T) {
			input[source] = nil
			r := classifiedRun(t, input)
			delete(input, source)
			if s := r.Sources[source]; s.Status != "completed" || s.Query.RequestedAt == nil || s.Pages != 0 {
				t.Fatalf("expected preparation without pages: %+v", s)
			}
			assertSummary(t, r, "incomplete", 0, 0, 1)
			assertAssessment(t, r, "vm", "unknown", "no_accepted_page_proof")
		})
	}
	for _, mutate := range []func(*SourceQuery){
		func(q *SourceQuery) { q.SubscriptionID = otherSubscription },
		func(q *SourceQuery) { q.Operation = "wrong" },
		func(q *SourceQuery) { q.APIVersion = "wrong" },
		func(q *SourceQuery) { q.MonitorCondition = "Resolved" },
		func(q *SourceQuery) { q.TimeRange = "7d" },
	} {
		s := r.Sources["azure_monitor"]
		original := s
		q := *s.Query
		mutate(&q)
		s.Query = &q
		r.Sources["azure_monitor"] = s
		r.RebuildCoverage()
		assertAssessment(t, r, "vm", "unknown", "unverified_query_policy")
		r.Sources["azure_monitor"] = original
	}
	for _, outcome := range []string{"in_progress", "failed", "cancelled"} {
		r.Outcome = outcome
		r.RebuildCoverage()
		assertAssessment(t, r, "vm", "unknown", "execution_not_finalized_successfully")
	}
	r.Outcome, r.Scope = "completed", nil
	r.RebuildCoverage()
	assertAssessment(t, r, "vm", "unknown", "unresolved_subscription_scope")
	empty := classifiedRun(t, nil)
	assertSummary(t, empty, "incomplete", 0, 0, 0)
	if !hasGap(empty, "no_resources_observed") {
		t.Fatal("empty inventory lacks an explicit gap")
	}
}

func TestAssessmentUsesDecidingResourceHealthAndRetainsHistory(t *testing.T) {
	older, newer := healthInput("vm", "Degraded"), healthInput("VM", "Available")
	older.ReportedAt = ptr(operationalReference.Add(-2 * time.Hour))
	newer.ReportedAt = ptr(operationalReference.Add(-time.Hour))
	pages := []Page{{Health: []HealthInput{older}}, {Health: []HealthInput{newer, newer}}}
	inventory := []Page{assessmentInventory("vm"), assessmentInventory("VM"), assessmentInventory("vm")}
	var first ResourceAssessment
	for _, reverse := range []bool{false, true} {
		if reverse {
			slices.Reverse(pages)
			slices.Reverse(inventory)
		}
		r := classifiedRun(t, map[string][]Page{"inventory": inventory, "resource_health": pages})
		assertSummary(t, r, "complete", 1, 0, 0)
		a := assertAssessment(t, r, "vm", "ok", "")
		available := slices.IndexFunc(r.Evidence, func(e Evidence) bool { return e.State == "Available" })
		if len(r.Evidence) != 4 || len(r.Findings) != 1 || len(a.FindingIDs) != 0 ||
			!slices.Equal(a.HistoricalFindingIDs, []string{r.Findings[0].ID}) ||
			available < 0 || !slices.Equal(a.Checks[1].EvidenceIDs, []string{r.Evidence[available].ID}) {
			t.Fatalf("history leaked into current proof: %+v / %+v", a, r.Evidence)
		}
		if reverse && !reflect.DeepEqual(first, a) {
			t.Fatal("page order changed the assessment")
		}
		first = a
	}
	for _, tc := range []struct {
		name   string
		health []HealthInput
		reason string
	}{
		{"current conflict", []HealthInput{healthInput("vm", "Available"), healthInput("vm", "Degraded")}, "conflicting_health_evidence"},
		{"source unknown", []HealthInput{healthInput("vm", "Unknown")}, "source_unknown"},
		{"unsupported state", []HealthInput{healthInput("vm", "Unsupported")}, "undetermined_health_evidence"},
		{"event only", []HealthInput{{StatusID: older.StatusID, EventPresent: true}}, "undetermined_health_evidence"},
		{"historical only", []HealthInput{{StatusID: strings.TrimSuffix(older.StatusID, "current") + "historic", PropertiesPresent: true, State: ptr("Available")}}, "missing_current_health_evidence"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := classifiedRun(t, map[string][]Page{"inventory": {assessmentInventory("vm")}, "resource_health": {{Health: tc.health}}})
			assertAssessment(t, r, "vm", "unknown", tc.reason)
		})
	}
}

func TestAssessmentLocalizedGapsAndDisjointCounts(t *testing.T) {
	fired, postponed := monitorInput(), advisorInput("postponed")
	fired.TargetResourceID, fired.WorkflowState, fired.IsSuppressed = resourcePrefix+"vm", "Closed", ptr(true)
	postponed.TargetResourceID, postponed.Status = resourcePrefix+"quiet", "Postponed"
	postponed.ID = postponed.TargetResourceID + "/providers/Microsoft.Advisor/recommendations/postponed"
	input := map[string][]Page{
		"inventory":       {assessmentInventory("vm", "quiet", "missing", "degraded")},
		"resource_health": {{Health: []HealthInput{healthInput("vm", "Available"), healthInput("quiet", "Available"), healthInput("degraded", "Unavailable")}}},
		"azure_monitor":   {{Monitor: []MonitorInput{fired}}},
		advisorSource:     {{Advisor: []AdvisorInput{postponed}}},
	}
	r := classifiedRun(t, input)
	assertSummary(t, r, "incomplete", 1, 2, 1)
	a := assertAssessment(t, r, "quiet", "ok", "")
	if len(a.PostponedEvidenceIDs) != 1 || len(a.FindingIDs) != 0 {
		t.Fatalf("Postponed recommendation should remain context: %+v", a)
	}
	assertAssessment(t, r, "degraded", "finding", "")
	assertAssessment(t, r, "vm", "finding", "")
	assertAssessment(t, r, "missing", "unknown", "missing_health_evidence")
	input["resource_health"][0].Health = append(input["resource_health"][0].Health, healthInput("missing", "Available"))
	assertSummary(t, classifiedRun(t, input), "complete", 2, 2, 0)
	fired.TargetResourceID = resourcePrefix + "missing"
	fired.ID += "-missing"
	r.NormalizePage("azure_monitor", Page{Monitor: []MonitorInput{fired}, CollectedAt: operationalReference})
	a = assertAssessment(t, r, "missing", "unknown", "missing_health_evidence")
	if len(a.FindingIDs) != 1 {
		t.Fatal("known Fired alert missing beside the health gap")
	}
	s := r.Sources["azure_monitor"]
	s.Status, s.Reason = "partial", "access_denied"
	r.Sources["azure_monitor"] = s
	r.Outcome = "partial"
	r.RebuildCoverage()
	assertSummary(t, r, "incomplete", 0, 0, 4)
	if len(assertAssessment(t, r, "missing", "unknown", "access_denied").FindingIDs) != 1 {
		t.Fatal("partial source hid a committed finding")
	}
}

func TestAssessmentDoesNotJoinDistinctUnicodeResourceIDs(t *testing.T) {
	input := map[string][]Page{
		"inventory":       {assessmentInventory("i", "İ", "quiet")},
		"resource_health": {{Health: []HealthInput{healthInput("I", "Available"), healthInput("quiet", "Available"), healthInput("outside", "Available")}}},
	}
	r := classifiedRun(t, input)
	assertSummary(t, r, "incomplete", 2, 0, 1)
	assertAssessment(t, r, "i", "ok", "")
	assertAssessment(t, r, "İ", "unknown", "missing_health_evidence")
	if !hasGap(r, "health_resource_absent_from_inventory") {
		t.Fatal("inventory mismatch disappeared from overall coverage")
	}
	conflict := assessmentInventory("I")
	conflict.Resources[0].Location = "conflicting-location"
	input["inventory"] = append(input["inventory"], conflict)
	r = classifiedRun(t, input)
	assertAssessment(t, r, "i", "unknown", "conflicting_inventory")
	assertAssessment(t, r, "quiet", "ok", "")
}

func TestSchema4InventoryEvidenceDeduplicatesCaseEquivalentIDs(t *testing.T) {
	first := assessmentInventory("vm").Resources[0]
	alias := first
	alias.ID = strings.ToUpper(first.ID)
	legacy := scopedRun()
	legacy.SchemaVersion = 3
	legacy.NormalizePage("inventory", Page{Resources: []ResourceInput{first, alias}, CollectedAt: operationalReference})
	if len(legacy.Evidence) != 2 {
		t.Fatal("legacy inventory evidence changed")
	}

	pages := []Page{{Resources: []ResourceInput{first}}, {Resources: []ResourceInput{alias}}}
	var earliest ResourceAssessment
	for _, reverse := range []bool{false, true} {
		if reverse {
			slices.Reverse(pages)
		}
		r := classifiedRun(t, map[string][]Page{
			"inventory":       pages,
			"resource_health": {{Health: []HealthInput{healthInput("VM", "Available")}}},
		})
		assertSummary(t, r, "complete", 1, 0, 0)
		a := assertAssessment(t, r, "vm", "ok", "")
		if len(r.Evidence) != 2 || len(a.Checks[0].EvidenceIDs) != 1 {
			t.Fatalf("expected one inventory evidence record and one health record: %+v", a)
		}
		e := r.Evidence[0]
		wantID := pages[0].Resources[0].ID
		if e.Resource == nil || e.ResourceID != wantID || e.Resource.ID != wantID ||
			e.Resource.ResourceGroup != strings.Split(wantID, "/")[4] ||
			a.Checks[0].EvidenceIDs[0] != e.ID || a.ResourceID != r.Resources[0].ID {
			t.Fatalf("source spelling or inventory reference changed: evidence %+v, assessment %+v", e, a)
		}
		if reverse && !reflect.DeepEqual(earliest, a) {
			t.Fatal("page order changed the assessment's inventory identity or evidence references")
		}
		earliest = a
	}
}

func TestAssessmentKeepsNativeOperationalFoldClassesIndependent(t *testing.T) {
	fire, resolved := monitorInput(), monitorInput()
	fire.ID, resolved.ID = alertIdentity+"-i", alertIdentity+"-İ"
	resolved.Condition, resolved.LastModifiedTime = "Resolved", ptr(operationalReference)
	activeEvent, resolvedEvent := serviceInput(), serviceInput()
	activeEvent.ID, resolvedEvent.ID = eventIdentity+"-i", eventIdentity+"-İ"
	resolvedEvent.Status, resolvedEvent.LastUpdateTime = "Resolved", ptr(operationalReference)
	resolvedEvent.Impact[0].Regions[0].Status = "Resolved"
	monitor := []Page{{Monitor: []MonitorInput{fire}}, {Monitor: []MonitorInput{resolved}}}
	service := []Page{{ServiceHealth: []ServiceHealthInput{activeEvent}}, {ServiceHealth: []ServiceHealthInput{resolvedEvent}}}
	var first []ResourceAssessment
	for _, reverse := range []bool{false, true} {
		if reverse {
			slices.Reverse(monitor)
			slices.Reverse(service)
		}
		r := classifiedRun(t, map[string][]Page{
			"inventory": {assessmentInventory("vm", "quiet")}, "azure_monitor": monitor, "service_health": service,
			"resource_health": {{Health: []HealthInput{healthInput("vm", "Available"), healthInput("quiet", "Available")}}},
		})
		assertSummary(t, r, "incomplete", 0, 0, 2)
		a := assertAssessment(t, r, "vm", "unknown", "service_health_impact_undetermined")
		if len(a.FindingIDs) != 1 || len(r.Findings) != 2 || a.BlockingReasons[0].FindingID == "" {
			t.Fatalf("independent positive or subscription incident disappeared: %+v / %+v", a, r.Findings)
		}
		if reverse && !reflect.DeepEqual(first, r.ResourceAssessments) {
			t.Fatal("page order changed the operational assessment")
		}
		first = r.ResourceAssessments
	}
}

func TestAssessmentServiceHealthRegionalConflictScopeAndWindow(t *testing.T) {
	cutoff := operationalReference.Add(-30 * 24 * time.Hour)
	resolvedAt := operationalReference.Add(-time.Hour)
	for _, tc := range []struct {
		name, status                string
		subscriptions               []string
		eventUpdated, regionUpdated time.Time
		blocked                     bool
	}{
		{"newer selected region", "Active", []string{testSubscription}, resolvedAt, resolvedAt.Add(time.Minute), true},
		{"same time", "Active", []string{testSubscription}, resolvedAt, resolvedAt, true},
		{"older region", "Active", []string{testSubscription}, resolvedAt, resolvedAt.Add(-time.Minute), false},
		{"foreign region", "Active", []string{otherSubscription}, resolvedAt, resolvedAt.Add(time.Minute), false},
		{"missing region scope", "Active", nil, resolvedAt, resolvedAt.Add(time.Minute), true},
		{"invalid region scope", "Active", []string{"invalid-subscription-canary"}, resolvedAt, resolvedAt.Add(time.Minute), true},
		{"foreign and incomplete region scope", "Active", []string{otherSubscription, ""}, resolvedAt, resolvedAt.Add(time.Minute), true},
		{"newer resolved region", "Resolved", []string{testSubscription}, resolvedAt, resolvedAt.Add(time.Minute), false},
		{"region at lower cutoff", "Active", []string{testSubscription}, cutoff.Add(-time.Minute), cutoff, true},
		{"region before lower cutoff", "Active", []string{testSubscription}, cutoff.Add(-2 * time.Minute), cutoff.Add(-time.Minute), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			event := serviceInput()
			event.Status, event.LastUpdateTime = "Resolved", &tc.eventUpdated
			region := &event.Impact[0].Regions[0]
			region.Status, region.LastUpdateTime = tc.status, &tc.regionUpdated
			region.ImpactedSubscriptions = tc.subscriptions
			r := classifiedRun(t, map[string][]Page{
				"inventory":       {assessmentInventory("vm")},
				"resource_health": {{Health: []HealthInput{healthInput("vm", "Available")}}},
				"service_health":  {{ServiceHealth: []ServiceHealthInput{event}}},
			})
			if tc.blocked {
				assertSummary(t, r, "incomplete", 0, 0, 1)
				a := assertAssessment(t, r, "vm", "unknown", conflictingServiceHealthRegionalState)
				if len(a.BlockingReasons) != 1 || a.BlockingReasons[0].ResourceID != "" {
					t.Fatalf("expected a subscription-wide blocker: %+v", a.BlockingReasons)
				}
			} else {
				assertSummary(t, r, "complete", 1, 0, 0)
			}
		})
	}
}

func TestAssessmentReconcilesServiceHealthRegionsAcrossPages(t *testing.T) {
	for _, tc := range []struct {
		name, service string
		regionAge     time.Duration
		blocked       bool
		unknownScope  bool
	}{
		{"older regional resolution", "Virtual Machines", -time.Hour, true, false},
		{"tied regional resolution", "Virtual Machines", -30 * time.Minute, true, false},
		{"newer regional resolution", "virtual machines", -20 * time.Minute, false, false},
		{"newer resolution for another service", "Storage", -20 * time.Minute, true, false},
		{"newer resolution with filtered service label", "\x00Virtual Machines", -20 * time.Minute, true, false},
		{"newer resolution with unknown scope", "Virtual Machines", -20 * time.Minute, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			active, resolved := serviceInput(), serviceInput()
			active.Status, active.LastUpdateTime = "Resolved", ptr(operationalReference.Add(-2*time.Hour))
			active.Impact[0].Regions[0].LastUpdateTime = ptr(operationalReference.Add(-30 * time.Minute))
			resolved.Status, resolved.LastUpdateTime = "Resolved", ptr(operationalReference.Add(-time.Hour))
			resolved.Impact[0].Service = tc.service
			resolved.Impact[0].Regions[0].Name = "west europe"
			resolved.Impact[0].Regions[0].Status = "Resolved"
			resolved.Impact[0].Regions[0].LastUpdateTime = ptr(operationalReference.Add(tc.regionAge))
			if tc.unknownScope {
				resolved.Impact[0].Regions[0].ImpactedSubscriptions = nil
			}
			pages := []Page{{ServiceHealth: []ServiceHealthInput{active}}, {ServiceHealth: []ServiceHealthInput{resolved}}}
			var previous []ResourceAssessment
			for _, reverse := range []bool{false, true} {
				if reverse {
					slices.Reverse(pages)
				}
				r := classifiedRun(t, map[string][]Page{
					"inventory":       {assessmentInventory("vm")},
					"resource_health": {{Health: []HealthInput{healthInput("vm", "Available")}}},
					"service_health":  pages,
				})
				if tc.blocked {
					assertSummary(t, r, "incomplete", 0, 0, 1)
					a := assertAssessment(t, r, "vm", "unknown", conflictingServiceHealthRegionalState)
					if len(a.BlockingReasons) != 1 || a.BlockingReasons[0].ResourceID != "" || len(a.BlockingReasons[0].EvidenceIDs) != 2 {
						t.Fatalf("expected one subscription blocker citing both event versions: %+v", a.BlockingReasons)
					}
				} else {
					assertSummary(t, r, "complete", 1, 0, 0)
				}
				if reverse && !reflect.DeepEqual(previous, r.ResourceAssessments) {
					t.Fatal("reversing pages changed the assessment")
				}
				previous = r.ResourceAssessments
			}
		})
	}
}

func TestAssessmentServiceHealthRegionalUncertainty(t *testing.T) {
	resolvedAt := operationalReference.Add(-time.Hour)
	newer := operationalReference.Add(-time.Minute)
	for _, tc := range []struct {
		name, category, regionalState string
		regionalUpdated, impactStart  *time.Time
		blocked                       bool
	}{
		{"active region with no time", "ServiceIssue", "Active", nil, nil, true},
		{"unknown newer regional state", "ServiceIssue", "Investigating", &newer, nil, true},
		{"unknown tied regional state", "ServiceIssue", "", &resolvedAt, nil, true},
		{"unknown older regional state", "ServiceIssue", "Investigating", ptr(resolvedAt.Add(-time.Minute)), nil, false},
		{"maintenance underway with newer Active region", "PlannedMaintenance", "Active", &newer, ptr(resolvedAt.Add(-time.Hour)), true},
		{"maintenance still future when read", "PlannedMaintenance", "Active", &newer, ptr(operationalReference.Add(time.Hour)), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			event := serviceInput()
			event.EventType, event.Status = tc.category, "Resolved"
			event.LastUpdateTime, event.ImpactStartTime = &resolvedAt, tc.impactStart
			event.Impact[0].Regions[0].Status = tc.regionalState
			event.Impact[0].Regions[0].LastUpdateTime = tc.regionalUpdated
			input := map[string][]Page{
				"inventory":       {assessmentInventory("vm")},
				"resource_health": {{Health: []HealthInput{healthInput("vm", "Available")}}},
				"service_health":  {{ServiceHealth: []ServiceHealthInput{event}}},
			}
			r := classifiedRun(t, input)
			if !tc.blocked {
				assertSummary(t, r, "complete", 1, 0, 0)
				return
			}
			assertSummary(t, r, "incomplete", 0, 0, 1)
			a := assertAssessment(t, r, "vm", "unknown", conflictingServiceHealthRegionalState)
			if len(a.BlockingReasons) != 1 || a.BlockingReasons[0].ResourceID != "" || len(a.BlockingReasons[0].EvidenceIDs) != 1 {
				t.Fatalf("expected one subscription-wide blocker citing the event: %+v", a.BlockingReasons)
			}

			// A trusted regional resolution recorded after the first read can
			// settle even the observation whose regional timestamp was absent.
			resolution := serviceInput()
			resolution.EventType, resolution.Status = tc.category, "Resolved"
			resolution.LastUpdateTime, resolution.ImpactStartTime = &resolvedAt, tc.impactStart
			resolution.Impact[0].Regions[0].Status = "Resolved"
			resolution.Impact[0].Regions[0].LastUpdateTime = ptr(operationalReference.Add(time.Second))
			input["service_health"] = append(input["service_health"], Page{ServiceHealth: []ServiceHealthInput{resolution},
				CollectedAt: operationalReference.Add(2 * time.Second)})
			r = classifiedRun(t, input)
			assertSummary(t, r, "complete", 1, 0, 0)
		})
	}
}

func TestAssessmentServiceHealthFutureImpactUsesObservationTime(t *testing.T) {
	for _, tc := range []struct {
		name                       string
		start, collected, reported time.Duration
		future                     bool
	}{
		{"starts before collection", 10 * time.Second, 20 * time.Second, 5 * time.Second, false},
		{"starts before source update", 10 * time.Second, 5 * time.Second, 20 * time.Second, false},
		{"starts at collection", 10 * time.Second, 10 * time.Second, 5 * time.Second, false},
		{"still future when observed", 30 * time.Second, 10 * time.Second, 20 * time.Second, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			maintenance := serviceInput()
			maintenance.EventType = "PlannedMaintenance"
			maintenance.ImpactStartTime = ptr(operationalReference.Add(tc.start))
			maintenance.LastUpdateTime = ptr(operationalReference.Add(tc.reported))
			r := classifiedRun(t, map[string][]Page{
				"inventory":       {assessmentInventory("vm")},
				"resource_health": {{Health: []HealthInput{healthInput("vm", "Available")}}},
				"service_health":  {{ServiceHealth: []ServiceHealthInput{maintenance}, CollectedAt: operationalReference.Add(tc.collected)}},
			})
			q := r.Sources["service_health"].Query
			if !q.ReferenceAt.Equal(operationalReference) || !q.Cutoff.Equal(operationalReference.Add(-30*24*time.Hour)) {
				t.Fatal("the observation time moved the fixed query window")
			}
			title := "Service Health Active PlannedMaintenance notice"
			if tc.future {
				assertSummary(t, r, "complete", 1, 0, 0)
				title = "Service Health future PlannedMaintenance notice"
			} else {
				assertSummary(t, r, "incomplete", 0, 0, 1)
				assertAssessment(t, r, "vm", "unknown", "service_health_impact_undetermined")
			}
			if len(r.Findings) != 1 || r.Findings[0].Title != title {
				t.Fatalf("maintenance timing was mislabeled: %+v", r.Findings)
			}
		})
	}
}

func TestAssessmentServiceHealthKeepsLatestRepeatedRead(t *testing.T) {
	maintenance := serviceInput()
	maintenance.EventType = "PlannedMaintenance"
	maintenance.ImpactStartTime = ptr(operationalReference.Add(10 * time.Second))
	first := operationalReference.Add(5 * time.Second)
	last := operationalReference.Add(15 * time.Second)
	firstPage := Page{ServiceHealth: []ServiceHealthInput{maintenance}, CollectedAt: first}
	input := map[string][]Page{
		"inventory":       {assessmentInventory("vm")},
		"resource_health": {{Health: []HealthInput{healthInput("vm", "Available")}}},
		"service_health":  {firstPage},
	}
	before := classifiedRun(t, input)
	assertSummary(t, before, "complete", 1, 0, 0)
	input["service_health"] = append(input["service_health"], Page{ServiceHealth: firstPage.ServiceHealth, CollectedAt: last})
	after := classifiedRun(t, input)
	assertSummary(t, after, "incomplete", 0, 0, 1)
	assertAssessment(t, after, "vm", "unknown", "service_health_impact_undetermined")
	if len(after.Evidence) != len(before.Evidence) || len(after.Findings) != 1 || after.Findings[0].ID != before.Findings[0].ID ||
		!slices.Equal(after.Findings[0].EvidenceIDs, before.Findings[0].EvidenceIDs) || after.Findings[0].Title != "Service Health Active PlannedMaintenance notice" {
		t.Fatalf("repeat changed evidence identity or kept a stale finding: %+v", after.Findings)
	}
	data, err := json.Marshal(after)
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{`"collected_at":"` + first.Format(time.RFC3339Nano) + `"`, `"last_collected_at":"` + last.Format(time.RFC3339Nano) + `"`} {
		if !strings.Contains(string(data), field) {
			t.Fatalf("checkpoint missing %s", field)
		}
	}
}

func TestAssessmentFindingReferencesAreStableAcrossPageOrder(t *testing.T) {
	first, second := monitorInput(), monitorInput()
	first.TargetResourceID = strings.ToLower(first.TargetResourceID)
	second.ID += "-second"
	alias := first
	alias.TargetResourceID = strings.ToUpper(first.TargetResourceID)

	degraded, unavailable, available := healthInput("vm", "Degraded"), healthInput("vm", "Unavailable"), healthInput("vm", "Available")
	degraded.StatusID = strings.ToLower(degraded.StatusID)
	degraded.ReportedAt = ptr(operationalReference.Add(-3 * time.Hour))
	unavailable.ReportedAt = ptr(operationalReference.Add(-2 * time.Hour))
	available.ReportedAt = ptr(operationalReference.Add(-time.Hour))
	degradedAlias := degraded
	degradedAlias.StatusID = strings.ToUpper(degraded.StatusID)

	monitor := []Page{{Monitor: []MonitorInput{first, second}}, {Monitor: []MonitorInput{alias}}}
	resourceHealth := []Page{{Health: []HealthInput{degraded, unavailable}}, {Health: []HealthInput{degradedAlias, available}}}
	var before ResourceAssessment
	for _, reverse := range []bool{false, true} {
		if reverse {
			slices.Reverse(monitor)
			slices.Reverse(resourceHealth)
		}
		r := classifiedRun(t, map[string][]Page{
			"inventory":       {assessmentInventory("vm")},
			"resource_health": resourceHealth,
			"azure_monitor":   monitor,
		})
		assertSummary(t, r, "complete", 0, 1, 0)
		a := assertAssessment(t, r, "vm", "finding", "")
		if len(a.FindingIDs) != 2 || len(a.HistoricalFindingIDs) != 2 {
			t.Fatalf("expected two current and two historical references: %+v", a)
		}
		if reverse && !reflect.DeepEqual(before, a) {
			t.Fatalf("page order changed finding references: %v / %v, history %v / %v",
				before.FindingIDs, a.FindingIDs, before.HistoricalFindingIDs, a.HistoricalFindingIDs)
		}
		before = a
	}
	if !slices.IsSorted(before.FindingIDs) || !slices.IsSorted(before.HistoricalFindingIDs) {
		t.Fatal("finding references must be sorted by ID")
	}
}

func TestAssessmentEvaluatesEffectiveOperationalRows(t *testing.T) {
	oldEvent, resolved := serviceInput(), serviceInput()
	resolved.Status, resolved.LastUpdateTime = "Resolved", ptr(operationalReference)
	oldEvent.Impact[0].Regions[0].LastUpdateTime = oldEvent.LastUpdateTime
	resolved.Impact[0].Regions[0].Status = "Resolved"
	future := serviceInput()
	future.ID, future.EventType = eventIdentity+"-future", "PlannedMaintenance"
	future.ImpactStartTime = ptr(operationalReference.Add(time.Hour))
	outside := serviceInput()
	outside.ID, outside.LastUpdateTime = eventIdentity+"-outside", ptr(operationalReference.Add(-31*24*time.Hour))
	unclassified, cleared := monitorInput(), monitorInput()
	unclassified.SignalType, unclassified.TargetResourceID = "Unknown", ""
	cleared.Condition, cleared.LastModifiedTime = "Resolved", ptr(operationalReference)
	input := map[string][]Page{
		"inventory":       {assessmentInventory("vm", "quiet")},
		"resource_health": {{Health: []HealthInput{healthInput("vm", "Available"), healthInput("quiet", "Available")}}},
		"service_health":  {{ServiceHealth: []ServiceHealthInput{oldEvent, future, outside}}, {ServiceHealth: []ServiceHealthInput{resolved}}},
		"azure_monitor":   {{Monitor: []MonitorInput{unclassified}}, {Monitor: []MonitorInput{cleared}}},
	}
	r := classifiedRun(t, input)
	assertSummary(t, r, "complete", 2, 0, 0)
	if len(r.Findings) != 1 || r.Findings[0].Title != "Service Health future PlannedMaintenance notice" {
		t.Fatalf("expected only the future notice: %+v", r.Findings)
	}
	input["azure_monitor"] = []Page{{Monitor: []MonitorInput{unclassified}}}
	r = classifiedRun(t, input)
	assertSummary(t, r, "incomplete", 0, 0, 2)
	assertAssessment(t, r, "quiet", "unknown", "unclassified_operational_evidence")

	fired := monitorInput()
	fired.TargetResourceID = ""
	input["azure_monitor"] = []Page{{Monitor: []MonitorInput{fired}}}
	r = classifiedRun(t, input)
	assertSummary(t, r, "incomplete", 0, 0, 2)
	a := assertAssessment(t, r, "quiet", "unknown", "unknown_resource_attribution")
	if len(a.FindingIDs) != 0 || len(a.BlockingReasons) != 1 || a.BlockingReasons[0].FindingID == "" {
		t.Fatalf("unattributed condition was lost or attached to an inferred target: %+v", a)
	}

	conflict, fired := monitorInput(), monitorInput()
	conflict.TargetResourceID = resourcePrefix + "quiet"
	input["inventory"] = []Page{assessmentInventory("vm", "quiet", "other")}
	input["resource_health"][0].Health = append(input["resource_health"][0].Health, healthInput("other", "Available"))
	input["azure_monitor"] = []Page{{Monitor: []MonitorInput{conflict, fired}}}
	r = classifiedRun(t, input)
	assertSummary(t, r, "incomplete", 1, 0, 2)
	assertAssessment(t, r, "vm", "unknown", "conflicting_operational_state_or_target")
	assertAssessment(t, r, "quiet", "unknown", "conflicting_operational_state_or_target")
	assertAssessment(t, r, "other", "ok", "")

	fired.TargetResourceID = "/subscriptions/" + testSubscription + "/resourceGroups/Group/providers/Microsoft.OperationalInsights/workspaces/central"
	input["azure_monitor"] = []Page{{Monitor: []MonitorInput{fired}}}
	r = classifiedRun(t, input)
	assertSummary(t, r, "incomplete", 0, 0, 3)
	assertAssessment(t, r, "other", "unknown", "workspace_target_only")
}

func TestAssessmentMonitorWorkspaceAliasDoesNotNarrowImpact(t *testing.T) {
	const workspace = "/subscriptions/" + testSubscription + "/resourceGroups/Group/providers/Microsoft.OperationalInsights/workspaces/central"
	const target = "/subscriptions/" + testSubscription + "/resourceGroups/Group/providers/Microsoft.OperationalInsights/workſpaces/central"
	inventory := assessmentInventory("vm")
	inventory.Resources = append(inventory.Resources, ResourceInput{ID: workspace})
	workspaceHealth := healthInput("central", "Available")
	workspaceHealth.StatusID = workspace + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current"
	fired := monitorInput()
	fired.SignalType, fired.TargetResourceID = "Log", target
	r := classifiedRun(t, map[string][]Page{
		"inventory":       {inventory},
		"resource_health": {{Health: []HealthInput{healthInput("vm", "Available"), workspaceHealth}}},
		"azure_monitor":   {{Monitor: []MonitorInput{fired}}},
	})
	assertSummary(t, r, "incomplete", 0, 0, 2)
	a := assertAssessment(t, r, "vm", "unknown", "workspace_target_only")
	if len(a.FindingIDs) != 0 || len(a.BlockingReasons) != 1 || a.BlockingReasons[0].ResourceID != "" ||
		len(r.Findings) != 1 || r.Findings[0].ResourceID != target || a.BlockingReasons[0].FindingID != r.Findings[0].ID {
		t.Fatalf("workspace evidence was lost or assigned to a VM: %+v, findings %+v", a, r.Findings)
	}
}

func TestAssessmentEvaluatesAdvisorWorkflowAndScope(t *testing.T) {
	review := advisorInput("vm")
	unknown := advisorInput("quiet")
	unknown.Status = "Unsupported"
	input := map[string][]Page{
		"inventory":       {assessmentInventory("vm", "quiet")},
		"resource_health": {{Health: []HealthInput{healthInput("vm", "Available"), healthInput("quiet", "Available")}}},
		advisorSource:     {{Advisor: []AdvisorInput{review, unknown}}},
	}
	r := classifiedRun(t, input)
	assertSummary(t, r, "incomplete", 0, 1, 1)
	assertAssessment(t, r, "vm", "finding", "")
	assertAssessment(t, r, "quiet", "unknown", "unresolved_advisor_workflow")

	completed, dismissed := review, unknown
	completed.Status, completed.LastUpdated = "Completed", ptr(review.LastUpdated.Add(time.Hour))
	dismissed.Status, dismissed.LastUpdated = "Dismissed", ptr(unknown.LastUpdated.Add(time.Hour))
	input[advisorSource] = append(input[advisorSource], Page{Advisor: []AdvisorInput{completed, dismissed}})
	r = classifiedRun(t, input)
	assertSummary(t, r, "complete", 2, 0, 0)
	if len(r.Findings) != 0 || len(r.AdvisorInstances) != 2 {
		t.Fatalf("expected workflow history without New findings: %+v / %+v", r.Findings, r.AdvisorInstances)
	}

	dismissed.LastUpdated = unknown.LastUpdated
	input[advisorSource][1] = Page{Advisor: []AdvisorInput{completed, dismissed}}
	r = classifiedRun(t, input)
	assertSummary(t, r, "incomplete", 1, 0, 1)
	assertAssessment(t, r, "quiet", "unknown", "conflicting_advisor_state_or_target")

	review.TargetResourceID = ""
	input[advisorSource] = []Page{{Advisor: []AdvisorInput{review}}}
	r = classifiedRun(t, input)
	assertSummary(t, r, "incomplete", 0, 0, 2)
	a := assertAssessment(t, r, "quiet", "unknown", "unknown_resource_attribution")
	if len(a.FindingIDs) != 0 || len(a.BlockingReasons) != 1 || len(r.Findings) != 1 || a.BlockingReasons[0].FindingID != r.Findings[0].ID {
		t.Fatalf("unattributed Advisor New finding was lost or inferred: %+v / %+v", a, r.Findings)
	}
}

func TestAssessmentKeepsTiedYearZeroObservations(t *testing.T) {
	timestamp := time.Date(0, time.February, 1, 0, 0, 0, 0, time.UTC)
	newRec, dismissed := advisorInput("vm"), advisorInput("vm")
	newRec.LastUpdated, dismissed.LastUpdated = &timestamp, &timestamp
	dismissed.Status = "Dismissed"
	fired, resolved := monitorInput(), monitorInput()
	fired.LastModifiedTime, resolved.LastModifiedTime = &timestamp, &timestamp
	resolved.Condition = "Resolved"

	for _, tc := range []struct {
		source string
		page   Page
		reason string
	}{
		{advisorSource, Page{Advisor: []AdvisorInput{newRec, dismissed}}, "conflicting_advisor_state_or_target"},
		{"azure_monitor", Page{Monitor: []MonitorInput{fired, resolved}}, "conflicting_operational_state_or_target"},
	} {
		t.Run(tc.source, func(t *testing.T) {
			r := classifiedRun(t, map[string][]Page{
				"inventory":       {assessmentInventory("vm")},
				"resource_health": {{Health: []HealthInput{healthInput("vm", "Available")}}},
				tc.source:         {tc.page},
			})
			assertSummary(t, r, "incomplete", 0, 0, 1)
			a := assertAssessment(t, r, "vm", "unknown", tc.reason)
			if len(a.BlockingReasons) != 1 || len(a.BlockingReasons[0].EvidenceIDs) != 2 {
				t.Fatalf("both observations must support the conflict: %+v", a.BlockingReasons)
			}
		})
	}
}

func TestAssessmentAdvisorWorkflowSurvivesMetadataDisagreement(t *testing.T) {
	for _, tc := range []struct {
		status, disposition string
		ok, findings        int
	}{
		{"New", "finding", 0, 1},
		{"Completed", "ok", 1, 0},
	} {
		t.Run(tc.status, func(t *testing.T) {
			low, high := advisorInput("vm"), advisorInput("vm")
			low.Status, high.Status = tc.status, tc.status
			low.Impact, high.Impact = "Low", "High"
			high.Category, high.TypeID = "Performance", ""
			r := classifiedRun(t, map[string][]Page{
				"inventory":       {assessmentInventory("vm")},
				"resource_health": {{Health: []HealthInput{healthInput("vm", "Available")}}},
				advisorSource:     {{Advisor: []AdvisorInput{low, high}}},
			})
			assertSummary(t, r, "complete", tc.ok, tc.findings, 0)
			a := assertAssessment(t, r, "vm", tc.disposition, "")
			row := r.AdvisorInstances[0]
			if row.Status != tc.status || row.Impact != advisorUnknown || row.Category != advisorUnknown || row.TypeID != "" ||
				len(row.EvidenceIDs) != 2 || len(a.FindingIDs) != tc.findings || !hasGap(r, "conflicting_advisor_evidence") {
				t.Fatalf("workflow or metadata disagreement lost: row=%+v assessment=%+v", row, a)
			}
			if tc.findings == 1 && !slices.Equal(r.Findings[0].EvidenceIDs, row.EvidenceIDs) {
				t.Fatal("New finding must reference both agreeing workflow observations")
			}
		})
	}
}

func TestAssessmentAdvisorAgreedNewRetainsFindingAcrossTargetConflict(t *testing.T) {
	vm, quiet := advisorInput("vm"), advisorInput("quiet")
	vm.ID = advisorSubscription + advisorSuffix + "shared"
	quiet.ID = vm.ID
	input := map[string][]Page{
		"inventory":       {assessmentInventory("vm", "quiet", "other")},
		"resource_health": {{Health: []HealthInput{healthInput("vm", "Available"), healthInput("quiet", "Available"), healthInput("other", "Available")}}},
		advisorSource:     {{Advisor: []AdvisorInput{vm}}},
	}
	baseline := classifiedRun(t, input)
	wantID := baseline.Findings[0].ID
	pages := []Page{{Advisor: []AdvisorInput{vm}}, {Advisor: []AdvisorInput{quiet}}}
	for _, reverse := range []bool{false, true} {
		if reverse {
			slices.Reverse(pages)
		}
		input[advisorSource] = pages
		r := classifiedRun(t, input)
		assertSummary(t, r, "incomplete", 1, 0, 2)
		if len(r.Findings) != 1 || r.Findings[0].ID != wantID || r.Findings[0].ResourceID != "" || len(r.Findings[0].EvidenceIDs) != 2 {
			t.Fatalf("expected one New finding with stable identity and uncertain target: %+v", r.Findings)
		}
		row := r.AdvisorInstances[0]
		if row.Status != "New" || row.FindingID != wantID || row.AttributionLimitation != "conflicting_resource_attribution" {
			t.Fatalf("expected New workflow and explicit target conflict: %+v", row)
		}
		for _, name := range []string{"vm", "quiet"} {
			a := assertAssessment(t, r, name, "unknown", "conflicting_advisor_state_or_target")
			if len(a.FindingIDs) != 0 || len(a.BlockingReasons) != 1 || a.BlockingReasons[0].FindingID != wantID {
				t.Fatalf("uncertain attribution must link the shared finding: %+v", a)
			}
		}
		assertAssessment(t, r, "other", "ok", "")
	}
}

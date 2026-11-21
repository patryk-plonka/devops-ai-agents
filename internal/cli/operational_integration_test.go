package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

var implementedSources = []string{"inventory", "resource_health", "service_health", "azure_monitor", "advisor"}

const integrationAdvisorType = "33333333-3333-4333-8333-333333333333"

// Exercise production defaults, including discovery and credential subprocesses.
// integrationApp installs an isolated fake az and restores the global transport.
func nativeSourceApp(t *testing.T, wire armFixture) (App, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	a, out, diag, log := integrationApp(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected injected collector")
		return nil, nil
	})
	a.Validate, a.Inventory, a.ResourceHealth, a.ServiceHealth, a.AzureMonitor, a.Advisor = nil, nil, nil, nil, nil, nil
	http.DefaultTransport = armFixture(func(req *http.Request) (*http.Response, error) {
		validHost := req.URL.Host == "management.azure.com" || req.URL.Host == "management.azure.com:443"
		validScope := req.URL.Path == "/subscriptions/"+first || strings.HasPrefix(req.URL.Path, "/subscriptions/"+first+"/")
		if req.Method != "GET" || req.Body != nil || !validHost || !validScope || requestSource(req) == "" || req.Header.Get("Authorization") != "Bearer synthetic-token-canary" {
			t.Fatal("request escaped selected-subscription read boundary")
		}
		if _, ok := req.Context().Deadline(); !ok {
			t.Fatal("unbounded native operation")
		}
		return wire(req)
	})
	return a, out, diag, log
}

// Additional stress records use independently written wire fields. The literal
// official-example fixtures and provenance remain in internal/azure/testdata.
func operationalRecord(source string, i int, now time.Time) string {
	if source == "inventory" {
		return inventoryRecord(i)
	}
	if source == "resource_health" {
		return healthRecord(i)
	}
	stamp := now.UTC().Format(time.RFC3339Nano)
	if source == "advisor" {
		resource := integrationResource(i)
		target := resource
		if i == 0 {
			target = "/subscriptions/" + second + "/resourceGroups/g/providers/Microsoft.Compute/virtualMachines/advisor-foreign-canary"
		}
		impact := []string{"High", "Medium", "Low"}[i%3]
		return fmt.Sprintf(`{"id":%q,"properties":{"recommendationTypeId":%q,"category":"Cost","impact":%q,"recommendationStatus":"New","lastRefreshed":%q,"lastUpdated":%q,"resourceMetadata":{"resourceId":%q},"shortDescription":{"problem":"advisor-prose-canary"},"extendedProperties":{"secret":"advisor-properties-canary"}}}`,
			resource+"/providers/Microsoft.Advisor/recommendations/suggestion", integrationAdvisorType, impact, stamp, now.Add(-time.Hour).UTC().Format(time.RFC3339Nano), target)
	}
	if source == "service_health" {
		category := []string{"ServiceIssue", "PlannedMaintenance", "HealthAdvisory", "SecurityAdvisory"}[i%4]
		status := "Resolved"
		impactStart := now.Add(-time.Hour)
		if i < 4 {
			status = "Active"
			impactStart = now.Add(time.Hour)
		}
		return fmt.Sprintf(`{
			"id":"/providers/Microsoft.ResourceHealth/events/event-%03d",
			"properties":{
				"eventType":%q,"status":%q,"lastUpdateTime":%q,"impactStartTime":%q,
				"isEventSensitive":true,"description":"description-canary",
				"impact":[
					{"impactedService":"Compute | <script>[link](https://bad.example)","impactedRegions":[
						{"impactedRegion":"East US","status":%q,"impactedSubscriptions":[%q,%q],"impactedTenants":["tenant-canary"]},
						{"impactedRegion":"foreign-region-canary","impactedSubscriptions":[%q]}
					]},
					{"impactedService":"foreign-service-canary","impactedRegions":[
						{"impactedRegion":"foreign-only-region-canary","impactedSubscriptions":[%q]}
					]}
				]
			}
		}`, i, category, status, stamp, impactStart.Format(time.RFC3339Nano), status, first, second, second, second)
	}
	condition := "Resolved"
	if i < 3 {
		condition = "Fired"
	}
	target := integrationResource(i)
	if i == 0 {
		target = "/subscriptions/" + second + "/resourceGroups/g/providers/Microsoft.Compute/virtualMachines/foreign-canary"
	} else if i == 1 {
		target = ""
	} else if i == 2 {
		target = "/subscriptions/" + first + "/resourceGroups/g/providers/Microsoft.OperationalInsights/workspaces/logs"
	}
	return integrationMonitorRecord(i, stamp, condition, target)
}

func integrationMonitorRecord(i int, stamp, condition, target string) string {
	return fmt.Sprintf(`{"id":"/subscriptions/%s/providers/Microsoft.AlertsManagement/alerts/alert-%03d","properties":{"essentials":{"signalType":"Log","monitorCondition":%q,"alertState":%q,"severity":"Sev2","targetResource":%q,"lastModifiedDateTime":%q,"actionStatus":{"isSuppressed":true},"lastModifiedUserName":"username-canary","description":"description-canary"},"context":{"SearchQuery":"query-canary","SearchResults":"results-canary"}}}`, first, i, condition, []string{"New", "Acknowledged", "Closed"}[i%3], target, stamp)
}

func TestFiveSourceNativeScaleAndPaging(t *testing.T) {
	for _, pages := range []int{3, 24} {
		t.Run(fmt.Sprintf("pages-%d", pages), func(t *testing.T) {
			counts := map[string]int{}
			sequence := []string{}
			var a App
			a, out, diag, log := nativeSourceApp(t, func(req *http.Request) (*http.Response, error) {
				source := requestSource(req)
				counts[source]++
				if source == "subscription" {
					return armResponse(200, subscriptionResponse()), nil
				}
				sequence = append(sequence, source)
				page := counts[source] - 1
				if page >= pages {
					t.Fatal("unbounded paging")
				}
				saved := snapshot(t, a.Directory)
				if saved.Sources[source].Status != "in_progress" || saved.Sources[source].Pages != page {
					t.Fatal("native request preceded committed source/page state")
				}
				if source == "service_health" || source == "azure_monitor" || source == "advisor" {
					q := saved.Sources[source].Query
					if q == nil || q.RequestedAt == nil || !q.ReferenceAt.Equal(saved.StartedAt) {
						t.Fatal("native query provenance missing")
					}
					if page == 0 && source == "service_health" && req.URL.Query().Get("queryStartTime") != saved.StartedAt.Add(-30*24*time.Hour).UTC().Format("1/2/2006") {
						t.Fatal("wrong fixed date query")
					}
					if page == 0 && source == "azure_monitor" && req.URL.Query().Encode() != "api-version=2019-03-01&includeContext=false&includeEgressConfig=false&monitorCondition=Fired&pageCount=100&timeRange=30d" {
						t.Fatal("wrong Monitor query")
					}
					if source == "advisor" {
						if q.Operation != "Recommendations.List" || q.APIVersion != "2026-10-01" || q.Cutoff != nil || q.QueryStartTime != "" || q.TimeRange != "" {
							t.Fatal("wrong saved Advisor query policy")
						}
						wantQuery := url.Values{"api-version": {"2026-10-01"}}
						if page > 0 {
							wantQuery.Set("$skiptoken", fmt.Sprintf("page-%d-token-canary", page))
							wantQuery.Set("$top", "25")
						}
						if req.URL.Host != "management.azure.com" ||
							req.URL.Path != "/subscriptions/"+first+"/providers/Microsoft.Advisor/recommendations" ||
							req.URL.Query().Encode() != wantQuery.Encode() {
							t.Fatal("wrong Advisor request")
						}
					}
				}
				if page > 0 && source != "advisor" {
					key := "$skiptoken"
					if source == "service_health" {
						key = "$skipToken"
					} else if source == "azure_monitor" {
						key = "ctoken"
					}
					if req.URL.Query().Get(key) != fmt.Sprintf("page-%d-token-canary", page) {
						t.Fatal("continuation lost")
					}
				}
				size := 2
				if source == "inventory" || source == "resource_health" {
					size = 120
				}
				if source == "advisor" {
					size = 10
				}
				if pages == 24 {
					size = 1
				}
				records := []string{}
				for i := page * size; i < (page+1)*size; i++ {
					records = append(records, operationalRecord(source, i, saved.StartedAt))
				}
				next := ""
				if page+1 < pages {
					next = fmt.Sprintf("page-%d-token-canary", page+1)
				}
				return armResponse(200, integrationList(source, records, next)), nil
			})
			if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first, "--debug"}); code != 0 {
				t.Fatalf("exit %d: %s", code, diag)
			}
			r := assertIntegrationArtifacts(t, a, out, diag, "incomplete", "Incomplete coverage · ")
			wantSequence := make([]string, 0, len(implementedSources)*pages)
			for _, source := range implementedSources {
				for range pages {
					wantSequence = append(wantSequence, source)
				}
			}
			if !slices.Equal(sequence, wantSequence) {
				t.Fatalf("wrong page order: got %v, want %v", sequence, wantSequence)
			}
			wantResources, wantEvidence, wantAdvisor := 360, 762, 30
			if pages == 24 {
				wantResources, wantEvidence, wantAdvisor = 24, 120, 24
			}
			if len(r.Resources) != wantResources || len(r.Observations) != wantResources || len(r.Evidence) != wantEvidence {
				t.Fatalf("scale loss: resources %d observations %d evidence %d", len(r.Resources), len(r.Observations), len(r.Evidence))
			}
			if want := (health.AssessmentSummary{Unknown: wantResources}); *r.AssessmentSummary != want {
				t.Fatalf("ambiguous attribution counts: got %+v, want %+v", *r.AssessmentSummary, want)
			}
			findings := map[string]int{}
			for _, f := range r.Findings {
				findings[f.Source]++
			}
			if findings["service_health"] != 4 || findings["azure_monitor"] != 3 || findings["advisor"] != wantAdvisor {
				t.Fatalf("findings by source: %v", findings)
			}
			advisor := r.Sources["advisor"]
			if advisor.Query == nil || advisor.Query.RequestedAt == nil ||
				advisor.FirstPageAt == nil || advisor.LastPageAt == nil ||
				advisor.FirstPageAt.Before(*advisor.Query.RequestedAt) || advisor.LastPageAt.Before(*advisor.FirstPageAt) {
				t.Fatal("Advisor accepted page timing missing or out of order")
			}
			if len(r.AdvisorInstances) != wantAdvisor {
				t.Fatalf("Advisor instances %d want %d", len(r.AdvisorInstances), wantAdvisor)
			}
			for _, row := range r.AdvisorInstances {
				if row.Status != "New" || row.TypeID != integrationAdvisorType || row.FindingID == "" || len(row.EvidenceIDs) != 1 {
					t.Fatal("Advisor grouping lost an individual finding or evidence reference")
				}
				if strings.HasPrefix(row.EntityID, strings.ToLower(integrationResource(0))+"/") && row.ResourceID != "" {
					t.Fatal("foreign Advisor target retained")
				}
			}
			reportBytes, err := os.ReadFile(strings.TrimSpace(out.String()))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(reportBytes), fmt.Sprintf("**Type %s (%d)**", integrationAdvisorType, wantAdvisor)) {
				t.Fatal("Advisor group count missing from published report")
			}
			for _, source := range implementedSources {
				if r.Sources[source].Status != "completed" || r.Sources[source].Pages != pages || counts[source] != pages {
					t.Fatalf("source %s: %+v requests %v", source, r.Sources[source], counts)
				}
			}
			for _, e := range r.Evidence {
				if e.Source == "service_health" {
					impact := e.ServiceHealth.Impact
					if len(impact) != 1 || len(impact[0].Regions) != 1 || impact[0].Regions[0].Attribution != "selected_subscription" {
						t.Fatal("selected service and region were not retained")
					}
				}
				if e.Source == "service_health" && e.ResourceID != "" || e.Source == "azure_monitor" && strings.HasSuffix(e.EntityID, "alert-000") && e.ResourceID != "" {
					t.Fatal("invented resource attribution")
				}
			}
			assertGap(t, r, "conflicting_subscription_attribution")
			assertGap(t, r, "service_health_impact_undetermined")
			assertSyntheticLogin(t, log, true)
		})
	}
}

func TestFiveSourceNativeRepresentativeAssessments(t *testing.T) {
	const pageSize = 120
	const resourceCount = 3 * pageSize
	const firstAlertVM = pageSize - 1
	const secondAlertVM = 2 * pageSize
	for _, tc := range []struct {
		name, coverage, headline string
		counts                   health.AssessmentSummary
		healthRows               int
	}{
		{"mixed", "incomplete", "Incomplete coverage · 358 resources OK, 1 with findings, 1 unknown.", health.AssessmentSummary{OK: 358, Finding: 1, Unknown: 1}, 359},
		{"complete", "complete", "Complete coverage · 358 resources OK, 2 with findings, 0 unknown.", health.AssessmentSummary{OK: 358, Finding: 2}, 360},
	} {
		t.Run(tc.name, func(t *testing.T) {
			pages := map[string]int{"inventory": 3, "resource_health": 3, "service_health": 2, "azure_monitor": 2, "advisor": 1}
			requests := map[string]int{}
			var a App
			a, out, diag, log := nativeSourceApp(t, func(req *http.Request) (*http.Response, error) {
				source := requestSource(req)
				if source == "subscription" {
					return armResponse(200, subscriptionResponse()), nil
				}
				page := requests[source]
				requests[source]++
				saved := snapshot(t, a.Directory)
				if page >= pages[source] || saved.Sources[source].Pages != page {
					t.Fatalf("unexpected %s page %d at revision %d", source, page, saved.Revision)
				}
				var records []string
				switch source {
				case "inventory", "resource_health":
					// Deliver descending IDs so the output must establish its own order.
					for offset := range pageSize {
						i := resourceCount - 1 - page*pageSize - offset
						if source == "inventory" {
							records = append(records, inventoryRecord(i))
						} else if i != secondAlertVM || tc.counts.Unknown == 0 {
							records = append(records, integrationHealthRecord(i, "current", `{"availabilityState":"Available"}`))
						}
					}
				case "service_health":
					// Indices 4 and 5 are resolved events within the saved window.
					records = []string{operationalRecord(source, 4+page, saved.StartedAt)}
				case "azure_monitor":
					target := []int{firstAlertVM, secondAlertVM}[page]
					records = []string{integrationMonitorRecord(page, saved.StartedAt.Format(time.RFC3339Nano), "Fired", integrationResource(target))}
				}
				next := ""
				if page+1 < pages[source] {
					next = fmt.Sprintf("page-%d-token-canary", page+1)
				}
				return armResponse(200, integrationList(source, records, next)), nil
			})
			if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first, "--debug"}); code != 0 {
				t.Fatalf("exit %d: %s", code, diag)
			}
			r := assertIntegrationArtifacts(t, a, out, diag, tc.coverage, tc.headline)
			if r.Outcome != "completed" || *r.AssessmentSummary != tc.counts || len(r.Resources) != resourceCount || len(r.ResourceAssessments) != resourceCount {
				t.Fatalf("scale classification: outcome %s, counts %+v, inventory %d, assessments %d", r.Outcome, *r.AssessmentSummary, len(r.Resources), len(r.ResourceAssessments))
			}
			if !strings.Contains(diag.String(), fmt.Sprintf("(revision %d).", r.Revision)) {
				t.Fatal("CLI did not identify the committed revision")
			}
			for source, want := range pages {
				s := r.Sources[source]
				if requests[source] != want || s.Pages != want || s.Status != "completed" || !r.HasAssessmentSourceProof(source) {
					t.Fatalf("missing %s terminal query proof: %+v, requests %d", source, s, requests[source])
				}
			}
			evidence := make(map[string]health.Evidence, len(r.Evidence))
			evidenceCounts := map[string]int{}
			for _, e := range r.Evidence {
				evidence[e.ID] = e
				evidenceCounts[e.Source]++
				if e.Source == "service_health" {
					event := e.ServiceHealth
					if event == nil || event.Status != "Resolved" || event.ImpactStartTime == nil || !event.ImpactStartTime.Before(r.StartedAt) ||
						len(event.Impact) != 1 || len(event.Impact[0].Regions) != 1 || event.Impact[0].Regions[0].Status != "Resolved" {
						t.Fatal("Service Health lost the resolved event or region state")
					}
				}
			}
			if len(r.AdvisorInstances) != 0 || evidenceCounts["advisor"] != 0 || evidenceCounts["service_health"] != 2 ||
				evidenceCounts["azure_monitor"] != 2 || evidenceCounts["resource_health"] != tc.healthRows || evidenceCounts["inventory"] != resourceCount {
				t.Fatalf("unexpected source evidence counts: %v", evidenceCounts)
			}
			if len(r.Findings) != 2 {
				t.Fatalf("got %d findings, want exactly two Monitor findings", len(r.Findings))
			}
			findings := map[string]health.Finding{}
			for _, f := range r.Findings {
				if f.Source != "azure_monitor" || evidence[f.EvidenceIDs[0]].Monitor == nil || evidence[f.EvidenceIDs[0]].Monitor.Condition != "Fired" {
					t.Fatal("finding is missing its Fired Monitor evidence")
				}
				findings[strings.ToLower(f.ResourceID)] = f
			}
			b, err := os.ReadFile(strings.TrimSpace(out.String()))
			if err != nil {
				t.Fatal(err)
			}
			reportText := html.UnescapeString(string(b))
			_, table, _ := strings.Cut(reportText, "\n## Resource assessments\n")
			table, _, _ = strings.Cut(table, "\n## ")
			var rows []string
			for line := range strings.SplitSeq(table, "\n") {
				if strings.HasPrefix(line, "| /subscriptions/") {
					rows = append(rows, line)
				}
			}
			if len(rows) != resourceCount || !strings.Contains(reportText, "The completed cached list contained no recommendations") {
				t.Fatalf("published table has %d resource rows or is missing the empty Advisor result", len(rows))
			}
			for i, assessment := range r.ResourceAssessments {
				resource := strings.ToLower(integrationResource(i))
				wantDisposition, wantLabel := "ok", "OK"
				f := findings[resource]
				if i == firstAlertVM || i == secondAlertVM {
					wantDisposition, wantLabel = "finding", "Finding"
					if f.ID == "" || !slices.Equal(assessment.FindingIDs, []string{f.ID}) || !strings.Contains(rows[i], "["+f.ID+"](run.json)") {
						t.Fatalf("resource %d lost its Fired finding", i)
					}
				} else if len(assessment.FindingIDs) != 0 {
					t.Fatalf("unexpected finding references for resource %d", i)
				}
				if i == secondAlertVM && tc.counts.Unknown != 0 {
					wantDisposition, wantLabel = "unknown", "Unknown"
					if len(assessment.BlockingReasons) != 1 || assessment.BlockingReasons[0].Source != "resource_health" ||
						assessment.BlockingReasons[0].Reason != "missing_health_evidence" || !strings.Contains(rows[i], "resource_health · missing_health_evidence") {
						t.Fatal("alerted unknown VM lost its missing Resource Health reason")
					}
				}
				if assessment.ResourceID != resource || assessment.Disposition != wantDisposition ||
					!strings.HasPrefix(rows[i], "| "+resource+" | "+wantLabel+" | ") {
					t.Fatalf("saved or rendered assessment %d is out of order or has the wrong disposition", i)
				}
				assertRepresentativeChecks(t, assessment, rows[i], evidence, f)
			}
			assertSyntheticLogin(t, log, true)
		})
	}
}

func assertRepresentativeChecks(t *testing.T, a health.ResourceAssessment, row string, evidence map[string]health.Evidence, f health.Finding) {
	t.Helper()
	if len(a.Checks) != len(implementedSources) {
		t.Fatalf("%s has %d checks", a.ResourceID, len(a.Checks))
	}
	for i, source := range implementedSources {
		check := a.Checks[i]
		if check.SourceRef != "sources."+source || !strings.Contains(row, "["+check.SourceRef+"](run.json)") {
			t.Fatalf("%s lost its %s source reference", a.ResourceID, source)
		}
		want := 0
		if source == "inventory" || source == "resource_health" && a.Disposition != "unknown" || source == "azure_monitor" && f.ID != "" {
			want = 1
		}
		if len(check.EvidenceIDs) != want {
			t.Fatalf("%s has %d %s evidence links, want %d", a.ResourceID, len(check.EvidenceIDs), source, want)
		}
		for _, id := range check.EvidenceIDs {
			e := evidence[id]
			if e.Source != source || !strings.EqualFold(e.ResourceID, a.ResourceID) || !strings.Contains(row, "["+id+"](run.json)") {
				t.Fatalf("%s has an incorrect %s evidence link", a.ResourceID, source)
			}
			if source == "resource_health" && (e.State != "Available" || e.StatusRole != "current") ||
				source == "azure_monitor" && !slices.Contains(f.EvidenceIDs, id) {
				t.Fatalf("%s has incorrect deciding evidence for %s", a.ResourceID, source)
			}
		}
	}
}

func TestFiveSourceNativeFailureIsolation(t *testing.T) {
	for _, mask := range []int{0, 4, 8, 12, 3, 15, 16, 31} {
		t.Run(fmt.Sprintf("failures-%05b", mask), func(t *testing.T) {
			counts := map[string]int{}
			var a App
			a, out, diag, log := nativeSourceApp(t, func(req *http.Request) (*http.Response, error) {
				source := requestSource(req)
				counts[source]++
				if source == "subscription" {
					return armResponse(200, subscriptionResponse()), nil
				}
				for i, name := range implementedSources {
					if name == source && mask&(1<<i) != 0 {
						status := 403
						if source == "service_health" {
							status = 401 // Local authentication failure must not stop Monitor.
						}
						return armResponse(status, `{"error":{"code":"Denied","message":"body-canary"}}`), nil
					}
				}
				return armResponse(200, integrationList(source, []string{operationalRecord(source, 0, snapshot(t, a.Directory).StartedAt)}, "")), nil
			})
			wantCode, outcome := 3, "partial"
			if mask == 0 {
				wantCode, outcome = 0, "completed"
			} else if mask == 31 {
				wantCode, outcome = 1, "failed"
			}
			if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first, "--debug"}); code != wantCode {
				t.Fatalf("exit %d want %d: %s", code, wantCode, diag)
			}
			r := assertIntegrationArtifacts(t, a, out, diag, "incomplete", "Incomplete coverage · ")
			wantEvidence := len(implementedSources)
			for i, source := range implementedSources {
				wantStatus, reason := "completed", ""
				if mask&(1<<i) != 0 {
					wantEvidence--
					wantStatus, reason = "failed", "access_denied"
					if source == "service_health" {
						reason = "login_required"
					}
				}
				if counts[source] != 1 || r.Sources[source].Status != wantStatus || r.Sources[source].Reason != reason {
					t.Fatalf("source isolation lost: %s %+v requests %v", source, r.Sources[source], counts)
				}
			}
			if r.Outcome != outcome || len(r.Evidence) != wantEvidence {
				t.Fatal("outcome/evidence lost")
			}
			if mask&16 == 0 && len(r.AdvisorInstances) != 1 {
				t.Fatal("earlier source failure lost the Advisor recommendation")
			}
			if mask&16 != 0 {
				b, err := os.ReadFile(strings.TrimSpace(out.String()))
				if err != nil {
					t.Fatal(err)
				}
				reportText := string(b)
				if len(r.AdvisorInstances) != 0 || !strings.Contains(reportText, "The cached list could not be read") ||
					strings.Contains(reportText, "The completed cached list contained no recommendations") {
					t.Fatal("Advisor denial did not remain explicit in the report")
				}
			}
			assertSyntheticLogin(t, log, true)
		})
	}
	t.Run("all empty", func(t *testing.T) {
		counts := map[string]int{}
		var sequence []string
		a, out, diag, _ := nativeSourceApp(t, func(req *http.Request) (*http.Response, error) {
			source := requestSource(req)
			counts[source]++
			if source == "subscription" {
				return armResponse(200, subscriptionResponse()), nil
			}
			sequence = append(sequence, source)
			return armResponse(200, `{"value":[]}`), nil
		})
		if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first}); code != 0 {
			t.Fatalf("exit %d: %s", code, diag)
		}
		r := assertIntegrationArtifacts(t, a, out, diag, "incomplete", "Incomplete coverage · ")
		if !slices.Equal(sequence, implementedSources) {
			t.Fatalf("wrong empty source order: got %v, want %v", sequence, implementedSources)
		}
		if len(r.Evidence) != 0 || len(r.Findings) != 0 || r.Outcome != "completed" {
			t.Fatal("empty lists invented evidence")
		}
		for _, source := range implementedSources {
			if counts[source] != 1 || r.Sources[source].Status != "completed" || r.Sources[source].Pages != 1 {
				t.Fatalf("empty source %s: %+v", source, r.Sources[source])
			}
		}
		assertGap(t, r, "no_resources_observed")
		b, err := os.ReadFile(strings.TrimSpace(out.String()))
		if err != nil {
			t.Fatal(err)
		}
		advisor := r.Sources["advisor"]
		if advisor.FirstPageAt == nil || advisor.LastPageAt == nil || !advisor.FirstPageAt.Equal(*advisor.LastPageAt) ||
			!strings.Contains(string(b), "The completed cached list contained no recommendations") {
			t.Fatal("successful empty Advisor page or its provenance missing")
		}
	})
}

func TestFiveSourceNativeOperationalPageFailures(t *testing.T) {
	for _, source := range []string{"service_health", "azure_monitor", "advisor"} {
		scenarios := []string{"401", "403", "404", "429", "timeout", "unsafe host", "wrong subscription", "redirect", "repeat", "invalid envelope", "alternate-case continuation", "mixed records"}
		if source == "advisor" {
			scenarios = []string{"403", "wrong subscription", "invalid envelope", "alternate-case continuation", "mixed records"}
		}
		for _, scenario := range scenarios {
			t.Run(source+"/"+scenario, func(t *testing.T) {
				counts := map[string]int{}
				var a App
				a, out, diag, _ := nativeSourceApp(t, func(req *http.Request) (*http.Response, error) {
					route := requestSource(req)
					counts[route]++
					if route == "subscription" {
						return armResponse(200, subscriptionResponse()), nil
					}
					record := operationalRecord(route, 0, snapshot(t, a.Directory).StartedAt)
					if route != source {
						return armResponse(200, integrationList(route, []string{record}, "")), nil
					}
					if counts[route] == 1 {
						body := integrationList(route, []string{record}, "next-token-canary")
						if scenario == "unsafe host" {
							body = strings.Replace(body, "https://management.azure.com", "https://evil.example", 1)
						} else if scenario == "wrong subscription" {
							body = strings.ReplaceAll(body, "management.azure.com/subscriptions/"+first, "management.azure.com/subscriptions/"+second)
							body = strings.ReplaceAll(body, "management.azure.com:443/subscriptions/"+first, "management.azure.com:443/subscriptions/"+second)
						}
						return armResponse(200, body), nil
					}
					if snapshot(t, a.Directory).Sources[source].Pages != 1 {
						t.Fatal("request/retry before committed page")
					}
					switch scenario {
					case "401", "403", "404", "429":
						status := map[string]int{"401": 401, "403": 403, "404": 404, "429": 429}[scenario]
						return armResponse(status, `{"error":{"code":"Denied","message":"body-canary"}}`), nil
					case "timeout":
						return nil, context.DeadlineExceeded
					case "redirect":
						r := armResponse(307, "redirect-canary")
						r.Header.Set("Location", "https://evil.example")
						return r, nil
					case "repeat":
						return armResponse(200, integrationList(route, []string{record}, "next-token-canary")), nil
					case "invalid envelope":
						return armResponse(200, `{"value":"body-canary"}`), nil
					case "alternate-case continuation":
						body := integrationList(route, nil, "alternate-token-canary")
						return armResponse(200, strings.Replace(body, `"nextLink":`, `"NextLink":`, 1)), nil
					case "mixed records":
						if source == "advisor" {
							record = operationalRecord(route, 1, snapshot(t, a.Directory).StartedAt)
						}
						foreign := strings.ReplaceAll(record, first, second)
						if source == "service_health" {
							foreign = strings.Replace(record, "/providers/Microsoft.ResourceHealth/events/", "/subscriptions/"+second+"/providers/Microsoft.ResourceHealth/events/", 1)
						}
						return armResponse(200, integrationList(route, []string{record, foreign, `{"id":"rejected-canary","properties":42}`}, "")), nil
					}
					t.Fatal("unsafe continuation reached wire")
					return nil, nil
				})
				if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first, "--debug"}); code != 3 {
					t.Fatalf("exit %d: %s", code, diag)
				}
				r := assertIntegrationArtifacts(t, a, out, diag, "incomplete", "Incomplete coverage · ")
				reason, calls, pages := "invalid_response", 2, 1
				switch scenario {
				case "401":
					reason = "login_required"
				case "403":
					reason = "access_denied"
				case "404":
					reason = "source_unavailable"
				case "429":
					reason, calls = "throttled", 5
				case "timeout":
					reason, calls = "timeout", 5
				case "unsafe host", "wrong subscription":
					reason, calls = "unsafe_continuation", 1
				case "redirect":
					reason = "read_boundary_rejected"
				case "repeat":
					reason, pages = "repeated_continuation", 2
				case "mixed records":
					reason, pages = "invalid_records", 2
					if r.Sources[source].RejectedRecords != 2 {
						t.Fatal("malformed/foreign siblings not counted")
					}
				}
				wantEvidence, wantFindings := 5, 4
				if source == "advisor" {
					wantInstances := 1
					if scenario == "mixed records" {
						wantEvidence, wantFindings, wantInstances = 6, 5, 2
					}
					if len(r.AdvisorInstances) != wantInstances {
						t.Fatal("partial Advisor collection lost valid recommendations")
					}
					b, err := os.ReadFile(strings.TrimSpace(out.String()))
					if err != nil {
						t.Fatal(err)
					}
					if !strings.Contains(string(b), "Coverage is limited to accepted pages") {
						t.Fatal("partial Advisor coverage hidden in report")
					}
				}
				if r.Sources[source].Status != "partial" || r.Sources[source].Reason != reason || r.Sources[source].Pages != pages || counts[source] != calls || len(r.Evidence) != wantEvidence || len(r.Findings) != wantFindings {
					t.Fatalf("lost retained evidence/state: source %+v requests %v evidence %d findings %d", r.Sources[source], counts, len(r.Evidence), len(r.Findings))
				}
				for _, other := range implementedSources {
					if other != source && (counts[other] != 1 || r.Sources[other].Status != "completed") {
						t.Fatal("operational failure suppressed another native collector")
					}
				}
				assertGap(t, r, reason)
			})
		}
	}
}

func TestFiveSourceNativeStorageAndPublicationFailures(t *testing.T) {
	for _, mode := range []string{"service_health", "azure_monitor", "advisor", "final write", "publication"} {
		t.Run(mode, func(t *testing.T) {
			counts := map[string]int{}
			var previousRevision int
			var a App
			a, out, diag, _ := nativeSourceApp(t, func(req *http.Request) (*http.Response, error) {
				source := requestSource(req)
				counts[source]++
				if source == "subscription" {
					return armResponse(200, subscriptionResponse()), nil
				}
				next := ""
				if source == mode {
					next = "next-token-canary"
					if counts[source] > 2 {
						t.Fatal("continued after failed checkpoint")
					}
				}
				saved := snapshot(t, a.Directory)
				if source == mode && counts[source] == 2 {
					previousRevision = saved.Revision
				}
				return armResponse(200, integrationList(source, []string{operationalRecord(source, counts[source]-1, saved.StartedAt)}, next)), nil
			})
			a.NewStore = func(root string, r *health.Run) (Storage, error) {
				s, err := runstore.New(root, r)
				if mode == "final write" || mode == "publication" {
					return publicationStore{Storage: s, finalWriteFailure: mode == "final write"}, err
				}
				return operationalFailureStore{Storage: s, source: mode}, err
			}
			if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first, "--debug"}); code != 1 || out.Len() != 0 || !strings.Contains(diag.String(), "Last saved snapshot:") || strings.Contains(diag.String(), "canary") {
				t.Fatalf("failed storage/publication feedback: exit %d %s", code, diag)
			}
			r := snapshot(t, a.Directory)
			want := len(implementedSources)
			for i, source := range implementedSources {
				if source != mode {
					continue
				}
				want = i + 1
				if counts[mode] != 2 || r.Sources[mode].Pages != 1 || r.Revision != previousRevision {
					t.Fatal("failed page replaced committed page")
				}
				for _, later := range implementedSources[i+1:] {
					if counts[later] != 0 || r.Sources[later].Reason != "not_attempted" {
						t.Fatal("storage failure did not stop later collectors")
					}
				}
			}
			if len(r.Evidence) != want || (r.EndedAt != nil) != (mode == "publication") {
				t.Fatal("last usable native checkpoint lost")
			}
			if r.Coverage != "incomplete" || r.AssessmentSummary == nil || *r.AssessmentSummary != (health.AssessmentSummary{Unknown: 1}) {
				t.Fatalf("failed native run classification: %s %+v", r.Coverage, r.AssessmentSummary)
			}
			if mode == "advisor" || mode == "final write" || mode == "publication" {
				if len(r.AdvisorInstances) != 1 || r.Sources["advisor"].Pages != 1 {
					t.Fatal("Advisor checkpoint lost")
				}
			}
			saved, err := os.ReadFile(filepath.Join(a.Directory, "runs", r.ID, "run.json"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(saved), "canary") || strings.Contains(string(saved), second) {
				t.Fatal("private fields leaked into retained checkpoint")
			}
			if _, err := os.Stat(filepath.Join(a.Directory, "runs", r.ID, "report.md")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("claimed a report after storage/publication failure")
			}
		})
	}
}

func TestFiveSourceNativeUnknownAndConflictingObservations(t *testing.T) {
	counts := map[string]int{}
	var a App
	a, out, diag, _ := nativeSourceApp(t, func(req *http.Request) (*http.Response, error) {
		source := requestSource(req)
		counts[source]++
		if source == "subscription" {
			return armResponse(200, subscriptionResponse()), nil
		}
		if source == "inventory" || source == "resource_health" || source == "advisor" {
			return armResponse(200, `{"value":[]}`), nil
		}
		saved := snapshot(t, a.Directory)
		if saved.Sources[source].Pages != counts[source]-1 {
			t.Fatal("conflicting observation requested before checkpoint")
		}
		base := operationalRecord(source, 0, saved.StartedAt)
		inactive := strings.Replace(base, `"status":"Active"`, `"status":"Resolved"`, 1)
		unknown := strings.Replace(base, `"eventType":"ServiceIssue"`, `"eventType":"FutureType"`, 1)
		if source == "azure_monitor" {
			inactive = strings.Replace(base, `"monitorCondition":"Fired"`, `"monitorCondition":"Resolved"`, 1)
			unknown = strings.Replace(base, `"signalType":"Log"`, `"signalType":"FutureType"`, 1)
		}
		unknown = strings.ReplaceAll(unknown, "-000", "-unknown")
		// A separate entity with a unique later Resolved update must supersede
		// its old active state. The original entity has a tied contradiction.
		ordered := strings.ReplaceAll(base, "-000", "-ordered")
		later := strings.ReplaceAll(inactive, "-000", "-ordered")
		later = strings.ReplaceAll(later, saved.StartedAt.Format(time.RFC3339Nano), saved.StartedAt.Add(time.Minute).Format(time.RFC3339Nano))
		if counts[source] == 1 {
			return armResponse(200, integrationList(source, []string{base, unknown, ordered}, "conflict-token-canary")), nil
		}
		if counts[source] != 2 {
			t.Fatal("unexpected conflicting page")
		}
		return armResponse(200, integrationList(source, []string{base, inactive, later}, "")), nil
	})
	if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first, "--debug"}); code != 0 {
		t.Fatalf("exit %d: %s", code, diag)
	}
	r := assertIntegrationArtifacts(t, a, out, diag, "incomplete", "Incomplete coverage · ")
	if len(r.Evidence) != 10 || len(r.Findings) != 2 || len(r.Resources) != 0 || len(r.Observations) != 0 {
		t.Fatalf("conflict/dedup semantics lost: evidence %d findings %d", len(r.Evidence), len(r.Findings))
	}
	for _, f := range r.Findings {
		if !strings.HasSuffix(f.EntityID, "-000") {
			t.Fatal("finding did not retain the known active side of the tied conflict")
		}
	}
	for _, source := range []string{"service_health", "azure_monitor"} {
		if r.Sources[source].Status != "completed" || r.Sources[source].UnclassifiedRecords != 1 || r.Sources[source].Pages != 2 {
			t.Fatalf("unknown domain fact became enumeration failure: %+v", r.Sources[source])
		}
		for _, reason := range []string{"conflicting_operational_evidence", "superseded_operational_evidence"} {
			found := false
			for _, gap := range r.Gaps {
				if gap.Source == source && gap.Reason == reason && len(gap.EvidenceIDs) > 0 {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing %s %s", source, reason)
			}
		}
	}
}

func TestFiveSourceNativeServiceHealthRegionalConflict(t *testing.T) {
	var a App
	a, out, diag, _ := nativeSourceApp(t, func(req *http.Request) (*http.Response, error) {
		switch requestSource(req) {
		case "subscription":
			return armResponse(200, subscriptionResponse()), nil
		case "inventory":
			return armResponse(200, integrationList("inventory", []string{inventoryRecord(0), inventoryRecord(1)}, "")), nil
		case "resource_health":
			available := `{"availabilityState":"Available"}`
			return armResponse(200, integrationList("resource_health", []string{
				integrationHealthRecord(0, "current", available), integrationHealthRecord(1, "current", available),
			}, "")), nil
		case "service_health":
			start := snapshot(t, a.Directory).StartedAt
			resolvedAt := start.Add(-time.Hour).Format(time.RFC3339Nano)
			regionAt := start.Add(-time.Minute).Format(time.RFC3339Nano)
			const event = `{"id":"/providers/Microsoft.ResourceHealth/events/regional-conflict","properties":{
				"eventType":"ServiceIssue","status":"Resolved","lastUpdateTime":%q,
				"impact":[{"impactedService":"Compute","impactedRegions":[{
					"impactedRegion":"West Europe","impactedSubscriptions":[%q],"status":"Active","lastUpdateTime":%q
				}]}]}}`
			return armResponse(200, integrationList("service_health", []string{fmt.Sprintf(event, resolvedAt, first, regionAt)}, "")), nil
		default:
			return armResponse(200, `{"value":[]}`), nil
		}
	})
	if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first}); code != 0 {
		t.Fatalf("exit %d: %s", code, diag)
	}
	r := assertIntegrationArtifacts(t, a, out, diag, "incomplete", "Incomplete coverage · 0 resources OK, 0 with findings, 2 unknown")
	if *r.AssessmentSummary != (health.AssessmentSummary{Unknown: 2}) || len(r.Findings) != 0 || r.Sources["service_health"].Status != "completed" {
		t.Fatalf("regional contradiction should withhold OK while preserving completed collection: %+v", r.AssessmentSummary)
	}
	const reason = "conflicting_service_health_regional_state"
	var blockers []health.Gap
	for _, gap := range r.Gaps {
		if gap.Blocking {
			blockers = append(blockers, gap)
		}
	}
	if len(blockers) != 1 || blockers[0].Reason != reason || blockers[0].ResourceID != "" || len(blockers[0].EvidenceIDs) != 1 {
		t.Fatalf("expected one subscription-scoped blocker with one evidence reference: %+v", blockers)
	}
	for _, assessment := range r.ResourceAssessments {
		if assessment.Disposition != "unknown" || len(assessment.BlockingReasons) != 1 || assessment.BlockingReasons[0].Reason != reason {
			t.Fatalf("resource missed subscription-wide conflict: %+v", assessment)
		}
	}
	b, err := os.ReadFile(strings.TrimSpace(out.String()))
	if err != nil {
		t.Fatal(err)
	}
	markdown := html.UnescapeString(string(b))
	for _, want := range []string{reason, "source status: Resolved", "status: Active; attribution: selected_subscription", "[" + blockers[0].EvidenceIDs[0] + "](run.json)"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("report omitted %q", want)
		}
	}
}

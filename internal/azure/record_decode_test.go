package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

func TestMixedSchemaPagesRetainValidRecordsAndCheckpoint(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute} {
		t.Run(route, func(t *testing.T) {
			run := health.NewRun("test", "test")
			run.Scope = &health.Scope{SubscriptionID: selectedID}
			store, err := runstore.New(t.TempDir(), &run)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			calls := 0
			c := collection(t, func(req *http.Request) (*http.Response, error) {
				calls++
				if calls == 2 {
					b, e := os.ReadFile(store.Path())
					if e != nil {
						t.Fatal(e)
					}
					var saved health.Run
					if json.Unmarshal(b, &saved) != nil {
						t.Fatal("invalid checkpoint")
					}
					if len(saved.Evidence) != 1 || saved.Sources[route].RejectedRecords != 2 || saved.Sources[route].Pages != 1 {
						t.Fatalf("valid siblings not checkpointed before next page: %+v", saved)
					}
					return response(200, `{"value":[]}`), nil
				}
				if calls > 2 {
					t.Fatal("unexpected retry")
				}
				good := fmt.Sprintf(`{"id":%q,"name":"vm","tags":{"endpoint":"rediss://app:credential-canary@cache.example:6380"}}`, collectionVM)
				bad := `{"id":42,"name":"malformed-record-canary"}`
				if route == healthRoute {
					good = fmt.Sprintf(`{"id":%q,"properties":{"availabilityState":"Unavailable"}}`, collectionVM+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current")
					bad = fmt.Sprintf(`{"id":%q,"properties":{"availabilityState":{"bad":"malformed-record-canary"}}}`, collectionVM+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current")
				}
				return response(200, fmt.Sprintf(`{"value":[%s,%s,null],"nextLink":%q}`, good, bad, listURL(route, "next"))), nil
			})
			collect := c.Inventory
			if route == healthRoute {
				collect = c.ResourceHealth
			}
			complete := func(context.Context, string, func(health.Page) error) error { return nil }
			inventory, resourceHealth := complete, complete
			if route == inventoryRoute {
				inventory = collect
			} else {
				resourceHealth = collect
			}
			code, err := health.Check(context.Background(), &run, []health.NamedCollector{{Name: "inventory", Collect: inventory}, {Name: "resource_health", Collect: resourceHealth}}, store.Save)
			if err != nil || code != 3 || calls != 2 || run.Outcome != "partial" || len(run.Evidence) != 1 || run.Sources[route].RejectedRecords != 2 || run.Sources[route].Pages != 2 || run.Sources[route].Reason != "invalid_records" {
				t.Fatalf("lost valid siblings: code %d error %v calls %d run %+v", code, err, calls, run)
			}
			b, e := os.ReadFile(store.Path())
			if e != nil {
				t.Fatal(e)
			}
			if strings.Contains(string(b), "malformed-record-canary") || strings.Contains(string(b), "credential-canary") {
				t.Fatal("rejected/secret payload reached saved checkpoint")
			}
			if route == healthRoute && len(run.Findings) != 1 {
				t.Fatal("valid issue evidence lost")
			}
		})
	}
}

func TestMalformedPageEnvelopesFailWithoutRetry(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute, serviceHealthRoute, monitorRoute, advisorRoute} {
		for _, body := range []string{`{invalid`, `null`, `[]`, `{}`, `{"value":null}`, `{"value":{}}`, `{"value":[],"nextLink":{"bad":"envelope-canary"}}`, `{"value":[]}{"value":[]}`} {
			t.Run(route+"/"+body, func(t *testing.T) {
				calls := 0
				c := collection(t, func(*http.Request) (*http.Response, error) { calls++; return response(200, body), nil })
				collect := collectorFor(c, route)
				err := collect(context.Background(), selectedID, func(health.Page) error { t.Fatal("malformed envelope was emitted"); return nil })
				var diagnostic interface{ Diagnostic() string }
				if err == nil || !errors.As(err, &diagnostic) || diagnostic.Diagnostic() != "invalid_response" || calls != 1 || strings.Contains(err.Error(), "envelope-canary") {
					t.Fatalf("invalid page outcome: %v calls %d", err, calls)
				}
			})
		}
	}
}

func TestAmbiguousCollectionFieldsRejectPageBeforeCheckpoint(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute, serviceHealthRoute, monitorRoute, advisorRoute} {
		t.Run(route, func(t *testing.T) {
			c := collection(t, func(*http.Request) (*http.Response, error) {
				return response(200, `{"value":[],"nextLink":null}`), nil
			})
			pages := 0
			if err := collectorFor(c, route)(context.Background(), selectedID, func(health.Page) error {
				pages++
				return nil
			}); err != nil || pages != 1 {
				t.Fatalf("valid empty terminal page: pages %d, error %v", pages, err)
			}
			for _, body := range []string{
				`{"value":[],"value":[]}`,
				`{"value":[],"nextLink":null,"nextLink":""}`,
				`{"value":[],"Value":[]}`,
				`{"value":[],"NextLink":"continuation"}`,
				`{"value":[],"NextLink":"continuation","nextLink":null}`,
			} {
				t.Run(body, func(t *testing.T) {
					calls := 0
					c := collection(t, func(*http.Request) (*http.Response, error) {
						calls++
						if calls == 1 {
							return response(200, fmt.Sprintf(`{"value":[],"nextLink":%q}`, listURL(route, "next"))), nil
						}
						return response(200, body), nil
					})
					run := health.NewRun("test", "test")
					run.SchemaVersion = health.AssessmentSchemaVersion
					run.Scope = &health.Scope{SubscriptionID: selectedID}
					run.StartedAt = c.ServiceHealthQuery.ReferenceAt
					collectors := []health.NamedCollector{{Name: route, Collect: collectorFor(c, route)}}
					var savedPages []int
					_, err := health.Check(context.Background(), &run, collectors, func(r *health.Run) error {
						savedPages = append(savedPages, r.Sources[route].Pages)
						return nil
					})
					s := run.Sources[route]
					if err != nil || calls != 2 || s.Status != "partial" || s.Reason != "invalid_response" || s.Pages != 1 || s.EnumerationComplete ||
						len(savedPages) != 3 || savedPages[0] != 0 || savedPages[1] != 1 || savedPages[2] != 1 {
						t.Fatalf("ambiguous envelope reached checkpoint: calls %d error %v source %+v checkpoints %v", calls, err, s, savedPages)
					}
				})
			}
		})
	}
}

func TestMonitorDuplicateDecisionFieldsPreserveValidSibling(t *testing.T) {
	valid := fmt.Sprintf(`{"id":%q,"properties":{"essentials":{"signalType":"Metric","monitorCondition":"Fired","targetResource":%q}}}`,
		"/subscriptions/"+selectedID+"/providers/Microsoft.AlertsManagement/alerts/valid", collectionVM)
	for _, tc := range []struct {
		name, fixed, first, second string
	}{
		{"condition", fmt.Sprintf(`"targetResource":%q`, collectionVM), `"monitorCondition":"Fired"`, `"monitorCondition":"Resolved"`},
		{"target", `"monitorCondition":"Fired"`, fmt.Sprintf(`"targetResource":%q`, collectionVM), fmt.Sprintf(`"targetResource":%q`, collectionVM+"-ambiguous")},
		{"case alias", fmt.Sprintf(`"targetResource":%q`, collectionVM), `"monitorCondition":"Fired"`, `"MonitorCondition":"Resolved"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, members := range [][2]string{{tc.first, tc.second}, {tc.second, tc.first}} {
				ambiguous := fmt.Sprintf(`{"id":%q,"properties":{"essentials":{"signalType":"Metric",%s,%s,%s}}}`,
					"/subscriptions/"+selectedID+"/providers/Microsoft.AlertsManagement/alerts/ambiguous", tc.fixed, members[0], members[1])
				c := collection(t, func(*http.Request) (*http.Response, error) {
					return response(200, fmt.Sprintf(`{"value":[%s,%s]}`, valid, ambiguous)), nil
				})
				run := health.NewRun("test", "test")
				run.SchemaVersion = health.AssessmentSchemaVersion
				run.Scope = &health.Scope{SubscriptionID: selectedID}
				run.StartedAt = c.ServiceHealthQuery.ReferenceAt
				collectors := []health.NamedCollector{
					{Name: "inventory", Collect: func(_ context.Context, _ string, emit func(health.Page) error) error {
						return emit(health.Page{Resources: []health.ResourceInput{{ID: collectionVM}}})
					}},
					{Name: monitorRoute, Collect: c.AzureMonitor},
				}
				code, err := health.Check(context.Background(), &run, collectors, func(*health.Run) error { return nil })
				s := run.Sources[monitorRoute]
				if err != nil || code != 3 || s.Status != "partial" || s.Reason != "invalid_records" || s.Pages != 1 ||
					s.RejectedRecords != 1 || run.Coverage != "incomplete" || len(run.Findings) != 1 || !strings.HasSuffix(run.Findings[0].EntityID, "/valid") ||
					len(run.ResourceAssessments) != 1 || len(run.ResourceAssessments[0].FindingIDs) != 1 {
					t.Fatalf("unexpected result for %v: code %d, error %v, source %+v, findings %+v", members, code, err, s, run.Findings)
				}
			}
		})
	}
}

func TestDuplicateDecisionFieldsAreRejectedPerRecord(t *testing.T) {
	for _, tc := range []struct{ route, record string }{
		{inventoryRoute, `{"id":"one","id":"two"}`},
		{healthRoute, `{"properties":{"availabilityState":"Available","availabilityState":"Degraded"}}`},
		{serviceHealthRoute, `{"properties":{"status":"Active","status":"Resolved"}}`},
	} {
		t.Run(tc.route, func(t *testing.T) {
			page, _, err := filterRecords([]byte(`{"value":[`+tc.record+`,{}]}`), tc.route)
			if err != nil || string(page) != `{"value":[null,{}]}` {
				t.Fatalf("ambiguous row should be rejected without dropping its sibling: %s, %v", page, err)
			}
		})
	}
}

func TestSchema4RejectedRowsKeepVerifiedCoverageScope(t *testing.T) {
	other, independent := collectionVM+"-other", collectionVM+"-independent"
	statusSuffix := "/providers/Microsoft.ResourceHealth/availabilityStatuses/current"
	alertID := "/subscriptions/" + selectedID + "/providers/Microsoft.AlertsManagement/alerts/test"
	advisorID := collectionVM + advisorListPath + "/test"
	canary := "rejected-payload-canary"
	alert := func(target, conditions string) string {
		return fmt.Sprintf(`{"id":%q,"properties":{"essentials":{"signalType":"Metric",%s,%s}}}`, alertID, target, conditions)
	}
	target := func(id string) string { return fmt.Sprintf(`"targetResource":%q`, id) }
	conflicted := `"monitorCondition":"Fired","monitorCondition":"` + canary + `"`
	fired := alert(target(collectionVM), `"monitorCondition":"Fired"`)
	const utcOverflow = "9999-12-31T23:59:59-01:00"
	const utcUnderflow = "0000-01-01T00:00:00+01:00"
	healthyRows := fmt.Sprintf(`{"id":%q,"properties":{"availabilityState":"Available"}},{"id":%q,"properties":{"availabilityState":"Available"}},{"id":%q,"properties":{"availabilityState":"Available"}}`,
		collectionVM+statusSuffix, other+statusSuffix, independent+statusSuffix)
	available := "Available"
	inventory := health.Page{Resources: []health.ResourceInput{{ID: collectionVM}, {ID: other}, {ID: independent}}}
	resourceHealth := health.Page{Health: []health.HealthInput{
		{StatusID: collectionVM + statusSuffix, PropertiesPresent: true, State: &available},
		{StatusID: other + statusSuffix, PropertiesPresent: true, State: &available},
		{StatusID: independent + statusSuffix, PropertiesPresent: true, State: &available},
	}}
	emitSynthetic := func(page health.Page) health.Collector {
		return func(_ context.Context, _ string, emit func(health.Page) error) error { return emit(page) }
	}
	for _, tc := range []struct {
		name, route, records, second string
		scoped, findings             int
		unknown                      int
	}{
		{"inventory", inventoryRoute, fmt.Sprintf(`{"id":%q},{"id":%q},{"id":%q},{"id":%q,"name":"first","name":%q}`, collectionVM, other, independent, collectionVM, canary), "", 1, 0, 1},
		{"resource health", healthRoute, healthyRows + fmt.Sprintf(`,{"id":%q,"properties":{"availabilityState":"Degraded","availabilityState":%q}}`,
			collectionVM+statusSuffix, canary), "", 1, 0, 1},
		{"resource health UTC overflow", healthRoute, healthyRows + fmt.Sprintf(`,{"id":%q,"properties":{"availabilityState":"Degraded","reportedTime":%q,"healthEventId":%q}}`,
			collectionVM+statusSuffix, utcOverflow, canary), "", 1, 0, 1},
		{"resource health UTC underflow", healthRoute, healthyRows + fmt.Sprintf(`,{"id":%q,"properties":{"availabilityState":"Degraded","occuredTime":%q}}`,
			collectionVM+statusSuffix, utcUnderflow), "", 1, 0, 1},
		{"service health", serviceHealthRoute, fmt.Sprintf(`{"id":"/providers/Microsoft.ResourceHealth/events/test","properties":{"status":"Active","status":%q}}`, canary), "", 0, 0, 3},
		{"service health UTC overflow", serviceHealthRoute, fmt.Sprintf(`{"id":"/providers/Microsoft.ResourceHealth/events/test","properties":{"eventType":"ServiceIssue","status":"Resolved","lastUpdateTime":%q}}`, utcOverflow), "", 0, 0, 3},
		{"service health region UTC underflow", serviceHealthRoute, fmt.Sprintf(`{"id":"/providers/Microsoft.ResourceHealth/events/test","properties":{"eventType":"ServiceIssue","status":"Resolved","impact":[{"impactedService":"Compute","impactedRegions":[{"impactedRegion":%q,"lastUpdateTime":%q}]}]}}`, canary, utcUnderflow), "", 0, 0, 3},
		{"advisor", advisorRoute, fmt.Sprintf(`{"id":%q,"properties":{"recommendationStatus":"New","recommendationStatus":%q,"resourceMetadata":{"resourceId":%q}}}`, advisorID, canary, collectionVM), "", 1, 0, 1},
		{"advisor UTC overflow", advisorRoute, fmt.Sprintf(`{"id":%q,"properties":{"recommendationStatus":"New","lastUpdated":%q,"resourceMetadata":{"resourceId":%q}}}`, advisorID, utcOverflow, collectionVM), "", 1, 0, 1},
		{"terminal monitor", monitorRoute, fired + "," + alert(target(collectionVM), conflicted), "", 1, 1, 1},
		{"monitor UTC overflow", monitorRoute, fired + "," + alert(target(collectionVM), fmt.Sprintf(`"monitorCondition":"Resolved","lastModifiedTimeByMonitorService":%q`, utcOverflow)), "", 1, 1, 1},
		{"monitor target alias", monitorRoute, alert(target(collectionVM)+fmt.Sprintf(`,"TargetResource":%q`, other), conflicted), "", 0, 0, 3},
		{"monitor target delimiter alias", monitorRoute, alert(target(collectionVM)+fmt.Sprintf(`,"target_resource":%q`, other), conflicted), "", 0, 0, 3},
		{"monitor foreign target", monitorRoute, alert(target(strings.Replace(collectionVM, selectedID, otherID, 1)+"-"+canary), conflicted), "", 0, 0, 3},
		{"monitor workspace target", monitorRoute, alert(target("/subscriptions/"+selectedID+"/resourceGroups/g/providers/Microsoft.OperationalInsights/workspaces/logs"), conflicted), "", 0, 0, 3},
		{"monitor case-equivalent workspace target", monitorRoute, alert(target("/subscriptions/"+selectedID+"/resourceGroups/g/providers/Microsoft.OperationalInsights/workſpaces/logs"), conflicted), "", 0, 0, 3},
		{"monitor mixed scoped and unscoped rows", monitorRoute, alert(target(collectionVM), conflicted) + ",null", "", 1, 0, 3},
		{"monitor denied continuation", monitorRoute, alert(target(collectionVM), conflicted), "denied", 1, 0, 3},
		{"monitor entity moved after rejected row", monitorRoute, alert(target(collectionVM), conflicted), alert(target(other), `"monitorCondition":"Resolved"`), 1, 0, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var saved health.Run
			calls := 0
			c := collection(t, func(*http.Request) (*http.Response, error) {
				calls++
				if calls == 2 {
					source := saved.Sources[tc.route]
					if source.Pages != 1 || source.EnumerationComplete || len(source.RejectedScopes) != tc.scoped {
						t.Fatalf("first page scope was not retained before continuation: %+v", source)
					}
					if tc.second == "denied" {
						return response(403, `{}`), nil
					}
					return response(200, `{"value":[`+tc.second+`]}`), nil
				}
				if tc.second != "" {
					return response(200, fmt.Sprintf(`{"value":[%s],"nextLink":%q}`, tc.records, listURL(tc.route, "next"))), nil
				}
				return response(200, `{"value":[`+tc.records+`]}`), nil
			})
			run := health.NewRun("test", "test")
			run.SchemaVersion = health.AssessmentSchemaVersion
			run.Scope = &health.Scope{SubscriptionID: selectedID}
			run.StartedAt = c.ServiceHealthQuery.ReferenceAt
			collectors := []health.NamedCollector{
				{Name: inventoryRoute, Collect: emitSynthetic(inventory)},
				{Name: healthRoute, Collect: emitSynthetic(resourceHealth)},
				{Name: serviceHealthRoute, Collect: emitSynthetic(health.Page{})},
				{Name: monitorRoute, Collect: emitSynthetic(health.Page{})},
				{Name: advisorRoute, Collect: emitSynthetic(health.Page{})},
			}
			for i := range collectors {
				if collectors[i].Name == tc.route {
					collectors[i].Collect = collectorFor(c, tc.route)
				}
			}
			code, err := health.Check(context.Background(), &run, collectors, func(r *health.Run) error {
				checkpoint, err := json.Marshal(r)
				if err != nil {
					return err
				}
				if strings.Contains(string(checkpoint), canary) || r.AssessmentSummary.OK != 0 {
					t.Fatalf("checkpoint leaked rejected data or asserted OK: %+v", r.AssessmentSummary)
				}
				saved = health.Run{}
				return json.Unmarshal(checkpoint, &saved)
			})
			if code != 3 || err != nil {
				t.Fatalf("expected a partial run, got %d (%v)", code, err)
			}
			now := time.Now().UTC()
			saved.Outcome, saved.EndedAt = run.Outcome, &now
			saved.RebuildCoverage()
			s := saved.Sources[tc.route]
			wantReason, wantPages, wantRejected := "invalid_records", 1, 1
			if tc.second == "denied" {
				wantReason = "access_denied"
			} else if tc.second != "" {
				wantPages = 2
			}
			if tc.name == "monitor mixed scoped and unscoped rows" {
				wantRejected = 2
			}
			if s.Status != "partial" || s.Reason != wantReason || s.Pages != wantPages || s.RejectedRecords != wantRejected ||
				len(s.RejectedScopes) != tc.scoped || s.EnumerationComplete != (tc.second != "denied") {
				t.Fatalf("unexpected source proof: %+v", s)
			}
			wantCounts := health.AssessmentSummary{OK: 3 - tc.unknown, Unknown: tc.unknown}
			if saved.Coverage != "incomplete" || *saved.AssessmentSummary != wantCounts || len(saved.Findings) != tc.findings {
				t.Fatalf("counts %+v, coverage %s, findings %+v, gaps %+v", saved.AssessmentSummary, saved.Coverage, saved.Findings, saved.Gaps)
			}
			for _, result := range saved.ResourceAssessments {
				rejectedTarget := strings.EqualFold(result.ResourceID, collectionVM)
				want := "ok"
				if rejectedTarget || tc.unknown == 3 || tc.unknown == 2 && strings.EqualFold(result.ResourceID, other) {
					want = "unknown"
				}
				if result.Disposition != want || rejectedTarget && len(result.FindingIDs) != tc.findings {
					t.Fatalf("classification for %s: %+v", result.ResourceID, result)
				}
			}
			assessments := saved.ResourceAssessments
			saved.RebuildCoverage()
			if !reflect.DeepEqual(assessments, saved.ResourceAssessments) {
				t.Fatal("rebuilding the saved source proof changed its assessments")
			}
		})
	}
}

func TestMixedSchemaCheckpointSurvivesLaterRetriesAndUnsafeContinuation(t *testing.T) {
	for _, route := range []string{inventoryRoute, healthRoute} {
		for _, scenario := range []string{"throttled", "unsafe continuation"} {
			t.Run(route+"/"+scenario, func(t *testing.T) {
				calls := 0
				c := collection(t, func(*http.Request) (*http.Response, error) {
					calls++
					if calls > 1 {
						return response(429, `{"error":{"message":"response-canary"}}`), nil
					}
					good := fmt.Sprintf(`{"id":%q}`, collectionVM)
					bad := `{"id":42}`
					if route == healthRoute {
						good = fmt.Sprintf(`{"id":%q,"properties":{"availabilityState":"Unavailable"}}`, collectionVM+"/providers/Microsoft.ResourceHealth/availabilityStatuses/current")
						bad = `{"properties":{"availabilityState":42}}`
					}
					next := listURL(route, "next")
					if scenario == "unsafe continuation" {
						next = strings.Replace(next, selectedID, otherID, 1)
					}
					return response(200, fmt.Sprintf(`{"value":[%s,%s],"nextLink":%q}`, good, bad, next)), nil
				})
				run := health.NewRun("test", "test")
				run.Scope = &health.Scope{SubscriptionID: selectedID}
				store, err := runstore.New(t.TempDir(), &run)
				if err != nil {
					t.Fatal(err)
				}
				defer store.Close()
				collect := c.Inventory
				if route == healthRoute {
					collect = c.ResourceHealth
				}
				complete := func(context.Context, string, func(health.Page) error) error { return nil }
				inventory, resourceHealth := complete, complete
				if route == inventoryRoute {
					inventory = collect
				} else {
					resourceHealth = collect
				}
				code, err := health.Check(context.Background(), &run, []health.NamedCollector{{Name: "inventory", Collect: inventory}, {Name: "resource_health", Collect: resourceHealth}}, store.Save)
				expectedCalls, reason := 5, "throttled"
				if scenario == "unsafe continuation" {
					expectedCalls, reason = 1, "unsafe_continuation"
				}
				if err != nil || code != 3 || calls != expectedCalls || run.Sources[route].Reason != reason || run.Sources[route].RejectedRecords != 1 {
					t.Fatalf("outcome %d %v calls %d source %+v", code, err, calls, run.Sources[route])
				}
				b, e := os.ReadFile(store.Path())
				if e != nil {
					t.Fatal(e)
				}
				var saved health.Run
				if json.Unmarshal(b, &saved) != nil {
					t.Fatal("invalid snapshot")
				}
				if len(saved.Evidence) != 1 || saved.Sources[route].Pages != 1 || saved.Coverage != "incomplete" || strings.Contains(string(b), "response-canary") {
					t.Fatal("valid checkpoint lost")
				}
				foundRejected := false
				for _, gap := range saved.Gaps {
					if gap.Source == route && gap.Reason == "invalid_records" {
						foundRejected = true
					}
				}
				if !foundRejected {
					t.Fatal("later failure concealed rejected-record gap")
				}
			})
		}
	}
}

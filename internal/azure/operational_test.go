package azure

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

func operationalFixture(t *testing.T, route string) string {
	t.Helper()
	file := "testdata/service-health-events.json"
	if route == monitorRoute {
		file = "testdata/monitor-alerts.json"
	}
	b, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestOperationalLiteralFixturesAndRequests(t *testing.T) {
	for _, route := range []string{serviceHealthRoute, monitorRoute} {
		t.Run(route, func(t *testing.T) {
			// Literal expectations intentionally do not use listURL or adapter helpers.
			initial := "https://management.azure.com/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.ResourceHealth/events?api-version=2025-05-01&queryStartTime=10%2F16%2F2026"
			next := "https://management.azure.com/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.ResourceHealth/events?api-version=2025-05-01&%24skipToken=service-token-canary"
			if route == monitorRoute {
				initial = "https://management.azure.com/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.AlertsManagement/alerts?api-version=2019-03-01&includeContext=false&includeEgressConfig=false&monitorCondition=Fired&pageCount=100&timeRange=30d"
				next = "https://management.azure.com:443/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.AlertsManagement/alerts?api-version=2019-03-01&ctoken=monitor-token-canary%2B%2F%3D%23opaque&timeRange=30d"
			}
			run := health.NewRun("test", "test")
			run.Scope = &health.Scope{SubscriptionID: selectedID}
			run.StartedAt = time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
			query := health.ServiceHealthQuery(run.StartedAt)
			if route == monitorRoute {
				query = health.MonitorQuery(run.StartedAt)
			}
			source := run.Sources[route]
			source.Query = &query
			run.Sources[route] = source
			store, err := runstore.New(t.TempDir(), &run)
			if err != nil {
				t.Fatal(err)
			}
			defer store.Close()
			var debug bytes.Buffer
			ConfigureDebugLogging(true, &debug)
			defer ConfigureDebugLogging(false, nil)
			calls := 0
			c := collection(t, func(req *http.Request) (*http.Response, error) {
				calls++
				want := initial
				if calls == 2 {
					want = next
				}
				wantURL, _ := url.Parse(want)
				if calls > 2 || req.Method != "GET" || req.URL.Scheme != wantURL.Scheme || req.URL.Host != wantURL.Host || req.URL.Path != wantURL.Path || req.URL.Query().Encode() != wantURL.Query().Encode() || req.Header.Get("Authorization") != "Bearer synthetic-test-token" {
					t.Fatalf("unexpected authenticated request: method %s url %s", req.Method, req.URL)
				}
				deadline, ok := req.Context().Deadline()
				if !ok || time.Until(deadline) <= 0 || time.Until(deadline) > OperationTimeout {
					t.Fatal("missing page deadline")
				}
				if calls == 2 {
					b, err := os.ReadFile(store.Path())
					if err != nil {
						t.Fatal(err)
					}
					var saved health.Run
					if json.Unmarshal(b, &saved) != nil || len(saved.Evidence) != 2 || saved.Sources[route].Pages != 1 {
						t.Fatal("page not committed before continuation")
					}
					return response(200, `{"value":[]}`), nil
				}
				return response(200, operationalFixture(t, route)), nil
			})
			code, err := health.Check(context.Background(), &run, []health.NamedCollector{{Name: route, Collect: collectorFor(c, route)}}, store.Save)
			if err != nil || code != 0 || calls != 2 || run.Sources[route].Status != "completed" || run.Sources[route].Pages != 2 || run.Sources[route].RejectedRecords != 0 || len(run.Evidence) != 2 || len(run.Findings) != 2 {
				t.Fatalf("collection: %d %v calls %d source %+v evidence %d findings %d", code, err, calls, run.Sources[route], len(run.Evidence), len(run.Findings))
			}
			e := run.Evidence[0]
			if e.Operation == "" || e.EntityID == "" || e.SubscriptionID != selectedID {
				t.Fatal("lost provenance")
			}
			if route == serviceHealthRoute {
				var incident, sensitive *health.Evidence
				for i := range run.Evidence {
					e := &run.Evidence[i]
					if e.ServiceHealth.Category == "ServiceIssue" {
						incident = e
					} else {
						sensitive = e
					}
				}
				if incident == nil || sensitive == nil || incident.APIVersion != "2025-05-01" || incident.ServiceHealth.Sensitive != nil || incident.ServiceHealth.Level != "Warning" || incident.ServiceHealth.EventSource != "ResourceHealth" || incident.ReportedAt == nil || incident.ReportedAt.Nanosecond() != 123456700 || incident.ServiceHealth.ImpactMitigationTime == nil || incident.ResourceID != "" {
					t.Fatal("lost event facts")
				}
				if sensitive.ServiceHealth.Sensitive == nil || !*sensitive.ServiceHealth.Sensitive || len(sensitive.ServiceHealth.Impact) != 0 {
					t.Fatal("metadata-only sensitive event rejected")
				}
				regions := incident.ServiceHealth.Impact[0].Regions
				if len(regions) != 1 || regions[0].Name != "West US" || regions[0].Status != "Active" || regions[0].LastUpdateTime == nil || regions[0].Attribution != "selected_subscription" {
					t.Fatalf("region mapping %+v", regions)
				}
			} else {
				workflows := map[string]bool{}
				for _, e := range run.Evidence {
					if e.APIVersion != "2019-03-01" || e.Monitor.Condition != "Fired" || e.Monitor.Severity == "" || e.ReportedAt == nil || e.OccurredAt == nil || e.ResourceID == "" || e.Monitor.IsSuppressed == nil {
						t.Fatal("lost nested essentials")
					}
					workflows[e.Monitor.WorkflowState] = true
					if e.Monitor.WorkflowState == "Closed" && !*e.Monitor.IsSuppressed {
						t.Fatal("suppression lost")
					}
				}
				if !workflows["Acknowledged"] || !workflows["Closed"] {
					t.Fatal("workflow treated as condition")
				}
			}
			b, err := os.ReadFile(store.Path())
			if err != nil {
				t.Fatal(err)
			}
			for _, excluded := range []string{"canary", otherID, "nextLink", "ctoken", "$skipToken", "synthetic-test-token"} {
				if strings.Contains(string(b), excluded) || strings.Contains(debug.String(), excluded) {
					t.Fatalf("private wire field leaked: %s", excluded)
				}
			}
		})
	}
}

func TestOperationalMixedRecordsAndRetainedSnapshots(t *testing.T) {
	for _, route := range []string{serviceHealthRoute, monitorRoute} {
		for _, later := range []string{"complete", "throttled", "unsafe", "cancelled"} {
			t.Run(route+"/"+later, func(t *testing.T) {
				var envelope map[string]json.RawMessage
				if json.Unmarshal([]byte(operationalFixture(t, route)), &envelope) != nil {
					t.Fatal("bad fixture")
				}
				var records []json.RawMessage
				if json.Unmarshal(envelope["value"], &records) != nil {
					t.Fatal("bad records")
				}
				// Each malformed sibling must be isolated independently, including invalid
				// nested structures and timestamps. Unknown enum strings are valid schema.
				bad := []string{`null`, `42`, `{"id":42}`, `{"properties":[]}`, `{"id":"/providers/Microsoft.ResourceHealth/events/bad","properties":{"lastUpdateTime":"time-canary"}}`, `{"properties":{"impact":[{"impactedRegions":[{"status":42}]}]}}`}
				unknown := `{"id":"/providers/Microsoft.ResourceHealth/events/unknown","properties":{"eventType":"FutureType","status":"Active","lastUpdateTime":"2026-11-14T12:00:00Z"}}`
				if route == monitorRoute {
					bad = []string{`null`, `42`, `{"id":42}`, `{"properties":[]}`, `{"properties":{"essentials":{"lastModifiedDateTime":"time-canary"}}}`, `{"properties":{"essentials":{"actionStatus":{"isSuppressed":"bool-canary"}}}}`}
					unknown = `{"id":"/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.AlertsManagement/alerts/unknown","properties":{"essentials":{"signalType":"FutureType","monitorCondition":"Fired"}}}`
				}
				for _, b := range bad {
					records = append(records, json.RawMessage(b))
				}
				records = append(records, json.RawMessage(unknown))
				envelope["value"], _ = json.Marshal(records)
				next := listURL(route, "token-canary")
				if later == "unsafe" {
					next = strings.Replace(next, selectedID, otherID, 1)
				}
				envelope["nextLink"], _ = json.Marshal(next)
				body, _ := json.Marshal(envelope)
				run := health.NewRun("test", "test")
				run.Scope = &health.Scope{SubscriptionID: selectedID}
				query := health.ServiceHealthQuery(time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC))
				if route == monitorRoute {
					query = health.MonitorQuery(query.ReferenceAt)
				}
				s := run.Sources[route]
				s.Query = &query
				run.Sources[route] = s
				store, err := runstore.New(t.TempDir(), &run)
				if err != nil {
					t.Fatal(err)
				}
				defer store.Close()
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				calls := 0
				c := collection(t, func(*http.Request) (*http.Response, error) {
					calls++
					if calls == 1 {
						return response(200, string(body)), nil
					}
					b, err := os.ReadFile(store.Path())
					if err != nil {
						t.Fatal(err)
					}
					var saved health.Run
					if json.Unmarshal(b, &saved) != nil || len(saved.Evidence) != 3 || saved.Sources[route].RejectedRecords != len(bad) || saved.Sources[route].UnclassifiedRecords != 1 || saved.Sources[route].Pages != 1 {
						t.Fatal("mixed page not committed")
					}
					if later == "throttled" {
						return response(429, `{"error":{"message":"error-canary"}}`), nil
					}
					if later != "complete" {
						t.Fatal("unsafe/cancelled read reached transport")
					}
					return response(200, `{"value":[]}`), nil
				})
				save := func(r *health.Run) error {
					err := store.Save(r)
					if err == nil && later == "cancelled" && r.Sources[route].Pages == 1 {
						cancel()
					}
					return err
				}
				code, err := health.Check(ctx, &run, []health.NamedCollector{{Name: route, Collect: collectorFor(c, route)}}, save)
				wantCalls, wantPages, reason := 2, 2, "invalid_records"
				if later == "throttled" {
					wantCalls, wantPages, reason = 5, 1, "throttled"
				}
				if later == "unsafe" {
					wantCalls, wantPages, reason = 1, 1, "unsafe_continuation"
				}
				if later == "cancelled" {
					wantCalls, wantPages = 1, 1
				}
				wantCode := 3
				if later == "cancelled" {
					wantCode = 130
				}
				if err != nil || code != wantCode || calls != wantCalls || run.Sources[route].Pages != wantPages || run.Sources[route].RejectedRecords != len(bad) || len(run.Evidence) != 3 || len(run.Findings) != 2 {
					t.Fatalf("code %d err %v calls %d source %+v evidence %d findings %d", code, err, calls, run.Sources[route], len(run.Evidence), len(run.Findings))
				}
				if later != "cancelled" && run.Sources[route].Reason != reason {
					t.Fatalf("reason %+v", run.Sources[route])
				}
				b, err := os.ReadFile(store.Path())
				if err != nil {
					t.Fatal(err)
				}
				if strings.Contains(string(b), "canary") || strings.Contains(string(b), otherID) {
					t.Fatal("raw payload reached saved JSON")
				}
			})
		}
	}
}

func TestServiceHealthRequiresFixedPolicy(t *testing.T) {
	good := health.ServiceHealthQuery(time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC))
	for _, q := range []health.SourceQuery{{}, {ReferenceAt: good.ReferenceAt}, {ReferenceAt: good.ReferenceAt, Cutoff: good.Cutoff, QueryStartTime: "11/15/2026"}, {ReferenceAt: good.ReferenceAt, Cutoff: &good.ReferenceAt, QueryStartTime: good.QueryStartTime}} {
		c := Collection{ServiceHealthQuery: q, Credential: func(string) (azcore.TokenCredential, error) {
			t.Fatal("invalid policy requested credentials")
			return nil, nil
		}}
		err := c.ServiceHealth(context.Background(), selectedID, func(health.Page) error { t.Fatal("invalid policy emitted page"); return nil })
		var failure *Failure
		if !errors.As(err, &failure) || failure.Category != "invalid_query_policy" {
			t.Fatalf("invalid policy accepted: %v", err)
		}
	}
}

func TestOperationalSourceIdentityAndProjectionVariants(t *testing.T) {
	for _, route := range []string{serviceHealthRoute, monitorRoute} {
		t.Run(route, func(t *testing.T) {
			body := `{"value":[{"id":"/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.ResourceHealth/events/scoped","properties":{"eventType":"PlannedMaintenance","status":"Active","lastUpdateTime":"2026-11-14T12:00:00Z","impact":[null,{"impactedService":"SQL","impactedRegions":[null,{"impactedRegion":"West US","status":"Resolved"}]}]}},{"id":"/providers/Microsoft.ResourceHealth/events/advisory","properties":{"eventType":"HealthAdvisory","status":"Resolved"}},{"id":"/providers/Microsoft.ResourceHealth/events/missing"},{"id":"/providers/Microsoft.ResourceHealth/events/null","properties":null}]}`
			if route == monitorRoute {
				body = `{"value":[{"id":"/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.AlertsManagement/alerts/workspace","properties":{"essentials":{"signalType":"Log","monitorCondition":"Fired","targetResource":"/subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft.OperationalInsights/workspaces/logs","lastModifiedTimeByMonitorService":"2026-11-14T12:00:00Z"}}},{"id":"/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.AlertsManagement/alerts/foreign-target","properties":{"essentials":{"signalType":"Metric","monitorCondition":"Resolved","targetResource":"/subscriptions/22222222-2222-2222-2222-222222222222/resourceGroups/g/providers/Microsoft.Compute/virtualMachines/foreign"}}},{"id":"/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.AlertsManagement/alerts/missing","properties":{}},{"id":"/subscriptions/22222222-2222-2222-2222-222222222222/providers/Microsoft.AlertsManagement/alerts/foreign","properties":{"essentials":{"signalType":"Metric","monitorCondition":"Fired"}}}]}`
			}
			c := collection(t, func(*http.Request) (*http.Response, error) { return response(200, body), nil })
			run := health.NewRun("test", "test")
			run.Scope = &health.Scope{SubscriptionID: selectedID}
			q := c.ServiceHealthQuery
			s := run.Sources[route]
			s.Query = &q
			run.Sources[route] = s
			err := collectorFor(c, route)(context.Background(), selectedID, func(p health.Page) error { run.NormalizePage(route, p); return nil })
			if err != nil || len(run.Evidence) != 2 || run.Sources[route].RejectedRecords != 2 {
				t.Fatalf("projection %v source %+v evidence %d", err, run.Sources[route], len(run.Evidence))
			}
			b, _ := json.Marshal(run)
			if strings.Contains(string(b), otherID) {
				t.Fatal("foreign scope persisted")
			}
			if route == serviceHealthRoute {
				if len(run.Findings) != 1 {
					t.Fatal("planned notice lost")
				}
			} else {
				if len(run.Findings) != 1 {
					t.Fatal("workspace alert lost")
				}
				found := false
				for _, e := range run.Evidence {
					if e.Monitor.LastModifiedTimeByMonitorService != nil {
						found = true
					}
				}
				if !found {
					t.Fatal("optional source timestamp lost")
				}
			}
		})
	}
}

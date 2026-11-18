package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

var implementedSources = []string{"inventory", "resource_health", "service_health", "azure_monitor"}

// Exercise production defaults, including discovery and credential subprocesses.
// integrationApp installs an isolated fake az and restores the global transport.
func fourSourceApp(t *testing.T, wire armFixture) (App, *bytes.Buffer, *bytes.Buffer, string) {
	t.Helper()
	a, out, diag, log := integrationApp(t, func(*http.Request) (*http.Response, error) {
		t.Fatal("unexpected injected collector")
		return nil, nil
	})
	a.Validate, a.Inventory, a.ResourceHealth, a.ServiceHealth, a.AzureMonitor = nil, nil, nil, nil, nil
	http.DefaultTransport = armFixture(func(req *http.Request) (*http.Response, error) {
		validHost := req.URL.Host == "management.azure.com" || req.URL.Host == "management.azure.com:443"
		validScope := req.URL.Path == "/subscriptions/"+first || strings.HasPrefix(req.URL.Path, "/subscriptions/"+first+"/")
		if req.Method != "GET" || !validHost || !validScope || req.Header.Get("Authorization") != "Bearer synthetic-token-canary" {
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
	if source == "service_health" {
		category := []string{"ServiceIssue", "PlannedMaintenance", "HealthAdvisory", "SecurityAdvisory"}[i%4]
		status := "Resolved"
		if i < 4 {
			status = "Active"
		}
		return fmt.Sprintf(`{
			"id":"/providers/Microsoft.ResourceHealth/events/event-%03d",
			"properties":{
				"eventType":%q,"status":%q,"lastUpdateTime":%q,"impactStartTime":%q,
				"isEventSensitive":true,"description":"description-canary",
				"impact":[
					{"impactedService":"Compute | <script>[link](https://bad.example)","impactedRegions":[
						{"impactedRegion":"East US","status":"Active","impactedSubscriptions":[%q,%q],"impactedTenants":["tenant-canary"]},
						{"impactedRegion":"foreign-region-canary","impactedSubscriptions":[%q]}
					]},
					{"impactedService":"foreign-service-canary","impactedRegions":[
						{"impactedRegion":"foreign-only-region-canary","impactedSubscriptions":[%q]}
					]}
				]
			}
		}`, i, category, status, stamp, now.Add(time.Hour).Format(time.RFC3339Nano), first, second, second, second)
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
	return fmt.Sprintf(`{"id":"/subscriptions/%s/providers/Microsoft.AlertsManagement/alerts/alert-%03d","properties":{"essentials":{"signalType":"Log","monitorCondition":%q,"alertState":%q,"severity":"Sev2","targetResource":%q,"lastModifiedDateTime":%q,"actionStatus":{"isSuppressed":true},"lastModifiedUserName":"username-canary","description":"description-canary"},"context":{"SearchQuery":"query-canary","SearchResults":"results-canary"}}}`, first, i, condition, []string{"New", "Acknowledged", "Closed"}[i%3], target, stamp)
}

func TestFourSourceNativeScaleAndPaging(t *testing.T) {
	for _, pages := range []int{3, 24} {
		t.Run(fmt.Sprintf("pages-%d", pages), func(t *testing.T) {
			counts := map[string]int{}
			sequence := []string{}
			var a App
			a, out, diag, log := fourSourceApp(t, func(req *http.Request) (*http.Response, error) {
				source := requestSource(req)
				counts[source]++
				if source == "subscription" {
					return armResponse(200, subscriptionResponse()), nil
				}
				page := counts[source] - 1
				if page >= pages {
					t.Fatal("unbounded paging")
				}
				if page == 0 {
					sequence = append(sequence, source)
				}
				saved := snapshot(t, a.Directory)
				if saved.Sources[source].Status != "in_progress" || saved.Sources[source].Pages != page {
					t.Fatal("native request preceded committed source/page state")
				}
				if source == "service_health" || source == "azure_monitor" {
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
				}
				if page > 0 {
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
			r := assertIntegrationArtifacts(t, a, out, diag)
			if !reflect.DeepEqual(sequence, implementedSources) {
				t.Fatalf("wrong source order %v", sequence)
			}
			wantResources, wantEvidence := 360, 732
			if pages == 24 {
				wantResources, wantEvidence = 24, 96
			}
			if len(r.Resources) != wantResources || len(r.Observations) != wantResources || len(r.Evidence) != wantEvidence {
				t.Fatalf("scale loss: resources %d observations %d evidence %d", len(r.Resources), len(r.Observations), len(r.Evidence))
			}
			active := 0
			for _, f := range r.Findings {
				if f.Source == "service_health" || f.Source == "azure_monitor" {
					active++
				}
			}
			if active != 7 {
				t.Fatalf("active operational findings %d want 7", active)
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
			assertSyntheticLogin(t, log, true)
		})
	}
}

func TestFourSourceNativeFailureIsolation(t *testing.T) {
	for _, mask := range []int{0, 4, 8, 12, 3, 15} {
		t.Run(fmt.Sprintf("failures-%04b", mask), func(t *testing.T) {
			counts := map[string]int{}
			var a App
			a, out, diag, log := fourSourceApp(t, func(req *http.Request) (*http.Response, error) {
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
			} else if mask == 15 {
				wantCode, outcome = 1, "failed"
			}
			if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first, "--debug"}); code != wantCode {
				t.Fatalf("exit %d want %d: %s", code, wantCode, diag)
			}
			r := assertIntegrationArtifacts(t, a, out, diag)
			wantEvidence := 4
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
			assertSyntheticLogin(t, log, true)
		})
	}
	t.Run("all empty", func(t *testing.T) {
		counts := map[string]int{}
		a, out, diag, _ := fourSourceApp(t, func(req *http.Request) (*http.Response, error) {
			source := requestSource(req)
			counts[source]++
			if source == "subscription" {
				return armResponse(200, subscriptionResponse()), nil
			}
			return armResponse(200, `{"value":[]}`), nil
		})
		if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first}); code != 0 {
			t.Fatalf("exit %d: %s", code, diag)
		}
		r := assertIntegrationArtifacts(t, a, out, diag)
		if len(r.Evidence) != 0 || len(r.Findings) != 0 || r.Outcome != "completed" {
			t.Fatal("empty lists invented evidence")
		}
		for _, source := range implementedSources {
			if counts[source] != 1 || r.Sources[source].Status != "completed" || r.Sources[source].Pages != 1 {
				t.Fatalf("empty source %s: %+v", source, r.Sources[source])
			}
		}
		assertGap(t, r, "no_resources_observed")
	})
}

func TestFourSourceNativeOperationalPageFailures(t *testing.T) {
	for _, source := range []string{"service_health", "azure_monitor"} {
		for _, scenario := range []string{"401", "403", "404", "429", "timeout", "unsafe host", "wrong subscription", "redirect", "repeat", "invalid envelope", "mixed records"} {
			t.Run(source+"/"+scenario, func(t *testing.T) {
				counts := map[string]int{}
				var a App
				a, out, diag, _ := fourSourceApp(t, func(req *http.Request) (*http.Response, error) {
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
					case "mixed records":
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
				r := assertIntegrationArtifacts(t, a, out, diag)
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
				if r.Sources[source].Status != "partial" || r.Sources[source].Reason != reason || r.Sources[source].Pages != pages || counts[source] != calls || len(r.Evidence) != 4 || len(r.Findings) != 3 {
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

func TestFourSourceNativeStorageAndPublicationFailures(t *testing.T) {
	for _, mode := range []string{"service_health", "azure_monitor", "final write", "publication"} {
		t.Run(mode, func(t *testing.T) {
			counts := map[string]int{}
			var a App
			a, out, diag, _ := fourSourceApp(t, func(req *http.Request) (*http.Response, error) {
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
				return armResponse(200, integrationList(source, []string{operationalRecord(source, counts[source]-1, snapshot(t, a.Directory).StartedAt)}, next)), nil
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
			want := 4
			if mode == "service_health" {
				want = 3
				if counts["azure_monitor"] != 0 || counts[mode] != 2 || r.Sources[mode].Pages != 1 {
					t.Fatal("storage failure did not stop reads")
				}
			} else if mode == "azure_monitor" && (counts[mode] != 2 || r.Sources[mode].Pages != 1) {
				t.Fatal("failed Monitor page replaced committed page")
			}
			if len(r.Evidence) != want || mode != "publication" && r.EndedAt != nil {
				t.Fatal("last usable native checkpoint lost")
			}
			if _, err := os.Stat(filepath.Join(a.Directory, "runs", r.ID, "report.md")); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("claimed a report after storage/publication failure")
			}
		})
	}
}

func TestFourSourceNativeUnknownAndConflictingObservations(t *testing.T) {
	counts := map[string]int{}
	var a App
	a, out, diag, _ := fourSourceApp(t, func(req *http.Request) (*http.Response, error) {
		source := requestSource(req)
		counts[source]++
		if source == "subscription" {
			return armResponse(200, subscriptionResponse()), nil
		}
		if source == "inventory" || source == "resource_health" {
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
	r := assertIntegrationArtifacts(t, a, out, diag)
	if len(r.Evidence) != 10 || len(r.Findings) != 0 || len(r.Resources) != 0 || len(r.Observations) != 0 {
		t.Fatalf("conflict/dedup semantics lost: evidence %d findings %d", len(r.Evidence), len(r.Findings))
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

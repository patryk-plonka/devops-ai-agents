package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/azure"
	"github.com/patryk-plonka/devops-ai-agents/internal/health"
	"github.com/patryk-plonka/devops-ai-agents/internal/runstore"
)

func operationalPage(source string, now time.Time) health.Page {
	p := health.Page{CollectedAt: now}
	switch source {
	case "inventory":
		p.Resources = []health.ResourceInput{{ID: cliVM, Name: "vm"}}
	case "resource_health":
		p.Health = []health.HealthInput{{StatusID: cliVM + "/providers/Microsoft.ResourceHealth/availabilityStatuses/current", PropertiesPresent: true, State: stringPointer("Unavailable"), ReportedAt: &now}}
	case "service_health":
		p.ServiceHealth = []health.ServiceHealthInput{{ID: "/providers/Microsoft.ResourceHealth/events/synthetic", PropertiesPresent: true, EventType: "ServiceIssue", Status: "Active", LastUpdateTime: &now}}
	case "azure_monitor":
		p.Monitor = []health.MonitorInput{{ID: "/subscriptions/" + first + "/providers/Microsoft.AlertsManagement/alerts/synthetic", PropertiesPresent: true, SignalType: "Metric", Condition: "Fired", WorkflowState: "Closed", TargetResourceID: cliVM, LastModifiedTime: &now}}
	}
	return p
}

func assignCollector(a *App, name string, c health.Collector) {
	switch name {
	case "inventory":
		a.Inventory = c
	case "resource_health":
		a.ResourceHealth = c
	case "service_health":
		a.ServiceHealth = c
	case "azure_monitor":
		a.AzureMonitor = c
	default:
		panic("unknown collector")
	}
}

func TestCLIFourSourceOrderPoliciesAndIndependentFailures(t *testing.T) {
	names := []string{"inventory", "resource_health", "service_health", "azure_monitor"}
	for failures := 0; failures < 16; failures++ {
		t.Run(fmt.Sprintf("failures-%04b", failures), func(t *testing.T) {
			a, out, diag, _ := testApp(t)
			sequence := []string{}
			for i, name := range names {
				assignCollector(&a, name, func(_ context.Context, id string, emit func(health.Page) error) error {
					if id != first {
						t.Fatal("wrong subscription")
					}
					sequence = append(sequence, name)
					saved := snapshot(t, a.Directory)
					if saved.Sources[name].Status != "in_progress" {
						t.Fatal("source state not checkpointed before collection")
					}
					if name == "service_health" || name == "azure_monitor" {
						q := saved.Sources[name].Query
						if q == nil || q.RequestedAt == nil || q.RequestedAt.Before(saved.StartedAt) || !q.ReferenceAt.Equal(saved.StartedAt) {
							t.Fatal("query timing/provenance not committed before source request")
						}
						if name == "service_health" {
							want := health.ServiceHealthQuery(saved.StartedAt)
							if q.QueryStartTime != want.QueryStartTime || q.Cutoff == nil || !q.Cutoff.Equal(*want.Cutoff) {
								t.Fatal("wrong fixed Service Health policy")
							}
						} else if q.TimeRange != "30d" || q.Cutoff != nil {
							t.Fatal("wrong Monitor policy")
						}
					}
					if failures&(1<<i) != 0 {
						category := "access_denied"
						if name == "service_health" {
							category = "login_required"
						}
						return &azure.Failure{Category: category, Message: "raw-error-canary", Code: 1}
					}
					if err := emit(operationalPage(name, time.Now().UTC())); err != nil {
						return err
					}
					if snapshot(t, a.Directory).Sources[name].Pages != 1 {
						t.Fatal("page not committed on callback return")
					}
					return nil
				})
			}
			wantCode, wantOutcome := 3, "partial"
			if failures == 0 {
				wantCode, wantOutcome = 0, "completed"
			}
			if failures == 15 {
				wantCode, wantOutcome = 1, "failed"
			}
			code := a.Run(context.Background(), []string{"health", "check", "--subscription", first})
			if code != wantCode || !reflect.DeepEqual(sequence, names) {
				t.Fatalf("code %d want %d sequence %v diagnostics %s", code, wantCode, sequence, diag)
			}
			r := assertPublished(t, a.Directory, strings.TrimSpace(out.String()))
			if r.Outcome != wantOutcome || r.Sources["advisor"].Status != "not_checked" || r.Coverage != "incomplete" {
				t.Fatal("wrong execution/coverage")
			}
			for i, name := range names {
				want := "completed"
				if failures&(1<<i) != 0 {
					want = "failed"
				}
				if r.Sources[name].Status != want || !strings.Contains(diag.String(), name+": "+want) {
					t.Fatalf("source state/diagnostic lost: %s", name)
				}
			}
			b, _ := json.Marshal(r)
			reportBytes, err := os.ReadFile(strings.TrimSpace(out.String()))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(b)+string(reportBytes)+diag.String(), "raw-error-canary") {
				t.Fatal("raw error leaked")
			}
			if strings.Count(strings.TrimSpace(out.String()), "\n") != 0 {
				t.Fatal("stdout contains more than published path")
			}
		})
	}
}

type operationalFailureStore struct {
	Storage
	source string
}

func (s operationalFailureStore) Save(r *health.Run) error {
	if r.Sources[s.source].Pages == 2 {
		return errors.New("checkpoint-secret-canary")
	}
	return s.Storage.Save(r)
}

func TestCLIOperationalCancellationAndStorageStops(t *testing.T) {
	for _, source := range []string{"service_health", "azure_monitor"} {
		for _, stop := range []string{"cancel", "save"} {
			t.Run(source+"/"+stop, func(t *testing.T) {
				a, out, diag, _ := testApp(t)
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if stop == "save" {
					a.NewStore = func(root string, r *health.Run) (Storage, error) {
						s, err := runstore.New(root, r)
						return operationalFailureStore{Storage: s, source: source}, err
					}
				}
				a.Inventory = func(_ context.Context, _ string, emit func(health.Page) error) error {
					return emit(operationalPage("inventory", time.Now().UTC()))
				}
				if source == "service_health" {
					a.AzureMonitor = func(context.Context, string, func(health.Page) error) error {
						t.Fatal("later source attempted after stop")
						return nil
					}
				}
				assignCollector(&a, source, func(ctx context.Context, _ string, emit func(health.Page) error) error {
					if err := emit(operationalPage(source, time.Now().UTC())); err != nil {
						return err
					}
					if snapshot(t, a.Directory).Sources[source].Pages != 1 {
						t.Fatal("first operational page not saved")
					}
					if stop == "cancel" {
						cancel()
						return ctx.Err()
					}
					p := operationalPage(source, time.Now().UTC())
					if source == "service_health" {
						p.ServiceHealth[0].ID += "-second"
					} else {
						p.Monitor[0].ID += "-second"
					}
					return emit(p)
				})
				code := a.Run(ctx, []string{"health", "check", "--subscription", first})
				wantCode := 130
				if stop == "save" {
					wantCode = 1
				}
				r := snapshot(t, a.Directory)
				if code != wantCode || r.Sources[source].Pages != 1 || len(r.Evidence) != 2 {
					t.Fatalf("code %d source %+v evidence %d diagnostics %s", code, r.Sources[source], len(r.Evidence), diag)
				}
				if stop == "cancel" {
					if r.Outcome != "cancelled" {
						t.Fatal("cancellation not saved")
					}
					assertPublished(t, a.Directory, strings.TrimSpace(out.String()))
				} else if out.Len() != 0 || r.EndedAt != nil || !strings.Contains(diag.String(), "Last saved snapshot") {
					t.Fatal("failed checkpoint claimed publication")
				}
				if strings.Contains(diag.String(), "secret-canary") {
					t.Fatal("raw storage error leaked")
				}
			})
		}
	}
}

func TestProductionDefaultsUseFourNativeCollectors(t *testing.T) {
	// Real discovery/credential subprocesses use a temporary fake az. All default
	// HTTP traffic is replaced at the Go boundary before any collector is enabled.
	a, out, diag, log := integrationApp(t, func(*http.Request) (*http.Response, error) { t.Fatal("unexpected injected transport"); return nil, nil })
	a.Inventory, a.ResourceHealth, a.ServiceHealth, a.AzureMonitor = nil, nil, nil, nil
	a.Validate = nil
	sequence := []string{}
	http.DefaultTransport = armFixture(func(req *http.Request) (*http.Response, error) {
		if req.Method != "GET" || req.URL.Host != "management.azure.com" || req.Header.Get("Authorization") != "Bearer synthetic-token-canary" {
			t.Fatal("default escaped authenticated read boundary")
		}
		saved := snapshot(t, a.Directory)
		switch req.URL.Path {
		case "/subscriptions/" + first:
			return armResponse(200, subscriptionResponse()), nil
		case "/subscriptions/" + first + "/resources":
			sequence = append(sequence, "inventory")
			return armResponse(200, `{"value":[]}`), nil
		case "/subscriptions/" + first + "/providers/Microsoft.ResourceHealth/availabilityStatuses":
			sequence = append(sequence, "resource_health")
			return armResponse(200, `{"value":[]}`), nil
		case "/subscriptions/" + first + "/providers/Microsoft.ResourceHealth/events":
			sequence = append(sequence, "service_health")
			q := health.ServiceHealthQuery(saved.StartedAt)
			if req.URL.Query().Get("api-version") != "2025-05-01" || req.URL.Query().Get("queryStartTime") != q.QueryStartTime || saved.Sources["service_health"].Query == nil || saved.Sources["service_health"].Query.RequestedAt == nil {
				t.Fatal("default Service Health query not bound to saved run policy")
			}
			return armResponse(200, fmt.Sprintf(`{"value":[{"id":"/providers/Microsoft.ResourceHealth/events/default","properties":{"status":"Active","eventType":"HealthAdvisory","lastUpdateTime":%q}}]}`, saved.StartedAt.Format(time.RFC3339Nano))), nil
		case "/subscriptions/" + first + "/providers/Microsoft.AlertsManagement/alerts":
			sequence = append(sequence, "azure_monitor")
			if req.URL.Query().Encode() != "api-version=2019-03-01&includeContext=false&includeEgressConfig=false&monitorCondition=Fired&pageCount=100&timeRange=30d" || saved.Sources["azure_monitor"].Query == nil || saved.Sources["azure_monitor"].Query.RequestedAt == nil {
				t.Fatal("default Monitor query not committed")
			}
			return armResponse(200, `{"value":[{"id":"/subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft.AlertsManagement/alerts/default","properties":{"essentials":{"signalType":"Log","monitorCondition":"Fired","alertState":"Acknowledged","description":"description-canary","lastModifiedUserName":"username-canary"},"context":{"SearchQuery":"query-canary"}}}]}`), nil
		default:
			t.Fatal("unexpected default operation")
			return nil, nil
		}
	})
	if code := a.Run(integrationContext(t), []string{"health", "check", "--subscription", first, "--debug"}); code != 0 {
		t.Fatalf("code %d diagnostics %s", code, diag)
	}
	if !reflect.DeepEqual(sequence, []string{"inventory", "resource_health", "service_health", "azure_monitor"}) {
		t.Fatalf("wrong native order %v", sequence)
	}
	r := assertIntegrationArtifacts(t, a, out, diag)
	if len(r.Findings) != 2 || len(r.Evidence) != 2 || len(r.Observations) != 0 {
		t.Fatal("operational evidence became resource faults or was lost")
	}
	b, err := os.ReadFile(filepath.Join(a.Directory, "runs", r.ID, "report.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := html.UnescapeString(string(b))
	if !strings.Contains(text, "workflow response: Acknowledged") || !strings.Contains(text, "HealthAdvisory") {
		t.Fatal("new-source facts missing from published report")
	}
	assertSyntheticLogin(t, log, true)
}

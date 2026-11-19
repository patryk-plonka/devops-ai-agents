package report

import (
	"bytes"
	"context"
	"encoding/json"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

// A representative synthetic schema-2 run: distinct categories/workflows,
// uncertainty, old/conflicting observations and filtered source markup.
func operationalExample() health.Run {
	r := example()
	r.SchemaVersion = 2
	now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
	r.ID = "synthetic-operational-run"
	r.Revision = 17
	r.StartedAt = now
	r.UpdatedAt = now
	r.EndedAt = &now
	serviceQuery, monitorQuery := health.ServiceHealthQuery(now), health.MonitorQuery(now)
	serviceQuery.RequestedAt = &now
	monitorQuery.RequestedAt = &now
	r.Sources["service_health"] = health.Source{Status: "completed", Query: &serviceQuery}
	r.Sources["azure_monitor"] = health.Source{Status: "completed", Query: &monitorQuery}
	r.Sources["advisor"] = health.Source{Status: "not_checked", Reason: "not_implemented"}
	previous, future := now.Add(-time.Hour), now.Add(24*time.Hour)
	events := []health.ServiceHealthInput{}
	for _, category := range []string{"ServiceIssue", "PlannedMaintenance", "HealthAdvisory", "SecurityAdvisory"} {
		event := health.ServiceHealthInput{ID: "/providers/Microsoft.ResourceHealth/events/" + category, PropertiesPresent: true, EventType: category, Status: "Active", Level: "Warning", LastUpdateTime: &now, ImpactStartTime: &previous, EventSource: "ResourceHealth", Impact: []health.ServiceImpactInput{{Service: "Compute <script>x</script>|[link](https://example.test)", Regions: []health.ServiceRegionInput{{Name: "West US", Status: "Active", LastUpdateTime: &now, ImpactedSubscriptions: []string{sub}}, {Name: "Unknown region", Status: "Active"}}}}}
		if category == "PlannedMaintenance" {
			event.ImpactStartTime = &future
		}
		if category == "SecurityAdvisory" {
			event.Impact = nil
			sensitive := true
			event.Sensitive = &sensitive
		}
		events = append(events, event)
	}
	for _, state := range []string{"Resolved", "FutureState"} {
		events = append(events, health.ServiceHealthInput{ID: "/providers/Microsoft.ResourceHealth/events/" + state, PropertiesPresent: true, EventType: "ServiceIssue", Status: state, LastUpdateTime: &now})
	}
	events = append(events, health.ServiceHealthInput{ID: "/providers/Microsoft.ResourceHealth/events/Billing", PropertiesPresent: true, EventType: "Billing", Status: "Active", LastUpdateTime: &now})
	r.NormalizePage("service_health", health.Page{CollectedAt: now, ServiceHealth: events})
	alerts := []health.MonitorInput{}
	for _, workflow := range []string{"New", "Acknowledged", "Closed"} {
		suppressed := workflow == "Closed"
		input := health.MonitorInput{ID: "/subscriptions/" + sub + "/providers/Microsoft.AlertsManagement/alerts/" + workflow, PropertiesPresent: true, TargetResourceID: resource + "vm-00", SignalType: "Metric", Condition: "Fired", WorkflowState: workflow, Severity: "Sev2", IsSuppressed: &suppressed, StartTime: &previous, LastModifiedTime: &now, LastModifiedTimeByMonitorService: &previous}
		if workflow == "Acknowledged" {
			input.SignalType = "Log"
			input.TargetResourceID = "/subscriptions/" + sub + "/resourceGroups/g/providers/Microsoft.OperationalInsights/workspaces/logs"
		}
		if workflow == "New" {
			input.TargetResourceID = "/subscriptions/99999999-9999-9999-9999-999999999999/resourceGroups/g/providers/Microsoft.Compute/virtualMachines/foreign-canary"
		}
		alerts = append(alerts, input)
	}
	for _, condition := range []string{"Resolved", "FutureCondition"} {
		alerts = append(alerts, health.MonitorInput{ID: "/subscriptions/" + sub + "/providers/Microsoft.AlertsManagement/alerts/" + condition, PropertiesPresent: true, SignalType: "Log", Condition: condition, LastModifiedTime: &now})
	}
	// Two timed observations for one alert, superseded by its resolved state.
	superseded := health.MonitorInput{ID: "/subscriptions/" + sub + "/providers/Microsoft.AlertsManagement/alerts/superseded", PropertiesPresent: true, SignalType: "Metric", Condition: "Fired", LastModifiedTime: &previous}
	alerts = append(alerts, superseded)
	superseded.Condition = "Resolved"
	superseded.LastModifiedTime = &now
	alerts = append(alerts, superseded)
	// A tied conflict must not appear as an active summary.
	conflict := superseded
	conflict.ID = "/subscriptions/" + sub + "/providers/Microsoft.AlertsManagement/alerts/conflict"
	alerts = append(alerts, conflict)
	conflict.Condition = "Fired"
	alerts = append(alerts, conflict)
	r.NormalizePage("azure_monitor", health.Page{CollectedAt: now.Add(time.Minute), Monitor: alerts})
	r.RebuildCoverage()
	return r
}

func TestOperationalReportMeaningPrivacyAndDeterminism(t *testing.T) {
	r := operationalExample()
	before, _ := json.Marshal(r)
	b, err := Markdown(r)
	if err != nil {
		t.Fatal(err)
	}
	text := html.UnescapeString(string(b))
	for _, want := range []string{"# Subscription health report", "Not checked: advisor.", "INCOMPLETE coverage", "30-day lower cutoff (UTC): 2026-10-16T12:00:00Z", "queryStartTime: 10/16/2026", "API timeRange: 30d", "not an atomic snapshot", "Older unchanged active events may be absent", "Resource impact: unknown", "Stateless alerts can remain Fired", "stateful conditions can outlive", "workspace target does not identify every resource", "workflow response: Acknowledged", "workflow response: Closed", "notification suppressed: true", "source_unknown", "unknown_region_attribution", "sensitive_details_not_retrieved", "Source entity", "Excluded records", "Unclassified records", "Service Health future PlannedMaintenance notice", "ServiceIssue", "HealthAdvisory", "SecurityAdvisory", "conflicting_operational_evidence", "superseded_operational_evidence", "Other saved observations"} {
		if !strings.Contains(text, want) {
			t.Errorf("missing %q", want)
		}
	}
	for _, bad := range []string{"foreign-canary", "99999999-9999-9999-9999-999999999999", "credential-canary", "<script>", "[link](https://example.test)", "\n# heading", "\x1b"} {
		if bytes.Contains(b, []byte(bad)) {
			t.Errorf("unsafe output %q", bad)
		}
	}
	activeText := text[strings.Index(text, "## Azure Monitor"):strings.Index(text, "## Resource Health findings")]
	activeText = strings.Split(activeText, "### Other saved observations")[0]
	if strings.Contains(activeText, "/superseded") || strings.Contains(activeText, "/conflict") {
		t.Fatal("inactive or conflicting state presented as active")
	}
	for _, e := range r.Evidence {
		if !strings.Contains(text, "### "+e.ID) || !strings.Contains(text, "["+e.ID+"](run.json)") {
			t.Fatal("evidence omitted")
		}
	}
	for _, f := range r.Findings {
		if !strings.Contains(text, f.ID) {
			t.Fatal("finding omitted")
		}
	}
	after, _ := json.Marshal(r)
	if !bytes.Equal(before, after) {
		t.Fatal("renderer mutated snapshot")
	}
	slices.Reverse(r.Evidence)
	slices.Reverse(r.Findings)
	slices.Reverse(r.Gaps)
	for i := range r.Evidence {
		if p := r.Evidence[i].ServiceHealth; p != nil {
			slices.Reverse(p.Impact)
			for j := range p.Impact {
				slices.Reverse(p.Impact[j].Regions)
			}
		}
		slices.Reverse(r.Evidence[i].Limitations)
	}
	other, err := Markdown(r)
	if err != nil || !bytes.Equal(b, other) {
		t.Fatal("operational report depends on input order")
	}
}

func TestOperationalCoverageWordingFollowsSavedStatus(t *testing.T) {
	for _, status := range []string{"completed", "failed", "partial", "not_checked"} {
		t.Run(status, func(t *testing.T) {
			r := operationalExample()
			r.Evidence = nil
			r.Findings = nil
			for _, name := range []string{"service_health", "azure_monitor"} {
				s := r.Sources[name]
				s.Status = status
				s.Reason = "test_reason"
				if status == "not_checked" {
					s.Query = nil
				}
				r.Sources[name] = s
			}
			r.RebuildCoverage()
			b, err := Markdown(r)
			if err != nil {
				t.Fatal(err)
			}
			text := html.UnescapeString(string(b))
			if !strings.Contains(text, "Collection status: "+status) || !strings.Contains(text, "INCOMPLETE coverage") || !strings.Contains(text, "No active summaries supported") {
				t.Fatal("misleading source/empty coverage")
			}
			header := strings.Split(text, "## Source coverage")[0]
			if status == "not_checked" {
				if !strings.Contains(header, "azure_monitor") || !strings.Contains(header, "service_health") || !strings.Contains(text, "No query policy recorded") {
					t.Fatal("unchecked sources omitted")
				}
			} else if strings.Contains(header, "azure_monitor") || strings.Contains(header, "service_health") {
				t.Fatal("attempted source labeled unchecked")
			}
		})
	}
}

func TestCancelledBeforeCollectionReportsOnlyConfiguredQuery(t *testing.T) {
	for _, source := range []string{"service_health", "azure_monitor"} {
		t.Run(source, func(t *testing.T) {
			r := health.NewRun("synthetic", "test")
			r.ID, r.Revision = "synthetic-cancelled", 1
			r.Scope = &health.Scope{SubscriptionID: sub}
			now := time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC)
			r.StartedAt, r.EndedAt = now, &now
			service, monitor := health.ServiceHealthQuery(now), health.MonitorQuery(now)
			r.Sources["service_health"] = health.Source{Status: "not_checked", Query: &service}
			r.Sources["azure_monitor"] = health.Source{Status: "not_checked", Query: &monitor}
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			collect := func(context.Context, string, func(health.Page) error) error {
				t.Fatal("collector ran after cancellation")
				return nil
			}
			code, err := health.Check(ctx, &r, []health.NamedCollector{{Name: source, Collect: collect}}, func(run *health.Run) error {
				if run.Sources[source].Query.RequestedAt == nil {
					t.Fatal("preparation was not checkpointed")
				}
				cancel()
				return nil
			})
			s := r.Sources[source]
			if err != nil || code != 130 || s.Reason != "not_attempted" || s.Query == nil || s.Query.RequestedAt != nil {
				t.Fatalf("cancelled source did not retain only its configured policy: %+v, exit %d, error %v", s, code, err)
			}
			b, err := Markdown(r)
			if err != nil {
				t.Fatal(err)
			}
			text := html.UnescapeString(string(b))
			want := "Configured API queryStartTime: 10/16/2026"
			if source == "azure_monitor" {
				want = "Configured API timeRange: 30d"
			}
			if !strings.Contains(text, "Collection did not start.") || !strings.Contains(text, want) || strings.Contains(text, "Query prepared (UTC)") || strings.Contains(text, "first requested") {
				t.Fatal("report did not distinguish configured policy from collection")
			}
		})
	}
}

func TestSyntheticOperationalReportExample(t *testing.T) {
	r := operationalExample()
	b, err := Markdown(r)
	if err != nil {
		t.Fatal(err)
	}
	want, err := os.ReadFile("testdata/operational/report.md")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(b, want) {
		t.Fatal("schema-2 report fixture differs")
	}
	data, err := os.ReadFile("testdata/operational/run.json")
	if err != nil {
		t.Fatal(err)
	}
	var saved health.Run
	if json.Unmarshal(data, &saved) != nil {
		t.Fatal("invalid synthetic snapshot")
	}
	fromSnapshot, err := Markdown(saved)
	if err != nil || !bytes.Equal(fromSnapshot, want) {
		t.Fatal("schema-2 fixture snapshot/report mismatch")
	}
}

func TestReportExampleLinksResolveToMatchingSnapshot(t *testing.T) {
	for _, reportPath := range []string{"testdata/mixed-report.md", "testdata/operational/report.md", "testdata/advisor/report.md"} {
		t.Run(reportPath, func(t *testing.T) {
			b, err := os.ReadFile(reportPath)
			if err != nil {
				t.Fatal(err)
			}
			text := html.UnescapeString(string(b))
			identity := regexp.MustCompile(`(?m)^Run ID: (.+)$`).FindStringSubmatch(text)
			revision := regexp.MustCompile(`(?m)^Revision: ([0-9]+)$`).FindStringSubmatch(text)
			if len(identity) != 2 || len(revision) != 2 {
				t.Fatal("example lacks run identity/revision")
			}
			revisionNumber, err := strconv.Atoi(revision[1])
			if err != nil {
				t.Fatal(err)
			}
			links := regexp.MustCompile(`\[([^\]\n]+)\]\(([^)\n]+)\)`).FindAllStringSubmatch(string(b), -1)
			if len(links) == 0 {
				t.Fatal("example has no snapshot/evidence links")
			}
			snapshots := map[string]health.Run{}
			for _, link := range links {
				label, target := link[1], link[2]
				if target != "run.json" {
					t.Fatalf("unexpected example link target %q", target)
				}
				snapshotPath := filepath.Join(filepath.Dir(reportPath), target)
				saved, exists := snapshots[snapshotPath]
				if !exists {
					data, err := os.ReadFile(snapshotPath)
					if err != nil {
						t.Fatal(err)
					}
					if err := json.Unmarshal(data, &saved); err != nil {
						t.Fatal(err)
					}
					snapshots[snapshotPath] = saved
				}
				if saved.ID != identity[1] || saved.Revision != revisionNumber {
					t.Fatalf("link opens %s revision %d; report identifies %s revision %d", saved.ID, saved.Revision, identity[1], revisionNumber)
				}
				if label == "run.json" {
					continue
				}
				found := false
				for _, e := range saved.Evidence {
					if e.ID == label {
						found = true
						break
					}
				}
				if !found || !strings.Contains(text, "### "+label+"\n") {
					t.Fatalf("evidence link %s does not resolve in matching snapshot/index", label)
				}
			}
		})
	}
}

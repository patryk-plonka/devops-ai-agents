package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

func TestNativeHealthCheckToSelectedDrafts(t *testing.T) {
	var app App
	app, out, diag, cliLog := nativeSourceApp(t, func(req *http.Request) (*http.Response, error) {
		source := requestSource(req)
		if source == "subscription" {
			return armResponse(200, subscriptionResponse()), nil
		}
		start := snapshot(t, app.Directory).StartedAt
		var records []string
		switch source {
		case "inventory":
			records = []string{inventoryRecord(0), inventoryRecord(1), inventoryRecord(2),
				strings.Replace(inventoryRecord(3), `"name":"vm-003"`, `"name":"unselected-vm-marker"`, 1)}
		case "resource_health":
			available := integrationHealthRecord(0, "current", `{"availabilityState":"Available"}`)
			degraded := integrationHealthRecord(0, "current", `{"availabilityState":"Degraded"}`)
			records = []string{
				strings.Replace(degraded, "2026-11-06T10:00:00Z", "2026-11-06T09:10:00Z", 1),
				available,
				strings.Replace(degraded, "2026-11-06T10:00:00Z", "2026-11-06T09:20:00Z", 1),
				integrationHealthRecord(2, "current", `{"availabilityState":"Available"}`),
				healthRecord(3),
			}
		case "service_health":
			selected := strings.Replace(operationalRecord(source, 1, start), "Compute | <script>[link](https://bad.example)", "Compute | <script>", 1)
			selected = strings.Replace(selected, `"impactedRegion":"East US","status":`,
				fmt.Sprintf(`"impactedRegion":"East US","lastUpdateTime":%q,"status":`, draftIntegrationTime(start.Add(-15*time.Minute))), 1)
			records = []string{
				selected,
				strings.Replace(operationalRecord(source, 2, start), "Compute | <script>[link](https://bad.example)", "unselected-service-marker", 1),
			}
		case "azure_monitor":
			stamp := start.Format(time.RFC3339Nano)
			records = []string{integrationMonitorRecord(1, stamp, "Fired", integrationResource(1)), integrationMonitorRecord(3, stamp, "Fired", integrationResource(3))}
		case "advisor":
			records = []string{operationalRecord(source, 2, start), operationalRecord(source, 3, start)}
		default:
			t.Fatalf("unexpected collection request %q", source)
		}
		return armResponse(200, integrationList(source, records, "")), nil
	})
	ctx := integrationContext(t)
	if code := app.Run(ctx, []string{"health", "check", "--subscription", first}); code != 0 {
		t.Fatalf("check exit %d\n%s", code, diag)
	}
	run := assertIntegrationArtifacts(t, app, out, diag, "incomplete", "Incomplete coverage · ")
	assertSyntheticLogin(t, cliLog, true)
	reportPath := strings.TrimSpace(out.String())
	runDir := filepath.Dir(reportPath)
	jsonPath := filepath.Join(runDir, "run.json")
	originalReport, originalJSON := readDraftCLIFile(t, reportPath), readDraftCLIFile(t, jsonPath)
	originalAzureCalls := readDraftCLIFile(t, cliLog)
	prohibitDraftCollection(t, &app)
	http.DefaultTransport = armFixture(func(*http.Request) (*http.Response, error) {
		t.Error("draft attempted network access")
		return nil, errors.New("network disabled during drafting")
	})

	evidence := make(map[string]health.Evidence, len(run.Evidence))
	inventory := make(map[string]health.Resource, len(run.Resources))
	for _, e := range run.Evidence {
		evidence[e.ID] = e
	}
	for _, resource := range run.Resources {
		inventory[strings.ToLower(resource.ID)] = resource
	}
	// Request only full IDs visibly printed in the published report. The JSON
	// identifies which of those findings belong to each synthetic scenario.
	reportedIDs := reportedFindingIDs(string(originalReport))
	var history []health.Finding
	var monitor, notice, advisor health.Finding
	for _, f := range run.Findings {
		if !slices.Contains(reportedIDs, f.ID) {
			t.Fatalf("published report omitted finding %s", f.ID)
		}
		switch {
		case f.Category == "resource_health_issue" && strings.EqualFold(f.ResourceID, integrationResource(0)):
			history = append(history, f)
		case f.Source == "azure_monitor" && strings.EqualFold(f.ResourceID, integrationResource(1)):
			monitor = f
		case f.Source == "service_health" && strings.HasSuffix(f.EntityID, "/event-001"):
			notice = f
		case f.Source == "advisor" && strings.EqualFold(f.ResourceID, integrationResource(2)):
			advisor = f
		}
	}
	if len(history) != 2 || monitor.ID == "" || notice.ID == "" || advisor.ID == "" || len(run.Findings) != 9 {
		t.Fatalf("expected two historical, three other selected and four unselected findings, got %d total", len(run.Findings))
	}
	for _, f := range []health.Finding{history[0], history[1], monitor, notice, advisor} {
		if len(f.EvidenceIDs) != 1 {
			t.Fatalf("unexpected supporting evidence for %s", f.ID)
		}
	}
	files := make(map[string][]byte)
	readBatch := func(kind string, findings ...health.Finding) {
		t.Helper()
		out.Reset()
		diag.Reset()
		var ids, want []string
		for _, f := range findings {
			ids = append(ids, reportedIDs[slices.Index(reportedIDs, f.ID)])
			path := filepath.Join(runDir, "drafts", kind+"-"+f.ID+".md")
			if !slices.Contains(want, path) {
				want = append(want, path)
			}
		}
		if code := app.Run(ctx, draftArgs(reportPath, kind, ids...)); code != 0 {
			t.Fatalf("%s exit %d\n%s", kind, code, diag)
		}
		got := strings.Split(strings.TrimSuffix(out.String(), "\n"), "\n")
		if !slices.Equal(got, want) {
			t.Fatalf("delivered paths %v, want %v", got, want)
		}
		for _, path := range got {
			files[path] = readDraftCLIFile(t, path)
		}
	}
	readBatch("ticket", monitor, history[1], history[0], monitor)
	readBatch("recommendation", notice, advisor)
	entries, err := os.ReadDir(filepath.Join(runDir, "drafts"))
	if err != nil || len(entries) != len(files) {
		t.Fatalf("draft directory contains %d entries, want %d (%v)", len(entries), len(files), err)
	}

	git := gitWithRepositoryIgnore(t, app.Directory)
	evidenceID := regexp.MustCompile(`ev-[0-9a-f]{64}`)
	checkDraft := func(kind string, f health.Finding, extraIDs []string, want ...string) {
		t.Helper()
		path := filepath.Join(runDir, "drafts", kind+"-"+f.ID+".md")
		raw := string(files[path])
		body := html.UnescapeString(raw)
		collection := map[string]string{"resource_health": "Resource Health", "azure_monitor": "Azure Monitor", "service_health": "Service Health", "advisor": "Azure Advisor"}
		source := evidence[f.EvidenceIDs[0]].Source
		want = append(want, "Run ID: `"+run.ID+"`", fmt.Sprintf("Revision: %d", run.Revision),
			fmt.Sprintf("Run outcome: %s. Overall coverage: %s.", run.Outcome, run.Coverage),
			collection[source]+" collection: "+run.Sources[source].Status,
			"Supporting evidence: `"+f.EvidenceIDs[0]+"`")
		for _, value := range want {
			if !strings.Contains(body, value) {
				t.Errorf("%s missing saved fact or qualification %q\n%s", filepath.Base(path), value, body)
			}
		}
		if got := reportedFindingIDs(body); !slices.Equal(got, []string{f.ID}) {
			t.Errorf("unexpected draft finding IDs %v", got)
		}
		expected := slices.Concat(f.EvidenceIDs, extraIDs, inventory[strings.ToLower(f.ResourceID)].EvidenceIDs)
		got := evidenceID.FindAllString(body, -1)
		slices.Sort(got)
		slices.Sort(expected)
		if !slices.Equal(slices.Compact(got), slices.Compact(expected)) {
			t.Errorf("%s cites %v, want %v", filepath.Base(path), got, expected)
		}
		for _, value := range []string{"/subscriptions/", first, tenant, app.Directory, "unselected-vm-marker", "unselected-service-marker", "canary", "consulting", "[link](", "<script>"} {
			if strings.Contains(raw, value) {
				t.Errorf("draft exposed %q", value)
			}
		}
		rel, err := filepath.Rel(app.Directory, path)
		if err != nil {
			t.Fatal(err)
		}
		if b, err := git("check-ignore", "--no-index", "--", rel); err != nil || strings.TrimSpace(string(b)) != filepath.ToSlash(rel) {
			t.Errorf("Git did not ignore generated draft %s (%v): %s", rel, err, b)
		}
	}
	t.Run("facts and trace", func(t *testing.T) {
		hasScopeGap := func(source, reason string) bool {
			return slices.ContainsFunc(run.Gaps, func(g health.Gap) bool {
				return g.Source == source && g.ResourceID == "" && g.EntityID == "" && g.Reason == reason
			})
		}
		var deciding health.Evidence
		for _, e := range run.Evidence {
			if e.Source == "resource_health" && e.State == "Available" && strings.EqualFold(e.ResourceID, integrationResource(0)) {
				deciding = e
			}
		}
		if deciding.ID == "" {
			t.Fatal("missing later Available evidence in saved run")
		}
		for _, f := range history {
			selected := evidence[f.EvidenceIDs[0]]
			target := inventory[strings.ToLower(f.ResourceID)]
			checkDraft("ticket", f, []string{deciding.ID}, "# Ticket: Resource Health reported Degraded",
				"resource name "+target.Name+", type "+target.Type, "historical. The later deciding evidence records "+deciding.State,
				"Workload impact is unknown from the Resource Health evidence.",
				healthDraftIntegrationRow(selected), healthDraftIntegrationRow(deciding))
		}

		e := evidence[monitor.EvidenceIDs[0]]
		target := inventory[strings.ToLower(monitor.ResourceID)]
		s := run.Sources["azure_monitor"]
		if e.Monitor == nil || e.Monitor.IsSuppressed == nil || !*e.Monitor.IsSuppressed || s.Query == nil || s.Query.TimeRange != "30d" ||
			!hasScopeGap(e.Source, "monitor_30_day_retention_and_stateless_limits") ||
			!slices.ContainsFunc(run.Gaps, func(g health.Gap) bool {
				return strings.EqualFold(g.ResourceID, monitor.ResourceID) && g.Source == "resource_health" && g.Reason == "missing_health_evidence"
			}) {
			t.Fatal("missing saved Monitor facts, query scope or Resource Health gap")
		}
		checkDraft("ticket", monitor, nil, "# Ticket: Azure Monitor Fired alert", "current Resource Health evidence is missing",
			"Azure Monitor reported target: resource name "+target.Name+", type "+target.Type,
			fmt.Sprintf("Signal %s, condition %s, workflow %s, suppressed yes.", e.Monitor.SignalType, e.Monitor.Condition, e.Monitor.WorkflowState),
			"Source start unknown, source update "+draftIntegrationTime(*e.ReportedAt)+", collected "+draftIntegrationTime(e.CollectedAt),
			"Workload impact is unknown from the alert evidence.", "30-day Fired view", "Saved run reference "+draftIntegrationTime(s.Query.ReferenceAt),
			"the 30-day view can omit older alerts and can include stateless alerts that remain Fired")

		e = evidence[notice.EvidenceIDs[0]]
		service := e.ServiceHealth
		s = run.Sources["service_health"]
		if service == nil || len(service.Impact) != 1 || len(service.Impact[0].Regions) != 1 || s.Query == nil || s.Query.Cutoff == nil ||
			!hasScopeGap(e.Source, "service_health_30_day_update_window") {
			t.Fatal("missing saved Service Health evidence or query scope")
		}
		impact, region := service.Impact[0], service.Impact[0].Regions[0]
		if region.Attribution != "selected_subscription" || region.LastUpdateTime == nil ||
			!region.LastUpdateTime.Equal(run.StartedAt.Add(-15*time.Minute)) || region.LastUpdateTime.Equal(*e.ReportedAt) || region.LastUpdateTime.Equal(e.CollectedAt) {
			t.Fatal("missing saved region attribution or distinct regional source update")
		}
		checkDraft("recommendation", notice, nil, "# Recommendation: Review the Azure Service Health notice",
			"fixed 30-day update window from run start "+draftIntegrationTime(s.Query.ReferenceAt)+" with an update cutoff of "+draftIntegrationTime(*s.Query.Cutoff)+". Older unchanged notices may be absent.",
			"At the time of the saved observation, the planned maintenance was scheduled to start later.",
			fmt.Sprintf("Notice category %s, status %s. Source update %s, impact start %s, impact mitigation unknown, collected %s",
				service.Category, service.Status, draftIntegrationTime(*e.ReportedAt), draftIntegrationTime(*service.ImpactStartTime), draftIntegrationTime(e.CollectedAt)),
			"Reported service: "+impact.Service,
			fmt.Sprintf("Region attributed to the checked subscription: %s. Status %s, source update %s.", region.Name, region.Status, draftIntegrationTime(*region.LastUpdateTime)),
			"The source did not enumerate affected resources. Workload impact is unknown.")

		e = evidence[advisor.EvidenceIDs[0]]
		i := slices.IndexFunc(run.AdvisorInstances, func(instance health.AdvisorInstance) bool { return instance.FindingID == advisor.ID })
		if i == -1 || !run.HasAssessmentSourceProof("advisor") {
			t.Fatal("missing saved Advisor instance or cached query proof")
		}
		instance := run.AdvisorInstances[i]
		target = inventory[strings.ToLower(instance.ResourceID)]
		checkDraft("recommendation", advisor, nil, "# Recommendation: Review the Azure Advisor recommendation",
			fmt.Sprintf("workflow %s, category %s, Advisor impact %s.", instance.Status, instance.Category, instance.Impact),
			"Source evaluation "+draftIntegrationTime(*instance.LastRefreshed)+". Source workflow update "+draftIntegrationTime(*instance.LastUpdated),
			"Advisor assessed target: resource name "+target.Name+", type "+target.Type,
			"cached recommendation list for the checked subscription, with no date filter or refresh.",
			"Workload impact is unknown.",
			"Supporting Advisor observation, collected "+draftIntegrationTime(e.CollectedAt))
	})

	if !bytes.Equal(originalReport, readDraftCLIFile(t, reportPath)) || !bytes.Equal(originalJSON, readDraftCLIFile(t, jsonPath)) {
		t.Fatal("drafting changed the reviewed report or its paired JSON")
	}
	if !bytes.Equal(originalAzureCalls, readDraftCLIFile(t, cliLog)) {
		t.Fatal("drafting invoked the Azure CLI")
	}

	t.Run("reject transposed historical evidence before publishing", func(t *testing.T) {
		copyApp, out, diag, _ := testApp(t)
		prohibitDraftCollection(t, &copyApp)
		if err := json.Unmarshal(originalJSON, &run); err != nil {
			t.Fatal(err)
		}
		a := slices.IndexFunc(run.Findings, func(f health.Finding) bool { return f.ID == history[0].ID })
		b := slices.IndexFunc(run.Findings, func(f health.Finding) bool { return f.ID == history[1].ID })
		run.Findings[a].EvidenceIDs, run.Findings[b].EvidenceIDs = run.Findings[b].EvidenceIDs, run.Findings[a].EvidenceIDs
		data, err := json.Marshal(run)
		if err != nil {
			t.Fatal(err)
		}
		dir := filepath.Join(copyApp.Directory, "runs", run.ID)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, "report.md")
		if err := os.WriteFile(path, originalReport, 0600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "run.json"), data, 0600); err != nil {
			t.Fatal(err)
		}
		if code := copyApp.Run(ctx, draftArgs(path, "ticket", monitor.ID, history[0].ID)); code != 2 || out.Len() != 0 ||
			!strings.Contains(diag.String(), "Cannot select findings. selected finding does not match its supporting evidence") {
			t.Fatalf("mixed request with transposed evidence returned %d, stdout %q\n%s", code, out, diag)
		}
		if _, err := os.Stat(filepath.Join(dir, "drafts")); !os.IsNotExist(err) {
			t.Fatal("a mixed request with transposed evidence created output")
		}
	})
}

func reportedFindingIDs(body string) []string {
	return regexp.MustCompile(`finding-[0-9a-f]{64}`).FindAllString(body, -1)
}

func draftIntegrationTime(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func healthDraftIntegrationRow(e health.Evidence) string {
	return fmt.Sprintf("%s. Source reported %s, source occurrence %s, collected %s (`%s`).",
		e.State, draftIntegrationTime(*e.ReportedAt), draftIntegrationTime(*e.OccurredAt), draftIntegrationTime(e.CollectedAt), e.ID)
}

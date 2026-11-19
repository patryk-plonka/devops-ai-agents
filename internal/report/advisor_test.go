package report

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

const advisorPrefix = "/subscriptions/" + sub + "/providers/Microsoft.Advisor/recommendations/"
const costType = "22222222-2222-2222-2222-222222222222"
const securityType = "11111111-1111-1111-1111-111111111111"
const performanceType = "33333333-3333-3333-3333-333333333333"

var advisorStart = time.Date(2026, 11, 18, 8, 0, 0, 0, time.UTC)

func advisorReportRun() health.Run {
	r := health.NewRun("Synthetic subscription", "test")
	r.ID, r.Revision = "synthetic-advisor-run", 18
	r.StartedAt = advisorStart
	r.UpdatedAt = advisorStart.Add(10 * time.Minute)
	r.EndedAt = &r.UpdatedAt
	r.Outcome = "completed"
	r.Scope = &health.Scope{SubscriptionID: sub, Name: "Synthetic subscription", TenantID: "44444444-4444-4444-4444-444444444444", Cloud: "AzureCloud"}
	prepared := advisorStart.Add(time.Minute)
	serviceQuery, monitorQuery, advisorQuery := health.ServiceHealthQuery(advisorStart), health.MonitorQuery(advisorStart), health.AdvisorQuery(advisorStart)
	serviceQuery.RequestedAt, monitorQuery.RequestedAt, advisorQuery.RequestedAt = &advisorStart, &advisorStart, &prepared
	r.Sources["inventory"] = health.Source{Status: "completed"}
	r.Sources["resource_health"] = health.Source{Status: "completed"}
	r.Sources["service_health"] = health.Source{Status: "completed", Query: &serviceQuery}
	r.Sources["azure_monitor"] = health.Source{Status: "completed", Query: &monitorQuery}
	r.Sources["advisor"] = health.Source{Status: "completed", Query: &advisorQuery}
	r.NormalizePage("inventory", health.Page{CollectedAt: advisorStart, Resources: []health.ResourceInput{{ID: resource + "vm", Name: "vm", Type: "Microsoft.Compute/virtualMachines", Location: "westeurope"}}})
	r.NormalizePage("resource_health", health.Page{CollectedAt: advisorStart, Health: []health.HealthInput{{StatusID: resource + "vm/providers/Microsoft.ResourceHealth/availabilityStatuses/current", PropertiesPresent: true, State: ptr("Degraded")}}})
	r.NormalizePage("service_health", health.Page{CollectedAt: advisorStart})
	r.NormalizePage("azure_monitor", health.Page{CollectedAt: advisorStart})
	return r
}

func advisorExample() health.Run {
	r := advisorReportRun()
	evaluated, updated := advisorStart.Add(-time.Hour), advisorStart.Add(-24*time.Hour)
	cost := health.AdvisorInput{
		ID: resource + "vm/providers/Microsoft.Advisor/recommendations/cost-vm", TypeID: costType,
		Category: "Cost", Impact: "High", Status: "New", TargetResourceID: resource + "vm",
		LastRefreshed: &evaluated, LastUpdated: &updated,
	}
	older := health.AdvisorInput{
		ID: advisorPrefix + "automation", TypeID: securityType, Category: "Security", Impact: "High",
		Status: "New", LastUpdated: &updated,
	}
	r.NormalizePage("advisor", health.Page{CollectedAt: advisorStart.Add(2 * time.Minute), Advisor: []health.AdvisorInput{
		cost, older,
		{ID: advisorPrefix + "cost-other", TypeID: costType, Category: "Cost", Impact: "Medium", Status: "New"},
		{ID: advisorPrefix + "performance", TypeID: performanceType, Category: "Performance", Impact: "Medium", Status: "Postponed", LastRefreshed: &evaluated},
		{ID: advisorPrefix + "uncategorized", Status: "New"},
		{ID: advisorPrefix + "retired", Category: "OperationalExcellence", Impact: "Low", Status: "Completed"},
		{ID: advisorPrefix + "unclassified"},
	}})
	newer := older
	newer.Status, newer.LastUpdated = "Dismissed", &evaluated
	r.NormalizePage("advisor", health.Page{CollectedAt: advisorStart.Add(3 * time.Minute), Advisor: []health.AdvisorInput{cost, newer}})
	return r
}

func advisorSection(text string) string {
	_, section, _ := strings.Cut(text, "## Azure Advisor recommendations\n")
	section, _, _ = strings.Cut(section, "\n## ")
	return section
}

func TestAdvisorGroupsOrderAndReferences(t *testing.T) {
	r := advisorReportRun()
	inputs := []health.AdvisorInput{
		{ID: advisorPrefix + "security", TypeID: securityType, Status: "New", Category: "Security", Impact: "High"},
		{ID: advisorPrefix + "performance", TypeID: performanceType, Status: "New", Category: "Performance", Impact: "Medium"},
		{ID: advisorPrefix + "untyped-a", Status: "New", Impact: "Low"},
		{ID: advisorPrefix + "untyped-b", Status: "New"},
	}
	for _, status := range []string{"Completed", "Dismissed", "Postponed", "unknown"} {
		inputs = append(inputs, health.AdvisorInput{ID: advisorPrefix + "workflow-" + status, TypeID: costType, Status: status, Category: "Cost", Impact: "High"})
	}
	for i := range 30 {
		id := fmt.Sprintf("cost-%02d", i)
		evaluated := advisorStart.Add(-time.Hour)
		inputs = append(inputs, health.AdvisorInput{
			ID: advisorPrefix + id, TypeID: costType, Category: "Cost", Status: "New",
			Impact: []string{"", "Low", "Medium", "High"}[i%4], TargetResourceID: resource + id, LastRefreshed: &evaluated,
		})
	}
	duplicate := inputs[len(inputs)-1]
	later := advisorStart.Add(-time.Minute)
	refreshed := duplicate
	refreshed.LastRefreshed = &later
	inputs = append(inputs, duplicate, refreshed)
	r.NormalizePage("advisor", health.Page{CollectedAt: advisorStart.Add(2 * time.Minute), Advisor: inputs})
	before, _ := json.Marshal(r)
	b, err := Markdown(r)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(r)
	if !bytes.Equal(before, after) {
		t.Fatal("rendering mutated the snapshot")
	}
	slices.Reverse(r.AdvisorInstances)
	slices.Reverse(r.Evidence)
	slices.Reverse(r.Findings)
	reversed, err := Markdown(r)
	if err != nil || !bytes.Equal(b, reversed) {
		t.Fatal("report changed when snapshot rows were reversed")
	}

	text := html.UnescapeString(string(b))
	section := advisorSection(text)
	newRows, otherRows, _ := strings.Cut(section, "### Other recommendation statuses")
	if strings.Count(newRows, "| **Type "+costType+" (30)**") != 1 ||
		strings.Count(newRows, "| **Type unknown (1)**") != 2 ||
		strings.Count(newRows, "/recommendations/cost-") != 30 {
		t.Fatal("type grouping lost distinct instances")
	}
	ordered := func(section string, parts ...string) {
		t.Helper()
		for _, part := range parts {
			_, rest, found := strings.Cut(section, part)
			if !found {
				t.Fatalf("missing or out of order: %s", part)
			}
			section = rest
		}
	}
	ordered(newRows, "### New recommendations", securityType+" (1)", costType+" (30)", performanceType+" (1)", "/recommendations/untyped-a", "/recommendations/untyped-b")
	wantCostOrder := []string{}
	for _, offset := range []int{3, 2, 1, 0} {
		for i := offset; i < 30; i += 4 {
			wantCostOrder = append(wantCostOrder, fmt.Sprintf("/recommendations/cost-%02d |", i))
		}
	}
	ordered(newRows, wantCostOrder...)
	ordered(otherRows, "/recommendations/workflow-postponed", " | Postponed |", "/recommendations/workflow-dismissed", " | Dismissed |", "/recommendations/workflow-completed", " | Completed |", "/recommendations/workflow-unknown", " | unknown |")
	byID := make(map[string]string)
	for _, line := range strings.Split(section, "\n") {
		if strings.HasPrefix(line, "| "+health.CanonicalID(advisorPrefix)) {
			cols := strings.Split(line, " | ")
			byID[strings.TrimPrefix(cols[0], "| ")] = line
		}
	}
	for _, row := range r.AdvisorInstances {
		line := byID[row.EntityID]
		if line == "" || len(row.EvidenceIDs) == 0 {
			t.Fatalf("missing row/evidence: %s", row.EntityID)
		}
		for _, id := range row.EvidenceIDs {
			if !strings.Contains(line, "["+id+"](run.json)") {
				t.Fatalf("row %s is missing evidence link %s", row.EntityID, id)
			}
		}
		if row.Status == "New" && (row.FindingID == "" || !strings.Contains(line, row.FindingID)) {
			t.Fatalf("missing New finding ID for %s", row.EntityID)
		}
	}
	if !strings.Contains(section, "Last evaluated (UTC)") || !strings.Contains(byID[health.CanonicalID(advisorPrefix)+"untyped-b"], "| **Resource unknown** | unknown |") ||
		!strings.Contains(byID[health.CanonicalID(duplicate.ID)], later.Format(time.RFC3339)) ||
		strings.Count(byID[health.CanonicalID(duplicate.ID)], "](run.json)") != 2 {
		t.Fatal("field labels, unknowns or refreshed evidence lost")
	}
	_, resourceSection, _ := strings.Cut(text, "## Resource Health findings")
	resourceSection, _, _ = strings.Cut(resourceSection, "## Resource Health issue states")
	legacy := r.Findings[slices.IndexFunc(r.Findings, func(f health.Finding) bool { return f.Source == "" })]
	if !strings.Contains(resourceSection, legacy.ID) || strings.Contains(resourceSection, "advisor_recommendation") ||
		strings.Contains(resourceSection, "Azure Advisor New recommendation") ||
		strings.Contains(strings.Split(text, "## Azure Advisor recommendations")[0], "/recommendations/") {
		t.Fatal("findings leaked across sources or legacy Resource Health finding was hidden")
	}
}

func TestAdvisorRejectsInconsistentReferences(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*health.Run, *health.AdvisorInstance, *health.Finding)
	}{
		{"missing evidence", func(_ *health.Run, row *health.AdvisorInstance, _ *health.Finding) {
			row.EvidenceIDs = []string{"ev-missing"}
		}},
		{"Resource Health evidence", func(r *health.Run, row *health.AdvisorInstance, _ *health.Finding) {
			i := slices.IndexFunc(r.Evidence, func(e health.Evidence) bool { return e.Source == "resource_health" })
			row.EvidenceIDs = []string{r.Evidence[i].ID}
		}},
		{"different Advisor instance", func(r *health.Run, row *health.AdvisorInstance, _ *health.Finding) {
			i := slices.IndexFunc(r.Evidence, func(e health.Evidence) bool {
				return e.Source == "advisor" && !strings.EqualFold(e.EntityID, row.EntityID)
			})
			row.EvidenceIDs = []string{r.Evidence[i].ID}
		}},
		{"different subscription", func(r *health.Run, row *health.AdvisorInstance, _ *health.Finding) {
			i := slices.IndexFunc(r.Evidence, func(e health.Evidence) bool { return e.ID == row.EvidenceIDs[0] })
			r.Evidence[i].SubscriptionID = "55555555-5555-5555-5555-555555555555"
		}},
		{"missing finding", func(_ *health.Run, row *health.AdvisorInstance, _ *health.Finding) {
			row.FindingID = "finding-missing"
		}},
		{"different Advisor finding", func(r *health.Run, row *health.AdvisorInstance, _ *health.Finding) {
			i := slices.IndexFunc(r.Findings, func(f health.Finding) bool { return f.Source == "advisor" && f.ID != row.FindingID })
			row.FindingID = r.Findings[i].ID
		}},
		{"finding with unrelated evidence", func(r *health.Run, _ *health.AdvisorInstance, f *health.Finding) {
			i := slices.IndexFunc(r.Evidence, func(e health.Evidence) bool { return e.Source == "inventory" })
			f.EvidenceIDs = []string{r.Evidence[i].ID}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := advisorExample()
			i := slices.IndexFunc(r.AdvisorInstances, func(row health.AdvisorInstance) bool { return row.Status == "New" })
			row := &r.AdvisorInstances[i]
			j := slices.IndexFunc(r.Findings, func(f health.Finding) bool { return f.ID == row.FindingID })
			tc.edit(&r, row, &r.Findings[j])
			if _, err := Markdown(r); err == nil {
				t.Fatal("published an Advisor row with an unsupported reference")
			}
		})
	}
}

func TestSyntheticAdvisorReportExample(t *testing.T) {
	jsonBytes, err := os.ReadFile("testdata/advisor/run.json")
	if err != nil {
		t.Fatal(err)
	}
	var saved health.Run
	if err := json.Unmarshal(jsonBytes, &saved); err != nil {
		t.Fatal(err)
	}
	expectedJSON, err := json.MarshalIndent(advisorExample(), "", "  ")
	if err != nil || !bytes.Equal(jsonBytes, append(expectedJSON, '\n')) {
		t.Fatal("schema-3 fixture snapshot differs from synthetic example")
	}
	want, err := os.ReadFile("testdata/advisor/report.md")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Markdown(saved)
	if err != nil || !bytes.Equal(b, want) {
		t.Fatal("schema-3 report fixture differs from its sibling snapshot")
	}
	text := html.UnescapeString(string(b))
	for _, fact := range []string{"Recommendations.List", "2026-10-01", "State changed (UTC)", "Last evaluated (UTC)", "Collected (UTC)", "Attribution limitation: unknown_resource_attribution"} {
		if !strings.Contains(text, fact) {
			t.Errorf("Advisor evidence index missing %s", fact)
		}
	}
	for _, e := range saved.Evidence {
		if e.Advisor != nil && e.Advisor.Status == "New" && e.EntityID == advisorPrefix+"automation" {
			daily, _, _ := strings.Cut(advisorSection(text), "### Other recommendation statuses")
			if strings.Contains(daily, health.CanonicalID(e.EntityID)) || !strings.Contains(text, "### "+e.ID+"\n") {
				t.Fatal("superseded New observation should be available in the evidence index")
			}
			return
		}
	}
	t.Fatal("fixture missing superseded New observation")
}

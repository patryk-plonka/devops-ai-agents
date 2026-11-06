// Package report renders normalized, committed evidence as literal Markdown.
package report

import (
	"errors"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

// Markdown never fetches or infers evidence. Invalid references fail publication
// rather than allowing an unsupported finding to appear in a saved report.
func Markdown(r health.Run) ([]byte, error) {
	if r.ID == "" || r.Revision < 1 || r.EndedAt == nil {
		return nil, errors.New("report requires a finalized snapshot")
	}
	switch r.Outcome {
	case "completed", "partial", "failed", "cancelled":
	default:
		return nil, errors.New("invalid final outcome")
	}
	evidence := map[string]bool{}
	for _, e := range r.Evidence {
		if e.ID == "" || evidence[e.ID] {
			return nil, errors.New("invalid evidence identity")
		}
		evidence[e.ID] = true
	}
	check := func(ids []string) bool {
		for _, id := range ids {
			if !evidence[id] {
				return false
			}
		}
		return true
	}
	for _, f := range r.Findings {
		if len(f.EvidenceIDs) == 0 || !check(f.EvidenceIDs) {
			return nil, errors.New("unsupported finding")
		}
	}
	for _, o := range r.Observations {
		if !check(o.EvidenceIDs) {
			return nil, errors.New("unresolved observation evidence")
		}
	}
	for _, v := range r.Resources {
		if !check(v.EvidenceIDs) {
			return nil, errors.New("unresolved inventory evidence")
		}
	}
	for _, g := range r.Gaps {
		if !check(g.EvidenceIDs) {
			return nil, errors.New("unresolved gap evidence")
		}
	}
	var b strings.Builder
	b.WriteString("# Subscription Resource Health report\n\n**INCOMPLETE coverage.** Advisor, Service Health, and Azure Monitor are not checked in S-01. No resource is classified overall OK.\n\n")
	fmt.Fprintf(&b, "Run ID: %s\n\nRevision: %d\n\nExecution outcome: **%s**\n\n", literal(r.ID), r.Revision, literal(r.Outcome))
	fmt.Fprintf(&b, "Started (UTC): %s; updated (UTC): %s; ended (UTC): %s.\n\nTool version: %s; schema version: %d; filtered values: %d.\n\n", stamp(&r.StartedAt), stamp(&r.UpdatedAt), stamp(r.EndedAt), literal(r.ToolVersion), r.SchemaVersion, r.Redactions)
	fmt.Fprintf(&b, "Requested scope: %s.\n\n", literal(r.RequestedScope))
	if r.Scope == nil {
		b.WriteString("Resolved scope: unresolved. Scope validation did not establish access.\n\n")
	} else {
		fmt.Fprintf(&b, "Subscription: %s; ID: %s; tenant: %s; cloud: %s.\n\n", literal(r.Scope.Name), literal(r.Scope.SubscriptionID), literal(r.Scope.TenantID), literal(r.Scope.Cloud))
	}
	b.WriteString("Evidence snapshot: [run.json](run.json). Match its run ID and revision to this report. If revisions differ, this report is stale.\n\n## Source coverage\n\n| Source | Status | Pages | Rejected records | Reason |\n| --- | --- | --- | --- | --- |\n")
	names := make([]string, 0, len(r.Sources))
	for name := range r.Sources {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		s := r.Sources[name]
		fmt.Fprintf(&b, "| %s | %s | %d | %d | %s |\n", literal(name), literal(s.Status), s.Pages, s.RejectedRecords, literal(s.Reason))
	}
	b.WriteString("\nCollector completion describes enumeration by that API. Inventory may omit child resources; an Available observation provides Resource Health evidence only. Failed checks and missing evidence do not establish resource faults.\n\n## Observed issues and events\n\nFindings retain source observations, including conflicting or superseded observations. Consult the current-state tables and timestamps before interpreting them as current faults.\n\n")
	findings := slices.Clone(r.Findings)
	sort.Slice(findings, func(i, j int) bool {
		a, c := findings[i], findings[j]
		if a.Category != c.Category {
			return a.Category < c.Category
		}
		if health.CanonicalID(a.ResourceID) != health.CanonicalID(c.ResourceID) {
			return health.CanonicalID(a.ResourceID) < health.CanonicalID(c.ResourceID)
		}
		return a.ID < c.ID
	})
	if len(findings) == 0 {
		b.WriteString("No issue/event findings in the collected evidence. This does not establish complete coverage.\n\n")
	} else {
		b.WriteString("| Category | Resource | Source observation | Finding ID | Evidence |\n| --- | --- | --- | --- | --- |\n")
		for _, f := range findings {
			fmt.Fprintf(&b, "| %s | %s | %s | %s | %s |\n", literal(f.Category), literal(f.ResourceID), literal(f.Title), literal(f.ID), refs(f.EvidenceIDs))
		}
		b.WriteByte('\n')
	}
	observations := slices.Clone(r.Observations)
	sort.Slice(observations, func(i, j int) bool {
		return health.CanonicalID(observations[i].ResourceID) < health.CanonicalID(observations[j].ResourceID)
	})
	for _, group := range []struct {
		title   string
		matches func(string) bool
	}{
		{"Resource Health issue states", func(s string) bool { return s == "Unavailable" || s == "Degraded" }},
		{"Resource Health Available observations", func(s string) bool { return s == "Available" }},
		{"Unknown and not-checked resources", func(s string) bool { return s != "Available" && s != "Unavailable" && s != "Degraded" }},
	} {
		fmt.Fprintf(&b, "## %s\n\n| Resource | Current Resource Health state | In observed inventory | Evidence |\n| --- | --- | --- | --- |\n", group.title)
		count := 0
		for _, o := range observations {
			if group.matches(o.State) {
				count++
				fmt.Fprintf(&b, "| %s | %s | %t | %s |\n", literal(o.ResourceID), literal(o.State), o.InInventory, refs(o.EvidenceIDs))
			}
		}
		if count == 0 {
			b.WriteString("| None observed | — | — | — |\n")
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "## Observed inventory (%d resources)\n\nAll observed resources are listed below. Conflicting inventory facts remain in the evidence index.\n\n| Resource ID | Name | Type | Resource group | Location | Tags | Evidence |\n| --- | --- | --- | --- | --- | --- | --- |\n", len(r.Resources))
	resources := slices.Clone(r.Resources)
	sort.Slice(resources, func(i, j int) bool { return health.CanonicalID(resources[i].ID) < health.CanonicalID(resources[j].ID) })
	for _, v := range resources {
		fmt.Fprintf(&b, "| %s | %s | %s | %s | %s | %s | %s |\n", literal(v.ID), literal(v.Name), literal(v.Type), literal(v.ResourceGroup), literal(v.Location), tags(v.Tags), refs(v.EvidenceIDs))
	}
	if len(resources) == 0 {
		b.WriteString("| None observed | — | — | — | — | — | — |\n")
	}
	b.WriteString("\n## Coverage gaps and failed checks\n\n| Source | Resource | Reason | Evidence |\n| --- | --- | --- | --- |\n")
	gaps := slices.Clone(r.Gaps)
	sort.Slice(gaps, func(i, j int) bool {
		a, c := gaps[i], gaps[j]
		return a.Source+"\x00"+health.CanonicalID(a.ResourceID)+"\x00"+a.Reason < c.Source+"\x00"+health.CanonicalID(c.ResourceID)+"\x00"+c.Reason
	})
	for _, g := range gaps {
		fmt.Fprintf(&b, "| %s | %s | %s | %s |\n", literal(g.Source), literal(g.ResourceID), literal(g.Reason), refs(g.EvidenceIDs))
	}
	b.WriteString("\n## Execution diagnostics\n\n")
	diagnostics := slices.Clone(r.Diagnostics)
	sort.Strings(diagnostics)
	if len(diagnostics) == 0 {
		b.WriteString("No execution diagnostics. Overall coverage remains incomplete.\n\n")
	} else {
		for _, d := range diagnostics {
			fmt.Fprintf(&b, "- %s\n", literal(d))
		}
		b.WriteByte('\n')
	}
	fmt.Fprintf(&b, "## Evidence index (%d records)\n\nEach reference identifies a record in the sibling JSON snapshot. Collection time and source time are distinct. Historical statuses cannot establish current health.\n\n", len(r.Evidence))
	records := slices.Clone(r.Evidence)
	sort.Slice(records, func(i, j int) bool {
		a, c := records[i], records[j]
		return a.Source+"\x00"+health.CanonicalID(a.ResourceID)+"\x00"+a.ID < c.Source+"\x00"+health.CanonicalID(c.ResourceID)+"\x00"+c.ID
	})
	for _, e := range records {
		fmt.Fprintf(&b, "### %s\n\n- Record: %s\n- Source: %s; operation: %s; API version: %s.\n- Resource: %s\n- Kind: %s; state: %s; status role: %s.\n- Status ID: %s\n- Event ID: %s; additional event IDs: %s.\n- Collected (UTC): %s; source reported (UTC): %s; source occurred (UTC): %s.\n", literal(e.ID), refs([]string{e.ID}), literal(e.Source), literal(e.Operation), literal(e.APIVersion), literal(e.ResourceID), literal(e.Kind), literal(e.State), literal(e.StatusRole), literal(e.StatusID), literal(e.EventID), literal(strings.Join(e.EventIDs, ", ")), stamp(&e.CollectedAt), stamp(e.ReportedAt), stamp(e.OccurredAt))
		if e.Resource != nil {
			v := e.Resource
			fmt.Fprintf(&b, "- Inventory facts: ID: %s; name: %s; type: %s; group: %s; location: %s; tags: %s.\n", literal(v.ID), literal(v.Name), literal(v.Type), literal(v.ResourceGroup), literal(v.Location), tags(v.Tags))
		}
		b.WriteByte('\n')
	}
	if len(records) == 0 {
		b.WriteString("No evidence collected. Consult source coverage and execution diagnostics.\n")
	}
	return []byte(b.String()), nil
}

// Character references render punctuation literally without creating Markdown
// links, HTML, table delimiters, headings, or autolinks from untrusted strings.
func literal(value string) string {
	value, _ = health.SafeText(value)
	if value == "" {
		return "—"
	}
	var b strings.Builder
	for _, r := range value {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == ' ' || r == '-' || r == '/' {
			b.WriteRune(r)
		} else {
			b.WriteString("&#")
			b.WriteString(strconv.Itoa(int(r)))
			b.WriteByte(';')
		}
	}
	return b.String()
}
func refs(ids []string) string {
	if len(ids) == 0 {
		return "—"
	}
	ids = slices.Clone(ids)
	sort.Strings(ids)
	values := make([]string, 0, len(ids))
	for _, id := range ids {
		values = append(values, "["+literal(id)+"](run.json)")
	}
	return strings.Join(values, ", ")
}
func tags(values map[string]string) string {
	input := make(map[string]*string, len(values))
	for k, v := range values {
		input[k] = &v
	}
	safe, _ := health.SafeTags(input)
	keys := make([]string, 0, len(safe))
	for k := range safe {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, literal(k)+"="+literal(safe[k]))
	}
	if len(parts) == 0 {
		return "—"
	}
	return strings.Join(parts, "; ")
}
func stamp(t *time.Time) string {
	if t == nil || t.IsZero() {
		return "not recorded"
	}
	return t.UTC().Format(time.RFC3339Nano)
}

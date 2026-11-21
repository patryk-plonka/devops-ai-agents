package report

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

func resourceAttribution(id string) string {
	if id == "" {
		return "Unknown; no affected resource established"
	}
	return literal(id) + " (reported target only)"
}

func optionalBool(v *bool) string {
	if v == nil {
		return "not recorded"
	}
	if *v {
		return "true"
	}
	return "false"
}

func writeOperationalSections(b *strings.Builder, r health.Run) {
	for _, source := range []string{"service_health", "azure_monitor"} {
		title := "Service Health"
		if source == "azure_monitor" {
			title = "Azure Monitor"
		}
		s := r.Sources[source]
		fmt.Fprintf(b, "## %s\n\nCollection status: %s.\n\n", title, literal(s.Status))
		if q := s.Query; q != nil {
			fmt.Fprintf(b, "Run reference (UTC): %s.\n", stamp(&q.ReferenceAt))
			if s.Status == "not_checked" {
				b.WriteString("Collection did not start.\n\n")
			} else {
				fmt.Fprintf(b, "Query prepared (UTC): %s.\n\n", stamp(q.RequestedAt))
			}
			if source == "service_health" {
				fmt.Fprintf(b, "Rolling 30-day lower cutoff (UTC): %s. Configured API queryStartTime: %s. Events are filtered locally by lastUpdateTime at the exact cutoff. Older unchanged active events may be absent.\n\n", stamp(q.Cutoff), literal(q.QueryStartTime))
			} else {
				fmt.Fprintf(b, "Configured API timeRange: %s. The API applies this relative window.\n\n", literal(q.TimeRange))
			}
		} else {
			b.WriteString("No query policy recorded; no time coverage established.\n\n")
		}
		var first, last *time.Time
		records := []health.Evidence{}
		byID := map[string]health.Evidence{}
		for _, e := range r.Evidence {
			if e.Source != source {
				continue
			}
			records = append(records, e)
			byID[e.ID] = e
			if !e.CollectedAt.IsZero() {
				t := e.CollectedAt
				if first == nil || t.Before(*first) {
					first = &t
				}
				latest := t
				if r.SchemaVersion >= health.AssessmentSchemaVersion {
					latest = e.LatestCollectionTime()
				}
				if last == nil || latest.After(*last) {
					last = &latest
				}
			}
		}
		if r.SchemaVersion >= health.AssessmentSchemaVersion {
			fmt.Fprintf(b, "Evidence collected (UTC): %s to %s. See accepted-page times under Query provenance for lists with no retained rows.\n\n", stamp(first), stamp(last))
		} else {
			fmt.Fprintf(b, "Evidence collected (UTC): %s to %s. Pages are collected sequentially; this is not an atomic snapshot. Empty completed lists do not establish OK.\n\n", stamp(first), stamp(last))
		}
		if source == "service_health" {
			b.WriteString("ServiceIssue identifies an incident notice; PlannedMaintenance, HealthAdvisory and SecurityAdvisory retain their distinct categories. Future notices do not establish an ongoing outage. The listing does not enumerate affected resources. Service/region attribution and uncertainty appear in the evidence index. Sensitive details are not retrieved; available metadata and detail limitations remain visible.\n\n")
		} else {
			b.WriteString("Fired is the source condition, not proof of a continuing fault. Stateless alerts can remain Fired; instances are retained for 30 days, and stateful conditions can outlive that retention. Workflow response and notification suppression do not change the condition. A workspace target does not identify every resource referenced by a log query.\n\n")
		}
		findings := []health.Finding{}
		for _, f := range r.Findings {
			if f.Source == source {
				findings = append(findings, f)
			}
		}
		sort.Slice(findings, func(i, j int) bool {
			return health.CanonicalID(findings[i].EntityID)+"\x00"+findings[i].ID < health.CanonicalID(findings[j].EntityID)+"\x00"+findings[j].ID
		})
		b.WriteString("### Active summaries\n\n")
		active := map[string]bool{}
		if len(findings) == 0 {
			b.WriteString("No active summaries supported by the collected evidence. Consult saved observations and coverage gaps.\n\n")
		}
		for _, f := range findings {
			if r.SchemaVersion >= health.AssessmentSchemaVersion && source == "service_health" {
				fmt.Fprintf(b, "<a id=\"%s\"></a>\n\n", incidentAnchor(f.ID))
			}
			fmt.Fprintf(b, "- %s\n  - Source entity: %s; selected subscription: %s.\n  - Finding ID: %s; evidence: %s.\n", literal(f.Title), literal(f.EntityID), literal(f.SubscriptionID), literal(f.ID), refs(f.EvidenceIDs))
			ids := slices.Clone(f.EvidenceIDs)
			sort.Strings(ids)
			for _, id := range ids {
				active[id] = true
				e, ok := byID[id]
				if !ok {
					continue
				} // Markdown validates references before rendering.
				if e.ServiceHealth != nil {
					p := e.ServiceHealth
					fmt.Fprintf(b, "  - Category: %s; status: %s; impact starts (UTC): %s; last updated (UTC): %s.\n", literal(p.Category), literal(p.Status), stamp(p.ImpactStartTime), stamp(e.ReportedAt))
					b.WriteString("  - Resource impact: unknown; not enumerated by this listing.\n")
				}
				if e.Monitor != nil {
					p := e.Monitor
					fmt.Fprintf(b, "  - Signal: %s; condition: %s; workflow response: %s; severity: %s; notification suppressed: %s.\n  - Target: %s; started (UTC): %s; last modified (UTC): %s.\n", literal(p.SignalType), literal(p.Condition), literal(p.WorkflowState), literal(p.Severity), optionalBool(p.IsSuppressed), resourceAttribution(e.ResourceID), stamp(e.OccurredAt), stamp(e.ReportedAt))
				}
			}
			b.WriteByte('\n')
		}
		b.WriteString("### Other saved observations\n\nInactive, excluded, unclassified, superseded or conflicting evidence is retained here. A saved source state alone does not establish an active summary; inspect its evidence and coverage gaps.\n\n| Source entity | Disposition | Source state | Source updated (UTC) | Evidence |\n| --- | --- | --- | --- | --- |\n")
		sort.Slice(records, func(i, j int) bool {
			return health.CanonicalID(records[i].EntityID)+"\x00"+records[i].ID < health.CanonicalID(records[j].EntityID)+"\x00"+records[j].ID
		})
		count := 0
		for _, e := range records {
			if active[e.ID] {
				continue
			}
			count++
			fmt.Fprintf(b, "| %s | %s | %s | %s | %s |\n", literal(e.EntityID), literal(e.Disposition), literal(e.State), stamp(e.ReportedAt), refs([]string{e.ID}))
		}
		if count == 0 {
			b.WriteString("| None collected | — | — | — | — |\n")
		}
		b.WriteByte('\n')
	}
}

func writeOperationalFacts(b *strings.Builder, e health.Evidence) {
	if p := e.ServiceHealth; p != nil {
		fmt.Fprintf(b, "- Service Health category: %s; source status: %s; level: %s; event source: %s; sensitive: %s.\n- Impact starts (UTC): %s; mitigation (UTC): %s.\n", literal(p.Category), literal(p.Status), literal(p.Level), literal(p.EventSource), optionalBool(p.Sensitive), stamp(p.ImpactStartTime), stamp(p.ImpactMitigationTime))
		impacts := slices.Clone(p.Impact)
		sort.Slice(impacts, func(i, j int) bool { return impactKey(impacts[i]) < impactKey(impacts[j]) })
		if len(impacts) == 0 {
			b.WriteString("- Service/region attribution is unknown.\n")
		}
		for _, impact := range impacts {
			fmt.Fprintf(b, "- Service: %s.\n", literal(impact.Service))
			regions := slices.Clone(impact.Regions)
			sort.Slice(regions, func(i, j int) bool { return regionKey(regions[i]) < regionKey(regions[j]) })
			if len(regions) == 0 {
				b.WriteString("  - Region attribution: unknown; no accepted region facts.\n")
			}
			for _, region := range regions {
				fmt.Fprintf(b, "  - Region: %s; status: %s; attribution: %s; last updated (UTC): %s.\n", literal(region.Name), literal(region.Status), literal(region.Attribution), stamp(region.LastUpdateTime))
			}
		}
	}
	if p := e.Monitor; p != nil {
		fmt.Fprintf(b, "- Monitor signal: %s; source condition: %s; workflow response: %s; severity: %s; notification suppressed: %s.\n- Target attribution: %s; monitor service last modified (UTC): %s.\n", literal(p.SignalType), literal(p.Condition), literal(p.WorkflowState), literal(p.Severity), optionalBool(p.IsSuppressed), literal(p.Attribution), stamp(p.LastModifiedTimeByMonitorService))
	}
	limits := slices.Clone(e.Limitations)
	sort.Strings(limits)
	if len(limits) > 0 {
		fmt.Fprintf(b, "- Limitations: %s.\n", literal(strings.Join(limits, ", ")))
	}
}

func regionKey(r health.ServiceRegionFacts) string {
	return r.Name + "\x00" + r.Status + "\x00" + r.Attribution + "\x00" + stamp(r.LastUpdateTime)
}
func impactKey(i health.ServiceImpactFacts) string {
	keys := make([]string, 0, len(i.Regions))
	for _, r := range i.Regions {
		keys = append(keys, regionKey(r))
	}
	sort.Strings(keys)
	return i.Service + "\x00" + strings.Join(keys, "\x00")
}

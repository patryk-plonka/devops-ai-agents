package report

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

func validateAdvisorReferences(r health.Run, evidence map[string]*health.Evidence) error {
	if len(r.AdvisorInstances) == 0 {
		return nil
	}
	if r.Scope == nil || r.Scope.SubscriptionID == "" {
		return errors.New("missing Advisor subscription scope")
	}
	subscription := r.Scope.SubscriptionID
	findings := make(map[string]*health.Finding)
	for i := range r.Findings {
		f := &r.Findings[i]
		if f.Source == "advisor" && f.ID != "" {
			findings[f.ID] = f
		}
	}
	for _, row := range r.AdvisorInstances {
		check := func(ids []string) bool {
			for _, id := range ids {
				e := evidence[id]
				if e == nil || e.Source != "advisor" || e.Advisor == nil ||
					!strings.EqualFold(e.EntityID, row.EntityID) || !strings.EqualFold(e.SubscriptionID, subscription) {
					return false
				}
			}
			return len(ids) > 0
		}
		if !check(row.EvidenceIDs) {
			return errors.New("unresolved Advisor evidence")
		}
		if row.Status == "New" {
			f := findings[row.FindingID]
			if f == nil || !strings.EqualFold(f.EntityID, row.EntityID) ||
				!strings.EqualFold(f.SubscriptionID, subscription) || !check(f.EvidenceIDs) {
				return errors.New("unresolved Advisor finding")
			}
		}
	}
	return nil
}

func writeAdvisorSection(b *strings.Builder, r health.Run) {
	s := r.Sources["advisor"]
	b.WriteString("## Azure Advisor recommendations\n\n")
	fmt.Fprintf(b, "Collection status: %s.\n\n", literal(s.Status))
	b.WriteString("Advisor returns cached suggestions for review. A recommendation alone does not establish a resource fault.\n\n")
	if q := s.Query; q != nil {
		scope := "selected subscription"
		if r.Scope != nil {
			scope += " " + literal(r.Scope.SubscriptionID)
		}
		fmt.Fprintf(b, "Cached list scope: %s. Operation: %s. API version: %s. Query prepared (UTC): %s.\n\n", scope, literal(q.Operation), literal(q.APIVersion), stamp(q.RequestedAt))
	} else {
		b.WriteString("No cached list policy was recorded.\n\n")
	}
	if s.Pages == 0 {
		b.WriteString("No Advisor pages were accepted.\n\n")
	} else {
		fmt.Fprintf(b, "First accepted page (UTC): %s. Last accepted page (UTC): %s. Pages are read sequentially.\n\n", stamp(s.FirstPageAt), stamp(s.LastPageAt))
	}
	switch s.Status {
	case "completed":
		if s.Pages > 0 && len(r.AdvisorInstances) == 0 {
			b.WriteString("The completed cached list contained no recommendations.\n\n")
		}
	case "partial":
		fmt.Fprintf(b, "The cached list could not be fully checked (%s). Coverage is limited to accepted pages.\n\n", literal(s.Reason))
	case "failed":
		fmt.Fprintf(b, "The cached list could not be read (%s).\n\n", literal(s.Reason))
	case "not_checked":
		b.WriteString("Advisor collection was not attempted.\n\n")
	}
	if len(r.AdvisorInstances) == 0 {
		return
	}
	byStatus := make(map[string][]health.AdvisorInstance)
	for _, row := range r.AdvisorInstances {
		byStatus[row.Status] = append(byStatus[row.Status], row)
	}
	b.WriteString("### New recommendations\n\n")
	if rows := byStatus["New"]; len(rows) > 0 {
		b.WriteString("| Type group / instance ID | Category | Impact | Assessed resource | Last evaluated (UTC) | Finding ID | Evidence |\n| --- | --- | --- | --- | --- | --- | --- |\n")
		writeAdvisorGroups(b, rows, "New")
		b.WriteByte('\n')
	} else {
		b.WriteString("No New recommendations among the accepted observations.\n\n")
	}
	b.WriteString("### Other recommendation statuses\n\n")
	if len(byStatus) == 1 && len(byStatus["New"]) > 0 {
		b.WriteString("No other recommendation statuses among the accepted observations.\n\n")
		return
	}
	b.WriteString("| Type group / instance ID | Category | Impact | Assessed resource | Last evaluated (UTC) | Status | Evidence |\n| --- | --- | --- | --- | --- | --- | --- |\n")
	for _, status := range []string{"Postponed", "Dismissed", "Completed", "unknown"} {
		writeAdvisorGroups(b, byStatus[status], status)
	}
	b.WriteByte('\n')
}

type advisorGroup struct {
	id, typeID string
	rank       int
	rows       []health.AdvisorInstance
}

func writeAdvisorGroups(b *strings.Builder, rows []health.AdvisorInstance, status string) {
	groups := make(map[string]*advisorGroup)
	for _, row := range rows {
		// Normalization stores canonical type and instance IDs on review rows.
		id := row.TypeID
		if id == "" {
			id = row.EntityID
		}
		group := groups[id]
		if group == nil {
			group = &advisorGroup{id: id, typeID: row.TypeID}
			groups[id] = group
		}
		group.rows = append(group.rows, row)
		group.rank = max(group.rank, advisorImpactRank(row.Impact))
	}
	ordered := make([]*advisorGroup, 0, len(groups))
	for _, group := range groups {
		ordered = append(ordered, group)
	}
	slices.SortFunc(ordered, func(a, c *advisorGroup) int {
		if a.rank != c.rank {
			return cmp.Compare(c.rank, a.rank)
		}
		return cmp.Compare(a.id, c.id)
	})
	for _, group := range ordered {
		slices.SortFunc(group.rows, func(a, c health.AdvisorInstance) int {
			if impact := cmp.Compare(advisorImpactRank(c.Impact), advisorImpactRank(a.Impact)); impact != 0 {
				return impact
			}
			return cmp.Compare(a.EntityID, c.EntityID)
		})
		typeLabel := advisorType(group.typeID)
		statusLabel := "—"
		if status != "New" {
			statusLabel = literal(status)
		}
		fmt.Fprintf(b, "| **%s (%d)** | — | %s | — | — | %s | — |\n", typeLabel, len(group.rows), literal(group.rows[0].Impact), statusLabel)
		for _, row := range group.rows {
			state := statusLabel
			if status == "New" {
				state = literal(row.FindingID)
			}
			fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s | %s |\n", literal(row.EntityID), literal(row.Category), literal(row.Impact), advisorTarget(row.ResourceID), advisorTime(row.LastRefreshed), state, refs(row.EvidenceIDs))
		}
	}
}

func advisorImpactRank(impact string) int {
	switch impact {
	case "High":
		return 3
	case "Medium":
		return 2
	case "Low":
		return 1
	default:
		return 0
	}
}

func advisorType(id string) string {
	if id == "" {
		return "Type unknown"
	}
	return "Type " + literal(id)
}

func advisorTarget(id string) string {
	if id == "" {
		return "**Resource unknown**"
	}
	return literal(id)
}

func advisorTime(t *time.Time) string {
	if t == nil {
		return "unknown"
	}
	return stamp(t)
}

func writeAdvisorEvidence(b *strings.Builder, e health.Evidence) {
	p := e.Advisor
	fmt.Fprintf(b, "### %s\n\n- Record: %s\n- Source: %s. Operation: %s. API version: %s.\n", literal(e.ID), refs([]string{e.ID}), literal(e.Source), literal(e.Operation), literal(e.APIVersion))
	fmt.Fprintf(b, "- Advisor instance: %s. Selected subscription: %s.\n- %s. Category: %s. Impact: %s. Status: %s.\n", literal(e.EntityID), literal(e.SubscriptionID), advisorType(p.TypeID), literal(p.Category), literal(p.Impact), literal(p.Status))
	fmt.Fprintf(b, "- Assessed resource: %s.\n", advisorTarget(e.ResourceID))
	if p.AttributionLimitation != "" {
		fmt.Fprintf(b, "- Attribution limitation: %s.\n", literal(p.AttributionLimitation))
	}
	fmt.Fprintf(b, "- Last evaluated (UTC): %s. State changed (UTC): %s. Collected (UTC): %s.\n\n", advisorTime(p.LastRefreshed), advisorTime(p.LastUpdated), stamp(&e.CollectedAt))
}

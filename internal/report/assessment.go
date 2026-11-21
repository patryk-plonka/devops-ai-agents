package report

import (
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

var assessmentSourceNames = [...]string{"inventory", "resource_health", "service_health", "azure_monitor", "advisor"}

func validateAssessmentReferences(r health.Run, evidence map[string]*health.Evidence) (map[string]health.Finding, error) {
	if r.AssessmentSummary == nil || len(r.ResourceAssessments) != len(r.Resources) {
		return nil, errors.New("missing resource assessments or counts")
	}
	inventory := make(map[string]health.Resource, len(r.Resources))
	for _, v := range r.Resources {
		inventory[v.ID] = v
	}
	observations := make(map[string]health.Observation, len(r.Observations))
	for _, o := range r.Observations {
		if o.InInventory {
			observations[o.ResourceID] = o
		}
	}
	findings := make(map[string]health.Finding, len(r.Findings))
	for _, f := range r.Findings {
		if _, duplicate := findings[f.ID]; f.ID == "" || duplicate {
			return nil, errors.New("invalid finding identity")
		}
		findings[f.ID] = f
	}
	proofs := make(map[string]bool, len(assessmentSourceNames))
	for _, name := range assessmentSourceNames {
		if _, exists := r.Sources[name]; !exists {
			return nil, errors.New("missing assessment source")
		}
		proofs[name] = r.HasAssessmentSourceProof(name)
	}
	for _, g := range r.Gaps {
		if _, ok := findings[g.FindingID]; g.FindingID != "" && !ok {
			return nil, errors.New("unresolved coverage gap finding")
		}
		if r.Coverage == "complete" && g.Blocking {
			return nil, errors.New("complete coverage has a blocking gap")
		}
	}
	var counts health.AssessmentSummary
	for _, a := range r.ResourceAssessments {
		v, exists := inventory[a.ResourceID]
		if !exists || a.ResourceID == "" {
			return nil, errors.New("assessment has no unique inventory entry")
		}
		delete(inventory, a.ResourceID)
		switch a.Disposition {
		case "ok":
			if len(a.BlockingReasons) > 0 || len(a.FindingIDs) > 0 {
				return nil, errors.New("OK assessment contains a blocking reason or current finding")
			}
			counts.OK++
		case "finding":
			if len(a.BlockingReasons) > 0 || len(a.FindingIDs) == 0 {
				return nil, errors.New("invalid resource assessment with findings")
			}
			counts.Finding++
		case "unknown":
			if len(a.BlockingReasons) == 0 {
				return nil, errors.New("unknown resource assessment requires a reason")
			}
			counts.Unknown++
		default:
			return nil, errors.New("invalid resource disposition")
		}
		blocked := make(map[string]bool, len(a.BlockingReasons))
		for _, g := range a.BlockingReasons {
			if _, ok := findings[g.FindingID]; g.FindingID != "" && !ok {
				return nil, errors.New("unresolved blocking finding")
			}
			if g.Source != "execution" && !slices.Contains(assessmentSourceNames[:], g.Source) {
				return nil, errors.New("invalid blocking source")
			}
			blocked[g.Source] = true
			for _, id := range g.EvidenceIDs {
				if evidence[id] == nil {
					return nil, errors.New("unresolved blocking evidence")
				}
			}
		}
		if err := validateAssessmentChecks(a, v, observations[a.ResourceID], evidence, proofs, blocked); err != nil {
			return nil, err
		}
	}
	if err := r.ValidateAssessmentFindingReferences(); err != nil {
		return nil, err
	}
	if err := r.ValidateAssessmentBlockingReasons(); err != nil {
		return nil, err
	}
	if counts != *r.AssessmentSummary {
		return nil, errors.New("assessment counts do not match inventory")
	}
	if r.Outcome != "completed" && r.Outcome != "partial" && counts.OK+counts.Finding > 0 {
		return nil, errors.New("unsuccessful execution contains affirmative assessments")
	}
	switch r.Coverage {
	case "complete":
		if r.Outcome != "completed" || counts.Unknown > 0 || len(r.Resources) == 0 {
			return nil, errors.New("invalid complete coverage")
		}
	case "incomplete":
	default:
		return nil, errors.New("invalid coverage")
	}
	return findings, nil
}

func validateAssessmentChecks(a health.ResourceAssessment, v health.Resource, o health.Observation, evidence map[string]*health.Evidence, proofs, blocked map[string]bool) error {
	seen := make(map[string]bool, len(a.Checks))
	for _, check := range a.Checks {
		name, ok := strings.CutPrefix(check.SourceRef, "sources.")
		if !ok || !slices.Contains(assessmentSourceNames[:], name) || seen[name] {
			return errors.New("invalid assessment source reference")
		}
		seen[name] = true
		if !blocked[name] && !proofs[name] {
			return errors.New("assessment source does not support its check")
		}
		for _, id := range check.EvidenceIDs {
			if e := evidence[id]; e == nil || e.Source != name {
				return errors.New("assessment evidence does not support its source")
			} else if !e.AppliesToAssessment(v.ID) {
				return errors.New("assessment evidence does not apply to its resource")
			}
		}
		switch name {
		case "inventory":
			if len(check.EvidenceIDs) == 0 || !slices.Equal(check.EvidenceIDs, v.EvidenceIDs) {
				return errors.New("assessment evidence does not support its inventory entry")
			}
			for _, id := range check.EvidenceIDs {
				if evidence[id].Resource == nil {
					return errors.New("assessment inventory evidence lacks resource facts")
				}
			}
		case "resource_health":
			if !blocked[name] && len(check.EvidenceIDs) == 0 || a.Disposition == "ok" && o.State != "Available" {
				return errors.New("missing deciding Resource Health evidence")
			}
			for _, id := range check.EvidenceIDs {
				e := evidence[id]
				if !o.InInventory || !slices.Contains(o.EvidenceIDs, id) || e.StatusRole != "current" || e.State != o.State {
					return errors.New("assessment does not cite deciding Resource Health evidence")
				}
			}
		case "advisor":
			for _, id := range a.PostponedEvidenceIDs {
				if e := evidence[id]; e == nil || e.Advisor == nil || e.Advisor.Status != "Postponed" || !slices.Contains(check.EvidenceIDs, id) {
					return errors.New("unresolved postponed recommendation")
				}
			}
		}
	}
	if len(seen) != len(assessmentSourceNames) {
		return errors.New("assessment is missing an applicable check")
	}
	return nil
}

func sourceRef(name string) string { return "[" + literal(name) + "](run.json)" }

func writeAssessmentQueries(b *strings.Builder, r health.Run) {
	b.WriteString("### Query provenance\n\nEach source link names an entry in the sibling JSON for this run and revision. Accepted-page times include empty pages. Pages are collected sequentially and reflect different points in time.\n\n")
	b.WriteString("| JSON source | Operation | API version | Subscription | Run reference (UTC) | Query prepared (UTC) | First accepted page (UTC) | Last accepted page (UTC) | Terminal enumeration |\n| --- | --- | --- | --- | --- | --- | --- | --- | --- |\n")
	for _, name := range assessmentSourceNames {
		s := r.Sources[name]
		var q health.SourceQuery
		if s.Query != nil {
			q = *s.Query
		}
		enumeration := "unverified"
		if s.EnumerationComplete {
			enumeration = "reached"
		}
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s | %s | %s | %s |\n",
			sourceRef("sources."+name), literal(q.Operation), literal(q.APIVersion), literal(q.SubscriptionID), stamp(&q.ReferenceAt), stamp(q.RequestedAt),
			stamp(s.FirstPageAt), stamp(s.LastPageAt), enumeration)
	}
	b.WriteByte('\n')
}

func incidentAnchor(findingID string) string {
	return "incident-" + hex.EncodeToString([]byte(findingID))
}

func assessmentFinding(f health.Finding) string {
	if f.Source == "service_health" {
		return "[" + literal(f.ID) + "](#" + incidentAnchor(f.ID) + ")"
	}
	return literal(f.Title) + " (" + refs([]string{f.ID}) + ")"
}

func writeAssessments(b *strings.Builder, r health.Run, findings map[string]health.Finding) {
	b.WriteString("## Resource assessments\n\n")
	if len(r.ResourceAssessments) == 0 {
		b.WriteString("No inventoried resources to assess. See source coverage and execution diagnostics.\n\n")
		return
	}
	b.WriteString("Each row cites the source entries and deciding evidence. A source link without evidence can support a bounded negative when the check is complete. See each collector's scope and limits below.\n\n")
	b.WriteString("| Resource ID | Result | Checks and deciding evidence | Missing checks or proof | Findings | History and Postponed context |\n| --- | --- | --- | --- | --- | --- |\n")
	assessments := slices.Clone(r.ResourceAssessments)
	sort.Slice(assessments, func(i, j int) bool { return assessments[i].ResourceID < assessments[j].ResourceID })
	for _, a := range assessments {
		checks, reasons, current, context := []string{}, []string{}, []string{}, []string{}
		for _, check := range a.Checks {
			label := sourceRef(check.SourceRef)
			if len(check.EvidenceIDs) > 0 {
				label += " " + refs(check.EvidenceIDs)
			}
			checks = append(checks, label)
		}
		for _, g := range a.BlockingReasons {
			label := literal(g.Source) + " · " + literal(g.Reason)
			if g.FindingID != "" {
				label += " " + assessmentFinding(findings[g.FindingID])
			} else if len(g.EvidenceIDs) > 0 {
				label += " " + refs(g.EvidenceIDs)
			}
			reasons = append(reasons, label)
		}
		for _, id := range a.FindingIDs {
			current = append(current, assessmentFinding(findings[id]))
		}
		for _, id := range a.HistoricalFindingIDs {
			context = append(context, "Historical: "+assessmentFinding(findings[id]))
		}
		if len(a.PostponedEvidenceIDs) > 0 {
			context = append(context, "Advisor Postponed: "+refs(a.PostponedEvidenceIDs))
		}
		result := map[string]string{"ok": "OK", "finding": "Finding", "unknown": "Unknown"}[a.Disposition]
		fmt.Fprintf(b, "| %s | %s | %s | %s | %s | %s |\n", literal(a.ResourceID), result,
			assessmentCell(checks), assessmentCell(reasons), assessmentCell(current), assessmentCell(context))
	}
	b.WriteByte('\n')
}

func assessmentCell(values []string) string {
	if len(values) == 0 {
		return "—"
	}
	return strings.Join(values, "<br>")
}

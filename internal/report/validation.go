package report

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

const subscriptionReportTitle = "# Subscription health report"
const reportOutcomePrefix = "Execution outcome: **"

func assessmentReportIntro(r health.Run) string {
	coverage := "Incomplete coverage"
	if r.Coverage == "complete" {
		coverage = "Complete coverage"
	}
	return fmt.Sprintf(subscriptionReportTitle+"\n\n**%s · %s.**\n\n"+
		"OK within the agreed checks means an inventoried resource had current Available Resource Health evidence and sufficient Service Health, Azure Monitor and Azure Advisor evidence within the saved query scopes and times. The counts cover observed inventory. A resource with a known finding and a missing check counts as unknown, with its finding shown in the assessment.\n\n",
		coverage, r.AssessmentSummary.ResourceCounts())
}

func reportIdentity(r health.Run) string {
	return fmt.Sprintf("Run ID: %s\n\nRevision: %d", literal(r.ID), r.Revision)
}

// ValidateReviewedMarkdown checks the published header and the references used
// by Markdown. The body is not a source of facts for a draft.
func ValidateReviewedMarkdown(r health.Run, contents []byte) error {
	if r.SchemaVersion != health.AssessmentSchemaVersion {
		return errors.New("drafts require a schema 4 snapshot")
	}
	if _, err := validateMarkdownRun(r); err != nil {
		return err
	}
	header, _, hasSection := bytes.Cut(contents, []byte("\n## "))
	beforeOutcome, _, hasOutcome := bytes.Cut(header, []byte("\n\n"+reportOutcomePrefix))
	if !hasSection || !hasOutcome || !bytes.HasPrefix(header, []byte(subscriptionReportTitle+"\n\n")) ||
		!bytes.HasSuffix(beforeOutcome, []byte("\n\n"+reportIdentity(r))) {
		return errors.New("report header does not match the snapshot run ID and revision")
	}
	identityLines := 0
	for line := range bytes.SplitSeq(header, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if bytes.HasPrefix(line, []byte("Run ID:")) || bytes.HasPrefix(line, []byte("Revision:")) {
			identityLines++
		}
	}
	if identityLines != 2 {
		return errors.New("ambiguous report header identity")
	}
	return nil
}

func validateMarkdownRun(r health.Run) (map[string]health.Finding, error) {
	hasAssessments := r.SchemaVersion >= health.AssessmentSchemaVersion
	if r.ID == "" || r.Revision < 1 || r.EndedAt == nil || hasAssessments && r.EndedAt.IsZero() {
		return nil, errors.New("report requires a finalized snapshot")
	}
	switch r.Outcome {
	case "completed", "partial", "failed", "cancelled":
	default:
		return nil, errors.New("invalid final outcome")
	}
	evidence := make(map[string]*health.Evidence, len(r.Evidence))
	for i := range r.Evidence {
		e := &r.Evidence[i]
		if e.ID == "" || evidence[e.ID] != nil {
			return nil, errors.New("invalid evidence identity")
		}
		evidence[e.ID] = e
	}
	check := func(ids []string) bool {
		for _, id := range ids {
			if evidence[id] == nil {
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
	if r.SchemaVersion >= 3 {
		if err := validateAdvisorReferences(r, evidence); err != nil {
			return nil, err
		}
	}
	if hasAssessments {
		return validateAssessmentReferences(r, evidence)
	}
	return nil, nil
}

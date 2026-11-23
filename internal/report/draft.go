package report

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

type DraftType string

const (
	Ticket         DraftType = "ticket"
	Recommendation DraftType = "recommendation"
)

func (t DraftType) Valid() bool { return t == Ticket || t == Recommendation }

// DraftMarkdown renders one finding from a validated selection. It uses only
// the selected source facts, checked inventory labels and local trace IDs.
func DraftMarkdown(kind DraftType, selection health.DraftSelection, item health.SelectedFinding) ([]byte, error) {
	if !kind.Valid() || len(item.Supporting) == 0 || !localRunID.MatchString(selection.RunID) ||
		azureUUID.MatchString(selection.RunID) {
		return nil, errors.New("unsupported draft type or trace identity")
	}
	for _, e := range slices.Concat(item.Supporting, item.Context) {
		if e.Source != item.Source || item.Source == "service_health" && e.ServiceHealth == nil ||
			item.Source == "azure_monitor" && e.Monitor == nil || item.Source == "advisor" && e.Advisor == nil {
			return nil, errors.New("selected evidence lacks the expected source facts")
		}
	}

	var detail strings.Builder
	var condition, review, reason, next, stateNote string
	switch item.Source {
	case "resource_health":
		if item.CurrentHealth == nil {
			return nil, errors.New("missing Resource Health summary")
		}
		condition = writeDraftHealth(&detail, item)
		review = "Resource Health observation"
		reason = "Check whether the saved observation needs follow-up for the affected resource."
		next = "Check the resource’s current Resource Health state and activity for the recorded times. Confirm whether the selected observation affected the workload."
	case "service_health":
		writeDraftService(&detail, item)
		condition, review = "Azure Service Health active notice", "Azure Service Health notice"
		if item.FuturePlannedMaintenance {
			condition = "Azure Service Health planned maintenance notice"
			stateNote = "At the time of the saved observation, the planned maintenance was scheduled to start later.\n\n"
		}
		if item.OperationalStateConflict {
			condition = "Azure Service Health notice with conflicting statuses"
			stateNote += "The saved observations leave the notice status unresolved. At least one observation recorded Active.\n\n"
		}
		reason = "Check whether the notice warrants follow-up for services and regions used by this workload."
		next = "Review the notice in Azure and confirm whether the cited services and regions overlap this workload. Check the current notice status."
	case "azure_monitor":
		writeDraftMonitor(&detail, item)
		condition, review = "Azure Monitor Fired alert", "Azure Monitor alert condition"
		if item.OperationalStateConflict {
			condition = "Azure Monitor alert with conflicting conditions"
			stateNote = "The saved observations leave the alert condition unresolved. At least one observation recorded Fired.\n\n"
		}
		reason = "Review what triggered the alert and whether the signal still needs attention."
		next = "Review the alert in Azure Monitor and verify the condition behind the signal, its target and any workload impact."
	case "advisor":
		if item.Advisor == nil {
			return nil, errors.New("missing Advisor instance summary")
		}
		writeDraftAdvisor(&detail, item)
		condition, review = "Azure Advisor New recommendation", "Azure Advisor recommendation"
		reason = "Azure Advisor recorded a recommendation for review. Confirm its relevance before deciding whether to act."
		next = "Open the recommendation in Azure Advisor, confirm that it is still applicable to the assessed target, and review the suggested action before deciding whether to proceed."
	default:
		return nil, errors.New("unsupported draft source")
	}

	var b strings.Builder
	if kind == Ticket {
		fmt.Fprintf(&b, "# Ticket: %s\n\n## Observed condition and impact\n\n%s", condition, stateNote)
	} else {
		fmt.Fprintf(&b, "# Recommendation: Review the %s\n\n## Reason for review\n\n%s%s\n\n## Supporting facts\n\n", review, stateNote, reason)
	}
	b.WriteString(detail.String())
	if len(item.Context) > 0 {
		b.WriteString("\n### Other saved observations for this finding\n\n")
		for _, e := range item.Context {
			writeDraftSourceRow(&b, e)
			if e.Source == "advisor" || e.Source == "azure_monitor" {
				writeDraftContextTarget(&b, item, e)
			}
		}
	}

	section := "Next verification step"
	if kind == Recommendation {
		section = "Proposed review"
	}
	fmt.Fprintf(&b, "\n## %s\n\n%s\n\n## Coverage\n\n", section, next)
	fmt.Fprintf(&b, "- Run outcome: %s. Overall coverage: %s.\n",
		draftEnum(selection.Outcome, "completed", "partial", "cancelled", "failed"),
		draftEnum(selection.Coverage, "complete", "incomplete"))
	fmt.Fprintf(&b, "- %s collection: %s", draftSourceName(item.Source), draftEnum(item.Collection.Status, "completed", "partial", "failed", "not_checked", "in_progress"))
	if item.Collection.Reason != "" {
		fmt.Fprintf(&b, " (%s)", draftGapReason(item.Collection.Reason))
	}
	b.WriteString(".\n")
	writeDraftGaps(&b, item.Gaps)

	fmt.Fprintf(&b, "\n## Local trace\n\n- Run ID: `%s`\n- Revision: %d\n- Finding ID: `%s`\n- Supporting evidence: ",
		selection.RunID, selection.Revision, item.Finding.ID)
	writeDraftEvidenceRefs(&b, item.Finding.EvidenceIDs)
	if len(item.TargetEvidenceIDs) > 0 {
		b.WriteString("\n- Inventory evidence for the target: ")
		writeDraftEvidenceRefs(&b, item.TargetEvidenceIDs)
	}
	b.WriteByte('\n')
	return []byte(b.String()), nil
}

func writeDraftGaps(b *strings.Builder, gaps []health.SelectionGap) {
	for _, g := range gaps {
		label := "Coverage limit"
		if g.Blocking {
			label = "Missing check or proof"
		}
		if g.Broader {
			label += " at broader scope"
		}
		fmt.Fprintf(b, "- %s · %s: %s", label, draftSourceName(g.Source), draftGapReason(g.Reason))
		if len(g.EvidenceIDs) > 0 {
			b.WriteString(" (")
			writeDraftEvidenceRefs(b, g.EvidenceIDs)
			b.WriteByte(')')
		}
		b.WriteString(".\n")
	}
}

func writeDraftEvidenceRefs(b *strings.Builder, ids []string) {
	for i, id := range ids {
		if i > 0 {
			b.WriteString(", ")
		}
		fmt.Fprintf(b, "`%s`", id)
	}
}

func draftSourceName(source string) string {
	switch source {
	case "resource_health":
		return "Resource Health"
	case "service_health":
		return "Service Health"
	case "azure_monitor":
		return "Azure Monitor"
	case "advisor":
		return "Azure Advisor"
	case "inventory":
		return "Inventory"
	case "execution":
		return "Run execution"
	default:
		return "Unknown source"
	}
}

func draftEnum(value string, allowed ...string) string {
	for _, v := range allowed {
		if v == value {
			return strings.ReplaceAll(v, "_", " ")
		}
	}
	return "unknown"
}

func draftGapReason(reason string) string {
	switch reason {
	case "missing_health_evidence", "missing_current_health_evidence":
		return "current Resource Health evidence is missing"
	case "health_not_checked":
		return "Resource Health check did not complete"
	case "service_health_impact_undetermined":
		return "workload impact is unknown"
	case "resource_impact_not_enumerated":
		return "the source did not enumerate affected resources"
	case "unknown_resource_attribution", "conflicting_resource_attribution":
		return "resource attribution is unconfirmed"
	case "invalid_records":
		return "some source records could not be used"
	case "workspace_target_only":
		return "the reported target establishes workspace scope only"
	case "non_atomic_source_snapshot":
		return "collected pages can reflect different points in time"
	case "monitor_30_day_retention_and_stateless_limits":
		return "the 30-day view can omit older alerts and can include stateless alerts that remain Fired"
	case "service_health_30_day_update_window":
		return "the update cutoff can omit older unchanged notices"
	case "not_attempted", "not_checked", "interrupted", "login_required", "access_denied",
		"source_unavailable", "unsupported", "timeout", "throttled", "invalid_response", "unsafe_continuation",
		"repeated_continuation", "page_limit", "read_boundary_rejected", "collection_failed",
		"partial", "failed", "in_progress", "missing_source_status", "inventory_unverified", "conflicting_inventory",
		"historical_health_evidence", "conflicting_health_evidence", "source_unknown",
		"undetermined_health_evidence", "health_resource_absent_from_inventory", "finding_resource_absent_from_inventory",
		"no_resources_observed", "no_accepted_page_proof", "unverified_query_policy", "unresolved_subscription_scope",
		"execution_not_finalized_successfully", "conflicting_advisor_evidence", "conflicting_advisor_state_or_target",
		"unknown_advisor_status", "unresolved_advisor_workflow", "advisor_group_target_only",
		"conflicting_service_health_regional_state", "unverified_service_health_region_labels",
		"unknown_time_coverage", "sensitive_details_not_retrieved", "missing_service_region_attribution",
		"conflicting_subscription_attribution", "unknown_region_attribution", "unknown_region_state",
		"missing_region_attribution", "missing_service_attribution", "unknown_signal_type",
		"unknown_condition", "unknown_workflow_state", "unknown_severity", "unknown_category", "unknown_state",
		"superseded_operational_evidence", "conflicting_operational_evidence", "conflicting_operational_state_or_target",
		"unclassified_operational_evidence":
		return strings.ReplaceAll(reason, "_", " ")
	default:
		return "reason unknown"
	}
}

var (
	localRunID   = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]*$`)
	azureUUID    = regexp.MustCompile(`(?i)[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}`)
	resourceType = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9]*(?:\.[a-zA-Z][a-zA-Z0-9]*)+(/[a-zA-Z][a-zA-Z0-9]*)+$`)
)

func draftLabel(value string) string {
	value, redacted := health.SafeText(value)
	value = strings.TrimSpace(value)
	// Only display labels are allowed here. A path or encoded path, URL or
	// embedded UUID can be present even in a whitelisted source field.
	if redacted != 0 || value == "" || value == "[REDACTED]" ||
		strings.ContainsAny(value, "/\\%") || azureUUID.MatchString(value) {
		return "unknown"
	}
	return literal(value)
}

func writeDraftTarget(b *strings.Builder, item health.SelectedFinding, label string) {
	if item.Finding.ResourceID == "" {
		fmt.Fprintf(b, "- %s: unknown.\n", label)
		return
	}
	if item.GroupTarget {
		fmt.Fprintf(b, "- %s: resource group scope. Individual resources are unknown.\n", label)
		return
	}
	kind := "unknown"
	if resourceType.MatchString(item.TargetType) && !azureUUID.MatchString(item.TargetType) {
		kind = literal(item.TargetType)
	}
	fmt.Fprintf(b, "- %s: resource name %s, type %s.\n", label, draftLabel(item.TargetName), kind)
}

func draftTime(value *time.Time) string {
	if value == nil || value.IsZero() {
		return "unknown"
	}
	return stamp(value)
}

func writeDraftCollectionTime(b *strings.Builder, e health.Evidence) {
	fmt.Fprintf(b, ", collected %s", draftTime(&e.CollectedAt))
	if latest := e.LatestCollectionTime(); latest.After(e.CollectedAt) {
		fmt.Fprintf(b, ", last collected %s", draftTime(&latest))
	}
}

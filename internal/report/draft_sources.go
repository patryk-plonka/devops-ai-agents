package report

import (
	"fmt"
	"slices"
	"strings"

	"github.com/patryk-plonka/devops-ai-agents/internal/health"
)

func writeDraftHealth(b *strings.Builder, item health.SelectedFinding) string {
	selected := item.Supporting[0]
	condition := "Resource Health reported " + draftHealthState(selected.State)
	if selected.Kind == "event" {
		condition = "Resource Health recorded an impacting event"
	}
	writeDraftTarget(b, item, "Affected resource")
	b.WriteString("- Scope: Resource Health availability observations for the checked subscription.\n")
	b.WriteString("- Selected observation:\n")
	writeDraftHealthRow(b, selected, "  ")
	state := item.CurrentHealth
	switch {
	case state.Conflict && state.MissingSourceTime:
		b.WriteString("- Current state is unknown. The observations disagree and at least one source timestamp is missing.\n")
	case state.Conflict:
		b.WriteString("- Current state is unknown. The latest source observations disagree.\n")
	case state.Historical:
		fmt.Fprintf(b, "- The selected observation is historical. The later deciding evidence records %s.\n", draftHealthState(state.State))
	case len(state.Observations) == 0 || state.State == "Unknown" || state.State == "undetermined":
		b.WriteString("- Current Resource Health state is unknown. The saved evidence does not establish a current availability state.\n")
	default:
		fmt.Fprintf(b, "- Saved current Resource Health state: %s.\n", draftHealthState(state.State))
	}
	if state.MissingSourceTime && !state.Conflict {
		b.WriteString("- At least one source timestamp is unknown. Collection times alone cannot establish recovery.\n")
	}
	for _, e := range state.Observations {
		if e.ID != selected.ID {
			writeDraftHealthRow(b, e, "  ")
		}
	}
	b.WriteString("- Workload impact is unknown from the Resource Health evidence.\n")
	return condition
}

func draftHealthState(state string) string {
	return draftEnum(state, "Available", "Unavailable", "Degraded")
}

func writeDraftHealthRow(b *strings.Builder, e health.Evidence, indent string) {
	observation := draftHealthState(e.State)
	if e.Kind == "event" {
		observation = "impacting event, availability state unknown"
	}
	fmt.Fprintf(b, "%s- %s. Source reported %s, source occurrence %s", indent, observation, draftTime(e.ReportedAt), draftTime(e.OccurredAt))
	writeDraftCollectionTime(b, e)
	fmt.Fprintf(b, " (`%s`).\n", e.ID)
}

func writeDraftService(b *strings.Builder, item health.SelectedFinding) {
	b.WriteString("- Scope: notice retrieved for the checked subscription. The source did not enumerate affected resources. Workload impact is unknown.\n")
	if query := item.Collection.Query; item.QueryPolicyVerified {
		fmt.Fprintf(b, "- The query uses a fixed 30-day update window from run start %s with an update cutoff of %s. Older unchanged notices may be absent.\n", draftTime(&query.ReferenceAt), draftTime(query.Cutoff))
	} else {
		b.WriteString("- Saved query scope could not be verified. The standard query uses a fixed 30-day update window and may omit older unchanged notices.\n")
		if query != nil {
			fmt.Fprintf(b, "- Recorded query reference %s and update cutoff %s.\n", draftTime(&query.ReferenceAt), draftTime(query.Cutoff))
		}
	}
	for _, e := range item.Supporting {
		writeDraftServiceRow(b, e)
	}
}

func writeDraftServiceRow(b *strings.Builder, e health.Evidence) {
	facts := e.ServiceHealth
	fmt.Fprintf(b, "- Notice category %s, status %s. Source update %s, impact start %s, impact mitigation %s",
		draftObservedEnum(e, "unknown_category", facts.Category, "ServiceIssue", "PlannedMaintenance", "HealthAdvisory", "SecurityAdvisory", "Billing", "EmergingIssues", "RCA"),
		draftObservedEnum(e, "unknown_state", facts.Status, "Active", "Resolved"), draftTime(e.ReportedAt), draftTime(facts.ImpactStartTime), draftTime(facts.ImpactMitigationTime))
	writeDraftCollectionTime(b, e)
	fmt.Fprintf(b, " (`%s`).\n", e.ID)
	if len(facts.Impact) == 0 || slices.Contains(e.Limitations, "unverified_service_health_region_labels") {
		b.WriteString("  - Service and region scope: unknown.\n")
		return
	}
	for _, impact := range facts.Impact {
		fmt.Fprintf(b, "  - Reported service: %s.\n", draftLabel(impact.Service))
		verified := false
		for _, region := range impact.Regions {
			if region.Attribution != "selected_subscription" {
				continue
			}
			verified = true
			fmt.Fprintf(b, "    - Region attributed to the checked subscription: %s. Status %s, source update %s.\n",
				draftLabel(region.Name), draftObservedEnum(e, "unknown_region_state", region.Status, "Active", "Resolved"), draftTime(region.LastUpdateTime))
		}
		if !verified || slices.ContainsFunc(impact.Regions, func(r health.ServiceRegionFacts) bool { return r.Attribution != "selected_subscription" }) {
			b.WriteString("    - Region scope is unknown for observations without confirmed subscription attribution.\n")
		}
	}
}

func writeDraftMonitor(b *strings.Builder, item health.SelectedFinding) {
	writeDraftTarget(b, item, "Azure Monitor reported target")
	for _, e := range item.Supporting {
		writeDraftMonitorRow(b, e)
	}
	if item.Finding.ResourceID != "" && slices.ContainsFunc(item.Supporting, func(e health.Evidence) bool {
		return slices.Contains(e.Limitations, "workspace_target_only")
	}) {
		b.WriteString("- The reported target supports a workspace observation only. Affected workloads are unknown.\n")
	} else {
		b.WriteString("- Fired records the alert condition. Workload impact is unknown from the alert evidence.\n")
	}
	if q := item.Collection.Query; item.QueryPolicyVerified {
		fmt.Fprintf(b, "- Query scope: 30-day Fired view of the checked subscription, across workflow and suppression states. Saved run reference %s.\n", draftTime(&q.ReferenceAt))
	} else {
		b.WriteString("- Saved query scope could not be verified. The standard view covers 30-day Fired alerts across workflow and suppression states.\n")
		if q != nil {
			fmt.Fprintf(b, "- Recorded query reference %s.\n", draftTime(&q.ReferenceAt))
		}
	}
}

func writeDraftMonitorRow(b *strings.Builder, e health.Evidence) {
	facts := e.Monitor
	suppression := "unknown"
	if facts.IsSuppressed != nil {
		suppression = "no"
		if *facts.IsSuppressed {
			suppression = "yes"
		}
	}
	fmt.Fprintf(b, "- Signal %s, condition %s, workflow %s, suppressed %s. Source start %s, source update %s",
		draftObservedEnum(e, "unknown_signal_type", facts.SignalType, "Metric", "Log"),
		draftObservedEnum(e, "unknown_condition", facts.Condition, "Fired", "Resolved"),
		draftObservedEnum(e, "unknown_workflow_state", facts.WorkflowState, "New", "Acknowledged", "Closed"),
		suppression, draftTime(e.OccurredAt), draftTime(e.ReportedAt))
	writeDraftCollectionTime(b, e)
	fmt.Fprintf(b, " (`%s`).\n", e.ID)
}

func draftObservedEnum(e health.Evidence, limitation, value string, allowed ...string) string {
	if slices.Contains(e.Limitations, limitation) {
		return "unknown"
	}
	return draftEnum(value, allowed...)
}

func writeDraftAdvisor(b *strings.Builder, item health.SelectedFinding) {
	facts := item.Advisor
	fmt.Fprintf(b, "- Saved instance summary: workflow %s, category %s, Advisor impact %s",
		advisorDraftWorkflow(facts.Status), draftEnum(facts.Category, "HighAvailability", "Security", "Performance", "Cost", "OperationalExcellence"),
		advisorDraftImpact(facts.Impact))
	if facts.Impact == "unknown" && len(item.Supporting) > 1 && slices.ContainsFunc(item.Supporting[1:], func(e health.Evidence) bool {
		return e.Advisor.Impact != item.Supporting[0].Advisor.Impact
	}) {
		b.WriteString(" because the observations disagree on impact")
	}
	fmt.Fprintf(b, ". Source evaluation %s. Source workflow update %s.\n", draftTime(facts.LastRefreshed), draftTime(facts.LastUpdated))
	writeDraftTarget(b, item, "Advisor assessed target")
	if item.QueryPolicyVerified {
		b.WriteString("- Scope: cached recommendation list for the checked subscription, with no date filter or refresh.\n")
	} else {
		b.WriteString("- Saved query scope could not be verified. The standard query reads the cached recommendation list with no date filter or refresh.\n")
	}
	b.WriteString("- Workload impact is unknown.\n")
	for _, e := range item.Supporting {
		fmt.Fprintf(b, "- Supporting Advisor observation")
		writeDraftCollectionTime(b, e)
		fmt.Fprintf(b, " (`%s`).\n", e.ID)
	}
}

func advisorDraftWorkflow(value string) string {
	return draftEnum(value, "New", "Postponed", "Dismissed", "Completed")
}

func advisorDraftImpact(value string) string { return draftEnum(value, "High", "Medium", "Low") }

func writeDraftSourceRow(b *strings.Builder, e health.Evidence) {
	switch e.Source {
	case "service_health":
		writeDraftServiceRow(b, e)
	case "azure_monitor":
		writeDraftMonitorRow(b, e)
	case "advisor":
		fmt.Fprintf(b, "- Advisor workflow %s, impact %s. Source evaluation %s, source workflow update %s",
			advisorDraftWorkflow(e.Advisor.Status), advisorDraftImpact(e.Advisor.Impact),
			draftTime(e.Advisor.LastRefreshed), draftTime(e.Advisor.LastUpdated))
		writeDraftCollectionTime(b, e)
		fmt.Fprintf(b, " (`%s`).\n", e.ID)
	case "resource_health":
		writeDraftHealthRow(b, e, "")
	}
}

func writeDraftContextTarget(b *strings.Builder, item health.SelectedFinding, e health.Evidence) {
	switch {
	case e.ResourceID == "":
		b.WriteString("  - Target attribution is unknown for this observation.\n")
	case item.Finding.ResourceID == "":
		b.WriteString("  - This observation reports a target. The selected finding’s target is unknown.\n")
	case !strings.EqualFold(e.ResourceID, item.Finding.ResourceID):
		b.WriteString("  - This observation reports a different target.\n")
	}
}

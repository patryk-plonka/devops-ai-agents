package health

import (
	"sort"
	"strings"
	"time"
)

// These inputs are whitelists, not SDK or wire objects. Narrative, tenant lists,
// query content, arbitrary links and personal identifiers have no input field.
type ServiceHealthInput struct {
	ID                                                    string
	PropertiesPresent                                     bool
	EventType, Status, Level, EventSource                 string
	LastUpdateTime, ImpactStartTime, ImpactMitigationTime *time.Time
	Sensitive                                             *bool
	Impact                                                []ServiceImpactInput
}

type ServiceImpactInput struct {
	Service string
	Regions []ServiceRegionInput
}

type ServiceRegionInput struct {
	Name, Status          string
	LastUpdateTime        *time.Time
	ImpactedSubscriptions []string
}

type ServiceHealthFacts struct {
	Category             string               `json:"category"`
	Status               string               `json:"status"`
	Level                string               `json:"level,omitempty"`
	EventSource          string               `json:"event_source,omitempty"`
	ImpactStartTime      *time.Time           `json:"impact_start_time,omitempty"`
	ImpactMitigationTime *time.Time           `json:"impact_mitigation_time,omitempty"`
	Sensitive            *bool                `json:"sensitive,omitempty"`
	Impact               []ServiceImpactFacts `json:"impact"`
}

type ServiceImpactFacts struct {
	Service string               `json:"service"`
	Regions []ServiceRegionFacts `json:"regions"`
}

type ServiceRegionFacts struct {
	Name           string     `json:"name"`
	Status         string     `json:"status,omitempty"`
	LastUpdateTime *time.Time `json:"last_update_time,omitempty"`
	Attribution    string     `json:"attribution"`
}

type MonitorInput struct {
	ID                                                               string
	PropertiesPresent                                                bool
	TargetResourceID, SignalType, Condition, WorkflowState, Severity string
	IsSuppressed                                                     *bool
	LastModifiedTime, StartTime, LastModifiedTimeByMonitorService    *time.Time
}

type MonitorFacts struct {
	SignalType                       string     `json:"signal_type"`
	Condition                        string     `json:"condition"`
	WorkflowState                    string     `json:"workflow_state,omitempty"`
	Severity                         string     `json:"severity,omitempty"`
	IsSuppressed                     *bool      `json:"is_suppressed,omitempty"`
	LastModifiedTimeByMonitorService *time.Time `json:"last_modified_time_by_monitor_service,omitempty"`
	Attribution                      string     `json:"attribution"`
}

// entityID validates native identity separately from affected-resource scope.
// Monitor identities can be extension resources; Service Health also uses global
// IDs from a subscription list. Neither identity authorizes an HTTP route.
func entityID(id, subscription, provider, resourceType string, global bool) bool {
	parts := strings.Split(id, "/")
	n := len(parts)
	if n < 5 || !strings.EqualFold(parts[n-4], "providers") || !strings.EqualFold(parts[n-3], provider) || !strings.EqualFold(parts[n-2], resourceType) {
		return false
	}
	if global && n == 5 {
		return ResourceID("/subscriptions/"+subscription+id, subscription)
	}
	if global && n != 7 {
		return false
	}
	return ResourceID(id, subscription)
}

func (r *Run) safeLabel(s string) string {
	clean, count := SafeText(s)
	r.Redactions += count
	return clean
}

func (r *Run) serviceEvidence(input ServiceHealthInput, collected time.Time) (Evidence, bool) {
	subscription := r.Scope.SubscriptionID
	if !input.PropertiesPresent || !entityID(input.ID, subscription, "Microsoft.ResourceHealth", "events", true) {
		return Evidence{}, false
	}
	facts := &ServiceHealthFacts{Category: r.safeLabel(input.EventType), Status: r.safeLabel(input.Status), Level: r.safeLabel(input.Level), EventSource: r.safeLabel(input.EventSource), ImpactStartTime: utcTime(input.ImpactStartTime), ImpactMitigationTime: utcTime(input.ImpactMitigationTime), Sensitive: copyBool(input.Sensitive), Impact: []ServiceImpactFacts{}}
	e := Evidence{Source: "service_health", Operation: "Events.ListBySubscriptionId", APIVersion: ServiceHealthAPI, EntityID: input.ID, SubscriptionID: subscription, Kind: "service_health_event", State: facts.Status, CollectedAt: collected, ReportedAt: utcTime(input.LastUpdateTime), OccurredAt: facts.ImpactStartTime, ServiceHealth: facts, Disposition: "inactive"}
	// Filtering persisted labels must not turn an unrecognized source enum into
	// a known category or state. Interpret the original values exactly.
	selected, known := false, true
	switch input.EventType {
	case "ServiceIssue", "PlannedMaintenance", "HealthAdvisory", "SecurityAdvisory":
		selected = true
	case "Billing", "EmergingIssues", "RCA":
	default:
		known = false
		e.Limitations = append(e.Limitations, "unknown_category")
	}
	if input.Status != "Active" && input.Status != "Resolved" {
		e.Limitations = append(e.Limitations, "unknown_state")
	}
	if selected && input.Status == "Active" {
		e.Disposition = "active"
	}
	if !known || (input.Status != "Active" && input.Status != "Resolved") {
		e.Disposition = "unclassified"
	}
	if !selected && known {
		e.Disposition = "excluded"
	}
	query := r.Sources[e.Source].Query
	if e.ReportedAt == nil || e.ReportedAt.IsZero() || query == nil || query.Cutoff == nil || query.ReferenceAt.IsZero() {
		e.Limitations = append(e.Limitations, "unknown_time_coverage")
		if e.Disposition != "excluded" {
			e.Disposition = "unclassified"
		}
	} else if e.ReportedAt.Before(*query.Cutoff) && e.Disposition != "unclassified" {
		e.Disposition = "excluded"
	}
	if input.EventType == "SecurityAdvisory" || facts.Sensitive != nil && *facts.Sensitive {
		e.Limitations = append(e.Limitations, "sensitive_details_not_retrieved")
	}
	if len(input.Impact) == 0 {
		e.Limitations = append(e.Limitations, "missing_service_region_attribution")
	}
	for _, impact := range input.Impact {
		accepted := ServiceImpactFacts{Regions: []ServiceRegionFacts{}}
		for _, region := range impact.Regions {
			attribution := "unknown"
			if len(region.ImpactedSubscriptions) > 0 {
				hasSelected := false
				for _, id := range region.ImpactedSubscriptions {
					if strings.EqualFold(id, subscription) {
						hasSelected = true
					}
				}
				if !hasSelected {
					e.Limitations = append(e.Limitations, "conflicting_subscription_attribution")
					continue // No foreign region/scope claim reaches a checkpoint.
				}
				attribution = "selected_subscription"
			} else {
				e.Limitations = append(e.Limitations, "unknown_region_attribution")
			}
			status := r.safeLabel(region.Status)
			if region.Status != "Active" && region.Status != "Resolved" {
				e.Limitations = append(e.Limitations, "unknown_region_state")
			}
			name := r.safeLabel(region.Name)
			if name == "" {
				e.Limitations = append(e.Limitations, "missing_region_attribution")
			}
			accepted.Regions = append(accepted.Regions, ServiceRegionFacts{Name: name, Status: status, LastUpdateTime: utcTime(region.LastUpdateTime), Attribution: attribution})
		}
		if len(accepted.Regions) == 0 {
			if len(impact.Regions) != 0 {
				continue // All reported regions conflicted with the selected scope.
			}
			e.Limitations = append(e.Limitations, "missing_region_attribution")
		}
		accepted.Service = r.safeLabel(impact.Service)
		if accepted.Service == "" {
			e.Limitations = append(e.Limitations, "missing_service_attribution")
		}
		sort.Slice(accepted.Regions, func(i, j int) bool { return stableID("", accepted.Regions[i]) < stableID("", accepted.Regions[j]) })
		accepted.Regions = uniqueFacts(accepted.Regions)
		facts.Impact = append(facts.Impact, accepted)
	}
	sort.Slice(facts.Impact, func(i, j int) bool { return stableID("", facts.Impact[i]) < stableID("", facts.Impact[j]) })
	facts.Impact = uniqueFacts(facts.Impact)
	e.Limitations = sortedUnique(e.Limitations)
	return e, true
}

func (r *Run) monitorEvidence(input MonitorInput, collected time.Time) (Evidence, bool) {
	subscription := r.Scope.SubscriptionID
	if !input.PropertiesPresent || !entityID(input.ID, subscription, "Microsoft.AlertsManagement", "alerts", false) {
		return Evidence{}, false
	}
	facts := &MonitorFacts{SignalType: r.safeLabel(input.SignalType), Condition: r.safeLabel(input.Condition), WorkflowState: r.safeLabel(input.WorkflowState), Severity: r.safeLabel(input.Severity), IsSuppressed: copyBool(input.IsSuppressed), LastModifiedTimeByMonitorService: utcTime(input.LastModifiedTimeByMonitorService), Attribution: "unknown"}
	e := Evidence{Source: "azure_monitor", Operation: "Alerts.GetAll", APIVersion: MonitorAPI, EntityID: input.ID, SubscriptionID: subscription, Kind: "monitor_alert", State: facts.Condition, Monitor: facts, CollectedAt: collected, ReportedAt: utcTime(input.LastModifiedTime), OccurredAt: utcTime(input.StartTime), Disposition: "inactive"}
	if ResourceID(input.TargetResourceID, subscription) {
		e.ResourceID = input.TargetResourceID
		facts.Attribution = "reported_target"
		if strings.Contains(strings.ToLower(e.ResourceID), "/providers/microsoft.operationalinsights/workspaces/") {
			e.Limitations = append(e.Limitations, "workspace_target_only")
		}
	} else {
		e.Limitations = append(e.Limitations, "unknown_resource_attribution")
	}
	// Source meaning uses exact input enums; filtered labels are for storage.
	switch input.SignalType {
	case "Metric", "Log":
	case "Activity Log", "Service Health", "Resource Health", "SmartDetector":
		e.Disposition = "excluded"
	default:
		e.Disposition = "unclassified"
		e.Limitations = append(e.Limitations, "unknown_signal_type")
	}
	if input.Condition != "Fired" && input.Condition != "Resolved" {
		e.Limitations = append(e.Limitations, "unknown_condition")
		if e.Disposition != "excluded" {
			e.Disposition = "unclassified"
		}
	} else if input.Condition == "Fired" && e.Disposition == "inactive" {
		e.Disposition = "active"
	}
	if input.WorkflowState != "" && input.WorkflowState != "New" && input.WorkflowState != "Acknowledged" && input.WorkflowState != "Closed" {
		e.Limitations = append(e.Limitations, "unknown_workflow_state")
	}
	if input.Severity != "" && input.Severity != "Sev0" && input.Severity != "Sev1" && input.Severity != "Sev2" && input.Severity != "Sev3" && input.Severity != "Sev4" {
		e.Limitations = append(e.Limitations, "unknown_severity")
	}
	e.Limitations = sortedUnique(e.Limitations)
	return e, true
}

func copyBool(value *bool) *bool {
	if value == nil {
		return nil
	}
	v := *value
	return &v
}
func sortedUnique(values []string) []string { sort.Strings(values); return compact(values) }
func uniqueFacts[T any](values []T) []T {
	result := values[:0]
	for _, v := range values {
		if len(result) == 0 || stableID("", result[len(result)-1]) != stableID("", v) {
			result = append(result, v)
		}
	}
	return result
}

func (r *Run) addOperational(e Evidence) {
	source := r.Sources[e.Source]
	switch e.Disposition {
	case "excluded":
		source.ExcludedRecords++
	case "unclassified":
		source.UnclassifiedRecords++
	}
	r.Sources[e.Source] = source
	r.addEvidence(e)
}

// Rebuild operational views from evidence; retrieval order never chooses state.
func (r *Run) rebuildOperationalCoverage() {
	groups := map[string][]Evidence{}
	for _, e := range r.Evidence {
		if e.Source != "service_health" && e.Source != "azure_monitor" {
			continue
		}
		key := e.Source + "\x00" + CanonicalID(e.SubscriptionID) + "\x00" + CanonicalID(e.EntityID)
		groups[key] = append(groups[key], e)
		for _, reason := range e.Limitations {
			r.Gaps = append(r.Gaps, Gap{Source: e.Source, EntityID: e.EntityID, ResourceID: e.ResourceID, Reason: reason, EvidenceIDs: []string{e.ID}})
		}
	}
	for _, source := range []string{"service_health", "azure_monitor"} {
		s := r.Sources[source]
		if s.Query == nil && s.Pages == 0 {
			continue
		}
		reasons := []string{"non_atomic_source_snapshot"}
		if source == "service_health" {
			reasons = append(reasons, "service_health_30_day_update_window", "resource_impact_not_enumerated")
		} else {
			reasons = append(reasons, "monitor_30_day_retention_and_stateless_limits")
		}
		for _, reason := range reasons {
			r.Gaps = append(r.Gaps, Gap{Source: source, Reason: reason})
		}
	}
	for key, observations := range groups {
		sort.Slice(observations, func(i, j int) bool { return observations[i].ID < observations[j].ID })
		selected, determined := 0, true
		if len(observations) > 1 {
			for i, e := range observations {
				if e.ReportedAt == nil || e.ReportedAt.IsZero() {
					determined = false
					break
				}
				if e.ReportedAt.After(*observations[selected].ReportedAt) {
					selected = i
				}
			}
			if determined {
				for i, e := range observations {
					if i != selected && e.ReportedAt.Equal(*observations[selected].ReportedAt) {
						determined = false
					}
				}
			}
			ids := []string{}
			for i, e := range observations {
				if !determined || i != selected {
					ids = append(ids, e.ID)
				}
			}
			reason := "superseded_operational_evidence"
			if !determined {
				reason = "conflicting_operational_evidence"
			}
			r.Gaps = append(r.Gaps, Gap{Source: observations[0].Source, EntityID: observations[0].EntityID, Reason: reason, EvidenceIDs: ids})
		}
		if !determined {
			continue
		}
		e := observations[selected]
		if e.Disposition != "active" {
			continue
		}
		category, title := "monitor_alert", "Azure Monitor source condition Fired"
		if e.ServiceHealth != nil {
			facts := e.ServiceHealth
			category, title = "service_health_"+facts.Category, "Service Health Active "+facts.Category+" notice"
			if facts.Category == "ServiceIssue" {
				title = "Service Health Active ServiceIssue"
			}
			query := r.Sources[e.Source].Query
			if facts.ImpactStartTime != nil && query != nil && facts.ImpactStartTime.After(query.ReferenceAt) {
				title = "Service Health future " + facts.Category + " notice"
			}
		}
		r.Findings = append(r.Findings, Finding{ID: stableID("finding-", key), Source: e.Source, EntityID: e.EntityID, SubscriptionID: e.SubscriptionID, ResourceID: e.ResourceID, Category: category, Title: title, EvidenceIDs: []string{e.ID}})
	}
}

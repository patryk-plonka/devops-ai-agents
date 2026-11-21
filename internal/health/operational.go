package health

import (
	"slices"
	"sort"
	"strings"
	"time"
)

const conflictingServiceHealthRegionalState = "conflicting_service_health_regional_state"
const unverifiedServiceHealthRegionLabels = "unverified_service_health_region_labels"

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
	updated, updatedValid := sourceTime(input.LastUpdateTime)
	start, startValid := sourceTime(input.ImpactStartTime)
	mitigation, mitigationValid := sourceTime(input.ImpactMitigationTime)
	if !updatedValid || !startValid || !mitigationValid {
		return Evidence{}, false
	}
	facts := &ServiceHealthFacts{Category: r.safeLabel(input.EventType), Status: r.safeLabel(input.Status), Level: r.safeLabel(input.Level), EventSource: r.safeLabel(input.EventSource), ImpactStartTime: start, ImpactMitigationTime: mitigation, Sensitive: copyBool(input.Sensitive), Impact: []ServiceImpactFacts{}}
	e := Evidence{Source: "service_health", Operation: serviceHealthOperation, APIVersion: ServiceHealthAPI, EntityID: input.ID, SubscriptionID: subscription, Kind: "service_health_event", State: facts.Status, CollectedAt: collected, ReportedAt: updated, OccurredAt: facts.ImpactStartTime, ServiceHealth: facts, Disposition: "inactive"}
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
	if r.SchemaVersion >= AssessmentSchemaVersion && e.Disposition == "unclassified" {
		switch {
		case query != nil && query.Cutoff != nil && e.ReportedAt != nil &&
			!e.ReportedAt.IsZero() && e.ReportedAt.Before(*query.Cutoff):
			e.Disposition = "excluded"
		case input.Status == "Resolved":
			e.Disposition = "inactive"
		}
	}
	if input.EventType == "SecurityAdvisory" || facts.Sensitive != nil && *facts.Sensitive {
		e.Limitations = append(e.Limitations, "sensitive_details_not_retrieved")
	}
	if len(input.Impact) == 0 {
		e.Limitations = append(e.Limitations, "missing_service_region_attribution")
	}
	checkRegionalState := r.SchemaVersion >= AssessmentSchemaVersion && input.EventType == "ServiceIssue" && input.Status == "Resolved" &&
		e.ReportedAt != nil && !e.ReportedAt.IsZero() && query != nil && query.Cutoff != nil && !query.ReferenceAt.IsZero()
	for _, impact := range input.Impact {
		accepted := ServiceImpactFacts{Regions: []ServiceRegionFacts{}}
		for _, region := range impact.Regions {
			attribution := "unknown"
			unknownScope := len(region.ImpactedSubscriptions) == 0
			for _, id := range region.ImpactedSubscriptions {
				if strings.EqualFold(id, subscription) {
					attribution = "selected_subscription"
					break
				}
				unknownScope = unknownScope || r.SchemaVersion >= AssessmentSchemaVersion && !subscriptionUUID.MatchString(id)
			}
			if attribution == "unknown" {
				if !unknownScope {
					e.Limitations = append(e.Limitations, "conflicting_subscription_attribution")
					continue // No foreign region/scope claim reaches a checkpoint.
				}
				e.Limitations = append(e.Limitations, "unknown_region_attribution")
			}
			regionUpdated, valid := sourceTime(region.LastUpdateTime)
			if !valid {
				return Evidence{}, false
			}
			if checkRegionalState && region.Status == "Active" &&
				regionUpdated != nil && regionUpdated.After(*e.ReportedAt) && !regionUpdated.Before(*query.Cutoff) {
				e.Limitations = append(e.Limitations, conflictingServiceHealthRegionalState)
			}
			status := r.safeLabel(region.Status)
			if region.Status != "Active" && region.Status != "Resolved" {
				e.Limitations = append(e.Limitations, "unknown_region_state")
			}
			name := r.safeLabel(region.Name)
			if r.SchemaVersion >= AssessmentSchemaVersion && name != region.Name {
				e.Limitations = append(e.Limitations, unverifiedServiceHealthRegionLabels)
			}
			if name == "" {
				e.Limitations = append(e.Limitations, "missing_region_attribution")
			}
			accepted.Regions = append(accepted.Regions, ServiceRegionFacts{Name: name, Status: status, LastUpdateTime: regionUpdated, Attribution: attribution})
		}
		if len(accepted.Regions) == 0 {
			if len(impact.Regions) != 0 {
				continue // All reported regions conflicted with the selected scope.
			}
			e.Limitations = append(e.Limitations, "missing_region_attribution")
		}
		accepted.Service = r.safeLabel(impact.Service)
		if r.SchemaVersion >= AssessmentSchemaVersion && accepted.Service != impact.Service {
			e.Limitations = append(e.Limitations, unverifiedServiceHealthRegionLabels)
		}
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
	modified, modifiedValid := sourceTime(input.LastModifiedTime)
	start, startValid := sourceTime(input.StartTime)
	serviceModified, serviceModifiedValid := sourceTime(input.LastModifiedTimeByMonitorService)
	if !modifiedValid || !startValid || !serviceModifiedValid {
		return Evidence{}, false
	}
	facts := &MonitorFacts{SignalType: r.safeLabel(input.SignalType), Condition: r.safeLabel(input.Condition), WorkflowState: r.safeLabel(input.WorkflowState), Severity: r.safeLabel(input.Severity), IsSuppressed: copyBool(input.IsSuppressed), LastModifiedTimeByMonitorService: serviceModified, Attribution: "unknown"}
	e := Evidence{Source: "azure_monitor", Operation: monitorOperation, APIVersion: MonitorAPI, EntityID: input.ID, SubscriptionID: subscription, Kind: "monitor_alert", State: facts.Condition, Monitor: facts, CollectedAt: collected, ReportedAt: modified, OccurredAt: start, Disposition: "inactive"}
	if ResourceID(input.TargetResourceID, subscription) {
		e.ResourceID = input.TargetResourceID
		facts.Attribution = "reported_target"
		if monitorWorkspaceTarget(e.ResourceID) {
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
	if r.SchemaVersion >= AssessmentSchemaVersion && input.Condition == "Resolved" && e.Disposition == "unclassified" {
		e.Disposition = "inactive"
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

func monitorWorkspaceTarget(id string) bool {
	return strings.Contains(canonicalFoldID(id), "/providers/microsoft.operationalinsights/workspaces/")
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
func (r *Run) rebuildOperationalCoverage(proof *assessmentEvidence) {
	groups := map[string][]Evidence{}
	for _, e := range r.Evidence {
		if e.Source != "service_health" && e.Source != "azure_monitor" {
			continue
		}
		key := e.Source + "\x00" + CanonicalID(e.SubscriptionID) + "\x00" + r.identityKey(e.EntityID)
		groups[key] = append(groups[key], e)
		for _, reason := range e.Limitations {
			if reason == conflictingServiceHealthRegionalState {
				continue // The effective event can contribute a blocking gap below.
			}
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
		var regionalConflict []string
		if proof != nil && observations[0].ServiceHealth != nil {
			regionalConflict = r.serviceHealthRegionalConflict(observations)
		}
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
				_, decidingRegion := slices.BinarySearch(regionalConflict, e.ID)
				if !determined || i != selected && !decidingRegion {
					ids = append(ids, e.ID)
				}
			}
			reason := "superseded_operational_evidence"
			if !determined {
				reason = "conflicting_operational_evidence"
			}
			if len(ids) > 0 {
				r.Gaps = append(r.Gaps, Gap{Source: observations[0].Source, EntityID: observations[0].EntityID, Reason: reason, EvidenceIDs: ids})
			}
		}
		if !determined {
			if proof != nil {
				r.classifyOperationalConflict(proof, key, observations, regionalConflict)
			}
			continue
		}
		e := observations[selected]
		finding := ""
		if e.Disposition == "active" {
			f := r.operationalFinding(key, e)
			finding = f.ID
			r.Findings = append(r.Findings, f)
		}
		if proof != nil {
			proof.add(e.Source, e.ResourceID, []string{e.ID})
			if len(regionalConflict) > 0 {
				r.block(e.Source, canonicalFoldID(e.EntityID), "", conflictingServiceHealthRegionalState, regionalConflict, finding)
			} else if reason := r.operationalBlocker(e); reason != "" {
				r.blockTargets(e.Source, canonicalFoldID(e.EntityID), reason, []Evidence{e}, []string{e.ID}, finding)
			}
		}
	}
}

type serviceHealthUpdate struct {
	at          *time.Time
	evidenceIDs []string
}

func (u *serviceHealthUpdate) observe(at *time.Time, id string) {
	switch {
	case at == nil || at.IsZero():
		if u.at == nil {
			u.evidenceIDs = append(u.evidenceIDs, id)
		}
	case u.at == nil || at.After(*u.at):
		u.at, u.evidenceIDs = at, []string{id}
	case at.Equal(*u.at):
		u.evidenceIDs = append(u.evidenceIDs, id)
	}
}

// Event headers and regional details may advance independently across pages.
// A resolution must be strictly newer than the unresolved regional observation.
func (r *Run) serviceHealthRegionalConflict(observations []Evidence) []string {
	query := r.Sources["service_health"].Query
	if query == nil || query.Cutoff == nil || query.ReferenceAt.IsZero() {
		return nil
	}
	type serviceRegion struct{ service, region string }
	var eventResolution serviceHealthUpdate
	unresolved, resolved := make(map[serviceRegion]serviceHealthUpdate), make(map[serviceRegion]serviceHealthUpdate)
	for _, e := range observations {
		facts := e.ServiceHealth
		if facts == nil || slices.Contains(e.Limitations, "unknown_category") {
			continue
		}
		if facts.Category != "ServiceIssue" && (facts.Category != "PlannedMaintenance" || r.futureServiceEvent(e)) {
			continue
		}
		if facts.Status == "Resolved" && !slices.Contains(e.Limitations, "unknown_state") {
			eventResolution.observe(e.ReportedAt, e.ID)
		}
		verifiedRegionLabels := !slices.Contains(e.Limitations, unverifiedServiceHealthRegionLabels)
		verifiedRegionState := !slices.Contains(e.Limitations, "unknown_region_state")
		observedAt := serviceHealthObservationTime(e, query.ReferenceAt)
		for _, impact := range facts.Impact {
			service := canonicalFoldID(impact.Service)
			for _, region := range impact.Regions {
				t := region.LastUpdateTime
				dated := t != nil && !t.IsZero()
				if dated && t.Before(*query.Cutoff) {
					continue
				}
				var key serviceRegion
				if verifiedRegionLabels {
					key = serviceRegion{service, canonicalFoldID(region.Name)}
				}
				if region.Status == "Resolved" && verifiedRegionState {
					if dated && region.Attribution == "selected_subscription" && key.service != "" && key.region != "" {
						u := resolved[key]
						u.observe(t, e.ID)
						resolved[key] = u
					}
					continue
				}
				if !dated {
					// An untimed state was still present at the saved read. Require
					// resolution after that observation before clearing its uncertainty.
					t = &observedAt
				}
				u := unresolved[key]
				u.observe(t, e.ID)
				unresolved[key] = u
			}
		}
	}
	if len(eventResolution.evidenceIDs) == 0 {
		return nil
	}
	var ids []string
	for key, update := range unresolved {
		regionResolution := resolved[key]
		if eventResolution.at != nil && eventResolution.at.After(*update.at) ||
			regionResolution.at != nil && regionResolution.at.After(*update.at) {
			continue
		}
		ids = append(ids, update.evidenceIDs...)
		ids = append(ids, regionResolution.evidenceIDs...)
	}
	if len(ids) > 0 {
		ids = append(ids, eventResolution.evidenceIDs...)
	}
	return sortedUnique(ids)
}

func (r *Run) operationalFinding(key string, e Evidence) Finding {
	category, title := "monitor_alert", "Azure Monitor source condition Fired"
	if e.ServiceHealth != nil {
		facts := e.ServiceHealth
		category, title = "service_health_"+facts.Category, "Service Health Active "+facts.Category+" notice"
		if facts.Category == "ServiceIssue" {
			title = "Service Health Active ServiceIssue"
		}
		if r.futureServiceEvent(e) {
			title = "Service Health future " + facts.Category + " notice"
		}
	}
	return Finding{ID: stableID("finding-", key), Source: e.Source, EntityID: e.EntityID, SubscriptionID: e.SubscriptionID, ResourceID: e.ResourceID, Category: category, Title: title, EvidenceIDs: []string{e.ID}}
}

func (r *Run) futureServiceEvent(e Evidence) bool {
	q := r.Sources[e.Source].Query
	start := e.ServiceHealth.ImpactStartTime
	if start == nil || q == nil {
		return false
	}
	asOf := q.ReferenceAt
	if r.SchemaVersion >= AssessmentSchemaVersion {
		asOf = serviceHealthObservationTime(e, asOf)
	}
	return start.After(asOf)
}

func serviceHealthObservationTime(e Evidence, asOf time.Time) time.Time {
	if collected := e.LatestCollectionTime(); collected.After(asOf) {
		asOf = collected
	}
	if e.ReportedAt != nil && e.ReportedAt.After(asOf) {
		asOf = *e.ReportedAt
	}
	return asOf
}

func (r *Run) operationalBlocker(e Evidence) string {
	switch e.Disposition {
	case "inactive", "excluded":
		return ""
	case "unclassified":
		return "unclassified_operational_evidence"
	case "active":
		if e.ServiceHealth != nil {
			if r.futureServiceEvent(e) && e.ServiceHealth.Category != "ServiceIssue" {
				return ""
			}
			return "service_health_impact_undetermined"
		}
		if e.ResourceID == "" {
			return "unknown_resource_attribution"
		}
		for _, reason := range e.Limitations {
			switch reason {
			case "unknown_workflow_state", "unknown_severity":
				continue
			default:
				return reason
			}
		}
		if e.Monitor != nil && e.Monitor.Attribution == "reported_target" {
			return ""
		}
	}
	return "unclassified_operational_evidence"
}

func (r *Run) classifyOperationalConflict(proof *assessmentEvidence, key string, observations []Evidence, regionalConflict []string) {
	current := latestCandidates(observations, func(e Evidence) *time.Time { return e.ReportedAt })
	first := current[0]
	target := first.ResourceID
	var positives []Evidence
	blocking := len(regionalConflict) > 0
	for _, e := range current {
		if !strings.EqualFold(e.ResourceID, target) {
			target = ""
		}
		if e.Disposition == "active" {
			positives = append(positives, e)
		}
		blocking = blocking || r.operationalBlocker(e) != ""
	}
	if !blocking && len(positives) == 0 {
		return
	}
	agreedMonitorFinding := !blocking && first.Monitor != nil && target != "" && len(positives) == len(current)
	finding := ""
	if len(positives) > 0 {
		f := r.operationalFinding(key, positives[0])
		f.ResourceID, f.EvidenceIDs = target, evidenceRefs(positives)
		if !agreedMonitorFinding {
			f.Title += " in conflicting observations"
		}
		finding = f.ID
		r.Findings = append(r.Findings, f)
	}
	if agreedMonitorFinding {
		proof.add(first.Source, target, evidenceRefs(positives))
		return
	}
	if !blocking && first.ServiceHealth != nil {
		return
	}
	if len(regionalConflict) > 0 {
		r.block(first.Source, canonicalFoldID(first.EntityID), "", conflictingServiceHealthRegionalState, regionalConflict, finding)
	} else {
		r.blockTargets(first.Source, canonicalFoldID(first.EntityID), "conflicting_operational_state_or_target", current, evidenceRefs(current), finding)
	}
}

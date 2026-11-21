package health

import (
	"errors"
	"maps"
	"slices"
	"strings"
	"time"
)

var assessmentSources = [...]string{"inventory", "resource_health", "service_health", "azure_monitor", advisorSource}

func assessmentQuery(source string, reference time.Time, subscription string) SourceQuery {
	q := SourceQuery{ReferenceAt: reference.UTC()}
	switch source {
	case "inventory":
		q.Operation, q.APIVersion = inventoryOperation, InventoryAPI
	case "resource_health":
		q.Operation, q.APIVersion = resourceHealthOperation, ResourceHealthAPI
	case "service_health":
		q = ServiceHealthQuery(reference)
		q.Operation, q.APIVersion = serviceHealthOperation, ServiceHealthAPI
	case "azure_monitor":
		q = MonitorQuery(reference)
		q.Operation, q.APIVersion, q.MonitorCondition = monitorOperation, MonitorAPI, "Fired"
	case advisorSource:
		q = AdvisorQuery(reference)
	}
	q.SubscriptionID = subscription
	return q
}

func (r *Run) sourceDecisionGaps() {
	if r.Scope == nil || !subscriptionUUID.MatchString(r.Scope.SubscriptionID) {
		r.block("inventory", "", "", "unresolved_subscription_scope", nil, "")
		return
	}
	for _, name := range assessmentSources {
		s := r.Sources[name]
		if s.Status != "completed" && !s.finishedWithRejections() {
			continue // The source status already carries a blocking reason.
		}
		if s.Reason != "" && s.Reason != "invalid_records" {
			r.block(name, "", "", s.Reason, nil, "")
		}
		if !s.hasAcceptedPageProof() {
			r.block(name, "", "", "no_accepted_page_proof", nil, "")
		}
		if !r.matchesAssessmentQuery(name, s.Query) {
			r.block(name, "", "", "unverified_query_policy", nil, "")
		}
	}
}

func (s Source) hasAcceptedPageProof() bool {
	return s.EnumerationComplete && s.Pages > 0 && s.FirstPageAt != nil && s.LastPageAt != nil &&
		!s.FirstPageAt.IsZero() && !s.LastPageAt.Before(*s.FirstPageAt)
}

func (r Run) matchesAssessmentQuery(name string, q *SourceQuery) bool {
	if q == nil || r.Scope == nil || !subscriptionUUID.MatchString(r.Scope.SubscriptionID) {
		return false
	}
	want := assessmentQuery(name, r.StartedAt, r.Scope.SubscriptionID)
	return !q.ReferenceAt.IsZero() && q.ReferenceAt.Equal(want.ReferenceAt) &&
		strings.EqualFold(q.SubscriptionID, want.SubscriptionID) &&
		q.Operation == want.Operation && q.APIVersion == want.APIVersion &&
		q.TimeRange == want.TimeRange && q.MonitorCondition == want.MonitorCondition &&
		q.QueryStartTime == want.QueryStartTime && equalCutoff(q.Cutoff, want.Cutoff)
}

// HasAssessmentSourceProof validates the read behind a saved check reference.
// It does not establish whether a resource is affected by a scoped gap.
func (r Run) HasAssessmentSourceProof(name string) bool {
	s := r.Sources[name]
	return slices.Contains(assessmentSources[:], name) &&
		((s.Status == "completed" && s.Reason == "" && s.RejectedRecords == 0) ||
			(s.finishedWithRejections() && len(s.RejectedScopes) == s.RejectedRecords)) &&
		s.hasAcceptedPageProof() && r.matchesAssessmentQuery(name, s.Query)
}

// Use only source time to exclude superseded versions. A version without
// source time can still compete with the latest timed one.
func latestCandidates(observations []Evidence, sourceTime func(Evidence) *time.Time) []Evidence {
	var latest *time.Time
	for _, e := range observations {
		if t := sourceTime(e); t != nil && !t.IsZero() && (latest == nil || t.After(*latest)) {
			latest = t
		}
	}
	if latest == nil {
		return observations
	}
	result := make([]Evidence, 0, len(observations))
	for _, e := range observations {
		t := sourceTime(e)
		if t == nil || t.IsZero() || t.Equal(*latest) {
			result = append(result, e)
		}
	}
	return result
}

func evidenceRefs(observations []Evidence) []string {
	ids := make([]string, 0, len(observations))
	for _, e := range observations {
		ids = append(ids, e.ID)
	}
	return sortedUnique(ids)
}

func equalCutoff(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

type assessmentEvidence struct {
	bySource  map[string]map[string][]string
	postponed map[string][]string
}

func (p *assessmentEvidence) add(source, target string, ids []string) {
	if target == "" {
		return
	}
	if p.bySource[source] == nil {
		p.bySource[source] = make(map[string][]string)
	}
	key := canonicalFoldID(target)
	p.bySource[source][key] = append(p.bySource[source][key], ids...)
}

func (r *Run) block(source, entity, target, reason string, ids []string, finding string) {
	r.Gaps = append(r.Gaps, Gap{Source: source, EntityID: entity, ResourceID: target,
		Reason: reason, EvidenceIDs: ids, Blocking: true, FindingID: finding})
}

// A conflict can be scoped only when every deciding observation has a
// trusted target. A native entity ID cannot substitute for a missing target.
func (r *Run) blockTargets(source, entity, reason string, observations []Evidence, ids []string, finding string) {
	targets := make(map[string]bool)
	for _, e := range observations {
		if e.ResourceID == "" || slices.Contains(e.Limitations, "workspace_target_only") {
			r.block(source, entity, "", reason, ids, finding)
			return
		}
		targets[canonicalFoldID(e.ResourceID)] = true
	}
	for target := range targets {
		r.block(source, entity, target, reason, ids, finding)
	}
}

func sameInventoryFacts(a, b ResourceFacts) bool {
	return strings.EqualFold(a.ID, b.ID) && strings.EqualFold(a.Name, b.Name) &&
		strings.EqualFold(a.Type, b.Type) && strings.EqualFold(a.ResourceGroup, b.ResourceGroup) &&
		strings.EqualFold(a.Location, b.Location) && maps.Equal(a.Tags, b.Tags)
}

func resourceGroupTarget(id string) string {
	parts := strings.SplitN(id, "/", 6)
	if len(parts) >= 5 && strings.EqualFold(parts[3], "resourceGroups") {
		return strings.Join(parts[:5], "/")
	}
	return ""
}

// AppliesToAssessment checks the normalized target of per-resource evidence.
// Advisor can also provide context scoped to the resource's group.
func (e Evidence) AppliesToAssessment(resourceID string) bool {
	if e.ResourceID == "" {
		return false
	}
	return strings.EqualFold(e.ResourceID, resourceID) ||
		e.Source == advisorSource && strings.EqualFold(e.ResourceID, resourceGroupTarget(resourceID))
}

func (f Finding) supersededForAssessment(deciding []string) bool {
	// Existing Resource Health findings omit source and cite one observation.
	return f.Source == "" && len(f.EvidenceIDs) == 1 &&
		len(deciding) > 0 && !slices.Contains(deciding, f.EvidenceIDs[0])
}

// ValidateAssessmentFindingReferences verifies deciding Resource Health evidence
// and the saved current and historical finding lists for inventoried resources.
func (r Run) ValidateAssessmentFindingReferences() error {
	currentHealth := make(map[string][]Evidence)
	for _, e := range r.Evidence {
		if e.Source == "resource_health" && e.StatusRole == "current" {
			key := canonicalFoldID(e.ResourceID)
			currentHealth[key] = append(currentHealth[key], e)
		}
	}
	byTarget := make(map[string][]Finding)
	for _, f := range r.Findings {
		if f.ResourceID != "" {
			target := canonicalFoldID(f.ResourceID)
			byTarget[target] = append(byTarget[target], f)
		}
	}
	for _, a := range r.ResourceAssessments {
		key := canonicalFoldID(a.ResourceID)
		_, deciding := decideResourceHealth(currentHealth[key], r.SchemaVersion)
		for _, check := range a.Checks {
			if check.SourceRef == "sources.resource_health" {
				if !slices.Equal(check.EvidenceIDs, deciding) {
					return errors.New("assessment does not cite deciding Resource Health evidence")
				}
				break
			}
		}
		var current, historical []string
		for _, f := range byTarget[key] {
			if f.supersededForAssessment(deciding) {
				historical = append(historical, f.ID)
			} else {
				current = append(current, f.ID)
			}
		}
		slices.Sort(current)
		slices.Sort(historical)
		if !slices.Equal(current, a.FindingIDs) || !slices.Equal(historical, a.HistoricalFindingIDs) {
			return errors.New("assessment does not cite the applicable current and historical findings")
		}
	}
	return nil
}

func (r Run) assessmentBlockingGaps() map[string][]Gap {
	gaps := make(map[string][]Gap)
	for _, gap := range r.Gaps {
		if gap.Blocking {
			key := canonicalFoldID(gap.ResourceID)
			gaps[key] = append(gaps[key], gap)
		}
	}
	return gaps
}

func assessmentBlockers(gaps map[string][]Gap, resource, group string) []Gap {
	if group == "" {
		return slices.Concat(gaps[""], gaps[resource])
	}
	return slices.Concat(gaps[""], gaps[resource], gaps[group])
}

// ValidateAssessmentBlockingReasons checks that saved assessments cite all
// applicable subscription, resource and resource-group gaps.
func (r Run) ValidateAssessmentBlockingReasons() error {
	gaps := r.assessmentBlockingGaps()
	for _, a := range r.ResourceAssessments {
		want := assessmentBlockers(gaps, canonicalFoldID(a.ResourceID), canonicalFoldID(resourceGroupTarget(a.ResourceID)))
		if !slices.EqualFunc(want, a.BlockingReasons, func(a, b Gap) bool {
			return a.Source == b.Source && a.EntityID == b.EntityID && a.ResourceID == b.ResourceID &&
				a.Reason == b.Reason && a.Blocking == b.Blocking && a.FindingID == b.FindingID && slices.Equal(a.EvidenceIDs, b.EvidenceIDs)
		}) {
			return errors.New("assessment blocking reasons do not match its applicable coverage gaps")
		}
	}
	return nil
}

func (r *Run) rebuildAssessments(proof *assessmentEvidence, inventory map[string]int) {
	final := r.EndedAt != nil && !r.EndedAt.IsZero() && (r.Outcome == "completed" || r.Outcome == "partial")
	if !final {
		r.block("execution", "", "", "execution_not_finalized_successfully", nil, "")
	}
	findings, history := make(map[string][]string), make(map[string][]string)
	for _, f := range r.Findings {
		if f.ResourceID == "" {
			continue
		}
		target := canonicalFoldID(f.ResourceID)
		if f.supersededForAssessment(proof.bySource["resource_health"][target]) {
			history[target] = append(history[target], f.ID)
			continue
		}
		if _, present := inventory[target]; f.Source != "" && !present && r.Scope != nil && ResourceID(f.ResourceID, r.Scope.SubscriptionID) {
			r.block("inventory", canonicalFoldID(f.EntityID), target, "finding_resource_absent_from_inventory", f.EvidenceIDs, f.ID)
		}
		findings[target] = append(findings[target], f.ID)
	}
	r.sortGaps()
	reasons := r.assessmentBlockingGaps()
	r.ResourceAssessments = make([]ResourceAssessment, 0, len(r.Resources))
	summary := &AssessmentSummary{}
	for _, resource := range r.Resources {
		key := canonicalFoldID(resource.ID)
		group := canonicalFoldID(resourceGroupTarget(resource.ID))
		a := ResourceAssessment{ResourceID: resource.ID, FindingIDs: findings[key], HistoricalFindingIDs: history[key]}
		slices.Sort(a.FindingIDs)
		slices.Sort(a.HistoricalFindingIDs)
		for _, source := range assessmentSources {
			ids := proof.bySource[source][key]
			if source == "inventory" {
				ids = resource.EvidenceIDs
			} else if source == advisorSource && group != "" {
				ids = slices.Concat(ids, proof.bySource[source][group])
			}
			a.Checks = append(a.Checks, AssessmentCheck{SourceRef: "sources." + source, EvidenceIDs: sortedUnique(slices.Clone(ids))})
		}
		a.PostponedEvidenceIDs = sortedUnique(slices.Concat(proof.postponed[key], proof.postponed[group]))
		a.BlockingReasons = assessmentBlockers(reasons, key, group)
		switch {
		case len(a.BlockingReasons) > 0:
			a.Disposition = "unknown"
			summary.Unknown++
		case len(a.FindingIDs) > 0:
			a.Disposition = "finding"
			summary.Finding++
		default:
			a.Disposition = "ok"
			summary.OK++
		}
		r.ResourceAssessments = append(r.ResourceAssessments, a)
	}
	r.AssessmentSummary = summary
	if final && r.Outcome == "completed" && len(r.Resources) > 0 && len(reasons) == 0 && summary.Unknown == 0 {
		r.Coverage = "complete"
	}
}

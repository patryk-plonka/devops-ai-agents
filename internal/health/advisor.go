package health

import (
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	AdvisorAPI           = "2026-10-01"
	AdvisorListOperation = "Recommendations.List"
	advisorSource        = "advisor"
	advisorUnknown       = "unknown"
)

// AdvisorInput contains only the facts used for review and traceability.
type AdvisorInput struct {
	ID, TypeID, Category, Impact, Status, TargetResourceID string
	LastRefreshed, LastUpdated                             *time.Time
}

type AdvisorFacts struct {
	TypeID                string     `json:"type_id,omitempty"`
	Category              string     `json:"category"`
	Impact                string     `json:"impact"`
	Status                string     `json:"status"`
	AttributionLimitation string     `json:"attribution_limitation,omitempty"`
	LastRefreshed         *time.Time `json:"last_refreshed,omitempty"`
	LastUpdated           *time.Time `json:"last_updated,omitempty"`
}

// AdvisorInstance is the review row derived from all observations of one
// native instance. An absent type ID leaves the instance in its own group.
type AdvisorInstance struct {
	EntityID    string   `json:"entity_id"`
	ResourceID  string   `json:"resource_id,omitempty"`
	FindingID   string   `json:"finding_id,omitempty"`
	EvidenceIDs []string `json:"evidence_ids"`
	AdvisorFacts
}

type advisorScope int

const (
	advisorInvalidScope advisorScope = iota
	advisorSubscriptionScope
	advisorResourceGroupScope
	advisorResourceScope
)

var advisorResourceGroupName = regexp.MustCompile(`^[\p{L}\p{Nd}_().-]{1,90}$`)

func advisorPathScope(parts []string) advisorScope {
	if len(parts) < 3 {
		return advisorInvalidScope
	}
	for _, part := range parts[3:] {
		if strings.TrimSpace(part) == "" {
			return advisorInvalidScope
		}
	}
	i, scope := 3, advisorSubscriptionScope
	if len(parts) > i && strings.EqualFold(parts[i], "resourceGroups") {
		if len(parts) <= i+1 || !advisorResourceGroupName.MatchString(parts[i+1]) || strings.HasSuffix(parts[i+1], ".") {
			return advisorInvalidScope
		}
		i, scope = i+2, advisorResourceGroupScope
	}
	if i == len(parts) {
		return scope
	}
	if !strings.EqualFold(parts[i], "providers") {
		return advisorInvalidScope
	}
	for i < len(parts) {
		if strings.EqualFold(parts[i], "providers") {
			// Each provider introduces a resource before another extension.
			if len(parts)-i < 4 || strings.EqualFold(parts[i+2], "providers") {
				return advisorInvalidScope
			}
			i += 4
		} else {
			if len(parts)-i < 2 {
				return advisorInvalidScope
			}
			i += 2
		}
	}
	return advisorResourceScope
}

// advisorInstanceParent returns the parent resource only for resource-scoped
// instances. The resource group and subscription forms also have valid IDs.
func advisorInstanceParent(id, subscription string) (string, bool) {
	parts := scopedARMPath(id, subscription)
	if len(parts) < 7 {
		return "", false
	}
	parent, suffix := parts[:len(parts)-4], parts[len(parts)-4:]
	if !strings.EqualFold(suffix[0], "providers") || !strings.EqualFold(suffix[1], "Microsoft.Advisor") ||
		!strings.EqualFold(suffix[2], "recommendations") {
		return "", false
	}
	switch advisorPathScope(parent) {
	case advisorSubscriptionScope, advisorResourceGroupScope:
		return "", true
	case advisorResourceScope:
		return strings.Join(parent, "/"), true
	default:
		return "", false
	}
}

func (r *Run) advisorEvidence(input AdvisorInput, collected time.Time) (Evidence, bool) {
	subscription := r.Scope.SubscriptionID
	parent, valid := advisorInstanceParent(input.ID, subscription)
	if !valid {
		return Evidence{}, false
	}
	lastRefreshed, refreshValid := advisorSourceTime(input.LastRefreshed)
	lastUpdated, updateValid := advisorSourceTime(input.LastUpdated)
	if !refreshValid || !updateValid {
		return Evidence{}, false
	}
	facts := AdvisorFacts{
		Category:      advisorEnum(input.Category, "HighAvailability", "Security", "Performance", "Cost", "OperationalExcellence"),
		Impact:        advisorEnum(input.Impact, "High", "Medium", "Low"),
		Status:        advisorEnum(input.Status, "New", "Postponed", "Dismissed", "Completed"),
		LastRefreshed: lastRefreshed,
		LastUpdated:   lastUpdated,
	}
	if subscriptionUUID.MatchString(input.TypeID) {
		facts.TypeID = CanonicalID(input.TypeID)
	}

	target := ""
	targetScope := advisorPathScope(scopedARMPath(input.TargetResourceID, subscription))
	switch {
	case targetScope != advisorResourceScope && targetScope != advisorResourceGroupScope:
		facts.AttributionLimitation = "unknown_resource_attribution"
	case parent != "" && !strings.EqualFold(parent, input.TargetResourceID):
		facts.AttributionLimitation = "conflicting_resource_attribution"
	default:
		target = input.TargetResourceID
	}
	return Evidence{
		Source: advisorSource, Operation: AdvisorListOperation, APIVersion: AdvisorAPI,
		EntityID: input.ID, SubscriptionID: subscription, ResourceID: target,
		Kind: "advisor_recommendation", Advisor: &facts, CollectedAt: collected,
	}, true
}

func advisorSourceTime(value *time.Time) (*time.Time, bool) {
	if value == nil || value.IsZero() {
		return nil, true
	}
	utc := utcTime(value)
	// An offset can move a parsed timestamp outside JSON's four-digit year range.
	if year := utc.Year(); year < 0 || year > 9999 {
		return nil, false
	}
	return utc, true
}

func advisorEnum(value string, options ...string) string {
	for _, option := range options {
		if value == option {
			return option
		}
	}
	return advisorUnknown
}

// canonicalAdvisorID chooses one spelling from the same Unicode simple-fold
// classes used by strings.EqualFold. ASCII IDs keep the usual lowercase form.
func canonicalAdvisorID(id string) string {
	return strings.Map(func(r rune) rune {
		canonical := r
		for folded := unicode.SimpleFold(r); folded != r; folded = unicode.SimpleFold(folded) {
			canonical = min(canonical, folded)
		}
		if canonical <= unicode.MaxASCII {
			return unicode.ToLower(canonical)
		}
		return canonical
	}, id)
}

func (r *Run) rebuildAdvisorCoverage() {
	r.AdvisorInstances = nil
	groups := map[string][]Evidence{}
	for _, e := range r.Evidence {
		if e.Source == advisorSource && e.Advisor != nil {
			key := advisorSource + "\x00" + CanonicalID(e.SubscriptionID) + "\x00" + canonicalAdvisorID(e.EntityID)
			groups[key] = append(groups[key], e)
		}
	}
	for key, observations := range groups {
		sort.Slice(observations, func(i, j int) bool { return observations[i].ID < observations[j].ID })
		first := observations[0]
		row := AdvisorInstance{EntityID: canonicalAdvisorID(first.EntityID), EvidenceIDs: make([]string, 0, len(observations))}
		for _, e := range observations {
			row.EvidenceIDs = append(row.EvidenceIDs, e.ID)
		}
		facts, resource, supportingIDs, resolved := summarizeAdvisor(observations)
		row.AdvisorFacts, row.ResourceID = facts, canonicalAdvisorID(resource)
		switch {
		case !resolved:
			r.Gaps = append(r.Gaps, Gap{Source: advisorSource, EntityID: row.EntityID, Reason: "conflicting_advisor_evidence", EvidenceIDs: row.EvidenceIDs})
		case row.Status == advisorUnknown:
			r.Gaps = append(r.Gaps, Gap{Source: advisorSource, EntityID: row.EntityID, ResourceID: row.ResourceID, Reason: "unknown_advisor_status", EvidenceIDs: supportingIDs})
		case row.Status == "New":
			row.FindingID = stableID("finding-", key)
			r.Findings = append(r.Findings, Finding{
				ID: row.FindingID, Source: advisorSource, EntityID: row.EntityID, SubscriptionID: first.SubscriptionID,
				ResourceID: row.ResourceID, Category: "advisor_recommendation", Title: "Azure Advisor New recommendation", EvidenceIDs: supportingIDs,
			})
		}
		if row.AttributionLimitation != "" {
			r.Gaps = append(r.Gaps, Gap{Source: advisorSource, EntityID: row.EntityID, Reason: row.AttributionLimitation, EvidenceIDs: supportingIDs})
		}
		r.AdvisorInstances = append(r.AdvisorInstances, row)
	}
	sort.Slice(r.AdvisorInstances, func(i, j int) bool {
		return r.AdvisorInstances[i].EntityID < r.AdvisorInstances[j].EntityID
	})
}

// The state-change time can order conflicting facts. Evaluation and collection
// times provide context without deciding which version applies.
func summarizeAdvisor(observations []Evidence) (AdvisorFacts, string, []string, bool) {
	latest := observations[0]
	for _, e := range observations[1:] {
		if afterAdvisorTime(e.Advisor.LastUpdated, latest.Advisor.LastUpdated) {
			latest = e
		}
	}
	changed := false
	for _, e := range observations {
		if sameAdvisorVersion(e, latest) {
			continue
		}
		changed = true
		if e.Advisor.LastUpdated == nil || !afterAdvisorTime(latest.Advisor.LastUpdated, e.Advisor.LastUpdated) {
			return AdvisorFacts{Category: advisorUnknown, Impact: advisorUnknown, Status: advisorUnknown}, "", nil, false
		}
	}
	facts := *latest.Advisor
	ids := make([]string, 0, len(observations))
	for _, e := range observations {
		if changed && (e.Advisor.LastUpdated == nil || !e.Advisor.LastUpdated.Equal(*facts.LastUpdated)) {
			continue
		}
		ids = append(ids, e.ID)
		if afterAdvisorTime(e.Advisor.LastRefreshed, facts.LastRefreshed) {
			facts.LastRefreshed = e.Advisor.LastRefreshed
		}
	}
	return facts, latest.ResourceID, ids, true
}

func sameAdvisorVersion(a, b Evidence) bool {
	x, y := *a.Advisor, *b.Advisor
	x.LastRefreshed, x.LastUpdated = nil, nil
	y.LastRefreshed, y.LastUpdated = nil, nil
	return x == y && strings.EqualFold(a.ResourceID, b.ResourceID)
}

func afterAdvisorTime(a, b *time.Time) bool {
	return a != nil && (b == nil || a.After(*b))
}

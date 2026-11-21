package health

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"
)

var subscriptionUUID = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// ResourceID validates a scoped ARM resource ID, including nested and extension
// resources. Keep source spelling in the evidence.
func ResourceID(id, subscription string) bool {
	parts := scopedARMPath(id, subscription)
	if len(parts) < 7 {
		return false
	}
	i := 3
	if strings.EqualFold(parts[i], "resourceGroups") {
		i += 2
	}
	if i+3 >= len(parts) || !strings.EqualFold(parts[i], "providers") {
		return false
	}
	i += 2 // provider namespace
	for i < len(parts) {
		if strings.EqualFold(parts[i], "providers") {
			if i+3 >= len(parts) {
				return false
			}
			i += 2
		}
		if i+1 >= len(parts) {
			return false
		}
		i += 2
	}
	return true
}

// scopedARMPath applies the subscription and privacy checks common to ARM IDs.
// Callers validate the resource shape for their source.
func scopedARMPath(id, subscription string) []string {
	if !subscriptionUUID.MatchString(subscription) || strings.ContainsAny(id, `\%?#`) {
		return nil
	}
	for _, r := range id {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return nil
		}
	}
	parts := strings.Split(id, "/")
	if len(parts) < 3 || parts[0] != "" || !strings.EqualFold(parts[1], "subscriptions") || !strings.EqualFold(parts[2], subscription) {
		return nil
	}
	for _, p := range parts[1:] {
		if p == "" || p == "." || p == ".." {
			return nil
		}
	}
	safe, n := SafeText(id)
	if safe != id || n != 0 {
		return nil
	}
	return parts
}
func CanonicalID(id string) string { return strings.ToLower(id) }

// canonicalFoldID uses the same equivalence classes as strings.EqualFold.
// Keep lowercase spellings where they belong to that same class.
func canonicalFoldID(id string) string {
	return strings.Map(func(r rune) rune {
		if r <= unicode.MaxASCII {
			return unicode.ToLower(r)
		}
		first := simpleFoldRepresentative(r)
		lower := unicode.ToLower(first)
		if strings.EqualFold(string(first), string(lower)) {
			return lower
		}
		return first
	}, id)
}

func (r *Run) identityKey(id string) string {
	if r.SchemaVersion >= AssessmentSchemaVersion {
		return canonicalFoldID(id)
	}
	return CanonicalID(id)
}

func statusResourceID(id, subscription string) (string, bool) {
	// Only the documented final ResourceHealth extension can establish attribution.
	parts := strings.Split(id, "/")
	if len(parts) < 5 {
		return "", false
	}
	n := len(parts)
	if !strings.EqualFold(parts[n-4], "providers") || !strings.EqualFold(parts[n-3], "Microsoft.ResourceHealth") || !strings.EqualFold(parts[n-2], "availabilityStatuses") || parts[n-1] == "" || parts[n-1] == "." || parts[n-1] == ".." {
		return "", false
	}
	// A historic status name may contain an escaped timestamp, but cannot change
	// the attributed resource. SafeText still filters the retained full status ID.
	statusName, err := url.PathUnescape(parts[n-1])
	if err != nil || statusName == "." || statusName == ".." || strings.ContainsAny(statusName, `/\?#`) {
		return "", false
	}
	safeName, redactions := SafeText(statusName)
	if redactions != 0 || safeName != statusName {
		return "", false
	}
	resource := strings.Join(parts[:n-4], "/")
	clean, count := SafeText(id)
	return resource, ResourceID(resource, subscription) && count == 0 && clean == id && !strings.ContainsAny(parts[n-1], `\?#`)
}

func stableID(prefix string, v any) string {
	b, _ := json.Marshal(v)
	sum := sha256.Sum256(b)
	return prefix + hex.EncodeToString(sum[:])
}
func (r *Run) addEvidence(e Evidence) {
	// Collection times are excluded from identity so repeated observations share an ID.
	identity := e
	identity.ID = ""
	identity.CollectedAt = time.Time{}
	identity.LastCollectedAt = nil
	identity.ResourceID = r.identityKey(identity.ResourceID)
	identity.StatusID = r.identityKey(identity.StatusID)
	identity.EntityID = r.identityKey(identity.EntityID)
	if r.SchemaVersion >= AssessmentSchemaVersion && identity.Resource != nil {
		facts := *identity.Resource
		facts.ID = canonicalFoldID(facts.ID)
		facts.ResourceGroup = canonicalFoldID(facts.ResourceGroup)
		identity.Resource = &facts
	}
	if e.Source == advisorSource {
		identity.EntityID = canonicalAdvisorID(e.EntityID)
		if r.SchemaVersion < AssessmentSchemaVersion {
			identity.ResourceID = canonicalAdvisorID(e.ResourceID)
		}
	}
	identity.SubscriptionID = CanonicalID(identity.SubscriptionID)
	e.ID = stableID("ev-", identity)
	for i := range r.Evidence {
		prior := &r.Evidence[i]
		if prior.ID == e.ID {
			if r.SchemaVersion >= AssessmentSchemaVersion && e.Source == "service_health" && e.CollectedAt.After(prior.LatestCollectionTime()) {
				prior.LastCollectedAt = &e.CollectedAt
			}
			return
		}
	}
	r.Evidence = append(r.Evidence, e)
}

// NormalizePage must run before Save, including for pages with mixed valid and
// invalid records. Rejected records leave a count and any independently
// verified scope, without retaining the rejected payload.
func (r *Run) NormalizePage(source string, page Page) {
	subscription := r.Scope.SubscriptionID
	collected := page.CollectedAt.UTC()
	if collected.IsZero() {
		collected = time.Now().UTC()
	}
	invalid, unclassified := 0, 0
	var rejectedScopes []RejectedRecordInput
	reject := func(input RejectedRecordInput) {
		invalid++
		if input.ID != "" {
			rejectedScopes = append(rejectedScopes, input)
		}
	}
	text := func(s string) string { v, n := SafeText(s); r.Redactions += n; return v }
	switch source {
	case "inventory":
		for _, input := range page.Resources {
			if !ResourceID(input.ID, subscription) {
				invalid++
				continue
			}
			tags, n := SafeTags(input.Tags)
			r.Redactions += n
			rg := ""
			parts := strings.Split(input.ID, "/")
			if strings.EqualFold(parts[3], "resourceGroups") {
				rg = parts[4]
			}
			facts := ResourceFacts{ID: input.ID, Name: text(input.Name), Type: text(input.Type), Location: text(input.Location), ResourceGroup: rg, Tags: tags}
			r.addEvidence(Evidence{Source: source, Operation: inventoryOperation, APIVersion: InventoryAPI, ResourceID: input.ID, Kind: "inventory", CollectedAt: collected, Resource: &facts})
		}
	case "resource_health":
		for _, input := range page.Health {
			id, valid := statusResourceID(input.StatusID, subscription)
			if !valid {
				invalid++
				continue
			}
			reported, reportedValid := sourceTime(input.ReportedAt)
			occurred, occurredValid := sourceTime(input.OccurredAt)
			if !reportedValid || !occurredValid {
				reject(RejectedRecordInput{ID: input.StatusID})
				continue
			}
			state, kind := "undetermined", "undetermined"
			if input.PropertiesPresent && input.State != nil {
				switch *input.State {
				case "Available", "Unavailable", "Degraded", "Unknown":
					state, kind = *input.State, "availability"
				}
			}
			events := make([]string, 0, len(input.EventIDs))
			for _, event := range input.EventIDs {
				events = append(events, text(event))
			}
			sort.Strings(events)
			events = compact(events)
			eventID := text(input.EventID)
			if state == "undetermined" && (input.EventPresent || eventID != "" || len(events) > 0) {
				kind = "event"
			}
			statusName, _ := url.PathUnescape(input.StatusID[strings.LastIndex(input.StatusID, "/")+1:])
			role := "historical"
			if strings.EqualFold(statusName, "current") {
				role = "current"
			}
			r.addEvidence(Evidence{Source: source, StatusRole: role, Operation: resourceHealthOperation, APIVersion: ResourceHealthAPI, ResourceID: id, StatusID: input.StatusID, State: state, Kind: kind, EventID: eventID, EventIDs: events, CollectedAt: collected, ReportedAt: reported, OccurredAt: occurred})
		}
	case "service_health":
		for _, input := range page.ServiceHealth {
			if e, valid := r.serviceEvidence(input, collected); valid {
				r.addOperational(e)
			} else {
				invalid++
			}
		}
	case "azure_monitor":
		for _, input := range page.Monitor {
			if e, valid := r.monitorEvidence(input, collected); valid {
				r.addOperational(e)
			} else {
				reject(RejectedRecordInput{ID: input.ID, TargetResourceID: input.TargetResourceID})
			}
		}
	case advisorSource:
		for _, input := range page.Advisor {
			if e, valid := r.advisorEvidence(input, collected); valid {
				if e.Advisor.Status == advisorUnknown {
					unclassified++
				}
				r.addEvidence(e)
			} else {
				reject(RejectedRecordInput{ID: input.ID, TargetResourceID: input.TargetResourceID})
			}
		}
	}
	s := r.Sources[source]
	if source == advisorSource {
		s.UnclassifiedRecords += unclassified
	}
	if source == advisorSource || r.SchemaVersion >= AssessmentSchemaVersion {
		if s.FirstPageAt == nil {
			s.FirstPageAt = &collected
		}
		s.LastPageAt = &collected
	}
	s.Pages++
	if invalid > 0 {
		s.Reason = "invalid_records"
		s.RejectedRecords += invalid
		if r.SchemaVersion >= AssessmentSchemaVersion {
			// Transport hints accompany placeholders and cannot also describe
			// rows rejected here after decoding.
			if len(page.RejectedScopes) <= invalid-len(rejectedScopes) {
				rejectedScopes = append(rejectedScopes, page.RejectedScopes...)
			}
			for _, input := range rejectedScopes {
				if scope, ok := scopedRejection(source, subscription, input); ok {
					s.RejectedScopes = append(s.RejectedScopes, scope)
				}
			}
		}
	}
	r.Sources[source] = s
	r.RebuildCoverage()
}
func compact(values []string) []string {
	result := values[:0]
	for _, v := range values {
		if len(result) == 0 || result[len(result)-1] != v {
			result = append(result, v)
		}
	}
	return result
}

// RebuildCoverage derives findings, coverage and assessments from saved evidence.
func (r *Run) RebuildCoverage() {
	r.Coverage = "incomplete"
	r.Resources = []Resource{}
	r.Observations = []Observation{}
	r.Findings = []Finding{}
	r.Gaps = []Gap{}
	var proof *assessmentEvidence
	if r.SchemaVersion >= AssessmentSchemaVersion {
		proof = &assessmentEvidence{bySource: make(map[string]map[string][]string), postponed: make(map[string][]string)}
	}
	resourceIndex := map[string]int{}
	inventoryConflicts := map[string]bool{}
	inventoryRepresentative := map[string]string{}
	healthByResource := map[string][]Evidence{}
	gap := func(source, id, reason string, evidence []string) {
		r.Gaps = append(r.Gaps, Gap{Source: source, ResourceID: id, Reason: reason, EvidenceIDs: evidence, Blocking: proof != nil})
	}
	for _, source := range assessmentSources {
		s := r.Sources[source]
		unscopedRejections, sourceUnfinished := s.RejectedRecords, s.Status != "completed"
		if proof != nil {
			unscopedRejections -= len(s.RejectedScopes)
			sourceUnfinished = sourceUnfinished && !s.finishedWithRejections()
			r.rebuildRejectionGaps(source, s.RejectedScopes)
		}
		if unscopedRejections > 0 && (s.Reason != "invalid_records" || !sourceUnfinished) {
			gap(source, "", "invalid_records", nil)
		}
		if sourceUnfinished {
			reason := s.Reason
			if reason == "" {
				reason = s.Status
			}
			if reason == "" {
				reason = "missing_source_status"
			}
			gap(source, "", reason, nil)
		}
	}
	if proof != nil {
		r.sourceDecisionGaps()
	}
	for _, e := range r.Evidence {
		key := r.identityKey(e.ResourceID)
		if e.Source == "inventory" && e.Resource != nil {
			i, found := resourceIndex[key]
			if !found {
				resourceIndex[key] = len(r.Resources)
				r.Resources = append(r.Resources, Resource{ResourceFacts: *e.Resource, EvidenceIDs: []string{e.ID}})
				if proof != nil {
					inventoryRepresentative[key] = e.ID
				}
			} else {
				resource := &r.Resources[i]
				if proof != nil {
					inventoryConflicts[key] = inventoryConflicts[key] || !sameInventoryFacts(resource.ResourceFacts, *e.Resource)
					if e.ID < inventoryRepresentative[key] {
						resource.ResourceFacts = *e.Resource
						inventoryRepresentative[key] = e.ID
					}
				}
				resource.EvidenceIDs = append(resource.EvidenceIDs, e.ID)
				if proof == nil {
					gap("inventory", e.ResourceID, "conflicting_inventory", resource.EvidenceIDs)
				}
			}
		} else if e.Source == "resource_health" {
			healthByResource[key] = append(healthByResource[key], e)
		}
	}
	if proof == nil && r.Sources["inventory"].Status != "completed" {
		gap("inventory", "", "inventory_unverified", nil)
	}
	for i := range r.Resources {
		resource := &r.Resources[i]
		key := r.identityKey(resource.ID)
		if proof != nil {
			// Keep derived identities stable and retain source spelling in evidence.
			resource.ID = key
			sort.Strings(resource.EvidenceIDs)
			if inventoryConflicts[key] {
				gap("inventory", resource.ID, "conflicting_inventory", resource.EvidenceIDs)
			}
		}
		if len(healthByResource[key]) == 0 {
			reason := "missing_health_evidence"
			if r.Sources["resource_health"].Status != "completed" {
				reason = "health_not_checked"
			}
			gap("resource_health", resource.ID, reason, resource.EvidenceIDs)
			r.Observations = append(r.Observations, Observation{ResourceID: resource.ID, State: "not_checked", InInventory: true, EvidenceIDs: []string{}})
		}
	}
	for key, evidence := range healthByResource {
		id := evidence[0].ResourceID
		index, inInventory := resourceIndex[key]
		if proof != nil {
			if inInventory {
				id = r.Resources[index].ID
			} else {
				for _, e := range evidence {
					id = min(id, e.ResourceID)
				}
			}
		}
		ids := make([]string, 0, len(evidence))
		current := make([]Evidence, 0, len(evidence))
		var historicalIDs []string
		for _, e := range evidence {
			ids = append(ids, e.ID)
			if e.StatusRole == "current" {
				current = append(current, e)
			} else {
				historicalIDs = append(historicalIDs, e.ID)
			}
		}
		sort.Strings(historicalIDs)
		sort.Strings(ids)
		if len(historicalIDs) > 0 {
			r.Gaps = append(r.Gaps, Gap{Source: "resource_health", ResourceID: id, Reason: "historical_health_evidence", EvidenceIDs: historicalIDs})
		}
		if len(current) == 0 {
			gap("resource_health", id, "missing_current_health_evidence", ids)
		}
		state, deciding := decideResourceHealth(current, r.SchemaVersion)
		if len(current) > 0 && len(deciding) == 0 {
			gap("resource_health", id, "conflicting_health_evidence", evidenceRefs(current))
		}
		r.Observations = append(r.Observations, Observation{ResourceID: id, State: state, InInventory: inInventory, EvidenceIDs: ids})
		if proof != nil {
			proof.add("resource_health", id, deciding)
		}
		if !inInventory {
			gap("inventory", id, "health_resource_absent_from_inventory", ids)
		}
		switch state {
		case "Available", "Degraded", "Unavailable":
		case "Unknown":
			if proof == nil {
				gap("resource_health", id, "source_unknown", ids)
			} else {
				gap("resource_health", id, "source_unknown", deciding)
			}
		default:
			switch {
			case proof == nil && state == "undetermined":
				gap("resource_health", id, "undetermined_health_evidence", ids)
			case proof != nil && len(deciding) > 0:
				gap("resource_health", id, "undetermined_health_evidence", deciding)
			}
		}
		for _, e := range current {
			category, title := "", ""
			switch e.State {
			case "Unavailable", "Degraded":
				category, title = "resource_health_issue", "Resource Health reported "+e.State
			}
			if e.Kind == "event" {
				category, title = "resource_health_event", "Resource Health impacting event; current state undetermined"
			}
			if category != "" {
				r.Findings = append(r.Findings, Finding{ID: stableID("finding-", e.ID), ResourceID: e.ResourceID, Category: category, Title: title, EvidenceIDs: []string{e.ID}})
			}
		}
	}
	r.rebuildOperationalCoverage(proof)
	r.rebuildAdvisorCoverage(proof)
	legacyEmpty := len(healthByResource) == 0 && r.Sources["resource_health"].Status == "completed"
	if len(r.Resources) == 0 && r.Sources["inventory"].Status == "completed" && (proof != nil || legacyEmpty) {
		gap("inventory", "", "no_resources_observed", nil)
	}
	sort.Slice(r.Resources, func(i, j int) bool { return r.identityKey(r.Resources[i].ID) < r.identityKey(r.Resources[j].ID) })
	sort.Slice(r.Observations, func(i, j int) bool {
		return r.identityKey(r.Observations[i].ResourceID) < r.identityKey(r.Observations[j].ResourceID)
	})
	sort.Slice(r.Findings, func(i, j int) bool {
		a, b := r.Findings[i], r.Findings[j]
		return a.Category+a.ResourceID+a.ID < b.Category+b.ResourceID+b.ID
	})
	if proof != nil {
		r.rebuildAssessments(proof, resourceIndex)
	} else {
		r.sortGaps()
	}
}

// decideResourceHealth selects deciding records from the current status versions.
// When states disagree, every version needs a source time and the newest must agree.
func decideResourceHealth(current []Evidence, schemaVersion int) (string, []string) {
	if len(current) == 0 {
		return "undetermined", nil
	}
	state := current[0].State
	agree := true
	for _, e := range current[1:] {
		if e.State != state {
			agree = false
			break
		}
	}
	if agree {
		return state, evidenceRefs(current)
	}
	var latest *time.Time
	tied := false
	for _, e := range current {
		if e.ReportedAt == nil || schemaVersion >= AssessmentSchemaVersion && e.ReportedAt.IsZero() {
			return "undetermined", nil
		}
		if latest == nil || e.ReportedAt.After(*latest) {
			latest, state, tied = e.ReportedAt, e.State, false
		} else if e.ReportedAt.Equal(*latest) && e.State != state {
			tied = true
		}
	}
	if tied {
		return "undetermined", nil
	}
	var deciding []string
	for _, e := range current {
		if e.ReportedAt.Equal(*latest) {
			deciding = append(deciding, e.ID)
		}
	}
	sort.Strings(deciding)
	return state, deciding
}

func (r *Run) sortGaps() {
	sort.SliceStable(r.Gaps, func(i, j int) bool {
		a, b := r.Gaps[i], r.Gaps[j]
		return a.Source+"\x00"+a.ResourceID+"\x00"+a.EntityID+"\x00"+a.Reason+"\x00"+strings.Join(a.EvidenceIDs, ",") < b.Source+"\x00"+b.ResourceID+"\x00"+b.EntityID+"\x00"+b.Reason+"\x00"+strings.Join(b.EvidenceIDs, ",")
	})
}

func sourceTime(value *time.Time) (*time.Time, bool) {
	if value == nil {
		return nil, true
	}
	utc := value.UTC()
	// An offset can move a parsed timestamp outside JSON's four-digit year range.
	if year := utc.Year(); year < 0 || year > 9999 {
		return nil, false
	}
	return &utc, true
}

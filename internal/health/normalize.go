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
// resources. Keep source spelling; use CanonicalID for case-insensitive joins.
func ResourceID(id, subscription string) bool {
	if !subscriptionUUID.MatchString(subscription) || strings.ContainsAny(id, `\%?#`) {
		return false
	}
	for _, r := range id {
		if unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
	}
	parts := strings.Split(id, "/")
	if len(parts) < 7 || parts[0] != "" || !strings.EqualFold(parts[1], "subscriptions") || !strings.EqualFold(parts[2], subscription) {
		return false
	}
	for _, p := range parts[1:] {
		if p == "" || p == "." || p == ".." {
			return false
		}
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
	safe, n := SafeText(id)
	return safe == id && n == 0
}
func CanonicalID(id string) string { return strings.ToLower(id) }

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
	// Collection time is intentionally excluded from identity: exact observations
	// repeated on another page collapse, keeping their first collection time.
	identity := e
	identity.ID = ""
	identity.CollectedAt = time.Time{}
	identity.ResourceID = CanonicalID(identity.ResourceID)
	identity.StatusID = CanonicalID(identity.StatusID)
	e.ID = stableID("ev-", identity)
	for _, prior := range r.Evidence {
		if prior.ID == e.ID {
			return
		}
	}
	r.Evidence = append(r.Evidence, e)
}

// NormalizePage must run before Save, including for pages with mixed valid and
// invalid records. Rejected records leave only a fixed diagnostic, no payload.
func (r *Run) NormalizePage(source string, page Page) {
	subscription := r.Scope.SubscriptionID
	collected := page.CollectedAt.UTC()
	if collected.IsZero() {
		collected = time.Now().UTC()
	}
	invalid := 0
	text := func(s string) string { v, n := SafeText(s); r.Redactions += n; return v }
	if source == "inventory" {
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
			r.addEvidence(Evidence{Source: source, Operation: "Resources.List", APIVersion: InventoryAPI, ResourceID: input.ID, Kind: "inventory", CollectedAt: collected, Resource: &facts})
		}
	} else {
		for _, input := range page.Health {
			id, valid := statusResourceID(input.StatusID, subscription)
			if !valid {
				invalid++
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
			var reported *time.Time
			if input.ReportedAt != nil {
				t := input.ReportedAt.UTC()
				reported = &t
			}
			statusName, _ := url.PathUnescape(input.StatusID[strings.LastIndex(input.StatusID, "/")+1:])
			role := "historical"
			if strings.EqualFold(statusName, "current") {
				role = "current"
			}
			r.addEvidence(Evidence{Source: source, StatusRole: role, Operation: "AvailabilityStatuses.ListBySubscriptionId", APIVersion: ResourceHealthAPI, ResourceID: id, StatusID: input.StatusID, State: state, Kind: kind, EventID: eventID, EventIDs: events, CollectedAt: collected, ReportedAt: reported, OccurredAt: utcTime(input.OccurredAt)})
		}
	}
	s := r.Sources[source]
	s.Pages++
	if invalid > 0 {
		s.Reason = "invalid_records"
		s.RejectedRecords += invalid
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

// RebuildCoverage derives all views from saved normalized evidence. No field
// expresses overall OK; source Available and execution success remain separate.
func (r *Run) RebuildCoverage() {
	r.Coverage = "incomplete"
	r.Resources = []Resource{}
	r.Observations = []Observation{}
	r.Findings = []Finding{}
	r.Gaps = []Gap{}
	resourceIndex := map[string]int{}
	healthByResource := map[string][]Evidence{}
	gap := func(source, id, reason string, evidence []string) {
		r.Gaps = append(r.Gaps, Gap{Source: source, ResourceID: id, Reason: reason, EvidenceIDs: evidence})
	}
	for _, source := range []string{"inventory", "resource_health", "advisor", "service_health", "azure_monitor"} {
		s := r.Sources[source]
		if s.RejectedRecords > 0 && s.Reason != "invalid_records" {
			gap(source, "", "invalid_records", nil)
		}
		if s.Status != "completed" {
			reason := s.Reason
			if reason == "" {
				reason = s.Status
			}
			gap(source, "", reason, nil)
		}
	}
	for _, e := range r.Evidence {
		key := CanonicalID(e.ResourceID)
		if e.Source == "inventory" && e.Resource != nil {
			i, found := resourceIndex[key]
			if !found {
				resourceIndex[key] = len(r.Resources)
				r.Resources = append(r.Resources, Resource{ResourceFacts: *e.Resource, EvidenceIDs: []string{e.ID}})
			} else {
				r.Resources[i].EvidenceIDs = append(r.Resources[i].EvidenceIDs, e.ID)
				gap("inventory", e.ResourceID, "conflicting_inventory", r.Resources[i].EvidenceIDs)
			}
		} else if e.Source == "resource_health" {
			healthByResource[key] = append(healthByResource[key], e)
		}
	}
	if r.Sources["inventory"].Status != "completed" {
		gap("inventory", "", "inventory_unverified", nil)
	}
	for _, resource := range r.Resources {
		if len(healthByResource[CanonicalID(resource.ID)]) == 0 {
			reason := "missing_health_evidence"
			if r.Sources["resource_health"].Status != "completed" {
				reason = "health_not_checked"
			}
			gap("resource_health", resource.ID, reason, resource.EvidenceIDs)
			r.Observations = append(r.Observations, Observation{ResourceID: resource.ID, State: "not_checked", InInventory: true, EvidenceIDs: []string{}})
		}
	}
	for key, evidence := range healthByResource {
		ids := make([]string, 0, len(evidence))
		states := map[string]bool{}
		current := make([]Evidence, 0, len(evidence))
		currentIDs, historicalIDs := []string{}, []string{}
		for _, e := range evidence {
			ids = append(ids, e.ID)
			if e.StatusRole == "current" {
				current = append(current, e)
				currentIDs = append(currentIDs, e.ID)
				states[e.State] = true
			} else {
				historicalIDs = append(historicalIDs, e.ID)
			}
		}
		sort.Strings(currentIDs)
		sort.Strings(historicalIDs)
		if len(historicalIDs) > 0 {
			gap("resource_health", evidence[0].ResourceID, "historical_health_evidence", historicalIDs)
		}
		if len(current) == 0 {
			gap("resource_health", evidence[0].ResourceID, "missing_current_health_evidence", ids)
		}
		sort.Strings(ids)
		state := "undetermined"
		if len(current) > 0 {
			state = current[0].State
		}
		if len(states) > 1 {
			state = "undetermined"
			// A unique latest timestamp resolves current state only when every competing
			// observation has a source timestamp and no latest states conflict.
			var latest *time.Time
			latestState := ""
			allTimed, tied := true, false
			for _, e := range current {
				if e.ReportedAt == nil {
					allTimed = false
					break
				}
				if latest == nil || e.ReportedAt.After(*latest) {
					latest, latestState, tied = e.ReportedAt, e.State, false
				} else if e.ReportedAt.Equal(*latest) && e.State != latestState {
					tied = true
				}
			}
			if allTimed && !tied {
				state = latestState
			} else {
				gap("resource_health", evidence[0].ResourceID, "conflicting_health_evidence", currentIDs)
			}
		}
		_, inInventory := resourceIndex[key]
		id := evidence[0].ResourceID
		r.Observations = append(r.Observations, Observation{ResourceID: id, State: state, InInventory: inInventory, EvidenceIDs: ids})
		if !inInventory {
			gap("inventory", id, "health_resource_absent_from_inventory", ids)
		}
		if state == "Unknown" {
			gap("resource_health", id, "source_unknown", ids)
		}
		if state == "undetermined" {
			gap("resource_health", id, "undetermined_health_evidence", ids)
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
	if len(r.Evidence) == 0 && r.Sources["inventory"].Status == "completed" && r.Sources["resource_health"].Status == "completed" {
		gap("inventory", "", "no_resources_observed", nil)
	}
	sort.Slice(r.Resources, func(i, j int) bool { return CanonicalID(r.Resources[i].ID) < CanonicalID(r.Resources[j].ID) })
	sort.Slice(r.Observations, func(i, j int) bool {
		return CanonicalID(r.Observations[i].ResourceID) < CanonicalID(r.Observations[j].ResourceID)
	})
	sort.Slice(r.Findings, func(i, j int) bool {
		a, b := r.Findings[i], r.Findings[j]
		return a.Category+a.ResourceID+a.ID < b.Category+b.ResourceID+b.ID
	})
	sort.Slice(r.Gaps, func(i, j int) bool {
		a, b := r.Gaps[i], r.Gaps[j]
		return a.Source+a.ResourceID+a.Reason < b.Source+b.ResourceID+b.Reason
	})
}

func utcTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	t := value.UTC()
	return &t
}

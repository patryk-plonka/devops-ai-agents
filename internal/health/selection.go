package health

import (
	"errors"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"
)

var findingIDPattern = regexp.MustCompile(`^finding-[0-9a-f]{64}$`)
var evidenceIDPattern = regexp.MustCompile(`^ev-[0-9a-f]{64}$`)

// ValidFindingID checks the full local identifier before it is used as a
// selection, trace reference or part of a filename.
func ValidFindingID(id string) bool { return findingIDPattern.MatchString(id) }

// DraftSelection is a batch of findings resolved from one reviewed revision.
type DraftSelection struct {
	RunID    string
	Revision int
	Outcome  string
	Coverage string
	Findings []SelectedFinding
}

type SelectedFinding struct {
	Finding                  Finding
	Source                   string
	Collection               Source
	QueryPolicyVerified      bool
	Supporting               []Evidence
	Context                  []Evidence
	OperationalStateConflict bool
	FuturePlannedMaintenance bool
	CurrentHealth            *CurrentResourceHealth
	Advisor                  *AdvisorFacts
	TargetName               string
	TargetType               string
	GroupTarget              bool
	TargetEvidenceIDs        []string
	Gaps                     []SelectionGap
}

type CurrentResourceHealth struct {
	State             string
	Observations      []Evidence
	Historical        bool
	Conflict          bool
	MissingSourceTime bool
}

// SelectionGap omits entity, resource and other finding IDs. Broader gaps may
// apply to the selection without providing evidence about its particular target.
type SelectionGap struct {
	Source      string
	Reason      string
	Blocking    bool
	Broader     bool
	EvidenceIDs []string
}

type selectionIndex struct {
	evidence  map[string]Evidence
	health    map[string][]Evidence
	inventory map[string][]Evidence
	byTarget  map[string][]int
	byEntity  map[string][]int
	byFinding map[string][]int
	advisor   map[string][]AdvisorInstance
}

// ResolveDraftSelection validates the whole request before returning any
// selection. Pass the snapshot of a validated, published report.
func (r Run) ResolveDraftSelection(ids []string) (DraftSelection, error) {
	if r.SchemaVersion != AssessmentSchemaVersion || len(ids) == 0 {
		return DraftSelection{}, errors.New("select at least one finding from a schema 4 snapshot")
	}
	var unique []string
	seen := make(map[string]bool, len(ids))
	for _, id := range ids {
		if !ValidFindingID(id) {
			return DraftSelection{}, errors.New("invalid finding ID")
		}
		if !seen[id] {
			seen[id] = true
			unique = append(unique, id)
		}
	}
	if r.Scope == nil || !subscriptionUUID.MatchString(r.Scope.SubscriptionID) {
		return DraftSelection{}, errors.New("finding subscription scope is unresolved")
	}

	saved := indexFindings(r.Findings)
	index, err := indexSelection(r)
	if err != nil {
		return DraftSelection{}, err
	}
	// Re-derive from the normalized evidence using the same source decisions as
	// collection. RebuildCoverage replaces the derived slices on this copy.
	derived := r
	derived.RebuildCoverage()
	expected := indexFindings(derived.Findings)
	expectedAdvisor := indexAdvisorFindings(derived.AdvisorInstances)
	selection := DraftSelection{RunID: r.ID, Revision: r.Revision, Outcome: r.Outcome, Coverage: r.Coverage}
	for _, id := range unique {
		if len(saved[id]) != 1 {
			return DraftSelection{}, errors.New("selected finding is missing or ambiguous")
		}
		if len(expected[id]) != 1 || !sameSelectionFinding(saved[id][0], expected[id][0]) {
			return DraftSelection{}, errors.New("selected finding does not match its supporting evidence")
		}
		f := expected[id][0]
		item, err := index.resolve(r, f, expectedAdvisor[id])
		if err != nil {
			return DraftSelection{}, err
		}
		selection.Findings = append(selection.Findings, item)
	}
	return selection, nil
}

func indexFindings(findings []Finding) map[string][]Finding {
	index := make(map[string][]Finding, len(findings))
	for _, f := range findings {
		index[f.ID] = append(index[f.ID], f)
	}
	return index
}

func indexAdvisorFindings(rows []AdvisorInstance) map[string][]AdvisorInstance {
	index := make(map[string][]AdvisorInstance)
	for _, row := range rows {
		if row.FindingID != "" {
			index[row.FindingID] = append(index[row.FindingID], row)
		}
	}
	return index
}

func selectionEntityKey(source, entity string) string {
	return source + "\x00" + canonicalFoldID(entity)
}

func indexSelection(r Run) (selectionIndex, error) {
	index := selectionIndex{
		evidence:  make(map[string]Evidence, len(r.Evidence)),
		health:    make(map[string][]Evidence),
		inventory: make(map[string][]Evidence),
		byTarget:  make(map[string][]int),
		byEntity:  make(map[string][]int),
		byFinding: make(map[string][]int),
		advisor:   indexAdvisorFindings(r.AdvisorInstances),
	}
	for _, e := range r.Evidence {
		if _, exists := index.evidence[e.ID]; exists {
			return selectionIndex{}, errors.New("ambiguous evidence ID")
		}
		index.evidence[e.ID] = e
		target := canonicalFoldID(e.ResourceID)
		switch {
		case e.Source == "resource_health" && e.StatusRole == "current":
			index.health[target] = append(index.health[target], e)
		case e.Source == "inventory" && e.Resource != nil && strings.EqualFold(e.Resource.ID, e.ResourceID):
			index.inventory[target] = append(index.inventory[target], e)
		}
	}
	for i, g := range r.Gaps {
		target := canonicalFoldID(g.ResourceID)
		index.byTarget[target] = append(index.byTarget[target], i)
		if g.EntityID != "" {
			key := selectionEntityKey(g.Source, g.EntityID)
			index.byEntity[key] = append(index.byEntity[key], i)
		}
		if g.FindingID != "" {
			index.byFinding[g.FindingID] = append(index.byFinding[g.FindingID], i)
		}
	}
	return index, nil
}

func sameSelectionFinding(saved, expected Finding) bool {
	return saved.ID == expected.ID && saved.Category == expected.Category && saved.Source == expected.Source &&
		strings.EqualFold(saved.ResourceID, expected.ResourceID) && strings.EqualFold(saved.EntityID, expected.EntityID) &&
		strings.EqualFold(saved.SubscriptionID, expected.SubscriptionID) &&
		slices.Equal(sortedUnique(slices.Clone(saved.EvidenceIDs)), expected.EvidenceIDs) &&
		len(saved.EvidenceIDs) == len(expected.EvidenceIDs)
}

func (x selectionIndex) resolve(r Run, f Finding, expectedAdvisor []AdvisorInstance) (SelectedFinding, error) {
	support, err := x.resolveEvidence(f.EvidenceIDs)
	if err != nil {
		return SelectedFinding{}, err
	}
	source := support[0].Source
	subscription := r.Scope.SubscriptionID
	if source != "resource_health" && !strings.EqualFold(f.SubscriptionID, subscription) {
		return SelectedFinding{}, errors.New("finding does not apply to the selected subscription")
	}
	resourceTarget := ResourceID(f.ResourceID, subscription)
	groupTarget := source == advisorSource && advisorPathScope(scopedARMPath(f.ResourceID, subscription)) == advisorResourceGroupScope
	if f.ResourceID != "" && !resourceTarget && !groupTarget {
		return SelectedFinding{}, errors.New("finding target is outside the selected resource scope")
	}

	s := r.Sources[source]
	s.RejectedScopes = nil // The per-row rejected targets are private run-wide details.
	item := SelectedFinding{
		Finding: f, Source: source, Collection: s, QueryPolicyVerified: r.matchesAssessmentQuery(source, s.Query),
		Supporting: support, GroupTarget: groupTarget,
	}
	target := canonicalFoldID(f.ResourceID)
	if resourceTarget {
		item.TargetName, item.TargetType, item.TargetEvidenceIDs = selectionTarget(x.inventory[target])
		if _, err := x.resolveEvidence(item.TargetEvidenceIDs); err != nil {
			return SelectedFinding{}, err
		}
	}
	if source == "resource_health" {
		item.CurrentHealth, err = x.currentHealth(f, target, r.SchemaVersion)
		if err != nil {
			return SelectedFinding{}, err
		}
	}
	var contextIDs []string
	if source == advisorSource {
		rows := x.advisor[f.ID]
		if len(rows) != 1 || len(expectedAdvisor) != 1 || !reflect.DeepEqual(rows[0], expectedAdvisor[0]) {
			return SelectedFinding{}, errors.New("selected finding does not match its saved Advisor instance")
		}
		row := rows[0]
		item.Advisor = &row.AdvisorFacts
		for _, id := range row.EvidenceIDs {
			e := x.evidence[id]
			if e.Advisor.LastUpdated != nil && afterAdvisorTime(row.LastUpdated, e.Advisor.LastUpdated) &&
				(e.Advisor.Status != row.Status || !strings.EqualFold(e.ResourceID, row.ResourceID) ||
					e.Advisor.AttributionLimitation != row.AttributionLimitation) {
				contextIDs = append(contextIDs, id)
			}
		}
	}
	if err := x.gatherGaps(r.Gaps, &item, target, contextIDs); err != nil {
		return SelectedFinding{}, err
	}
	if source == "service_health" || source == "azure_monitor" {
		item.OperationalStateConflict = operationalStateConflict(item)
	}
	if source == "service_health" {
		item.FuturePlannedMaintenance = !slices.ContainsFunc(support, func(e Evidence) bool {
			return e.ServiceHealth == nil || e.ServiceHealth.Category != "PlannedMaintenance" || !r.futureServiceEvent(e)
		})
	}
	return item, nil
}

func operationalStateConflict(item SelectedFinding) bool {
	if len(item.Context) == 0 {
		return false
	}
	observations := slices.Concat(item.Supporting, item.Context)
	current := latestCandidates(observations, func(e Evidence) *time.Time { return e.ReportedAt })
	expectedState := item.Supporting[0].State
	unknownState := "unknown_state"
	if item.Source == "azure_monitor" {
		unknownState = "unknown_condition"
	}
	return slices.ContainsFunc(current, func(e Evidence) bool {
		return e.State != expectedState || slices.Contains(e.Limitations, unknownState)
	})
}

func (x selectionIndex) resolveEvidence(ids []string) ([]Evidence, error) {
	var rows []Evidence
	for _, id := range ids {
		e, exists := x.evidence[id]
		if !exists || !evidenceIDPattern.MatchString(id) {
			return nil, errors.New("invalid or unresolved evidence ID in selection")
		}
		rows = append(rows, e)
	}
	return rows, nil
}

func selectionTarget(rows []Evidence) (name, kind string, ids []string) {
	if len(rows) == 0 {
		return "", "", nil
	}
	first := rows[0].Resource
	name, kind = first.Name, first.Type
	for _, e := range rows[1:] {
		if !strings.EqualFold(e.Resource.Name, first.Name) {
			name = ""
		} else if name != "" {
			name = min(name, e.Resource.Name)
		}
		if !strings.EqualFold(e.Resource.Type, first.Type) {
			kind = ""
		} else if kind != "" {
			kind = min(kind, e.Resource.Type)
		}
	}
	if name != "" || kind != "" {
		ids = evidenceRefs(rows)
	}
	return name, kind, ids
}

func (x selectionIndex) currentHealth(f Finding, target string, schema int) (*CurrentResourceHealth, error) {
	current := x.health[target]
	state, deciding := decideResourceHealth(current, schema)
	conflict := len(current) > 0 && len(deciding) == 0
	ids := deciding
	if conflict {
		ids = evidenceRefs(current)
	}
	observations, err := x.resolveEvidence(ids)
	if err != nil {
		return nil, err
	}
	return &CurrentResourceHealth{
		State: state, Observations: observations, Historical: f.supersededForAssessment(deciding), Conflict: conflict,
		MissingSourceTime: slices.ContainsFunc(observations, func(e Evidence) bool { return e.ReportedAt == nil || e.ReportedAt.IsZero() }),
	}, nil
}

func (x selectionIndex) gatherGaps(gaps []Gap, item *SelectedFinding, target string, contextIDs []string) error {
	f := item.Finding
	group := canonicalFoldID(resourceGroupTarget(target))
	positions := slices.Concat(x.byTarget[""], x.byFinding[f.ID])
	if target != "" {
		positions = append(positions, x.byTarget[target]...)
	}
	if group != "" && group != target {
		positions = append(positions, x.byTarget[group]...)
	}
	if f.EntityID != "" {
		positions = append(positions, x.byEntity[selectionEntityKey(item.Source, f.EntityID)]...)
	}
	slices.Sort(positions)
	type broaderReason struct {
		source, reason string
		blocking       bool
	}
	seenBroader := make(map[broaderReason]bool)
	for _, i := range slices.Compact(positions) {
		g := gaps[i]
		sameEntity := g.Source == item.Source && f.EntityID != "" && strings.EqualFold(g.EntityID, f.EntityID)
		matchesFinding := g.FindingID == f.ID || sameEntity && g.FindingID == ""
		sameTarget := target != "" && strings.EqualFold(g.ResourceID, target)
		if !g.Blocking && !matchesFinding && !(g.Source == item.Source && g.EntityID == "" &&
			(g.ResourceID == "" || sameTarget)) {
			continue
		}
		direct := matchesFinding || sameTarget && g.FindingID == "" && g.EntityID == ""
		info := SelectionGap{Source: g.Source, Reason: g.Reason, Blocking: g.Blocking, Broader: !direct}
		if direct {
			rows, err := x.resolveEvidence(g.EvidenceIDs)
			if err != nil {
				return err
			}
			for _, e := range rows {
				switch {
				case sameEntity && e.Source == item.Source && strings.EqualFold(e.EntityID, f.EntityID):
					contextIDs = append(contextIDs, e.ID)
				case slices.Contains(f.EvidenceIDs, e.ID):
				case !sameEntity && sameTarget && strings.EqualFold(e.ResourceID, target):
				default:
					return errors.New("coverage gap evidence does not apply to the selected finding")
				}
				info.EvidenceIDs = append(info.EvidenceIDs, e.ID)
			}
		} else {
			key := broaderReason{g.Source, g.Reason, g.Blocking}
			if seenBroader[key] {
				continue
			}
			seenBroader[key] = true
		}
		item.Gaps = append(item.Gaps, info)
	}
	contextIDs = slices.DeleteFunc(sortedUnique(contextIDs), func(id string) bool {
		return slices.Contains(f.EvidenceIDs, id)
	})
	var err error
	item.Context, err = x.resolveEvidence(contextIDs)
	return err
}

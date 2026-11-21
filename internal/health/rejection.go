package health

func scopedRejection(source, subscription string, input RejectedRecordInput) (ScopedRejection, bool) {
	switch source {
	case "inventory":
		if ResourceID(input.ID, subscription) {
			return ScopedRejection{ResourceID: input.ID}, true
		}
	case "resource_health":
		if id, ok := statusResourceID(input.ID, subscription); ok {
			return ScopedRejection{ResourceID: id}, true
		}
	case "azure_monitor":
		target := input.TargetResourceID
		if entityID(input.ID, subscription, "Microsoft.AlertsManagement", "alerts", false) &&
			ResourceID(target, subscription) && !monitorWorkspaceTarget(target) {
			return ScopedRejection{ResourceID: target, EntityID: input.ID}, true
		}
	case advisorSource:
		if parent, valid := advisorInstanceParent(input.ID, subscription); valid {
			if target, limit := advisorAttribution(input.TargetResourceID, parent, subscription); limit == "" {
				return ScopedRejection{ResourceID: target, EntityID: input.ID}, true
			}
		}
	}
	return ScopedRejection{}, false
}

func (r *Run) rebuildRejectionGaps(source string, rejected []ScopedRejection) {
	if len(rejected) == 0 {
		return
	}
	direct := make(map[string]bool)
	byEntity := make(map[string][]Evidence)
	for _, s := range rejected {
		target, entity := canonicalFoldID(s.ResourceID), canonicalFoldID(s.EntityID)
		if entity == "" {
			direct[target] = true
		} else {
			byEntity[entity] = append(byEntity[entity], Evidence{ResourceID: target})
		}
	}
	for target := range direct {
		r.block(source, "", target, "invalid_records", nil, "")
	}
	// A later version of the same alert or recommendation can change its
	// target. Include the related accepted observations in the gap's scope.
	related := make(map[string][]Evidence)
	if len(byEntity) > 0 {
		for _, e := range r.Evidence {
			if e.Source == source {
				key := canonicalFoldID(e.EntityID)
				if _, rejected := byEntity[key]; rejected {
					related[key] = append(related[key], e)
				}
			}
		}
	}
	for entity, targets := range byEntity {
		observations := related[entity]
		r.blockTargets(source, entity, "invalid_records", append(targets, observations...), evidenceRefs(observations), "")
	}
}

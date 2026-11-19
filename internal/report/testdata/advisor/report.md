# Subscription health report

**INCOMPLETE coverage.** No resource is classified overall OK.

Run ID: synthetic-advisor-run

Revision: 18

Execution outcome: **completed**

Started (UTC): 2026-11-18T08:00:00Z; updated (UTC): 2026-11-18T08:10:00Z; ended (UTC): 2026-11-18T08:10:00Z.

Tool version: test; schema version: 3; filtered values: 0.

Requested scope: Synthetic subscription.

Subscription: Synthetic subscription; ID: 11111111-1111-1111-1111-111111111111; tenant: 44444444-4444-4444-4444-444444444444; cloud: AzureCloud.

Evidence snapshot: [run.json](run.json). Match its run ID and revision to this report. If revisions differ, this report is stale.

## Source coverage

| Source | Status | Pages | Rejected records | Excluded records | Unclassified records | Reason |
| --- | --- | --- | --- | --- | --- | --- |
| advisor | completed | 2 | 0 | 0 | 1 | — |
| azure&#95;monitor | completed | 1 | 0 | 0 | 0 | — |
| inventory | completed | 1 | 0 | 0 | 0 | — |
| resource&#95;health | completed | 1 | 0 | 0 | 0 | — |
| service&#95;health | completed | 1 | 0 | 0 | 0 | — |

Collector completion describes enumeration by that API. Inventory may omit child resources; an Available observation provides Resource Health evidence only. Failed checks and missing evidence do not establish resource faults.

## Service Health

Collection status: completed.

Run reference (UTC): 2026-11-18T08:00:00Z.
Query prepared (UTC): 2026-11-18T08:00:00Z.

Rolling 30-day lower cutoff (UTC): 2026-10-19T08:00:00Z. Configured API queryStartTime: 10/19/2026. Events are filtered locally by lastUpdateTime at the exact cutoff. Older unchanged active events may be absent.

Evidence collected (UTC): not recorded to not recorded. Pages are collected sequentially; this is not an atomic snapshot. Empty completed lists do not establish OK.

ServiceIssue identifies an incident notice; PlannedMaintenance, HealthAdvisory and SecurityAdvisory retain their distinct categories. Future notices do not establish an ongoing outage. The listing does not enumerate affected resources. Service/region attribution and uncertainty appear in the evidence index. Sensitive details are not retrieved; available metadata and detail limitations remain visible.

### Active summaries

No active summaries supported by the collected evidence. Consult saved observations and coverage gaps.

### Other saved observations

Inactive, excluded, unclassified, superseded or conflicting evidence is retained here. A saved source state alone does not establish an active summary; inspect its evidence and coverage gaps.

| Source entity | Disposition | Source state | Source updated (UTC) | Evidence |
| --- | --- | --- | --- | --- |
| None collected | — | — | — | — |

## Azure Monitor

Collection status: completed.

Run reference (UTC): 2026-11-18T08:00:00Z.
Query prepared (UTC): 2026-11-18T08:00:00Z.

Configured API timeRange: 30d. The API applies this relative window.

Evidence collected (UTC): not recorded to not recorded. Pages are collected sequentially; this is not an atomic snapshot. Empty completed lists do not establish OK.

Fired is the source condition, not proof of a continuing fault. Stateless alerts can remain Fired; instances are retained for 30 days, and stateful conditions can outlive that retention. Workflow response and notification suppression do not change the condition. A workspace target does not identify every resource referenced by a log query.

### Active summaries

No active summaries supported by the collected evidence. Consult saved observations and coverage gaps.

### Other saved observations

Inactive, excluded, unclassified, superseded or conflicting evidence is retained here. A saved source state alone does not establish an active summary; inspect its evidence and coverage gaps.

| Source entity | Disposition | Source state | Source updated (UTC) | Evidence |
| --- | --- | --- | --- | --- |
| None collected | — | — | — | — |

## Azure Advisor recommendations

Collection status: completed.

Advisor returns cached suggestions for review. A recommendation alone does not establish a resource fault.

Cached list scope: selected subscription 11111111-1111-1111-1111-111111111111. Operation: Recommendations&#46;List. API version: 2026-10-01. Query prepared (UTC): 2026-11-18T08:01:00Z.

First accepted page (UTC): 2026-11-18T08:02:00Z. Last accepted page (UTC): 2026-11-18T08:03:00Z. Pages are read sequentially.

### New recommendations

| Type group / instance ID | Category | Impact | Assessed resource | Last evaluated (UTC) | Finding ID | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| **Type 22222222-2222-2222-2222-222222222222 (2)** | — | High | — | — | — | — |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/vm/providers/microsoft&#46;advisor/recommendations/cost-vm | Cost | High | /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/vm | 2026-11-18T07:00:00Z | finding-5ae96edbec1f8f4a8110ca2d49fc76f8e67f6510086e4b6b7739935795fe2391 | [ev-f711a04f3f5cbcacce60acdb5334d0373bf4b52577ecac291210b813652065cc](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/cost-other | Cost | Medium | **Resource unknown** | unknown | finding-7b1cfce038765e80055987806a15c73c12a3bd462653c4f4de1c94cc48e4cae3 | [ev-223f761b14431f2559420acddb6fc96787b8022cf01db87735bb4658c50c54be](run.json) |
| **Type unknown (1)** | — | unknown | — | — | — | — |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/uncategorized | unknown | unknown | **Resource unknown** | unknown | finding-141ffdb92abc6049e7248b51f7d630e5b501231c0b47d9a2ab0480264d57dcd8 | [ev-6bc2539d1cdc3b1a74e8ee5741ff147ba62288accca3d43120e7b6ed76b24462](run.json) |

### Other recommendation statuses

| Type group / instance ID | Category | Impact | Assessed resource | Last evaluated (UTC) | Status | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| **Type 33333333-3333-3333-3333-333333333333 (1)** | — | Medium | — | — | Postponed | — |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/performance | Performance | Medium | **Resource unknown** | 2026-11-18T07:00:00Z | Postponed | [ev-331d1007d5bb5736119d91000fd39e33663983cc032dd01023151bdeb3314f9d](run.json) |
| **Type 11111111-1111-1111-1111-111111111111 (1)** | — | High | — | — | Dismissed | — |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/automation | Security | High | **Resource unknown** | unknown | Dismissed | [ev-61b8c29db5cf655e2099217bf63d146437a23c006c74c564850fb8a2126637da](run.json), [ev-da655681b9c9807122bc3e4ae0be62d5f39d14a6d92b4244dc60c515a31c8aba](run.json) |
| **Type unknown (1)** | — | Low | — | — | Completed | — |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/retired | OperationalExcellence | Low | **Resource unknown** | unknown | Completed | [ev-cba4025e782535be406d4f47aefdf54977c7b18e3936ae46901a21065c941243](run.json) |
| **Type unknown (1)** | — | unknown | — | — | unknown | — |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/unclassified | unknown | unknown | **Resource unknown** | unknown | unknown | [ev-a9fd2dc6092652a7a2086ecdb95698b867f616d00579318d5af6332caed7f029](run.json) |

## Resource Health findings

These findings retain Resource Health source observations, including conflicting or superseded observations. Consult current-state tables and timestamps before interpreting them as current faults.

| Category | Resource | Source observation | Finding ID | Evidence |
| --- | --- | --- | --- | --- |
| resource&#95;health&#95;issue | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm | Resource Health reported Degraded | finding-e4bca30b174decf6e96bb1d3fe80c7d749ab1d5848e7fb744e051ae36ecb8227 | [ev-651a8c5a5c73e8f25de5d9ea4c24ccecc8c6d849fe8c58df7c79dedd224912de](run.json) |

## Resource Health issue states

| Resource | Current Resource Health state | In observed inventory | Evidence |
| --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm | Degraded | true | [ev-651a8c5a5c73e8f25de5d9ea4c24ccecc8c6d849fe8c58df7c79dedd224912de](run.json) |

## Resource Health Available observations

| Resource | Current Resource Health state | In observed inventory | Evidence |
| --- | --- | --- | --- |
| None observed | — | — | — |

## Unknown and not-checked resources

| Resource | Current Resource Health state | In observed inventory | Evidence |
| --- | --- | --- | --- |
| None observed | — | — | — |

## Observed inventory (1 resources)

All observed resources are listed below. Conflicting inventory facts remain in the evidence index.

| Resource ID | Name | Type | Resource group | Location | Tags | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm | vm | Microsoft&#46;Compute/virtualMachines | g | westeurope | — | [ev-246a380dc7fd587bb15b541f22ff11916070f5891a1692c47ad0bd6b99064c2b](run.json) |

## Coverage gaps and failed checks

| Source | Source entity | Resource attribution | Reason | Evidence |
| --- | --- | --- | --- | --- |
| advisor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/automation | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-da655681b9c9807122bc3e4ae0be62d5f39d14a6d92b4244dc60c515a31c8aba](run.json) |
| advisor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/cost-other | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-223f761b14431f2559420acddb6fc96787b8022cf01db87735bb4658c50c54be](run.json) |
| advisor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/performance | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-331d1007d5bb5736119d91000fd39e33663983cc032dd01023151bdeb3314f9d](run.json) |
| advisor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/retired | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-cba4025e782535be406d4f47aefdf54977c7b18e3936ae46901a21065c941243](run.json) |
| advisor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/uncategorized | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-6bc2539d1cdc3b1a74e8ee5741ff147ba62288accca3d43120e7b6ed76b24462](run.json) |
| advisor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/unclassified | Unknown; no affected resource established | unknown&#95;advisor&#95;status | [ev-a9fd2dc6092652a7a2086ecdb95698b867f616d00579318d5af6332caed7f029](run.json) |
| advisor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/unclassified | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-a9fd2dc6092652a7a2086ecdb95698b867f616d00579318d5af6332caed7f029](run.json) |
| azure&#95;monitor | — | Unknown; no affected resource established | monitor&#95;30&#95;day&#95;retention&#95;and&#95;stateless&#95;limits | — |
| azure&#95;monitor | — | Unknown; no affected resource established | non&#95;atomic&#95;source&#95;snapshot | — |
| service&#95;health | — | Unknown; no affected resource established | non&#95;atomic&#95;source&#95;snapshot | — |
| service&#95;health | — | Unknown; no affected resource established | resource&#95;impact&#95;not&#95;enumerated | — |
| service&#95;health | — | Unknown; no affected resource established | service&#95;health&#95;30&#95;day&#95;update&#95;window | — |

## Execution diagnostics

No execution diagnostics. Overall coverage remains incomplete.

## Evidence index (10 records)

Each reference identifies a record in the sibling JSON snapshot. Collection time and source time are distinct. Historical statuses cannot establish current health.

### ev-223f761b14431f2559420acddb6fc96787b8022cf01db87735bb4658c50c54be

- Record: [ev-223f761b14431f2559420acddb6fc96787b8022cf01db87735bb4658c50c54be](run.json)
- Source: advisor. Operation: Recommendations&#46;List. API version: 2026-10-01.
- Advisor instance: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;Advisor/recommendations/cost-other. Selected subscription: 11111111-1111-1111-1111-111111111111.
- Type 22222222-2222-2222-2222-222222222222. Category: Cost. Impact: Medium. Status: New.
- Assessed resource: **Resource unknown**.
- Attribution limitation: unknown&#95;resource&#95;attribution.
- Last evaluated (UTC): unknown. State changed (UTC): unknown. Collected (UTC): 2026-11-18T08:02:00Z.

### ev-331d1007d5bb5736119d91000fd39e33663983cc032dd01023151bdeb3314f9d

- Record: [ev-331d1007d5bb5736119d91000fd39e33663983cc032dd01023151bdeb3314f9d](run.json)
- Source: advisor. Operation: Recommendations&#46;List. API version: 2026-10-01.
- Advisor instance: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;Advisor/recommendations/performance. Selected subscription: 11111111-1111-1111-1111-111111111111.
- Type 33333333-3333-3333-3333-333333333333. Category: Performance. Impact: Medium. Status: Postponed.
- Assessed resource: **Resource unknown**.
- Attribution limitation: unknown&#95;resource&#95;attribution.
- Last evaluated (UTC): 2026-11-18T07:00:00Z. State changed (UTC): unknown. Collected (UTC): 2026-11-18T08:02:00Z.

### ev-61b8c29db5cf655e2099217bf63d146437a23c006c74c564850fb8a2126637da

- Record: [ev-61b8c29db5cf655e2099217bf63d146437a23c006c74c564850fb8a2126637da](run.json)
- Source: advisor. Operation: Recommendations&#46;List. API version: 2026-10-01.
- Advisor instance: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;Advisor/recommendations/automation. Selected subscription: 11111111-1111-1111-1111-111111111111.
- Type 11111111-1111-1111-1111-111111111111. Category: Security. Impact: High. Status: New.
- Assessed resource: **Resource unknown**.
- Attribution limitation: unknown&#95;resource&#95;attribution.
- Last evaluated (UTC): unknown. State changed (UTC): 2026-11-17T08:00:00Z. Collected (UTC): 2026-11-18T08:02:00Z.

### ev-6bc2539d1cdc3b1a74e8ee5741ff147ba62288accca3d43120e7b6ed76b24462

- Record: [ev-6bc2539d1cdc3b1a74e8ee5741ff147ba62288accca3d43120e7b6ed76b24462](run.json)
- Source: advisor. Operation: Recommendations&#46;List. API version: 2026-10-01.
- Advisor instance: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;Advisor/recommendations/uncategorized. Selected subscription: 11111111-1111-1111-1111-111111111111.
- Type unknown. Category: unknown. Impact: unknown. Status: New.
- Assessed resource: **Resource unknown**.
- Attribution limitation: unknown&#95;resource&#95;attribution.
- Last evaluated (UTC): unknown. State changed (UTC): unknown. Collected (UTC): 2026-11-18T08:02:00Z.

### ev-a9fd2dc6092652a7a2086ecdb95698b867f616d00579318d5af6332caed7f029

- Record: [ev-a9fd2dc6092652a7a2086ecdb95698b867f616d00579318d5af6332caed7f029](run.json)
- Source: advisor. Operation: Recommendations&#46;List. API version: 2026-10-01.
- Advisor instance: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;Advisor/recommendations/unclassified. Selected subscription: 11111111-1111-1111-1111-111111111111.
- Type unknown. Category: unknown. Impact: unknown. Status: unknown.
- Assessed resource: **Resource unknown**.
- Attribution limitation: unknown&#95;resource&#95;attribution.
- Last evaluated (UTC): unknown. State changed (UTC): unknown. Collected (UTC): 2026-11-18T08:02:00Z.

### ev-cba4025e782535be406d4f47aefdf54977c7b18e3936ae46901a21065c941243

- Record: [ev-cba4025e782535be406d4f47aefdf54977c7b18e3936ae46901a21065c941243](run.json)
- Source: advisor. Operation: Recommendations&#46;List. API version: 2026-10-01.
- Advisor instance: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;Advisor/recommendations/retired. Selected subscription: 11111111-1111-1111-1111-111111111111.
- Type unknown. Category: OperationalExcellence. Impact: Low. Status: Completed.
- Assessed resource: **Resource unknown**.
- Attribution limitation: unknown&#95;resource&#95;attribution.
- Last evaluated (UTC): unknown. State changed (UTC): unknown. Collected (UTC): 2026-11-18T08:02:00Z.

### ev-da655681b9c9807122bc3e4ae0be62d5f39d14a6d92b4244dc60c515a31c8aba

- Record: [ev-da655681b9c9807122bc3e4ae0be62d5f39d14a6d92b4244dc60c515a31c8aba](run.json)
- Source: advisor. Operation: Recommendations&#46;List. API version: 2026-10-01.
- Advisor instance: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;Advisor/recommendations/automation. Selected subscription: 11111111-1111-1111-1111-111111111111.
- Type 11111111-1111-1111-1111-111111111111. Category: Security. Impact: High. Status: Dismissed.
- Assessed resource: **Resource unknown**.
- Attribution limitation: unknown&#95;resource&#95;attribution.
- Last evaluated (UTC): unknown. State changed (UTC): 2026-11-18T07:00:00Z. Collected (UTC): 2026-11-18T08:03:00Z.

### ev-f711a04f3f5cbcacce60acdb5334d0373bf4b52577ecac291210b813652065cc

- Record: [ev-f711a04f3f5cbcacce60acdb5334d0373bf4b52577ecac291210b813652065cc](run.json)
- Source: advisor. Operation: Recommendations&#46;List. API version: 2026-10-01.
- Advisor instance: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm/providers/Microsoft&#46;Advisor/recommendations/cost-vm. Selected subscription: 11111111-1111-1111-1111-111111111111.
- Type 22222222-2222-2222-2222-222222222222. Category: Cost. Impact: High. Status: New.
- Assessed resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm.
- Last evaluated (UTC): 2026-11-18T07:00:00Z. State changed (UTC): 2026-11-17T08:00:00Z. Collected (UTC): 2026-11-18T08:02:00Z.

### ev-246a380dc7fd587bb15b541f22ff11916070f5891a1692c47ad0bd6b99064c2b

- Record: [ev-246a380dc7fd587bb15b541f22ff11916070f5891a1692c47ad0bd6b99064c2b](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-18T08:00:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm; name: vm; type: Microsoft&#46;Compute/virtualMachines; group: g; location: westeurope; tags: —.

### ev-651a8c5a5c73e8f25de5d9ea4c24ccecc8c6d849fe8c58df7c79dedd224912de

- Record: [ev-651a8c5a5c73e8f25de5d9ea4c24ccecc8c6d849fe8c58df7c79dedd224912de](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm
- Kind: availability; state: Degraded; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-18T08:00:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.


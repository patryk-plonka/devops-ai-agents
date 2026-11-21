# Subscription health report

**Incomplete coverage · 1 resource OK, 1 with findings, 1 unknown.**

OK within the agreed checks means an inventoried resource had current Available Resource Health evidence and sufficient Service Health, Azure Monitor and Azure Advisor evidence within the saved query scopes and times. The counts cover observed inventory. A resource with a known finding and a missing check counts as unknown, with its finding shown in the assessment.

Run ID: synthetic-assessment-run

Revision: 17

Execution outcome: **completed**

Started (UTC): 2026-11-20T08:00:00Z; updated (UTC): 2026-11-20T08:12:00Z; ended (UTC): 2026-11-20T08:12:00Z.

Tool version: test; schema version: 4; filtered values: 0.

Requested scope: Synthetic subscription.

Subscription: Synthetic subscription; ID: 11111111-1111-1111-1111-111111111111; tenant: —; cloud: AzureCloud.

Evidence snapshot: [run.json](run.json). Match its run ID and revision to this report. If revisions differ, this report is stale.

## Source coverage

| Source | Status | Pages | Rejected records | Excluded records | Unclassified records | Reason |
| --- | --- | --- | --- | --- | --- | --- |
| advisor | completed | 1 | 0 | 0 | 0 | — |
| azure&#95;monitor | completed | 1 | 0 | 0 | 0 | — |
| inventory | completed | 1 | 0 | 0 | 0 | — |
| resource&#95;health | completed | 2 | 0 | 0 | 0 | — |
| service&#95;health | completed | 1 | 0 | 0 | 0 | — |

Collector completion describes enumeration by that API. Inventory may omit child resources; an Available observation provides Resource Health evidence only. Failed checks and missing evidence do not establish resource faults.

### Query provenance

Each source link names an entry in the sibling JSON for this run and revision. Accepted-page times include empty pages. Pages are collected sequentially and reflect different points in time.

| JSON source | Operation | API version | Subscription | Run reference (UTC) | Query prepared (UTC) | First accepted page (UTC) | Last accepted page (UTC) | Terminal enumeration |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| [sources&#46;inventory](run.json) | Resources&#46;List | 2025-04-01 | 11111111-1111-1111-1111-111111111111 | 2026-11-20T08:00:00Z | 2026-11-20T08:00:59Z | 2026-11-20T08:01:00Z | 2026-11-20T08:01:00Z | reached |
| [sources&#46;resource&#95;health](run.json) | AvailabilityStatuses&#46;ListBySubscriptionId | 2025-05-01 | 11111111-1111-1111-1111-111111111111 | 2026-11-20T08:00:00Z | 2026-11-20T08:02:59Z | 2026-11-20T08:03:00Z | 2026-11-20T08:04:00Z | reached |
| [sources&#46;service&#95;health](run.json) | Events&#46;ListBySubscriptionId | 2025-05-01 | 11111111-1111-1111-1111-111111111111 | 2026-11-20T08:00:00Z | 2026-11-20T08:04:59Z | 2026-11-20T08:05:00Z | 2026-11-20T08:05:00Z | reached |
| [sources&#46;azure&#95;monitor](run.json) | Alerts&#46;GetAll | 2019-03-01 | 11111111-1111-1111-1111-111111111111 | 2026-11-20T08:00:00Z | 2026-11-20T08:06:59Z | 2026-11-20T08:07:00Z | 2026-11-20T08:07:00Z | reached |
| [sources&#46;advisor](run.json) | Recommendations&#46;List | 2026-10-01 | 11111111-1111-1111-1111-111111111111 | 2026-11-20T08:00:00Z | 2026-11-20T08:08:59Z | 2026-11-20T08:09:00Z | 2026-11-20T08:09:00Z | reached |

## Resource assessments

Each row cites the source entries and deciding evidence. A source link without evidence can support a bounded negative when the check is complete. See each collector's scope and limits below.

| Resource ID | Result | Checks and deciding evidence | Missing checks or proof | Findings | History and Postponed context |
| --- | --- | --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/eligible | OK | [sources&#46;inventory](run.json) [ev-4f4f52d2552cf3942b5a37562548ad2dfffb4ca4dc1bf5375dc54e1c159de317](run.json)<br>[sources&#46;resource&#95;health](run.json) [ev-e2b78157e3d3fbed60cd962ab940ded169cab4f94f19155503c251532e599f0c](run.json)<br>[sources&#46;service&#95;health](run.json)<br>[sources&#46;azure&#95;monitor](run.json)<br>[sources&#46;advisor](run.json) [ev-cfafeaaddbd6912a71e4db22c43a29d005c4290d6e4cfc2c912801657b6c6e48](run.json) | — | — | Historical: Resource Health reported Degraded ([finding-75813f86395f098158d9c490f1d93e028b7a73f07594fdc4d059b79ed56ab359](run.json))<br>Advisor Postponed: [ev-cfafeaaddbd6912a71e4db22c43a29d005c4290d6e4cfc2c912801657b6c6e48](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/finding | Finding | [sources&#46;inventory](run.json) [ev-2ef62cb8e78c888c0bfd83d31ee52ccec34c51b0e2d813353e930d88cf6b3b7d](run.json)<br>[sources&#46;resource&#95;health](run.json) [ev-62573ed376e0b599c3d5e5a7fee3427144d4ef70f95da8e6d15452c91aff6c32](run.json)<br>[sources&#46;service&#95;health](run.json)<br>[sources&#46;azure&#95;monitor](run.json) [ev-c029c6baebf507d1b83da685113762a44873900df25ef69567a403d6e10d751b](run.json)<br>[sources&#46;advisor](run.json) | — | Azure Monitor source condition Fired ([finding-14b900683c6a721690f7dac5a1b0f00ae892efbd5735728d2fef3350513dcbec](run.json)) | — |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/potential-gap | Unknown | [sources&#46;inventory](run.json) [ev-f3ecc329d503e7b8e0a002f7624bc4d11e705e9dae2f1b291a3f024111f8569b](run.json)<br>[sources&#46;resource&#95;health](run.json)<br>[sources&#46;service&#95;health](run.json)<br>[sources&#46;azure&#95;monitor](run.json) [ev-847a6361c11628f2385ce41a6a59925f7c7cbe21f8b4a9bd58f779afd024d238](run.json)<br>[sources&#46;advisor](run.json) | resource&#95;health · missing&#95;health&#95;evidence [ev-f3ecc329d503e7b8e0a002f7624bc4d11e705e9dae2f1b291a3f024111f8569b](run.json) | Azure Monitor source condition Fired ([finding-1cf8d96e0272707245799b339e017a18e32a61e573455e27f624941a6a68ed5c](run.json)) | — |

## Service Health

Collection status: completed.

Run reference (UTC): 2026-11-20T08:00:00Z.
Query prepared (UTC): 2026-11-20T08:04:59Z.

Rolling 30-day lower cutoff (UTC): 2026-10-21T08:00:00Z. Configured API queryStartTime: 10/21/2026. Events are filtered locally by lastUpdateTime at the exact cutoff. Older unchanged active events may be absent.

Evidence collected (UTC): not recorded to not recorded. See accepted-page times under Query provenance for lists with no retained rows.

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

Run reference (UTC): 2026-11-20T08:00:00Z.
Query prepared (UTC): 2026-11-20T08:06:59Z.

Configured API timeRange: 30d. The API applies this relative window.

Evidence collected (UTC): 2026-11-20T08:07:00Z to 2026-11-20T08:07:00Z. See accepted-page times under Query provenance for lists with no retained rows.

Fired is the source condition, not proof of a continuing fault. Stateless alerts can remain Fired; instances are retained for 30 days, and stateful conditions can outlive that retention. Workflow response and notification suppression do not change the condition. A workspace target does not identify every resource referenced by a log query.

### Active summaries

- Azure Monitor source condition Fired
  - Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/finding; selected subscription: 11111111-1111-1111-1111-111111111111.
  - Finding ID: finding-14b900683c6a721690f7dac5a1b0f00ae892efbd5735728d2fef3350513dcbec; evidence: [ev-c029c6baebf507d1b83da685113762a44873900df25ef69567a403d6e10d751b](run.json).
  - Signal: Metric; condition: Fired; workflow response: Acknowledged; severity: Sev2; notification suppressed: not recorded.
  - Target: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/finding (reported target only); started (UTC): not recorded; last modified (UTC): 2026-11-20T07:00:00Z.

- Azure Monitor source condition Fired
  - Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/potential-gap; selected subscription: 11111111-1111-1111-1111-111111111111.
  - Finding ID: finding-1cf8d96e0272707245799b339e017a18e32a61e573455e27f624941a6a68ed5c; evidence: [ev-847a6361c11628f2385ce41a6a59925f7c7cbe21f8b4a9bd58f779afd024d238](run.json).
  - Signal: Metric; condition: Fired; workflow response: Acknowledged; severity: Sev2; notification suppressed: not recorded.
  - Target: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/potential-gap (reported target only); started (UTC): not recorded; last modified (UTC): 2026-11-20T07:00:00Z.

### Other saved observations

Inactive, excluded, unclassified, superseded or conflicting evidence is retained here. A saved source state alone does not establish an active summary; inspect its evidence and coverage gaps.

| Source entity | Disposition | Source state | Source updated (UTC) | Evidence |
| --- | --- | --- | --- | --- |
| None collected | — | — | — | — |

## Azure Advisor recommendations

Collection status: completed.

Advisor returns cached suggestions for review. A recommendation alone does not establish a resource fault.

Cached list scope: selected subscription 11111111-1111-1111-1111-111111111111. Operation: Recommendations&#46;List. API version: 2026-10-01. Query prepared (UTC): 2026-11-20T08:08:59Z.

First accepted page (UTC): 2026-11-20T08:09:00Z. Last accepted page (UTC): 2026-11-20T08:09:00Z. Pages are read sequentially.

### New recommendations

No New recommendations among the accepted observations.

### Other recommendation statuses

| Type group / instance ID | Category | Impact | Assessed resource | Last evaluated (UTC) | Status | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| **Type 22222222-2222-2222-2222-222222222222 (1)** | — | Low | — | — | Postponed | — |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/microsoft&#46;advisor/recommendations/postponed | Cost | Low | /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/eligible | unknown | Postponed | [ev-cfafeaaddbd6912a71e4db22c43a29d005c4290d6e4cfc2c912801657b6c6e48](run.json) |

## Resource Health findings

These findings retain Resource Health source observations, including conflicting or superseded observations. Consult current-state tables and timestamps before interpreting them as current faults.

| Category | Resource | Source observation | Finding ID | Evidence |
| --- | --- | --- | --- | --- |
| resource&#95;health&#95;issue | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/eligible | Resource Health reported Degraded | finding-75813f86395f098158d9c490f1d93e028b7a73f07594fdc4d059b79ed56ab359 | [ev-b3462e15cc388416a9439b93b21afe92f41373c8b84c011952a1fe19ddb39ef7](run.json) |

## Resource Health issue states

| Resource | Current Resource Health state | In observed inventory | Evidence |
| --- | --- | --- | --- |
| None observed | — | — | — |

## Resource Health Available observations

| Resource | Current Resource Health state | In observed inventory | Evidence |
| --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/eligible | Available | true | [ev-b3462e15cc388416a9439b93b21afe92f41373c8b84c011952a1fe19ddb39ef7](run.json), [ev-e2b78157e3d3fbed60cd962ab940ded169cab4f94f19155503c251532e599f0c](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/finding | Available | true | [ev-62573ed376e0b599c3d5e5a7fee3427144d4ef70f95da8e6d15452c91aff6c32](run.json) |

## Unknown and not-checked resources

| Resource | Current Resource Health state | In observed inventory | Evidence |
| --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/potential-gap | not&#95;checked | true | — |

## Observed inventory (3 resources)

All observed resources are listed below. Conflicting inventory facts remain in the evidence index.

| Resource ID | Name | Type | Resource group | Location | Tags | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/eligible | vm &#124; &#91;link&#93;&#40;https&#58;//bad&#46;example&#41; &#60;script&#62; | — | g | — | — | [ev-4f4f52d2552cf3942b5a37562548ad2dfffb4ca4dc1bf5375dc54e1c159de317](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/finding | finding | — | g | — | — | [ev-2ef62cb8e78c888c0bfd83d31ee52ccec34c51b0e2d813353e930d88cf6b3b7d](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/potential-gap | potential-gap | — | g | — | — | [ev-f3ecc329d503e7b8e0a002f7624bc4d11e705e9dae2f1b291a3f024111f8569b](run.json) |

## Coverage gaps and failed checks

| Source | Source entity | Resource scope | Reason | Evidence |
| --- | --- | --- | --- | --- |
| resource&#95;health | — | /subscriptions/11111111-1111-1111-1111-111111111111/resourcegroups/g/providers/microsoft&#46;compute/virtualmachines/potential-gap | missing&#95;health&#95;evidence | [ev-f3ecc329d503e7b8e0a002f7624bc4d11e705e9dae2f1b291a3f024111f8569b](run.json) |

## Source limits and evidence notes

| Source | Source entity | Resource scope | Reason | Evidence |
| --- | --- | --- | --- | --- |
| azure&#95;monitor | — | — | monitor&#95;30&#95;day&#95;retention&#95;and&#95;stateless&#95;limits | — |
| azure&#95;monitor | — | — | non&#95;atomic&#95;source&#95;snapshot | — |
| service&#95;health | — | — | non&#95;atomic&#95;source&#95;snapshot | — |
| service&#95;health | — | — | resource&#95;impact&#95;not&#95;enumerated | — |
| service&#95;health | — | — | service&#95;health&#95;30&#95;day&#95;update&#95;window | — |

## Execution diagnostics

No execution diagnostics.

## Evidence index (9 records)

Each reference identifies a record in the sibling JSON snapshot. Collection time and source time are distinct. Historical statuses cannot establish current health.

### ev-cfafeaaddbd6912a71e4db22c43a29d005c4290d6e4cfc2c912801657b6c6e48

- Record: [ev-cfafeaaddbd6912a71e4db22c43a29d005c4290d6e4cfc2c912801657b6c6e48](run.json)
- Source: advisor. Operation: Recommendations&#46;List. API version: 2026-10-01.
- Advisor instance: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;Advisor/recommendations/postponed. Selected subscription: 11111111-1111-1111-1111-111111111111.
- Type 22222222-2222-2222-2222-222222222222. Category: Cost. Impact: Low. Status: Postponed.
- Assessed resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/eligible.
- Last evaluated (UTC): unknown. State changed (UTC): 2026-11-20T07:00:00Z. Collected (UTC): 2026-11-20T08:09:00Z.

### ev-c029c6baebf507d1b83da685113762a44873900df25ef69567a403d6e10d751b

- Record: [ev-c029c6baebf507d1b83da685113762a44873900df25ef69567a403d6e10d751b](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/finding; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/finding (reported target only).
- Kind: monitor&#95;alert; disposition: active.
- Collected (UTC): 2026-11-20T08:07:00Z; source reported (UTC): 2026-11-20T07:00:00Z; source occurred (UTC): not recorded.
- Monitor signal: Metric; source condition: Fired; workflow response: Acknowledged; severity: Sev2; notification suppressed: not recorded.
- Target attribution: reported&#95;target; monitor service last modified (UTC): not recorded.

### ev-847a6361c11628f2385ce41a6a59925f7c7cbe21f8b4a9bd58f779afd024d238

- Record: [ev-847a6361c11628f2385ce41a6a59925f7c7cbe21f8b4a9bd58f779afd024d238](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/potential-gap; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/potential-gap (reported target only).
- Kind: monitor&#95;alert; disposition: active.
- Collected (UTC): 2026-11-20T08:07:00Z; source reported (UTC): 2026-11-20T07:00:00Z; source occurred (UTC): not recorded.
- Monitor signal: Metric; source condition: Fired; workflow response: Acknowledged; severity: Sev2; notification suppressed: not recorded.
- Target attribution: reported&#95;target; monitor service last modified (UTC): not recorded.

### ev-4f4f52d2552cf3942b5a37562548ad2dfffb4ca4dc1bf5375dc54e1c159de317

- Record: [ev-4f4f52d2552cf3942b5a37562548ad2dfffb4ca4dc1bf5375dc54e1c159de317](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/eligible
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-20T08:01:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/eligible; name: vm &#124; &#91;link&#93;&#40;https&#58;//bad&#46;example&#41; &#60;script&#62;; type: —; group: g; location: —; tags: —.

### ev-2ef62cb8e78c888c0bfd83d31ee52ccec34c51b0e2d813353e930d88cf6b3b7d

- Record: [ev-2ef62cb8e78c888c0bfd83d31ee52ccec34c51b0e2d813353e930d88cf6b3b7d](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/finding
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-20T08:01:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/finding; name: finding; type: —; group: g; location: —; tags: —.

### ev-f3ecc329d503e7b8e0a002f7624bc4d11e705e9dae2f1b291a3f024111f8569b

- Record: [ev-f3ecc329d503e7b8e0a002f7624bc4d11e705e9dae2f1b291a3f024111f8569b](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/potential-gap
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-20T08:01:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/potential-gap; name: potential-gap; type: —; group: g; location: —; tags: —.

### ev-b3462e15cc388416a9439b93b21afe92f41373c8b84c011952a1fe19ddb39ef7

- Record: [ev-b3462e15cc388416a9439b93b21afe92f41373c8b84c011952a1fe19ddb39ef7](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/eligible
- Kind: availability; state: Degraded; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/eligible/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-20T08:03:00Z; source reported (UTC): 2026-11-20T06:00:00Z; source occurred (UTC): not recorded.

### ev-e2b78157e3d3fbed60cd962ab940ded169cab4f94f19155503c251532e599f0c

- Record: [ev-e2b78157e3d3fbed60cd962ab940ded169cab4f94f19155503c251532e599f0c](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/eligible
- Kind: availability; state: Available; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/eligible/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-20T08:04:00Z; source reported (UTC): 2026-11-20T07:00:00Z; source occurred (UTC): not recorded.

### ev-62573ed376e0b599c3d5e5a7fee3427144d4ef70f95da8e6d15452c91aff6c32

- Record: [ev-62573ed376e0b599c3d5e5a7fee3427144d4ef70f95da8e6d15452c91aff6c32](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/finding
- Kind: availability; state: Available; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/finding/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-20T08:04:00Z; source reported (UTC): 2026-11-20T07:00:00Z; source occurred (UTC): not recorded.


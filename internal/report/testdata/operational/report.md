# Subscription health report

**INCOMPLETE coverage.** No resource is classified overall OK.

Not checked: advisor. Consult source coverage for reasons.

Run ID: synthetic-operational-run

Revision: 17

Execution outcome: **completed**

Started (UTC): 2026-11-15T12:00:00Z; updated (UTC): 2026-11-15T12:00:00Z; ended (UTC): 2026-11-15T12:00:00Z.

Tool version: test; schema version: 2; filtered values: 8.

Requested scope: Synthetic subscription.

Subscription: Synthetic subscription; ID: 11111111-1111-1111-1111-111111111111; tenant: 22222222-2222-2222-2222-222222222222; cloud: AzureCloud.

Evidence snapshot: [run.json](run.json). Match its run ID and revision to this report. If revisions differ, this report is stale.

## Source coverage

| Source | Status | Pages | Rejected records | Excluded records | Unclassified records | Reason |
| --- | --- | --- | --- | --- | --- | --- |
| advisor | not&#95;checked | 0 | 0 | 0 | 0 | not&#95;implemented |
| azure&#95;monitor | completed | 1 | 0 | 0 | 1 | — |
| inventory | completed | 1 | 0 | 0 | 0 | — |
| resource&#95;health | completed | 1 | 0 | 0 | 0 | — |
| service&#95;health | completed | 1 | 0 | 1 | 1 | — |

Collector completion describes enumeration by that API. Inventory may omit child resources; an Available observation provides Resource Health evidence only. Failed checks and missing evidence do not establish resource faults.

## Service Health

Collection status: completed.

Run reference (UTC): 2026-11-15T12:00:00Z.
Query prepared (UTC): 2026-11-15T12:00:00Z.

Rolling 30-day lower cutoff (UTC): 2026-10-16T12:00:00Z. Configured API queryStartTime: 10/16/2026. Events are filtered locally by lastUpdateTime at the exact cutoff. Older unchanged active events may be absent.

Evidence collected (UTC): 2026-11-15T12:00:00Z to 2026-11-15T12:00:00Z. Pages are collected sequentially; this is not an atomic snapshot. Empty completed lists do not establish OK.

ServiceIssue identifies an incident notice; PlannedMaintenance, HealthAdvisory and SecurityAdvisory retain their distinct categories. Future notices do not establish an ongoing outage. The listing does not enumerate affected resources. Service/region attribution and uncertainty appear in the evidence index. Sensitive details are not retrieved; available metadata and detail limitations remain visible.

### Active summaries

- Service Health Active HealthAdvisory notice
  - Source entity: /providers/Microsoft&#46;ResourceHealth/events/HealthAdvisory; selected subscription: 11111111-1111-1111-1111-111111111111.
  - Finding ID: finding-bb9e98c1e6e5d5a4882307f5c4ef6d6ba49f0ad1845345747835927e12783150; evidence: [ev-c313f76f1100adbd3e2c810f513eb8490fb76a45fe21aab4a5fdd4bd988e225d](run.json).
  - Category: HealthAdvisory; status: Active; impact starts (UTC): 2026-11-15T11:00:00Z; last updated (UTC): 2026-11-15T12:00:00Z.
  - Resource impact: unknown; not enumerated by this listing.

- Service Health future PlannedMaintenance notice
  - Source entity: /providers/Microsoft&#46;ResourceHealth/events/PlannedMaintenance; selected subscription: 11111111-1111-1111-1111-111111111111.
  - Finding ID: finding-424dd9e3601f88705ea2169d5b26b1318380fc12fd1ea0b66d5f22cfce92f846; evidence: [ev-0a2dde1d5635f2a78dbc2712ae497f2a557d809940d0ea92e06face19a3dbc7d](run.json).
  - Category: PlannedMaintenance; status: Active; impact starts (UTC): 2026-11-16T12:00:00Z; last updated (UTC): 2026-11-15T12:00:00Z.
  - Resource impact: unknown; not enumerated by this listing.

- Service Health Active SecurityAdvisory notice
  - Source entity: /providers/Microsoft&#46;ResourceHealth/events/SecurityAdvisory; selected subscription: 11111111-1111-1111-1111-111111111111.
  - Finding ID: finding-444854522f41643d15eb9c42819da6b55e915f0ab66e9aba97c1e7ed769903e7; evidence: [ev-c14ae959b5867d47be0d3aa8410f374b9e586954d2cb04c55147c523a534ff9e](run.json).
  - Category: SecurityAdvisory; status: Active; impact starts (UTC): 2026-11-15T11:00:00Z; last updated (UTC): 2026-11-15T12:00:00Z.
  - Resource impact: unknown; not enumerated by this listing.

- Service Health Active ServiceIssue
  - Source entity: /providers/Microsoft&#46;ResourceHealth/events/ServiceIssue; selected subscription: 11111111-1111-1111-1111-111111111111.
  - Finding ID: finding-187557b9a03a307c98a3cf712216942f51bf10a3222845df8319ded369166165; evidence: [ev-66da6eace263b71c71f03bfd00927e463baf4c58ce1f6baca594005ee7a23a6b](run.json).
  - Category: ServiceIssue; status: Active; impact starts (UTC): 2026-11-15T11:00:00Z; last updated (UTC): 2026-11-15T12:00:00Z.
  - Resource impact: unknown; not enumerated by this listing.

### Other saved observations

Inactive, excluded, unclassified, superseded or conflicting evidence is retained here. A saved source state alone does not establish an active summary; inspect its evidence and coverage gaps.

| Source entity | Disposition | Source state | Source updated (UTC) | Evidence |
| --- | --- | --- | --- | --- |
| /providers/Microsoft&#46;ResourceHealth/events/Billing | excluded | Active | 2026-11-15T12:00:00Z | [ev-7a473cf82462d25a62e8a6047ebe9e6238d1f0b9f231510bfde597266ad01601](run.json) |
| /providers/Microsoft&#46;ResourceHealth/events/FutureState | unclassified | FutureState | 2026-11-15T12:00:00Z | [ev-6240d4de7e6506dbd8340250571220bc0db38827ce2e1462255380f5bd176a87](run.json) |
| /providers/Microsoft&#46;ResourceHealth/events/Resolved | inactive | Resolved | 2026-11-15T12:00:00Z | [ev-7a4cdf7d0aeac74118ddc6aad83063042a74320ee7695c82f18312e90a5fc15f](run.json) |

## Azure Monitor

Collection status: completed.

Run reference (UTC): 2026-11-15T12:00:00Z.
Query prepared (UTC): 2026-11-15T12:00:00Z.

Configured API timeRange: 30d. The API applies this relative window.

Evidence collected (UTC): 2026-11-15T12:01:00Z to 2026-11-15T12:01:00Z. Pages are collected sequentially; this is not an atomic snapshot. Empty completed lists do not establish OK.

Fired is the source condition, not proof of a continuing fault. Stateless alerts can remain Fired; instances are retained for 30 days, and stateful conditions can outlive that retention. Workflow response and notification suppression do not change the condition. A workspace target does not identify every resource referenced by a log query.

### Active summaries

- Azure Monitor source condition Fired
  - Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/Acknowledged; selected subscription: 11111111-1111-1111-1111-111111111111.
  - Finding ID: finding-4928632072ccc87589e59bd3cfded05ee624767d83a3f99d97da471f4c8d3d18; evidence: [ev-f75453eb6397ccd2d9417a2bafbe81bdeee119876e372cc3f62d74124bacee7a](run.json).
  - Signal: Log; condition: Fired; workflow response: Acknowledged; severity: Sev2; notification suppressed: false.
  - Target: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;OperationalInsights/workspaces/logs (reported target only); started (UTC): 2026-11-15T11:00:00Z; last modified (UTC): 2026-11-15T12:00:00Z.

- Azure Monitor source condition Fired
  - Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/Closed; selected subscription: 11111111-1111-1111-1111-111111111111.
  - Finding ID: finding-43cdcfc45c96755e5f82bd7b119bda31f3837aa436579db46e434895a2e63c5b; evidence: [ev-f0b36799a130497539b5e28cc9caf1b7c837abc28e17a4f038883be622c0c2f4](run.json).
  - Signal: Metric; condition: Fired; workflow response: Closed; severity: Sev2; notification suppressed: true.
  - Target: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-00 (reported target only); started (UTC): 2026-11-15T11:00:00Z; last modified (UTC): 2026-11-15T12:00:00Z.

- Azure Monitor source condition Fired
  - Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/New; selected subscription: 11111111-1111-1111-1111-111111111111.
  - Finding ID: finding-f4f2691ced23bd2eb8ed24b1404127b856fd9c3a74314079747120eff4b093ad; evidence: [ev-8dee4498e347ae992c6dec901060fe7841702dae554d035da79d21fd89a241f9](run.json).
  - Signal: Metric; condition: Fired; workflow response: New; severity: Sev2; notification suppressed: false.
  - Target: Unknown; no affected resource established; started (UTC): 2026-11-15T11:00:00Z; last modified (UTC): 2026-11-15T12:00:00Z.

### Other saved observations

Inactive, excluded, unclassified, superseded or conflicting evidence is retained here. A saved source state alone does not establish an active summary; inspect its evidence and coverage gaps.

| Source entity | Disposition | Source state | Source updated (UTC) | Evidence |
| --- | --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/conflict | active | Fired | 2026-11-15T12:00:00Z | [ev-2b163fb5d4d81778cb5c2fb0e1b225092c42a11c5c11e4587ca17e2c9c733546](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/conflict | inactive | Resolved | 2026-11-15T12:00:00Z | [ev-e9d6278f8a873a2527fafcf343895b56ef5d64d29501459591fdbfdea260a168](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/FutureCondition | unclassified | FutureCondition | 2026-11-15T12:00:00Z | [ev-ff6bbdd348745a8e6f7919b1395b388d05a66e28d999ce8027ae1eae44a42214](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/Resolved | inactive | Resolved | 2026-11-15T12:00:00Z | [ev-4f5c973d3cb2a4f310d6edeecdd89d1dd6f44978524a3cf772ba403dcd540867](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/superseded | inactive | Resolved | 2026-11-15T12:00:00Z | [ev-107d9afa05200a3aa16020ee8b90fd353a22fa07938102cbd22186ab48dc286d](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/superseded | active | Fired | 2026-11-15T11:00:00Z | [ev-4fb4747361456c025cd94a7e7942227752e4bd562460f5d74e39944fe1f3fac0](run.json) |

## Resource Health findings

These findings retain Resource Health source observations, including conflicting or superseded observations. Consult current-state tables and timestamps before interpreting them as current faults.

| Category | Resource | Source observation | Finding ID | Evidence |
| --- | --- | --- | --- | --- |
| resource&#95;health&#95;event | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-04 | Resource Health impacting event&#59; current state undetermined | finding-ec71331ac506b315bc44950d8ba87378f703f52d20e5bfaf2a7db55054f45004 | [ev-9bfec019efc4d146cc2faf32320252eb6a3212a9e47ed932c5b2c3ffa450d2ef](run.json) |
| resource&#95;health&#95;issue | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-01 | Resource Health reported Unavailable | finding-091df1ca9c770f25d2611b5b77a8f3fd38c04fc985295a507111d3bcdaa9cb98 | [ev-278aafd423da08b341545924b01d68c7911227159e60bf8f67ef1c67355a01b5](run.json) |
| resource&#95;health&#95;issue | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-02 | Resource Health reported Degraded | finding-777ce2ee752256bbea3a9925aa70ea08b9133ba6cda709a28ecaec75fce4d12f | [ev-a9533034b28a447268852f37334b30d3b05b391ec8548df953164f42739dc4c4](run.json) |
| resource&#95;health&#95;issue | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07 | Resource Health reported Unavailable | finding-76f110baec274dcd65a557bff7b985fe4e9080be76b2d5ccc1b0fb51ed43dda5 | [ev-bc13f9c21702c4f955b3d789dc73bf5f3be3c4170f76e6c8cab70e5874d61443](run.json) |

## Resource Health issue states

| Resource | Current Resource Health state | In observed inventory | Evidence |
| --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-01 | Unavailable | true | [ev-278aafd423da08b341545924b01d68c7911227159e60bf8f67ef1c67355a01b5](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-02 | Degraded | true | [ev-a9533034b28a447268852f37334b30d3b05b391ec8548df953164f42739dc4c4](run.json) |

## Resource Health Available observations

| Resource | Current Resource Health state | In observed inventory | Evidence |
| --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-00 | Available | true | [ev-f03965cda921d638ce23af0188fbe799a5529cce363138ee67c56a1be9aa3598](run.json) |

## Unknown and not-checked resources

| Resource | Current Resource Health state | In observed inventory | Evidence |
| --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-03 | Unknown | true | [ev-c2451506df20a6e334ae1e5a3dbe168c6cde81a5a8d0c0809e4629713e286dd9](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-04 | undetermined | true | [ev-9bfec019efc4d146cc2faf32320252eb6a3212a9e47ed932c5b2c3ffa450d2ef](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-05 | not&#95;checked | true | — |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06 | undetermined | true | [ev-96c7408b8ad1e85da9cd45c237d7a5aefea21fcca68b7f9bdceb3bb72261978f](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07 | undetermined | true | [ev-218db3fe554d31d7e5a033b795252a7876d9049c17a23f9f8b39bf33b2aa55bf](run.json), [ev-bc13f9c21702c4f955b3d789dc73bf5f3be3c4170f76e6c8cab70e5874d61443](run.json) |

## Observed inventory (8 resources)

All observed resources are listed below. Conflicting inventory facts remain in the evidence index.

| Resource ID | Name | Type | Resource group | Location | Tags | Evidence |
| --- | --- | --- | --- | --- | --- | --- |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-00 | vm-00 | Microsoft&#46;Compute/virtualMachines | g | westeurope | display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93; | [ev-1082bbb061a34421a52e175289125f32997d06ca5fa89c6dbdf32d788d0f2d09](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-01 | vm-01 | Microsoft&#46;Compute/virtualMachines | g | westeurope | display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93; | [ev-6892a1927b2d1a542b6418a192e1168f8f311b16c431797174df42e1bcf0aa04](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-02 | vm-02 | Microsoft&#46;Compute/virtualMachines | g | westeurope | display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93; | [ev-349a0fd1912c098e85e03d418c1830ae6bacceef3e354c729a1b040764008254](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-03 | vm-03 | Microsoft&#46;Compute/virtualMachines | g | westeurope | display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93; | [ev-649d8c2628b5fc2112ce76a15f3c9bc1bb94b9602a7aedf19ef3f80679e12e67](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-04 | vm-04 | Microsoft&#46;Compute/virtualMachines | g | westeurope | display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93; | [ev-861d36581cadbaacd8ca9d71822933c780b289d110ecf423a3e2b15e8770c141](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-05 | vm-05 | Microsoft&#46;Compute/virtualMachines | g | westeurope | display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93; | [ev-b1ef467e128f5fdf959b3a5314603032934ebb94033d57a471377ee9e39f1947](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06 | vm-06 | Microsoft&#46;Compute/virtualMachines | g | westeurope | display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93; | [ev-e5302b8a48b87512955e33253a7176978dcec675105c26e3b4a457191329d9e4](run.json) |
| /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07 | vm-07 | Microsoft&#46;Compute/virtualMachines | g | westeurope | display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93; | [ev-c6b6d591f8b63c6c8c9f4e647f2acea171b1b82b55e1496c8261427705aae37a](run.json) |

## Coverage gaps and failed checks

| Source | Source entity | Resource attribution | Reason | Evidence |
| --- | --- | --- | --- | --- |
| advisor | — | Unknown; no affected resource established | not&#95;implemented | — |
| azure&#95;monitor | — | Unknown; no affected resource established | monitor&#95;30&#95;day&#95;retention&#95;and&#95;stateless&#95;limits | — |
| azure&#95;monitor | — | Unknown; no affected resource established | non&#95;atomic&#95;source&#95;snapshot | — |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/Acknowledged | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;OperationalInsights/workspaces/logs | workspace&#95;target&#95;only | [ev-f75453eb6397ccd2d9417a2bafbe81bdeee119876e372cc3f62d74124bacee7a](run.json) |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/conflict | Unknown; no affected resource established | conflicting&#95;operational&#95;evidence | [ev-2b163fb5d4d81778cb5c2fb0e1b225092c42a11c5c11e4587ca17e2c9c733546](run.json), [ev-e9d6278f8a873a2527fafcf343895b56ef5d64d29501459591fdbfdea260a168](run.json) |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/conflict | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-2b163fb5d4d81778cb5c2fb0e1b225092c42a11c5c11e4587ca17e2c9c733546](run.json) |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/conflict | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-e9d6278f8a873a2527fafcf343895b56ef5d64d29501459591fdbfdea260a168](run.json) |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/FutureCondition | Unknown; no affected resource established | unknown&#95;condition | [ev-ff6bbdd348745a8e6f7919b1395b388d05a66e28d999ce8027ae1eae44a42214](run.json) |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/FutureCondition | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-ff6bbdd348745a8e6f7919b1395b388d05a66e28d999ce8027ae1eae44a42214](run.json) |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/New | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-8dee4498e347ae992c6dec901060fe7841702dae554d035da79d21fd89a241f9](run.json) |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/Resolved | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-4f5c973d3cb2a4f310d6edeecdd89d1dd6f44978524a3cf772ba403dcd540867](run.json) |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/superseded | Unknown; no affected resource established | superseded&#95;operational&#95;evidence | [ev-4fb4747361456c025cd94a7e7942227752e4bd562460f5d74e39944fe1f3fac0](run.json) |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/superseded | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-107d9afa05200a3aa16020ee8b90fd353a22fa07938102cbd22186ab48dc286d](run.json) |
| azure&#95;monitor | /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/superseded | Unknown; no affected resource established | unknown&#95;resource&#95;attribution | [ev-4fb4747361456c025cd94a7e7942227752e4bd562460f5d74e39944fe1f3fac0](run.json) |
| resource&#95;health | — | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-03 | source&#95;unknown | [ev-c2451506df20a6e334ae1e5a3dbe168c6cde81a5a8d0c0809e4629713e286dd9](run.json) |
| resource&#95;health | — | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-04 | undetermined&#95;health&#95;evidence | [ev-9bfec019efc4d146cc2faf32320252eb6a3212a9e47ed932c5b2c3ffa450d2ef](run.json) |
| resource&#95;health | — | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-05 | missing&#95;health&#95;evidence | [ev-b1ef467e128f5fdf959b3a5314603032934ebb94033d57a471377ee9e39f1947](run.json) |
| resource&#95;health | — | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06 | historical&#95;health&#95;evidence | [ev-96c7408b8ad1e85da9cd45c237d7a5aefea21fcca68b7f9bdceb3bb72261978f](run.json) |
| resource&#95;health | — | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06 | missing&#95;current&#95;health&#95;evidence | [ev-96c7408b8ad1e85da9cd45c237d7a5aefea21fcca68b7f9bdceb3bb72261978f](run.json) |
| resource&#95;health | — | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06 | undetermined&#95;health&#95;evidence | [ev-96c7408b8ad1e85da9cd45c237d7a5aefea21fcca68b7f9bdceb3bb72261978f](run.json) |
| resource&#95;health | — | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07 | conflicting&#95;health&#95;evidence | [ev-218db3fe554d31d7e5a033b795252a7876d9049c17a23f9f8b39bf33b2aa55bf](run.json), [ev-bc13f9c21702c4f955b3d789dc73bf5f3be3c4170f76e6c8cab70e5874d61443](run.json) |
| resource&#95;health | — | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07 | undetermined&#95;health&#95;evidence | [ev-218db3fe554d31d7e5a033b795252a7876d9049c17a23f9f8b39bf33b2aa55bf](run.json), [ev-bc13f9c21702c4f955b3d789dc73bf5f3be3c4170f76e6c8cab70e5874d61443](run.json) |
| service&#95;health | — | Unknown; no affected resource established | non&#95;atomic&#95;source&#95;snapshot | — |
| service&#95;health | — | Unknown; no affected resource established | resource&#95;impact&#95;not&#95;enumerated | — |
| service&#95;health | — | Unknown; no affected resource established | service&#95;health&#95;30&#95;day&#95;update&#95;window | — |
| service&#95;health | /providers/Microsoft&#46;ResourceHealth/events/Billing | Unknown; no affected resource established | missing&#95;service&#95;region&#95;attribution | [ev-7a473cf82462d25a62e8a6047ebe9e6238d1f0b9f231510bfde597266ad01601](run.json) |
| service&#95;health | /providers/Microsoft&#46;ResourceHealth/events/FutureState | Unknown; no affected resource established | missing&#95;service&#95;region&#95;attribution | [ev-6240d4de7e6506dbd8340250571220bc0db38827ce2e1462255380f5bd176a87](run.json) |
| service&#95;health | /providers/Microsoft&#46;ResourceHealth/events/FutureState | Unknown; no affected resource established | unknown&#95;state | [ev-6240d4de7e6506dbd8340250571220bc0db38827ce2e1462255380f5bd176a87](run.json) |
| service&#95;health | /providers/Microsoft&#46;ResourceHealth/events/HealthAdvisory | Unknown; no affected resource established | unknown&#95;region&#95;attribution | [ev-c313f76f1100adbd3e2c810f513eb8490fb76a45fe21aab4a5fdd4bd988e225d](run.json) |
| service&#95;health | /providers/Microsoft&#46;ResourceHealth/events/PlannedMaintenance | Unknown; no affected resource established | unknown&#95;region&#95;attribution | [ev-0a2dde1d5635f2a78dbc2712ae497f2a557d809940d0ea92e06face19a3dbc7d](run.json) |
| service&#95;health | /providers/Microsoft&#46;ResourceHealth/events/Resolved | Unknown; no affected resource established | missing&#95;service&#95;region&#95;attribution | [ev-7a4cdf7d0aeac74118ddc6aad83063042a74320ee7695c82f18312e90a5fc15f](run.json) |
| service&#95;health | /providers/Microsoft&#46;ResourceHealth/events/SecurityAdvisory | Unknown; no affected resource established | missing&#95;service&#95;region&#95;attribution | [ev-c14ae959b5867d47be0d3aa8410f374b9e586954d2cb04c55147c523a534ff9e](run.json) |
| service&#95;health | /providers/Microsoft&#46;ResourceHealth/events/SecurityAdvisory | Unknown; no affected resource established | sensitive&#95;details&#95;not&#95;retrieved | [ev-c14ae959b5867d47be0d3aa8410f374b9e586954d2cb04c55147c523a534ff9e](run.json) |
| service&#95;health | /providers/Microsoft&#46;ResourceHealth/events/ServiceIssue | Unknown; no affected resource established | unknown&#95;region&#95;attribution | [ev-66da6eace263b71c71f03bfd00927e463baf4c58ce1f6baca594005ee7a23a6b](run.json) |

## Execution diagnostics

No execution diagnostics. Overall coverage remains incomplete.

## Evidence index (32 records)

Each reference identifies a record in the sibling JSON snapshot. Collection time and source time are distinct. Historical statuses cannot establish current health.

### ev-107d9afa05200a3aa16020ee8b90fd353a22fa07938102cbd22186ab48dc286d

- Record: [ev-107d9afa05200a3aa16020ee8b90fd353a22fa07938102cbd22186ab48dc286d](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/superseded; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: monitor&#95;alert; disposition: inactive.
- Collected (UTC): 2026-11-15T12:01:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): not recorded.
- Monitor signal: Metric; source condition: Resolved; workflow response: —; severity: —; notification suppressed: not recorded.
- Target attribution: unknown; monitor service last modified (UTC): not recorded.
- Limitations: unknown&#95;resource&#95;attribution.

### ev-2b163fb5d4d81778cb5c2fb0e1b225092c42a11c5c11e4587ca17e2c9c733546

- Record: [ev-2b163fb5d4d81778cb5c2fb0e1b225092c42a11c5c11e4587ca17e2c9c733546](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/conflict; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: monitor&#95;alert; disposition: active.
- Collected (UTC): 2026-11-15T12:01:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): not recorded.
- Monitor signal: Metric; source condition: Fired; workflow response: —; severity: —; notification suppressed: not recorded.
- Target attribution: unknown; monitor service last modified (UTC): not recorded.
- Limitations: unknown&#95;resource&#95;attribution.

### ev-4f5c973d3cb2a4f310d6edeecdd89d1dd6f44978524a3cf772ba403dcd540867

- Record: [ev-4f5c973d3cb2a4f310d6edeecdd89d1dd6f44978524a3cf772ba403dcd540867](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/Resolved; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: monitor&#95;alert; disposition: inactive.
- Collected (UTC): 2026-11-15T12:01:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): not recorded.
- Monitor signal: Log; source condition: Resolved; workflow response: —; severity: —; notification suppressed: not recorded.
- Target attribution: unknown; monitor service last modified (UTC): not recorded.
- Limitations: unknown&#95;resource&#95;attribution.

### ev-4fb4747361456c025cd94a7e7942227752e4bd562460f5d74e39944fe1f3fac0

- Record: [ev-4fb4747361456c025cd94a7e7942227752e4bd562460f5d74e39944fe1f3fac0](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/superseded; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: monitor&#95;alert; disposition: active.
- Collected (UTC): 2026-11-15T12:01:00Z; source reported (UTC): 2026-11-15T11:00:00Z; source occurred (UTC): not recorded.
- Monitor signal: Metric; source condition: Fired; workflow response: —; severity: —; notification suppressed: not recorded.
- Target attribution: unknown; monitor service last modified (UTC): not recorded.
- Limitations: unknown&#95;resource&#95;attribution.

### ev-8dee4498e347ae992c6dec901060fe7841702dae554d035da79d21fd89a241f9

- Record: [ev-8dee4498e347ae992c6dec901060fe7841702dae554d035da79d21fd89a241f9](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/New; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: monitor&#95;alert; disposition: active.
- Collected (UTC): 2026-11-15T12:01:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): 2026-11-15T11:00:00Z.
- Monitor signal: Metric; source condition: Fired; workflow response: New; severity: Sev2; notification suppressed: false.
- Target attribution: unknown; monitor service last modified (UTC): 2026-11-15T11:00:00Z.
- Limitations: unknown&#95;resource&#95;attribution.

### ev-e9d6278f8a873a2527fafcf343895b56ef5d64d29501459591fdbfdea260a168

- Record: [ev-e9d6278f8a873a2527fafcf343895b56ef5d64d29501459591fdbfdea260a168](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/conflict; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: monitor&#95;alert; disposition: inactive.
- Collected (UTC): 2026-11-15T12:01:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): not recorded.
- Monitor signal: Metric; source condition: Resolved; workflow response: —; severity: —; notification suppressed: not recorded.
- Target attribution: unknown; monitor service last modified (UTC): not recorded.
- Limitations: unknown&#95;resource&#95;attribution.

### ev-ff6bbdd348745a8e6f7919b1395b388d05a66e28d999ce8027ae1eae44a42214

- Record: [ev-ff6bbdd348745a8e6f7919b1395b388d05a66e28d999ce8027ae1eae44a42214](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/FutureCondition; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: monitor&#95;alert; disposition: unclassified.
- Collected (UTC): 2026-11-15T12:01:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): not recorded.
- Monitor signal: Log; source condition: FutureCondition; workflow response: —; severity: —; notification suppressed: not recorded.
- Target attribution: unknown; monitor service last modified (UTC): not recorded.
- Limitations: unknown&#95;condition&#44; unknown&#95;resource&#95;attribution.

### ev-f0b36799a130497539b5e28cc9caf1b7c837abc28e17a4f038883be622c0c2f4

- Record: [ev-f0b36799a130497539b5e28cc9caf1b7c837abc28e17a4f038883be622c0c2f4](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/Closed; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-00 (reported target only).
- Kind: monitor&#95;alert; disposition: active.
- Collected (UTC): 2026-11-15T12:01:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): 2026-11-15T11:00:00Z.
- Monitor signal: Metric; source condition: Fired; workflow response: Closed; severity: Sev2; notification suppressed: true.
- Target attribution: reported&#95;target; monitor service last modified (UTC): 2026-11-15T11:00:00Z.

### ev-f75453eb6397ccd2d9417a2bafbe81bdeee119876e372cc3f62d74124bacee7a

- Record: [ev-f75453eb6397ccd2d9417a2bafbe81bdeee119876e372cc3f62d74124bacee7a](run.json)
- Source: azure&#95;monitor; operation: Alerts&#46;GetAll; API version: 2019-03-01.
- Source entity: /subscriptions/11111111-1111-1111-1111-111111111111/providers/Microsoft&#46;AlertsManagement/alerts/Acknowledged; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;OperationalInsights/workspaces/logs (reported target only).
- Kind: monitor&#95;alert; disposition: active.
- Collected (UTC): 2026-11-15T12:01:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): 2026-11-15T11:00:00Z.
- Monitor signal: Log; source condition: Fired; workflow response: Acknowledged; severity: Sev2; notification suppressed: false.
- Target attribution: reported&#95;target; monitor service last modified (UTC): 2026-11-15T11:00:00Z.
- Limitations: workspace&#95;target&#95;only.

### ev-1082bbb061a34421a52e175289125f32997d06ca5fa89c6dbdf32d788d0f2d09

- Record: [ev-1082bbb061a34421a52e175289125f32997d06ca5fa89c6dbdf32d788d0f2d09](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-00
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-00; name: vm-00; type: Microsoft&#46;Compute/virtualMachines; group: g; location: westeurope; tags: display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93;.

### ev-6892a1927b2d1a542b6418a192e1168f8f311b16c431797174df42e1bcf0aa04

- Record: [ev-6892a1927b2d1a542b6418a192e1168f8f311b16c431797174df42e1bcf0aa04](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-01
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-01; name: vm-01; type: Microsoft&#46;Compute/virtualMachines; group: g; location: westeurope; tags: display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93;.

### ev-349a0fd1912c098e85e03d418c1830ae6bacceef3e354c729a1b040764008254

- Record: [ev-349a0fd1912c098e85e03d418c1830ae6bacceef3e354c729a1b040764008254](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-02
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-02; name: vm-02; type: Microsoft&#46;Compute/virtualMachines; group: g; location: westeurope; tags: display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93;.

### ev-649d8c2628b5fc2112ce76a15f3c9bc1bb94b9602a7aedf19ef3f80679e12e67

- Record: [ev-649d8c2628b5fc2112ce76a15f3c9bc1bb94b9602a7aedf19ef3f80679e12e67](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-03
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-03; name: vm-03; type: Microsoft&#46;Compute/virtualMachines; group: g; location: westeurope; tags: display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93;.

### ev-861d36581cadbaacd8ca9d71822933c780b289d110ecf423a3e2b15e8770c141

- Record: [ev-861d36581cadbaacd8ca9d71822933c780b289d110ecf423a3e2b15e8770c141](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-04
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-04; name: vm-04; type: Microsoft&#46;Compute/virtualMachines; group: g; location: westeurope; tags: display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93;.

### ev-b1ef467e128f5fdf959b3a5314603032934ebb94033d57a471377ee9e39f1947

- Record: [ev-b1ef467e128f5fdf959b3a5314603032934ebb94033d57a471377ee9e39f1947](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-05
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-05; name: vm-05; type: Microsoft&#46;Compute/virtualMachines; group: g; location: westeurope; tags: display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93;.

### ev-e5302b8a48b87512955e33253a7176978dcec675105c26e3b4a457191329d9e4

- Record: [ev-e5302b8a48b87512955e33253a7176978dcec675105c26e3b4a457191329d9e4](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06; name: vm-06; type: Microsoft&#46;Compute/virtualMachines; group: g; location: westeurope; tags: display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93;.

### ev-c6b6d591f8b63c6c8c9f4e647f2acea171b1b82b55e1496c8261427705aae37a

- Record: [ev-c6b6d591f8b63c6c8c9f4e647f2acea171b1b82b55e1496c8261427705aae37a](run.json)
- Source: inventory; operation: Resources&#46;List; API version: 2025-04-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07
- Kind: inventory; state: —; status role: —.
- Status ID: —
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): not recorded; source occurred (UTC): not recorded.
- Inventory facts: ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07; name: vm-07; type: Microsoft&#46;Compute/virtualMachines; group: g; location: westeurope; tags: display=&#60;b&#62;x&#60;/b&#62;&#124;&#91;click&#93;&#40;https&#58;//example&#46;test&#41;&#35; heading; owner=consultant; password=&#91;REDACTED&#93;.

### ev-f03965cda921d638ce23af0188fbe799a5529cce363138ee67c56a1be9aa3598

- Record: [ev-f03965cda921d638ce23af0188fbe799a5529cce363138ee67c56a1be9aa3598](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-00
- Kind: availability; state: Available; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-00/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): 2026-11-06T10:00:00Z; source occurred (UTC): not recorded.

### ev-278aafd423da08b341545924b01d68c7911227159e60bf8f67ef1c67355a01b5

- Record: [ev-278aafd423da08b341545924b01d68c7911227159e60bf8f67ef1c67355a01b5](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-01
- Kind: availability; state: Unavailable; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-01/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): 2026-11-06T10:00:00Z; source occurred (UTC): not recorded.

### ev-a9533034b28a447268852f37334b30d3b05b391ec8548df953164f42739dc4c4

- Record: [ev-a9533034b28a447268852f37334b30d3b05b391ec8548df953164f42739dc4c4](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-02
- Kind: availability; state: Degraded; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-02/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): 2026-11-06T10:00:00Z; source occurred (UTC): not recorded.

### ev-c2451506df20a6e334ae1e5a3dbe168c6cde81a5a8d0c0809e4629713e286dd9

- Record: [ev-c2451506df20a6e334ae1e5a3dbe168c6cde81a5a8d0c0809e4629713e286dd9](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-03
- Kind: availability; state: Unknown; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-03/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): 2026-11-06T10:00:00Z; source occurred (UTC): not recorded.

### ev-9bfec019efc4d146cc2faf32320252eb6a3212a9e47ed932c5b2c3ffa450d2ef

- Record: [ev-9bfec019efc4d146cc2faf32320252eb6a3212a9e47ed932c5b2c3ffa450d2ef](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-04
- Kind: event; state: undetermined; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-04/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: event-123; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): 2026-11-06T10:00:00Z; source occurred (UTC): 2026-11-06T10:00:00Z.

### ev-96c7408b8ad1e85da9cd45c237d7a5aefea21fcca68b7f9bdceb3bb72261978f

- Record: [ev-96c7408b8ad1e85da9cd45c237d7a5aefea21fcca68b7f9bdceb3bb72261978f](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06
- Kind: availability; state: Available; status role: historical.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/2026-11-05
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): 2026-11-06T10:00:00Z; source occurred (UTC): not recorded.

### ev-218db3fe554d31d7e5a033b795252a7876d9049c17a23f9f8b39bf33b2aa55bf

- Record: [ev-218db3fe554d31d7e5a033b795252a7876d9049c17a23f9f8b39bf33b2aa55bf](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07
- Kind: availability; state: Available; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): 2026-11-06T10:00:00Z; source occurred (UTC): not recorded.

### ev-bc13f9c21702c4f955b3d789dc73bf5f3be3c4170f76e6c8cab70e5874d61443

- Record: [ev-bc13f9c21702c4f955b3d789dc73bf5f3be3c4170f76e6c8cab70e5874d61443](run.json)
- Source: resource&#95;health; operation: AvailabilityStatuses&#46;ListBySubscriptionId; API version: 2025-05-01.
- Resource: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07
- Kind: availability; state: Unavailable; status role: current.
- Status ID: /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07/providers/Microsoft&#46;ResourceHealth/availabilityStatuses/current
- Event ID: —; additional event IDs: —.
- Collected (UTC): 2026-11-06T10:00:00Z; source reported (UTC): 2026-11-06T10:00:00Z; source occurred (UTC): not recorded.

### ev-0a2dde1d5635f2a78dbc2712ae497f2a557d809940d0ea92e06face19a3dbc7d

- Record: [ev-0a2dde1d5635f2a78dbc2712ae497f2a557d809940d0ea92e06face19a3dbc7d](run.json)
- Source: service&#95;health; operation: Events&#46;ListBySubscriptionId; API version: 2025-05-01.
- Source entity: /providers/Microsoft&#46;ResourceHealth/events/PlannedMaintenance; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: service&#95;health&#95;event; disposition: active.
- Collected (UTC): 2026-11-15T12:00:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): 2026-11-16T12:00:00Z.
- Service Health category: PlannedMaintenance; source status: Active; level: Warning; event source: ResourceHealth; sensitive: not recorded.
- Impact starts (UTC): 2026-11-16T12:00:00Z; mitigation (UTC): not recorded.
- Service: Compute &#60;script&#62;x&#60;/script&#62;&#124;&#91;link&#93;&#40;https&#58;//example&#46;test&#41;.
  - Region: Unknown region; status: Active; attribution: unknown; last updated (UTC): not recorded.
  - Region: West US; status: Active; attribution: selected&#95;subscription; last updated (UTC): 2026-11-15T12:00:00Z.
- Limitations: unknown&#95;region&#95;attribution.

### ev-6240d4de7e6506dbd8340250571220bc0db38827ce2e1462255380f5bd176a87

- Record: [ev-6240d4de7e6506dbd8340250571220bc0db38827ce2e1462255380f5bd176a87](run.json)
- Source: service&#95;health; operation: Events&#46;ListBySubscriptionId; API version: 2025-05-01.
- Source entity: /providers/Microsoft&#46;ResourceHealth/events/FutureState; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: service&#95;health&#95;event; disposition: unclassified.
- Collected (UTC): 2026-11-15T12:00:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): not recorded.
- Service Health category: ServiceIssue; source status: FutureState; level: —; event source: —; sensitive: not recorded.
- Impact starts (UTC): not recorded; mitigation (UTC): not recorded.
- Service/region attribution is unknown.
- Limitations: missing&#95;service&#95;region&#95;attribution&#44; unknown&#95;state.

### ev-66da6eace263b71c71f03bfd00927e463baf4c58ce1f6baca594005ee7a23a6b

- Record: [ev-66da6eace263b71c71f03bfd00927e463baf4c58ce1f6baca594005ee7a23a6b](run.json)
- Source: service&#95;health; operation: Events&#46;ListBySubscriptionId; API version: 2025-05-01.
- Source entity: /providers/Microsoft&#46;ResourceHealth/events/ServiceIssue; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: service&#95;health&#95;event; disposition: active.
- Collected (UTC): 2026-11-15T12:00:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): 2026-11-15T11:00:00Z.
- Service Health category: ServiceIssue; source status: Active; level: Warning; event source: ResourceHealth; sensitive: not recorded.
- Impact starts (UTC): 2026-11-15T11:00:00Z; mitigation (UTC): not recorded.
- Service: Compute &#60;script&#62;x&#60;/script&#62;&#124;&#91;link&#93;&#40;https&#58;//example&#46;test&#41;.
  - Region: Unknown region; status: Active; attribution: unknown; last updated (UTC): not recorded.
  - Region: West US; status: Active; attribution: selected&#95;subscription; last updated (UTC): 2026-11-15T12:00:00Z.
- Limitations: unknown&#95;region&#95;attribution.

### ev-7a473cf82462d25a62e8a6047ebe9e6238d1f0b9f231510bfde597266ad01601

- Record: [ev-7a473cf82462d25a62e8a6047ebe9e6238d1f0b9f231510bfde597266ad01601](run.json)
- Source: service&#95;health; operation: Events&#46;ListBySubscriptionId; API version: 2025-05-01.
- Source entity: /providers/Microsoft&#46;ResourceHealth/events/Billing; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: service&#95;health&#95;event; disposition: excluded.
- Collected (UTC): 2026-11-15T12:00:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): not recorded.
- Service Health category: Billing; source status: Active; level: —; event source: —; sensitive: not recorded.
- Impact starts (UTC): not recorded; mitigation (UTC): not recorded.
- Service/region attribution is unknown.
- Limitations: missing&#95;service&#95;region&#95;attribution.

### ev-7a4cdf7d0aeac74118ddc6aad83063042a74320ee7695c82f18312e90a5fc15f

- Record: [ev-7a4cdf7d0aeac74118ddc6aad83063042a74320ee7695c82f18312e90a5fc15f](run.json)
- Source: service&#95;health; operation: Events&#46;ListBySubscriptionId; API version: 2025-05-01.
- Source entity: /providers/Microsoft&#46;ResourceHealth/events/Resolved; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: service&#95;health&#95;event; disposition: inactive.
- Collected (UTC): 2026-11-15T12:00:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): not recorded.
- Service Health category: ServiceIssue; source status: Resolved; level: —; event source: —; sensitive: not recorded.
- Impact starts (UTC): not recorded; mitigation (UTC): not recorded.
- Service/region attribution is unknown.
- Limitations: missing&#95;service&#95;region&#95;attribution.

### ev-c14ae959b5867d47be0d3aa8410f374b9e586954d2cb04c55147c523a534ff9e

- Record: [ev-c14ae959b5867d47be0d3aa8410f374b9e586954d2cb04c55147c523a534ff9e](run.json)
- Source: service&#95;health; operation: Events&#46;ListBySubscriptionId; API version: 2025-05-01.
- Source entity: /providers/Microsoft&#46;ResourceHealth/events/SecurityAdvisory; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: service&#95;health&#95;event; disposition: active.
- Collected (UTC): 2026-11-15T12:00:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): 2026-11-15T11:00:00Z.
- Service Health category: SecurityAdvisory; source status: Active; level: Warning; event source: ResourceHealth; sensitive: true.
- Impact starts (UTC): 2026-11-15T11:00:00Z; mitigation (UTC): not recorded.
- Service/region attribution is unknown.
- Limitations: missing&#95;service&#95;region&#95;attribution&#44; sensitive&#95;details&#95;not&#95;retrieved.

### ev-c313f76f1100adbd3e2c810f513eb8490fb76a45fe21aab4a5fdd4bd988e225d

- Record: [ev-c313f76f1100adbd3e2c810f513eb8490fb76a45fe21aab4a5fdd4bd988e225d](run.json)
- Source: service&#95;health; operation: Events&#46;ListBySubscriptionId; API version: 2025-05-01.
- Source entity: /providers/Microsoft&#46;ResourceHealth/events/HealthAdvisory; selected subscription: 11111111-1111-1111-1111-111111111111.
- Resource attribution: Unknown; no affected resource established.
- Kind: service&#95;health&#95;event; disposition: active.
- Collected (UTC): 2026-11-15T12:00:00Z; source reported (UTC): 2026-11-15T12:00:00Z; source occurred (UTC): 2026-11-15T11:00:00Z.
- Service Health category: HealthAdvisory; source status: Active; level: Warning; event source: ResourceHealth; sensitive: not recorded.
- Impact starts (UTC): 2026-11-15T11:00:00Z; mitigation (UTC): not recorded.
- Service: Compute &#60;script&#62;x&#60;/script&#62;&#124;&#91;link&#93;&#40;https&#58;//example&#46;test&#41;.
  - Region: Unknown region; status: Active; attribution: unknown; last updated (UTC): not recorded.
  - Region: West US; status: Active; attribution: selected&#95;subscription; last updated (UTC): 2026-11-15T12:00:00Z.
- Limitations: unknown&#95;region&#95;attribution.


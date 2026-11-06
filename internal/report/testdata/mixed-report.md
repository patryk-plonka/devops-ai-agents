# Subscription Resource Health report

**INCOMPLETE coverage.** Advisor, Service Health, and Azure Monitor are not checked in S-01. No resource is classified overall OK.

Run ID: synthetic-run

Revision: 9

Execution outcome: **completed**

Started (UTC): 2026-11-06T10:00:00Z; updated (UTC): 2026-11-06T10:00:00Z; ended (UTC): 2026-11-06T10:00:00Z.

Tool version: test; schema version: 1; filtered values: 8.

Requested scope: Synthetic subscription.

Subscription: Synthetic subscription; ID: 11111111-1111-1111-1111-111111111111; tenant: 22222222-2222-2222-2222-222222222222; cloud: AzureCloud.

Evidence snapshot: [run.json](run.json). Match its run ID and revision to this report. If revisions differ, this report is stale.

## Source coverage

| Source | Status | Pages | Rejected records | Reason |
| --- | --- | --- | --- | --- |
| advisor | not&#95;checked | 0 | 0 | not&#95;implemented&#95;in&#95;s01 |
| azure&#95;monitor | not&#95;checked | 0 | 0 | not&#95;implemented&#95;in&#95;s01 |
| inventory | completed | 1 | 0 | — |
| resource&#95;health | completed | 1 | 0 | — |
| service&#95;health | not&#95;checked | 0 | 0 | not&#95;implemented&#95;in&#95;s01 |

Collector completion describes enumeration by that API. Inventory may omit child resources; an Available observation provides Resource Health evidence only. Failed checks and missing evidence do not establish resource faults.

## Observed issues and events

Findings retain source observations, including conflicting or superseded observations. Consult the current-state tables and timestamps before interpreting them as current faults.

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

| Source | Resource | Reason | Evidence |
| --- | --- | --- | --- |
| advisor | — | not&#95;implemented&#95;in&#95;s01 | — |
| azure&#95;monitor | — | not&#95;implemented&#95;in&#95;s01 | — |
| resource&#95;health | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-03 | source&#95;unknown | [ev-c2451506df20a6e334ae1e5a3dbe168c6cde81a5a8d0c0809e4629713e286dd9](run.json) |
| resource&#95;health | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-04 | undetermined&#95;health&#95;evidence | [ev-9bfec019efc4d146cc2faf32320252eb6a3212a9e47ed932c5b2c3ffa450d2ef](run.json) |
| resource&#95;health | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-05 | missing&#95;health&#95;evidence | [ev-b1ef467e128f5fdf959b3a5314603032934ebb94033d57a471377ee9e39f1947](run.json) |
| resource&#95;health | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06 | historical&#95;health&#95;evidence | [ev-96c7408b8ad1e85da9cd45c237d7a5aefea21fcca68b7f9bdceb3bb72261978f](run.json) |
| resource&#95;health | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06 | missing&#95;current&#95;health&#95;evidence | [ev-96c7408b8ad1e85da9cd45c237d7a5aefea21fcca68b7f9bdceb3bb72261978f](run.json) |
| resource&#95;health | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-06 | undetermined&#95;health&#95;evidence | [ev-96c7408b8ad1e85da9cd45c237d7a5aefea21fcca68b7f9bdceb3bb72261978f](run.json) |
| resource&#95;health | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07 | conflicting&#95;health&#95;evidence | [ev-218db3fe554d31d7e5a033b795252a7876d9049c17a23f9f8b39bf33b2aa55bf](run.json), [ev-bc13f9c21702c4f955b3d789dc73bf5f3be3c4170f76e6c8cab70e5874d61443](run.json) |
| resource&#95;health | /subscriptions/11111111-1111-1111-1111-111111111111/resourceGroups/g/providers/Microsoft&#46;Compute/virtualMachines/vm-07 | undetermined&#95;health&#95;evidence | [ev-218db3fe554d31d7e5a033b795252a7876d9049c17a23f9f8b39bf33b2aa55bf](run.json), [ev-bc13f9c21702c4f955b3d789dc73bf5f3be3c4170f76e6c8cab70e5874d61443](run.json) |
| service&#95;health | — | not&#95;implemented&#95;in&#95;s01 | — |

## Execution diagnostics

No execution diagnostics. Overall coverage remains incomplete.

## Evidence index (16 records)

Each reference identifies a record in the sibling JSON snapshot. Collection time and source time are distinct. Historical statuses cannot establish current health.

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


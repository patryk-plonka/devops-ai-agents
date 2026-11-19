# S-03 · Azure Advisor recommendations

[Full implementation plan](plan.md)

## What & Why

The consultant wants to see which suggestions Azure Advisor already has for the selected subscription during the daily health check. The report should help them prioritize a suggestion, identify the assessed resource when possible, and find the original recommendation in Azure.

## Starting Point

The CLI already collects inventory, Resource Health, Service Health and Azure Monitor in sequence. It saves filtered evidence after each page and publishes Markdown from the final saved revision. Today it labels Advisor as unchecked.

## Desired End State

The same command adds an Advisor section. If 30 resources share a recommendation type and status, the section shows one group of 30 instances. Every row has its own evidence reference. An interrupted list can still show the rows already collected and the point at which coverage became partial.

## Key Decisions

| Operator decision | Report behavior | Reason |
| --- | --- | --- |
| Daily review | Lead with New. Show Postponed, Dismissed, Completed and unknown statuses in a separate table. | Keeps the review queue short while accounting for returned instances. |
| Content | Retain structured facts only. | Limits sensitive content in JSON and Markdown. The operator can look up the action in Advisor using the instance ID. |
| Grouping | Group by recommendation type within each status, with a count and individual rows. | Repeated advice is easy to scan and each instance stays traceable. |
| Order | Rank groups by their highest reported impact. Show each row’s impact and place unknown impact last. | Surfaces high-impact suggestions without assigning the group maximum to every resource. |
| Attribution | Keep a valid instance in its group and display **Resource unknown** when attribution is uncertain. | Missing target data should be visible at the point of review. |
| Timing | Put **Last evaluated** on the row. Keep state-change and collection times in the evidence index. | The operator can judge the age of cached advice at a glance. |
| Empty live result | Accept a completed empty list after a same-scope portal comparison, review of a populated synthetic report and explicit operator confirmation. | Acceptance can proceed with an honest record of which cases were exercised live. |

## Scope and approach

Add the subscription-scoped Advisor list GET as the fifth sequential collector. Use the existing Azure SDK pipeline and page checkpoints. Derive instance findings from normalized evidence, then group them for display. The public API exposes both workflow status and evaluation time. [Microsoft Advisor list contract](https://learn.microsoft.com/en-us/rest/api/advisor/recommendations/list?view=rest-advisor-2026-10-01)

S-04 owns the overall resource OK classification. S-05 owns drafting. Recommendation generation, remediation, custom rules and historical comparisons are outside S-03.

## Phases at a glance

| Phase | Deliverable | Main risk |
| --- | --- | --- |
| 1. Advisor evidence | Scoped instance identities, structured facts, findings and gaps | Combining unrelated instances or assigning the wrong resource |
| 2. Read-only collection | One paged Azure operation and synthetic wire fixtures | Losing an accepted page when a later request fails |
| 3. Daily review in Markdown | Fifth collector wired into the CLI, grouped report and traceable example | Mixing suggestions into Resource Health findings or changing old reports |
| 4. Verification and operator acceptance | Five-collector checks and sanitized portal comparison | Claiming more live coverage than the subscription provided |

**Prerequisites.** The S-01 and S-02 acceptance records are available. Live acceptance requires an authorized subscription and an externally established Azure CLI session with Advisor read access.

**Planning estimate.** Three to five focused development sessions plus operator acceptance. This assumes the documented paging contract works in the acceptance subscription.

## Open risks and assumptions

- Azure’s list example omits a recommendation type ID on valid instances. Each such instance needs its own identifiable row even when the report cannot group it with peers.
- The service returns cached data across sequential pages. A portal view taken later may have different filters or updated data. Record any discrepancy that affects acceptance.

## Success criteria

- The consultant can prioritize New suggestions, see the per-instance facts and trace every displayed instance to the matching saved run.
- Partial reads preserve earlier evidence. Empty, inaccessible and interrupted reads have distinct report wording. Overall coverage stays incomplete.
- Required automated checks pass, and the operator records an explicit result for the live comparison.

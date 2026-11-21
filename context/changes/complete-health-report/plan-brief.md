# Complete health report — plan brief

[Full implementation plan](plan.md)

## What & Why

A consultant reviewing a subscription needs to know which resources passed the agreed checks, which have findings, and where evidence ran out. Today even a run that reads every source successfully ends with an incomplete banner.

## Starting Point

The CLI already saves inventory, Resource Health, Service Health, Azure Monitor and Azure Advisor in one private snapshot. Markdown already links findings to that evidence.

## Desired End State

“OK within the agreed checks” means the resource appeared in inventory, Resource Health reported it Available, and the four checks gave sufficient evidence to exclude a current issue in their stated scope. A reader can follow that conclusion to the rows and completed queries that support it. A resource with a Fired alert and missing health evidence shows both.

## Key Decisions Made

| Decision | Choice | Why |
| --- | --- | --- |
| 1A · Empty results | Accept a fully read, scoped list as evidence for that query. | Readers can verify the bounds of a negative result without assuming every resource has an alert rule or Advisor evaluation. |
| 2A · Service Health | Show an active incident once. Withhold OK wherever its impact cannot be ruled out. | Subscription notices often lack affected resource IDs. |
| 3A · Advisor | Queue New recommendations for review. Show Postponed beside an otherwise-OK resource. | Acknowledged recommendations retain their context. |
| 4A · Mixed evidence | Show a known finding alongside any outstanding check. | A failed read must preserve the alert the consultant can already act on. |
| 5A · Headline | Lead with coverage, then resource counts. | “Incomplete coverage · 1 resource OK, 1 unknown” makes a local gap visible even if every collector finished. |
| 6A · Exit codes | Preserve collection and publication exit codes. Expose coverage in JSON and CLI output. | Existing callers can distinguish successful execution from evidence completeness. |
| 7A · Acceptance | If the live run has zero eligible resources, verify the live reason and trace one qualifying synthetic resource. Record both separately. | The operator can accept a correct conservative result without claiming to have seen a live OK. |

## Scope

S-04 covers classification, the saved JSON and Markdown, CLI output and operator acceptance. The existing read-only Azure calls supply the evidence. Draft selection belongs to S-05. Baselines, custom checks and automated ticket submission are outside this change.

## Architecture / Approach

The health package computes each assessment once from normalized evidence and source completion records. The CLI saves the final assessment and publishes Markdown from that revision. Older snapshots keep their original report format.

## Phases at a Glance

| Phase | Deliverable | Main risk |
| --- | --- | --- |
| 1. Classify saved evidence | Per-resource assessments, reasons, evidence references and counts | A missing page or ambiguous target could look like a clean result. |
| 2. Publish classifications | Matching JSON, Markdown and CLI summaries | A publication failure could leave an affirmative classification in a failed revision. |
| 3. Verify and accept | Representative integration checks and a sanitized operator record | A synthetic OK could be mistaken for live evidence. |

S-02 and S-03 have operator acceptance. Phase 3 needs authorized read access to a representative subscription. Estimated effort is three to five development sessions plus operator availability.

## Open Risks & Assumptions

The existing APIs can leave incident impact unresolved across an entire subscription. Zero eligible OK resources is a valid outcome. The report retains the query windows, cached Advisor scope and collection times needed to interpret that result.

## Success Criteria (Summary)

- Every OK can be traced to inventory, current Resource Health and the applicable completed source queries.
- Findings survive partial reads, and the headline identifies outstanding checks even when the command exits successfully.
- The operator reconciles the report with source evidence and records which scenarios were observed live and which used fixtures.

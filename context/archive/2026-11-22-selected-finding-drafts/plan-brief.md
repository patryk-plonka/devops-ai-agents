# Selected finding drafts — plan brief

[Full implementation plan](plan.md)

## What and why

After reading a subscription health report, the consultant chooses which findings warrant a ticket or recommendation. S-05 turns those choices into editable Markdown files for manual submission. Each recipient sees the observation, what can be concluded from it, and what still needs checking.

## Starting point

S-04 already puts stable finding IDs in all four source sections. Its saved snapshot contains the evidence, coverage gaps and revision used to publish the report. This is enough to draft offline.

## Desired end state

The consultant runs `health draft` with the reviewed report path, a draft type and exact finding IDs. Selecting two findings produces two files beside the original artifacts, under `drafts/`. Each file traces back to the reviewed run. Generating it again preserves the consultant’s edits.

## Operator decisions

| Decision | Choice | Reason |
| --- | --- | --- |
| 1A · Selection | Copy full finding IDs from the report into the command. | The selection stays tied to the entries the consultant reviewed. |
| 2A · History | Allow an explicitly selected historical finding. Cite the later deciding evidence. | An earlier degradation may merit follow-up even after recovery. |
| 3A · Type | Choose ticket or recommendation once per command. | The consultant decides how to present the evidence to the client. |
| 4A · Files | Save one editable Markdown file per finding. | Each item can be reviewed and submitted independently. |
| 5A · Client details | Include the resource name and type when supported. Keep full Azure identifiers in the original report and snapshot. | The recipient can recognize a target while the consultant retains a private evidence trail. |
| 6A · Repeat requests | Keep the existing file and report its path. | Regeneration preserves manual edits. Moving the file allows a fresh copy. |
| 7A · Acceptance | Review one real saved finding. Exercise cases absent there with fixtures. | The consultant evaluates actual client wording and can identify which scenarios were simulated. |

## Scope and approach

The CLI reads the published schema 4 report and its sibling snapshot, verifies the pair, and resolves the entire selection. It builds each draft from the selected finding and the evidence needed to interpret it. A fired alert can be drafted even when a separate check for that resource is missing. A subscription notice with unknown resource impact stays at subscription scope.

The command writes private files locally. Azure collection stays in `health check`, and the consultant handles submission. Interactive selection, ticket-system integrations, and upgrades of older snapshots fall outside this slice.

## Phases

| Phase | Deliverable | Risk addressed |
| --- | --- | --- |
| 1. Resolve the reviewed selection | Verify the report and snapshot, then resolve exact IDs to evidence. | A stale report or an invalid ID could produce the wrong draft. |
| 2. Render and save the drafts | Ship the command, both templates and private files that preserve existing edits. | A draft could overstate impact, disclose a full identifier, or overwrite a working copy. |
| 3. Verify and accept | Exercise the normal CLI path and review a real draft for client use. | Fixture wording may miss issues that matter to the consultant. |

S-04 was accepted on 2026-11-21. Development can use checked-in synthetic artifacts immediately. Manual acceptance needs one real saved schema 4 report containing a selectable finding. Allow roughly three development sessions plus operator review as a planning estimate.

## Risks and assumptions

- Missing, conflicting or unattributed evidence can leave the target unknown. The draft should say so plainly. A workspace alert alone cannot establish that a particular VM is affected.
- Each output file is published independently. If storage fails midway, the command identifies completed files and the consultant can repeat the same selection.
- A real report containing zero findings can exercise the empty case. Operator acceptance stays pending until a real draft can be reviewed.

## Success criteria

- The consultant can trace each selected draft to its finding, evidence and original report revision.
- The draft communicates history, uncertainty and missing checks in wording suitable for manual submission.
- A repeat request preserves an edited draft, and the health report and snapshot retain their original contents.

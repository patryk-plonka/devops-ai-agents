# Selected finding drafts implementation plan

## Overview

S-05 completes the consultant’s report-to-draft workflow. After reviewing a published health report, the consultant names the findings to turn into editable tickets or recommendations. Every draft carries the facts and limitations needed to interpret that finding and a private trace back to the reviewed revision. See the [approved decisions](plan-brief.md).

## Current state analysis

- `internal/cli/cli.go:50` dispatches the collection command. That path creates a run, discovers the Azure session and ultimately delivers a report path on stdout. A draft invocation can consume this path in a separate process.
- `internal/runstore/store.go:72` publishes Markdown from committed JSON. Snapshot and report replacements happen separately. Failed publication can leave an older report beside a newer snapshot, and interruption can leave only a checkpoint.
- `internal/health/model.go:68` and `internal/health/evidence.go` already hold run identity, revision, findings, source evidence, times and gaps. All four report sections print individual finding IDs. Advisor type headings and Postponed rows are context. A Service Health finding can lack a resource assessment.
- `internal/health/assessment.go:180` distinguishes superseded Resource Health findings. These findings can have an empty source field and cite a single Resource Health evidence record. The saved observation includes all versions. The deciding evidence is a subset determined by source time and agreement.
- `internal/report/testdata/assessment/report.md:53` illustrates two different follow-ups. One resource has recovered from a historical degradation. Another has a Fired alert and missing Resource Health evidence. Selecting by assessment disposition would lose legitimate findings.

## Desired end state

From the repository tree, the consultant can run the following command after reviewing the named report. Substitute full values copied from the report for the placeholders.

```sh
go run ./cmd/devops-ai-agents health draft \
  --report "runs/<run-id>/report.md" \
  --type ticket \
  --finding "<first-finding-id>" \
  --finding "<second-finding-id>"
```

The two files appear at `runs/<run-id>/drafts/ticket-<finding-id>.md`. Choosing `--type recommendation` produces a separate working copy per selected finding. The consultant can edit or move each file independently. Running the same command again returns the path of an existing draft and preserves its bytes. Each draft quotes its run ID, revision, finding ID and evidence IDs so its origin remains traceable when copied out of the directory.

## Scope

S-05 covers offline drafting from published schema 4 runs, selection, evidence interpretation, private local delivery, documentation and operator acceptance. Schemas 1–3 retain their existing report renderers. Older snapshot migration, a new collector or schema, interactive selection, inferred grouping, generated remediation advice, and external submission are outside this change.

## Implementation approach

Keep filesystem access in `runstore`, the interpretation of normalized evidence in `health`, Markdown presentation in `report`, and command dispatch and diagnostics in `cli`. Reuse the saved run as the source for content. A rendered report contains private inventory and entire evidence tables. Copying its sections into a shareable draft would include unrelated data.

### Ordering and failure boundaries

Resolve and validate the complete request and prepare all draft contents before publishing a file. Once publication begins, each file has its own completion point. A later error can leave earlier complete files on disk. The retry rule must work for that mixed state and for two processes requesting the same draft.

## Phase 1: Resolve the reviewed selection

### Overview

Build the read-only path from a published report to selected findings and the evidence each draft may use. This phase can be verified on saved artifacts without exposing a partially functional public command.

### Changes required

**Open the existing run** · `internal/runstore/store.go` and a focused sibling file if needed.

**Intent.** Load the pair the operator reviewed without allocating a new run or advancing a checkpoint.

**Contract.** Accept the relative or absolute path to the repository’s `runs/<run-id>/report.md`, resolving relative paths from the invocation directory. Open the real run directory with the existing rooted filesystem conventions. Require regular sibling files, matching directory and snapshot run ID, supported schema 4, positive revision and a finalized snapshot. Verify that the Markdown belongs to the same run and revision before using the snapshot. Use the fixed report identity fields in its generated header, accounting for Markdown literal encoding of punctuation in production run IDs. Anchor the check to the header and reject ambiguous identity. Validate saved references with the same rules as report publication. Derive content from the saved snapshot. Reads must leave both artifacts byte-identical.

**Resolve exact selections** · `internal/health/evidence.go`, `internal/health/assessment.go` and the new selection logic in `internal/health`.

**Intent.** Resolve IDs against the run-wide finding list, including subscription-scoped, non-inventoried, historical and known findings on unknown resources.

**Contract.** Require at least one exact finding ID. Validate the entire string for each requested or saved finding ID used in a filename or trace (`^finding-[0-9a-f]{64}$`) and each evidence ID included in a trace (`^ev-[0-9a-f]{64}$`), including evidence cited for later observations or coverage gaps. Treat a repeated valid selection as one request in first-requested order.

Resolve each selection against the saved findings. Use the existing source-specific rules in `internal/health/normalize.go`, `operational.go` and `advisor.go` to derive the expected finding from the normalized evidence. Require agreement on finding ID, category, source and entity where recorded, target, and the exact set of supporting evidence IDs. A Resource Health finding identifies one original observation even when several observations describe the same resource and state. Operational and Advisor findings can cite multiple supporting rows. Additional rows that explain a conflict or later state belong in the draft as separately identified context. Reject the entire request before publication if any ID is malformed, ambiguous, missing or inconsistent with its evidence.

Only saved findings qualify for selection. Advisor group headings, evidence rows and assessment rows alone are ineligible. A finalized matching report from an incomplete or interrupted run can supply a supported finding. Preserve the recorded outcome and limitations.

**Gather context** · `internal/health/normalize.go:492`, `internal/health/assessment.go:245`, `internal/health/advisor.go`, `internal/health/operational.go`.

**Intent.** Collect the selected finding’s supporting facts, deciding evidence and applicable gaps while preserving the distinctions used in the report.

**Contract.** Resolve the selected evidence set, including conflicting siblings. Qualify source, query window, scope and collection status from the saved run. For a matched assessment carry relevant resource, group and subscription blockers. For findings outside inventory or without an established target, consult the finding, source entity, evidence limitations and applicable saved gaps directly. Preserve a run-wide incomplete label even when the particular gap is elsewhere. Summarize broader gaps without importing unrelated findings, resource names or IDs.

For Resource Health, use the existing deciding-evidence rule for the same target, including outside inventory. Show the originally selected state and the later deciding state with their source times and references. Report unresolved conflicts or missing timestamps as such. A later collection time alone cannot prove recovery. A reference to a later observation provides context within the selected draft.

### Success criteria

#### Automated verification

- Verify published schema 4 pairs, stale and checkpoint-only failures, real-format run IDs and confined reads with `go test -count=1 ./internal/runstore ./internal/report`.
- Verify exact selection, duplicates, unsupported references, historical evidence and known findings with missing proof using `go test -count=1 ./internal/health`.

## Phase 2: Render and save the drafts

### Overview

Deliver the complete offline command. The two formats serve different client conversations while sharing the same evidence, privacy and delivery rules.

### Changes required

**Draft Markdown** · New draft renderer in `internal/report`, alongside `internal/report/markdown.go:318`.

**Intent.** Produce wording the consultant can paste into a client ticket or recommendation, with enough local references to answer a follow-up.

**Contract.** A ticket names the observed condition, affected scope, established impact or uncertainty, and the next verification step. A recommendation states the opportunity or reason for review, supporting source facts and a proposed review step. Use deterministic wording grounded in the saved source. For example, a ticket can ask the recipient to verify the condition behind a Fired alert. A cached Advisor suggestion can justify reviewing the recommendation in Azure. Avoid inventing an outage, root cause, fix, savings figure or a recommendation description that the collector never retained.

| Source | Information the selected draft carries |
| --- | --- |
| Resource Health | Selected state or event, source and collection times, later deciding evidence when superseded, or the reason current state is undetermined. |
| Service Health | Notice category, status, timing, fixed update-window limit and supported service or region scope. State when affected resources were never enumerated. |
| Azure Monitor | Signal, Fired condition, recorded workflow and suppression, reported target qualification and 30-day query scope. A workspace target supports a workspace observation only. |
| Azure Advisor | Match the selected finding to its saved `AdvisorInstance` by finding ID. Use the instance summary for workflow status, category, impact, assessed target and evaluation time when available. Include cached-list scope and cite the selected finding’s supporting evidence IDs. If observations agree on New but disagree on impact, report New and explain that impact is unknown because the observations disagree. |

Include run outcome and coverage, relevant blockers, and a compact trace with stable local IDs and revision. Distinguish collection times from source times. Use “unknown” when a needed value is unavailable. For affected resources, expose only evidence-supported name and type, with qualified wording for a reported target. Resolve inventory facts by the verified target in this subscription. Withhold labels when conflicting facts leave them uncertain. A source entity’s name cannot stand in for an affected resource. Apply the existing text filter and literal Markdown rendering to allowed source values. Even an inventory name can contain a full Azure ID, so enforce the allowed content on the value itself. Keep full ARM paths, tenant and subscription IDs, tags, unrelated findings and local absolute paths out of the draft body. Evidence references must remain usable as text after copying.

**Private draft files** · `internal/runstore/store.go:157` and new draft publication logic in `internal/runstore`.

**Intent.** Preserve a single editable working copy per run, type and selected finding.

**Contract.** Create `drafts/` privately under the existing run and use deterministic filenames derived only from the validated type and finding ID. Use owner-only directory and file permissions on supported filesystems. Publish each complete file with an operation that cannot replace an existing destination. The report store’s ordinary replacement rename is unsuitable here. An existing regular draft is a successful reuse, even if edited. Report its path and continue with missing files in the same validated request. Reject symlinks, directories and other obstructions. Two simultaneous attempts must preserve whichever complete copy won publication. Clean up ordinary staging failures. Report retained files and pending selections after a mid-batch failure or interruption.

**CLI and help** · `internal/cli/cli.go:50`, a draft command in `internal/cli`, and `cmd/devops-ai-agents/main.go:28` for the existing signal contract.

**Intent.** Make drafting available after report review, including after the Azure CLI session has ended.

**Contract.** Dispatch `health draft --report <path> --type ticket|recommendation --finding <id> [--finding <id> ...]` before collection setup. Require an explicit type and selection. Help performs no filesystem or Azure work. Print one available draft path per unique finding to stdout in request order. Distinguish created and already-existing files on stderr. Exit 0 for help or successful delivery of all selected paths, 2 for usage, unsupported input, mismatched pair or invalid selection, 1 for storage, publication or output failure, and 130 for interruption. Keep collector exit code 3 specific to `health check`. If stdout fails, identify saved outputs on stderr. A draft request must never trigger discovery, authentication, collection, submission or a new run.

### Success criteria

#### Automated verification

- Verify both Markdown formats, four-source wording, trace IDs, uncertainty and identifier filtering with `go test -count=1 ./internal/report`.
- Verify private publication, mixed new and edited files, concurrent same-destination requests and failure recovery with `go test -race -count=1 ./internal/runstore`.
- Verify CLI selection, offline dispatch, paths, reuse diagnostics and exit behavior with `go test -count=1 ./internal/cli ./cmd/devops-ai-agents`.

## Phase 3: Verify the workflow and obtain operator acceptance

### Overview

Prove that a consultant can move from the normal four-source report to exactly the intended draft, then establish whether the wording works on a real finding.

### Changes required

**End-to-end verification** · `internal/cli/operational_integration_test.go:113`, `internal/cli/integration_test.go:628` and package test fixtures.

**Intent.** Verify collection, reporting, selection and local delivery together, including realistic incomplete evidence.

**Contract.** Use synthetic Azure CLI and ARM transports to collect and publish through `health check`. Pass the actual stdout report path and IDs shown in that report to the normal draft entry point. Fail the draft test on any attempted Azure operation. Mix selected and unselected findings across sources, test both types, compare output files to the requested set, and trace every included fact and qualification to the paired JSON. Include an existing S-04 pair in compatibility coverage. Assert the original files are byte-identical afterwards and Git ignores the draft paths.

**Operator instructions and record** · `README.md`, `AGENTS.md` and change-local `verification.md` at implementation time.

**Intent.** Document the collection and draft commands, exact exit meanings, edit-preserving retry, and how to check a draft against its private evidence. Capture the operator’s actual judgment.

**Contract.** Record synthetic and real outcomes separately. The operator uses any authorized, locally saved schema 4 report with a selectable finding. Manual submission is their decision after review. A sanitized acceptance record can name the source, draft type, trace result and wording corrections without client identifiers, live file paths or copied draft text.

### Success criteria

#### Automated verification

- Verify report-to-draft selection, immutable source artifacts and ignored outputs through the normal CLI using `go test -count=1 ./internal/cli`.
- Pass every repository validation command listed under Testing strategy with real packages included in the test and vulnerability audit.

#### Manual verification

- Review one real saved finding as a client-facing draft, trace it to the reviewed revision, and record the operator’s explicit acceptance and fixture-only scenarios.

## Testing strategy

| Scenario | Expected result |
| --- | --- |
| Two exact IDs plus a duplicate | Two files, in first-selection order. A mixed selection containing an unknown ID produces zero new files. |
| Two historical Degraded observations of one resource before a later Available observation | First verify the intact pair traces each historical finding to its own timestamp. In a synthetic copy, exchange their supporting evidence references while preserving both finding IDs. A request including an affected finding and an otherwise valid selection fails before publishing any draft. |
| Malformed saved finding or trace ID in a mixed selection | Exercise incorrect length, uppercase hex, a non-hex character and a path separator. Include an invalid contextual evidence ID. Even an exact match to a malformed saved ID must fail before any draft is created. |
| Older degraded VM now Available | Historical draft cites original and deciding evidence with times. An unresolved tie describes uncertainty. Exercise a target outside inventory too. |
| Fired alert with missing Resource Health | The alert remains draftable. The draft names the missing check and qualifies what Fired proves. |
| Subscription notice or uncertain Advisor target | The finding remains identifiable. Individual workload impact stays unknown. A Postponed evidence row alone is ineligible. |
| Two observations of one New Advisor instance report Low and High impact | Extend the scenario in `internal/health/assessment_test.go` (`TestAssessmentAdvisorWorkflowSurvivesMetadataDisagreement`) through both draft formats. Each shows New, explains the unknown impact, and cites both supporting evidence IDs. Reversing the input order preserves that conclusion. |
| Unrelated inventory or gap contains a private marker | It never appears in the draft. Include markup, credentials and an ARM path embedded in an otherwise allowed display field. |
| Edited ticket, new ticket and recommendation for the same finding | Preserve the edited bytes, create the missing requested files and keep the two types distinct. |
| Cancelled run, stale report, checkpoint only, older or future schema | Accept the finalized schema 4 pair with a supported finding and visible cancellation. Reject unsupported inputs before creating any drafts. Retain legacy rendering tests. |
| Publication interruption or failed stdout | Every published file is complete and recoverable. Repeating the selection preserves it. Synchronize interruption tests on an observable boundary. |

Run the repository checks with the pins in `context/foundation/development-tools.md` and `.github/workflows/checks.yml`.

```sh
go build ./...
go test -race -cover -count=1 ./...
go vet ./...
staticcheck ./...
govulncheck ./...
./scripts/check-format.sh
shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh
actionlint
```

### Manual acceptance procedure

1. Open a real local report and its sibling snapshot. Match the run ID and revision. Choose one reported finding, its full ID and the draft type appropriate for the client.
2. Run the draft command and check stderr for `Draft created`. If a draft already exists, choose another finding or type, or move the existing copy aside and generate a fresh draft. Read the fresh draft as if you had received the ticket. Check the action requested, supported target, evidence times, coverage limits and absence of unnecessary identifiers. Follow its finding and evidence IDs back to the same snapshot.
3. Add an edit to the draft and repeat the command. Confirm the CLI identifies the existing file and that the edit survives. Use synthetic fixtures for any absent historical, partial, privacy or source-specific scenarios.
4. Record which check used a real finding, what used fixtures, and the explicit acceptance decision. If no real report contains a selectable finding, leave the manual step pending.

## Compatibility and effort

The plan adds a local command and files under the existing ignored run directory. Existing schema 4 pairs can be used in place. No snapshot migration or scan is required. Resolve selected findings and evidence through per-run indexes so selecting several IDs does not require repeated full report reads. The representative workload is hundreds of resources per run. The plan introduces no new latency threshold.

## References

- [S-05 roadmap entry](../../foundation/roadmap.md#s-05-draft-only-selected-findings) and [PRD functional requirements](../../foundation/prd.md#functional-requirements)
- [S-04 operator acceptance](../../archive/2026-11-19-complete-health-report/verification.md)
- [Synthetic assessment report](../../../internal/report/testdata/assessment/report.md) and [matching snapshot](../../../internal/report/testdata/assessment/run.json)
- [Operator decisions and phase summary](plan-brief.md)

## Progress

> All steps begin pending. Preserve the numbered titles and append the closing commit SHA when a step lands in a commit.

### Phase 1: Resolve the reviewed selection

#### Automated

- [x] 1.1 Verify published schema 4 pairs, stale and checkpoint-only failures, real-format run IDs and confined reads with `go test -count=1 ./internal/runstore ./internal/report`. — dcf5292
- [x] 1.2 Verify exact selection, duplicates, unsupported references, historical evidence and known findings with missing proof using `go test -count=1 ./internal/health`. — dcf5292

### Phase 2: Render and save the drafts

#### Automated

- [x] 2.1 Verify both Markdown formats, four-source wording, trace IDs, uncertainty and identifier filtering with `go test -count=1 ./internal/report`. — dcf5292
- [x] 2.2 Verify private publication, mixed new and edited files, concurrent same-destination requests and failure recovery with `go test -race -count=1 ./internal/runstore`. — dcf5292
- [x] 2.3 Verify CLI selection, offline dispatch, paths, reuse diagnostics and exit behavior with `go test -count=1 ./internal/cli ./cmd/devops-ai-agents`. — dcf5292

### Phase 3: Verify the workflow and obtain operator acceptance

#### Automated

- [x] 3.1 Verify report-to-draft selection, immutable source artifacts and ignored outputs through the normal CLI using `go test -count=1 ./internal/cli`. — dcf5292
- [x] 3.2 Pass every repository validation command listed under Testing strategy with real packages included in the test and vulnerability audit. — dcf5292

#### Manual

- [x] 3.3 Review one real saved finding as a client-facing draft, trace it to the reviewed revision, and record the operator’s explicit acceptance and fixture-only scenarios. — dcf5292

<!-- IMPL-REVIEW-REPORT -->
# Implementation Review: Active Operational Alerts

- **Plan**: [Implementation plan](../plan.md)
- **Scope**: Phase 3 of 4 — CLI and Report Integration
- **Date**: 2026-11-15
- **Verdict**: APPROVED — F1 fixed
- **Findings**: 0 critical, 1 warning (fixed), 0 observations; 0 open
- **Progress**: Phase 3 has 3/3 completed criteria; the plan has 10/17 completed criteria. Phase 4 is next. This review did not change Progress.

## Verdicts

| Dimension | Verdict |
|-----------|---------|
| Plan Adherence | PASS |
| Scope Discipline | PASS |
| Safety & Quality | PASS |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS |

Production orchestration, reporting and committed-snapshot publication match the
plan. The user selected "Fix now" for F1. The schema-2 example now lives in its
own directory with the production filenames, documentation/readers are updated,
and a regression follows the actual relative links. All dimensions pass; no
findings remain open. The original issue and reproduction below are historical.

## Findings

### F1 — Schema-2 example evidence links open the schema-1 snapshot

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW — quick decision; fix is obvious and narrowly scoped
- **Dimension**: Plan Adherence
- **Location**: Originally `internal/report/testdata/operational-report.md:21`, also finding links at lines 51, 57, 63, 69, 99, 105 and 111; advertised by `README.md:153`.
- **Detail**: At initial review, the new report identified `synthetic-operational-run`, revision 17, but its snapshot and evidence links all target sibling `run.json`. That file is the preserved schema-1 example (`synthetic-run`, revision 9); the intended matching snapshot is named `operational-run.json`. The primary reviewer independently counted 100 links targeting `run.json` and verified that none of the seven operational finding evidence IDs exist in that linked file. Readers following the documented current example open the wrong run/revision and cannot trace its operational findings. Existing tests explicitly load the differently named schema-2 snapshot, so byte equality and serialization checks pass without checking link resolution. Actual run directories correctly contain `report.md` and `run.json`; production publication is unaffected.
- **Fix**: Move the schema-2 example into its own `testdata/operational/` directory containing `report.md` and `run.json`; update fixture readers and documentation paths, and add a regression that resolves the report's snapshot/evidence link target and checks its matching identity/revision and evidence IDs. Preserve the original schema-1 pair.
- **Decision**: FIXED — user selected "Fix now". The schema-2 pair was moved byte-for-byte to `internal/report/testdata/operational/{report.md,run.json}`. README, fixture documentation and both package readers use the new paths. `TestReportExampleLinksResolveToMatchingSnapshot` fails on the original layout and passes after the move, resolving the actual link targets and checking run identity/revision plus evidence IDs for both example schemas. No production code changed.

## Initial reproduction (before fix)

A read-only Python check loaded both snapshots and inspected the Markdown link
targets, without changing application files or the test suite:

```text
Schema-2 report identity: synthetic-operational-run 17
Linked sibling run.json: synthetic-run 9
Markdown links targeting run.json: 100
Operational finding references found in linked sibling: 0 of 7
```

The report's line 21 explicitly instructs the reader to match identity/revision
with the linked JSON, which makes the mismatch directly observable. The new
fixture README also describes the differently named files as a sibling pair.

## Scope and grounding

No feature commits after the plan finalization date identify this work. Review
therefore used the uncommitted tracked diff against HEAD plus new files, with
the earlier phase checkpoints/reviews separating Phase 3 changes from Phases
1–2. Existing S-01 commits, archives and private runs were excluded.

Phase 3 paths reviewed: `internal/cli/cli.go`, `cli_test.go`,
`collection_test.go`, `integration_test.go`, `operational_test.go`;
`internal/report/markdown.go`, `operational.go`, `markdown_test.go`,
`operational_test.go`, and report fixtures; `internal/runstore/operational_test.go`
and the existing publication path; README, AGENTS.md and
`context/foundation/development-tools.md`. New helper/test files implement the
planned integration concerns. Existing integration expectation updates are
necessary for four-source semantics; they do not claim Phase 4 stress coverage.
No unexplained extra implementation or missing production area was found.

The primary reviewer and two skill-required independent reviewers examined plan
drift, safety and pattern compliance. The plan-drift reviewer found F1; the
primary reviewer independently reproduced it. The safety reviewer confirmed no
additional substantive finding. No lessons register is present; the shared
10x-plan progress-format reference applies because the review-local reference
is absent.

## Verified implementation

| Area | Evidence and result |
|------|---------------------|
| Production wiring | Four named collectors run in the fixed order. New policies derive from the persisted run start. Query/request provenance is checkpointed before the respective first request. The native-default test uses fake CLI credentials and a synthetic transport, proving the production defaults reach both adapters without live Azure access. |
| Outcomes and diagnostics | All 16 source-failure combinations, including collector-local authentication failure, preserve independent collection and the expected exits. Source status/page/counter diagnostics remain filtered; stdout is the published path only. Cancellation and checkpoint failure retain saved evidence and stop subsequent reads. |
| Operational report | Status-derived unchecked wording, separate active/other-observation sections, source condition versus workflow/suppression, distinct categories and resource-attribution uncertainty match the contract. Operational evidence does not enter Resource Health tables. Query windows, source/collection timing, sensitive details and stateless/retention/workspace limits remain visible; Advisor is unchecked and coverage stays incomplete. |
| Rendering and references | Normalized source strings render literally, including markup canaries. Reordered source collections retain deterministic bytes without mutating the snapshot. All evidence remains in the index. Missing references reject publication. F1 affected the checked-in example's relative target and is fixed below; production mechanisms were unaffected. |
| Committed publication | Schema-1/schema-2 Save/JSON/render round trips retain the new facts and provenance. Publication uses saved facts after caller mutation, identifies the committed run/revision and preserves the older report when references fail. Existing schema-1 fixtures and module files have no diff. |
| Operator guidance | Current command, source access, query limits, exits and inspection guidance are updated, while S-01 acceptance/history remains explicitly historical and links point into the archive. Phase 4/live acceptance is still pending. F1's fixture-path correction is now applied to the current example guidance. |

## Verification

The primary reviewer reran every Phase 3 automated command against real packages
with writable temporary Go/module caches. `-count=1` avoids cached test results.
Both independent reviewers also ran the targeted race suite successfully.

| Command | Review result |
|---------|---------------|
| `go test -count=1 ./internal/cli ./internal/report ./internal/runstore` | PASS; 4.787s / 0.445s / 0.266s |
| `go test -race -count=1 ./internal/cli ./internal/report ./internal/runstore` | PASS; 22.961s / 5.337s / 1.793s |
| `go build ./...` | PASS |
| `./scripts/check-format.sh` | PASS |
| `git diff --check` | PASS |
| `git diff --exit-code -- go.mod go.sum internal/report/testdata/run.json internal/report/testdata/mixed-report.md` | PASS; dependencies and schema-1 fixture pair unchanged |

The immediately preceding implementation checkpoint in [change.md](../change.md)
records the passing full race/coverage suite, vet, Staticcheck, vulnerability
audit, ShellCheck and actionlint. Those were not represented as fresh review
runs. No automated failure was found; the additional relative-link check exposed
F1 beyond the existing fixture tests.

## Limits and next step

This is synthetic verification and code review. No operator session or live ARM
requests were used. Phase 3 has no manual gate. All Phase 4 manual rows remain
pending, and native multipage stress/process interruption/live acceptance are
Phase 4 work. Review did not change application source or canonical Progress.

The report is saved and `change.md` is stamped `impl_reviewed`. The user's F1 fix is recorded below. Canonical Progress is unchanged at 10/17; Phase 4
and all manual acceptance remain pending. Changes remain uncommitted.


## Triage verification — 2026-11-15

The user selected "Fix now". Only fixture paths/layout, documentation, fixture
readers and the link-resolution regression changed; production rendering and
stored fixture bytes remain unchanged. The regression first reproduced the
original layout failure (`synthetic-run` revision 9 instead of
`synthetic-operational-run` revision 17), with the legacy example passing. After
the move, all 100 schema-2 relative links resolve to the matching snapshot and
all linked evidence IDs exist in its evidence array and report index. README
links resolve to the dedicated example directory.

| Command/check | Result after fix |
|---------------|------------------|
| `go test -count=1 ./internal/report ./internal/runstore` | PASS |
| `go build ./...` | PASS |
| `go test -race -cover -count=1 ./...` | PASS; all six packages |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS; zero reachable/imported-package vulnerabilities; three required-module advisories outside called code |
| `./scripts/check-format.sh` | PASS |
| `shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh` | PASS |
| `actionlint` | PASS |
| `git diff --check` | PASS |
| Legacy fixture and dependency diff checks | PASS; original schema-1 pair, go.mod and go.sum unchanged |

Temporary writable Go/module/Staticcheck caches were used. The vulnerability
scan used permitted access to the public database. Azure tests stayed synthetic;
no live Azure access, acceptance claim, staging or commit occurred.

Triage complete: F1 FIXED; no skipped, accepted or pending findings. Verdict:
APPROVED with all dimensions PASS. Phase 4 remains next.

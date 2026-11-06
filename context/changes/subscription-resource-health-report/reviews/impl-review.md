<!-- IMPL-REVIEW-REPORT -->
# Implementation Review: Subscription Resource Health Report

- **Plan**: context/changes/subscription-resource-health-report/plan.md
- **Scope**: Full S-01 — phases 1–4 of 4, including review fixes and live-validation corrections
- **Date**: 2026-11-06
- **Verdict**: APPROVED
- **Findings**: 0 critical, 1 warning, 1 observation
- **Unresolved findings**: 0 — F1 and F2 fixed and regression-tested on 2026-11-06
- **Completion**: 18/18 Progress rows; operator acceptance confirmed; triage complete

## Verdicts

| Dimension | Verdict |
|-----------|---------|
| Plan Adherence | PASS |
| Scope Discipline | PASS |
| Safety & Quality | PASS — F1 and F2 fixed |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS — required checks pass; manual confirmation is recorded with scenario limits |

The original review approved two minor findings; their original severity counts
remain above for traceability. Both are now fixed following operator authorization,
and all dimensions pass. No critical defect, major plan drift, production write
path, or evidence/privacy boundary violation was identified in this review.

## Findings

### F1 — Cancellation during successful path delivery returns the earlier exit code

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW — quick decision; fix is obvious and narrowly scoped
- **Dimension**: Safety & Quality
- **Location**: internal/cli/cli.go:180
- **Detail**: After publishing the report, `finish` rechecks cancellation only if `fmt.Fprintln(a.Out, store.ReportPath())` fails. If interruption cancels the context while that write is in progress but the writer returns success, execution reaches `return code` with the earlier completed/partial code. This matters for redirected output or a temporarily blocked pipe: an observed interruption can exit 0 or 3 instead of the plan's 130 contract. A controlled writer cancelled the context and returned `len(b), nil`; the CLI returned 0. Previously saved JSON/Markdown remain intact, so this is a narrow cancellation/feedback defect rather than lost evidence. The prior stdout fix covers write errors, including cancellation with an error, but not this successful-write branch.
- **Fix**: Recheck cancellation after path delivery even when the write succeeds, return 130 with accurate saved-artifact feedback, and add a regression for cancellation during a successful write.
- **Decision**: FIXED on 2026-11-06 at the operator's request. Successful path delivery now rechecks cancellation, emits saved-report feedback, and returns 130. Permanent completed/partial-run regressions reproduce the earlier 0/3 exits and verify byte-identical published JSON/Markdown after cancellation.

### F2 — A report-path directory is described as an older saved report

- **Severity**: ℹ️ OBSERVATION
- **Impact**: 🏃 LOW — quick decision; fix is obvious and narrowly scoped
- **Dimension**: Safety & Quality
- **Location**: internal/runstore/store.go:54
- **Detail**: `ReportExists` treats any successful `Lstat("report.md")` as an existing report. When a directory obstructs report publication, the CLI's retained-artifact diagnostic says "An older report exists" even if no Markdown report has ever been published. A controlled run with a directory at the report path returned exit 1 and retained the JSON checkpoint correctly, but incorrectly identified that directory as an older report. This is a failure-feedback accuracy issue; it does not overwrite evidence or claim the current report was saved.
- **Fix**: Inspect the entry type without following symlinks and classify non-regular entries as an obstruction or indeterminate artifact, rather than claiming an older report exists; verify the CLI diagnostic for that case.
- **Decision**: FIXED on 2026-11-06 at the operator's request. `ReportExists` now accepts only regular files and returns an inspection error for other entry types without following symlinks. The CLI uses its existing indeterminate-report diagnostic. Permanent regressions cover directory-obstructed publication and missing/regular/directory/symlink/dangling-symlink classification.

## Review method and limits

The requested two parallel reviewers were launched for plan drift and safety/
pattern compliance. Both failed before delivering a review because of the
account usage limit. The primary reviewer completed both passes directly.
**This full review therefore has no fresh independent sub-agent corroboration.**
Earlier phase reviews remain available as history; their approvals were not
treated as proof that the current implementation has no remaining defects.

The plan was read fully. `change.md` started as `implemented`, updated 2026-11-06.
All four phases were selected because all 18 canonical Progress rows are checked.
No accepted `context/foundation/lessons.md` exists; the separate lesson capture
awaits user-authored fields. The manual-check investigation was read as evidence
about fixture provenance, not treated as an already accepted recurring rule.

Git has no commits: `git log --oneline --after=2026-11-05 -- .` exits 128 with
"current branch 'main' does not have any commits yet". All implementation paths
are untracked by the operator's instruction. Commit-range diff analysis is
unavailable. Scope was reconstructed from the approved plan, current source tree,
and the recorded implementation/fix history. Pre-existing foundation documents,
archive guides, and bootstrap-verification records were excluded from purported
new application changes. This limits historical attribution of individual edits.

The review did not change application/tests, re-run Azure collection, access the
operator's live login, alter Progress, stage files, commit, or archive the change.
Only this report and the required `impl_reviewed` change stamp/notes were written.

## Plan and implementation comparison

| Phase / planned area | Files examined | Verdict and evidence |
| --- | --- | --- |
| 1 — command, scope and process entry | `cmd/devops-ai-agents/main.go`, CLI source/tests, Azure session/subscription source/tests | MATCH. Required command/flags, exact UUID/full-name matching, stable ambiguity selection, no default-on-EOF, explicit failure categories, externally established Azure CLI user identity. Credential is subscription-bound with TenantID unset; challenges fail safely. Help avoids login and run creation. |
| 1 — run initialization and checks | `internal/runstore`, model, `.gitignore`, README, workflow, format script, module files | MATCH. Repository-root discovery, private unique run directories, rooted filesystem operations, initial snapshot before Azure work, atomic replacements, ignored output, real package checks and pinned tools. Go/module identity preserved; dependency fixes are documented. |
| 2 — adapters and read boundary | `internal/azure/{inventory,resourcehealth,paging,transport,record_decode}.go` and tests | MATCH. Native API versions/pagers, GET-only selected-scope routes, disabled provider registration, no filters/expansions, bounded retries/deadlines/pages, independent collectors, page checkpoints before further reads. Invalid records preserve valid siblings; malformed envelopes fail safely. |
| 2 — normalized evidence and coverage | `internal/health/{model,evidence,normalize,privacy,check}.go` and tests | MATCH. Filtered structured evidence and stable IDs, current/historical roles, case-insensitive joins, retained conflicting observations, explicit Unknown/missing/event/failure gaps, ordinary tags retained with defined credential filtering. Three sources stay not checked; no overall OK. |
| 3 — report and publication | `internal/report`, report fixtures, runstore publication tests, CLI report tests | MATCH in design. Normalized committed snapshots feed literal deterministic Markdown; all finding references resolve, all observed resources/evidence remain enumerable, run/revision match, JSON precedes report publication. F2 identifies a narrow artifact-classification defect in failure feedback. |
| 3 — interruption | Process entry/tests, orchestration, CLI finalization tests | MATCH in design. First interrupt stops Azure work and finalizes locally; second may force exit with checkpoints retained. Request/retry/failed-write paths are covered. F1 identifies a missed late cancellation branch after report publication. |
| 4 — full flow and operating guide | CLI integration tests/fake CLI, README, AGENTS, development tools, workflow, verification record | MATCH. Real discovery/credential subprocess/native SDK/health/storage/report paths over isolated synthetic producers; scale and adverse scenarios, private output exclusion, current operating instructions and honest live scenario limits. |

Helper files for decoding, paging, privacy/evidence, and safe debug logging fit
the planned package boundaries and test seams. The `--debug` addition was
explicitly requested by the operator during earlier review. The later manual
investigation is a requested change-specific document. No unjustified EXTRA,
missing product feature within S-01, new production endpoint/token flag, release
workflow, other-source collector, scheduling, database, or concurrency fan-out
was found.

Pattern comparison included the CLI's existing report/collection failure tests,
Azure session/subscription/transport and mixed-record tests, health orchestration
versus normalization tests, runstore publication tests, and process signal tests.
External interfaces remain at the boundaries that need injection; package roles
and standard Go test/error-handling conventions are consistent. No substantive
pattern or dependency-direction mismatch was found.

## Manual-discovered fixes rechecked

- Account decoding now uses `cloudName`. The independent literal CLI-shaped
  regression exercises discovery plus UUID/name matching and rejects invalid
  cloud/account metadata; the fake CLI matches the command output shape.
- Resource Health accepts observed lowercase `$skiptoken`/`$top` pagination and
  the supported token alias with strict duplicate/type/scope/API checks.
  Alias normalization prevents bypass of repeated-link detection. Both service-
  shaped boundary tests and full-flow fixtures cover the corrected shape.
- The privacy assertion now scans the actual committed JSON bytes. It no longer
  hides unknown fields by decoding and re-serializing the normalized model.
- The supplied tenant/subscription ID confusion is documented separately from
  application bugs. The operating checklist uses an explicit `SubscriptionID`
  column label, avoiding Azure CLI's hidden `id` table column.

## Automated verification

Duplicate repository commands across phases were verified once. The fresh full
race/coverage run covers all phase-specific CLI/Azure/health/report/runstore
package groups, including their transport, malformed-record, cancellation,
persistence, revision/reference, scale and process integration assertions.
Writable temporary Go/module/Staticcheck caches were used where needed.

| Command / criterion | Result | Actual output or evidence |
| --- | --- | --- |
| `go build ./...` | PASS | Exit 0, no output. |
| `go test -race -cover -count=1 ./...` | PASS | All six packages `ok`; process 57.1%, Azure 87.6%, CLI 93.1%, health 94.2%, report 98.6%, runstore 81.5%; no races. |
| `go vet ./...` | PASS | Exit 0, no output. |
| `staticcheck ./...` | PASS | Exit 0, no output. |
| `govulncheck ./...` | PASS after permitted network retry | Zero vulnerabilities affecting code and zero imported-package vulnerabilities; three advisories in required modules outside called code. Initial sandbox DNS failure was not counted as a passing scan. |
| `./scripts/check-format.sh` | PASS | Exit 0, no output. |
| `shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh` | PASS | Exit 0, no output. |
| `actionlint` | PASS | Exit 0, no output. |
| Git exclusion of generated JSON/Markdown | PASS | Actual live artifact paths match `.gitignore:1:/runs/`; `git ls-files -- runs/` is empty. Full suite also creates real synthetic artifacts in an isolated Git repository and verifies exclusion/status. |
| Eligibility of documentation/source fixtures | PASS | `git check-ignore --no-index` returns the expected exit 1 with empty output for README, AGENTS, integration source, fake CLI, report fixtures, and verification record. |

CI matches local commands, has read-only repository permissions, no Azure
credentials or live artifact uploads, and no release scope. A GitHub-hosted
workflow execution is not claimed. Known module advisories outside called code
are recorded as audit limits rather than new reachable-vulnerability findings.

## Targeted finding probes

The two additional review probes were added only to a temporary copy of
`internal/cli/report_test.go` through a Go overlay:

```sh
go test -overlay=/tmp/s01-full-review-5t7eh8fm/overlay.json \
  -count=1 -run '^TestFullReview' ./internal/cli
```

Both probes fail at their intended behavioral assertions:

- `TestFullReviewCancellationDuringSuccessfulPathDelivery`: a writer cancels the
  run context and reports a successful full write; observed exit 0, expected 130.
- `TestFullReviewReportDirectoryIsNotAnOlderReport`: a real runstore has a
  directory at `report.md`; observed exit 1 with "An older report exists" despite
  there being no previously published report.

These are newly demonstrated edge cases, not compilation failures or failures
of the required baseline suite. Both use synthetic data and temporary artifacts;
the overlay leaves repository application/test files unchanged. The existing
tests cover cancellation with failed stdout delivery and report rename failure,
but do not assert these two successful-write/entry-classification outcomes.

## Manual verification and completion

The four manual Progress rows 4.4–4.7 are checked on the operator's explicit
confirmation, "all manual tests are ok", following the live fixes and ID
clarification. `verification.md` distinguishes agent-observed successful runs
and Git exclusion from operator-reported source comparison, report acceptance,
and live Ctrl+C behavior. This is valid acceptance evidence; no rubber-stamped
or silently skipped manual row was found.

Guest access, unhealthy-resource scenarios, and additional adverse API cases
remain synthetic-only where not explicitly live-exercised. The agent's live
runs had no issue findings, and live finding-reference usability is consequently
not overstated. Full four-source coverage and overall OK classification remain
outside S-01, even after a completed collection and accepted implementation.

Progress remains 18/18; this review does not rewrite accepted manual outcomes or
close findings automatically. Work remains uncommitted, with SHA-less Progress
rows by instruction. The change is stamped `impl_reviewed` to record this review.

## Triage resolution — 2026-11-06

The operator requested "fix both". F1 and F2 are fixed with minimal changes to
`internal/cli/cli.go` and `internal/runstore/store.go`; permanent regressions are
in their existing test files. The successful-delivery cases extend the existing
artifact-preservation test rather than relying on the temporary review overlay.

Before the fixes, the new completed/partial successful-write cases returned 0/3
instead of 130, the CLI misidentified a directory as an older report, and entry
classification accepted directories and both symlink types. All targeted cases
pass after the fixes. Late cancellation preserves the already published run
outcome/revision and artifacts; it does not claim publication failed or rewrite
the saved evidence after delivering its path.

The complete required checks were rerun after the fixes; results are recorded in
`../follow-ups/review-fixes.md`. No new live Azure collection was needed. All
18 Progress rows and existing manual acceptance remain unchanged. Work remains
uncommitted; no pending full-review decisions remain.

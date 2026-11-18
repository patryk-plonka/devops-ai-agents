<!-- IMPL-REVIEW-REPORT -->
# Implementation Review: Active Operational Alerts

- **Plan**: [Implementation plan](../plan.md)
- **Scope**: Phase 1 of 4 — Evidence and Coverage Contracts
- **Date**: 2026-11-15
- **Verdict**: APPROVED — F1 fixed
- **Findings**: 0 critical, 1 warning (fixed), 0 observations; 0 open
- **Progress**: Phase 1 has 4/4 completed criteria; the plan has 4/17 completed criteria. Phase 2 is next. Progress was not changed by this review.

## Verdicts

| Dimension | Verdict |
|-----------|---------|
| Plan Adherence | PASS |
| Scope Discipline | PASS |
| Safety & Quality | PASS |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS |

F1 was fixed after the user selected "Fix now" during triage. Original input enums now determine source meaning; filtering still applies to persisted labels. All dimensions pass for Phase 1, with 48 new adversarial regression cases and the full race suite passing. The original reproduction below records the defect before correction. The separately disclosed Go toolchain vulnerability audit remains outstanding.

## Findings

### F1 — Text filtering promotes unknown source enums into active findings

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW — quick decision; fix is obvious and narrowly scoped
- **Dimension**: Plan Adherence; Safety & Quality
- **Location**: Fixes in `internal/health/operational.go:106`, `:114`, `:167`, `:203`, `:211`, `:219`, `:222`; regression in `internal/health/operational_test.go:391`.
- **Detail**: At initial review, both operational normalizers constructed filtered facts with `SafeText`, then classified those facts. Removing controls or Unicode format characters could turn a source value that was not a recognized enum into a recognized value. For example, Monitor condition `"Fi\u200bred"` became `"Fired"` and created an active finding. The same occurred for signal `"Me\ntric"`, Service Health status `"Act\nive"`, and category `"Service\nIssue"`. Every reproduced case persisted active evidence, produced one finding, incremented no unclassified counter, and recorded no unknown-enum gap. This violated the plan's exact source-condition requirement and its treatment of unknown domain facts. Regional status checks had the same filter-before-classify order. Existing Resource Health classifies its raw source state before accepting its meaning.
- **Fix**: Classify original input enums using exact matching independently of filtering persisted text; retain unrecognized values as unclassified evidence with explicit gaps. Add regression cases for newline, terminal-sequence, and Unicode format-character transformations across both sources and regional status.
- **Decision**: FIXED — user selected "Fix now". Source category/status, Monitor signal/condition/workflow/severity, and regional status checks now use original input enums. Persisted text remains filtered. The regression failed on the original implementation and passes after correction. No commit was made.

## Initial reproduction (before fix)

The primary reviewer and both independent reviewers reached the same finding. Temporary Go test overlays added adversarial tests without modifying repository source or its test suite. The primary reproduction ran:

```sh
GOCACHE=/private/tmp/active-alert-report-cache/go-build \
GOMODCACHE=/private/tmp/active-alert-report-cache/go-mod \
go test -overlay /private/tmp/active-alert-report-review/overlay.json \
  -count=1 -run '^TestReviewExactSourceEnums$' -v ./internal/health
```

All four subcases failed the expected assertion that an unrecognized enum produces no active finding and remains unclassified. Observed output:

| Input | Saved state | Disposition | Findings | Unclassified records | Limitations |
| --- | --- | --- | --- | --- | --- |
| Monitor condition `Fi\u200bred` | Fired | active | 1 | 0 | none |
| Monitor signal `Me\ntric` | Fired | active | 1 | 0 | none |
| Service Health status `Act\nive` | Active | active | 1 | 0 | none |
| Service Health category `Service\nIssue` | Active | active | 1 | 0 | none |

The overlay was a temporary diagnostic artifact. Its cases are now covered by the tracked regression `TestOperationalEnumMeaningIsIndependentOfTextFiltering`.

## Scope and plan adherence

The review compared the uncommitted tracked diff against HEAD plus new `internal/health/operational.go` and `operational_test.go`. Existing commits belong to S-01 and were excluded from this change's scope. Planning/review documents were treated as supporting context rather than application changes. Two skill-requested independent reviewers examined plan drift and safety/patterns; the primary reviewer checked their finding and independently reproduced it.

- Schema 2, typed operational facts, optional targets, query provenance and distinct counters are present. Evidence keeps source operation/API, identity, selected subscription and timestamps.
- Time-window construction, exact Service Health cutoff, independent target validation, foreign identifier omission, sensitivity limits and lack of invented resource impact match the plan.
- Duplicate/conflict handling uses source updates, preserves evidence and gaps, and keeps operational finding identity stable. Resource Health semantics and explicit source dispatch are preserved.
- Named collectors preserve sequential execution, independent failures, cancellation, save-failure precedence and source metadata. Outcomes use the supplied collectors; coverage remains incomplete.
- Production still supplies inventory and Resource Health only. Shared test constructors inject the future sources, including the integration constructor through `testApp`. Both Azure coordinator test callers were migrated.
- `privacy.go` is reused rather than modified; existing privacy/normalization regressions remain in the suite. This is not missing implementation.
- The report golden-test adjustment explicitly selects schema 1 to retain the existing fixture pair. It is necessary compatibility support for the schema change, not premature Phase 3 report integration.
- Adapters, production four-source wiring, operational Markdown sections and live acceptance belong to later phases and were not treated as omissions.

No other substantive drift, security, reliability, attribution, architecture or pattern finding was confirmed.

## Verification

All Phase 1 success-criteria commands were rerun against real packages. `-count=1` was added to package tests to avoid relying on cached successes. Temporary writable Go and module caches were used.

| Command | Result |
| --- | --- |
| `go test -count=1 ./internal/health` | PASS; 0.532s |
| `go test -count=1 ./internal/cli ./cmd/devops-ai-agents` | PASS; 5.468s / 0.375s |
| `go test -count=1 ./internal/azure` | PASS; 0.479s |
| `go build ./...` | PASS |
| `./scripts/check-format.sh` | PASS |
| `go test -count=1 ./internal/report ./internal/runstore` (additional compatibility checks) | PASS; 0.506s / 0.343s |

The initial adversarial overlay was additional review evidence and failed as described under F1. The new tracked regression now covers those cases and passes after the fix; original review timings above describe the pre-triage verification.

The earlier full race suite and static-check results are recorded in [change.md](../change.md). Every required repository check was rerun after the triage fix; results are recorded below. The `govulncheck` failure on pinned Go 1.27.3 (`GO-2026-6859`, scanner-reported fix Go 1.27.4) remains outstanding. This is a previously disclosed toolchain issue, not a vulnerability introduced by the Phase 1 diff.

Phase 1 has no manual gate. Later manual Progress rows remain pending. No live Azure session, ARM network call, historical run rewrite, or commit occurred. Application edits were limited to the user-authorized F1 fix and regressions. `change.md` remains stamped `impl_reviewed` as the review workflow requires; later phases remain unimplemented.

## Triage verification — 2026-11-15

The user selected "Fix now" for F1. `TestOperationalEnumMeaningIsIndependentOfTextFiltering` adds 16 cases across three input transformations (newline, terminal sequence, Unicode format character), for 48 subcases. It checks original source meaning, unknown-enum gaps and counters, safe serialized labels, and evidence references. Unknown condition/signal/category/status stays unclassified; unknown regional status or optional workflow/severity preserves an otherwise supported finding with an explicit gap. Existing exact-value classification, privacy, conflict and Resource Health tests remain enabled.

| Command | Result after fix |
| --- | --- |
| `go test -count=1 ./internal/health` | PASS |
| `go build ./...` | PASS |
| `go test -race -cover -count=1 ./...` | PASS; all six packages, health coverage 97.4% |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `./scripts/check-format.sh` | PASS |
| `shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh` | PASS |
| `actionlint` | PASS |
| `govulncheck ./...` | FAIL (exit 3); existing `GO-2026-6859` in Go 1.27.3; three module advisories outside called code |

Temporary writable Go/module/Staticcheck caches were used. The vulnerability check had network access to the Go vulnerability database. Azure tests remained synthetic and did not use the operator's session.

Triage complete: F1 FIXED; no skipped, accepted, or pending findings. Verdict after fixes: APPROVED for Phase 1. Canonical Progress remains 4/17, with Phase 2 next. Changes remain uncommitted.

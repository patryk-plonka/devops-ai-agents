<!-- IMPL-REVIEW-REPORT -->
# Implementation Review — Selected finding drafts

- **Plan** — [Selected finding drafts implementation plan](../plan.md)
- **Scope** — Phase 3 of 3
- **Date** — 2026-11-23
- **Verdict** — APPROVED
- **Findings** — 0 critical, 1 warning, 3 observations

The phase is in the working tree. The review used its integration tests,
operator documentation and verification record. Earlier phases were consulted
through the CLI and renderer paths used by those tests. Seven of eight plan
items are checked. Operator acceptance remains pending.

## Verdicts

| Dimension | Verdict |
| --- | --- |
| Plan Adherence | PASS |
| Scope Discipline | PASS |
| Safety & Quality | WARNING |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS |

## Findings

### F1 — The Git ignore test can write outside its scratch repository

- **Severity** — ⚠️ WARNING
- **Impact** — 🏃 LOW · Quick decision. Fix is obvious and narrowly scoped.
- **Dimension** — Safety & Quality
- **Detail** — Phase 3 reuses the Git test helper, which inherits the
  caller's environment and overrides three repository paths. An exported
  `GIT_COMMON_DIR` still redirects `git init`. With two temporary directories,
  the helper's command returned 0 and created `config` and object directories
  in the inherited common directory outside the scratch repository. A second
  probe pointed at a disposable bare repository. Its config changed from
  `core.bare=true` to `core.bare=false`.
- **Fix** — Strip inherited `GIT_*` variables from the child environment before
  adding the helper's scratch and configuration settings. Verify that a
  separate temporary repository's config stays byte-identical.
- **Decision**: FIXED (2026-11-23) · The scratch Git environment excludes inherited `GIT_*` variables. The regression preserves a separate bare repository's config.
- **Location** — [Git test helper](../../../../internal/cli/integration_test.go#L643)

### F2 — End-to-end assertions miss some draft qualifications

- **Severity** — ℹ️ OBSERVATION
- **Impact** — 🏃 LOW · Quick decision. Fix is obvious and narrowly scoped.
- **Dimension** — Plan Adherence
- **Detail** — The plan calls for tracing draft facts and qualifications to the
  paired JSON. The Service Health check ends at the regional status. It passes
  if the renderer omits the regional source update or uses collection time in
  its place. The test also passed when the renderer omitted the workload-impact
  uncertainty from Resource Health or Advisor drafts. These probes used
  temporary synthetic variations.
- **Fix** — Give the synthetic region a source update time distinct from
  the notice and collection times. Assert the full regional clause against the
  saved JSON and check the workload-impact and saved query-window qualifications
  alongside the existing evidence references.
- **Decision**: FIXED (2026-11-23) · The native workflow checks the regional source update from the saved JSON, both operational query limits and the Resource Health and Advisor impact qualifications.
- **Location** — [Regional assertion](../../../../internal/cli/draft_integration_test.go#L227), [Resource Health assertion](../../../../internal/cli/draft_integration_test.go#L194), [Advisor assertion](../../../../internal/cli/draft_integration_test.go#L241)

### F3 — The transposed-evidence test can pass on a stale report

- **Severity** — ℹ️ OBSERVATION
- **Impact** — 🏃 LOW · Quick decision. Fix is obvious and narrowly scoped.
- **Dimension** — Safety & Quality
- **Detail** — The test accepts any exit 2 with empty stdout and no drafts.
  Replacing the transposition with a change to the copied snapshot's revision
  still passed the test. Pair validation rejects that stale revision before
  reaching the evidence-binding check that the test is meant to verify.
- **Fix** — Keep the no-output assertions and require stderr to identify the
  selection failure caused by mismatched supporting evidence.
- **Decision**: FIXED (2026-11-23) · The test requires the supporting-evidence mismatch diagnostic as well as exit 2, empty stdout and no draft output.
- **Location** — [Transposed-evidence assertion](../../../../internal/cli/draft_integration_test.go#L280)

### F4 — Operator acceptance can review a previously edited draft

- **Severity** — ℹ️ OBSERVATION
- **Impact** — 🏃 LOW · Quick decision. Fix is obvious and narrowly scoped.
- **Dimension** — Success Criteria
- **Detail** — The operator procedure asks for a draft and then reviews the
  file returned on the first run. An existing working copy is returned
  successfully, so acceptance could review wording saved or edited before
  the current implementation.
- **Fix** — Require the first run to report `Draft created`. If a working copy
  already exists, choose another finding or draft type, or move the existing
  copy aside before generating a fresh one.
- **Decision**: FIXED (2026-11-23) · The first acceptance run must report `Draft created` before the operator reviews the wording and checks edit preservation.
- **Location** — [Operator procedure](../verification.md#operator-acceptance)

## Verification

| Check | Result |
| --- | --- |
| `go test -count=1 ./internal/cli` | PASS · CLI 11.277s |
| `go build ./...` | PASS |
| `go test -race -cover -count=1 ./...` | PASS · all six packages. Coverage was 71.4% command, 92.2% Azure, 91.2% CLI, 92.2% health, 93.3% report and 81.1% runstore. |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS · zero vulnerabilities in called code or imported packages. Three advisories in required modules outside called code. |
| `./scripts/check-format.sh` | PASS |
| `shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh` | PASS |
| `actionlint` | PASS |
| `git diff --check` | PASS |
| Temporary synthetic variations | Reproduced writes outside the scratch Git directory and both assertion gaps. |

The native test uses the published report path and printed finding IDs, checks
both draft types, verifies the exact file set, and confirms that drafting
attempts no Azure operations. It checks source artifact bytes and the Git
ignore rule under the test environment. The S-04 pair verifies compatibility
and edit-preserving reuse.
The README and repository guidelines describe the offline workflow, evidence
trace, and exit behavior.

Manual item 3.3 is unchecked. The verification record documents a pending
real-finding review and the synthetic scenarios separately. No live report or
operator result was used for this review.

## Triage outcome

All four findings were fixed during [full-slice triage](impl-review.md#triage-outcome).
Operator acceptance remains pending.

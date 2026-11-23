<!-- IMPL-REVIEW-REPORT -->
# Implementation Review — Selected finding drafts

- **Plan** — [Selected finding drafts implementation plan](../plan.md)
- **Scope** — Entire S-05 slice, phases 1–3
- **Date** — 2026-11-23
- **Verdict** — NEEDS ATTENTION
- **Findings** — 0 critical, 3 warnings, 4 observations

The review compared the working tree, including untracked files, with `32deb2c`.
The change has no commits after its plan date. All seven automated progress
items are checked. The explicit full-slice request includes the Phase 3 work
and its unchecked operator acceptance item.

## Verdicts

| Dimension | Verdict |
| --- | --- |
| Plan Adherence | WARNING |
| Scope Discipline | PASS |
| Safety & Quality | WARNING |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS · Automated checks passed. Manual item 3.3 is pending. |

## Findings

### F1 — Stalled stderr prevents graceful draft interruption

- **Severity** — ⚠️ WARNING
- **Impact** — 🔎 MEDIUM · A real tradeoff. Pause to reason through it.
- **Dimension** — Safety & Quality
- **Location** — [Process output wrapper](../../../../cmd/devops-ai-agents/main.go#L50), [draft publication diagnostics](../../../../internal/cli/draft.go#L134)
- **Detail** — The process wraps stdout with cancellation. The CLI writes the
  created or reused message to stderr synchronously before delivering the path.
  A built-process probe filled a stderr pipe, waited for a complete ticket to
  appear from a synthetic S-04 pair, and sent SIGINT. The process stayed blocked.
  The second SIGINT terminated it by signal, with the complete ticket retained.
  A stalled shared sink with `2>&1` can also block the first diagnostic.
- **Fix** — Make draft diagnostic delivery bounded on interruption and retain
  the saved path and pending selections when stderr is writable. Add a process
  regression synchronized on a published file with stderr still full.
  - Strength — Delivers the planned first-interrupt exit 130 even when a diagnostic sink stalls.
  - Tradeoff — Recovery messages need a best-effort path when the sink itself cannot accept data.
  - Confidence — HIGH · Reproduced against the built binary with a synthetic pair on macOS.
  - Blind spot — The additional process probe was macOS-only.
- **Decision**: FIXED (2026-11-23) · The first SIGINT cancels stdout and gives stderr a shared 250 ms window to report retained files and pending selections. Process tests passed with full stdout and full stderr pipes and with closed stdout.

### F2 — Absolute paths can fail when the repository has two spellings

- **Severity** — ⚠️ WARNING
- **Impact** — 🔎 MEDIUM · A real tradeoff. Pause to reason through it.
- **Dimension** — Plan Adherence
- **Location** — [Reviewed report path resolution](../../../../internal/runstore/reviewed.go#L29)
- **Detail** — The plan accepts an absolute or relative path to a reviewed
  report in the repository. `filepath.Rel` compares lexical paths. In a
  temporary repository on macOS, the built command accepted the relative path
  and rejected the absolute path via `/tmp` when its invocation directory was
  obtained via `/private/tmp`. Both paths identified the same report.
- **Fix** — Verify that the repository prefix in the supplied path identifies
  the invocation repository, then open the remaining `runs/<run-id>/report.md`
  components through the same rooted checks. Test both spellings of a temporary
  repository and retain the existing symlink rejection cases inside `runs/`.
  - Strength — Accepts equivalent absolute paths while keeping the run and its files confined.
  - Tradeoff — The path resolution has to distinguish the repository prefix from the run components.
  - Confidence — HIGH · The temporary S-04 pair succeeded with the relative path and returned exit 2 with the equivalent absolute path.
  - Blind spot — The additional process probe used the macOS temporary-directory alias.
- **Decision**: FIXED (2026-11-23) · The repository prefix is matched by filesystem identity when spellings differ. Tests accept both directions and reject a linked run directory reached through the alias.

### F3 — The Git ignore test can write outside its scratch repository

- **Severity** — ⚠️ WARNING
- **Impact** — 🏃 LOW · Quick decision. Fix is obvious and narrowly scoped.
- **Dimension** — Safety & Quality
- **Location** — [Git fixture environment](../../../../internal/cli/integration_test.go#L643)
- **Detail** — The fixture inherits `GIT_COMMON_DIR`. A prior Phase 3 probe
  against two disposable repositories showed that `git init` wrote outside
  the scratch repository and changed the other repository's `core.bare`
  setting. The same helper and environment handling are still in the tree.
- **Fix** — Remove inherited `GIT_*` variables before supplying the scratch
  settings. Verify that a separate temporary repository's config stays
  byte-identical.
- **Decision**: FIXED (2026-11-23) · The fixture filters inherited Git variables. The disposable bare repository's config stayed byte-identical.

### F4 — Drafts omit the future qualification from planned maintenance

- **Severity** — ℹ️ OBSERVATION
- **Impact** — 🏃 LOW · Quick decision. Fix is obvious and narrowly scoped.
- **Dimension** — Plan Adherence
- **Location** — [Service Health draft wording](../../../../internal/report/draft.go#L48), [saved future-notice decision](../../../../internal/health/operational.go#L505)
- **Detail** — A synthetic Active `PlannedMaintenance` notice with an impact
  start one day after the saved observation appeared as future in the report.
  The ticket led with “active notice” and the recommendation omitted the future
  qualification. Both drafts showed the start timestamp and source status.
- **Fix** — Carry the existing future-notice decision into both drafts. State
  that impact was scheduled to begin later as of the saved observation and
  continue to render the recorded Active status.
- **Decision**: FIXED (2026-11-23) · Both formats use the same saved-time decision as the report and preserve the source status and start time. Tests exercise starts before and after collection.

### F5 — End-to-end assertions miss some draft qualifications

- **Severity** — ℹ️ OBSERVATION
- **Impact** — 🏃 LOW · Quick decision. Fix is obvious and narrowly scoped.
- **Dimension** — Plan Adherence
- **Location** — [Service Health assertion](../../../../internal/cli/draft_integration_test.go#L227), [Resource Health assertion](../../../../internal/cli/draft_integration_test.go#L194), [Advisor assertion](../../../../internal/cli/draft_integration_test.go#L241)
- **Detail** — The native test asserts the regional status without checking
  its source update. Phase 3 probes passed with the regional update omitted
  or sourced from collection time. They also passed with the Resource Health
  or Advisor workload-impact uncertainty removed. The assertions are unchanged.
- **Fix** — Give the synthetic region a distinct source update and assert the
  complete regional statement against the saved JSON. Check the workload-impact
  and query-window qualifications alongside the evidence references.
- **Decision**: FIXED (2026-11-23) · The native test traces the distinct regional time to JSON and asserts Resource Health and Advisor impact uncertainty and the two operational query limits.

### F6 — The transposed-evidence test can pass on a stale report

- **Severity** — ℹ️ OBSERVATION
- **Impact** — 🏃 LOW · Quick decision. Fix is obvious and narrowly scoped.
- **Dimension** — Safety & Quality
- **Location** — [Transposed-evidence assertion](../../../../internal/cli/draft_integration_test.go#L280)
- **Detail** — The native test checks exit 2, empty stdout and no draft
  directory. A Phase 3 probe kept the references intact and incremented the
  copied snapshot revision. That fixture passed all three assertions after
  failing at pair validation, before reaching selection.
- **Fix** — Keep the no-output checks and require stderr to identify the
  mismatched supporting evidence during selection.
- **Decision**: FIXED (2026-11-23) · The mixed request must reach selection and identify the evidence mismatch, with empty stdout and no draft directory.

### F7 — Operator acceptance can review a previously edited draft

- **Severity** — ℹ️ OBSERVATION
- **Impact** — 🏃 LOW · Quick decision. Fix is obvious and narrowly scoped.
- **Dimension** — Success Criteria
- **Location** — [Operator acceptance procedure](../verification.md#operator-acceptance)
- **Detail** — The procedure reviews the file returned on the first run. A
  working copy can already exist for the requested type and finding, so the
  operator can see wording from an earlier version or from their own edits.
- **Fix** — Require `Draft created` for the first run. If a file already exists,
  select a fresh finding or type or move that copy aside before reviewing.
- **Decision**: FIXED (2026-11-23) · The implementation plan and operator procedure both require a fresh draft before the client-facing review and edit-preserving retry.

F3 and F5–F7 carry over the open findings in the [Phase 3 review](impl-review-phase-3.md).
The [Phase 1](impl-review-phase-1.md) and [Phase 2](impl-review-phase-2.md) review
fixes are present in the working tree. F1 covers a separate output stream from
the earlier stdout fix.

## Verification

| Check | Result |
| --- | --- |
| `go test -count=1 ./internal/runstore ./internal/report` | PASS · runstore 0.311s, report 0.508s |
| `go test -count=1 ./internal/health` | PASS · 0.573s |
| `go test -count=1 ./internal/report` | PASS · 0.566s |
| `go test -race -count=1 ./internal/runstore` | PASS · 2.333s |
| `go test -count=1 ./internal/cli ./cmd/devops-ai-agents` | PASS · CLI 11.793s, command 0.522s |
| `go test -count=1 ./internal/cli` | PASS · 11.799s |
| `go build ./...` | PASS |
| `go test -race -cover -count=1 ./...` | PASS · All six packages. Coverage was 71.4% command, 92.2% Azure, 91.2% CLI, 92.2% health, 93.3% report and 81.1% runstore. |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS · Zero vulnerabilities in called code or imported packages. Three advisories in required modules outside called code. |
| `./scripts/check-format.sh` | PASS |
| `shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh` | PASS |
| `actionlint` | PASS |
| `git diff --check` | PASS |
| Additional synthetic checks | Reproduced F1, F2 and F4. |

The review confirmed the offline entry point, full-batch reference preflight,
per-file no-overwrite publication, compatibility with the S-04 fixture, and
byte-identical source artifacts on retry. The collection and manual submission
boundaries and schema 4 storage match the plan. The operator's real-finding
review remains pending under [item 3.3](../plan.md#progress). Verification used
synthetic data throughout.

## Triage outcome

All seven findings were fixed with the recommended approaches on 2026-11-23.
The verdict above records the review before those changes. Manual item 3.3
remains pending.

| Post-fix check | Result |
| --- | --- |
| `go test -race -count=5 -run '^TestProcessDraftPathPipe$' ./cmd/devops-ai-agents` | PASS · Closed stdout and first SIGINT with either stdout or stderr full. |
| Focused runstore, report and CLI tests | PASS · Repository aliases, planned maintenance, Git fixture isolation, native draft facts and evidence mismatch. |
| `go test -race -cover -count=1 ./...` | PASS · All six packages. Coverage was 82.0% command, 92.2% Azure, 91.2% CLI, 92.2% health, 93.4% report and 81.4% runstore. |
| `go build ./...`, `go vet ./...`, `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS · Zero vulnerabilities in called code or imported packages. Three advisories in required modules outside called code. |
| Formatting, ShellCheck, actionlint and `git diff --check` | PASS |

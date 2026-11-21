<!-- IMPL-REVIEW-REPORT -->
# Implementation Review — Complete health report

- **Plan**: [Complete health report](../plan.md#phase-2-publish-classifications-in-markdown-and-cli)
- **Scope**: Phase 2 of 3
- **Date**: 2026-11-20
- **Verdict**: NEEDS ATTENTION
- **Findings**: 0 critical, 3 warnings, 1 observation
- **Review baseline**: working tree against `ad7fa4b`. Phases 1 and 2 are uncommitted. The review covered schema-4 activation, report validation and presentation, the CLI and storage publication path, and compatibility tests.

## Verdicts

| Dimension | Verdict |
| --- | --- |
| Plan Adherence | WARNING (F2, F3) |
| Scope Discipline | PASS |
| Safety & Quality | WARNING (F1, F2) |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS for the planned automated suite. Additional probes found the cases below. |

## Findings

### F1 — A closed stdout pipe bypasses the documented delivery exit

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW · Quick decision. The fix is narrowly scoped.
- **Dimension**: Safety & Quality
- **Location**: [Command signal handling](../../../../cmd/devops-ai-agents/main.go#L27), [CLI path delivery](../../../../internal/cli/cli.go#L186)
- **Detail**: A child process with five synthetic collectors and a closed reader on real stdout published both artifacts, then terminated with SIGPIPE. The path error handler never ran. README promises exit 1 and a delivery diagnostic when stdout cannot accept the path. The existing writer-error test exercises the CLI method without the operating system's pipe behavior. [Exit contract](../../../../README.md#L82)
- **Fix**: Handle SIGPIPE at the process entry so a closed pipe reaches the CLI write-error handler. Add a process regression that checks exit 1, the delivery diagnostic, and the preserved revision pair.
- **Decision**: FIXED (2026-11-20). The command lets the closed pipe reach the CLI write-error handler. A child-process regression verifies exit 1, the stderr diagnostic, and matching saved run ID and revision. The command package tests passed.

### F2 — Assessment validation accepts misattributed evidence and current findings marked historical

- **Severity**: ⚠️ WARNING
- **Impact**: 🔎 MEDIUM · Requires checking the affected resource and the role of each saved reference.
- **Dimension**: Plan Adherence
- **Location**: [Finding references](../../../../internal/health/assessment.go#L186), [Source evidence checks](../../../../internal/report/assessment.go#L129), [Postponed context checks](../../../../internal/report/assessment.go#L155)
- **Detail**: A schema-4 fixture with altered references rendered an OK VM using a Fired Monitor record for another VM. It also rendered another VM's Postponed Advisor context. Moving a current Fired finding to `historical_finding_ids` and updating the saved counts produced OK while the active summary still listed that alert. These probes exercised malformed snapshots. Normal collection cases in the planned suite produced consistent references.
- **Fix**: Verify the applicability of per-resource evidence, including supported Advisor resource-group scope. Require current applicable findings to remain visible and reserve historical references for superseded Resource Health findings.
  - **Strength**: Enforces the plan's reference contract at publication.
  - **Tradeoff**: Validation must use the health package's attribution and history rules so valid retained context still publishes.
  - **Confidence**: HIGH. All three fixture mutations reproduced the gap.
  - **Blind spot**: The probes cover altered saved data. No native collector path produced these inconsistencies during the review.
- **Decision**: FIXED (2026-11-20). Schema-4 publication validates the applicable resource for every check evidence reference and checks both finding lists against the shared Resource Health supersession rule. The regression cases reject wrong-target Monitor and Advisor evidence and omitted or miscategorized current findings. A positive test verifies that group-scoped Postponed context still publishes. The Phase 2 package suite, vet, Staticcheck, and formatting checks passed.

### F3 — Complete reports label ordinary query limits as coverage gaps

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW · Quick decision. The change is limited to schema-4 Markdown.
- **Dimension**: Plan Adherence
- **Location**: [Gap table](../../../../internal/report/markdown.go#L256)
- **Detail**: With complete coverage and zero blocking gaps, the renderer lists Monitor retention, Service Health window limits, and sequential collection under **Coverage gaps and failed checks**. The entries use the same unknown-attribution wording as actual gaps. A reader cannot tell which entries withheld a conclusion from that table.
- **Fix**: Use the saved `blocking` flag to distinguish coverage gaps from ordinary source limits and evidence notes in schema-4 Markdown.
- **Decision**: FIXED (2026-11-20). Schema-4 Markdown places blocking gaps and ordinary notes in distinct sections. Complete reports explicitly state when there are no decision-blocking gaps. The complete, mixed, and shared-incident regressions passed. The schema-4 Markdown fixture was refreshed from its existing JSON, and the schema 1–3 byte-compatibility checks passed.

### F4 — A zero end time passes the finalization check

- **Severity**: ℹ️ OBSERVATION
- **Impact**: 🏃 LOW · Quick decision. The fix is a validation check.
- **Dimension**: Safety & Quality
- **Location**: [Markdown finalization check](../../../../internal/report/markdown.go#L20)
- **Detail**: A schema-4 fixture with a zero-valued `ended_at` rendered Complete coverage and printed **ended (UTC): not recorded**. The renderer checks for a pointer, while the health package also requires a nonzero value for finalized assessments. [Health finalization rule](../../../../internal/health/assessment.go#L169)
- **Fix**: Reject a zero end time when validating a finalized schema-4 snapshot.
- **Decision**: FIXED (2026-11-20). Schema-4 publication now requires a nonzero end time. The regression reproduced the original invalid report and passed after the fix. All renderer tests passed, including the schema 1–3 fixtures.

## Verification

| Check | Result |
| --- | --- |
| `go test -count=1 ./internal/health ./internal/report ./internal/cli ./internal/runstore ./cmd/devops-ai-agents` | PASS. All five packages ran. This includes schemas 1–3 byte comparison, schema-4 complete and mixed reports, committed revisions, and the planned failure and interruption cases. |
| `go test -count=1 ./internal/azure` | PASS. Supplemental adapter regression check after switching the default schema. |
| `git diff --check` | PASS. |
| Supplemental synthetic process probe | Reproduced SIGPIPE after publication (F1). |
| Supplemental schema-4 report probes | Reproduced incorrect resource evidence and finding role (F2), routine limits under the gap heading (F3), and an accepted zero end time (F4). |

Phase 2 has no manual checklist. The Phase 3 operator acceptance item remains pending. Review probes used synthetic processes and temporary overlays.

## Triage outcome

All four findings were fixed on 2026-11-20. The verdict above describes the initial review. The original supplemental probes all passed after the fixes.

Build, race and coverage tests across all six packages, vet, Staticcheck, the vulnerability audit, formatting, ShellCheck, actionlint, and `git diff --check` passed. The audit found zero reachable or imported-package vulnerabilities and three advisories in required modules outside called code. The command also cross-built for Windows after the stdout fix.

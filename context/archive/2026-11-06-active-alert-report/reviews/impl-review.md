<!-- IMPL-REVIEW-REPORT -->
# Implementation Review: Active Operational Alerts

- **Plan**: [Implementation plan](../plan.md)
- **Scope**: All four phases against the S-01 baseline at `2ca6447`
- **Date**: 2026-11-18
- **Verdict**: APPROVED — all warnings fixed
- **Findings**: 0 critical, 3 warnings (all fixed), 1 observation (accepted), 0 pending
- **Progress**: 17/17 complete. Operator acceptance was recorded on 2026-11-18.

## Verdicts

| Dimension | Verdict |
| --- | --- |
| Plan Adherence | PASS (F2 and F3 fixed) |
| Scope Discipline | PASS |
| Safety & Quality | PASS (F3 fixed) |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS (F1 fixed, evidence limitation recorded in F4) |

## Findings

### F1 — Go 1.27.3 fails the required vulnerability audit

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW — quick decision. The fix is narrowly scoped.
- **Dimension**: Success Criteria
- **Location**: [go.mod:3](../../../../go.mod#L3) and [checks.yml:19](../../../../.github/workflows/checks.yml#L19)
- **Detail**: CI selects Go via the go.mod directive and runs `govulncheck ./...`. The local review used both toolchains. With `GOTOOLCHAIN=go1.27.3`, the audit reported GO-2026-6859 in the standard library and exited 3. With Go 1.27.4, it exited 0 with zero findings in called code or imported packages. This confirms the failure previously recorded in Phase 1. The review did not run GitHub CI or establish that the server-specific advisory is exploitable through this CLI. The failing required scan is enough to make this a verification issue.
- **Fix**: Update the toolchain selection used by CI to Go 1.27.4 or a verified patched release, then rerun the repository checks with that selection. Keep this infrastructure correction explicit because S-02 kept module files unchanged.
- **Decision**: FIXED — the user approved updating the Go directive to 1.27.4. CI resolves the directive through its existing setup-go configuration. The tool guidance reflects the new pin, and all required local checks passed on 1.27.4.

### F2 — A cancelled source can claim it made a request

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW — quick decision. The fix is narrowly scoped.
- **Dimension**: Plan Adherence
- **Location**: [check.go:39](../../../../internal/health/check.go#L39) and [operational.go:37](../../../../internal/report/operational.go#L37)
- **Detail**: The coordinator saves `RequestedAt` before invoking a collector. Cancellation during that checkpoint restores `not_checked` and `not_attempted` but retains the timestamp. An isolated test cancelled from the save callback, observed zero collector calls and exit 130, and rendered a nonempty “first requested” timestamp and an “actual API queryStartTime”. The date is a planned parameter, and the timestamp records preparation. The same wording can overstate an attempt that failed during local credential acquisition.
- **Fix**: Keep the configured policy in the snapshot, clear the timestamp when collection never starts, and label preflight timing and the configured query date distinctly from actual evidence collection.
- **Decision**: FIXED — the user approved clearing the preparation timestamp for unattempted sources while keeping the configured query. Markdown now labels preparation and identifies an unstarted collection. The regression covers cancellation during the pre-collection checkpoint for both operational sources.

### F3 — A service label can survive exclusively foreign attribution

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW — quick decision. The fix is narrowly scoped.
- **Dimension**: Safety & Quality
- **Location**: [health/operational.go:141](../../../../internal/health/operational.go#L141) and [report/operational.go:135](../../../../internal/report/operational.go#L135)
- **Detail**: Service Health normalization drops region facts explicitly attributed only to other subscriptions and records the scope conflict. It still retains the parent service label, even when every region in that impact group was rejected for scope. A synthetic case produced an unqualified “Service:” line for that group in the selected subscription's report. The region and foreign IDs were correctly omitted. The parent line still risks implying impact for this subscription.
- **Fix**: Omit impact groups with only explicitly foreign regions and retain the sanitized scope conflict. Keep service facts with selected, mixed or unknown scope subject to the existing attribution rules.
- **Decision**: FIXED — the user approved omitting impact groups when every listed region conflicts with the selected subscription. Normalization keeps the sanitized scope conflict and the event evidence. Groups with selected or uncertain regions and service-only groups still appear with the appropriate attribution.

### F4 — Operator acceptance has limited scenario detail

- **Severity**: 🔵 OBSERVATION
- **Impact**: 🏃 LOW — quick decision. The resolution is narrowly scoped.
- **Dimension**: Success Criteria
- **Location**: [verification.md:178](../verification.md#L178) and [plan.md:295](../plan.md#L295)
- **Detail**: The operator explicitly confirmed all tests and accepted S-02. The verification record preserves that confirmation and says the interruption stage and availability of individual live variants were unspecified. That leaves no way to establish which interruption stage was exercised or which scenarios were absent, as called for by the detailed procedure.
- **Fix**: Add sanitized particulars to the acceptance record if the operator has them. Preserve the accepted verdict and the stated limits of the evidence.
- **Decision**: ACCEPTED — the operator chose to keep the documented limit on live scenario detail. The interruption stage and availability of individual scenarios remain unspecified in [the verification record](../verification.md#L178).

## Phase coverage

| Phase | Result |
| --- | --- |
| 1. Evidence and coverage contracts | Schema 2, explicit source dispatch, source-local failure handling, exact classification and evidence references match the plan. F2 affects timing metadata. F3 affects attribution. The Phase 1 enum filtering fix remains in place. |
| 2. Source adapters and read boundary | Both approved operations use their intended Azure SDK pipelines. The GET-only scope and continuation policies, record isolation, page checkpoint ordering, and synthetic wire fixtures match the plan. |
| 3. CLI and report integration | Four-source ordering, committed publication, incomplete coverage, literal output, and schema-1 compatibility match the plan. F2 and F3 affect wording of operational evidence. Both synthetic Markdown examples link to their matching JSON snapshots, including the Phase 3 fixture fix. |
| 4. Integrated verification and acceptance | The native synthetic command matrix and process signal cases cover scale, independent failures, retained evidence and interruption. Both local and CI-selected toolchain audit results are recorded here. Operator acceptance is explicit, with the scenario detail limitation in F4. |

The review covered tracked differences and untracked implementation files against
the S-01 archive commit. Date-based history contains the S-01 implementation and
archive commits. No S-02 commits exist. Both independent reviews contributed to
the assessment. At review entry, module files and legacy report fixtures matched the baseline.

## Verification

Fresh checks ran on 2026-11-18 using writable temporary Go and Staticcheck caches.
All application tests used synthetic accounts and ARM transports.

| Check | Result |
| --- | --- |
| `go test ./internal/health` | PASS (0.522s) |
| `go test ./internal/cli ./cmd/devops-ai-agents` | PASS (8.515s, 0.424s) |
| `go test ./internal/azure` | PASS (0.680s) |
| `go test -race -count=1 ./internal/azure` | PASS (1.581s) |
| `go test ./internal/cli ./internal/report ./internal/runstore` | PASS (8.515s, 0.487s, 0.496s) |
| `go test -race -count=1 ./internal/cli ./internal/report ./internal/runstore` | PASS (38.828s, 5.960s, 2.133s) |
| `go test -race -count=1 ./internal/cli ./cmd/devops-ai-agents` | PASS (38.825s, 1.798s) |
| `go build ./...` | PASS |
| `go test -race -cover -count=1 ./...` | PASS across six packages. Command 57.1%, Azure 91.7%, CLI 95.9%, health 97.4%, report 96.5%, runstore 82.3%. |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `govulncheck ./...` with Go 1.27.4 | PASS. Zero findings in called code or imported packages. Three required-module advisories outside called code. |
| `GOTOOLCHAIN=go1.27.3 govulncheck ./...` | FAIL (exit 3). GO-2026-6859, fixed in Go 1.27.4. |
| `./scripts/check-format.sh` | PASS |
| `shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh` | PASS |
| `actionlint` | PASS |
| `git diff --check` | PASS |
| `git check-ignore` for synthetic run paths and `git ls-files -- runs/` | PASS. Both synthetic paths match `/runs/` and there are no tracked run files. |
| Temporary focused tests through a Go overlay | Confirmed F2 and F3. No probe files are part of the implementation. |

The [operator acceptance record](../verification.md) is the source for manual
confirmation. The review itself had no live Azure access. The workflow status
`impl_reviewed` records this review, and the accepted milestone still has 17/17
completed plan items.

## F1 triage verification — 2026-11-18

Go 1.27.4 (darwin/arm64) passed build, the full uncached race/coverage suite
across six packages, vet, Staticcheck, govulncheck, formatting, ShellCheck,
actionlint, and `git diff --check`. The audit reported zero vulnerabilities in
called code or imported packages and three advisories in required modules
outside called code. The go.mod directive is the only module change. Dependency
versions, go.sum and the workflow file are unchanged. GitHub CI was not run.

## F2 triage verification — 2026-11-18

The focused synthetic case reproduced the stale timestamp on both sources before
the fix. It now passes through the real coordinator and Markdown renderer. The
configured query survives cancellation, exit 130 is preserved, and no collector
starts. The schema-2 Markdown example reflects the wording and continues to match
its saved JSON.

Health, report and runstore tests passed. Build, the full uncached race/coverage
suite across six packages, vet, Staticcheck, govulncheck, formatting, ShellCheck,
actionlint and `git diff --check` passed on Go 1.27.4. The vulnerability audit
reported zero findings in called code or imported packages and three advisories
in required modules outside called code. Checks used synthetic Azure data.

## F3 triage verification — 2026-11-18

The attribution unit cases and the native CLI integration case failed before
the fix and pass after it. The integration case sends a foreign-only group
alongside a group with selected and foreign regions. It verifies that the
foreign label and region stay out of the saved JSON, Markdown and diagnostics,
while the selected facts and scope gap remain. The unit cases also cover
unknown region scope and services reported without regions. The schema-2
Markdown examples and sibling links passed their regression tests.

Build, full uncached race/coverage tests across six packages, vet, Staticcheck,
govulncheck, formatting, ShellCheck, actionlint and `git diff --check` passed on
Go 1.27.4. The audit found zero vulnerabilities in called code or imported
packages and three advisories in required modules outside called code.

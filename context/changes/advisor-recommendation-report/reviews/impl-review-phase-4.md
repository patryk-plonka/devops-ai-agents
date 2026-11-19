<!-- IMPL-REVIEW-REPORT -->
# Implementation Review · Azure Advisor recommendation report

- **Plan** [Advisor recommendation report](../plan.md)
- **Scope** Phase 4 of 4
- **Date** 2026-11-18
- **Verdict** APPROVED
- **Findings** 0 critical, 2 warnings, 2 observations

## Verdicts

| Dimension | Verdict |
| --- | --- |
| Plan Adherence | WARNING |
| Scope Discipline | PASS |
| Safety & Quality | WARNING |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS |

## Findings

### F1 — The wire-order check misses interleaved and empty pages

- **Severity** ⚠️ WARNING
- **Impact** 🏃 LOW · Quick decision. The fix is obvious and narrowly scoped.
- **Dimension** Plan Adherence
- **Location** [Populated paging](../../../../internal/cli/operational_integration_test.go#L118) and [empty run](../../../../internal/cli/operational_integration_test.go#L322)
- **Detail** The populated test records only the first request per collector. Interleaved later pages would leave the recorded sequence unchanged. The empty test checks request counts and completion without checking order. Phase 4 calls for all five native collectors to run sequentially on both populated and empty lists.
- **Fix** Record every collector request and compare it with the expected sequence of contiguous pages in both scenarios.
- **Decision**: FIXED. The populated and empty cases compare the full request order. Both targeted race tests passed.

### F2 — The signal subprocess inherits access to the real Azure CLI

- **Severity** ⚠️ WARNING
- **Impact** 🏃 LOW · Quick decision. The fix is obvious and narrowly scoped.
- **Dimension** Safety & Quality
- **Location** [Child injection](../../../../cmd/devops-ai-agents/main_test.go#L132) and [inherited environment](../../../../cmd/devops-ai-agents/main_test.go#L175)
- **Detail** The child currently injects synthetic credentials and transports for all five collectors. If a collector loses its injection, the CLI falls back to the production adapter. The child inherits the operator’s PATH, Azure CLI configuration, and default HTTP transport, so that regression could attempt to use the real session or network before the test fails.
- **Fix** Restrict the child’s PATH and Azure CLI configuration to its scratch directory and install a rejecting default HTTP transport alongside its injected transport.
- **Decision**: FIXED. The child uses a scratch-only PATH and Azure CLI configuration and a default transport that rejects network connections. The process race tests passed.

### F3 — Artifact checks accept a longer revision with the same prefix

- **Severity** ℹ️ OBSERVATION
- **Impact** 🏃 LOW · Quick decision. The fix is obvious and narrowly scoped.
- **Dimension** Success Criteria
- **Location** [Shared publication assertion](../../../../internal/cli/report_test.go#L32) and [cancelled publication assertion](../../../../cmd/devops-ai-agents/main_test.go#L318)
- **Detail** The checks search for `Revision: <number>` as a substring, so a report showing revision 170 can satisfy a snapshot at revision 17. A temporary Go overlay appended a zero to every rendered revision. Both the five-source integration tests and the process signal tests still passed.
- **Fix** Compare the complete revision header line with the saved revision in both assertions.
- **Decision**: FIXED. Both tests compare the full revision line and reject the temporary renderer that appends a zero.

### F4 — Forced-interrupt check accepts unrelated process exits

- **Severity** ℹ️ OBSERVATION
- **Impact** 🏃 LOW · Quick decision. The fix is obvious and narrowly scoped.
- **Dimension** Success Criteria
- **Location** [Second-interrupt assertions](../../../../cmd/devops-ai-agents/main_test.go#L283)
- **Detail** Once the child reaches blocked finalization, the test accepts any process exit error if the checkpoint and stdout are as expected. A crash or watchdog kill at that point could meet those checks without the second interrupt ending the child.
- **Fix** Require the process wait status to show termination by SIGINT in the second-interrupt case.
- **Decision**: FIXED. The test requires SIGINT and rejects a temporary test variant that kills the child with SIGKILL at the same marker.

## Verification

This review covers Phase 4’s integration and signal tests, helpers, and acceptance record. The change has no phase commits, so the working tree and prior phase review records supplied the scope. The test additions use the existing native collector and synthetic transport boundaries.

| Command | Result |
| --- | --- |
| `go test -race -count=1 -run 'TestFiveSourceNative\|TestProcessInterruptPreservesCheckpoint\|TestGeneratedRunFilesAreIgnored' ./internal/cli ./cmd/devops-ai-agents` | PASS · CLI and process packages |
| `go build ./...` | PASS |
| `go test -race -cover -count=1 ./...` | PASS · six packages, CLI 95.9%, process 57.1% |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS · zero reachable and imported-package vulnerabilities, with three advisories in required modules outside called code |
| `./scripts/check-format.sh` | PASS |
| `shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh` | PASS |
| `actionlint` | PASS |
| `git diff --check` | PASS |

Both synthetic run paths matched the root `/runs/` ignore rule. Git lists no tracked run files. The artifact tests also verify that the synthetic example remains eligible for version control.

Progress rows 4.1 and 4.2 record automated verification. Rows 4.3 and 4.4 remain pending for the live Advisor comparison and the operator’s explicit decision. The [acceptance record](../verification.md) keeps the live outcome pending and documents the synthetic scenario limits.

## Post-triage verification

All four findings were fixed. The full repository checks passed after the edits, including the uncached race and coverage run for all six Go packages. All Advisor interruption cases passed in five consecutive race-test runs. The wrong-revision and SIGKILL probes failed at the intended assertions. The public Go vulnerability audit found zero reachable and imported-package vulnerabilities and three advisories outside called code. The verdict remains APPROVED.

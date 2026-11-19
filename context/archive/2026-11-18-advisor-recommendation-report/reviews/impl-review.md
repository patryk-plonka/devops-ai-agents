<!-- IMPL-REVIEW-REPORT -->
# Implementation Review · Azure Advisor recommendation report

- **Plan** [S-03 Advisor recommendation report](../plan.md)
- **Scope** Full plan, phases 1–4 of 4
- **Date** 2026-11-19
- **Verdict** APPROVED
- **Findings** 0 critical, 2 warnings, 0 observations

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

### F1 — Duplicate Advisor record fields can produce a New finding

- **Severity** ⚠️ WARNING
- **Impact** 🏃 LOW · Quick decision. The fix is obvious and narrowly scoped.
- **Dimension** Safety & Quality
- **Location** [Advisor record filter](../../../../internal/azure/record_decode.go#L87) and [wire projection](../../../../internal/azure/advisor.go#L23)
- **Detail** The decoder rejects duplicates in the page envelope but uses last-value and merge behavior inside records. A synthetic record that reported `Dismissed` and `New` in the same workflow field produced a New finding. A second case repeated the instance ID and kept the later value. With one valid sibling per case, the collector returned exit 0, reported zero rejections, and saved both findings. The plan calls for rejecting malformed typed records while keeping valid siblings.
- **Fix** Use strict decoding for the Advisor record projection so repeated members reject the affected record and valid siblings still checkpoint. The pinned Go decoder passed the Advisor race tests with this change in a temporary overlay.
- **Decision**: FIXED. Advisor’s strict record decoder rejects duplicate members before the pager receives the page. The regression covers repeated status, instance ID, properties, and assessed-resource fields. Each case checkpoints its valid sibling and returns exit 3 with one rejected record. The Azure and health race suites passed. [Regression](../../../../internal/azure/advisor_test.go#L181)

### F2 — Blank ARM path components pass Advisor validation

- **Severity** ⚠️ WARNING
- **Impact** 🏃 LOW · Quick decision. The fix is obvious and narrowly scoped.
- **Dimension** Plan Adherence
- **Location** [Parent and target path validation](../../../../internal/health/advisor.go#L55)
- **Detail** A whitespace-only provider namespace, resource type, or resource name passes the path shape check. Synthetic tests for all three cases each saved a malformed resource-scoped instance as New and accepted the same malformed path as the assessed resource on a valid instance. The plan requires a validated parent and unknown attribution when the supplied target is invalid.
- **Fix** Reject whitespace-only ARM components during Advisor validation. Preserve the spelling of valid paths and keep recommendation names opaque. This change passed the Advisor health race tests in a temporary overlay.
- **Decision**: FIXED. Advisor path validation rejects whitespace-only parent and target components. Regressions cover blank provider namespaces, resource types, and names, including Unicode whitespace. They verify that an invalid parent is rejected, an invalid target produces unknown attribution, and a name with an internal space retains its source spelling. The targeted health race test passed. [Regression](../../../../internal/health/advisor_test.go#L109)

## Review scope

The S-03 work is uncommitted. The review used the tracked diff from `14bc0f1` and the untracked S-03 source, fixtures, and records. Two reviewers independently checked plan adherence and safety. The four phase reviews record their earlier fixes. The current adapter uses the approved cached subscription list operation and shared read boundary. [Microsoft Advisor List](https://learn.microsoft.com/en-us/rest/api/advisor/recommendations/list?view=rest-advisor-2026-10-01)

## Verification

Checks ran on macOS arm64 with Go 1.27.4. Application tests use synthetic credentials and ARM responses.

| Command | Result |
| --- | --- |
| `go test -race -count=1 ./internal/health` | PASS · health |
| `go test -race -count=1 ./internal/azure ./internal/health` | PASS · Azure and health |
| `go test -race -count=1 ./internal/cli ./internal/report ./internal/runstore ./cmd/devops-ai-agents` | PASS · CLI, report, runstore, and command |
| `go test -race -count=1 -run 'TestFiveSourceNative\|TestProcessInterruptPreservesCheckpoint\|TestGeneratedRunFilesAreIgnored' ./internal/cli ./cmd/devops-ai-agents` | PASS · five-collector, signal, and ignore cases |
| `go build ./...` | PASS |
| `go test -race -cover -count=1 ./...` | PASS · all six packages, 57.1% command, 92.2% Azure, 95.9% CLI, 98.0% health, 96.2% report, 82.3% runstore |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS · zero reachable or imported-package vulnerabilities, with three advisories in required modules outside called code |
| `./scripts/check-format.sh` | PASS |
| `shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh` | PASS |
| `actionlint` | PASS |
| `git diff --check` | PASS |

Temporary synthetic regression probes reproduced both warnings. The duplicate-member cases finished with two findings and zero rejections. The blank-component cases accepted both records and retained the invalid target. The probes ran outside the working tree.

All 11 plan checks are complete. The operator confirmed the portal comparison and accepted S-03 on 2026-11-19. Individual portal field results were unspecified. Local review independently confirmed matching run ID and revision 18, all eight Advisor rows linked to matching saved evidence, retained earlier-source evidence, and incomplete coverage due to the Service Health partial result. Both artifacts matched the Git ignore rule and Git tracks no run files. [Acceptance record](../verification.md#live-acceptance--2026-11-19)

## Post-triage verification

Both findings were fixed. The duplicate-member regression reproduced all four cases before the decoder fix. The ARM regression reproduced six blank-path failures before the validator fix. Both regressions passed after the changes. The build, race and coverage tests across all six packages, vet, Staticcheck, format check, ShellCheck, and actionlint passed. Govulncheck reported zero reachable or imported-package vulnerabilities and three advisories outside called code. `git diff --check` passed. The verdict remains APPROVED.

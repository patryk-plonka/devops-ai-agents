<!-- IMPL-REVIEW-REPORT -->
# Implementation Review: Azure Advisor recommendation report

- **Plan**: [Implementation plan](../plan.md)
- **Scope**: Phase 1 of 4, Advisor evidence
- **Date**: 2026-11-18
- **Verdict**: APPROVED
- **Findings**: 0 critical, 3 warnings, 0 observations. All three fixed.
- **Progress**: 2/2 automated items checked for Phase 1 and 2/11 items checked overall. Phase 2 is next in the plan.

## Verdicts

| Dimension | Verdict |
| --- | --- |
| Plan Adherence | PASS |
| Scope Discipline | PASS |
| Safety & Quality | PASS |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS |

## Findings

### F1 — Advisor ARM validation accepts malformed paths and loses valid group targets

- **Severity**: ⚠️ WARNING
- **Impact**: 🔎 MEDIUM — real tradeoff, pause to reason through it
- **Dimension**: Plan Adherence and Safety & Quality
- **Location**: [Advisor path validation](../../../../internal/health/advisor.go#L55), [target acceptance](../../../../internal/health/advisor.go#L134), [shared scope and privacy checks](../../../../internal/health/normalize.go#L49)
- **Detail**: Under `/subscriptions/<selected>/resourceGroups/Group`, the parent path `/providers/Microsoft.Compute/providers/Microsoft.Storage/storageAccounts/demo` omits the first provider's type and name. Adding the Advisor suffix produced one New finding with zero rejected records. Supplying the same malformed path only in `resourceMetadata.resourceId` persisted it as the assessed target. An instance under `/resourceGroups/not a group` also passed. An explicitly reported valid target `/subscriptions/<selected>/resourceGroups/Group` was discarded with an unknown-attribution limitation. Phase 1 requires validated instance parents and valid selected-subscription assessed resource IDs. [Azure resource group naming rules](https://learn.microsoft.com/en-us/azure/azure-resource-manager/management/resource-name-rules#microsoftresources) · [Azure resource group ID format](https://learn.microsoft.com/en-us/azure/azure-resource-manager/templates/template-functions-scope#resourcegroup)
- **Fix**: Validate Advisor parent and target paths with the same scoped rules. Check resource-group names, require a resource type and name after each provider segment, and accept an explicitly supplied resource-group target. Keep the recommendation name opaque.
  - **Strength**: Rejects malformed identities and keeps valid metadata-backed targets.
  - **Tradeoff**: Advisor validation must distinguish subscription scope, resource groups, and provider resources.
  - **Confidence**: High. Synthetic inputs reproduced all four outcomes, and the Azure documentation confirms the resource-group shape.
  - **Blind spot**: The review used synthetic responses.
- **Decision**: FIXED — the operator selected “Fix now.” The new Advisor path check requires a resource type and name after each provider and validates resource-group names. Explicit, valid resource-group targets are retained. The [12-case regression](../../../../internal/health/advisor_test.go#L109) reproduced the validation failures before the fix and passed after it, including tests for opaque names and foreign group targets.

### F2 — An out-of-range UTC timestamp can block the page checkpoint

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW — quick decision, fix is obvious and narrowly scoped
- **Dimension**: Safety & Quality
- **Location**: [Advisor source timestamp normalization](../../../../internal/health/advisor.go#L150), [evidence hashing](../../../../internal/health/normalize.go#L100)
- **Detail**: Go parses `9999-12-31T23:59:59-01:00` as an RFC 3339 timestamp, but converting it to UTC produces year 10000. Two otherwise distinct recommendations carrying that timestamp collapsed into one evidence record because the hash helper discarded the JSON error. Serializing the run then failed with `year outside of range [0,9999]`. The coordinator treats save failures as fatal, so a mixed page containing otherwise valid records can fail to checkpoint.
- **Fix**: Verify that both Advisor source times can be represented in the UTC snapshot before adding evidence. Reject an offending record and test that valid siblings still checkpoint.
- **Decision**: FIXED — the operator selected “Fix now.” Both source times are normalized and validated before evidence is added. The [checkpoint regression](../../../../internal/health/advisor_test.go#L418) reproduced the fatal save failure before the fix. It now verifies rejection of four out-of-range inputs, preservation of valid timestamps at both year boundaries, and a saved page with a partial enumeration result.

### F3 — Case-equivalent Advisor IDs can yield different rows or duplicate findings

- **Severity**: ⚠️ WARNING
- **Impact**: 🔎 MEDIUM — real tradeoff, pause to reason through it
- **Dimension**: Plan Adherence
- **Location**: [Advisor case folding](../../../../internal/health/advisor.go#L173), [instance key and derived row](../../../../internal/health/advisor.go#L186), [canonical evidence fields](../../../../internal/health/normalize.go#L105)
- **Detail**: Two observations with the same facts and IDs that differ only in ASCII case collapse to one evidence ID and one finding ID. Reversing those observations with fixed times changes the derived row and finding because their strings come from the first retained evidence. The current reversal regression varies the evaluation timestamp, which prevents deduplication and misses this path. For Unicode group names, `ΟΣ` and `ος` compare equal under `EqualFold`, but their lowercase keys differ. The probe produced two evidence records, review rows, and findings for the same case-insensitive instance.
- **Fix**: Use one case-folding rule for Advisor comparisons, evidence identity, grouping and finding IDs. Use a deterministic ID spelling in the derived view, while evidence retains its first observation time and source spelling.
  - **Strength**: Deduplication, attribution and review rows agree on instance identity.
  - **Tradeoff**: Derived IDs can use canonical spelling. The evidence index can display the original source spelling.
  - **Confidence**: High. Both page orders and the Unicode case pair reproduced the discrepancies.
  - **Blind spot**: The probes cover ASCII and a documented Unicode resource-group name shape.
- **Decision**: FIXED — the operator selected “Fix now.” Advisor uses Unicode simple-fold classes consistently for attribution, evidence hashes, grouping, and finding IDs. Derived views use a fixed spelling. Evidence keeps the first source spelling and collection time. The [case regressions](../../../../internal/health/advisor_test.go#L197) reproduced the order-dependent row, duplicate findings, and an incorrect merge of distinct IDs before the fix. They now verify both page orders, a later evaluation that keeps the same finding ID, and distinct IDs that lowercase to the same string.

## Scope and verification

The review compared the working tree against HEAD and included the two untracked Advisor Go files. The change folder contained the approved plan and supporting documents. Two independent reviewers checked plan adherence and implementation quality. Phase 1 adds typed evidence, exact enum classification, query and page timing, and derived findings on the existing checkpoint path. Source state-change ordering, unresolved conflict gaps, and the existing schema-1 and schema-2 report fixtures passed their checks. The Azure adapter and production report wiring are scheduled for Phases 2 and 3.

| Check | Result |
| --- | --- |
| `go test -race -count=1 ./internal/health` | PASS, 2.492s |
| `go test -race -cover -count=1 ./...` | PASS, all six packages |
| `go build ./...`, `go vet ./...`, `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS, zero reachable vulnerabilities and three required-module advisories outside called code |
| Format script, ShellCheck, actionlint, and `git diff --check` | PASS |
| Temporary Go test overlay | Reproduced F1–F3 with synthetic inputs |

Live operator checks are scheduled for Phase 4.

## F1 verification after fix

The targeted ARM path test passed, and all six packages passed the uncached race and coverage suite. Build, vet, Staticcheck, formatting, ShellCheck, actionlint, and `git diff --check` passed. The vulnerability audit found zero reachable vulnerabilities and three advisories in required modules outside called code.

## F2 verification after fix

The targeted checkpoint regression, health race suite, and full race and coverage suite passed. Build, vet, Staticcheck, formatting, ShellCheck, actionlint, and `git diff --check` passed. The vulnerability audit found zero reachable vulnerabilities and three advisories in required modules outside called code.

## F3 verification and triage outcome

The targeted health race suite and all six packages in the uncached race and coverage suite passed. Build, vet, Staticcheck, formatting, ShellCheck, actionlint, and `git diff --check` passed. The vulnerability audit found zero reachable vulnerabilities and three advisories in required modules outside called code.

F1, F2, and F3 are fixed and verified. Phase 1 is approved. The change remains marked `impl_reviewed` and the work is uncommitted.

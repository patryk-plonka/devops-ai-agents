<!-- IMPL-REVIEW-REPORT -->
# Implementation Review · Azure Advisor recommendation report

- **Plan**: [Implementation plan](../plan.md)
- **Scope**: Phase 2 of 4, read-only collection
- **Date**: 2026-11-18
- **Verdict**: APPROVED
- **Findings**: 0 critical, 1 warning, 1 observation

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

### F1 — Duplicate envelope fields can turn a malformed page into a completed empty list

- **Severity** ⚠️ WARNING
- **Impact** 🏃 LOW — quick decision, fix is obvious and narrowly scoped
- **Dimension** Plan Adherence
- **Location** [Record envelope validation](../../../../internal/azure/record_decode.go#L51)
- **Detail** Decoding the envelope into a map collapses duplicate keys before validation. A synthetic Advisor response with `"value":null` followed by `"value":[]` reached the checkpoint as a completed empty list with exit code 0. A response with a wrongly typed `nextLink` followed by a null `nextLink` had the same outcome. The plan calls for malformed list and continuation fields to fail the envelope.
- **Fix** Reject duplicate `value` and `nextLink` fields while parsing the Advisor envelope. Preserve valid siblings when an individual record is malformed.
- **Decision**: FIXED — the operator selected “Fix now.” The Advisor decoder rejects repeat occurrences of either documented envelope key before accepting a page. The [regression](../../../../internal/azure/advisor_test.go#L157) reproduced successful empty enumeration for four cases, including a duplicate with an escaped key. All four now fail with `invalid_response`, one request, and zero accepted pages. The Azure and health race suites passed, including mixed pages and valid empty responses.

### F2 — A successful decode can be accepted after its page deadline

- **Severity** 🔎 OBSERVATION
- **Impact** 🏃 LOW — quick decision, fix is obvious and narrowly scoped
- **Dimension** Safety & Quality
- **Location** [Shared page loop](../../../../internal/azure/paging.go#L167), [Advisor decode](../../../../internal/azure/advisor.go#L65)
- **Detail** The inherited page loop consults the bounded context after `NextPage` only when the pager returns an error. The Advisor fetcher decodes JSON after the SDK pipeline returns. A synthetic runtime pager that returned a decoded result after the deadline triggered the accept callback and reported success. In a separate probe, the full SDK pipeline correctly rejected an HTTP response that arrived after the deadline.
- **Fix** Check the bounded context after `NextPage` before accepting the page, even when decoding succeeded. Keep previously committed pages.
- **Decision**: FIXED — the operator selected “Fix now.” The shared pager checks the bounded context after every decode and before emitting its result. The [checkpoint regression](../../../../internal/azure/advisor_test.go#L181) reproduced an incorrect completed result with two saved pages before the fix. It now verifies a partial result with reason `timeout`, the original page timestamps and evidence, and zero evidence from the expired page.

## Scope and verification

The review compared the Phase 2 working tree against HEAD `14bc0f1` and included untracked Azure code and fixtures. Phase 1 health code provided the normalization and checkpoint contracts. Both Phase 2 progress entries are checked. Manual acceptance is scheduled for Phase 4.

The adapter uses the subscription-bound credential, the existing ARM client and native pager, and the shared read transport. Tests verify the initial bodyless public ARM GET, both token aliases, documented page size, equivalent continuation detection, and rejection of out-of-scope URLs and redirects before another request. Literal fixtures verify typed mapping and successful empty pages. Mixed-page tests verify that checkpoints survive later denial, throttling, network timeout, a malformed next page, and the page ceiling. [Advisor fixtures and provenance](../../../../internal/azure/testdata/README.md#azure-advisor)

| Check | Result |
| --- | --- |
| `go test -race -count=1 ./internal/azure ./internal/health` | PASS, Azure 3.150s and health 2.787s |
| `go test -race -cover -count=1 ./...` | PASS, all six packages, 92.3% Azure and 98.0% health coverage |
| `go build ./...`, `go vet ./...`, `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS, zero reachable vulnerabilities, three advisories in required modules outside called code |
| Format script, ShellCheck, actionlint, and `git diff --check` | PASS |
| Temporary synthetic Go test overlays | Reproduced F1 and the post-decode behavior in F2 |

Verification used synthetic transports and data. No Azure session was accessed.

## F1 verification after the fix

The duplicate-envelope regression and the Azure and health race tests passed. All six packages passed the uncached race and coverage suite. Build, vet, Staticcheck, formatting, ShellCheck, actionlint, and `git diff --check` passed. The Go vulnerability audit found zero reachable vulnerabilities and three advisories in required modules outside called code.

## F2 verification and triage outcome

The late-decode regression and the Azure and health race tests passed. All six packages passed the uncached race and coverage suite. Build, vet, Staticcheck, formatting, ShellCheck, actionlint, and `git diff --check` passed. The Go vulnerability audit found zero reachable vulnerabilities and three advisories in required modules outside called code.

F1 and F2 are fixed. Phase 2 remains approved and the changes are uncommitted.

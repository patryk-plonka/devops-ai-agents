<!-- IMPL-REVIEW-REPORT -->
# Implementation Review: Subscription Resource Health Report

- **Plan**: [plan.md](../plan.md)
- **Scope**: Phase 1 of 4 — Subscription access and run setup
- **Date**: 2026-11-05
- **Verdict**: APPROVED after F1 resolution on 2026-11-06 (original verdict: NEEDS ATTENTION)
- **Findings**: 0 critical, 1 warning, 0 observations

## Verdicts

| Dimension | Verdict |
|-----------|---------|
| Plan Adherence | PASS — F1 diagnostic privacy gap fixed and verified |
| Scope Discipline | PASS |
| Safety & Quality | PASS — SDK diagnostics are opt-in and filtered |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS — all prescribed automated commands exit 0 |

The original review identified one non-critical privacy contract failure. The
finding and reproduction below retain that history; the resolution section
records the operator's preferred fix and the subsequent passing checks.

## Findings

### F1 — SDK logging bypasses diagnostic sanitization

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW — quick decision; fix is obvious and narrowly scoped
- **Dimension**: Safety & Quality
- **Location**: cmd/devops-ai-agents/main.go:14; internal/azure/subscription.go:61
- **Detail**: The plan requires filtering before diagnostic output and excludes raw error bodies. When the process inherits `AZURE_SDK_GO_LOGGING=all`, azcore enables a stderr logger during package initialization. SDK response-error construction logs the entire Azure error body before `Validator.Validate` converts the returned error to a fixed safe message. Startup does not disable that logger. The existing denied-access test passes while the synthetic `secret-canary` from its response body appears in process output. Normal execution without this explicitly enabled logging setting does not produce this leakage. Request headers and SDK diagnostics can also bypass the application's chosen diagnostic output path.
- **Fix**: Disable the Azure SDK logger with `azcore/log.SetListener(nil)` at process startup before credential/client operations, and add a subprocess regression launched with `AZURE_SDK_GO_LOGGING=all` that asserts actual stderr excludes response-body canaries. Merely unsetting the environment variable after SDK package initialization does not remove the installed logger.
- **Decision**: FIXED on 2026-11-06 via the operator's alternative: retain filtered SDK diagnostics behind `--debug`, with SDK logging disabled by default even when enabled by the environment. See the resolution below and [review-fixes.md](../follow-ups/review-fixes.md).

### Reproduction evidence for F1

The existing synthetic denied-access test was executed in a fresh process with
SDK logging enabled. It uses fake credentials and a fake HTTP transport, so no
Azure login, live request, or real subscription data was used:

```sh
AZURE_SDK_GO_LOGGING=all \
GOCACHE=/tmp/devops-ai-agents-checks/gocache \
GOMODCACHE=/tmp/devops-ai-agents-checks/modcache \
go test -count=1 -run TestLiveValidationAndReadBoundary/denied -v ./internal/azure
```

Exit code: 0. In addition to the passing test, process output included:

```text
ResponseError: Request information not available
RESPONSE 403:
ERROR CODE: AuthorizationFailed
{
  "error": {
    "code": "AuthorizationFailed",
    "message": "secret-canary"
  }
}
```

This is synthetic evidence, not retained customer output. Inspection of the
resolved SDK confirms the behavior in `sdk/internal@v1.13.0/log/log.go` (environment
initialization and default stderr listener) and
`sdk/azcore@v1.23.3/internal/exported/response_error.go` (logging of
`ResponseError.Error()`).

## Scope and plan comparison

No Git commits exist and the implementation remains uncommitted at the operator's
request. A historical commit range cannot be constructed. Review scope was
resolved from the completed phase 1 Progress rows and the explicit 25-path touched
set recorded during implementation. The 11 pre-existing unrelated paths were
excluded. Two independent reviewers examined plan alignment and safety/patterns.

| Planned area | Result |
| --- | --- |
| CLI entry and scope interaction | Matches: injectable streams/adapters, help without run creation, exact scope selection, stable terminal ambiguity prompt, no default on EOF/blank, preliminary partial outcome |
| Discovery and credentials | Matches: argument-based cached discovery, AzureCloud user-account validation, explicit subscription binding with TenantID unset, synthetic guest/default-subscription coverage |
| Live subscription validation and transport | Matches: current access is checked, response scope is verified, mutations and redirects are rejected, provider registration is disabled; diagnostic logging exception is F1 |
| Local run initialization | Matches: module-root lookup, storage before Azure access, unique private run directories, rooted filesystem operations, atomic versioned snapshots, failed scope records, no placeholder report |
| Automated checks and documentation | Matches: read-only CI, pinned actions/tools, required checks, untracked-source formatting checks, honest phase limitations and security dependency update |

Introducing `internal/health/privacy.go` and its test before phase 2 is justified
by phase 1's requirement to filter the requested scope and subscription names.
The compatible vulnerability-driven `golang.org/x/net` update and its dependency
requirements were expressly permitted by the plan. Neither is scope creep.
There was no pre-existing application convention against which to claim a
historical pattern violation; new package boundaries and sibling patterns are
consistent.

## Automated verification

Every prescribed phase 1 command was run during this review. Go commands used
writable temporary `GOCACHE` and `GOMODCACHE` directories; staticcheck also used
temporary `STATICCHECK_CACHE`. The vulnerability scan used the public database.

| Command | Result | Observed output |
| --- | --- | --- |
| `go test ./internal/cli ./internal/azure ./internal/runstore` | PASS, exit 0 | All three packages `ok` |
| `go build ./...` | PASS, exit 0 | No diagnostics |
| `go test -race -cover ./...` | PASS, exit 0 | Azure 88.1%, CLI 91.9%, health 64.3%, runstore 81.1%; executable entry package 0.0%; no race failures |
| `go vet ./...` | PASS, exit 0 | No diagnostics |
| `staticcheck ./...` | PASS, exit 0 | No diagnostics |
| `govulncheck ./...` | PASS, exit 0 | Zero reachable vulnerabilities; three advisories in required modules outside called code |
| `./scripts/check-format.sh` | PASS, exit 0 | No formatting differences |
| `shellcheck scripts/check-format.sh` | PASS, exit 0 | No diagnostics |
| `actionlint` | PASS, exit 0 | No diagnostics |

The test-command passes do not prove the diagnostic privacy invariant: the
existing tests inspect returned sanitized errors rather than actual SDK stderr,
which is why F1 can coexist with passing automated criteria.

## Manual verification and remaining phases

Phase 1 has no manual acceptance items. All four phase 4 manual rows (4.4–4.7)
remain unchecked, correctly reflecting that live subscription access, Resource
Health comparison, report usability, and live interruption have not been
accepted. Overall Progress remains 3 of 18 rows checked; the next pending work is
phase 2. Missing collectors, full tag normalization, Markdown reporting, and
complete signal integration belong to later phases and are not findings against
phase 1. CI has been checked locally, not run on GitHub in this review.

During the initial review, only this report and the skill-required `change.md`
review stamp were written. Application code and plan Progress were not changed
at that point; no files were staged or committed.

## F1 resolution — 2026-11-06

The operator requested a debug flag to retain useful development diagnostics.
`health check --subscription "<id-or-name>" --debug` now enables a filtered SDK
listener on stderr. Without that flag, the CLI removes the SDK's environment
listener before any Azure operation. It removes its own listener again on exit.

The listener emits only request attempt numbers, response timings and fixed
outcomes, numeric HTTP statuses, and retry delays. It never forwards raw SDK
messages, including retry errors, response-error bodies, headers, URLs, or token
text. Unknown event types or message formats are dropped. The SDK log API is
process-global and configured before client operations, matching the existing
one-run-at-a-time contract. README and command help document the flag.

`internal/cli/debug_test.go` launches fresh processes so the SDK reads
`AZURE_SDK_GO_LOGGING` during package initialization. The test first reproduced
the original output leak before the fix. After the fix, it passes all four cases:
default execution, inherited SDK logging, explicit debug mode, and debug mode
with inherited SDK logging. A real SDK subscription client uses synthetic
credentials and a fake denied-access transport. Actual stderr excludes body,
header, and token canaries in every case, debug summaries appear only with the
flag, stdout excludes debug diagnostics, and the access-failure exit remains 1.
Separate summary tests cover retry delays, failed requests, malformed messages,
and excluded event types. No live Azure account or data was used.

Post-fix verification passed:

- `go test -count=1 ./internal/azure ./internal/cli`
- `go build ./...`
- `go test -race -cover ./...`
- `go vet ./...`
- `staticcheck ./...`
- `govulncheck ./...` (zero reachable vulnerabilities; three advisories outside called code)
- `./scripts/check-format.sh`
- `shellcheck scripts/check-format.sh`
- `actionlint`

Triage is complete: F1 fixed, zero pending decisions. Plan Progress and phase 4
manual acceptance remain unchanged. All changes remain uncommitted as requested.

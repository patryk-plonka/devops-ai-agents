<!-- IMPL-REVIEW-REPORT -->
# Implementation Review: Subscription Resource Health Report

- **Plan**: [plan.md](../plan.md)
- **Scope**: Phase 2 of 4 — Evidence collection and coverage
- **Date**: 2026-11-06
- **Verdict**: APPROVED after F1–F3 resolution on 2026-11-06 (original verdict: NEEDS ATTENTION)
- **Findings**: 0 critical, 2 warnings, 1 observation

## Verdicts

| Dimension | Verdict |
|-----------|---------|
| Plan Adherence | PASS — malformed-record retention and URL filtering gaps fixed and verified |
| Scope Discipline | PASS |
| Safety & Quality | PASS — all three findings fixed and verified |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS — prescribed automated checks pass; phase 2 has no manual gate |

The findings and original review evidence below retain their history. The
resolution section records the operator-approved fixes and post-fix verification.

## Findings

### F1 — Non-HTTP credential URLs survive tag filtering

- **Severity**: ⚠️ WARNING
- **Impact**: 🏃 LOW — quick decision; fix is obvious and narrowly scoped
- **Dimension**: Safety & Quality
- **Location**: internal/health/privacy.go:11; internal/health/privacy.go:65
- **Detail**: The plan explicitly filters recognizable credential-bearing URLs before persistence. `credentialValue` recognizes userinfo only for HTTP(S). An ordinary `endpoint` tag containing the synthetic value `rediss://app:credential-canary@cache.example:6380` survives inventory normalization and JSON serialization unchanged, with zero redactions. Redis/PostgreSQL connection URLs do not need an HTTP scheme or a credential-like tag key to contain credentials. This expands the phase 1 text-filter limitation into phase 2's retained resource tags; it is a concrete violation of a defined filtering pattern, not a demand to certify arbitrary free text. Local run storage is private and Git-ignored, but the record still contains a recognizable credential and later rendering would consume it.
- **Fix**: Recognize userinfo-bearing URLs across URI schemes before and after control stripping, and add text/tag JSON regressions for Redis, PostgreSQL, and existing HTTP(S) cases.
- **Decision**: FIXED on 2026-11-06 — scheme-independent userinfo URL detection now filters normal text and tags before persistence; Redis/PostgreSQL and control-stripping regressions pass.

### F2 — One malformed record discards valid siblings on the page

- **Severity**: ⚠️ WARNING
- **Impact**: 🔎 MEDIUM — real tradeoff; pause to reason through it
- **Dimension**: Plan Adherence
- **Location**: internal/azure/paging.go:139; internal/azure/inventory.go:28; internal/azure/resourcehealth.go:20
- **Detail**: The data contract says to reject malformed or wrong-scope records, retain other valid records, and record the gap without saving the rejected payload. The adapters first decode the entire page through typed SDK pagers, before `NormalizePage` can reject individual entries. In a syntactically valid inventory page containing a valid VM followed by `{"id":42}`, the SDK aborts the page: the collector returns `invalid_response`, emits zero pages, and retains zero evidence. The same happens to a valid Unavailable observation followed by a record whose availability state is an object. The valid sibling never reaches the checkpoint and rejected-record counts stay zero. Previously checkpointed pages remain intact and the result is honestly incomplete, which limits the failure, but valid records on this page are unnecessarily lost. Existing tests cover invalid IDs after typed decoding and malformed whole-page JSON; neither covers this case.
- **Fix**: Add record-level decoding/filtering at the response boundary before typed page acceptance, preserving native SDK pagination and the existing authentication/read boundary; checkpoint valid siblings and count rejected records, while keeping malformed envelopes as page failures.
  - Strength: Meets the approved per-record retention contract without weakening the transport boundary or hiding the rejected-record coverage gap.
  - Tradeoff: Requires shared response handling and SDK compatibility tests; this is more involved than changing the final error category.
  - Confidence: MED — the defect is reproduced for both collectors, but the proposed SDK integration has not been prototyped during this review.
  - Blind spot: SDK payload buffering and continuation handling need explicit regression coverage when implementing the fix; live incidence has not been measured.
- **Decision**: FIXED on 2026-11-06 — collection response handling isolates schema-invalid entries before native SDK page decoding; valid siblings are checkpointed, rejected records are counted, and malformed envelopes remain page failures.

### F3 — Accepted historical statuses can determine current health

- **Severity**: ℹ️ OBSERVATION
- **Impact**: 🔎 MEDIUM — real tradeoff; pause to reason through it
- **Dimension**: Safety & Quality
- **Location**: internal/health/normalize.go:62; internal/health/normalize.go:248
- **Detail**: The parser deliberately accepts historical timestamp status names, but normalization does not retain their current-versus-historical role. A historical Available record alone produces an Available observation; adding an undated historical Available record to an explicit `/current` Unavailable record makes the current observation undetermined. This behavior is reproduced synthetically. Microsoft documents the subscription listing as current status and the per-resource listing as historical transitions/events, so a historical record is not expected from the production operation. That substantially limits practical likelihood and makes this a robustness observation rather than a confirmed normal-flow failure. Retaining historical issue evidence with source-qualified titles is not separately considered a defect.
- **Fix**: Preserve accepted history as evidence while making it ineligible to establish or conflict with current state; add historical-only and mixed `/current`/historical regression cases.
  - Strength: Retains traceability and keeps unexpected historical records from changing current-state conclusions.
  - Tradeoff: Requires an explicit current-state eligibility distinction in normalization and coverage handling.
  - Confidence: MED — behavior and API distinction are verified; the production subscription operation should not return these mixed records.
  - Blind spot: No live historical/mixed response has been observed from the subscription endpoint.
- **Decision**: FIXED on 2026-11-06 — evidence records current/historical status roles; only current evidence determines current state and findings, while historical-only data leaves an explicit missing-current gap.

Microsoft's [subscription operation](https://learn.microsoft.com/en-us/rest/api/resourcehealth/availability-statuses/list-by-subscription-id?view=rest-resourcehealth-2025-05-01) documents current statuses. Its separate [resource history operation](https://learn.microsoft.com/en-us/rest/api/resourcehealth/availability-statuses/list?view=rest-resourcehealth-2025-05-01) documents historical transitions and events and illustrates dated IDs. F3 is an inference from those documented semantics and the synthetic normalization result.

## Scope and plan comparison

The review follows the immediately preceding request to review completed phase 2.
There are no Git commits: every repository file remains untracked because the
operator requested leaving work uncommitted. No historical commit diff can be
constructed. The review therefore uses the explicit 23-path phase 2 touched set
from implementation, the plan contracts, and the current files; untracked phase 1
and pre-existing foundation files are not treated as newly added phase 2 scope.
Two independent agents reviewed plan drift and safety/pattern compliance.

Phase 2 touched paths:

- `README.md`, `go.mod`, `go.sum`, `context/foundation/development-tools.md`
- `context/changes/subscription-resource-health-report/change.md` and `plan.md`
- `internal/azure/collection_test.go`, `inventory.go`, `resourcehealth.go`, `paging.go`, `transport.go`, `transport_test.go`
- `internal/health/model.go`, `evidence.go`, `normalize.go`, `normalize_test.go`, `privacy.go`, `privacy_test.go`, `check.go`, `check_test.go`
- `internal/cli/cli.go`, `cli_test.go`, `collection_test.go`

| Planned area | Result |
| --- | --- |
| Native inventory/Resource Health clients and versions | Matches: armresources/v4 v4.0.0, armresourcehealth v1.4.0, native API versions/pagers, no source filters or raw prose expansion |
| Read/authentication boundary | Matches: selected-subscription public ARM GET routes only, same-operation continuations, redirects/mutations rejected, registration disabled |
| Bounded sequential independent collection | Matches: 60-second per-page deadline, three retries, 10,000-page ceiling, repeat detection, cancellation stops subsequent reads |
| Page normalization and checkpoints | Matches for successfully decoded pages: checkpoint before next request; storage failure stops reads; malformed record granularity exception is F2 |
| Evidence, joins, states and coverage | Matches stable references, exact normalized duplicates, conflict/timestamp handling, Unknown versus missing evidence, event/null handling, unmatched observations and three not-checked sources; historical eligibility concern is F3 |
| Privacy and tags | Ordinary tags and credential-like keys handled, fixed diagnostics and filtered SDK debug logging retained; URL-scheme gap is F1 |
| Orchestration/outcomes | Matches: state and storage ownership remain in orchestration, execution outcome independent of incomplete coverage, partial evidence retained |
| Persistence implementation | Existing `runstore.Save` is reused appropriately; it atomically replaces snapshots and advances revisions only after publication. Unchanged store source is not missing implementation |
| Interim CLI and docs | Matches: evidence snapshot identified on stderr, no Markdown report claimed or created, accurate interim exit semantics |

`paging.go` and `evidence.go` factor planned responsibilities into small files;
they do not add product scope. Inline synthetic fixtures satisfy the focused
phase 2 tests; the broader testdata/integration rollout remains phase 4. No
architecture or substantive sibling-pattern violations were identified. No
Azure writes, other-source collection, ticket drafts, arbitrary endpoint/token
flags, or concurrent runs were added.

## Automated verification

All prescribed phase 2 checks were rerun during review. Go commands used
`GOCACHE=/tmp/devops-ai-agents-checks/gocache` and
`GOMODCACHE=/tmp/devops-ai-agents-checks/modcache`; staticcheck additionally used
`STATICCHECK_CACHE=/tmp/devops-ai-agents-checks/staticcheck`. The vulnerability
check read the public database. No live Azure account or subscription was used.

| Command | Result | Actual output summary |
| --- | --- | --- |
| `go test ./internal/azure ./internal/health ./internal/runstore` | PASS, exit 0 | All three packages `ok` |
| `go test -count=1 -run 'TestTransport\|TestCollectionTransport\|TestLaterPageFailures\|TestIndependentCollection\|TestCheckpointFailure\|TestCLICollection' ./internal/azure ./internal/health ./internal/cli` | PASS, exit 0 | All three packages `ok`; transport, later-page failures, independence and storage-failure contracts exercised |
| `go build ./...` | PASS, exit 0 | No diagnostics |
| `go test -race -cover -count=1 ./...` | PASS, exit 0 | Azure 86.3%, CLI 88.2%, health 94.0%, runstore 81.1%; executable entry 0.0%; no race failures |
| `go vet ./...` | PASS, exit 0 | No diagnostics |
| `staticcheck ./...` | PASS, exit 0 | No diagnostics |
| `govulncheck ./...` | PASS, exit 0 | Zero reachable vulnerabilities; three required-module advisories outside called code |
| `./scripts/check-format.sh` | PASS, exit 0 | No formatting differences |
| `shellcheck scripts/check-format.sh` | PASS, exit 0 | No diagnostics |
| `actionlint` | PASS, exit 0 | No diagnostics |

The passing suite does not cover the new counterexamples below. Their probe
assertions confirm the observed behavior; a passing probe is not evidence that
the behavior satisfies the plan.

## Controlled reproduction evidence

Review probes used only synthetic IDs/data in an isolated `/tmp` copy or a Go
overlay. Application and test files in this repository were not changed.

F1 and F2 were independently rerun with:

```sh
cd /tmp/phase2-review-v9eiytod
GOCACHE=/tmp/devops-ai-agents-checks/gocache \
GOMODCACHE=/tmp/devops-ai-agents-checks/modcache \
go test -count=1 -run TestReview -v ./internal/azure
```

Probe source: `/tmp/phase2-review-v9eiytod/internal/azure/review_probe_test.go`.
Exit 0, with these synthetic observations:

```text
route=inventory err=Source returned invalid JSON data. evidence=0 pages=0
route=resource_health err=Source returned invalid JSON data. evidence=0 pages=0
redactions=0 retained endpoint="rediss://app:credential-canary@cache.example:6380"
```

F3 was independently rerun from the repository with:

```sh
GOCACHE=/tmp/devops-ai-agents-checks/gocache \
GOMODCACHE=/tmp/devops-ai-agents-checks/modcache \
go test -overlay=/tmp/devops-phase2-drift-review/overlay.json \
  -count=1 -run TestReviewHistoricCurrent -v ./internal/health
```

Exit 0, with these synthetic observations:

```text
explicit current Unavailable + undated historical Available: state=undetermined findings=1 conflict=true
historical Available only: state=Available findings=0
explicit current Available + undated historical Unavailable: state=undetermined
```

These `/tmp` paths are temporary review aids, not durable project fixtures. The
minimal inputs/results above preserve the relevant evidence in the report;
accepted fixes should introduce permanent regression tests during triage.

## Manual verification and remaining work

Phase 2 explicitly has no manual gate. Progress remains 7 of 18 rows checked;
phase 3 is next. All four phase 4 manual rows (4.4–4.7) remain unchecked. Live
subscription runs, report usability, source comparison and live interruption are
pending and are not claimed by this review. Markdown/report publication and
complete signal integration belong to phase 3 and are not missing phase 2 work.
CI was checked locally; this review did not run it on GitHub.

At the initial review, only this report and the skill-required `change.md` stamp
were written, with all three decisions pending. The subsequent approved triage
changes and verification are recorded below. Plan Progress, staging, and commits
remain unchanged.


## F1–F3 resolution — 2026-11-06

The operator requested applying all three fixes after reviewing the combined
triage recommendations. All decisions are now FIXED; zero pending findings remain.
No unrelated refactoring, plan Progress edits, live Azure calls, or commits were
performed.

### F1 — Credential-bearing URLs

`credentialValue` now recognizes URI schemes with letters, digits, plus, period,
and hyphen before `://` and detects authority userinfo before path/query/fragment
boundaries. The existing filtering before and after control stripping is retained.
Regression cases cover Redis, PostgreSQL, compound schemes, HTTP(S), embedded
connection URLs and control-obfuscated schemes. Ordinary URLs without userinfo,
email addresses and query-string contact addresses remain unchanged. A real
collector/checkpoint regression also verifies the Redis credential canary is
absent from saved JSON.

### F2 — Record-granular page acceptance

`collectionTransport` first calls the existing read-only transport, then validates
successful list envelopes and each record with the selected SDK record types.
Schema-invalid records become null placeholders before the native pager decodes
the page. The existing adapters and normalizer turn those placeholders into
rejected-record counts and explicit `invalid_records` coverage gaps. No rejected
payload enters the run snapshot. Native SDK pagination/authentication, the route
boundary, retry policy and checkpoint ordering remain in use.

The response handling runs below the SDK payload cache, closes the original body,
and supplies the validated body to the SDK. Non-transient application failures
are explicitly non-retriable; transient HTTP/network failures retain SDK retries.
Malformed envelopes (including missing/non-array lists and malformed nextLink)
remain failed pages, without retries or misleading page checkpoints.

Regressions use both real SDK pagers, fake HTTP responses and the actual runstore.
A valid resource/issue survives alongside a schema-invalid record and a null entry,
with a committed checkpoint containing one evidence record and two rejections
before the next request. Further cases prove that later throttling retries or an
unsafe cross-subscription continuation retain the valid checkpoint and rejected
record gap. Existing transport, redirect, registration, cancellation and storage
failure tests continue to pass.

### F3 — Current versus historical evidence

Health evidence now records `status_role` as `current` or `historical`, based on
the validated availability-status name. Historical evidence and references remain
in the snapshot with a `historical_health_evidence` gap. Only current evidence
participates in current-state selection, conflict resolution and findings.
Historical-only data yields an undetermined current observation and
`missing_current_health_evidence`. A newer historical timestamp cannot override
an explicit current state. Regression cases cover historical-only Available,
current Unavailable with historical Available, and current Available with
historical Unavailable. All evidence references remain resolvable.

The README documents all three behaviors. Changed implementation/test paths:

- `internal/health/privacy.go`, `privacy_test.go`
- `internal/health/evidence.go`, `normalize.go`, `normalize_test.go`
- `internal/azure/paging.go`, `record_decode.go`, `record_decode_test.go`
- `README.md`

### Post-fix verification

The three new counterexample regression tests first failed against the reviewed
implementation, then passed after the fixes. Final verification passed:

| Command | Result |
| --- | --- |
| `go test -count=1 ./internal/azure ./internal/health ./internal/cli ./internal/runstore` | PASS |
| `go build ./...` | PASS |
| `go test -race -cover -count=1 ./...` | PASS — Azure 87.3%, CLI 88.2%, health 94.4%, runstore 81.1%, executable entry 0.0%; no race failures |
| `go vet ./...` | PASS |
| `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS — zero reachable vulnerabilities; three required-module advisories outside called code |
| `./scripts/check-format.sh` | PASS |
| `shellcheck scripts/check-format.sh` | PASS |
| `actionlint` | PASS |

Triage complete: F1, F2 and F3 fixed. Phases 3–4 and all four live/manual acceptance
rows remain pending. `change.md` stays `impl_reviewed`, dated 2026-11-06. Work
remains uncommitted as requested.

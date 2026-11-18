# Active Operational Alerts Implementation Plan

## Overview

Deliver roadmap S-02 by extending the existing subscription health check with Service Health events and existing Azure Monitor metric/log alert instances. Preserve useful evidence across independent source failures and give the consultant a traceable Markdown report with explicit scope, time-window, and attribution limits.

The operator answered seven planning questions and approved this four-phase structure on 2026-11-06. Decisions are recorded in [change.md](change.md); external contracts and code references are in [research.md](research.md). This is a plan, not evidence that implementation or live acceptance has occurred.

Finalized on 2026-11-15. Start with [the plan brief](plan-brief.md) for scope, decisions and phase outcomes.

## Current State Analysis

S-01 is implemented and accepted. The CLI selects and validates one public-Azure subscription using the existing Azure CLI user identity, sequentially collects inventory and Resource Health, filters and checkpoints each completed page, and publishes Markdown only from committed JSON. Both artifacts identify the same run and revision. Scope/access failures, partial collection, interruption, and publication failures are already explicit.

Service Health, Monitor, and Advisor currently have unimplemented source entries. Source dispatch, collector outcomes, report wording, and several tests assume exactly two collectors. Evidence and findings largely assume a single affected resource. These assumptions must be extended without changing Resource Health semantics or weakening the GET-only request boundary.

The S-01 manual-check investigation found that implementation and fixtures had shared incorrect assumptions about external JSON and continuation URLs. S-02 therefore requires independent fixture provenance and positive compatibility cases alongside rejection tests. Accepted S-01 behavior remains the regression baseline; archived files and existing runs remain unchanged.

## Desired End State

The existing command runs inventory, Resource Health, Service Health, and Azure Monitor sequentially for the selected subscription. Its report includes all four agreed Service Health categories and source-reported Fired metric/log alerts, including acknowledged or closed alert instances. Category, condition, workflow response, attribution, and timestamps remain distinct.

Service Health uses a documented date query plus a local rolling 30-day lower cutoff. Monitor uses the API's explicit 30-day window. Both expose their limits. Missing resource attribution, unknown states, sensitive-detail restrictions, and inaccessible or failed checks remain visible. Advisor is still not checked and overall coverage remains incomplete; no overall resource OK classification is introduced.

All findings resolve to filtered, saved evidence. Tests prove source isolation, same-subscription boundaries, prior-page retention, cancellation, privacy in actual output bytes, and matching publication revisions. Final acceptance compares the two new sources with the portal in the agreed S-01 environment and records any unavailable scenarios honestly.

### Key Discoveries

- `internal/health/normalize.go:111` treats every non-inventory page as Resource Health; explicit dispatch is required before new page types are used.
- `internal/health/check.go:10` and its outcome logic name two collectors. Execution success must be generalized independently of complete coverage.
- `internal/azure/paging.go:124` builds its initial repetition key from a two-route assumption. New operations need their actual initial query policy.
- Service Health can return a global event ID from a subscription-scoped listing. Its listing supplies services/regions, not affected-resource IDs; identity and attribution are different contracts.
- Downloaded `armalertsmanagement v0.11.0` source uses a preview GetAll operation and escapes the entire scope. Use stable REST `2019-03-01` over the already installed Azure SDK ARM pipeline/pager instead of adding or modifying that module.
- `internal/cli/cli_test.go:40`, `internal/cli/integration_test.go:80`, and `cmd/devops-ai-agents/main_test.go:91` inject only the old sources. Update these boundaries before enabling new production defaults.
- `internal/runstore/store.go:71` already enforces committed-snapshot publication. Its mechanism is reusable; new evidence needs round-trip verification.

## What We're NOT Doing

- Advisor collection, overall OK classification, cross-source incident correlation, selected drafts, or ticket submission.
- Alert-rule discovery/configuration, metric/log queries, alert history/enrichment, or creation of monitoring.
- Per-event impacted-resource lookup, sensitive-detail POST calls, tenant/global collection routes, or cross-subscription resource expansion.
- Inferring affected resources from matching inventory service/region labels or from a log workspace's contents.
- Raw source descriptions, articles, arbitrary links, log query/results, usernames, tokens, or raw errors in saved artifacts.
- Historical comparisons, run resume, new retention rules, concurrent collectors/subscriptions, scheduling, or a UI.
- A new Monitor SDK dependency, changes to accepted authentication policy, production endpoint/token flags, automatic login/permission/provider repair, or commits.

## Implementation Approach

Extend the existing `azure → health → runstore → report` path. Adapters expose whitelisted input facts, health owns interpretation and source state, storage owns atomic snapshots, and rendering consumes only committed normalized evidence. The collector callback remains page-based; the run coordinator executes a fixed ordered list of named collectors.

Reuse `armresourcehealth v1.4.0` for Service Health. Implement Monitor's single stable REST list operation using existing `azcore v1.23.3` `arm.NewClient`, `runtime.NewPager`, and `runtime.FetcherForNextLink`, with private wire structs and the existing transport, credential, retry and page-deadline machinery. No raw response becomes a persistent artifact and no alternate HTTP/authentication stack is introduced.

Use schema version 2 for newly created runs because evidence can have a source entity and subscription scope without a resource target, and source entries now carry query provenance. Preserve schema-1 fixture/render compatibility where the existing renderer is exercised; do not migrate or rewrite old run directories. No new command to load old runs is required.

### Interpretation contract

| Area | Required behavior |
| --- | --- |
| Monitor active findings | Exact source condition `Fired`, with signal type `Metric` or `Log`; do not filter by New/Acknowledged/Closed response or notification suppression. Show these facts separately when present. |
| Monitor scope | Validate the alert's own ARM identity against the selected subscription. Validate its target independently. Missing, malformed, or foreign targets cannot become an affected-resource claim; retain selected-scope alert metadata with an attribution gap, omitting unsafe target values. A foreign alert identity is rejected. |
| Monitor limits | A workspace target is the reported target, not an enumeration of resources queried by the rule. Fired is a source condition, not proof of a continuing fault; stateless and retention limitations appear in the report. |
| Service Health findings | Source status Active in ServiceIssue, PlannedMaintenance, HealthAdvisory, or SecurityAdvisory, within the selected time scope. Keep categories distinct. Future-dated maintenance/advisories are notices, never ongoing outages. |
| Service Health identity | Accept a valid global ResourceHealth event identity or a valid selected-subscription event identity. Bind query provenance to the selected subscription; neither form authorizes global HTTP access. Do not filter solely on `eventSource`. |
| Service Health attribution | Preserve selected-subscription service/region facts and their source states/times. Omit foreign scope identifiers and tenant lists. Unattributed regions remain explicitly uncertain; contradictory explicit scope produces a gap and no confirmed affected-scope claim. No resource IDs are invented. |
| Sensitive advisories | Retain allowed metadata and an explicit detail limitation. Optional sensitivity flags may be absent. Do not retrieve additional details or treat withheld narrative as a malformed record. |
| Inactive and excluded records | Resolved observations and records outside the local time cutoff cannot create active findings. Known unselected Service Health categories are counted as excluded. Unknown category/signal/state remains unclassified evidence with a gap, not a healthy result. |
| Duplicates and conflicts | Collapse exact duplicate normalized evidence, excluding collection time from identity. Group new operational observations by source, selected subscription and native entity ID. Use a unique latest source update only when all conflicting observations are timed; conflicting ties or missing ordering facts remain undetermined. |
| Finding identity | New operational finding IDs are stable for the source/scope/entity, with references to the evidence supporting the selected state. Superseded or unresolved conflicts remain traceable in evidence and gaps without multiple contradictory active findings. Preserve existing Resource Health finding behavior. |

Recognizable but unknown domain facts can coexist with completed enumeration. Schema-invalid records still increment rejected-record counts and make collection partial. Do not use one counter or status to conflate known exclusions, unclassified evidence, and rejected records.

### Time-window contract

For Service Health, derive one UTC reference from `run.StartedAt` and a lower cutoff exactly 30 × 24 hours earlier. Send the UTC calendar date containing that cutoff as documented `M/D/YYYY` `queryStartTime`. Retain the reference, exact cutoff, and actual query date in source provenance. Compare returned `lastUpdateTime` against the exact cutoff locally; the extra portion of the queried calendar day must not create active findings. Missing source time produces unknown time coverage rather than an assumed in-window finding. A successful response cannot establish that older unchanged active events do not exist.

For Monitor, explicitly request `timeRange=30d` and record that API policy plus collection/request timing. Do not invent a local filter on a timestamp field that the API contract does not identify as equivalent. Neither API provides an atomic snapshot across pages; report collection timestamps and keep new-source observations separate from resource-fault assertions.

### Execution and coverage contract

The final fixed sequence is inventory → Resource Health → Service Health → Azure Monitor. All four are attempted independently unless scope validation, cancellation, or storage failure stops the run. Advisor remains not implemented. Query metadata must survive source-state updates and be checkpointed before the first request for that source.

| Exit | Final behavior |
| --- | --- |
| 0 | All four implemented collectors completed enumeration and publication succeeded; overall coverage remains incomplete. Empty successful lists do not establish OK. |
| 3 | At least one implemented collector is partial/failed and useful evidence or at least one completed collector remains. Preserve earlier pages and other source results. |
| 1 | Pre-collection scope/authentication failure, fatal execution/storage/publication failure, or no useful evidence/completed collector after all attempted collection fails. |
| 2 | Existing invalid-input, ambiguous/missing session subscription, unsupported identity/cloud, or repository-context cases. |
| 130 | Existing interruption precedence and retained-artifact diagnostics; stop new Azure reads and finalize from committed evidence when possible. |

Attribution/time/detail limitations are coverage facts, not automatically HTTP collection failures. A completed source may still have unknown coverage. Invalid records, failed pages, unsafe links, and exhausted bounds continue to produce partial/failed source status. Preserve separate execution outcome, source status, and report coverage.

A collector-local authentication or authorization failure follows the ordinary source-failure and partial-result rules; it does not suppress later collectors or override retained evidence with a pre-collection failure outcome. No collector repairs credentials or retries through another identity.

## Critical Implementation Details

### State sequencing

Normalize and filter each page, save it, then validate/follow its next link; an unsafe continuation must not discard the completed page. Update shared test injection before any new production default becomes reachable. Final report publication must continue to read the committed revision rather than the mutable in-memory run.

### Request construction

The Monitor adapter constructs the selected subscription path from the validated UUID, not from a caller-controlled scope or escaped whole ARM ID. Use a fixed valid semantic adapter version for `arm.NewClient` telemetry rather than an arbitrary CLI build-version string, because azcore validates module version syntax. Keep provider registration disabled and credential TenantID unset.

## Phase 1: Evidence and Coverage Contracts

### Overview

Introduce the new evidence forms, temporal provenance and deterministic interpretation rules, then make orchestration capable of named independent collectors. Keep production collection of the new sources disconnected until Phase 3.

### Changes Required

#### 1. Normalized model and source inputs

**Files**: `internal/health/model.go`, `internal/health/evidence.go`, new `internal/health/operational.go`, associated package tests.

**Intent**: Represent source entity identity and subscription attribution without forcing every event into a resource row. Keep structured source facts sufficient for generated summaries and evidence tracing.

**Contract**: Schema 2 adds typed Service Health and Monitor inputs/facts, optional target-resource attribution, source query provenance, and separate excluded/unclassified accounting. New evidence records retain operation/API version, source identity, selected subscription, collection/source timestamps and the accepted category/state/severity/impact fields. Finding resource targets may be absent. Existing Resource Health inputs and interpretation remain compatible.

#### 2. Normalization, current-state selection and privacy

**Files**: `internal/health/normalize.go`, `internal/health/privacy.go`, new `internal/health/operational_test.go`, existing normalization/privacy tests.

**Intent**: Derive active operational findings and explicit gaps under the interpretation and time-window contracts. Prevent source text and foreign scope details from reaching checkpoints.

**Contract**: Explicit source dispatch; identity validation before acceptance; SafeText filtering of permitted labels; distinct invalid, excluded and unclassified cases; stable identities and deterministic conflict resolution. New evidence never becomes a Resource Health observation. Resource-enumeration emptiness is derived from resource evidence, not total evidence length. Preserve raw-byte privacy assertions and current Resource Health regressions.

#### 3. Named collectors and test isolation

**Files**: `internal/health/check.go`, `internal/health/check_test.go`, `internal/cli/cli.go`, `internal/azure/record_decode_test.go`, CLI test constructors and `cmd/devops-ai-agents/main_test.go`.

**Intent**: Remove the two-collector outcome assumption and prepare safe injection for the two new collectors.

**Contract**: Check accepts an ordered collection of source-name/collector pairs, retains page callbacks and save-failure precedence, and calculates outcomes over the supplied implemented sources. The CLI still supplies its existing production sources during this phase. Add the new injectable fields and inert synthetic defaults to test constructors before Phase 3 wiring; no test may fall through to real session/network access. Source transitions preserve provenance/counters.

Migrate both `health.Check` calls in `internal/azure/record_decode_test.go` to the named-collector signature during this phase so existing record-isolation tests remain compilable and preserve their assertions.

### Success Criteria

#### Automated Verification

- Health tests prove temporal boundaries, attribution, classification, duplicate/conflict handling, privacy and unchanged Resource Health behavior: `go test ./internal/health`.
- Orchestration and synthetic injection tests prove independent sources, outcome rules and no unintended production fallback: `go test ./internal/cli ./cmd/devops-ai-agents`.
- All packages compile and Go formatting passes: `go build ./...` and `./scripts/check-format.sh`.
- Azure record-isolation tests pass after migrating coordinator callers: `go test ./internal/azure`.

**Implementation Note**: Continue after automated verification. No intermediate operator or live Azure check is required.

---

## Phase 2: Source Adapters and Read Boundary

### Overview

Implement both source adapters behind narrowly defined operation policies and prove the external contracts through independent synthetic wire fixtures.

### Changes Required

#### 1. Service Health adapter

**Files**: new `internal/azure/servicehealth.go`, `internal/azure/record_decode.go`, Azure collection tests.

**Intent**: Collect subscription Service Health events with the existing generated Events pager and project only agreed structured facts.

**Contract**: `Collection.ServiceHealth` implements the existing Collector shape using `armresourcehealth v1.4.0` subscription Events API `2025-05-01`. Its immutable per-run query policy contains the approved cutoff/date. It emits every successfully decoded page, including valid empty pages and invalid-record placeholders, through the normalization/checkpoint callback. No per-event or impacted-resource calls. Preserve the optional-sensitive and global-event-ID cases.

#### 2. Monitor adapter

**Files**: new `internal/azure/monitor.go`, `internal/azure/record_decode.go`, Azure collection tests.

**Intent**: Retrieve existing alert instances through the stable list contract while retaining the current Azure SDK authentication/retry/paging behavior.

**Contract**: `Collection.AzureMonitor` implements Collector through existing azcore ARM pipeline/runtime pager. Private wire types describe only the list envelope, alert identity, and accepted essentials. Initial query uses `api-version=2019-03-01`, `timeRange=30d`, `monitorCondition=Fired`, `includeContext=false`, `includeEgressConfig=false`, and `pageCount=100`. Select Metric/Log locally; the stable operation has no signalType query filter. Do not request histories, rules, summary, enrichments, or tenant routes. Leave go.mod/go.sum unchanged.

#### 3. Route policy, continuation and record isolation

**Files**: `internal/azure/transport.go`, `internal/azure/paging.go`, `internal/azure/record_decode.go`, transport/collection/record-decode tests.

**Intent**: Admit the verified operations and valid paging shapes without weakening existing subscription, route, method or payload protections.

**Contract**: Use the operation policy below for initial requests and every continuation/retry. The shared page helper receives the actual initial request identity, preserves existing deadlines/retries/page limit, and canonicalizes equivalent continuation keys. Validate envelopes and isolate malformed individual records before page decoding; no raw body or decoder error text crosses the adapter boundary.

| Operation | Allowed path/version | Query and continuation policy |
| --- | --- | --- |
| Service Health | `/subscriptions/{selected}/providers/Microsoft.ResourceHealth/events`; `2025-05-01` | Initial fixed queryStartTime; documented `$skipToken` continuation. A continuation may omit the date, but any supplied date must match the run policy. Reject arbitrary filters and undocumented paging additions; retain prior pages if a service variant is unsupported. |
| Monitor | `/subscriptions/{selected}/providers/Microsoft.AlertsManagement/alerts`; `2019-03-01` | Fixed initial options above. Continuation requires a nonempty single `ctoken`; it may omit original options, but any repeated options must retain the selected window/condition and false context/egress flags. Permit a single documented pageCount in 1–250 and optional default sortBy=lastModifiedDateTime/sortOrder=desc. No additional filters/expansions. |

Permit `management.azure.com:443` only for the new Monitor route, as demonstrated by the external paging example, and canonicalize it to the same repetition identity as the implicit port. Other hosts/ports, redirects, user info, fragments, encoded path aliases, mismatched API versions, duplicate query keys, different subscriptions/operations, and non-GET methods remain rejected. Existing route behavior is unchanged. Continuation tokens remain memory-only and never appear in saved provenance or diagnostics.

The Monitor documentation's example changes subscription and API version. Use the `ctoken`/port syntax as a positive structural example with explicitly documented synthetic substitutions, and retain the original cross-scope/version changes as negative cases. Never weaken the boundary to accept that example wholesale.

#### 4. Independent external fixtures

**Files**: new `internal/azure/testdata/service-health-events.json`, `monitor-alerts.json`, and `README.md`; Azure adapter tests.

**Intent**: Test compatibility against independently sourced field names and paging shapes instead of reproducing the implementation's assumptions.

**Contract**: Literal synthetic payloads cite the official operation/version and enumerate intentional substitutions. Include Service Health global IDs and omitted-date continuation, Monitor nested essentials and ctoken/default-port continuation, and metadata-only sensitive events. Expected requests are independently written; do not produce both payloads and assertions from the model under test. Additional adversarial fixtures exercise valid siblings and excluded sensitive fields.

### Success Criteria

#### Automated Verification

- Adapter and literal-fixture tests prove both external shapes, exact initial requests and legitimate continuations: `go test ./internal/azure`.
- Azure race tests prove unsafe requests never reach the base transport and prior pages survive failures, malformed siblings, retry exhaustion and cancellation: `go test -race -count=1 ./internal/azure`.
- All packages compile and Go formatting passes: `go build ./...` and `./scripts/check-format.sh`.

**Implementation Note**: Continue after automated verification. Documented compatibility is not live service acceptance; no operator Azure session is used by these checks.

---

## Phase 3: CLI and Report Integration

### Overview

Wire both adapters into the existing command and publish an operational report that preserves source meaning, traceability and incomplete coverage.

### Changes Required

#### 1. Production orchestration and diagnostics

**Files**: `internal/cli/cli.go`, CLI collection/debug/report tests.

**Intent**: Execute the four implemented collectors in the fixed order and expose each source's outcome using the established CLI behavior.

**Contract**: Build immutable new-source query policies from the run reference and checkpoint them before collection. Wire the two new injected/production collectors; no new user flags. Add source status diagnostics and apply the execution table above. Preserve stdout as the published report path only, filtered stderr/debug output, graceful cancellation and saved-artifact messages.

#### 2. Operational report and evidence index

**Files**: `internal/report/markdown.go`, `internal/report/markdown_test.go`, synthetic report fixtures under `internal/report/testdata/`.

**Intent**: Make the two new sources useful without presenting subscription notices or source Fired states as proven resource outages.

**Contract**: Use a subscription-health report title and derive checked/unchecked wording from source status. Present Service Health categories and Monitor alert conditions/workflow/severity with timestamps, explicit attribution and evidence references. Separate generated active summaries from inactive/unclassified evidence and coverage gaps. Preserve all saved evidence in the index; filtered source values render literally. Do not show empty resource IDs as confirmed resources or create Resource Health observations for subscription events.

Show the Service Health cutoff/query date, Monitor 30-day API policy, collection times, lack of an atomic snapshot, missing resource-impact coverage and any sensitive-detail limits. Explain stateless Fired/retention limits once in the source section. Advisor remains not checked and the report remains INCOMPLETE, including when every implemented collector completes or returns an empty list.

#### 3. Snapshot compatibility and operator guidance

**Files**: `internal/runstore/store_test.go`, `README.md`, `AGENTS.md`, `context/foundation/development-tools.md` where descriptions of implemented sources require updating.

**Intent**: Preserve publication guarantees and document the expanded behavior without changing accepted historical records.

**Contract**: New facts survive Save → committed JSON → render with matching run/revision; unresolved references still fail publication. Keep a schema-1 rendering regression without modifying old runs. Update command behavior, source permissions, query limits, exit meanings and inspection guidance. Correct touched S-01 verification links to the archived location. Keep historical verification sections historical; do not claim S-02 acceptance before Phase 4.

### Success Criteria

#### Automated Verification

- CLI/report/store tests prove four-source wiring, honest source wording, deterministic literal output, evidence references and schema-1/schema-2 publication: `go test ./internal/cli ./internal/report ./internal/runstore`.
- CLI/report/store race tests preserve independent failure, cancellation, storage failure and raw-byte privacy behavior: `go test -race -count=1 ./internal/cli ./internal/report ./internal/runstore`.
- All packages compile and Go formatting passes: `go build ./...` and `./scripts/check-format.sh`.

**Implementation Note**: Continue after automated verification. Operator evaluation of report usefulness and portal comparison belongs to Phase 4.

---

## Phase 4: Integrated Verification and Operator Acceptance

### Overview

Verify the complete command path under representative scale and adverse conditions, then complete controlled manual acceptance in the agreed environment.

### Changes Required

#### 1. Four-collector process and integration scenarios

**Files**: `internal/cli/integration_test.go`, `cmd/devops-ai-agents/main_test.go`, shared synthetic helpers and fixtures.

**Intent**: Exercise the actual command, fake CLI credential process, Azure SDK pipelines/pagers, normalization, storage and report together.

**Contract**: Cover hundreds of inventory resources, multiple pages from every collector, fewer than ten active operational findings in a representative case, and additional paging stress without inventing a latency SLA. Cover each new source failing independently, both new sources failing with retained S-01 evidence, old sources failing while operational evidence succeeds, all sources failing, and all lists empty. Check checkpoints before subsequent requests.

Inject source failures after a committed page, unsafe continuations/redirects, 401/403/404/429/timeouts, malformed envelopes and mixed records, storage/publication failures, and unknown/conflicting states. Verify that collector-local authentication failures preserve independent collection and partial-result behavior. Synchronize process interruption after a Service Health or Monitor checkpoint without sleeps; prove no later collection starts and prior evidence remains. Retain existing first/second interrupt semantics and output-delivery diagnostics.

#### 2. Verification record and acceptance instructions

**Files**: new `context/changes/active-alert-report/verification.md`, operator guidance as needed.

**Intent**: Separate synthetic assurance, actual service compatibility and the operator's acceptance.

**Contract**: Record exact automated commands/results and sanitized manual pass/fail/limitations. Follow the manual procedure below using the agreed S-01 environment and externally established login. Keep live evidence in ignored runs; do not copy real identifiers or payloads into fixtures/docs, create incidents, or change permissions/providers. Missing live scenarios remain explicitly not live-verified even when synthetic tests pass.

### Success Criteria

#### Automated Verification

- Full-flow tests cover four collectors, source isolation, paging, attribution, raw-byte privacy and synchronized interruption: `go test -race -count=1 ./internal/cli ./cmd/devops-ai-agents`.
- All required repository checks in Testing Strategy pass against real packages, with results recorded in verification.md.
- Synthetic artifact checks prove matching run/revision, resolvable findings, explicit incomplete coverage, and generated run files excluded from Git.

#### Manual Verification

- The operator confirms selected scope and compares both new source results with portal evidence using the documented windows and category/condition rules; absent scenarios are recorded explicitly.
- The operator accepts generated summaries, evidence tracing, workflow/category separation, time limits, uncertain resource attribution and incomplete coverage in the Markdown report.
- The operator confirms interruption retains committed evidence and produces cancelled/incomplete output when finalization succeeds, with actual limits recorded.
- Sanitized acceptance results and unavailable scenarios are recorded; private run artifacts remain ignored and no live identifiers or source payloads enter tracked files.

**Implementation Note**: After automated verification, wait for operator confirmation of these manual items before marking them complete or treating S-02 as accepted. Do not infer manual success from automated tests or the approval of this plan. Keep changes uncommitted unless separately authorized.

---

## Testing Strategy

### Unit Tests

- All four Service Health categories, Active/Resolved/unknown state, optional sensitivity, global versus scoped event IDs, foreign/missing regional attribution, missing source time and exact 30-day cutoff boundaries.
- Metric/Log Fired with every workflow response and suppression state; Resolved/unknown condition; unknown signal type; nested alert identities; missing/foreign target; workspace attribution limits; optional/malformed severity and timestamps.
- Exact duplicates across pages, uniquely ordered updates, conflicting ties, missing ordering facts, and stable finding-to-evidence references. Do not infer state from retrieval order or start time when a source update time is needed.
- Pre-checkpoint filtering and actual serialized JSON/Markdown/stderr canaries for excluded descriptions, articles, query/results, foreign IDs, usernames, credentials, tokens and raw errors. Literal markup stays literal.
- Source status/counters/provenance persistence; empty resource inventory with operational evidence; partial versus completed enumeration; saved schema-2 round trips and existing schema-1 rendering.

### Integration Tests

Use the existing fake CLI process and strict synthetic ARM transports; never use the operator's session or live network. Service Health runs through its generated pager; Monitor runs through the same SDK ARM auth/retry infrastructure and runtime pager used by production. Assert method, route, subscription, version and fixed query policy at the fake wire, including attempts during retries.

Independent fixtures must establish both positive compatibility and negative boundaries. Document which fields/URLs originate in official examples and which are synthetic adaptations. A self-consistent fake producer/consumer pair alone is not sufficient evidence. Existing Resource Health and inventory regressions remain enabled.

### Required Repository Checks

Run these after the final implementation/test changes. Follow the existing temporary-cache guidance if the sandbox blocks user cache writes; do not describe empty package sets or failed vulnerability-database access as passing results.

```sh
go build ./...
go test -race -cover -count=1 ./...
go vet ./...
staticcheck ./...
govulncheck ./...
./scripts/check-format.sh
shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh
actionlint
```

### Manual Testing Steps

1. Use the same authorized environment as S-01 after external login. Build the binary into a temporary location and run `health check --subscription "<subscription-id>"`; repeat with the intended unique full name as useful. Inspect the binary's actual exit status, not the `go run` wrapper status.
2. Open report.md beside run.json and verify subscription, run ID, revision, source states and time-window provenance. Compare Monitor with a 30-day portal view and Fired condition across New/Acknowledged/Closed responses; compare Service Health across the four selected categories using update-time scope rather than assuming every portal view has the same filter semantics.
3. Trace each available new-source finding to its saved evidence and original source identity. Confirm that regional/subscription notices do not become invented resource faults and that Monitor workflow state is independent of Fired. Record absent active examples as not live-verified.
4. Inspect source gaps and all-empty behavior where naturally available. Confirm Advisor remains unchecked and no overall OK appears. Do not create outages, rules, permission failures or synthetic live events to manufacture scenarios.
5. Interrupt an in-progress run once, inspect retained pages and cancelled/incomplete publication, and record the actual stage exercised. If no new-source page can be interrupted in the environment, retain the automated process evidence and disclose that manual limitation.
6. Check generated paths with `git check-ignore -v -- runs/<run-id>/run.json runs/<run-id>/report.md` and `git ls-files -- runs/`. Record only sanitized outcomes in verification.md and obtain explicit operator acceptance.

## Performance Considerations

Continue sequential collection with the existing per-page 60-second deadline, three SDK retries and 10,000-page source ceiling. Monitor requests 100 records per page; Service Health uses service paging. No per-event/resource fan-out is introduced. Retained evidence and repeated checkpoints still consume time and memory proportional to observed data; use representative multi-page tests to detect regressions without adding a product SLA or replacing the store speculatively.

The 30-day Monitor query may return many retained stateless Fired instances even when few distinct conditions currently matter. Preserve instance identity and source meaning; do not silently cap findings, merge alerts by rule, or label retention-expired instances healthy.

## Migration Notes

New runs use schema 2. Old artifacts remain immutable and continue to be inspectable as saved; schema-1 report tests prevent accidental assumptions that new metadata always exists. There is no persisted resume state, database migration or artifact rewrite.

Rollback is a source/build change applied to future invocations, not deletion or modification of existing runs. Coordinate shared health/CLI/report contracts with a future S-03 change; this plan does not authorize parallel runtime collection or modifications to an in-progress S-03 implementation.

## References

- Operator decisions: [change.md](change.md); contract provenance and verified SDK behavior: [research.md](research.md).
- Scope: `context/foundation/prd.md`, `context/foundation/roadmap.md` S-02; stack: `context/foundation/tech-stack.md`.
- Regression baseline: `context/archive/2026-11-05-subscription-resource-health-report/verification.md` and `manual-check-investigation.md` in the same archive folder.
- Existing contracts: `internal/health/check.go:10`, `internal/health/normalize.go:111`, `internal/azure/transport.go:23`, `internal/azure/paging.go:104`, `internal/report/markdown.go:64`, `internal/runstore/store.go:71` (planning-time locations).
- Service API: [Events list](https://learn.microsoft.com/en-us/rest/api/resourcehealth/events/list-by-subscription-id?view=rest-resourcehealth-2025-05-01); [sensitive advisory access](https://learn.microsoft.com/en-us/azure/service-health/access-service-advisories-api).
- Monitor API: [Alerts Get All](https://learn.microsoft.com/en-us/rest/api/alerts-management/alerts/alerts/get-all?view=rest-alerts-management-alerts-2019-03-01); [alert state and retention semantics](https://learn.microsoft.com/en-us/azure/azure-monitor/alerts/alerts-overview).
- Required tools and CI: `context/foundation/development-tools.md`, `.github/workflows/checks.yml`.

## Progress

> Convention: `- [ ]` pending, `- [x]` done. Append a commit SHA only if a commit is separately authorized. Do not rename step titles. Phase criteria above use plain bullets; this section alone owns checkbox state.

### Phase 1: Evidence and Coverage Contracts

#### Automated

- [x] 1.1 Health tests prove temporal boundaries, attribution, classification, duplicate/conflict handling, privacy and unchanged Resource Health behavior: `go test ./internal/health`.
- [x] 1.2 Orchestration and synthetic injection tests prove independent sources, outcome rules and no unintended production fallback: `go test ./internal/cli ./cmd/devops-ai-agents`.
- [x] 1.3 All packages compile and Go formatting passes: `go build ./...` and `./scripts/check-format.sh`.
- [x] 1.4 Azure record-isolation tests pass after migrating coordinator callers: `go test ./internal/azure`.

### Phase 2: Source Adapters and Read Boundary

#### Automated

- [x] 2.1 Adapter and literal-fixture tests prove both external shapes, exact initial requests and legitimate continuations: `go test ./internal/azure`.
- [x] 2.2 Azure race tests prove unsafe requests never reach the base transport and prior pages survive failures, malformed siblings, retry exhaustion and cancellation: `go test -race -count=1 ./internal/azure`.
- [x] 2.3 All packages compile and Go formatting passes: `go build ./...` and `./scripts/check-format.sh`.

### Phase 3: CLI and Report Integration

#### Automated

- [x] 3.1 CLI/report/store tests prove four-source wiring, honest source wording, deterministic literal output, evidence references and schema-1/schema-2 publication: `go test ./internal/cli ./internal/report ./internal/runstore`.
- [x] 3.2 CLI/report/store race tests preserve independent failure, cancellation, storage failure and raw-byte privacy behavior: `go test -race -count=1 ./internal/cli ./internal/report ./internal/runstore`.
- [x] 3.3 All packages compile and Go formatting passes: `go build ./...` and `./scripts/check-format.sh`.

### Phase 4: Integrated Verification and Operator Acceptance

#### Automated

- [x] 4.1 Full-flow tests cover four collectors, source isolation, paging, attribution, raw-byte privacy and synchronized interruption: `go test -race -count=1 ./internal/cli ./cmd/devops-ai-agents`.
- [x] 4.2 All required repository checks in Testing Strategy pass against real packages, with results recorded in verification.md.
- [x] 4.3 Synthetic artifact checks prove matching run/revision, resolvable findings, explicit incomplete coverage, and generated run files excluded from Git.

#### Manual

- [x] 4.4 The operator confirms selected scope and compares both new source results with portal evidence using the documented windows and category/condition rules; absent scenarios are recorded explicitly.
- [x] 4.5 The operator accepts generated summaries, evidence tracing, workflow/category separation, time limits, uncertain resource attribution and incomplete coverage in the Markdown report.
- [x] 4.6 The operator confirms interruption retains committed evidence and produces cancelled/incomplete output when finalization succeeds, with actual limits recorded.
- [x] 4.7 Sanitized acceptance results and unavailable scenarios are recorded; private run artifacts remain ignored and no live identifiers or source payloads enter tracked files.

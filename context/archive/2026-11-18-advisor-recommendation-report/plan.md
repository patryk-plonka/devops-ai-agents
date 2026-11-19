# S-03 · Azure Advisor recommendation report implementation plan

## Overview

Extend `health check --subscription <id-or-full-name>` with existing Azure Advisor recommendations. The consultant gets a prioritized daily review in the same Markdown report, with per-instance evidence in the sibling JSON snapshot. The operator approved the four phases and the seven report and acceptance decisions on 2026-11-18. [Decision summary](plan-brief.md)

## Current State Analysis

- **Execution.** The CLI overwrites the initial Advisor state with `not_checked/not_implemented` before creating the store. Its default collector list contains inventory, Resource Health, Service Health and Monitor. The coordinator already saves before the first request, after each accepted page and after the source finishes. A storage error stops further reads. [CLI initialization](../../../internal/cli/cli.go#L96) · [Collector order](../../../internal/cli/cli.go#L272) · [Coordinator](../../../internal/health/check.go#L16)
- **Azure boundary.** The existing ARM pipeline binds the credential to the selected subscription, disables provider registration and restricts the transport to one route. The pager checks continuations before fetching. The record decoder needs a route-specific projection to isolate invalid entries. [Collection options and paging](../../../internal/azure/paging.go#L31) · [Route validation](../../../internal/azure/transport.go#L27) · [Record decoder](../../../internal/azure/record_decode.go#L47)
- **Stored evidence.** New runs currently use schema 2. Evidence already has separate entity, subscription and resource IDs. Evidence identity excludes collection time. The current operational entity validator expects path shapes that differ from a recommendation nested beneath an assessed resource. [Model](../../../internal/health/model.go#L65) · [Evidence](../../../internal/health/evidence.go) · [Identity and normalization](../../../internal/health/normalize.go#L92) · [Operational entity validation](../../../internal/health/operational.go#L75)
- **Rendering.** Existing Resource Health findings can have an empty source field. The renderer selects them by excluding the two S-02 sources. Simply adding Advisor findings would put suggestions in the Resource Health table. Both historical schema examples have tests and matching snapshots. [Finding creation](../../../internal/health/normalize.go#L338) · [Report](../../../internal/report/markdown.go#L118) · [Fixture pairs](../../../internal/report/testdata/README.md)

## Desired End State

One run reads inventory and all four agreed health sources. The Advisor section leads with New suggestions. Other returned statuses remain reviewable. A group of repeated suggestions expands into individually identifiable rows with category, impact, assessed resource or **Resource unknown**, last evaluation time or **unknown**, and evidence references. A group count represents distinct instances. The evidence index carries native identity, API version and times for comparison with Azure.

The operator can distinguish successful empty enumeration, a list that stopped partway, and a source that could not be checked. Earlier findings survive an Advisor failure. Even a successful five-collector run retains the S-03 **INCOMPLETE** coverage statement.

## Scope boundaries

S-03 reads cached recommendations for one selected subscription. Recommendation generation, status changes, suppression or remediation actions, tenant-wide reads, extra lookups, custom detection and comparisons across runs are outside the slice. S-04 owns overall resource classification. S-05 owns finding selection and drafts. Keep all live evidence in the local ignored run directory.

## Implementation Approach

Use the documented stable `2026-10-01` subscription list through the project's existing azcore ARM client and runtime pager. This operation exposes the workflow and time fields required by the approved design. Request the unfiltered list, classify locally, and record the operation, version, scope and query preparation time. The report describes the cache returned at collection time. There is no 30-day Advisor window or cache-refresh call. [Microsoft Advisor list operation and example](https://learn.microsoft.com/en-us/rest/api/advisor/recommendations/list?view=rest-advisor-2026-10-01)

Use schema 3 for new S-03 runs when the CLI integration lands. Preserve schema 1 and 2 rendering from the existing snapshots. Keep a stable instance identity beneath the report grouping so later status changes or a change in group order cannot break traceability. Reuse the coordinator, committed-snapshot publisher, diagnostic categories and literal Markdown escaping.

## Critical Implementation Details

**Identity and target.** The list URL is subscription-scoped, but a returned instance ID can end in an Advisor extension below a resource. Its final name can be an opaque string. The type GUID used for grouping is a separate optional field. A recommendation ID is evidence of the instance; assessed-resource attribution requires its own validation.

**Checkpoint order.** Normalize and commit the current page before interpreting its continuation. An invalid next link should leave that page available for a partial report. The stored query records configured policy and preparation time independently so cancellation before Advisor starts is visible.

## Phase 1: Advisor evidence

### Changes required

**Health inputs and saved facts** · `internal/health/evidence.go`, `internal/health/model.go`, new Advisor-specific health code and package tests.

**Intent.** Preserve the minimum structured facts needed to identify and prioritize a suggestion and distinguish source time from collection time.

**Contract.** Extend `Page` with Advisor inputs and `Evidence` with optional typed Advisor facts. Carry instance ID, selected subscription, optional validated type GUID, category, impact, workflow status, optional assessed resource, source evaluation time (`lastRefreshed`), source state-change time (`lastUpdated`) and collection time. Record the list operation and API version. Read original enum values exactly before classifying. Persist only accepted values or a fixed unknown marker. Exclude source-authored text, links, action payloads, arbitrary properties, identities and credential-like values before any checkpoint. Adding optional fields must preserve identities of existing evidence.

When configuring the Advisor query, save the list operation and API version on the Advisor source. In the same checkpoint as the first accepted page, set both the first and last page times to that page’s UTC collection time. For each subsequent accepted page, update the last page time, including when the page is empty or contains only duplicates. Omit both timestamps until a page is accepted.

**Normalization and derived views** · `internal/health/normalize.go`, new `internal/health/advisor.go` and corresponding tests.

**Intent.** Turn individually traceable, subscription-scoped records into a daily review without inventing targets or overstating ambiguous states.

**Contract.** Apply these rules on each rebuild from evidence:

| Input | Saved or derived result |
| --- | --- |
| Scoped, safe ARM instance ending in `providers/Microsoft.Advisor/recommendations/<name>` | Accept subscription-, resource-group- and resource-scoped instances with a validated parent path. Compare ARM IDs case-insensitively. Reject foreign or malformed instance IDs, count the rejection and discard their content. |
| Assessed resource | Use a valid selected-subscription `resourceMetadata.resourceId`. If a resource-scoped instance parent identifies a different resource, record an attribution conflict. Missing, unsafe, foreign or conflicting targets leave the instance visible with an unknown target and a fixed limitation. Use no inventory-name or `impactedValue` matching. |
| Missing type GUID, category or impact | Preserve the instance and mark the field unknown. An absent or invalid type GUID yields an individual group labelled “Type unknown” and identified by instance. Prevent unrelated untyped records from merging. |
| New, Postponed, Dismissed, Completed, or missing/unrecognized status | Use the source status. Only a determined New instance enters the daily findings list. Preserve other instances and their evidence in the secondary review table. Unknown state produces an explicit classification gap. |
| Duplicate or changing evidence | Collapse exact repeated observations across pages. When source facts for one native instance differ only in `lastRefreshed`, show one row with the agreed workflow status, the latest known evaluation time and every distinct evidence reference. For contradictory observations, use source state-change times to select a unique latest version. Tied or unordered contradictions stay traceable with an explicit gap and an unknown summary. A later evaluation or collection time alone cannot select a conflicting workflow state. |
| Multiple instances of one type | Maintain distinct instance identities and evidence references. A New finding has a stable ID based on source, subscription and canonical native instance ID. Grouping is a presentation decision. |

Keep missing source timestamps unknown. Preserve distinct times in the evidence index, including the first observation time for an exact duplicate. Source completion reports whether enumeration finished. Invalid records make the source partial under the existing coordinator contract. Individually valid records may have unknown facts even when enumeration completes.

### Success criteria

#### Automated verification

- Verify scoped identity, structured facts, attribution and privacy with health package tests.
- Verify duplicate, conflict, unknown-field and stable per-instance finding behavior with health package tests.

Run `go test -race -count=1 ./internal/health`. Include known and unknown enums, mixed-case ARM joins, distinct targets of one type and two unrelated missing-type instances. Compare rebuilt output after reversing input order using fixed collection times. In both page orders, verify that two New observations differing only in `lastRefreshed` yield one New finding with the later evaluation time and both evidence IDs. With New and Dismissed tied on `lastUpdated`, verify an unknown status, an explicit gap and both evidence IDs.

## Phase 2: Read-only collection

### Changes required

**Advisor list adapter** · new `internal/azure/advisor.go`, `internal/azure/record_decode.go`, `internal/azure/testdata/` and Azure package tests.

**Intent.** Read the cache through the existing credential and transport. Keep valid siblings on a mixed page and reject an invalid page envelope as a source failure.

**Contract.** Start with a bodyless GET to `/subscriptions/{selected-id}/providers/Microsoft.Advisor/recommendations?api-version=2026-10-01` on the fixed public ARM host. Use one private projection for the record filter and the pager. An empty `value` array is a successful page. A missing, null or wrongly typed list, or malformed continuation field, fails the envelope. A malformed typed record becomes a rejected entry. Include a literal fixture derived independently from the published response shape, using synthetic IDs, a resource-extension instance, missing type GUID, distinct source times and excluded fields. Document its provenance alongside existing fixtures.

**Transport and pager** · `internal/azure/transport.go`, `internal/azure/paging.go`, existing Azure transport and collector tests.

**Intent.** Add exactly the Advisor list operation to the allowlist while preserving authentication, retry and redirection boundaries.

**Contract.** Validate method, host, selected subscription, path, fixed version and query on every attempt. Allow the documented token spelling `$skipToken` and the sample's `$skiptoken` on same-route continuations, once and nonempty. Accept a single positive int32 `$top` on a token-bearing link. Reject duplicate aliases, changed versions, extra filters, unexpected routes, redirects and foreign scope. Canonicalize equivalent continuation identities across token spellings, query order and page-size metadata for loop detection. Treat tokens as opaque and transient. Retain the shared 60-second per-page timeout, retry policy and 10,000-page ceiling. Preserve fixed failure reasons without raw bodies or URLs.

### Success criteria

#### Automated verification

- Verify the native Advisor pager against independent success, empty and malformed response fixtures.
- Verify the exact request boundary and retention of committed pages across continuation and source failures.

Run `go test -race -count=1 ./internal/azure ./internal/health`. Inspect outgoing synthetic requests and saved pages. Cover the example continuation, equivalent repeats, late denial/throttling, deadline, malformed next page and mixed siblings. Assert forbidden requests never reach the underlying transport.

## Phase 3: Daily review in Markdown

### Changes required

**CLI and saved schema** · `internal/cli/cli.go`, `internal/health/model.go`, CLI, command and runstore test constructors.

**Intent.** Make Advisor available through the normal command and account for it from the first checkpoint.

**Contract.** Activate schema 3, initialize Advisor as `not_checked/not_attempted`, prepare the cached list policy after scope validation, and append Advisor after Monitor. Extend the existing Go injection boundary and audit every test constructor that can reach default collection. Keep tests on fake credentials and transports. Preserve subscription-bound CLI credentials, unset credential TenantID, provider-registration settings, stdout report path, stderr source diagnostics and exit codes. State which source failed when the others succeeded.

**Report and operator guidance** · `internal/report/markdown.go`, new `internal/report/advisor.go`, report tests and fixture pair, `README.md`, current-flow statements in `AGENTS.md`.

**Intent.** Let the consultant prioritize suggestions and inspect evidence without mistaking an Advisor recommendation for a confirmed fault.

**Contract.** Add a dedicated Advisor section for schema 3 with saved source status and separate wording for completed-empty, failed, partial and unattempted reads. Render the cached query scope, list operation, API version, query preparation time and first and last accepted-page times from committed source metadata. If the source has zero accepted pages, say so. Show retained rows on partial reads. Preserve Resource Health findings with an omitted source field. Suppress Advisor from Resource Health and operational tables. Use the existing literal rendering and publish solely from the committed run.

| Report element | Operator-visible behavior |
| --- | --- |
| Daily list | New instances first. One separate table covers Postponed, Dismissed, Completed and unknown in that order, with status visible. Superseded observations remain accessible in the evidence index. |
| Groups | Within each status group by type GUID, count distinct instances and sort groups by maximum recognized Advisor impact (High, Medium, Low, unknown). Break ties by canonical type/instance ID. Count a repeatedly observed instance once. |
| Rows | Keep each instance’s category, individual impact, assessed resource or **Resource unknown**, **Last evaluated (UTC)** or **unknown**, instance identifier and evidence references visible. Sort by individual impact then canonical instance ID within the group. Include the finding ID for New items. |
| Evidence index | Include source operation, version, instance/type identities, status, target limitation, source evaluation and state-change times, collection time and links to the matching sibling JSON. The report renders structured facts only. |

Provide a synthetic schema-3 report with its own sibling `run.json`. Keep schema-1 and schema-2 examples paired with their original artifacts. Document the lookup from the row to the saved evidence ID and the Azure instance. Explain the cache, unknown target and overall coverage limit.

### Success criteria

#### Automated verification

- Verify fifth-collector wiring, initial state, query timing, exit status and committed-snapshot publication.
- Verify deterministic grouped rows, field labels, impact ordering and per-instance evidence links.
- Verify schema-1 and schema-2 golden output and legacy Resource Health finding visibility.

Run `go test -race -count=1 ./internal/cli ./internal/report ./internal/runstore ./cmd/devops-ai-agents`. Include 30 same-type instances, mixed impacts, unknown types and resources, empty and partial sources. Follow each fixture link and compare run ID, revision and evidence ID. Verify the renderer does not mutate the snapshot.

Render from committed `run.json` with fixed, distinct preparation and page times. For one completed empty page, verify the operation, version and equal first and last page times. For a duplicate-only final page, verify the last page time advances while the original evidence retains its collection time. For failure before the first page, verify both page timestamps are absent from JSON and the report states that zero pages were accepted.

## Phase 4: Verification and operator acceptance

### Changes required

**Integrated scenarios** · `internal/cli/operational_integration_test.go`, `cmd/devops-ai-agents/main_test.go`, related integration helpers.

**Intent.** Verify the real command orchestration from subscription selection to publication using synthetic processes and native ARM pagers.

**Contract.** Exercise all five collectors in order on populated and empty pages. Check Advisor denial after earlier successful sources, an earlier collector failing with Advisor evidence retained, all five failing, invalid siblings, and an interruption after an Advisor page commit. Synchronize signals with a committed-page marker. Exercise checkpoint or publication failure, retained artifacts and matching revisions. Check JSON, Markdown, stdout and debug stderr for synthetic canaries in excluded fields, foreign targets, continuations and error bodies.

**Acceptance record** · create `verification.md` in this change folder during phase 4.

**Intent.** Make the operator’s live result and any limits of the available examples explicit.

**Contract.** Record commands, automated outcomes, sanitized live comparison and explicit operator acceptance or rejection. A rejection or unavailable live comparison leaves manual acceptance pending. Only the operator’s acceptance closes the slice. Keep subscription identifiers, raw service data, run files and credentials local.

### Success criteria

#### Automated verification

- Verify five-collector success, empty, partial, failure and synchronized interruption scenarios.
- Verify artifact identity, privacy and ignore rules and pass the complete repository check suite.

#### Manual verification

- Compare the authorized subscription’s completed Advisor list with the matching portal view and inspect the evidence.
- Record explicit operator acceptance or rejection and distinguish live examples from synthetic-only cases.

Check item 4.4 only after the operator explicitly accepts the result. If the operator rejects it, record the reason in `verification.md` and leave 4.4 pending.

## Testing Strategy

Use narrow package tests in phases 1–3 and integrated synthetic tests in phase 4. All application tests run without the operator’s Azure session or live ARM. Check exits 0 for five completed collectors, 3 for partial execution with useful results or another completed source, 1 for fatal execution or all failing without useful evidence, and 130 for cancellation. Validate that a failed source can never produce successful-empty wording. Keep existing invalid-input and subscription-denial behavior.

Run the repository checks from the project root. Use the pinned toolchain and writable temporary caches when needed. Confirm tests and vulnerability analysis cover actual Go packages. [Development tools](../../foundation/development-tools.md) · [CI](../../../.github/workflows/checks.yml)

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

### Manual acceptance procedure

1. With login established externally, run the built binary from this repository using `health check --subscription "<authorized-subscription-id-or-full-name>"`. Inspect the process exit, Advisor status and the published report beside its saved snapshot. Match run ID and revision.
2. In Azure Advisor, select the same subscription. Align category and workflow filters across the statuses being compared and note any timing difference. For a populated result, trace an instance from its row to saved evidence and the portal. Compare type/group membership, status, impact, assessed resource when available and last evaluation time where exposed. Record unavailable fields or cases as such.
3. For a completed empty list, confirm zero recommendations under comparable portal filters and review the populated synthetic report. Record that populated findings were verified synthetically. A denied, partial or mismatched list does not satisfy this empty-result case. Investigate any material discrepancy before acceptance without changing Azure to manufacture a scenario.
4. Confirm useful earlier-source evidence and coverage wording in the report. Check locally that the run artifacts are ignored. Record the sanitized result and the operator’s explicit decision.

## Performance and migration

Use a subscription list with sequential pagination and one local checkpoint per accepted page. There are no per-resource calls. Exercise hundreds of inventory resources and repeated Advisor types across multiple pages. Verify deterministic grouping without a timing SLA. Schema 3 applies to new runs at integration. Historical directories require no migration. A rollback affects future invocations and preserves existing evidence.

## References

- Product and slice ownership. [PRD](../../foundation/prd.md) · [Roadmap S-03 and S-04](../../foundation/roadmap.md#s-03-see-existing-advisor-recommendations)
- Prior operator acceptance and synthetic scenario limits. [S-01 verification](../../archive/2026-11-05-subscription-resource-health-report/verification.md) · [S-02 verification](../../archive/2026-11-06-active-alert-report/verification.md)
- ARM operation, sample continuation, resource shapes and status semantics. [Microsoft Advisor REST API](https://learn.microsoft.com/en-us/rest/api/advisor/recommendations/list?view=rest-advisor-2026-10-01)

## Progress

> Pending items use empty checkboxes. Check each item after verification. Append the closing commit SHA when a commit is authorized and lands. Keep step titles stable.

### Phase 1: Advisor evidence

#### Automated

- [x] 1.1 Verify scoped identity, structured facts, attribution and privacy with health package tests. — ec3de07
- [x] 1.2 Verify duplicate, conflict, unknown-field and stable per-instance finding behavior with health package tests. — ec3de07

### Phase 2: Read-only collection

#### Automated

- [x] 2.1 Verify the native Advisor pager against independent success, empty and malformed response fixtures. — ec3de07
- [x] 2.2 Verify the exact request boundary and retention of committed pages across continuation and source failures. — ec3de07

### Phase 3: Daily review in Markdown

#### Automated

- [x] 3.1 Verify fifth-collector wiring, initial state, query timing, exit status and committed-snapshot publication. — ec3de07
- [x] 3.2 Verify deterministic grouped rows, field labels, impact ordering and per-instance evidence links. — ec3de07
- [x] 3.3 Verify schema-1 and schema-2 golden output and legacy Resource Health finding visibility. — ec3de07

### Phase 4: Verification and operator acceptance

#### Automated

- [x] 4.1 Verify five-collector success, empty, partial, failure and synchronized interruption scenarios. — ec3de07
- [x] 4.2 Verify artifact identity, privacy and ignore rules and pass the complete repository check suite. — ec3de07

#### Manual

- [x] 4.3 Compare the authorized subscription’s completed Advisor list with the matching portal view and inspect the evidence. — ec3de07
- [x] 4.4 Record explicit operator acceptance or rejection and distinguish live examples from synthetic-only cases. — ec3de07

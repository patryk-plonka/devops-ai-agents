---
date: 2026-11-06
topic: S-02 active operational alerts
researcher: Codex
status: complete_for_planning
---

# S-02 research

## Task and accepted decisions

Extend the accepted S-01 report with Service Health and existing Azure Monitor metric/log alert instances. The seven operator decisions are recorded in [change.md](change.md). Scope is one selected public-Azure subscription, read-only, with 30-day queries, structured evidence, independent source failures, and no overall OK classification. Advisor and drafts remain later slices.

## Current state and integration points

| File | Finding and implication |
| --- | --- |
| `internal/health/model.go:42` | Source initialization names the later sources but marks them unimplemented in S-01. Update attempted-source initialization without implying complete overall coverage. |
| `internal/health/evidence.go:17` | Page inputs and evidence are shaped around inventory and Resource Health. Event identity, subscription provenance, alert condition/workflow, time-window metadata, and attribution certainty need explicit representation. |
| `internal/health/normalize.go:111` | Every non-inventory page currently takes the Resource Health branch. Introduce explicit dispatch before passing new source pages through it. |
| `internal/health/normalize.go:193` | Findings and observations derive from normalized evidence. Extend source-specific derivation while preserving Resource Health rules and evidence references. Resource enumeration must no longer use total evidence count as a proxy for observed resources. |
| `internal/health/check.go:10` | Collector arguments and success/partial rules assume exactly two collectors. Extend sequential execution and outcome calculation to all implemented sources. |
| `internal/azure/transport.go:23` | Request validation permits exact operations, versions and queries. Add operation-specific policies rather than allowing arbitrary subscription GET requests. |
| `internal/azure/paging.go:104` | Continuations pass through the request boundary; repetition keys normalize equivalent links. Initial request identity also needs to support the new operations and query policy. |
| `internal/azure/record_decode.go:47` | Record-level schema isolation only understands inventory and Resource Health. New adapters must preserve valid siblings and reject invalid envelopes. |
| `internal/cli/cli.go:29` | Add injectable collectors and production wiring; update shared test constructors before enabling defaults to prevent use of the operator's session. |
| `internal/report/markdown.go:64` | Report title, unchecked-source text, finding tables and evidence index assume S-01. Add operational evidence presentation without merging source condition with resource health. |
| `internal/runstore/store.go:71` | Publication reads committed JSON and verifies identity/revision. Preserve this mechanism; test new-field round trips rather than introducing another store. |

Shared test injection sites include `internal/cli/cli_test.go:40`, `internal/cli/integration_test.go:80`, and `cmd/devops-ai-agents/main_test.go:91`. Preserve the native fake-CLI/SDK transport path, actual output-byte privacy checks, and signal synchronization after a committed page.

## External-contract evidence

The table distinguishes documented service contracts from implementation decisions. Nothing here claims live verification of either new source.

| Boundary | Evidence | Contract and planning consequence |
| --- | --- | --- |
| Service Health listing | [REST operation and example](https://learn.microsoft.com/en-us/rest/api/resourcehealth/events/list-by-subscription-id?view=rest-resourcehealth-2025-05-01); installed `armresourcehealth v1.4.0` `events_client.go:104` | Use the existing SDK's subscription Events pager, GET `/subscriptions/{id}/providers/Microsoft.ResourceHealth/events`, API `2025-05-01`. |
| Service Health time query | Same REST operation; installed SDK options/request builder | `queryStartTime` uses documented `M/D/YYYY` examples and filters by last update. Compute the query date once; record it alongside the intended rolling cutoff. |
| Service Health continuation | Independent REST example | Same operation/version with `$skipToken`; the example omits the original query date. Do not require every continuation to repeat it. Any repeated date must match the original query. |
| Service Health identity | Independent REST example | Event IDs can be global `/providers/Microsoft.ResourceHealth/events/{id}` even though the query is subscription-scoped. Validate identity separately; do not permit global HTTP requests. |
| Service Health projection | Installed SDK `models.go:231`, `:411`, `:432`; `constants.go:102`, `:156` | Active/Resolved states; four selected categories among seven known categories. Impact supplies services/regions and scope lists, not affected-resource IDs. `eventSource=ResourceHealth` occurs in the official positive example. |
| Sensitive advisory metadata | [Microsoft advisory access documentation](https://learn.microsoft.com/en-us/azure/service-health/access-service-advisories-api); installed SDK event client | GET omits some sensitive details. Retain allowed metadata and explicit limits; exclude POST detail/impacted-resource calls. Missing optional sensitivity flags are valid. |
| Monitor list | [Alerts Get All REST reference](https://learn.microsoft.com/en-us/rest/api/alerts-management/alerts/alerts/get-all?view=rest-alerts-management-alerts-2019-03-01) | Subscription-scoped GET, API `2019-03-01`, `timeRange=30d`; request no context or egress configuration. Keep workflow response separate from monitor condition. |
| Monitor pagination | Same independent REST example | Uses `ctoken` and an explicit HTTPS port 443. Its sample also changes subscription/API version: preserve those as rejection cases, not permission to follow a changed scope or version. |
| Monitor interpretation | [Official alerts overview](https://learn.microsoft.com/en-us/azure/azure-monitor/alerts/alerts-overview) | Stateless alerts remain Fired; alert instances are retained for 30 days, and a stateful condition may persist beyond instance retention. Source-reported Fired is not proof of a current outage. |

The remaining Monitor SDK question was resolved by downloading the public `armalertsmanagement v0.11.0` module source from the Go module proxy into temporary storage on 2026-11-06. The GitHub connector required reauthentication and web retrieval of the tagged raw source failed; public module retrieval succeeded. Inspection of `alerts_client.go` confirmed that **GetAll itself uses `2025-05-25-preview`** and passes the complete scope through `url.PathEscape`. Its `go.mod` declares Go 1.24 and azcore 1.21. The earlier package-documentation result was not enough to establish the desired stable operation.

Decision: do not add that generated module or rewrite its requests in a transport. Implement the single documented stable `2019-03-01` operation with the existing `azcore v1.23.3` ARM client/pipeline and runtime pager, using a small private wire projection. Existing local `azcore/arm/client.go` and `azcore/runtime/pager.go` were read to verify `arm.NewClient`, `runtime.NewPager`, and `runtime.FetcherForNextLink`. This preserves SDK authentication, retries, disabled provider registration, the shared page loop, and the read boundary without another dependency. [Reviewed module source](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/alertsmanagement/armalertsmanagement@v0.11.0), [existing azcore ARM API](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore/arm@v1.23.3).

## Decisions derived from research

- Use the generated Service Health pager and a narrow Monitor REST adapter over the existing Azure SDK ARM pipeline/pager behind the read boundary; no new dependencies or production token, endpoint, or live-test flags.
- Keep independent sequential collection and existing operational bounds: page deadline, bounded SDK retries and page ceiling. A checkpoint failure stops further reads; source failure alone does not suppress other collectors.
- Service Health subscription-list membership establishes query provenance, not confirmed resource impact. Do not infer affected resources by joining service/region labels to inventory.
- Exclude unrelated nested subscriptions/tenants before persistence. Preserve uncertainty when region attribution is absent; conflicting scope facts require a sanitized gap.
- Generate prose from accepted structured facts. Exclude source titles/descriptions/articles, arbitrary links, log queries/results, usernames, and raw errors. Identity fields remain validated and accepted labels remain filtered before saving.
- Distinguish inactive evidence, known out-of-scope categories, and unknown/malformed facts. Do not turn an unknown state into an active finding or a healthy conclusion.
- Resolve duplicate observations only with adequate source identity and ordering evidence. Preserve contradictions as uncertainty rather than selecting whichever page arrived last.
- Source enumeration completion and coverage completeness are separate. Bounded windows, missing resource attribution, sensitive details and the unimplemented Advisor source must remain visible.
- A Monitor workspace target does not establish the scope of every resource referenced by its log query. Preserve the explicit selected-subscription target without fetching or interpreting query content or inferring underlying affected resources.

## Verification implications

S-01's [manual-check investigation](../../archive/2026-11-05-subscription-resource-health-report/manual-check-investigation.md) found that production and fixtures shared unverified external-schema assumptions. Each new boundary therefore needs an independently sourced literal positive fixture with its source/version and deliberate synthetic substitutions documented. Do not build every expected URL or payload from the implementation under test.

Test valid continuation forms as well as unsafe ones; mixed-validity pages; unknown fields/states; all selected categories; acknowledged/closed Fired alerts; selected-subscription attribution; privacy in actual JSON/Markdown/stderr bytes; independent failures; interruption after either new source commits a page; and final snapshot/report revision agreement.

The required final checks remain those in `AGENTS.md` and `.github/workflows/checks.yml`. No application tests have been run during this documentation-only planning work. Live acceptance remains a later operator activity in the agreed S-01 environment; absent alert/event examples must be recorded as not live-verified.

## Planning checkpoint

The operator approved the four-phase outline on 2026-11-06: (1) evidence/coverage contracts, (2) source adapters and read boundary, (3) CLI/report integration, (4) integrated verification and operator acceptance. Technical choices are settled; see `plan.md` for the execution contract. No implementation has started.

Service Health date precision is settled as follows: record a fixed UTC reference at run start and a rolling lower cutoff 30 days earlier, query the UTC calendar date containing that cutoff using the documented date syntax, and apply the exact lower cutoff locally to `lastUpdateTime`. Keep the actual API query date in provenance. Missing source times remain unclassified evidence with a gap; events before the exact cutoff are excluded from active findings. The listing has no upper-bound parameter, so report collection times and do not imply an atomic snapshot. Monitor uses its documented relative `timeRange=30d` directly; record request/collection timing without inventing an undocumented timestamp-field filter.

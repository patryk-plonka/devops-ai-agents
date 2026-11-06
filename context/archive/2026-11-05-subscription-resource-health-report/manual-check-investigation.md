# Why manual acceptance required fixes

Date: 2026-11-06. Scope: the failures found during S-01's private-subscription
acceptance, after the automated phase checks and implementation reviews passed.

The two application bugs shared a cause: production code and synthetic fixtures
encoded the same unverified assumptions about external data. Passing tests
established consistency with those fixtures, while compatibility with actual
CLI output and service-generated pagination remained unproven. The plan required
live acceptance, which caught the failures before the slice was marked complete.
It did not require independent evidence for the external shapes used in fixtures.

This investigation uses the code, recorded reproductions, research, plan, reviews,
and installed dependency source. No new live requests were made. The repository
has no commits, so causal conclusions use the saved records rather than a complete
historical diff. The artifacts do not establish why an individual chose a field
name or query spelling; those choices are distinguished from the observed gaps.

## What failed

| Incident | Actual cause | Effect | Resolution |
| --- | --- | --- | --- |
| Name selection rejected a valid public-cloud user account | The application decoded `environmentName`; `az account list` emits `cloudName`. | Exit 2 before live access validation, with a failed diagnostic report. | Correct the JSON tag and CLI fixture; add a literal external-schema regression that exercises discovery followed by ID/name matching. |
| Resource Health stopped after its first page | The allowlist admitted `$skipToken` but the service returned lowercase `$skiptoken` plus `$top`. | Exit 3 with retained evidence and an explicit `unsafe_continuation` gap. | Accept the verified paging forms with strict parameter validation and token-alias normalization; update full-flow fixtures and boundary regressions. |
| ID selection failed while name selection worked | The supplied UUID was the tenant ID, rather than the selected subscription ID. | Correct missing-subscription feedback; this was separate from the parser bug. | Extract the subscription ID from the CLI and repeat the check. |
| Instructions printed a table with no ID column | The suggested query named the column `id`, which Azure CLI's table formatter suppresses. | The instructions failed to expose the value needed to resolve the ID confusion. | Use `SubscriptionID:id` as the table label or extract `.id` as TSV. |

The last two items concern operator guidance and identifier interpretation. They
should not be counted as two more application parser/pager defects. The earlier
phase 4 F1 privacy-assertion observation was found during review and is also
separate from these manual compatibility failures.

## Why tests missed the account schema

The original discovery test generated its response by serializing the production
`Account` type. That makes the producer and consumer share the same JSON tags:
the wrong tag still produces a response the decoder understands. The test also
checked command arguments and account count without exercising the subsequent
public-cloud selection decision. Matching tests constructed Go accounts directly,
bypassing JSON decoding. The full-flow shell fixture repeated `environmentName`.

Evidence remains visible in
[`TestDiscoverUsesOnlyCachedReadCommands`](../../../internal/azure/session_test.go)
(`json.Marshal([]Account{...})`) and the direct construction in `TestMatches`.
Those are useful focused tests, but neither independently establishes the command's
JSON contract. The new `TestDiscoverDecodesAzureCLIAccountSchema` uses literal
`cloudName` JSON independent of the production tags and checks discovery followed
by matching. It failed before the correction and passes afterward.

The installed Azure CLI 2.91.0 source explains the distinction: its core profile
uses `environmentName`, while
`azure/cli/command_modules/profile/custom.py:list_subscriptions` explicitly renames
that property to `cloudName` before returning command output. The recorded
[research](research.md#authentication-and-subscription-resolution) cites core
`_profile.py`, but does not record this command-handler transformation. Checking
the exact command output layer was possible without a live subscription.

## Why tests missed valid pagination

The synthetic `listURL` helper built Resource Health links with `$skipToken` and
no `$top`. Both the fake response's next link and expected request came from that
helper. The original full-flow fixture used the same shape. Tests thoroughly
exercised failures and attacks around an assumed valid base URL, while omitting
the legitimate positive case later returned by Azure.

See [`collection_test.go`](../../../internal/azure/collection_test.go), especially
`listURL` and `TestNativePagersCheckpointAndWhitelist`. The new independent
`TestResourceHealthServiceContinuationShape` in
[`transport_test.go`](../../../internal/azure/transport_test.go) covers the observed
shape, aliases, encoded keys, duplicates, invalid page sizes, and scope/host/API
boundaries. The full CLI fixture now exercises lowercase pagination and page size.

Using a native SDK pager did not validate our additional URL allowlist. In the
selected `armresourcehealth v1.4.0`, `NewListBySubscriptionIDPager` takes the
service's `page.NextLink` and passes it to `runtime.FetcherForNextLink`. The SDK's
initial request builder does not enumerate every query parameter the service can
return in a next link. Native SDK execution against our own fake response proved
the pager mechanics, not completeness of the fake service contract.

## What the plan and reviews missed

| Layer | What was specified or checked | Missing evidence that mattered |
| --- | --- | --- |
| Research | Cached CLI discovery, explicit subscription binding, native API versions, `nextLink`, read-only constraints. | Exact account command output versus internal profile representation; service continuation examples covering benign paging parameters. |
| Phase 1 plan | Read CLI metadata, require AzureCloud/user accounts, resolve ID/name safely. | A positive raw-JSON fixture sourced independently of the application's model and passed through discovery plus matching. |
| Phase 2 plan | Native SDK pagers, strict continuation checks, retries and retained pages. | An evidence-backed query allowlist distinguishing pagination parameters from filters or expansions; legitimate variants as positive tests. |
| Phase 4 plan | Real command/orchestration/storage/report path over fake CLI/ARM; representative scale and adverse cases. | Fixture provenance and a requirement to challenge whether the fake producer matches the real producer. More records did not add a new external shape. |
| Implementation reviews | Plan adherence, safety, architecture, patterns, green checks, explicit pending live acceptance. | An independent check of field names and valid continuation shapes against the producing command/service. The reviews accepted internally consistent fixtures. |

Relevant [plan](plan.md) sections are Phase 1's Azure CLI discovery contract,
Phase 2's adapter/read-boundary contract, Phase 4's full-flow integration contract,
and the Testing Strategy. The [phase 4 review](reviews/impl-review-phase-4.md)
approved the synthetic implementation and explicitly left live acceptance pending.

The plan did not prescribe the wrong field name or forbid valid page-size
parameters. Those were implementation assumptions that became test expectations.
The planning gap was the absence of a concrete check for the evidence supporting
those assumptions. The research itself already said tests should avoid merely
mirroring implementation details; the phase criteria did not turn that principle
into a verifiable requirement for external fixtures.

Live access was intentionally supplied later. It would be inaccurate to claim
earlier live testing was available and simply skipped. The avoidable part was
treating synthetic success as sufficient support for the fixture shapes. When
live access is unavailable, that uncertainty should remain explicit in research,
test descriptions, and review conclusions until an external check resolves it.

## Why the guidance needed correction

Both tenant IDs and subscription IDs have UUID syntax. A UUID-format test cannot
establish that the value belongs to the requested scope. The resolver correctly
refused an ID absent from the subscription list, but the instructions should have
made the source of the subscription ID explicit.

The installed CLI's Knack table formatter defines
`_TableOutput.SKIP_KEYS = ['id', 'type', 'etag']`. Therefore the original suggested
query `{name:name,id:id}` hid the useful ID column. Correct template:

```sh
az account list --all --query "[].{Name:name,SubscriptionID:id,TenantID:tenantId}" --output table
```

This is a documentation defect in the manual-check instructions. The core
resolver does not need to accept tenant IDs as subscriptions.

## Improvements for future work

1. Record an external-contract evidence table during research: consumed command
   or operation, relevant output fields/query parameters, version, evidence
   source, and whether each claim is observed, documented, or inferred.
2. Require at least one independently sourced positive fixture at each external
   boundary. Do not establish wire compatibility solely by serializing the type
   under test or generating both actual and expected URLs with the same helper.
3. Exercise legitimate service variations alongside rejection tests. Preserve
   fixed host, subscription, operation, API version, and method restrictions;
   add only justified pagination forms with their own validation.
4. Review fixture provenance as well as code and passing checks. Ask which
   assertion would fail if the external producer used a different field name or
   a valid additional pagination parameter.
5. Run a narrow read-only smoke check when authorized access becomes available.
   If access remains unavailable, carry the compatibility uncertainty forward
   explicitly. Keep CI synthetic and keep real identifiers/tokens out of fixtures.
6. Test operator instructions as written, including table output and exit-code
   inspection. Label subscription and tenant identifiers distinctly.

These changes target the shared cause: independent verification of external
contracts. They are recommendations from this investigation; the recurring
lesson entry remains subject to the invoked skill's user-authored-field and
confirmation steps.

## Outcome and limits

Both application fixes reproduced their failures in focused regressions before
the correction. Updated regressions, the complete uncached race suite, and all
required repository checks passed. Subsequent live runs by name and resolved
subscription ID completed with exit 0 and matching saved revisions. The operator
then confirmed all manual tests; S-01 remains implemented with 18/18 Progress rows.

The conservative failure behavior worked: denied or unresolved scope stopped
collection, rejected pagination retained prior evidence, and incomplete coverage
remained visible. The fixes preserved those protections. Guest access and adverse
scenarios not explicitly exercised live retain the limits recorded in
[verification.md](verification.md).

This investigation changes documentation only. It neither changes the approved
phase blocks nor adds production flags, Azure writes, live evidence to source
control, or a commit. The recurring lesson has not yet been appended.

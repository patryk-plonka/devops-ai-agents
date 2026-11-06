---
topic: subscription-resource-health-report
researcher: Codex
created: 2026-11-05
updated: 2026-11-05
status: complete
---

# Subscription Resource Health Report Research

## Scope and baseline

This research supports roadmap S-01, not the complete health-agent MVP. The repository has `go.mod`, installed-tool documentation, and planning artifacts; it has no application source, tests, Azure integration, CI workflow, or existing implementation conventions to reuse. No live Azure account was accessed during research.

The slice must select one subscription by ID or name, resolve ambiguous names interactively, validate access, and produce a Resource Health Markdown report with evidence and coverage gaps. Advisor, Service Health, and Monitor remain explicitly not checked. A source-specific Available observation must never become an overall OK classification.

## Code and document references

- `context/foundation/roadmap.md:67` — S-01 scope and first executable flow.
- `context/foundation/roadmap.md:59` — introduce minimum authentication and automated verification within S-01.
- `context/foundation/roadmap.md:139` — live acceptance is necessary; simulated results alone do not complete the slice.
- `context/foundation/prd.md:40` — read-only, evidence, privacy, missing-evidence, and partial-failure guardrails.
- `context/foundation/prd.md:94` — current-state reporting and complete-applicable-coverage rule.
- `context/foundation/tech-stack.md:23` — local Go CLI, GitHub Actions, manually triggered releases, no required LLM or server.
- `AGENTS.md:5` — critical behavior and privacy constraints.
- `AGENTS.md:9` — required Go verification; empty package sets are not passing validation.
- `go.mod:1` — module identity; Go 1.27.3 is already selected.
- `context/foundation/development-tools.md` — installed tools and local checks.
- `context/changes/bootstrap-verification/setup-check.md` — historical bootstrap evidence and sandbox cache handling.

## Confirmed operator decisions

| Area | Decision | Origin |
|---|---|---|
| Identity | Operator signs in with `az login`; client subscriptions use guest access. | Planning answer 1 |
| Cloud | Public Azure only. | Planning answer 2 |
| Output | A Git-ignored directory within this repository, organized for traceability. | Planning answer 3 |
| Audience | Operator alone. | Planning answer 4 |
| Identifiers and tags | Keep needed subscription/tenant/resource identifiers and resource tags; exclude secrets and unrelated personal details. | Answer 5 and follow-up |
| Interruption | Save evidence already collected and an incomplete report. | Planning answer 6 |
| Live validation | Operator's private subscription, supplied later. | Planning answer 7 |

The private subscription is an execution prerequisite, not an unresolved design choice. Its ID, permissions, resource mix, and live behavior have not been verified. A private subscription might not exercise guest-tenant authentication; that path also needs controlled test coverage and must not be described as live-verified without evidence.

## Authentication and subscription resolution

Use explicit `AzureCLICredential`, not a credential chain that might silently select another identity. Discover already logged-in subscriptions through `az account list --all --output json` using argument-based process execution. Require the active cloud to be AzureCloud, retain only real public-Azure user-account subscriptions, and omit unrelated account-profile fields from persisted data. The CLI's profile is discovery input, not proof of present Azure authorization. [Azure CLI account commands](https://learn.microsoft.com/en-us/cli/azure/account?view=azure-cli-latest)

Resolve IDs exactly and names case-insensitively. Distinct matching IDs require a prompt showing identifying scope information. Noninteractive ambiguity must fail with instructions to supply an ID. Do not run `az account set`, perform automatic sign-in, or refresh the CLI account cache. Missing cached subscriptions should be described as unavailable in this CLI session, not as a proven Azure denial. [Azure CLI profile implementation](https://raw.githubusercontent.com/Azure/azure-cli/dev/src/azure-cli-core/azure/cli/core/_profile.py)

Bind `AzureCLICredentialOptions.Subscription` to the resolved UUID and leave `TenantID` unset. The SDK can emit both arguments, but the CLI rejects a token request that supplies subscription and tenant together. Subscription binding selects the associated CLI account and tenant. If authentication requires renewed login or a claims challenge, fail with sanitized guidance; do not switch identities or print raw credential errors. This combination constraint is inferred from the two implementations. [Go Azure CLI credential](https://raw.githubusercontent.com/Azure/azure-sdk-for-go/main/sdk/azidentity/azure_cli_credential.go), [Azure CLI profile implementation](https://raw.githubusercontent.com/Azure/azure-cli/dev/src/azure-cli-core/azure/cli/core/_profile.py)

After resolution, perform a live subscription Get. A 403 is explicit denied access; authentication failure, missing cached scope, and not-found/inaccessible scope are distinct errors. If this validation fails, do not proceed with collection against another subscription.

## Read-only enforcement

Every ARM client must set `DisableRPRegistration: true`: auto-registration is enabled by default and can introduce a write even when application code only calls read operations. This is a required configuration, not a permission repair opportunity. [ARM client policy options](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azcore/arm/policy)

Use narrow read adapters and a transport boundary that permits only intended GET operations against public ARM and the resolved subscription. Validate continuation URLs and returned resource IDs; reject cross-subscription or off-host results without fetching or persisting their content. Do not allow redirects to bypass that boundary. This restriction covers ARM collection; token acquisition remains the responsibility of Azure CLI.

## Collection and coverage

Resource Health support varies by resource type. Independently list ARM resource inventory so missing observations can be identified. An absent health record is unknown/missing evidence, not proof of unsupported status or health. Explicit unsupported errors may be recorded as unsupported; do not invent a complete support catalog. Independent inventory is a design inference from the separate inventory and Resource Health contracts. [Resource Health FAQ](https://learn.microsoft.com/en-us/azure/service-health/resource-health-faq), [ARM resource inventory](https://learn.microsoft.com/en-us/rest/api/resources/resources/list?view=rest-resources-2021-04-01)

Inventory and Resource Health collection should remain independent after subscription validation. If inventory fails, keep returned health observations and mark inventory coverage unverified. If health collection fails, keep inventory and mark the affected resources not checked. Preserve successful pages after later-page errors. Retain health observations for resources absent from inventory with an explicit attribution/coverage limitation rather than silently discarding them.

Preserve Available, Unavailable, Degraded, and Unknown as source observations. Unknown is not proof of an outage, and Available is not an overall all-source OK result. Keep collection time separate from source-reported observation time. [Resource Health overview](https://learn.microsoft.com/en-us/azure/service-health/resource-health-overview)

The API schema defines a null `availabilityState` as an availability-impacting event, not an Available state. Preserve its evidence and any event identity without inventing a current Unavailable classification. Missing properties and unrecognized states remain undetermined. Pagination uses `nextLink`; partial results remain useful when later requests fail. [Resource Health subscription listing and schema](https://learn.microsoft.com/en-us/rest/api/resourcehealth/availability-statuses/list-by-subscription-id?view=rest-resourcehealth-2025-05-01)

Use bounded SDK retries plus a finite per-page operation deadline and cancellation. Detect repeated continuation URLs and impose a generous page ceiling; hitting a bound yields incomplete collection, never a clean empty result. These are implementation safeguards, not newly imposed product latency or retention targets. [Azure SDK retry contracts](https://raw.githubusercontent.com/Azure/azure-sdk-for-go/main/sdk/azcore/policy/policy.go)

## Verified dependency candidates

The following published modules and operation families were checked. They are concrete candidates, not a claim that each is the newest available release; implementation must record selected versions in `go.mod` and `go.sum` and run the repository's vulnerability checks.

| Module under `github.com/Azure/azure-sdk-for-go/` | Verified version | Use |
|---|---|---|
| `sdk/azidentity` | v1.15.0 | Explicit Azure CLI credential |
| `sdk/resourcemanager/resources/armsubscriptions/v2` | v2.0.0 | Subscription Get, native API 2022-12-01 |
| `sdk/resourcemanager/resources/armresources/v4` | v4.0.0 | Inventory list pager, native API 2025-04-01 |
| `sdk/resourcemanager/resourcehealth/armresourcehealth` | v1.4.0 | Availability-status subscription pager, native API 2025-05-01 |

Sources: [Identity](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/azidentity), [Subscriptions](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions/v2), [Resources](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources/v4), [Resource Health](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resourcehealth/armresourcehealth).

## Proposed local run records

Use a simple local layout, with exact command contracts to be captured in the plan:

```text
runs/                            # ignored in its entirety by /.gitignore
  <UTC-timestamp>-<random-id>/
    run.json                     # normalized metadata, evidence, coverage, diagnostics
    report.md                    # Markdown derived from the same saved revision
```

Keep the layout's documentation in the tracked root README. Use generated run IDs instead of Azure names in filesystem paths, avoid overwriting previous runs, and keep one directory per attempted run. `run.json` should contain schema version, run ID, revision, UTC timestamps, selected scope, execution outcome, source coverage, evidence identifiers, and sanitized diagnostics. A separate raw-response archive is unnecessary.

Execution outcome and report coverage are separate: Resource Health collection may finish successfully while the overall report remains incomplete because three agreed sources are not yet implemented.

Filter before storage and rendering. Retain needed identifiers and resource tags per the user's decision. Remove recognized credential-bearing tag keys/values, exclude tokens, headers, raw errors, unrelated account fields, and arbitrary resource properties, and record redaction without reproducing removed content. No claim is made that arbitrary free text can be automatically certified free of every possible secret. Render allowed resource names/tags as escaped text, never executable instructions or untrusted Markdown/HTML.

Create writable storage before collection and checkpoint after completed pages. Write snapshots and reports through same-directory temporary files and atomic replacement; include matching run IDs and revision numbers so a stale report is detectable. A failed write must not destroy the previous checkpoint or produce a false saved/success message. Git exclusion prevents ordinary addition of untracked run data, but does not untrack data already committed. [Git ignore semantics](https://git-scm.com/docs/gitignore)

The first Ctrl+C cancels collection and finalizes gathered evidence as cancelled/incomplete. Local finalization must not reuse the cancelled Azure-operation context. A second forced interruption or process kill may prevent finalization, so retain the last checkpoint. Return a nonzero cancellation result, conventionally 130. [Go signal handling](https://pkg.go.dev/os/signal#NotifyContext)

## Architecture and verification

Use a small entry point in `cmd/devops-ai-agents/` and internal packages for CLI interaction, Azure adapters, health orchestration/models, Markdown rendering, and run storage. Introduce interfaces only at external boundaries needed for testing. No database, service host, plugin registry, LLM, or generic multi-agent framework is required.

Automated checks use synthetic data and fake Azure transports/CLI processes. They must exercise ambiguous names, guest subscription binding, permission failure, partial paging, unsupported/missing/null health observations, wrong-scope results, tags and secret filtering, Markdown escaping, cancellation after collected evidence, and storage failures. Test report semantics and evidence references rather than only mirroring implementation details.

Add GitHub Actions checks with the first source packages and extend them as behavior grows. Use repository-required build, race-enabled tests, vet, staticcheck, govulncheck, formatting, and workflow checks. CI must not require Azure credentials or upload live reports. Configuring a release workflow is not necessary for this slice.

Live acceptance uses the operator's private subscription after they supply it and sign in. Compare scope and representative observed health evidence against Azure, inspect local artifacts and Git exclusion, and exercise graceful interruption. Record what was actually verified; use controlled fixtures for adverse cases unavailable live rather than changing resources to manufacture failures.

## Approved phase breakdown

1. **Subscription access and run setup** — executable CLI, login-session discovery, ID/name resolution, safe access failures, local run initialization, and first CI checks.
2. **Evidence collection and coverage** — independent inventory and Resource Health paging, read-only boundary, filtered evidence checkpoints, and partial-failure accounting.
3. **Markdown reports and interruption** — readable findings and source-specific observations, explicit coverage gaps, traceable local artifacts, and Ctrl+C finalization.
4. **Complete-flow validation** — adversarial integration tests, operator documentation, required tool checks, and manual acceptance using the private subscription.

The user approved this phase breakdown on 2026-11-05. See `plan.md` for the finalized contracts and verification steps and `plan-brief.md` for the handoff. No application implementation or live validation has been performed.

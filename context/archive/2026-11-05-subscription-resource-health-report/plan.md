# Subscription Resource Health Report Implementation Plan

## Overview

Implement roadmap S-01: a manually invoked Go CLI that uses the operator's Azure CLI login to select one public-Azure subscription, collect Resource Health evidence, and save an explicitly incomplete Markdown report. Preserve evidence and coverage gaps when collection fails or the operator interrupts the run.

The operator approved the four phases below on 2026-11-05. The seven planning answers and identifier/tag clarification are recorded in [change.md](change.md); technical grounding is in [research.md](research.md). This plan authorizes no implementation by itself.

## Current State Analysis

Only the Go module and planning documents exist. There is no executable, Azure adapter, test suite, CI workflow, root README, or .gitignore. The local toolchain is installed; Azure access has not been verified. Preserve the existing module identity and Go version.

The current PRD supersedes older shaping answers. S-01 advances FR-001 through FR-004; the complete four-source report belongs to S-04 and drafting belongs to S-05. The operator will supply a private subscription for live acceptance. That input is required for the manual checks, not for developing or testing with synthetic evidence.

## Desired End State

From this repository, the operator signs in separately with az login and runs:

~~~sh
go run ./cmd/devops-ai-agents health check --subscription "<subscription-id-or-name>"
~~~

A built binary supports the identical arguments. The CLI locates the repository root by walking upward to the go.mod with this project's module identity; running outside that tree fails with an actionable message. Keep the output root fixed at the repository's runs/ directory for this slice.

Each accepted check invocation creates a unique local run directory:

~~~text
runs/
  <UTC-timestamp>-<random-id>/
    run.json
    report.md
~~~

The Markdown report identifies scope, run time, execution outcome, Resource Health observations, source findings, evidence references, and coverage gaps. Advisor, Service Health, and Azure Monitor are explicitly not checked. No resource receives an overall OK classification in S-01.

The operator can trace each finding to a normalized evidence record in run.json. Identifiers and tags are retained subject to secret filtering. The first Ctrl+C stops Azure collection and saves a cancelled/incomplete report containing the evidence already gathered. No Azure changes or ticket submission occur.

### Key Discoveries

- S-01 deliberately delivers an incomplete report: context/foundation/roadmap.md:79.
- Evidence, privacy, and read-only behavior apply from the first executable flow: AGENTS.md:7.
- The CLI session contains discovery metadata; cached membership does not establish current access. Verify the resolved subscription live.
- Explicit subscription binding avoids selecting the CLI's default subscription accidentally. Do not also set the credential's TenantID; the combined token arguments are rejected by Azure CLI.
- Disable automatic ARM provider registration on every client. Otherwise a read operation can trigger a registration write.
- Resource Health results do not establish a complete resource inventory. Collect inventory independently and represent missing observations as gaps.
- Null availabilityState can represent an impacting event; it is not a healthy default. SDK/API references and the reasoning behind these constraints are documented in research.md.

## What We're NOT Doing

- Advisor, Service Health, or Azure Monitor collection; overall OK classification; ticket drafts.
- Baseline comparison, custom proactive rules, scheduling, multiple-subscription runs, or concurrent runs.
- Sovereign clouds, service-principal/managed-identity authentication, token input, automatic login, or CLI account switching.
- Azure mutations, provider registration, automatic remediation, or changes to permissions.
- A database, hosted service, web UI, LLM, generic agent framework, raw HTTP archive, or release workflow.
- Automatic report deletion, a retention policy, signed/tamper-proof audit records, or crash/power-loss durability guarantees.

## Implementation Approach

Use the standard library for CLI parsing, JSON, filesystem operations, and Markdown generation. Use Azure's Go identity and ARM clients for authentication and collection. Keep six small areas: cmd/devops-ai-agents for process lifecycle, internal/cli for interaction, internal/azure for external adapters, internal/health for the normalized model and orchestration, internal/runstore for local persistence, and internal/report for Markdown. Interfaces belong at Azure, process, and storage boundaries that tests need to control; do not introduce a plugin registry.

Build the command incrementally across the approved phases. Phase 1 exposes scope validation and a preliminary run record; phase 2 adds collection and evidence; phase 3 completes the report and graceful interruption. Interim versions must explicitly state which capability is not implemented and must never masquerade as a completed health check.

### Command and result contract

The command is health check with required --subscription and optional --non-interactive. Standard help performs no login, network request, or run creation. Reject unknown flags and extra positional arguments. Use exact UUID matching or case-insensitive full-name matching, not substring selection. Prompt only on terminal input when multiple distinct subscription IDs match; show names, IDs, and tenant IDs in a stable order. EOF/cancel ends selection without choosing a default. --non-interactive or a nonterminal input fails ambiguity with instructions to supply an ID.

Print progress, prompts, and sanitized diagnostics to stderr. On finalized output, stdout contains the saved report path; the terminal message also states the execution outcome and that overall coverage is incomplete. Do not stream raw SDK responses or tokens. Usage errors before a valid check invocation do not create run data; later scope/access failures create a failed run record and, once phase 3 is present, a diagnostic report when storage is available.

| Exit | Meaning |
|---|---|
| 0 | Both intended collectors completed and their output was saved; overall report coverage still remains incomplete in S-01. Resource Health issues/Unknown observations do not by themselves change this exit code. |
| 1 | Pre-collection authentication/scope-access failure, storage or fatal execution failure; also both collectors failing without useful collected results. |
| 2 | Invalid input, missing/ambiguous CLI-session subscription, unsupported cloud/account, or missing repository context. |
| 3 | Partial collection: a collector/page failed or hit a bound while useful results or a completed collector remain. |
| 130 | Operator interruption; saved output is cancelled/incomplete when finalization succeeds. Any finalization failure is explicitly reported too. |

Execution outcome and coverage are independent fields. After subscription validation, collector-local authentication/authorization errors follow the partial-result rule: return 3 if the other collector completed or useful evidence remains, otherwise 1. Storage failure overrides a completed/partial result with exit 1; explicit interruption remains exit 130 with any storage failure also disclosed. Keep diagnostic categories explicit even when they share an exit code. Missing coverage alone is not evidence of resource unavailability.

### Data and evidence contract

run.json is a versioned normalized snapshot, not a serialized SDK object. Schema version 1 contains:

- Run ID, monotonically increasing revision, tool/build version when available, UTC start/update/end times, and execution outcome (in_progress, completed, partial, cancelled, or failed).
- Requested scope after safe text filtering and resolved subscription ID/name, tenant ID, and public-cloud identity when available. Scope may be unresolved on early failure.
- Inventory and Resource Health collection statuses, page counts, sanitized diagnostic categories, and the three deliberately not-checked sources.
- Resource inventory with IDs, names, types, resource groups/locations when returned, and retained filtered tags.
- Evidence records with stable IDs within the run, source/operation and API version, resource/status/event IDs where available, collection time, source-reported time when present, and the structured observation used by the finding.
- Findings and coverage gaps referencing those evidence IDs. No finding can exist without supporting evidence; diagnostic failures are coverage gaps, not invented resource faults.

Use case-insensitive canonical ARM IDs for joins while preserving the permitted source spelling for traceability. Derive a resource ID from an availability-status ID only by parsing its documented trailing ResourceHealth segment. Validate subscription boundaries before accepting any record. Reject malformed or wrong-scope records, retain other valid records, and record a coverage gap without saving the rejected payload.

Retain distinct observations rather than silently overwriting conflicting evidence. Collapse exact duplicates; conflicting states without an unambiguous current observation remain undetermined and link to all relevant evidence. Null state/event-only entries and missing/unrecognized properties cannot establish Available. Keep source Unknown separate from missing evidence. Do not infer unsupported status from absence alone.

Inventory pagination completion describes that API's enumeration, not a guarantee of universal child-resource coverage. Report the observed inventory scope and keep same-subscription health observations absent from inventory with an explicit attribution limitation.

### Privacy and presentation contract

Retain the user-approved identifiers and tags. Filter before persistence, diagnostic output, and Markdown rendering. Exclude account usernames, tokens, credentials, raw error bodies, request headers, arbitrary resource properties, and arbitrary source descriptions. Generate finding titles from known source states; evidence IDs and structured source facts provide traceability without saving entire payloads.

Keep ordinary tag keys/values. Redact values for normalized credential-like keys (password, secret, client secret, access/refresh token, authorization, API/account key, connection string, private key, SAS) and recognizable credential values such as bearer tokens, JWTs, private-key blocks, credential-bearing URLs, SAS signatures, and connection strings containing keys. Apply the same value filtering to other accepted text. Use a redaction marker and count, not the removed content. This is a defined filtering contract, not a claim that arbitrary text can be certified free of every secret; the operator has stated their tags contain no private information.

Encode allowed strings as literal text in Markdown and strip terminal/control sequences. Resource names and tags must not introduce HTML, links, table rows, headings, or terminal commands. Any clickable Azure reference is constructed from a validated resource ID, not an arbitrary URL supplied by Azure data.

## Critical Implementation Details

### Authentication and read-only boundary

Resolve the subscription before constructing AzureCLICredential with Subscription set to its UUID and TenantID unset. Credential requests that demand another tenant or fresh claims fail with login guidance; do not broaden authentication or fall back to another identity. Set DisableRPRegistration on all ARM clients and constrain their HTTP transport to the approved GET routes, public ARM host, and selected subscription, including continuation requests and redirects.

### Cancellation and persistence ordering

Initialize local storage before Azure work, then checkpoint filtered evidence after each successful page. On interruption, cancel collection but finalize storage/reporting independently of the cancelled request context. Commit run.json before publishing report.md for that revision; both carry run ID and revision so a stale report is detectable. A failed write preserves the previous committed snapshot and must not produce a saved/success message.

## Phase 1: Subscription access and run setup

### Overview

Create the executable, validate one subscription from the operator's existing CLI session, initialize a local run record, and introduce automated checks with the first source packages.

### Changes Required

#### 1. CLI entry and scope interaction

**Files**: cmd/devops-ai-agents/main.go; internal/cli/cli.go; internal/cli/cli_test.go.

**Intent**: Give the operator the agreed command, usable scope selection, and honest failure feedback.

**Contract**: Implement the command/exit contract above with injectable streams and external adapters. Return the final exit code to the process entry so cleanup runs before os.Exit. While later phases are absent, successful access validation remains a preliminary, incomplete check rather than a success claim for Resource Health.

#### 2. Azure CLI discovery, credentials, and live scope validation

**Files**: internal/azure/session.go; internal/azure/subscription.go; internal/azure/transport.go; corresponding tests; go.mod; go.sum.

**Intent**: Bind all reads to the requested subscription and the operator's existing user login.

**Contract**: Read active cloud and account metadata through argument-based Azure CLI invocations; do not invoke a shell with user input. Require AzureCloud and a user account, exclude tenant-only placeholders, and distinguish missing cache entries from denied live access. Resolve names from the existing account list without automatic refresh or account switching. Validate the resolved subscription with live Get and reject mismatched response identity. Do not persist the full CLI profile or child-process stderr.

Use the verified azidentity v1.15.0 and armsubscriptions/v2 v2.0.0 as initial module selections, with azcore resolved compatibly. Add no independent authentication cache. Record exact selected dependencies and checksums; if a vulnerability check requires a fixed compatible release, update that dependency and rerun the affected contract tests. Verify AzureCLI Subscription binding with fake CLI processes, including a different default subscription and guest tenant. No token appears in user output.

Enforce a GET-only public ARM transport for subscription Get now and extend its route allowlist in phase 2. Disable automatic provider registration. Reject redirects rather than allowing them to escape validation.

#### 3. Local run initialization

**Files**: internal/health/model.go; internal/runstore/store.go; internal/runstore/store_test.go; .gitignore; README.md.

**Intent**: Establish the traceable run artifact before collecting client data.

**Contract**: Locate the repository root, generate a collision-resistant UTC timestamp/run ID, create runs/<run-id>/ with owner-only directory/file permissions where supported, and write a minimal schema-versioned run.json. Add the root-anchored /runs/ ignore rule before live use. Paths must not derive from subscription/resource names or follow symlinks outside the intended run root. Never overwrite another run. Invalid storage prevents Azure requests. No automatic deletion or rotation.

The initial snapshot starts in_progress; scope failures finalize it as failed with sanitized diagnostics and scope unresolved when appropriate. Atomic same-directory temporary writes preserve any prior snapshot. Do not create a placeholder report claiming evidence was collected.

#### 4. First automated checks

**Files**: .github/workflows/checks.yml; scripts/check-format.sh; README.md; context/foundation/development-tools.md.

**Intent**: Run the repository's required checks from the first real packages onward.

**Contract**: GitHub Actions runs on pull requests and pushes to main with read-only repository permissions and no Azure credentials. Use the go.mod toolchain version, supported pinned checkout/setup-go actions, and explicit compatible versions of staticcheck, govulncheck, and actionlint. Record tool pins in development documentation and keep local commands aligned. The formatting script scans project Go source directories regardless of Git tracking state, excludes generated run data, and fails on gofmt differences without rewriting files; validate its shell with ShellCheck. Workflow checks must run actionlint. No release/upload step and no live run artifacts in CI.

### Success Criteria

#### Automated Verification

- CLI and scope tests pass: go test ./internal/cli ./internal/azure ./internal/runstore
- Build, race tests, and vet pass on real packages: go build ./...; go test -race -cover ./...; go vet ./...
- Static analysis, vulnerability, formatting, and workflow checks pass: staticcheck ./...; govulncheck ./...; ./scripts/check-format.sh; shellcheck scripts/check-format.sh; actionlint

No manual gate is needed in this phase. Real subscription acceptance is consolidated in phase 4; automated tests cannot mark that acceptance complete.

## Phase 2: Evidence collection and coverage

### Overview

Collect inventory and Resource Health with bounded pagination, enforce the subscription/read boundary, and checkpoint filtered observations with explicit gaps.

### Changes Required

#### 1. Inventory and Resource Health adapters

**Files**: internal/azure/inventory.go; internal/azure/resourcehealth.go; internal/azure/transport.go; corresponding tests and synthetic testdata; go.mod; go.sum.

**Intent**: Fetch the evidence required to show observed health and reveal missing coverage.

**Contract**: Add armresources/v4 v4.0.0 and armresourcehealth v1.4.0, using their native API versions and list pagers. Allow only subscription Get, subscription resource inventory, and subscription availability-status GET routes, including pagination for the same operation. Use no filters that silently omit healthy/unknown resources; do not expand raw recommended-action prose or poll individual resources speculatively.

Both sources run sequentially and independently after successful scope validation. A failed inventory request does not suppress health collection, and vice versa. A cancellation stops both. Each valid page is normalized and checkpointed before another page is requested. A checkpoint failure stops collection with a storage failure.

Use a 60-second deadline per external operation/page including authentication and retries, three SDK retries for retryable failures, and a ceiling of 10,000 pages per collector. These are operational defaults, not end-to-end service targets. Detect repeated continuation URLs; a deadline/ceiling/repetition produces an explicit incomplete result. Tests use controlled time/transports rather than real waits. Authentication and authorization failures are not transient retries.

#### 2. Evidence normalization, tags, and coverage model

**Files**: internal/health/model.go; internal/health/normalize.go; internal/health/privacy.go; corresponding tests.

**Intent**: Turn API responses into evidence-bound observations safe to retain and render.

**Contract**: Implement the data/privacy contracts above, including tag retention/redaction, stable evidence IDs, source versus collection timestamps, duplicate/conflict handling, and inventory/health joins. Store gaps for unavailable inventory, unmatched records, source Unknown, missing/unsupported evidence, failed pages, invalid responses, and limits reached. Mark the other three sources not_checked with reason not_implemented_in_s01.

Available is only a Resource Health observation. Null state with event metadata remains event evidence with current state undetermined. Empty successful inventory and health pages mean no resources observed, not a healthy subscription; nonempty inventory with no health records means missing health evidence. Never infer all-source coverage from these requests.

#### 3. Orchestration and checkpointing

**Files**: internal/health/check.go; internal/health/check_test.go; internal/runstore/store.go; internal/runstore/store_test.go; internal/cli/cli.go.

**Intent**: Preserve useful evidence while exposing collection failures.

**Contract**: Orchestration owns run/evidence state and source statuses; Azure adapters report pages and typed failures without owning storage. Advance revisions after committed snapshots. Select completed/partial/failed independently of report completeness according to the exit contract. Emit only sanitized stage diagnostics, with no response-body logging. Until phase 3 exists, identify run.json as collected evidence and explicitly say Markdown reporting is not yet implemented.

### Success Criteria

#### Automated Verification

- Collection and privacy tests pass: go test ./internal/azure ./internal/health ./internal/runstore
- Transport tests reject mutations, cross-subscription requests, unsafe continuations, and escaping redirects
- Multi-page failure and inventory/health independence tests preserve evidence and explicit coverage gaps
- Repository checks pass: go build ./...; go test -race -cover ./...; go vet ./...; staticcheck ./...; govulncheck ./...; ./scripts/check-format.sh; actionlint

No manual gate is needed in this phase. Synthetic failures establish controlled behavior; live evidence comparison remains pending.

## Phase 3: Markdown reports and interruption

### Overview

Complete the operator-facing report and ensure graceful interruption and write failures retain traceable, honest output.

### Changes Required

#### 1. Markdown report

**Files**: internal/report/markdown.go; internal/report/markdown_test.go; internal/report/testdata/.

**Intent**: Present a readable current-state report that can be checked against saved evidence.

**Contract**: Render from the normalized snapshot only. Include run ID/revision, scope, timestamps, execution outcome, an explicit INCOMPLETE coverage notice, source coverage table, observed issues/events, Resource Health Available observations, unknown/not-checked resources, and an evidence index. Clearly distinguish findings from failed checks and health observations from an all-source assurance.

Every finding lists its evidence IDs and links to sibling run.json; include enough structured source facts to locate the source observation. Group or sort deterministically by category and resource ID, with sorted tags. Keep the hundreds-of-resources list readable without truncating coverage silently. No overall OK resource list. Display subscription-denied/failed runs as failed, never empty successful reports.

#### 2. Consistent report publication and failure feedback

**Files**: internal/runstore/store.go; internal/runstore/store_test.go; internal/health/check.go; internal/cli/cli.go.

**Intent**: Publish a report corresponding to a known saved snapshot and identify retained output precisely on failure.

**Contract**: Finalize the filtered run.json revision, then atomically replace report.md from that revision; both identify the same run and revision. Print a saved report path only after publication succeeds. If rendering or publication fails, return failure and identify the last saved snapshot and whether an older report exists; do not describe a stale report as current. Storage failure after collection does not discard prior committed evidence. Do not pretend two file replacements form a filesystem transaction.

#### 3. Graceful Ctrl+C

**Files**: cmd/devops-ai-agents/main.go; internal/health/check.go; internal/cli/cli.go; cancellation and process integration tests.

**Intent**: Honor the operator's request to save partial work on interruption.

**Contract**: The first interrupt cancels discovery/collection, prevents further Azure calls, and finalizes any initialized run as cancelled/incomplete with evidence collected so far. Report source stages that were not attempted. Finalization uses a non-cancelled local path and never waits for new Azure data. Preserve completed pages even if interruption occurs during the next request or a retry.

Return 130 after graceful interruption. A second interrupt may force termination; the last committed checkpoint remains the recovery artifact. If storage never initialized, report that no artifact could be saved. If it fails during finalization, disclose that failure and the retained checkpoint rather than claiming a completed report.

### Success Criteria

#### Automated Verification

- Report and orchestration tests pass: go test ./internal/report ./internal/health ./internal/cli ./internal/runstore
- Signal and persistence integration tests preserve collected evidence, return 130 on graceful interruption, and expose failed writes
- Report assertions prove evidence references resolve, revisions agree, allowed tags survive, and missing evidence never becomes overall OK
- Repository checks pass: go build ./...; go test -race -cover ./...; go vet ./...; staticcheck ./...; govulncheck ./...; ./scripts/check-format.sh; actionlint

No manual gate is needed before final integration checks. Report usability and live interruption acceptance are included in phase 4.

## Phase 4: Complete-flow validation

### Overview

Prove the S-01 flow with controlled failure scenarios and the operator's private subscription, and leave an accurate operating guide.

### Changes Required

#### 1. Full-flow integration coverage

**Files**: internal/cli/integration_test.go; internal/azure/testdata/; internal/health/testdata/; internal/report/testdata/; relevant package tests.

**Intent**: Test the interactions that unit-level success cannot establish.

**Contract**: Exercise the real command/orchestration/report/storage path using synthetic CLI credentials/profile data and fake ARM transport. Any dependency injection is a Go test boundary, not a production flag that accepts arbitrary ARM endpoints or tokens. Process signal tests use controlled synchronization so interruption occurs after a checkpoint; avoid sleep-based race tests.

Cover several hundred inventory resources and multi-page health results with fewer than ten issue observations, plus independent adverse scenarios from the Testing Strategy. Verify counts/evidence references, no accidental overall OK, no live network dependency, and no committed real subscription fixtures.

#### 2. Operator and contributor documentation

**Files**: README.md; AGENTS.md; context/foundation/development-tools.md; .github/workflows/checks.yml.

**Intent**: Make the implemented behavior usable and keep onboarding accurate.

**Contract**: Document login prerequisites (including guest-tenant login), command/examples using placeholders, expected read access for subscription metadata/inventory/Resource Health, local layout, exit semantics, tags/privacy, incomplete coverage, interruption behavior, and how to inspect evidence. State that missing CLI-session entries require the operator to sign in/refresh externally and that no permission/provider changes are performed.

Update AGENTS.md and tool documentation to reflect actual implemented packages/checks and the ignored runs/ directory. Ensure CI and local commands match and all checks operate on real packages. Do not claim passing checks that could not run.

#### 3. Live acceptance record

**Files**: context/changes/subscription-resource-health-report/verification.md (implementation-time sanitized summary); local runs/<run-id>/ artifacts.

**Intent**: Validate against the chosen private subscription while accurately recording the limits of that evidence.

**Contract**: After the operator supplies the subscription and logs in, run the command by ID and by its unique name, compare representative records with current Resource Health evidence, review tags/redaction and explicit gaps, and exercise interruption where possible. Do not create outages, resources, role changes, or duplicate subscription names to manufacture cases. If the real environment lacks a scenario, use controlled integration evidence and record that scenario as not live-verified.

Commit only a sanitized pass/fail/limitations summary if the change is later committed; keep subscription IDs, resource data, and run artifacts in ignored storage. Guest-tenant behavior stays marked synthetic-only if the private subscription does not exercise guest access. Manual rows remain pending until the operator confirms their results; unavailable access is not a passing outcome.

### Success Criteria

#### Automated Verification

- Full synthetic CLI integration suite passes: go test -race -count=1 ./...
- Build, vet, static analysis, vulnerability, formatting, and workflow checks pass: go build ./...; go vet ./...; staticcheck ./...; govulncheck ./...; ./scripts/check-format.sh; actionlint
- Synthetic run files are Git-ignored and generated output is untracked; documentation and source fixtures remain eligible for version control

#### Manual Verification

- Private-subscription runs by ID and name resolve the intended scope and match representative live Resource Health observations
- Operator accepts the incomplete Markdown report, evidence references, retained identifiers/tags, and local run-directory layout
- Graceful Ctrl+C saves a cancelled/incomplete report and collected evidence when exercised during a run
- Live acceptance findings and limitations are recorded without committing subscription evidence or claiming untested guest access

**Implementation note:** After automated verification, request the operator's live acceptance results for the manual items. Do not mark the slice complete from fixtures alone. These are the only planned human acceptance gates.

## Testing Strategy

### Unit Tests

- Subscription resolution: ID/name, case differences, duplicate names, duplicate profile entries, missing/stale CLI session, noninteractive ambiguity, cancelled selection, unsupported cloud/account, and wrong subscription response.
- Authentication: selected subscription differs from default; guest-tenant account metadata; subscription-only token binding; no token/raw error leakage; expired-login/claims failures fail explicitly.
- Coverage: all known health states, event/null/missing state, invalid IDs, absent inventory, empty results, missing observations, duplicate/conflicting evidence, and inventory/health mismatch.
- Privacy: permitted tags remain; credential-like tag keys and token/secret canaries are redacted before JSON/Markdown/diagnostics. Test control characters and Markdown syntax in resource names/tags.
- Storage: distinct run IDs, initial write failure, checkpoint failure, report publication failure, prior snapshot preservation, matching revisions, ignored output, and safe paths.

### Integration Tests

| Scenario | Required evidence |
|---|---|
| Successful paginated collection | Both collectors completed; every reported finding resolves to saved evidence; three sources explicitly not checked; no overall OK. |
| Subscription Get denied | Failed run/report, nonzero exit, no collection or alternate subscription access. |
| Missing CLI-session target | Clear session-resolution error, never mislabeled a proven Azure denial. |
| Inventory denied, health succeeds | Health evidence retained; inventory coverage unverified; exit 3. |
| Inventory succeeds, health denied | Inventory retained; resources not checked; exit 3. |
| Later-page timeout/429 exhaustion/invalid continuation | Earlier valid pages retained; missing remainder explicitly reported. |
| ARM registration trigger or mutation attempt | No registration/POST/PUT/PATCH/DELETE reaches the test transport. |
| Wrong host/subscription/redirect | Request rejected; rejected response content is not retained. |
| Ctrl+C after a checkpoint | No more collection; saved cancelled/incomplete report; evidence survives; exit 130. |
| Final write fails | Nonzero result, no false saved message, last valid checkpoint remains. |
| Representative scale | Hundreds of resources remain enumerable and evidence counts agree; no product latency SLA is invented. |

### Manual Testing Steps

1. Operator supplies the private subscription and performs az login, using the required tenant if applicable. Record only non-sensitive prerequisites in the tracked verification summary.
2. Build the executable and run health check from the repository by subscription ID. Confirm selected scope and inspect the report alongside run.json.
3. Repeat by the subscription's unique name. Confirm the same target and a distinct run directory.
4. Compare available Resource Health states and timestamps with Azure for representative resources; account for source changes between observations. Review unknown/missing coverage honestly.
5. Inspect ordinary retained tags and the evidence links; confirm all three unimplemented sources are listed as not checked and no overall OK claim exists.
6. Interrupt during a run. Confirm a cancelled/incomplete report, retained evidence, and nonzero cancellation result. If a live run finishes too quickly, inspect the controlled interruption test and record the live limitation instead of claiming an observed result.
7. Run git check-ignore on the generated files and inspect git status to confirm local run data is excluded. Record manual outcomes/limits without copying client or subscription evidence into tracked docs.

## Performance Considerations

One subscription runs at a time; its collectors run sequentially. Workload is hundreds of resources, so one in-memory normalized snapshot with page checkpoints is appropriate. Do not add parallel fan-out, a database, or caching between runs. Request deadlines/retries/page limits bound individual operations, not the full run. Cancellation must not wait for the normal deadline to expire.

Checkpoint cost grows with the accumulated snapshot, but the agreed workload does not justify an append-only event system. A scale fixture checks complete enumeration and evidence integrity; there is no new latency or memory acceptance threshold.

## Migration Notes

This is the first implementation: no deployed state or data migration exists. Keep the module identity and Go version. New run snapshots begin at schema version 1; no history rewrite or automatic deletion is required. Reverting the implementation must preserve local run directories. Avoid force-adding runs/; keep sample fixtures synthetic and under package testdata.

## Open Risks & Assumptions

- The operator must supply the private subscription and establish readable access before live acceptance. That prerequisite does not justify claiming a completed slice earlier.
- A personal subscription may lack guest-tenant access, unavailable resources, or unsupported cases. Automated coverage is required; actual live coverage must be stated accurately.
- APIs may return changing resources, missing evidence, or conflicting observations. Preserve observations and uncertainty rather than inferring health.
- Retained tag text is user-approved and filtered for defined secret patterns; arbitrary free text is not a formal confidentiality guarantee.
- Forced termination, power loss, or disk failure may prevent a final report. The supported guarantee is graceful interruption with writable storage plus retention of the last successful checkpoint.
- Exact dependency/action/tool pins are recorded during implementation and must pass checks; researched SDK candidates are documented, with compatibility changes validated by adapter tests.

There are no unresolved product or architecture decisions. Implementation prerequisites and residual risks above remain explicit.

## References

- [Research and official API/SDK sources](research.md).
- [Confirmed operator decisions](change.md).
- [PRD](../../foundation/prd.md): FR-001–FR-004, US-01, Guardrails and Access Control.
- [Roadmap](../../foundation/roadmap.md): S-01, line 67; incomplete report, line 79.
- [Stack](../../foundation/tech-stack.md) and [development tools](../../foundation/development-tools.md).
- [Repository guidance](../../../AGENTS.md) and [module](../../../go.mod).
- Progress follows /Users/patrykplonka/.codex/skills/10x-plan/references/progress-format.md; only the following section owns execution checkboxes.

## Progress

> Convention: unchecked means pending; checked means done. Append a closing commit SHA when a step lands. Do not rename step titles.

### Phase 1: Subscription access and run setup

#### Automated

- [x] 1.1 CLI and scope tests pass: go test ./internal/cli ./internal/azure ./internal/runstore
- [x] 1.2 Build, race tests, and vet pass on real packages: go build ./...; go test -race -cover ./...; go vet ./...
- [x] 1.3 Static analysis, vulnerability, formatting, and workflow checks pass: staticcheck ./...; govulncheck ./...; ./scripts/check-format.sh; shellcheck scripts/check-format.sh; actionlint

### Phase 2: Evidence collection and coverage

#### Automated

- [x] 2.1 Collection and privacy tests pass: go test ./internal/azure ./internal/health ./internal/runstore
- [x] 2.2 Transport tests reject mutations, cross-subscription requests, unsafe continuations, and escaping redirects
- [x] 2.3 Multi-page failure and inventory/health independence tests preserve evidence and explicit coverage gaps
- [x] 2.4 Repository checks pass: go build ./...; go test -race -cover ./...; go vet ./...; staticcheck ./...; govulncheck ./...; ./scripts/check-format.sh; actionlint

### Phase 3: Markdown reports and interruption

#### Automated

- [x] 3.1 Report and orchestration tests pass: go test ./internal/report ./internal/health ./internal/cli ./internal/runstore
- [x] 3.2 Signal and persistence integration tests preserve collected evidence, return 130 on graceful interruption, and expose failed writes
- [x] 3.3 Report assertions prove evidence references resolve, revisions agree, allowed tags survive, and missing evidence never becomes overall OK
- [x] 3.4 Repository checks pass: go build ./...; go test -race -cover ./...; go vet ./...; staticcheck ./...; govulncheck ./...; ./scripts/check-format.sh; actionlint

### Phase 4: Complete-flow validation

#### Automated

- [x] 4.1 Full synthetic CLI integration suite passes: go test -race -count=1 ./...
- [x] 4.2 Build, vet, static analysis, vulnerability, formatting, and workflow checks pass: go build ./...; go vet ./...; staticcheck ./...; govulncheck ./...; ./scripts/check-format.sh; actionlint
- [x] 4.3 Synthetic run files are Git-ignored and generated output is untracked; documentation and source fixtures remain eligible for version control

#### Manual

- [x] 4.4 Private-subscription runs by ID and name resolve the intended scope and match representative live Resource Health observations
- [x] 4.5 Operator accepts the incomplete Markdown report, evidence references, retained identifiers/tags, and local run-directory layout
- [x] 4.6 Graceful Ctrl+C saves a cancelled/incomplete report and collected evidence when exercised during a run
- [x] 4.7 Live acceptance findings and limitations are recorded without committing subscription evidence or claiming untested guest access


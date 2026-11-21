# Complete health report implementation plan

## Overview

Roadmap S-04 completes the consultant's current-state report for one selected subscription. The change adds an evidence-backed resource assessment to the existing five-collector run. Inventory defines the observed resources. Resource Health, Service Health, Azure Monitor and Azure Advisor supply the four agreed checks.

## Current State Analysis

- S-02 and S-03 have operator acceptance. The S-03 live run returned eight New Advisor recommendations and two rejected Service Health rows. That is a real example of useful findings in a partial run. See the [S-03 verification record](../../archive/2026-11-18-advisor-recommendation-report/verification.md).
- `internal/health/model.go:13` records source status, pages, rejected rows and query metadata. `internal/health/check.go:42` saves before the read, after each accepted page and at source completion. Preparing a query and receiving a successful empty page are distinguishable.
- `internal/health/normalize.go:245` rebuilds resources, observations, findings and gaps at checkpoints. Coverage is always incomplete. `internal/health/operational.go:263` retains both standing limits and unresolved observations in the gaps list. `internal/health/advisor.go:207` already selects the current Advisor instance and gives New recommendations stable finding IDs.
- `internal/cli/cli.go:115` finalizes the outcome before the last save. `internal/runstore/store.go:71` loads the committed revision for publication. A later diagnostic save after publication failure can change the outcome again. `internal/report/markdown.go:64` and CLI output currently hard-code the incomplete verdict.

## Desired End State

A consultant opening the Markdown sees coverage and resource counts first. For example, two inventoried VMs can both have successful list queries, while only one has a current Available observation. The heading reads **Incomplete coverage · 1 resource OK, 1 unknown**. The first VM links to its proof. The second identifies the missing observation. If that VM also has a Fired alert, the same row shows the alert and the missing check. The command can exit 0 in this example.

A run with complete applicable evidence for all observed resources can report **Complete coverage** even when some resources have current findings. “OK within the agreed checks” applies only to individually eligible resources, within the stated source scopes and times. Evidence, counts, findings, source status and publication revision agree across the saved artifacts.

## Scope boundaries

Use the existing single-subscription command, collectors, Azure CLI identity and public ARM read boundary. S-05 owns selection and drafting. Additional API enumeration, Advisor refresh, per-VM rule discovery, custom checks, baselines, scheduling, and ticket submission are outside S-04.

## Implementation Approach

Derive the decision in `internal/health` from facts that can be inspected in the same snapshot. Reuse the existing latest-observation, workflow and attribution rules. Persist the result in schema 4. The renderer and CLI consume this result. They must never independently infer that a resource passed from an empty findings list.

## Phase 1: Classify each resource from saved evidence

### Changes required

**Model and ownership** — `internal/health/model.go`, `evidence.go` and `normalize.go` in the same package.

**Intent** — Give the report one auditable answer per distinct inventoried resource and preserve the reasons behind it.

**Contract** — Keep `coverage` at run level (`complete` or `incomplete`). Add schema-4 resource assessments with the resource identity, a disposition (`ok`, `finding` or `unknown`), applicable checks, blocking reasons, supporting evidence/source references and relevant finding IDs. Persist summary counts derived from the same assessments. Count each resource once. A resource with both a finding and missing proof counts as unknown and still lists its known findings. A fully assessed resource with an issue or New recommendation counts as finding. A Postponed recommendation is adjacent context. Subscription-level findings and entities absent from inventory stay visible outside the inventoried denominator.

**Source decisions** — `internal/health/normalize.go`, `operational.go`, `advisor.go` and `check.go`.

**Intent** — Make each positive or negative decision reproducible from the accepted records. Apply the following rules to the effective current observation for each entity.

| Input | Sufficient evidence within the agreed scope | Condition requiring a finding or withholding OK |
| --- | --- | --- |
| Inventory | Completed selected-subscription enumeration, accepted page and an unambiguous resource identity with an inventory evidence ID | An unresolved or conflicting identity blocks that resource. An incomplete enumeration or unscoped rejection leaves overall coverage incomplete. An empty inventory yields zero OK and a visible gap. |
| Resource Health | A current, resolved Available observation for this resource with its deciding evidence ID(s) | Current Degraded or Unavailable is an issue. Unknown, unsupported, missing, event-only or unresolved conflicting current state is unknown. When a newer Available supersedes an older Degraded observation, cite the newer evidence ID as the deciding Resource Health evidence for OK. Keep the older evidence and finding ID traceable in the resource’s history. |
| Service Health | A completed subscription list under the saved run-start 30-day update policy. Known resolved, superseded or out-of-scope events can be excluded for the stated query. | Present a current active incident once at subscription scope. Where saved evidence cannot rule out impact, withhold OK and reference that incident. Never match a VM by region alone. Keep future maintenance and informational notices labeled by type and timing. If current impact is indeterminate, retain the reason. |
| Azure Monitor | A completed selected-subscription Fired list under the saved 30-day policy, with relevant current records accounted for | A current metric/log Fired alert is a finding regardless of workflow or suppression. Preserve source target and uncertainty for workspace-only, missing or conflicting attribution. Do not assign inferred downstream faults. |
| Azure Advisor | A completed cached subscription list and a resolved current workflow state for relevant instances | New is a review finding. Postponed remains visible beside the resource. Dismissed and Completed remain source history. Unknown or conflicting state blocks a conclusion wherever it can affect the resource. Retain recommendation identity and distinguish it from a health incident. |

**Contract** — A bounded negative result requires resolved subscription scope, the expected source operation and query policy, successful enumeration to its terminal page, at least one accepted page, and no unresolved loss affecting the decision. For a completed empty Monitor list, for example, the assessment points to `sources.azure_monitor` and its query in this run/revision. It creates no synthetic observation ID. Persist any missing safe query provenance using the existing fixed operation/version and run scope, including for empty lists. `RequestedAt` alone proves preparation. Keep the existing cutoff, cached-list semantics, collection timestamps and non-atomic snapshot disclosures. Do not add a generic age threshold.

**Collection page validation** — In `internal/azure/record_decode.go`, apply Advisor’s duplicate-field checks to inventory, Resource Health, Service Health and Azure Monitor. Reject a page with duplicate `value` or `nextLink` fields before counting it as accepted. Preserve earlier committed pages and mark the source incomplete.

For inventory, Resource Health, Service Health and Azure Monitor, reject an individual record when repeated fields make its identity, state or target ambiguous. Preserve valid records on the same page, count the rejection and withhold OK wherever the rejected record could apply.

**Scope of uncertainty** — For schema 4, use the same Unicode case-folding rule for validated resource IDs during evidence deduplication, inventory grouping and cross-source joins. Preserve existing evidence references. If an identity collision prevents unique attribution, record a coverage gap and withhold OK for the affected resources. Test that an Available observation cannot cross between groups named `i` and `İ`, and that equivalent spellings of the same resource still join. Use only a trusted, normalized target to narrow a gap to a resource or group. When saved evidence cannot narrow a decision-relevant gap, apply it to every potentially affected resource. Source entity IDs, display names and geography do not establish resource attribution. Track targetless positives once with their original finding IDs. An inventory mismatch remains a run-level gap even when a distinct resource has complete proof.

Apply the schema-4 Unicode case-folding rule when grouping observations by native Monitor alert ID or Service Health event ID in `internal/health/operational.go`. Resolve the current state of each entity within its own group.

**Aggregation and lifecycle** — Separate decision-blocking gaps from the query's ordinary limits. Unclassified or rejected records require examination of their effective state and retained scope. Unknown decision-relevant reasons fail closed. Historical or resolved rows do not undo a newer decisive observation. Compute complete run coverage only after a finalized completed execution with nonempty verified inventory, sufficient source reads, no unresolved run-wide gaps and no unknown resource. Completed collectors with a localized gap can still yield OK for another resource. In-progress, failed and cancelled snapshots cannot affirm OK. A partial source retains completed pages and findings and blocks all resources for which its missing evidence cannot be excluded.

### Success criteria

#### Automated Verification

- Health package tests prove each source rule, including accepted empty lists, zero-page completion, current conflicts, superseded rows and Postponed context.
- Health package tests prove localized and subscription-wide uncertainty, disjoint counts, positive findings beside unknowns, empty inventory and checkpoint eligibility.
- Adapter tests accept a valid empty terminal page and reject duplicate `value` and duplicate `nextLink` fields for each of the four additional collectors.
- A mixed Monitor page preserves a valid Fired alert, counts the ambiguous alert as rejected and leaves source coverage incomplete. Conflicting conditions and conflicting targets each produce the same result in either field order.
- Two distinct Monitor alert IDs with the same lowercase spelling retain independent states. The Fired alert remains visible in either page order when the other alert has a later Resolved observation.

Run `go test -count=1 ./internal/health ./internal/azure`. Cover both page orders, duplicate evidence, distinct resources with similar names, case-equivalent IDs and repeated rebuilds. Assert stable assessments, reasons and references. For Degraded followed by Available, verify the saved assessment cites only the newer ID for its Resource Health conclusion and retains the earlier finding in history.

## Phase 2: Publish classifications in Markdown and CLI

### Changes required

**Report** — `internal/report/markdown.go` and the existing operational and Advisor report sections.

**Intent** — Let a reader verify why each resource passed or why the conclusion was withheld, including when a check returned no rows.

**Contract** — Lead schema-4 Markdown with coverage and disjoint resource counts. State the limited meaning of OK next to its first use. Display source windows and limitations, actionable findings and resources with missing checks. Place Postponed context and known findings beside the affected resource. Link shared incidents to one detailed subscription entry. For each assessment expose the deciding inventory and health evidence IDs, relevant finding IDs, and source references for bounded negatives. Source references must resolve to named entries in the sibling `run.json` for the printed run and revision. Render source strings literally. Validate references before publishing, including that the named source supports the cited check.

**Finalization and CLI** — `internal/cli/cli.go`, `internal/runstore/store.go` and their existing test seams.

**Intent** — Keep the displayed assessment consistent with the saved revision through successful publication, interruption and failure.

**Contract** — Finalize outcome and coverage before saving, publish from committed JSON, then emit the path to stdout and execution, coverage/counts, path and revision on stderr. Preserve codes 0 for all collectors completed and report delivered, 3 for partial execution with useful results, 2 for input or subscription-selection errors, 1 for execution failure and 130 for interruption. Refresh the assessment before a failed/cancelled publication diagnostic is saved. If final save or publish fails, identify the last retained checkpoint and any stale report. A cancellation during rendering produces the cancelled revision through finalization. A cancellation during path delivery preserves the already published pair. Update CLI help, README and the repository guidelines to describe S-04's scoped OK and exit semantics.

**Compatibility** — `internal/report/advisor_test.go`, `operational_test.go`, `testdata/` and `internal/cli/report_test.go`.

**Contract** — New runs use version 4. Pin the existing Advisor fixture builder to version 3 when changing the default. Preserve byte-for-byte rendering for the three existing schema fixtures and leave prior runs on disk. Add a version-4 fixture pair with production filenames, plus tests for mixed and complete reports. Update the new-run schema assertions in `internal/health/operational_test.go` and `internal/cli/cli_test.go` to expect version 4. In `internal/cli/integration_test.go`, make `assertIntegrationArtifacts` check version 4 and accept the scenario’s expected coverage and headline. Retain its privacy and artifact checks. Update the corresponding expectations in `internal/cli/report_test.go`. Keep existing finding/evidence IDs and the report-from-committed-revision check.

### Success criteria

#### Automated Verification

- Report tests reproduce schemas 1–3 byte-for-byte and verify schema-4 complete and mixed output, literal text, counts and resolvable evidence and query references.
- CLI and runstore tests verify stdout, stderr, exit codes, checkpoint retention, revision matching and removal of affirmative classifications after failure or cancellation.

Run `go test -count=1 ./internal/health ./internal/report ./internal/cli ./internal/runstore ./cmd/devops-ai-agents`. Verify the rendered OK row cites the newer Available evidence as decisive and the historical Degraded finding links to its original evidence. Retain exit-code-2 regressions for invalid input and ambiguous noninteractive subscription selection, including the diagnostic report published in the latter case. Exercise both cancellation timings, a failed final save, a render/publication failure after an eligible classification, and an unresolved scope. Assert no affirmative claim in any failed diagnostic revision.

## Phase 3: Verify representative scale and obtain operator acceptance

### Changes required

**Representative integration** — `internal/cli/operational_integration_test.go:106`, `cmd/devops-ai-agents/main_test.go:160` and existing synthetic fixtures.

**Intent** — Exercise the normal CLI-to-artifact path on a workload that matches the consultant's subscription.

**Contract** — Keep the existing 360-resource fixture with ambiguous Monitor and Advisor targets as the zero-OK case. Using the same five-source CLI harness, add a paged variant with 360 inventoried resources, resolved Service Health events, a completed empty Advisor list and two Fired alerts with distinct, trusted VM targets. Give every resource a current Available observation except one alerted VM. Assert exact counts of 358 OK, 1 finding and 1 unknown in JSON, Markdown and the CLI summary. Verify the unknown VM still lists its Fired alert. Run a complete-coverage variant with the missing Available observation restored and assert 358 OK, 2 findings and 0 unknown. Count Advisor recommendations separately from Monitor alerts. Retain adverse runs with denied/partial queries, unresolved incident impact, ambiguous attribution, rejected records, failed publication and interruption. Verify meaningful evidence survives, the report references the committed revision and sorted counts reconcile. Reuse the synthetic process and transports and synchronize signals on committed pages.

**Operator record** — Create `verification.md` in this change folder during implementation. Record the automated results, manual outcome, counts or ranges, portal comparison limits and explicit operator decision. Keep identifiers, run paths containing live identifiers, raw responses and credentials out of tracked files.

### Success criteria

#### Automated Verification

- The five-source CLI integration tests verify representative scale, bounded alert counts, complete and mixed classifications, adverse coverage and traceability.
- All repository build, race, vet, static analysis, vulnerability, formatting, shell and workflow checks pass against the intended packages and scripts.

#### Manual Verification

- The operator reconciles an authorized representative report with Azure, traces a live OK or accepts the documented zero-OK case with a qualifying synthetic trace, and records the decision in verification.md.

## Testing strategy

Run phase-specific tests as each change lands. For the final repository checks use the pinned tools from `context/foundation/development-tools.md` and `.github/workflows/checks.yml`.

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

Use writable temporary Go and staticcheck caches when necessary. Verify that the intended Go packages were matched. Automated tests always use synthetic processes and transports.

### Manual acceptance procedure

1. With the operator's existing authorized login, run `go run ./cmd/devops-ai-agents health check --subscription "<id-or-full-name>"` in the repository. Use the built binary when checking its exact exit status, since `go run` wraps a nonzero process exit. Open the reported Markdown and its sibling JSON. Compare run ID, revision, selected scope, time, execution, coverage and counts.
2. Trace an eligible resource through inventory, current Resource Health and the other three source entries. Compare with the selected subscription in Azure using the printed query windows and original workflow states. Check available alert, incident and recommendation examples, including the target and the difference between the source time and the collection time. For a gap, follow its reason and verify why it blocks that resource. Keep absent live scenarios identified as such.
3. If the representative live report has zero eligible resources, verify the live findings and the actual reason OK was withheld. Trace one qualifying resource through the representative synthetic report and its matching JSON. Record the two observations separately and obtain explicit acceptance of that combination. An unavailable live comparison or unresolved mismatch leaves acceptance pending.
4. Confirm local run artifacts are ignored and record only sanitized results, scenario provenance and the operator's decision. Mark manual progress only after the operator confirms the outcome.

## Performance and migration

Reuse grouped evidence and source records across assessments. Keep one detailed entry for a subscription incident and share its reference across affected resources. Avoid rescanning every evidence row for each resource during every page checkpoint. Use the representative paged fixture to verify deterministic counts and ordering. Schema 4 applies to new runs only. Preserve historical fixtures and run directories for direct comparison.

## References

- [S-04 outcome and workload](../../foundation/roadmap.md) and [US-01, FR-002 through FR-004](../../foundation/prd.md).
- [S-02 operator verification](../../archive/2026-11-06-active-alert-report/verification.md) and [S-03 operator verification](../../archive/2026-11-18-advisor-recommendation-report/verification.md).
- [Operator-approved decisions](plan-brief.md) and [pinned verification tools](../../foundation/development-tools.md).

## Progress

### Phase 1: Classify each resource from saved evidence

#### Automated

- [x] 1.1 Health package tests prove each source rule, including accepted empty lists, zero-page completion, current conflicts, superseded rows and Postponed context.
- [x] 1.2 Health package tests prove localized and subscription-wide uncertainty, disjoint counts, positive findings beside unknowns, empty inventory and checkpoint eligibility.
- [x] 1.3 Adapter tests accept a valid empty terminal page and reject duplicate `value` and duplicate `nextLink` fields for each of the four additional collectors.
- [x] 1.4 A mixed Monitor page preserves a valid Fired alert, counts the ambiguous alert as rejected and leaves source coverage incomplete. Conflicting conditions and conflicting targets each produce the same result in either field order.
- [x] 1.5 Two distinct Monitor alert IDs with the same lowercase spelling retain independent states. The Fired alert remains visible in either page order when the other alert has a later Resolved observation.

### Phase 2: Publish classifications in Markdown and CLI

#### Automated

- [x] 2.1 Report tests reproduce schemas 1–3 byte-for-byte and verify schema-4 complete and mixed output, literal text, counts and resolvable evidence and query references.
- [x] 2.2 CLI and runstore tests verify stdout, stderr, exit codes, checkpoint retention, revision matching and removal of affirmative classifications after failure or cancellation.

### Phase 3: Verify representative scale and obtain operator acceptance

#### Automated

- [x] 3.1 The five-source CLI integration tests verify representative scale, bounded alert counts, complete and mixed classifications, adverse coverage and traceability.
- [x] 3.2 All repository build, race, vet, static analysis, vulnerability, formatting, shell and workflow checks pass against the intended packages and scripts.

#### Manual

- [x] 3.3 The operator reconciles an authorized representative report with Azure, traces a live OK or accepts the documented zero-OK case with a qualifying synthetic trace, and records the decision in [verification.md](verification.md#operator-result).

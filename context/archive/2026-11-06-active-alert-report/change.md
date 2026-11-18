---
change_id: active-alert-report
title: See active operational alerts
status: archived
created: 2026-11-06
updated: 2026-11-18
archived_at: 2026-11-18T09:51:13Z
---

## Notes

Roadmap S-02 extends the accepted S-01 CLI report with Service Health events and existing Azure Monitor metric/log alert instances for the selected subscription. S-01 is archived at `context/archive/2026-11-05-subscription-resource-health-report/`. The plan and brief are in `plan.md` and `plan-brief.md`. All four phases are complete. The operator confirmed all tests and accepted S-02 on 2026-11-18. Canonical Progress is 17/17.

The user confirmed MEDIUM complexity and answered all seven planning questions. Continue the conversation and write planning documents in English.

## Confirmed decisions

1. Azure Monitor findings use the source's `Fired` condition regardless of acknowledgment or closure. Preserve and display workflow state separately.
2. Include all four Service Health categories: service incidents, planned maintenance, health advisories, and security advisories, distinguished explicitly.
3. Query Azure Monitor over 30 days and disclose the temporal coverage boundary.
4. Query Service Health events updated within the last 30 days and disclose the temporal coverage boundary.
5. Attribute Service Health events using the selected subscription, affected services/regions, and resource IDs explicitly supplied by the listing. Missing resource attribution remains unknown; do not add resource-impact lookup requests or infer affected resources from inventory matches.
6. Retain structured facts and generate summaries. Preserve the existing exclusion of arbitrary source descriptions, articles, log queries/results, and unrelated personal data; filter accepted text before checkpointing.
7. Use the same environment as S-01 for final operator acceptance, comparing both sources with the portal. Record unavailable scenarios as not live-verified; do not create incidents or change Azure permissions/resources to manufacture cases.

Existing constraints remain: GET-only public ARM access within the selected subscription's permitted operations, external Azure CLI login, sequential independent collectors, checkpoint retention, explicit incomplete coverage, no overall OK, and no drafting. Keep work uncommitted and preserve old runs.

## Planning status

The user approved the four-phase outline on 2026-11-06: evidence/coverage contracts, source adapters/read boundary, CLI/report integration, and integrated verification/operator acceptance. The detailed plan and brief were finalized on 2026-11-15 after continuation of the planning session. Automated phases proceed consecutively; manual confirmation is required only for final acceptance. Repository and external-contract research are complete; all Progress entries start pending.

The generated Monitor SDK inspected during research uses a preview operation. The plan instead uses the documented stable REST operation through the existing Azure SDK ARM pipeline and pager, without a new module dependency. Service Health reuses the installed generated SDK. The research records source provenance and the exact time-window interpretation. No live Azure requests, application implementation, commits, or changes to old runs were made during planning.

## Implementation review status

The 2026-11-18 [full implementation review](reviews/impl-review.md) returned
NEEDS ATTENTION. At review entry, the vulnerability audit passed on Go 1.27.4
and failed on CI's Go 1.27.3 selection. The operator approved fixing F1. The Go
directive now selects 1.27.4, and all required local checks passed on that
version. The operator also approved F2. Cancelled sources keep their configured
query and clear the preparation timestamp if collection never starts. Synthetic
regressions for both sources and all required local checks passed after the fix.
The operator approved F3. Service Health now omits groups whose regions all
conflict with the selected subscription, and retains the scope gap. Targeted
regressions and all required checks passed. The full review is APPROVED with
all three warnings fixed. The operator accepted the documented limit on live
scenario detail (F4), completing triage. S-02 is accepted and all 17 Progress
items are complete. The workflow status was `impl_reviewed` at triage close.

The 2026-11-15 [Phase 3 implementation review](reviews/impl-review-phase-3.md) is APPROVED after triage: F1 is FIXED, with no open findings. The user selected "Fix now" for the schema-2 example's links to the old schema-1 snapshot. The example now uses its own `operational/report.md` and `operational/run.json`, with updated documentation/readers and a regression that follows the actual relative links. The regression reproduced the original mismatch and passes after the move; all 100 links now resolve to the matching run/revision and evidence. The full race suite and every required repository check passed after the fix, including the vulnerability audit. Production code and fixture contents were unchanged by triage. Progress remains 10/17 with Phase 4/manual acceptance pending; status is `impl_reviewed`. All changes remain uncommitted.

The 2026-11-15 [Phase 2 implementation review](reviews/impl-review-phase-2.md) is APPROVED with no findings. The primary reviewer and two independent reviewers confirmed the adapters, narrow read/paging policies, fixture provenance, record isolation and retained-checkpoint behavior match the plan. Phase 2 automated checks passed again; no source or Progress changes were made during review. At that review checkpoint, canonical Progress was 7/17 with Phase 3 next and final manual acceptance pending, and status was `impl_reviewed`. That approval covers Phase 2, not subsequent implementation or live acceptance.

The 2026-11-15 [Phase 1 implementation review](reviews/impl-review-phase-1.md) is APPROVED after triage: F1 is FIXED, with no open findings. The user selected "Fix now" for filtering of source enums before exact classification. Source meaning now uses original input enums, while persisted text remains filtered. A tracked 48-case regression failed before the correction and passes afterward. The full race suite, build, vet, Staticcheck, formatting, ShellCheck and actionlint passed after the fix; `govulncheck` still reports the separately disclosed Go 1.27.3 advisory. At that review checkpoint, Phase 1 was 4/4 in canonical Progress with Phases 2–4 pending, and status was `impl_reviewed`. That approval applies to Phase 1, not later implementation or final acceptance. The earlier [pre-implementation review](reviews/impl-review-pre-implementation.md) remains a historical NOT ASSESSED record.

## Plan review status

The 2026-11-15 deep [plan review](reviews/plan-review.md) found the plan SOUND with one low-impact warning. The user approved fixing F1 in the plan: Phase 1 now includes both Azure test callers of `health.Check` and verifies the Azure test package, with matching Progress item 1.4. That plan-review F1 is FIXED; the plan verdict remains SOUND with all dimensions PASS and no open plan findings. At that pre-implementation checkpoint, status was `plan_reviewed` and all 17 execution criteria were pending.

## Phase 1 implementation checkpoint — 2026-11-15

Progress items 1.1–1.4 passed. New runs use schema 2 with typed operational facts, source identities, optional resource targets, query provenance, distinct excluded/unclassified/rejected accounting, and deterministic current-state selection. Named collectors preserve independent failures, cancellation, storage-failure precedence, counters and checkpointed query policy. The production CLI still supplies only inventory and Resource Health. Test constructors inject the future sources; both Azure coordinator callers are migrated. The existing schema-1 report fixture remains unchanged and its golden comparison now explicitly selects schema 1.

Passed with writable temporary Go/module/Staticcheck caches:

```sh
go test ./internal/health
go test ./internal/cli ./cmd/devops-ai-agents
go test ./internal/azure
go build ./...
go test -race -cover -count=1 ./...
go vet ./...
staticcheck ./...
./scripts/check-format.sh
shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh
actionlint
```

The final full race suite passed all six packages; health coverage was 97.4%. Synthetic tests cover exact temporal boundaries, all four Service Health categories, independent alert condition/workflow/suppression, uncertain and foreign attribution, duplicates/conflicts, saved-byte privacy, all 16 combinations of four-source failure, and cancellation/storage stops during either operational source. Tests did not use the operator's Azure session or live ARM access.

`govulncheck ./...` **failed** (exit 3) after a network-enabled retry: it reports reachable standard-library advisory `GO-2026-6859` in pinned Go 1.27.3, fixed in Go 1.27.4. It also reports three module advisories outside called code. The initial sandbox attempt could not reach the vulnerability database and was not counted as a pass. The Go pin and dependencies were not changed in this phase; the vulnerability audit remains an outstanding repository check.

No manual/live acceptance or commit occurred. At this implementation checkpoint, status was `implementing`; the next pending plan step was Phase 2, item 2.1. The subsequent review and completed F1 triage are recorded above. The fix and review metadata remain uncommitted.


## Phase 2 implementation checkpoint — 2026-11-15

Progress items 2.1–2.3 passed; canonical Progress is now 7/17. Status is
`implementing` because the source adapters are implemented and Phases 3–4 remain
pending. The production CLI still invokes only inventory and Resource Health.

`Collection.ServiceHealth` uses the existing generated subscription Events pager
with a validated, fixed run query policy. It projects event metadata and
service/region facts, preserving global IDs, missing optional sensitivity flags,
and metadata-only sensitive advisories. `Collection.AzureMonitor` uses the stable
2019-03-01 list operation through the existing azcore ARM pipeline/runtime pager,
with an adapter semantic version and the approved fixed initial parameters.
`go.mod` and `go.sum` are unchanged. Provider registration remains disabled and
subscription-bound credentials retain the existing policy.

Both routes enforce GET-only public ARM access to the selected subscription and
exact operation/version on every attempt. Service Health continuations may omit
the original date; a repeated date must match. Monitor requires a nonempty ctoken
for continuations and admits only the documented bounded page size, default sort
and matching repeated fixed options. Explicit HTTPS port 443 is allowed only on
the Monitor route. Equivalent links share repetition identities. Existing
inventory/Resource Health policies are preserved; the shared page helper now
receives each adapter's initial request identity explicitly.

The new literal fixtures in `internal/azure/testdata/` cite the official operation
references and document every synthetic substitution, including the Monitor
example's changed scope/version/window. Tests assert independently written
initial and continuation requests. They exercise all four Service Health
categories, global/scoped identity, sensitive metadata, nested Monitor essentials,
acknowledged/closed Fired alerts, suppression, workspace and foreign targets,
unknown enums, malformed siblings/envelopes, and JSON/debug-byte privacy.
Completed pages are saved before further requests; tests retain them through
unsafe continuations, retry exhaustion and cancellation. Rejected records cannot
conceal valid siblings. Cancellation and checkpoint failures stop further reads.

Passed using writable temporary Go/module/Staticcheck caches:

```sh
go test ./internal/azure
go test -race -count=1 ./internal/azure
go build ./...
go test -race -cover -count=1 ./...
go vet ./...
staticcheck ./...
govulncheck ./...
./scripts/check-format.sh
shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh
actionlint
git diff --check
```

The full uncached race suite passed all six packages; Azure statement coverage
was 91.7%. The current installed toolchain reports Go 1.27.4. The vulnerability
audit passed with zero reachable and zero imported-package vulnerabilities; three
advisories remain in required modules outside called code. Its initial sandbox
attempt failed DNS resolution; the network-enabled retry completed successfully.
The earlier Phase 1 failure on Go 1.27.3 is historical and was not reproduced with
the current installed toolchain. This phase changed neither the Go directive nor
module dependencies.

All Azure tests used synthetic credentials/transports. No live Azure access,
operator acceptance, historical run changes, staging or commit occurred. Phase 2
has no manual gate; portal comparison remains Phase 4. The next pending item is
Phase 3, 3.1 (CLI/report/store integration).


## Phase 3 implementation checkpoint — 2026-11-15

Progress items 3.1–3.3 passed; canonical Progress is now 10/17. Status is
`implementing`; Phase 4 integrated verification and all operator acceptance items
remain pending. No manual success or final S-02 acceptance is claimed.

The production command now runs inventory → Resource Health → Service Health →
Azure Monitor. New-source query policies derive from the saved run start;
source/request provenance is committed before the respective first request.
All collectors retain injectable Go boundaries. Per-source diagnostics expose
status, pages and rejected/excluded/unclassified counts without raw errors.
Four-source failure/outcome handling uses the existing coordinator; a local
source authentication failure does not suppress later sources.

Schema-2 Markdown uses a subscription-health title, status-derived unchecked
wording and separate Service Health/Monitor sections. Active generated summaries
remain separate from other saved observations and Resource Health findings.
The evidence index includes operational identities, subscription provenance,
category/condition/workflow/severity/suppression, service/region attribution,
source/collection times and limitations. Query dates/cutoffs, the 30-day Monitor
policy, lack of an atomic snapshot, missing resource-impact coverage, sensitive
details and stateless/retention/workspace limits are visible. Advisor remains
unchecked; complete or empty enumeration still has INCOMPLETE coverage.
Source strings render literally and no blank resource ID becomes a resource claim.

The new synthetic schema-2 snapshot/report pair has 32 evidence records and
resolvable finding references. Both existing schema-1 fixtures remain unchanged.
Save/JSON/render tests prove complete new-field round trips and committed-revision
publication even after the caller mutates its memory; unresolved references
reject publication and preserve the older report. README, AGENTS.md and living
tool guidance describe the expanded command and correct touched S-01 links to
the archive, while preserving historical verification as historical.

Passed with writable temporary Go/module/Staticcheck caches:

```sh
go test ./internal/cli ./internal/report ./internal/runstore
go test -race -count=1 ./internal/cli ./internal/report ./internal/runstore
go build ./...
go test -race -cover -count=1 ./...
go vet ./...
staticcheck ./...
govulncheck ./...
./scripts/check-format.sh
shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh
actionlint
git diff --check
```

The targeted race suite passed (CLI 22.755s; report 5.343s; runstore 1.873s).
The full uncached race/coverage suite passed all six packages: command 57.1%,
Azure 91.7%, CLI 95.9%, health 97.4%, report 96.5%, runstore 82.3%. Build, vet,
Staticcheck, formatting, ShellCheck and actionlint passed. The network-enabled
vulnerability audit on installed Go 1.27.4 passed with zero reachable and zero
imported-package vulnerabilities; three required-module advisories are outside
called code. The initial sandbox DNS failure was not counted as a passing audit.
Neither the Go directive nor module dependencies changed.

New CLI tests verify all 16 source-failure combinations, exact collector order,
committed query timing/policy, private error bytes, retained pages on either
operational source's cancellation/storage failure, and production defaults
through fake-CLI credentials and native SDK pipelines over a synthetic default
transport. Existing integration expectations now reflect successful inert
operational collectors and four-source outcome rules; full native multi-page
stress and new-source process interruption remain Phase 4 work. The earlier
Phase 1 disconnected-source test was replaced by active four-source wiring checks.

All Azure tests remained synthetic; no operator session or live ARM access was
used. Existing runs, archive files and schema-1 example bytes remain unchanged.
No staging or commit occurred. Phase 3 has no manual gate. Next pending item:
Phase 4, 4.1.


## Phase 4 automated checkpoint — 2026-11-15

Progress items 4.1–4.3 passed; canonical Progress is 13/17. Manual items 4.4–4.7
remain pending and status is `implementing`. The full verification results,
scenario limits, built-binary location and operator procedure are recorded in
[verification.md](verification.md). S-02 is not yet accepted.

New native full-flow scenarios exercise production defaults with fake CLI
credentials: 360 resources and three pages per source with seven operational
findings, 24-page stress per source, independent source failures/all-empty lists,
post-checkpoint HTTP/retry/boundary/decode failures, unknown and conflicting
observations, attribution/privacy, storage stops and publication failures.
Process tests now synchronize interruption after inventory, Service Health or
Monitor checkpoints in all four original interruption/finalization modes.
Earlier evidence is retained and later collectors stay unattempted.

The targeted race suite and all eight required repository checks passed. The
final uncached full race/coverage run passed all six packages; CLI coverage is
95.9% and command coverage is 57.1%. The network-enabled vulnerability audit on
installed Go 1.27.4 found zero reachable/imported-package vulnerabilities and
three required-module advisories outside called code. Artifact references,
matching saved revisions, incomplete coverage and actual Git-ignore behavior
passed. The schema-1 examples and module files remain unchanged. No production
code changed in this phase.

All test Azure access was synthetic. No live S-02 command, portal comparison or
operator acceptance occurred. Existing runs and archived documents were left
untouched. Changes remain uncommitted. Next pending item: 4.4, operator scope and
portal comparison; the plan requires explicit manual results before completion.


## Final operator acceptance — 2026-11-18

The operator confirmed "all tests are good, S-02 accepted" after receiving the
manual checklist. Progress items 4.4–4.7 are complete; all four phases and all
17 Progress items are done. Status is `implemented`. The sanitized confirmation
and limits of the supplied detail are recorded in [verification.md](verification.md).
No exact live exit codes, source counts, interruption stage or available scenario
variants were supplied, so none have been invented. Earlier review and automated
checkpoints above remain historical records of their respective stages.

This closeout changed documentation only. The agent did not rerun Azure access,
modify private runs or archived documents, stage files or create a commit.
The work remained uncommitted when acceptance was recorded.

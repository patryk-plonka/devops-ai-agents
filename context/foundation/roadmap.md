---
project: "AI Agents for Devops"
version: 1
status: draft
created: 2026-11-02
updated: 2026-11-02
prd_version: 1
main_goal: quality
top_blocker: none
---

# Roadmap: AI Agents for Devops

> Derived from `context/foundation/prd.md` (v1), the confirmed sequencing decisions, and a refreshed codebase inventory.
> Edit in place; archive when superseded.
> Items are listed in dependency order. The "At a glance" table is the index.

## Vision recap

Evidence is scattered and time-consuming to gather. Clients expect prices to fall, so the consultant wants to offload repeatable, deterministic, and time-consuming tasks to lower offering prices. The first release includes only the health agent, supporting manually initiated current-state checks and a Markdown report; baseline comparisons and the remaining agents follow later.

## North star

The north star here means the first complete user workflow that demonstrates the product's value.

**S-04: Consultant can review the complete four-source current-state report, including evidence-backed OK resources and explicit gaps.** This is the user's confirmed first proof of value, before drafting: it brings the agreed evidence together while preserving the quality requirements.

Sequencing follows the user's explicit quality-first preference. Invest most in collection correctness, evidence interpretation, and coverage reporting; keep the command-line interaction and local execution setup simple. This does not imply a separate server or database.

The user confirmed no known blocker after stack selection and bootstrap. Confirm Azure access during S-01; live access remains unverified and is recorded below. Earlier baseline and milestone choices remain unchanged.

## At a glance

| ID | Change ID | Outcome (user can …) | Prerequisites | PRD refs | Status |
|---|---|---|---|---|---|
| S-01 | subscription-resource-health-report | Consultant can select one subscription with the existing work identity and receive a scoped Resource Health Markdown report or an explicit access failure. | — | US-01, FR-001, FR-002, FR-003, FR-004 | ready |
| S-02 | active-alert-report | Consultant can see relevant Service Health events and existing Azure Monitor metric/log alerts in the subscription report. | S-01 | US-01, FR-002, FR-003, FR-004 | proposed |
| S-03 | advisor-recommendation-report | Consultant can review existing Azure Advisor recommendations and supporting evidence in the subscription report. | S-01 | US-01, FR-002, FR-003, FR-004 | proposed |
| S-04 | complete-health-report | Consultant can run one check and review a consolidated Markdown report of active alerts, existing recommendations, evidence-backed OK resources, and explicit coverage gaps. | S-02, S-03 | US-01, FR-002, FR-003, FR-004 | proposed |
| S-05 | selected-finding-drafts | Consultant can select findings from the reviewed report and generate ticket or recommendation drafts for manual submission. | S-04 | US-01, FR-004, FR-005 | proposed |

The report slices incrementally advance FR-002–FR-004; their repeated references do not mean each early report satisfies the complete four-source requirement. FR-001 is delivered through S-01, the complete report requirements through S-04, and selected drafting through S-05.

## Baseline

The user previously confirmed the documentation-only baseline. The following inventory refreshes it with the subsequently installed tools and verified bootstrap as of 2026-11-02.

- **Frontend:** absent, intentionally outside scope — a local command-line interface is selected; no interface implementation exists. Evidence: `context/foundation/prd.md:119`.
- **Backend / API:** partial — only the module declaration is present; no executable application or server exists. A hosted API is not required by the PRD. Evidence: `go.mod:1`, `AGENTS.md:3`.
- **Data:** absent — no Azure collection or other application data integration exists. A database has not been required or selected. Evidence: `context/changes/bootstrap-verification/setup-check.md:29`.
- **Auth:** absent — the Azure CLI is installed, but application authentication and subscription permissions have not been verified. Evidence: `context/changes/bootstrap-verification/setup-check.md:33`.
- **Deploy / infra:** partial — the local toolchain and analysis tools are installed and smoke-tested; automated checks and manual releases are selected but unconfigured. No hosted deployment is required. Evidence: `context/foundation/development-tools.md:8`, `AGENTS.md:25`.
- **Observability:** absent — no application diagnostics or runtime instrumentation exists. Installed development tools do not provide runtime reporting. Evidence: `AGENTS.md:3`.

Repository onboarding is present. No roadmap, application tests, or implemented health-agent capabilities preceded this generation. Product decisions come from the current PRD; older shaping-note gaps already resolved there are not reopened.

## Foundations

No separate foundation item is required before planning S-01. The Go module, local tools, and agent onboarding already exist and must not be re-scaffolded.

Introduce the minimum authentication, scope selection, safe failure reporting, and verification path within S-01, where the first user-visible report exercises them. Include automated checks with that first executable flow; extend verification as each later slice adds behavior. Do not treat an empty package set as passing application validation.

Coverage and evidence rules apply from the first slice. Complete resource classification is delivered in S-04 after all source paths exist. This avoids postponing safety while also avoiding a separate, speculative data or authentication layer.

## Slices

### S-01: Inspect one subscription's resource health

- **Outcome:** Consultant can select one subscription with the existing work identity and receive a scoped Resource Health Markdown report or an explicit access failure.
- **Change ID:** subscription-resource-health-report
- **PRD refs:** US-01, FR-001, FR-002, FR-003, FR-004
- **Prerequisites:** —
- **Parallel with:** —
- **Blockers:** —
- **Unknowns:** —
- **Risk:** Missing access or incomplete collection could look like a healthy result; make those states explicit in the first usable report.
- **Status:** ready

Resolve subscription IDs and names, prompt on ambiguous names, and make denied subscription access visibly unsuccessful. Show observed resource-health findings with traceable, privacy-filtered evidence. Identify resources or checks whose evidence is missing, unsupported, inaccessible, or failed. Label this as an intentionally incomplete report: the other three agreed sources are not checked yet, so source-specific healthy observations do not establish overall OK. Introduce the minimum local and automated verification path, read-only enforcement, and useful failure diagnostics with this flow.

### S-02: See active operational alerts

- **Outcome:** Consultant can see relevant Service Health events and existing Azure Monitor metric/log alerts in the subscription report.
- **Change ID:** active-alert-report
- **PRD refs:** US-01, FR-002, FR-003, FR-004
- **Prerequisites:** S-01
- **Parallel with:** S-03
- **Blockers:** —
- **Unknowns:** —
- **Risk:** Incorrect event-to-scope attribution could put unrelated alerts in a client's report; validate the selected-subscription boundary.
- **Status:** proposed

Extend the working report with the two operational-alert sources, preserving source evidence and the distinction between active findings and unavailable coverage. Keep useful findings when either source fails and mark the report incomplete. Restrict conclusions to the selected subscription and make uncertainty about affected resources visible. This slice does not configure alert rules or create new monitoring.

### S-03: See existing Advisor recommendations

- **Outcome:** Consultant can review existing Azure Advisor recommendations and supporting evidence in the subscription report.
- **Change ID:** advisor-recommendation-report
- **PRD refs:** US-01, FR-002, FR-003, FR-004
- **Prerequisites:** S-01
- **Parallel with:** S-02
- **Blockers:** —
- **Unknowns:** —
- **Risk:** A recommendation could be overstated as a confirmed fault; preserve what the source actually supports.
- **Status:** proposed

Add source-provided suggestions as the agreed proactive-check behavior, with references and resource attribution where supported. Distinguish recommendations from observed health failures. Expose inaccessible, failed, or unsupported checks without discarding other report findings. No custom detection rules or saved-baseline comparisons are introduced.

### S-04: Trust the complete four-source current-state report

- **Outcome:** Consultant can run one check and review a consolidated Markdown report of active alerts, existing recommendations, evidence-backed OK resources, and explicit coverage gaps.
- **Change ID:** complete-health-report
- **PRD refs:** US-01, FR-002, FR-003, FR-004
- **Prerequisites:** S-02, S-03
- **Parallel with:** —
- **Blockers:** —
- **Unknowns:** —
- **Risk:** An apparently complete report could omit applicable evidence and falsely label resources OK; validate coverage before making that classification.
- **Status:** proposed

Combine all four sources in one run and apply the PRD's complete-applicable-coverage rule to each resource. State the limited meaning of OK; missing or failed evidence never establishes it. Retain useful findings in incomplete runs and clearly distinguish them from successful complete runs. Validate report contents against source evidence for a representative subscription with hundreds of resources and usually fewer than ten active alerts, including adverse coverage cases. This completes the report capability before selected drafting begins.

### S-05: Draft only selected findings

- **Outcome:** Consultant can select findings from the reviewed report and generate ticket or recommendation drafts for manual submission.
- **Change ID:** selected-finding-drafts
- **PRD refs:** US-01, FR-004, FR-005
- **Prerequisites:** S-04
- **Parallel with:** —
- **Blockers:** —
- **Unknowns:** —
- **Risk:** Draft generation could change or exceed the consultant's selection; preserve selected-finding identity through the output.
- **Status:** proposed

Carry the selected findings' evidence and relevant coverage limitations into the drafts while excluding secrets and unnecessary sensitive details. Generate no drafts for unselected findings and make the selected source findings identifiable. Keep external submission under the consultant's control. Validate selection and traceability using reports from the completed four-source flow.

S-02 and S-03 may be planned and implemented in parallel once S-01 is complete. Coordinate changes to their shared report behavior and integrate both before S-04. These are parallel development tasks, not permission for concurrent subscription checks at runtime.

Live acceptance for S-01–S-04 needs an authorized subscription and permitted evidence. This is the cross-cutting open question below, not an assumed current access grant. Planning can proceed while access is confirmed; a slice is not complete based solely on simulated results.

## Backlog Handoff

| Roadmap ID | Change ID | Suggested issue title | Ready for `/10x-plan` | Notes |
|---|---|---|---|---|
| S-01 | subscription-resource-health-report | Inspect one subscription's resource health | yes | Run `/10x-plan subscription-resource-health-report`; confirm the live acceptance environment alongside planning. |
| S-02 | active-alert-report | See active operational alerts | no | Await S-01. |
| S-03 | advisor-recommendation-report | See existing Advisor recommendations | no | Await S-01. |
| S-04 | complete-health-report | Trust the complete four-source current-state report | no | Await S-02, S-03. |
| S-05 | selected-finding-drafts | Draft only selected findings | no | Await S-04. |

## Open Roadmap Questions

The PRD contains no open product questions.

1. **Which authorized subscription and permitted source evidence will be used for live acceptance?** — Owner: user. Block: live verification of S-01, S-02, S-03, and S-04; does not block planning. Confirm that the existing work identity can read the applicable sources. Until then, do not claim access or complete coverage. Exercise inaccessible and unsupported cases through controlled verification rather than changing client resources to manufacture failures.

## Parked

- **Agents beyond the health agent** — Why parked: PRD Non-Goals. The broader direction remains change-risk reviewer → incident scribe → quality reviewer → FinOps → drift reconciler → the rest. These require their own product scope before becoming implementation slices.
- **Saved-baseline comparisons and custom proactive checks** — Why parked: PRD Non-Goals; v1 summarizes current findings and recommendations already present in the four sources.
- **Scheduled checks, multi-subscription runs, and tenant-wide runs** — Why parked: PRD Non-Goals; a run is manually initiated for one selected subscription.
- **Concurrent subscription checks** — Why parked: PRD Non-Goals; one run at a time is sufficient.
- **Azure changes and automatic ticket or recommendation submission** — Why parked: PRD Non-Goals; Azure reads and manually submitted drafts define v1. Any later production write access requires the token-burn guard and self-security scanner under PRD Access Control, as well as newly agreed scope.
- **Client/team access and a web interface** — Why parked: PRD Non-Goals; the consultant is the sole operator of the local CLI.

## Done

---
project: "AI Agents for Devops"
context_type: greenfield
product_type: cli
target_scale:
  users: small
timeline_budget:
  # mvp_weeks remains undecided; see Open Questions.
  hard_deadline: null
  after_hours_only: true
created: 2026-10-31
updated: 2026-10-31
checkpoint:
  current_phase: 8
  phases_completed: [1, 2, 3, 4, 5, 6, 7]
  gray_areas_resolved:
    - topic: "Project context"
      decision: "Greenfield — new toolkit"
    - topic: "Primary operator"
      decision: "The consultant, to deliver their own consultancy services"
    - topic: "Initial recurring task"
      decision: "Daily checks of Azure Advisor, Service Health, and Resource Health; produce a ticket or recommendation"
    - topic: "Obstacle to automation"
      decision: "Evidence is scattered and time-consuming to gather"
    - topic: "Toolkit access"
      decision: "Only the consultant, using their existing work identity"
    - topic: "First-version agent permissions"
      decision: "Read Azure data and draft tickets/recommendations for the consultant to submit"
    - topic: "First-release agent scope"
      decision: "Health agent only; the other agents follow later"
    - topic: "Run initiation and report format"
      decision: "Start manually on demand, supply the environment scope, and produce a Markdown report"
    - topic: "Delivery preference"
      decision: "No hurry; quality over quantity. No week estimate supplied."
    - topic: "Run scope"
      decision: "One Azure subscription per run"
    - topic: "Active-alert coverage"
      decision: "Include existing Azure Monitor metric/log alerts alongside Azure Advisor, Service Health, and Resource Health"
    - topic: "Selected findings"
      decision: "Generate selected drafts; the consultant submits them manually"
    - topic: "Report quality"
      decision: "Traceable evidence for every finding and explicit unknown/not-checked coverage gaps are mandatory"
    - topic: "Secondary success outcome"
      decision: "None for now"
    - topic: "Proactive-check scope"
      decision: "Summarize issues and recommendations already present in the four selected Azure sources"
    - topic: "Baseline comparison"
      decision: "Current-state report only in the first release; baseline comparison comes later"
    - topic: "Subscription input and primary use case"
      decision: "Supply a subscription ID or name and run the daily routine check to obtain alerts and health suggestions"
    - topic: "Denied subscription access"
      decision: "Clearly report denied access and do not present the run as successful"
    - topic: "Ambiguous subscription names"
      decision: "Keep ID or name input; ask the consultant to choose when the name is ambiguous"
    - topic: "Meaning of OK"
      decision: "No issues found in the agreed checks, with complete applicable coverage; explicitly state that limited meaning"
    - topic: "Sensitive evidence"
      decision: "Keep traceability; exclude secrets and unnecessary sensitive details from reports and drafts"
    - topic: "Draft timing"
      decision: "Produce the report first; generate drafts only for findings explicitly selected by the consultant"
    - topic: "Partial source failure"
      decision: "Produce a partial report clearly marked incomplete, with gathered findings and explicit coverage gaps"
    - topic: "Additional measurable quality targets"
      decision: "No additional run-time, evidence-age, or retention targets for now"
    - topic: "Product form"
      decision: "A command-line tool"
    - topic: "Initial user scale"
      decision: "Single user: the consultant, as established during persona and access-control discovery"
    - topic: "Development schedule"
      decision: "No deadline; after-hours development"
    - topic: "Scale probe"
      decision: "The same evidence-and-coverage rule for OK holds at 100 times the current usage"
    - topic: "First-release exclusions"
      decision: "Exclude other agents, baseline comparisons, custom proactive checks, scheduled and multi-subscription runs, Azure changes, automatic submissions, client/team access, and a web interface"
    - topic: "Closing review"
      decision: "Accepted on 2026-10-31: finish with MVP duration unestimated and current manual-check time unquantified; preserve both gaps in the PRD handoff"
  frs_drafted: 5
  quality_check_status: warned
---

## Seed idea

AI agents to improve my consultancy offer. These agents are: **health agent → change-risk reviewer → incident scribe → quality reviewer → FinOps → drift reconciler → the rest.** The logic: health agent creates the telemetry baseline and check current state against it, change-risk reviewer intercepts new debt at the source, incident scribe compounds the runbook/context corpus, and the quality reviewer needs all three as inputs to be evidence-bound rather than vague. The token-burn guard and self-security scanner become mandatory the moment any agent gets write access to production.

## Vision & Problem Statement

The consultant performs daily checks in the Azure portal across Azure Advisor, Service Health, and Resource Health, producing a ticket or recommendation. Evidence is scattered and time-consuming to gather. Clients expect prices to fall, so the consultant wants to offload repeatable, deterministic, and time-consuming tasks to lower offering prices. The time spent per daily check has not yet been quantified.

The proposed agent sequence starts with a health agent that establishes a telemetry baseline and checks the current state against it. A change-risk reviewer intercepts new debt, and an incident scribe builds the runbook/context corpus. The quality reviewer uses evidence from all three. FinOps and drift reconciliation follow. The first release includes only the health agent, supporting manually initiated current-state checks and a Markdown report; baseline comparisons and the remaining agents follow later. The same evidence-and-coverage rule for OK holds at 100 times the current usage.

## User & Persona

The primary persona is the consultant building the toolkit, who will operate its first version as a command-line tool to deliver their own consultancy services. They reach for it during daily Azure health and advisory checks that lead to a ticket or recommendation. The initial user scale is one person.

## Success Criteria

### Primary

- For a manually initiated run against one supplied Azure subscription, the consultant receives a Markdown report covering Azure Advisor, Service Health, Resource Health, and existing Azure Monitor metric/log alerts. The report lists active alerts, possible issues from proactive checks, and resources that are OK.
- The consultant can select findings from the report and obtain ticket or recommendation drafts to submit manually.

### Secondary

None for now; the first release focuses on the core report-and-draft flow.

### Guardrails

- The agent reads Azure data without changing Azure resources; ticket and recommendation submission remains manual.
- Every finding must be traceable to supporting evidence.
- Missing, inaccessible, or unsupported evidence and failed checks must be explicitly reported as unknown/not checked, never as OK.
- Reports and drafts exclude secrets and unnecessary sensitive details while preserving traceability.
- Partial source failures produce a report clearly marked incomplete, with available findings and explicit coverage gaps.

## User Stories

### US-01: Consultant performs a daily routine check

- **Given** the consultant supplies an Azure subscription ID or name using their existing work identity
- **When** they manually run the daily routine check
- **Then** they receive a Markdown report listing alerts and health suggestions, and can select findings to generate ticket or recommendation drafts for manual submission

#### Acceptance Criteria

- One run covers one supplied subscription and the four agreed sources: Azure Advisor, Service Health, Resource Health, and existing Azure Monitor metric/log alerts.
- The report lists active alerts, possible issues and recommendations already present in those sources, resources that are OK, and explicit unknown/not-checked coverage gaps.
- Every finding has traceable supporting evidence; missing, inaccessible, unsupported, or failed checks are never presented as OK.
- Drafts are generated for findings selected by the consultant and submitted manually by the consultant.
- The first release reports current state; comparison against a saved baseline is deferred.
- If some sources return useful findings while another fails or is inaccessible, the result is a partial report clearly marked incomplete, including gathered findings and explicit coverage gaps.
- Denied subscription access is clearly reported and the run is not presented as successful.
- An ambiguous subscription name prompts the consultant to choose the intended subscription.
- OK means no issues found in the agreed checks with complete applicable coverage; the report states this limited meaning.
- Reports and drafts exclude secrets and unnecessary sensitive details while preserving traceability.

## Functional Requirements

Requirements refined through one counter-argument challenge each; all five remain must-have.

- FR-001: Consultant can access the toolkit using their existing work identity, with clear feedback when access to the supplied subscription is denied. Priority: must-have
  > Socrates: Counter-argument considered: denied client-subscription access could look like an empty successful report. Resolution: kept; clearly report denied access and do not present the run as successful.
- FR-002: Consultant can manually run a daily routine check for one Azure subscription supplied by ID or name, choosing the intended subscription when a name is ambiguous, using Azure Advisor, Service Health, Resource Health, and existing Azure Monitor metric/log alerts. Priority: must-have
  > Socrates: Counter-argument considered: a name might match multiple accessible subscriptions. Resolution: keep ID or name input; ask the consultant to choose when the name is ambiguous.
- FR-003: Consultant can receive a current-state Markdown report listing active alerts, possible issues and recommendations already present in the four selected Azure sources, resources that are OK, and explicit unknown/not-checked coverage gaps, with partial reports clearly marked incomplete when some sources fail or are inaccessible. Priority: must-have
  > Socrates: Counter-argument considered: OK could overstate confidence if it merely means no alert was found. Resolution: kept; OK means no issues found in the agreed checks with complete applicable coverage, and the report explicitly states that limited meaning.
- FR-004: Consultant can trace every reported finding to supporting evidence that excludes secrets and unnecessary sensitive details from reports and drafts. Priority: must-have
  > Socrates: Counter-argument considered: evidence in shared outputs could disclose secrets or unnecessary client details. Resolution: keep traceability while excluding secrets and unnecessary sensitive details from reports and drafts.
- FR-005: Consultant can explicitly select findings after reviewing the report to generate ticket or recommendation drafts for manual submission. Priority: must-have
  > Socrates: Counter-argument considered: drafting for every finding during the scan could create unnecessary review work. Resolution: produce the report first and generate drafts only for findings explicitly selected by the consultant.

## Non-Functional Requirements

- Every reported finding is traceable to supporting evidence.
- Reports and drafts contain no secrets or unnecessary sensitive details.
- Missing, inaccessible, or unsupported evidence and failed checks are explicitly visible as unknown/not checked, never OK.
- A run with denied access to the supplied subscription is visibly unsuccessful.
- A report produced after partial source failure is visibly incomplete and includes both available findings and explicit coverage gaps.
- Runs do not change Azure resources or submit tickets/recommendations externally; the consultant submits drafts manually.

The user explicitly chose no additional run-time, evidence-age, or retention targets for now. No thresholds or retention policy are implied by this decision.

## Business Logic

A resource is reported as OK only when the agreed checks find no issues and applicable coverage is complete, with that limited meaning explicitly stated in the report.

The input is one Azure subscription selected by ID or name and current findings from Azure Advisor, Service Health, Resource Health, and existing Azure Monitor metric/log alerts. An ambiguous name requires the consultant to choose the intended subscription; denied subscription access is clearly reported and the run is not presented as successful.

The Markdown report lists active alerts, existing issues and recommendations, resources that satisfy the OK rule, and unknown/not-checked coverage gaps. Missing, inaccessible, unsupported, or failed checks cannot establish OK status. When some sources return useful findings but another fails or is inaccessible, a partial report retains those findings and explicitly identifies the result as incomplete with coverage gaps. Every finding is traceable to evidence without exposing secrets or unnecessary sensitive details. Custom proactive checks and baseline comparisons are outside this first-release flow.

After reviewing the report, the consultant explicitly selects findings for ticket or recommendation drafts. Only those selected drafts are generated, and the consultant submits them manually.

## Access Control

Only the consultant accesses the first version of the toolkit, using their existing work identity.

The health agent can read Azure data and draft tickets or recommendations. The consultant submits those outputs. The first version does not grant the agent permission to submit tickets automatically or change Azure resources.

The token-burn guard and self-security scanner become mandatory whenever any agent gains write access to production, as specified in the seed idea.

## Non-Goals

- Agents beyond the health agent — the first release focuses on the agreed health-reporting workflow; the remaining agents follow later in the seed's sequence.
- Saved-baseline comparisons and custom proactive checks — the first release reports current findings and recommendations already present in the four selected Azure sources.
- Scheduled checks and multi-subscription or tenant-wide runs — each check is started manually and covers one explicitly supplied subscription.
- Azure resource changes and automatic ticket/recommendation submission — Azure access remains read-only and the consultant manually submits selected drafts.
- Client/team access and a web interface — the first release is a command-line tool operated only by the consultant.

## Open Questions

1. How much time does a daily Azure check currently take for one client? — Owner: user. Needed when defining measurable success criteria; no estimate supplied yet.
2. What is the estimated MVP duration? — Owner: user. The user prioritizes quality and is not in a hurry; no duration has been chosen. Do not infer a three-week commitment or a longer-timeline acknowledgment.

## Quality cross-check

Closing review completed. On 2026-10-31, the user accepted finishing with the MVP duration unestimated and current manual-check time unquantified. Quality status is warned; both gaps remain explicit in Open Questions for the PRD handoff.

| Check | Result |
| --- | --- |
| Access Control | Present: existing work identity, single consultant, read-only Azure access, manual submission. |
| Business Logic | Present: one-sentence evidence-and-coverage rule for OK, with partial-failure and draft-selection behavior. |
| Project artifacts | Present: shape-notes.md with checkpoint and greenfield context. |
| Timeline-cost acknowledgment | Missing: no MVP week estimate or longer-timeline acknowledgment; delivery effort remains unestimated. No deadline and after-hours work are explicitly selected. |
| Non-Goals | Present: all five exclusion groups confirmed. |
| Preserved behavior | Not applicable: greenfield. |

- Timeline-cost acknowledgment: without a week estimate, the duration-based check cannot be completed; the user accepted retaining this warning in the PRD handoff. This acceptance does not supply a week estimate or a longer-timeline acknowledgment.
- Additional business-outcome gap: current manual-check time remains unquantified, so the expected time savings and price reduction cannot yet be measured. Functional success criteria are captured.

## Discovery notes

User's stated motivation (verbatim):

> I need to offload all repeatable, deterministic, and time-consuming tasks to lower offering prices, as clients expect prices to fall.

Phases 1–6 are captured, including five functional requirements, one primary user story, CLI form, single-user scale, no deadline, after-hours development, and explicit non-goals. Every requirement has one recorded counter-argument response. Phase 7 is complete with the user's acceptance of the recorded gaps, and Phase 8 finalizes the handoff to /10x-prd. The MVP duration and current manual-check time remain open; quality is prioritized over speed.

User's acceptance example (verbatim):

> Given a subscription with id or name, when I run the check and select daily routine check, then the report and draft should produce list of alerts, health suggestions.

US-01 formats this example together with the previously confirmed report contents, quality requirements, and selected-draft/manual-submission flow.

User's proposed first-run flow (verbatim):

> lets make it work on demand - start manually, it will check environments ( pointed as input, like azure subscription, tenant, etc. what is the simplest thing) then it will produce report ( as markdown ) what are active alerts, what are a possible issues ( proactive check) and what resources are ok, just a list. then i will instruct it what findings to register as a ticket or recommendation based on the report

Confirmed flow:

1. The consultant manually starts a run and supplies one Azure subscription.
2. The health agent checks Azure Advisor, Service Health, Resource Health, and existing Azure Monitor metric/log alerts for that scope.
3. The agent produces a Markdown report listing active alerts, possible issues from proactive checks, and resources that are OK.
4. The consultant selects findings from the report and instructs the agent which should become a ticket or recommendation draft.
5. The consultant submits the selected drafts manually.

User's delivery preference (verbatim):

> no hurry, quality over quantity

---
project: "AI Agents for Devops"
version: 1
status: draft
created: 2026-11-01
context_type: greenfield
product_type: cli
target_scale:
  users: small
  qps: "One manually initiated check per subscription per day; one run at a time is sufficient for the first release."
  data_volume: "Hundreds of resources per subscription; usually fewer than 10 active alerts."
timeline_budget:
  mvp_weeks: 8
  hard_deadline: null
  after_hours_only: true
---

## Vision & Problem Statement

The consultant performs daily checks in the Azure portal across Azure Advisor, Service Health, and Resource Health, producing a ticket or recommendation. Evidence is scattered and time-consuming to gather. Clients expect prices to fall, so the consultant wants to offload repeatable, deterministic, and time-consuming tasks to lower offering prices. A manual daily check takes 30 minutes to 2 hours per subscription.

The proposed agent sequence starts with a health agent that establishes a telemetry baseline and checks the current state against it. A change-risk reviewer intercepts new debt, and an incident scribe builds the runbook/context corpus. The quality reviewer uses evidence from all three. FinOps and drift reconciliation follow. The first release includes only the health agent, supporting manually initiated current-state checks and a Markdown report; baseline comparisons and the remaining agents follow later. The same evidence-and-coverage rule for OK holds at 100 times the current usage.

## User & Persona

The primary persona is the consultant building the toolkit, who will operate its first version as a command-line tool to deliver their own consultancy services. They reach for it during daily Azure health and advisory checks that lead to a ticket or recommendation. The initial user scale is one person, with one manually initiated check per subscription per day.

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
- Concurrent subscription checks — one run at a time is sufficient for the first release.
- Azure resource changes and automatic ticket/recommendation submission — Azure access remains read-only and the consultant manually submits selected drafts.
- Client/team access and a web interface — the first release is a command-line tool operated only by the consultant.

## Open Questions

None currently; all recorded product-discovery questions have been answered.

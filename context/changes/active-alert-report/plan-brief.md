# Active Operational Alerts — Plan Brief

> Full plan: [plan.md](plan.md)
> Research: [research.md](research.md)
> Operator decisions: [change.md](change.md)
> Finalized: 2026-11-15. Implementation has not started.

## What & Why

S-02 adds Service Health events and existing Azure Monitor metric/log alerts to the consultant's subscription report. It brings operational evidence into the established daily check while keeping source failures, uncertain attribution, and incomplete coverage visible.

## Starting Point

S-01 already selects one subscription, checks access, collects inventory and Resource Health, saves filtered evidence page by page, and publishes Markdown from the committed snapshot. The new work extends this flow and its tests.

## Desired End State

The same command includes Service Health notices and Monitor alerts with source identities, timestamps, categories, workflow states, and evidence references. Useful results survive another source's failure. The report explains its 30-day windows and attribution limits; Advisor remains unchecked and overall coverage remains incomplete.

## Key Decisions Made

| Decision | Choice | Why | Source |
| --- | --- | --- | --- |
| Monitor active criterion | Source condition Fired, regardless of acknowledgment or closure | Keep source condition separate from operator workflow. | Plan: operator |
| Service Health categories | Incidents, maintenance, health advisories, security advisories | Include operational notices and upcoming work, with categories distinguished. | Plan: operator |
| Lookback | 30 days for each source, with the actual query semantics recorded | Make temporal coverage explicit. | Plan: operator; Research |
| Attribution | Subscription/service/region facts; only explicitly supported resource targets | Avoid inventing affected resources. | Plan: operator |
| Content | Structured facts and generated summaries | Preserve the existing privacy approach. | Plan: operator |
| Collection | Sequential independent collectors with page checkpoints | Retain useful evidence on source failure or cancellation. | Existing S-01 |
| Adapter approach | Existing Service Health SDK; stable Monitor REST over existing Azure SDK pipeline/pager | Use verified operations without another dependency. | Research |
| Stored format | Schema 2 for new runs; existing artifacts untouched | Represent subscription events and query provenance explicitly. | Plan |
| Acceptance | Final portal comparison in the S-01 environment | Verify actual service behavior and report usefulness. | Plan: operator |

## Scope

**In scope:**

- Two new source collectors, filtered evidence, source-specific findings and coverage gaps.
- CLI/report integration, strict request boundaries, independent external fixtures and regression tests.
- Final automated checks and operator acceptance with sanitized outcomes.

**Out of scope:** Advisor, overall OK, drafts, rule changes, raw descriptions/log data, resource-impact lookups, sensitive-detail POST requests, cross-subscription reads, scheduling and concurrent runs.

## Architecture / Approach

Azure adapters emit accepted facts to health normalization; each completed page is committed before the next request. Findings and gaps derive from that evidence, and Markdown reads the saved revision. Service Health event identity is separate from resource attribution. Monitor Fired is reported as a source condition, not a confirmed ongoing outage.

## Phases at a Glance

| Phase | What it delivers | Key risk |
| --- | --- | --- |
| 1. Evidence and Coverage Contracts | New source facts, time/attribution rules, extensible collector outcomes | Treating every finding as a resource fault. |
| 2. Source Adapters and Read Boundary | Both collectors with validated paging and independent fixtures | Fixtures repeating incorrect API assumptions. |
| 3. CLI and Report Integration | One usable report with explicit limits and matching saved evidence | Misleading source status or lost traceability. |
| 4. Integrated Verification and Operator Acceptance | Full-flow failure/scale checks and portal comparison | Mistaking synthetic success for live acceptance. |

**Prerequisites:** Accepted S-01, existing Go toolchain, and source-read access in the agreed environment for final acceptance. New-source access is not yet live-verified.
**Estimated effort:** Roughly 4–6 focused development sessions plus operator acceptance; service-contract discrepancies may add work. This is a planning estimate, not a delivery commitment.

## Open Risks & Assumptions

- Service-generated continuation variants may differ from documented examples; unsupported variants must retain prior pages and produce an explicit gap.
- A 30-day list cannot prove complete current health: unchanged older Service Health events may be omitted, and Monitor retention/stateless behavior limits interpretation.
- Service Health listing lacks affected-resource IDs; sensitive details and unavailable attribution remain explicit limitations.
- Live active examples may be absent. Record those cases as not live-verified and preserve the distinction from controlled test results.

## Success Criteria (Summary)

- Relevant new-source findings are traceable to filtered evidence without invented resource impact or hidden time limits.
- Source failure or interruption preserves committed results and produces honest incomplete output.
- Required checks pass and the operator accepts the final report, with live scenario limits recorded.

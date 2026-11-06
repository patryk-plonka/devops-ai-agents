# Subscription Resource Health Report — Plan Brief

> Full plan: [plan.md](plan.md)
> Research: [research.md](research.md)

## What & Why

Build the first usable slice of the Azure health CLI: select one subscription, gather Resource Health evidence, and save a Markdown report. This starts replacing repeated portal checks while making missing evidence visible.

## Starting Point

The repository contains a Go module, planning documents, and installed-tool guidance. There is no application, Azure integration, test suite, or CI workflow yet.

## Desired End State

After signing in with az login, the operator runs health check with a subscription ID or name. A unique Git-ignored runs/ directory contains the report and a structured evidence record; Ctrl+C preserves collected evidence in an incomplete report.

The report remains explicitly incomplete because Advisor, Service Health, and Monitor are later slices. Available is a Resource Health observation, never overall OK.

## Key Decisions Made

| Decision | Choice | Why | Source |
|---|---|---|---|
| Authentication | Existing Azure CLI user login, bound to the selected subscription | Fits guest access without pasted tokens or account switching | User / Research |
| Cloud and audience | Public Azure; operator alone | Matches the agreed first release | User |
| Evidence collection | Independent inventory and Resource Health reads | Missing health observations must remain visible | Research |
| Local records | runs/<timestamp-id>/run.json and report.md; ignore /runs/ | Preserves scope, evidence, and history per run | User / Plan |
| Identifiers and tags | Retain them; filter credentials/secrets before storage | Supports traceability and the requested tag context | User |
| Interruption | Cancel collection, save cancelled/incomplete output | Keeps evidence already gathered | User |
| Read-only enforcement | Disable SDK registration and restrict ARM requests | Prevents implicit or accidental writes | Research |
| Acceptance | Synthetic failure tests, then the user's private subscription | Covers adverse cases and verifies real integration | User / Plan |

## Scope

**In scope:** subscription selection/access errors, inventory, Resource Health, evidence/coverage, Markdown, local persistence, cancellation, and CI checks.

**Out of scope:** other sources and agents, overall OK classification, drafts, baselines, custom checks, scheduling, Azure writes, releases, and a hosted service.

## Architecture / Approach

A small Go CLI coordinates Azure adapters, a normalized health/evidence model, local run storage, and Markdown rendering. Successful pages are checkpointed before the next request; findings reference saved evidence. No database or LLM is required.

## Phases at a Glance

| Phase | What it delivers | Key risk |
|---|---|---|
| 1. Subscription access and run setup | CLI, scope validation, local records, first CI checks | Wrong identity/subscription or misleading access success |
| 2. Evidence collection and coverage | Inventory, health observations, filtered checkpoints, gaps | Missing or partial results mistaken for healthy |
| 3. Markdown reports and interruption | Traceable report and Ctrl+C preservation | Lost evidence or a stale report presented as current |
| 4. Complete-flow validation | Integration tests, operating guide, live acceptance | Simulated success mistaken for verified Azure access |

**Prerequisites:** existing Go setup; the user supplies a private subscription and logs in before live acceptance.
**Estimated effort:** medium, across four phases; part of the existing eight-week MVP estimate, with no separate deadline.

## Open Risks & Assumptions

- Live access is unverified; the private subscription may not exercise guest-tenant or unhealthy-resource cases.
- Tag filtering handles defined credential patterns; retained arbitrary text is not a formal confidentiality guarantee.
- Graceful cancellation preserves evidence with writable storage; forced termination/disk failure can prevent final publication.

## Success Criteria (Summary)

- The requested subscription produces a readable Resource Health report with resolvable evidence and explicit gaps.
- Denied access, failed pages, and Ctrl+C never appear as a complete healthy result.
- Local run files stay Git-ignored; automated checks pass and the operator confirms live acceptance.


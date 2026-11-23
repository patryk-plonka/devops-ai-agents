<!-- IMPL-REVIEW-REPORT -->
# Implementation Review — Selected finding drafts

- **Plan** — [Selected finding drafts implementation plan](../plan.md)
- **Scope** — Phase 2 of 3
- **Date** — 2026-11-23
- **Verdict** — NEEDS ATTENTION
- **Findings** — 0 critical, 3 warnings, 1 observation

The review used the working tree, including untracked phase 2 files and their phase 1 dependencies. All five automated progress items through phase 2 are checked. Phase 3 is next in the plan.

## Verdicts

| Dimension | Verdict |
| --- | --- |
| Plan Adherence | WARNING |
| Scope Discipline | PASS |
| Safety & Quality | WARNING |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS |

## Findings

### F1 — Draft headlines lose the qualifier on conflicting source states

- **Severity** — ⚠️ WARNING
- **Impact** — 🔎 MEDIUM · Preserving the source decision requires distinguishing a conflict in status from differences in other fields.
- **Dimension** — Plan Adherence
- **Detail** — A tied Fired and Resolved Monitor alert produces a finding titled “Azure Monitor source condition Fired in conflicting observations.” The ticket headline says “Azure Monitor Fired alert.” A tied Active and Resolved Service Health notice similarly becomes an “active notice.” Both formats show the opposing row under other observations and a coverage gap farther down. A reader can miss that the saved evidence leaves the status unresolved.
- **Fix** — Carry the source’s conflict qualification into the headline or opening observation for both draft formats. Keep the observed condition, timestamps and supporting and contextual references.
  - **Strength** — The client sees the unresolved status before interpreting the recorded positive observation.
  - **Tradeoff** — The rendering needs to preserve the distinction between conflicting states and observations that agree on state but differ elsewhere.
  - **Confidence** — HIGH · Synthetic tied opposite states reproduced the unqualified headline for both sources.
  - **Blind spot** — No real client notice was used.
- **Decision**: FIXED (2026-11-23) · Both formats call out unresolved operational state before showing the observations. Regression tests cover tied opposite states, older superseded states and tied observations that agree on state. All repository checks passed after the fix.
- **Location** — [draft.go:48](../../../../internal/report/draft.go#L48), [operational.go:584](../../../../internal/health/operational.go#L584)

### F2 — Drafts report known values for enums marked unknown

- **Severity** — ⚠️ WARNING
- **Impact** — 🔎 MEDIUM · The field-level checks are small. Service Health’s regional validity flag applies to the full observation.
- **Dimension** — Safety & Quality
- **Detail** — Normalization checks the original value and records unknown enums even when filtering produces text identical to a recognized value. Both drafts reinterpret that text. A synthetic Monitor observation displayed a recognized workflow alongside an unknown-workflow gap. A synthetic Service Health notice displayed a recognized regional status alongside an unknown-regional-state gap.
- **Fix** — Honor the saved limitations when rendering operational enums. Show an unknown value where the original enum was unrecognized. Qualify regional status conservatively when the saved evidence has only an observation-wide limitation.
  - **Strength** — Keeps the wording aligned with validation and prevents a filtered value from becoming a confirmed source fact.
  - **Tradeoff** — An observation-wide regional limitation may also withhold otherwise recognizable statuses in the same observation.
  - **Confidence** — HIGH · Synthetic normalized input reproduced both contradictions in both formats.
  - **Blind spot** — Schema 4 does not identify which regional entry triggered the observation-wide flag.
- **Decision**: FIXED (2026-11-23) · Both formats respect saved enum validity for supporting and contextual observations. The report tests cover filtered lookalikes for Monitor workflow, signal and condition and Service Health category, notice status and regional status.
- **Location** — [draft_sources.go:73](../../../../internal/report/draft_sources.go#L73), [draft_sources.go:129](../../../../internal/report/draft_sources.go#L129), [operational.go:189](../../../../internal/health/operational.go#L189), [operational.go:267](../../../../internal/health/operational.go#L267)

### F3 — An open, stalled stdout pipe blocks graceful interruption

- **Severity** — ⚠️ WARNING
- **Impact** — 🔎 MEDIUM · The writer must support cancellation at the command boundary while preserving published paths and diagnostics.
- **Dimension** — Safety & Quality
- **Detail** — Draft publication completes before the path is written synchronously to stdout. The CLI checks cancellation after that write returns. In a built-process probe, a full pipe with a live reader blocked stdout after the first draft was reported on stderr. One SIGINT left the command running. It returned 130 and reported the pending selection once the pipe resumed draining. The published draft was complete and the later draft was absent.
- **Fix** — Make production draft-path delivery cancellable and report the retained path and pending selections when the first SIGINT interrupts a blocked write. Preserve handling of a closed pipe.
  - **Strength** — The operator can stop the batch while the downstream reader is stalled and retry from the retained files.
  - **Tradeoff** — The command needs an interruptible delivery path. The generic CLI writer interface offers no cancellation primitive.
  - **Confidence** — HIGH · Reproduced with an actual SIGINT after the first file’s publication diagnostic.
  - **Blind spot** — The process probe ran on macOS.
- **Decision**: FIXED (2026-11-23) · The executable makes draft stdout delivery cancellable. Subprocess tests confirm exit 130 while the pipe stays full and exit 1 for a closed pipe. Both cases report the retained file and the pending selection.
- **Location** — [draft.go:131](../../../../internal/cli/draft.go#L131), [main.go:28](../../../../cmd/devops-ai-agents/main.go#L28)

### F4 — A draft can claim a query window the saved run could not verify

- **Severity** — ℹ️ OBSERVATION
- **Impact** — 🏃 LOW · Use the existing saved-policy check before stating the query window as verified.
- **Dimension** — Plan Adherence
- **Detail** — A synthetic matching report and snapshot contained a Service Health cutoff one day before the reference time and an `unverified_query_policy` gap. The draft stated a fixed 30-day window alongside the one-day cutoff. The renderer checks for the presence of a cutoff. The health assessment also checks the cutoff, reference time, operation, API version and subscription scope.
- **Fix** — Reuse the saved query-policy decision to qualify the window. When it cannot be verified, show recorded timestamps as such and identify the standard policy separately.
- **Decision**: FIXED (2026-11-23) · Drafts reuse assessment query-policy verification. Tests cover a mismatched Service Health cutoff, Monitor subscription scope and Advisor API version. Both formats qualify unverified scopes and distinguish recorded timestamps from standard policy.
- **Location** — [draft_sources.go:60](../../../../internal/report/draft_sources.go#L60), [assessment.go:60](../../../../internal/health/assessment.go#L60)

## Verification

| Check | Result |
| --- | --- |
| `go test -count=1 ./internal/report` | PASS · report 0.452s |
| `go test -race -count=1 ./internal/runstore` | PASS · runstore 2.271s |
| `go test -count=1 ./internal/cli ./cmd/devops-ai-agents` | PASS · CLI 13.502s, command 0.500s |
| Synthetic operational-state and enum probes | Reproduced F1 and F2. |
| Built CLI with a full stdout pipe and SIGINT after the first file was reported | Reproduced F3. The process returned 130 after draining resumed. |
| Synthetic reviewed Service Health pair with an unverified cutoff | Reproduced F4. |
| `git diff --check` | PASS |

Phase 2 has no manual verification items. Operator acceptance remains in phase 3.

## Triage outcome

All four findings are fixed. The Phase 2 verification commands and the full repository checks passed after the last fix. The vulnerability audit found zero vulnerabilities in called code or imported packages and three advisories in required modules outside called code. A built CLI returned 130 without draining a stalled stdout pipe. A retry preserved an edited draft and both source artifacts byte for byte.

<!-- IMPL-REVIEW-REPORT -->
# Implementation Review — Complete health report

- **Plan** — [Complete health report implementation plan](../plan.md)
- **Scope** — Phase 1 of 3
- **Date** — 2026-11-19
- **Verdict** — REJECTED
- **Findings** — 1 critical, 4 warnings, 0 observations

The review covers the Phase 1 working tree against HEAD, including the untracked assessment implementation and tests. Phase 2 owns CLI activation and publication of schema 4.

## Verdicts

| Dimension | Verdict |
|---|---|
| Plan Adherence | FAIL |
| Scope Discipline | PASS |
| Safety & Quality | FAIL |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | FAIL |

## Findings

### F1 — Year-zero timestamps bypass the conflict guard

- **Severity** — ❌ CRITICAL
- **Impact** — 🏃 LOW · Quick decision with a narrowly scoped fix
- **Dimension** — Safety & Quality
- **Location** — [assessment.go:63](../../../../internal/health/assessment.go#L63), [operational.go:411](../../../../internal/health/operational.go#L411)
- **Detail** — Go accepts year 0000 in RFC3339 timestamps. The candidate selector starts its comparison at the year-one zero value. Two tied year-0000 Advisor rows for one VM, one New and one Dismissed, yield an empty candidate list and no blocking gap. The synthetic run returned complete coverage and one OK. Tied Fired and Resolved Monitor observations with the same timestamp also yield an empty candidate list and panic before the page can be checkpointed.
- **Fix** — Track whether a nonzero source timestamp has been seen independently of its value, and regress both conflicts.
- **Decision**: FIXED (2026-11-20) · The selector accepts the first nonzero source time, including year 0000. Both regression cases reproduced the original failure and passed after the fix.

### F2 — Advisor conflicts can erase a known New recommendation

- **Severity** — ⚠️ WARNING
- **Impact** — 🔎 MEDIUM · A non-trivial edit to the Advisor decision and evidence mapping
- **Dimension** — Plan Adherence
- **Location** — [advisor.go:212](../../../../internal/health/advisor.go#L212), [advisor.go:287](../../../../internal/health/advisor.go#L287)
- **Detail** — Two equally current rows of one subscription-level recommendation can both say New and report different validated targets. The conflict branch emits no finding or finding reference, even though the workflow is known. The same branch also treats disagreeing Impact labels as a workflow conflict. Synthetic rows with the same target and workflow lost their New finding or blocked coverage for an agreed Completed status.
- **Fix** — Determine the workflow and attribution from the effective candidates independently of descriptive metadata. Preserve one finding with the existing native finding ID when the candidates agree on New. Link that ID to any target blockers and retain the conflicting facts in evidence.
  - **Strength** — Preserves known work for the consultant and the evidence behind disputed attribution.
  - **Tradeoff** — Requires explicit handling for conflicting workflow, target, and presentation fields.
  - **Confidence** — HIGH · Isolated probes reproduced both the target and metadata cases.
  - **Blind spot** — Live Advisor inputs were outside this review.
- **Decision**: FIXED (2026-11-20) · Schema 4 derives workflow and attribution from the effective candidates independently of the descriptive fields. Regressions cover a stable New finding across target conflicts and agreed New and Completed statuses across metadata disagreements.

### F3 — A rejected row discards independently verified scope

- **Severity** — ⚠️ WARNING
- **Impact** — 🔬 HIGH · The fix spans adapter decoding, checkpoints, and coverage
- **Dimension** — Plan Adherence
- **Location** — [record_decode.go:69](../../../../internal/azure/record_decode.go#L69), [check.go:112](../../../../internal/health/check.go#L112), [normalize.go:290](../../../../internal/health/normalize.go#L290)
- **Detail** — A synthetic terminal Monitor page with a duplicate condition and a single valid target for VM1 makes VM1 and VM2 unknown, even when VM2 has complete proof. The decoder replaces the rejected record with null. The source then stores only an unscoped `invalid_records` gap, so the assessment cannot distinguish a known-target rejection from loss of a page or a target.
- **Fix** — Record the validated target separately when the rejected fields leave attribution unambiguous. Distinguish a terminal read with scoped rejections from a failed or unfinished read. Keep the source visibly partial and the run incomplete.
  - **Strength** — Lets unrelated resources retain supported conclusions while preserving the rejection count.
  - **Tradeoff** — Requires new normalized rejection metadata and conservative fallback for uncertain attribution.
  - **Confidence** — HIGH · A synthetic adapter and assessment probe reproduced the two-resource result.
  - **Blind spot** — The probe covered Monitor condition conflicts. Scope recovery for other source and field combinations needs its own tests.
- **Decision**: FIXED (2026-11-20) · Schema 4 saves terminal enumeration and independently validated rejection scopes. Synthetic pager tests verify localized Inventory, Resource Health, Monitor, and Advisor rejections, shared entity targets, and coverage with an unscoped Service Health rejection or failed continuation.

### F4 — Case-only duplicates can reorder finding references

- **Severity** — ⚠️ WARNING
- **Impact** — 🏃 LOW · Quick decision with a narrowly scoped fix
- **Dimension** — Success Criteria
- **Location** — [normalize.go:476](../../../../internal/health/normalize.go#L476), [assessment.go:181](../../../../internal/health/assessment.go#L181)
- **Detail** — The first spelling of a deduplicated Monitor target controls the raw resource sort key. In a synthetic run with two Fired alerts on one VM and a case-equivalent copy of one alert on another page, reversing pages reversed the assessment's finding IDs. The plan requires stable references across page orders.
- **Fix** — Order each assessment's current and historical finding references by stable ID and add a two-finding regression.
- **Decision**: FIXED (2026-11-20) · Assessment references sort by stable finding ID. Reversing pages with case-equivalent targets reproduced the original reordering and now preserves both current and historical references.

### F5 — Inventory evidence hashes retain case aliases

- **Severity** — ⚠️ WARNING
- **Impact** — 🔎 MEDIUM · Evidence identity and compatibility need an explicit decision
- **Dimension** — Plan Adherence
- **Location** — [normalize.go:128](../../../../internal/health/normalize.go#L128), [assessment_test.go:157](../../../../internal/health/assessment_test.go#L157)
- **Detail** — The initial schema-4 hash folded the top-level resource ID. The embedded inventory facts still contributed the original ID and path-derived resource group, so two otherwise identical rows that differed only in ARM ID casing received separate evidence IDs. The denominator correctly counted one resource. The earlier page-order test also varied the inventory name and missed the exact duplicate case.
- **Fix** — Apply the shared fold to the embedded, ID-derived inventory fields used for schema-4 identity. Keep source spelling on retained evidence and preserve IDs on previously saved snapshots.
  - **Strength** — Aligns evidence deduplication with grouping and joins.
  - **Tradeoff** — Requires coverage for stable schema-4 references across page order and schema 1–3 compatibility.
  - **Confidence** — HIGH · A probe that varied only ARM ID casing yielded two references.
  - **Blind spot** — Phase 2 publication of schema-4 references remains pending.
- **Decision**: FIXED (2026-11-20) · Schema 4 applies the shared fold to the embedded inventory identity fields and uses one stable derived resource ID. The retained evidence keeps the first accepted source spelling, and existing saved IDs are preserved. The regression covers both page orders and checks schema-3 deduplication behavior.

## Verification

| Check | Result |
|---|---|
| `go test -count=1 ./internal/health ./internal/azure` | PASS · Both packages passed. |
| `go test -race -cover -count=1 ./...` | PASS · All six packages passed, including existing report compatibility tests. |
| `go build ./...`, `go vet ./...`, `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS · Zero reachable or imported-package vulnerabilities. Three advisories in required modules outside called code. |
| Formatting script, ShellCheck, actionlint, and `git diff --check` | PASS |
| Supplemental synthetic probes | Confirmed F1–F5 independently of the existing suite. |

Phase 1 has no manual verification items. The live operator check belongs to Phase 3 and remains pending.

## Triage outcome

All five findings were fixed on 2026-11-20. The Phase 1 tests and full repository checks passed after triage. The verdict above records the initial review.

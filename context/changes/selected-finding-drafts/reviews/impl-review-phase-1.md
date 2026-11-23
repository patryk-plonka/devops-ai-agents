<!-- IMPL-REVIEW-REPORT -->
# Implementation Review — Selected finding drafts

- **Plan** — [Selected finding drafts implementation plan](../plan.md)
- **Scope** — Phase 1 of 3
- **Date** — 2026-11-23
- **Verdict** — NEEDS ATTENTION
- **Findings** — 0 critical, 1 warning, 1 observation

The review covers the Phase 1 working tree against HEAD, including the untracked implementation and tests. Both Phase 1 progress items are checked. Phase 2 is the next pending phase.

## Verdicts

| Dimension | Verdict |
| --- | --- |
| Plan Adherence | FAIL |
| Scope Discipline | PASS |
| Safety & Quality | PASS |
| Architecture | WARNING |
| Pattern Consistency | PASS |
| Success Criteria | PASS |

## Findings

### F1 — Selection drops superseded Advisor workflow evidence

- **Severity** — ⚠️ WARNING
- **Impact** — 🔎 MEDIUM · A non-trivial change to how the resolver gathers Advisor context
- **Dimension** — Plan Adherence
- **Location** — [selection.go:218](../../../../internal/health/selection.go#L218), [selection.go:323](../../../../internal/health/selection.go#L323), [advisor.go:204](../../../../internal/health/advisor.go#L204)
- **Detail** — An earlier `Dismissed` observation and a strictly newer `New` observation for one Advisor instance produce a selectable `New` finding. The saved instance references both records. The finding correctly cites the newer row. The selection has zero context rows because it gathers context from saved gaps, and Advisor emits no conflict gap once source timestamps settle the disagreement. The resulting selection cannot trace the workflow difference separately from the finding's supporting evidence.
- **Fix** — Use the selected instance's saved evidence references to collect superseded observations that explain disputed workflow or attribution. Validate their IDs, combine them with the existing contextual references, and keep the exact supporting set.
  - **Strength** — Gives the renderer the saved source times and references for both versions, preserving the current `New` judgment.
  - **Tradeoff** — The renderer will need wording that identifies chronologically superseded observations and unresolved disagreements.
  - **Confidence** — HIGH · A synthetic probe returned two instance records, one supporting record and zero contextual records.
  - **Blind spot** — Client-facing wording is scheduled for Phase 2.
- **Decision**: FIXED (2026-11-23) · Selection retains earlier observations of the selected Advisor instance when workflow or attribution differs and source timestamps establish their order. The health tests passed with regressions for changed workflow, changed target, missing attribution and invalid contextual IDs. All repository checks passed after the fix.

### F2 — Report identity depends on explanatory prose

- **Severity** — ℹ️ OBSERVATION
- **Impact** — 🏃 LOW · Quick decision with a narrowly scoped fix
- **Dimension** — Architecture
- **Location** — [validation.go:36](../../../../internal/report/validation.go#L36)
- **Detail** — The identity check compares the report's current introductory copy and count line along with the fixed identity fields. A synthetic copy with only the explanatory wording changed returned a run ID and revision mismatch. Future editorial changes in the same schema would therefore make already-published pairs fail validation. The checked-in S-04 pair opens today.
- **Fix** — Compare the exact, Markdown-encoded identity fields at their fixed position in the generated header and keep the duplicate check. Use stable surrounding fields to anchor the identity check.
- **Decision**: FIXED (2026-11-23) · The report validator matches the encoded identity fields directly before the execution outcome. The package tests passed with changed introductory wording, an extra paragraph, and the existing stale, misplaced and duplicated identity cases.

## Verification

| Check | Result |
| --- | --- |
| `go test -count=1 ./internal/runstore ./internal/report` | PASS · runstore 0.203s, report 0.392s |
| `go test -count=1 ./internal/health` | PASS · health 0.308s |
| Synthetic Advisor context probe | Reproduced F1 with two saved records for a single instance. |
| Synthetic report header prose probe | Reproduced F2 with unchanged run ID and revision. |
| `go build ./...`, `go vet ./...`, `staticcheck ./...` | PASS |
| `go test -race -cover -count=1 ./...` | PASS · All six packages. |
| `govulncheck ./...` | PASS · Zero vulnerabilities in called code or imported packages. Three advisories in required modules outside called code. |
| Formatting, ShellCheck, actionlint and `git diff --check` | PASS |

Phase 1 has no manual verification items. Operator acceptance is pending in Phase 3.

## Triage outcome

F1 and F2 were fixed on 2026-11-23. Both original review probes, the Phase 1 verification commands, and all repository checks passed with both fixes applied.

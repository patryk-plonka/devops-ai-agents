<!-- IMPL-REVIEW-REPORT -->
# Implementation Review · Azure Advisor recommendation report

- **Plan** [Implementation plan](../plan.md)
- **Scope** Phase 3 of 4, daily review in Markdown
- **Date** 2026-11-18
- **Verdict** APPROVED
- **Findings** 0 critical, 1 warning, 2 observations

## Verdicts

| Dimension | Verdict |
| --- | --- |
| Plan Adherence | PASS |
| Scope Discipline | PASS |
| Safety & Quality | PASS |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS |

## Findings

### F1 — Advisor reports zero unclassified records when workflow status is unknown

- **Severity** ⚠️ WARNING
- **Impact** 🏃 LOW — quick decision, fix is obvious and narrowly scoped
- **Dimension** Safety & Quality
- **Location** [Advisor page normalization](../../../../internal/health/normalize.go#L207), [CLI diagnostics](../../../../internal/cli/cli.go#L292)
- **Detail** The saved schema-3 example lists zero unclassified Advisor records in source coverage. The same report shows an unknown workflow row and an `unknown_advisor_status` coverage gap. The CLI prints the same zero from the saved source counter. Advisor never increments it when accepting a record with missing or unrecognized workflow status. [Source coverage](../../../../internal/report/testdata/advisor/report.md#L23) · [Unknown workflow row](../../../../internal/report/testdata/advisor/report.md#L114) · [Coverage gap](../../../../internal/report/testdata/advisor/report.md#L159)
- **Fix** Count each accepted record with unknown workflow status before evidence deduplication, following the other source counters. Keep enumeration status and the separate gap for conflicting known statuses. Verify the saved counter and regenerate the schema-3 example.
- **Decision**: FIXED — the operator selected “Fix now.” Advisor now counts accepted records with missing or unrecognized workflow status before deduplicating evidence. The checkpoint regression reproduced a count of zero and now verifies three unclassified records across two pages, two distinct evidence records, no New findings, and completed enumeration. The schema-3 report and its sibling snapshot both show one unclassified record. The health, report, and runstore race tests passed. [Checkpoint regression](../../../../internal/health/advisor_test.go#L348)

### F2 — Advisor rows can link to unrelated saved evidence

- **Severity** 🔎 OBSERVATION
- **Impact** 🏃 LOW — quick decision, fix is obvious and narrowly scoped
- **Dimension** Safety & Quality
- **Location** [Advisor reference validation](../../../../internal/report/advisor.go#L14), [New row rendering](../../../../internal/report/advisor.go#L160)
- **Detail** A temporary synthetic test altered the schema-3 example one reference at a time. The renderer accepted a Resource Health evidence ID, a different Advisor instance's evidence ID, and a nonexistent New finding ID. The current check verifies that evidence IDs exist somewhere in the snapshot. The coordinator generates coherent references during normal collection.
- **Fix** Before rendering schema-3 rows, require linked evidence and the saved New finding to belong to the same Advisor instance in the selected scope. Add a regression for mismatched references.
- **Decision**: FIXED — the operator selected “Fix now.” Before rendering an Advisor row, schema-3 publication now checks the source, case-insensitive instance identity, and selected subscription of each linked evidence record. Each New row must identify a saved Advisor finding with matching scope, identity, and supporting evidence. The regression reproduced six accepted mismatches before the fix and now rejects all seven invalid reference cases. The report and runstore tests passed, including all historical fixtures. [Reference regression](../../../../internal/report/advisor_test.go#L184)

### F3 — The setup guide combines subscription validation with collector permissions

- **Severity** 🔎 OBSERVATION
- **Impact** 🏃 LOW — quick decision, fix is obvious and narrowly scoped
- **Dimension** Plan Adherence
- **Location** [README access instructions](../../../../README.md#L28), [Subscription validation](../../../../internal/azure/subscription.go#L47)
- **Detail** The README describes permission to read Advisor recommendations and the other source lists as requirements for live scope validation. That step reads subscription metadata. After it succeeds, the command attempts each source independently and can publish a partial report when Advisor access is denied. [CLI collection order](../../../../internal/cli/cli.go#L276)
- **Fix** Describe subscription metadata access as the scope-validation requirement and explain that each collector needs read permission for its own data.
- **Decision**: FIXED — the operator selected “Fix now.” The README distinguishes subscription metadata validation from per-collector data access and explains that denied sources are reported individually. The wording matches the subscription validator and the CLI failure-isolation tests. [README access instructions](../../../../README.md#L28)

## Scope and verification

The Phase 3 review compared the working tree with HEAD `14bc0f1`, the last committed baseline. It included the untracked schema-3 renderer, tests and fixture pair. Earlier phases' uncommitted health and Azure changes supplied the normalization and read contracts. [Plan progress](../plan.md#progress)

Phase 3 activates schema 3 before discovery and commits the Advisor query preparation before its read. Native synthetic coverage exercised the fifth collector after Azure Monitor on the original subscription-bound, read-only path. The CLI failures matrix verified exit codes and source diagnostics. The report tests covered 30 instances of one type, ordering across impacts and workflow statuses, distinct unknown-type groups, unchanged input snapshots, and matching evidence links for every example. Store tests reproduced completed-empty, duplicate-only final page, denied, partial and unattempted reads from committed data. Both historical golden reports matched byte for byte.

| Check | Result |
| --- | --- |
| `go test -race -count=1 ./internal/cli ./internal/report ./internal/runstore ./cmd/devops-ai-agents` | PASS, CLI 35.974s, report 5.866s, runstore 2.073s, command 1.775s |
| `go test -race -count=1 ./internal/azure ./internal/health` | PASS, Azure 3.393s and health 2.316s |
| `go test -race -cover -count=1 ./...` | PASS, all six packages, 95.9% CLI and 96.2% report statement coverage |
| `go build ./...`, `go vet ./...`, and `staticcheck ./...` | PASS |
| `govulncheck ./...` | PASS, zero reachable vulnerabilities and three advisories in required modules outside called code |
| Format script, ShellCheck, actionlint, and `git diff --check` | PASS |
| Synthetic fixture inspection and GFM rendering | Reproduced F1 in the published example and confirmed readable tables and relative JSON links |
| Temporary Go test overlay | Reproduced all three inconsistent-reference cases in F2 |

All three Phase 3 progress entries are checked. Live portal comparison and operator acceptance are pending in Phase 4. Review verification used synthetic data and transports.

## Verification after triage

All three findings are fixed and Phase 3 remains approved. The full uncached race and coverage suite passed across all six packages, including the counter and reference regressions. Build, vet, Staticcheck, formatting, ShellCheck, actionlint, and `git diff --check` passed. The Go vulnerability audit found zero reachable vulnerabilities and three advisories in required modules outside called code. The schema-1 and schema-2 fixture files remain unchanged. The changes are uncommitted.

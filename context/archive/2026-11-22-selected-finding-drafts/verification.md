# S-05 verification and operator acceptance

## Synthetic workflow · 2026-11-23

The integration used a synthetic Azure CLI process and in-memory ARM responses.
It exercised all five collectors through `health check` and used the published
report path and full finding IDs with `health draft`. The draft step blocked
ARM requests and verified that the synthetic Azure CLI log was byte-identical
afterwards.

| Scenario | Verified result |
| --- | --- |
| Mixed source selection | Nine findings appeared in the report. The command delivered three tickets and two recommendations in the requested order, deduplicated a repeated selection, and left four findings undrafted. |
| Two earlier Resource Health degradations | Each ticket cited the selected observation and its source time. Both cited the later Available evidence and its own timestamps. Exchanging just the two supporting references in a synthetic copy caused a mixed request to exit 2 before creating output. |
| Fired alert with missing current Resource Health | The ticket kept the reported target, alert and workflow states, suppression, and 30-day query scope. It carried the saved missing-check reason and qualified workload impact as unknown. |
| Active Service Health notice for planned maintenance | The recommendation distinguished future maintenance timing from the saved Active status. It traced separate notice and regional source updates and the fixed cutoff to JSON. Individual resource impact remained unknown. |
| New Azure Advisor instance | The recommendation used the saved instance summary, target, source times and cached-list scope. |
| Trace, privacy and local delivery | Each draft cited its expected finding, run, revision and evidence. Unselected identifiers and display markers were absent. The report and snapshot were byte-identical. All generated draft paths matched the repository's Git ignore rule. |
| Existing S-04 schema 4 pair | Finding IDs copied from the checked-in Monitor report worked with its snapshot. A retry preserved an edited ticket, delivered the other selected ticket and kept the recommendation separate. Both source files were byte-identical. |

The package tests also exercise repository aliases, stale, checkpoint-only and
unsupported pairs, finalized cancellation, bad or missing IDs, conflicting and
superseded observations, uncertain target attribution, literal rendering,
private publication, concurrent requests, Git fixture isolation, storage
failures, failed stdout, and SIGINT while either output stream is stalled.
These are fixture and process-test scenarios.

[Native workflow test](../../../internal/cli/draft_integration_test.go) ·
[S-04 compatibility test](../../../internal/cli/draft_test.go) ·
[S-04 synthetic pair](../../../internal/report/testdata/assessment/report.md) ·
[Phase 2 triage](reviews/impl-review-phase-2.md#triage-outcome) ·
[Full-slice triage](reviews/impl-review.md#triage-outcome)

## Repository checks

All commands below passed on 2026-11-23 with Go 1.27.4, Staticcheck 2026.2.2,
govulncheck 1.9.0, ShellCheck 0.11.0 and actionlint 1.7.12. The uncached race
suite covered all six Go packages. The audit found zero vulnerabilities in
called code or imported packages and three advisories in required modules
outside called code.

```sh
go test -count=1 ./internal/cli
go build ./...
go test -race -cover -count=1 ./...
go vet ./...
staticcheck ./...
govulncheck ./...
./scripts/check-format.sh
shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh
actionlint
```

## Operator acceptance

On 2026-11-23, the operator confirmed completion and accepted the workflow
under [plan item 3.3](plan.md#progress). The confirmation covered the review of
one real saved finding, tracing the fresh draft to the reviewed revision, and
preserving a manual edit on retry. The source and draft type were unspecified.
The broader scenarios documented above were verified with synthetic fixtures.

1. From the repository root, open an authorized, locally saved schema 4
   report and its sibling `run.json`. Confirm their run ID and revision agree.
   Copy a full finding ID from the report and choose the draft type for the client.
2. Run the command below with the values from the report and check stderr for
   `Draft created`. If it reports `Draft already exists`, choose another
   finding or type, or move the existing copy aside and generate a fresh draft.
   Review the fresh draft's requested action, target, source and collection
   times, coverage limits and included details. Match its run, revision,
   finding and evidence IDs to the same report and snapshot.
3. Make an edit to the draft and rerun the identical command. Check stderr for
   `Draft already exists` and confirm the edit survived.
4. Record the source, draft type, trace and retry results, wording corrections,
   and explicit acceptance decision. Keep real identifiers, file paths and
   copied draft text private. Use the synthetic scenarios above for cases
   absent from the selected live finding.

```sh
go run ./cmd/devops-ai-agents health draft \
  --report "runs/<run-id>/report.md" \
  --type ticket \
  --finding "<full-finding-id>"
```

Use `--type recommendation` when appropriate. See the
[README](../../../README.md#draft-selected-findings) for the built executable,
exact exit codes and trace instructions.

| Item | Real finding result |
| --- | --- |
| Source and draft type | Unspecified in the operator's confirmation |
| Trace and client wording review | Accepted by the operator |
| Edit-preserving retry | Confirmed by the operator |
| Explicit decision and date | Accepted on 2026-11-23 |

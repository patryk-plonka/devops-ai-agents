# S-04 verification and operator acceptance

The operator confirmed completion of the manual check and accepted S-04 on
2026-11-21 under [plan item 3.3](plan.md#progress). The automated runs used
synthetic Azure CLI credentials and in-memory ARM responses.

## Automated verification · 2026-11-20

The targeted five-source integration, process interruption and ignore tests
passed. All repository checks exited 0 with Go 1.27.4, Staticcheck 2026.2.2,
govulncheck 1.9.0, ShellCheck 0.11.0 and actionlint 1.7.12.

```sh
go test -count=1 ./internal/cli ./cmd/devops-ai-agents -run 'TestFiveSourceNative|TestProcessInterrupt|TestGeneratedRunFilesAreIgnored'
go build ./...
go test -race -cover -count=1 ./...
go vet ./...
staticcheck ./...
govulncheck ./...
./scripts/check-format.sh
shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh
actionlint
```

The uncached race run covered all six Go packages. Statement coverage was 59.1%
in the command, 92.3% in Azure, 96.0% in CLI, 91.5% in health, 94.5% in report
and 82.0% in runstore. The public vulnerability audit found zero reachable and
zero imported-package vulnerabilities, with three advisories in required modules
outside called code.

## Representative integration results

Each 360-resource case exercised the normal CLI path through subscription selection,
all five native collectors, page checkpoints, and paired Markdown and JSON publication.

| Synthetic scenario | Coverage | OK | Finding | Unknown | Fired Monitor alerts | New Advisor recommendations |
| --- | --- | ---: | ---: | ---: | ---: | ---: |
| Ambiguous Monitor and Advisor targets with unresolved Service Health impact | Incomplete | 0 | 0 | 360 | 3 | 30 |
| Two trusted alert targets, one missing current Available observation | Incomplete | 358 | 1 | 1 | 2 | 0 |
| The missing Available observation restored | Complete | 358 | 2 | 0 | 2 | 0 |

All three commands returned 0. Both trusted-target cases read two resolved
Service Health events and one terminal empty Advisor page. The mixed case kept
the second VM's Fired finding and recorded `missing_health_evidence` for that VM.
The tests reconcile the JSON, Markdown, and CLI summaries, verify ascending
assessment rows from descending input across pages, match the saved revision,
and trace assessment links to source queries, evidence, and current findings.

Native tests retain denied and partial reads, rejected siblings, unresolved impact,
ambiguous attribution, and injected checkpoint or publication failure. The
subprocess tests send SIGINT after a committed page. They preserve the checkpoint's
evidence and findings, and verify incomplete coverage. Graceful cancellation
publishes both artifacts at the same revision and returns 130. An interrupted
finalization retains the checkpoint without publishing a report. Artifact tests
check private fixture values and the repository's `/runs/` ignore rule.

[Native integration tests](../../../internal/cli/operational_integration_test.go) ·
[Signal tests](../../../cmd/devops-ai-agents/main_test.go) ·
[Artifact and ignore checks](../../../internal/cli/integration_test.go)

## Operator procedure

Run from the repository using the existing authorized Azure CLI login. Use the
built executable to record the CLI's exact exit status.

```sh
binary="$(mktemp -d)/devops-ai-agents"
go build -o "$binary" ./cmd/devops-ai-agents
"$binary" health check --subscription "<authorized-id-or-full-name>"
```

1. Open the published `report.md` and its sibling `run.json`. Compare the run ID,
   revision, selected subscription, execution, coverage, counts and collection
   times. Keep the two artifacts in ignored local storage.
2. Trace one live OK resource from its inventory evidence through current
   Available Resource Health and the other three completed source queries.
   Compare available alerts, incidents and recommendations with Azure. For each
   example, check its target, source timestamp and collection timestamp. Follow
   an available blocking reason to its check and scope.
3. If the live run has zero OK resources, validate the live findings and reasons
   that withheld OK, then trace one qualifying synthetic resource. In the
   360-resource synthetic mixed report, `vm-000` links to inventory and current
   Available evidence and to the three completed bounded queries. `vm-240`
   shows both its Fired finding and the missing Resource Health evidence. The
   [checked-in synthetic report](../../../internal/report/testdata/assessment/report.md)
   and its [matching snapshot](../../../internal/report/testdata/assessment/run.json)
   also provide a durable qualifying trace. Record the live and synthetic
   observations and explicitly accept or reject that combination.
4. Check local artifact paths with `git check-ignore` and confirm
   `git ls-files -- runs/` is empty. Record sanitized outcomes and the explicit
   decision below. Keep real IDs, run paths, raw service data and credentials local.

### Portal comparison limits

- Service Health uses the saved run-start 30-day update cutoff and API date.
  The portal view may use different filters or get later updates. A subscription
  or region notice alone leaves individual resource impact unresolved.
- Monitor uses the 30-day Fired view across New, Acknowledged and Closed workflow
  states. Compare suppression and the recorded target independently.
- Advisor reads the cached subscription list without a date filter or refresh.
  Compare recommendations with aligned scope, category and workflow filters.
- Collectors read sequentially. Compare the source and accepted-page times, and
  record unavailable portal fields, absent live scenarios and unresolved differences.

### Operator result

| Item | Result |
| --- | --- |
| Live execution, coverage and resource counts or ranges | Operator confirmed the manual check. Exit code, coverage and counts were unspecified. |
| Live source counts, finding trace and portal comparison | Operator confirmed the manual check. Source counts and individual comparisons were unspecified. |
| Live OK trace or validated zero-OK reasons with synthetic trace | Operator confirmed the manual check. The trace used for acceptance was unspecified. |
| Actual portal differences, unavailable fields and absent scenarios | Unspecified in the operator confirmation. |
| Local artifact privacy check | The repository ignore rule covers both artifact paths under `runs/`. `git ls-files -- runs/` returned no entries. |
| Explicit operator decision and date | Accepted on 2026-11-21. The operator confirmed the manual check was good and asked to mark it done. |

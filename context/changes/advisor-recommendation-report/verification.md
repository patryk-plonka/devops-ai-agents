# S-03 · Advisor recommendation report verification

## Automated results · 2026-11-18

All commands below passed locally using Go 1.27.4 on macOS arm64, with writable
temporary Go and Staticcheck caches.

```sh
go test -race -count=1 -run 'TestFiveSourceNative|TestProcessInterruptPreservesCheckpoint|TestGeneratedRunFilesAreIgnored' ./internal/cli ./cmd/devops-ai-agents
go build ./...
go test -race -cover -count=1 ./...
go vet ./...
staticcheck ./...
govulncheck ./...
./scripts/check-format.sh
shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh
actionlint
git diff --check
```

The full race run covered all six packages. Statement coverage was 57.1% for
the command, 92.2% for Azure, 95.9% for CLI, 98.0% for health, 96.2% for report,
and 82.3% for runstore. The public Go vulnerability audit found zero reachable
and zero imported-package vulnerabilities. It found three advisories in required
modules outside called code. The acceptance binary also passed its `--help`
smoke check. The root ignore rule matched both expected run file paths, and
`git ls-files -- runs/` returned no tracked files.

## Automated scenarios

The integrated tests exercise subscription selection, the native ARM pagers,
normalization, storage, and Markdown publication. The CLI tests use an isolated
synthetic `az` and an in-memory ARM transport. The process tests use synthetic
credentials and an in-memory transport. Both paths avoid the operator’s Azure
session and live ARM.

| Scenario | Evidence |
| --- | --- |
| Populated and empty runs | All five collectors finish in order with three pages apiece, including 360 inventory resources and 30 New Advisor instances sharing one type. A separate run exercises 24 pages per source. The empty run accepts one page from each source, returns 0, and reports zero cached recommendations and incomplete overall coverage. |
| Source isolation | Advisor denial retains earlier results. An earlier denial retains Advisor evidence, including when Advisor is the only completed source. Partial runs return 3. All five source reads failing returns 1. |
| Partial Advisor collection | A later denial, foreign-scope continuation, or invalid page envelope preserves the first committed page. The mixed-record run preserves valid records from both pages and counts the malformed and foreign siblings as rejected. |
| Checkpoint, publication and signals | A failed second-page checkpoint preserves the previous revision. Final-save and publication failures retain the last successful snapshot and return 1. The process tests signal after reading the committed Advisor page. A graceful interrupt returns 130 and publishes a cancelled report with matching run ID and revision. The tests also exercise retry delay, failed final writes, and forced termination on a second interrupt. |
| Privacy and artifact identity | Published reports link Advisor rows to evidence in the matching JSON revision. The tests check JSON, Markdown, stdout, and debug stderr for synthetic values from excluded properties, foreign targets, continuations, and error bodies. Generated artifacts are ignored and synthetic fixtures remain eligible for version control. |

[Native integration tests](../../../internal/cli/operational_integration_test.go) ·
[Process tests](../../../cmd/devops-ai-agents/main_test.go) ·
[Artifact and ignore checks](../../../internal/cli/integration_test.go)

## Live acceptance · 2026-11-19

On 2026-11-19, the operator ran the built command against the authorized
subscription. Advisor completed one page and retained eight New recommendations.
The run returned 3 because Service Health rejected two records and was partial.
Inventory, Resource Health, and Azure Monitor completed.

Local inspection confirmed that the report and snapshot share the same run ID
and revision, all eight Advisor links resolve to their corresponding saved
evidence, and the report retains inventory and Resource Health evidence with
incomplete overall coverage. Git ignores both artifacts and tracks no run files.

The operator explicitly accepted S-03 on 2026-11-19 and confirmed completion of
the manual checklist. The portal comparison and evidence inspection are recorded
as operator-confirmed. Individual portal field results were unspecified.

## Reproduce the manual checks

Run from the repository with an externally established Azure CLI session.

```sh
go build -o /tmp/devops-ai-agents ./cmd/devops-ai-agents
/tmp/devops-ai-agents health check --subscription "<authorized-subscription-id-or-full-name>"
```

1. Record the exit status and check that Advisor shows `completed`. Compare
   the run ID and revision in the published `report.md` and sibling `run.json`.
   Review useful evidence from earlier sources and the incomplete coverage statement.
2. In the Azure portal, select the same subscription and align category and
   workflow filters. Note timing differences. For a populated list, trace a row
   through its `ev-...` reference to `evidence` in the snapshot, then use its
   instance identity to compare with Azure. Check type or group membership,
   workflow status, impact, assessed resource and evaluation time where visible.
3. For a completed empty list, confirm zero under comparable portal filters and
   inspect the [populated synthetic report](../../../internal/report/testdata/advisor/report.md)
   with its [matching snapshot](../../../internal/report/testdata/advisor/run.json).
   Record populated evidence as synthetic in that case. A denied, partial or
   mismatched list leaves the comparison pending.
4. Confirm that both local artifacts are ignored and report the sanitized
   comparison outcome, any unavailable fields or scenarios, and your explicit
   acceptance or rejection. Keep subscription and resource identifiers, actual
   run files, raw service data and credentials local.

```sh
git check-ignore -- "runs/<run-id>/run.json" "runs/<run-id>/report.md"
git ls-files -- runs/
```

Advisor denial, interruption, scale, and privacy canary scenarios were
exercised synthetically.

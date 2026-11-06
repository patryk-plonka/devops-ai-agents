# S-01 verification

Updated: 2026-11-06. **Automated verification passed; operator confirmed all manual checks.**
This record contains synthetic results, live execution results, and a live
checklist. Initial operator runs exposed CLI-schema and pagination validation
bugs, now corrected. Subsequent live runs by name and resolved UUID both completed
with exit 0, matching scope, and matching report/snapshot revisions. The operator
subsequently confirmed "all manual tests are ok" on 2026-11-06, including source
comparison, report acceptance, interruption, and the sanitized results/limitations.
All 18 Progress rows are complete; S-01 is implemented and accepted.
Changes remain uncommitted at the operator's request.

## Automated results

All required commands passed locally on macOS using the Go directive/tool pins
in `go.mod` and `context/foundation/development-tools.md`:

```sh
go build ./...
go test -race -cover -count=1 ./...
go vet ./...
staticcheck ./...
govulncheck ./...
./scripts/check-format.sh
shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh
actionlint
```

Writable temporary Go/module/Staticcheck caches were used. The vulnerability
database request initially failed under the network sandbox; the permitted retry
completed and found zero reachable or imported-package vulnerabilities, plus
three advisories in required modules outside called code. CI configuration was
checked locally; a GitHub-hosted workflow execution is not claimed.

| Scenario | Evidence and result |
| --- | --- |
| Full flow by UUID and case-insensitive full name | `TestFullFlowScaleByIDAndName`: real command parsing, CLI discovery and AzureCLICredential subprocesses, native ARM SDK requests/pagers, orchestration, normalization, atomic storage, and Markdown publication; both select the synthetic non-default guest subscription. |
| Representative scale | 360 inventory resources and 359 health observations across three pages per source; 719 unique evidence records, 360 resource observations, six issue/event findings; complete enumeration, resolvable references, matching run/revision, and explicit incomplete coverage. No latency SLA is inferred. |
| Checkpoint ordering | The fake wire reads the saved snapshot before each subsequent page; earlier pages must be committed first. All source states, timestamps, ordinary tags, redactions, and evidence references survive publication. |
| Scope/access failures | `TestFullFlowScopeAndIndependentFailures`: subscription Get denial, mismatched identity, provider-registration error, missing session target, and expired login produce explicit failed reports and stop before collection. Missing session returns 2; access/authentication failures return 1. |
| Independent collectors | Inventory denied/health succeeds and the reverse retain the successful evidence and return 3; both denied returns 1. Empty results, missing health, and health absent from inventory have explicit gaps. |
| Later-page failures, both sources | `TestFullFlowLaterPageFailures`: timeout, four 429 attempts, off-host/cross-subscription continuation, redirect, registration error, repeated continuation, invalid envelope, and mixed invalid records retain valid earlier evidence with partial coverage and exit 3. Rejected payloads are absent. |
| Read boundary | `TestTransportRejectsUnsafeRequestsBeforeSending` and `TestCollectionTransportAndContinuationBoundaries` reject mutations, registration routes, wrong host/subscription/API/query, and redirects before further transport access. Full-flow wire also rejects any unexpected method/scope. |
| Storage failures | `TestFullFlowFinalWriteFailureRetainsNativePages` and `TestFullFlowCheckpointFailureStopsNativeCollection` retain prior evidence, return 1, stop further reads on checkpoint failure, and avoid claiming a saved report. Existing publication tests retain stale reports explicitly when applicable. |
| Interruption | `TestProcessInterruptPreservesCheckpoint` synchronizes real process signals after a native SDK page checkpoint, including during a next request/retry. Graceful cases return 130 with cancelled/incomplete matching artifacts; failed final writes are disclosed; second interrupt retains the last checkpoint. This is synthetic process evidence. |
| Privacy/presentation | Full-flow secret/profile/token/error canaries do not appear in JSON, Markdown, or stderr, including with `--debug` and inherited SDK logging. Ordinary tags survive; credential-like values are redacted; source markup remains literal. Current/historical/Unknown/event/missing states stay distinct. |
| Git exclusion | `TestGeneratedRunFilesAreIgnored` creates real snapshots/reports in an isolated Git repository using this project's `.gitignore`. Generated files are ignored, absent from ordinary status and tracked files; documentation and synthetic package fixtures remain eligible for version control. |

Package statement coverage after the pagination correction: CLI 93.1%, Azure 87.6%, health 94.2%, report 98.6%,
runstore 81.5%, process entry 57.1%. Race detection passed. These figures describe
synthetic validation, not real Azure API behavior or operator report acceptance.

## Live preflight — 2026-11-06

The operator's initial UUID run returned 2 because that UUID was absent from the
cached CLI account list. The name run returned 2 because the application rejected
the matching account as unsupported. Both produced explicit failed diagnostic
reports; neither reached live subscription validation or collection.

A local read of cached CLI metadata found one name match with `cloudName` set to
`AzureCloud` and user type `user`; its UUID differed from the supplied UUID.
The application incorrectly decoded `environmentName` instead of the CLI output's
`cloudName`. The synthetic CLI fixture repeated that schema error. The parser and
fixture are corrected, and an independent literal-JSON regression now exercises
public-cloud user selection by UUID/name while rejecting sovereign clouds,
missing cloud metadata, and service-principal accounts. The regression reproduced
the failure before the fix and passes afterward, together with the full uncached
race suite and all required repository checks.

The local executable was rebuilt. The operator then reached live validation and
collection by name. A later failed ID attempt used the tenant ID; the operator
was shown how to obtain the subscription ID and subsequently confirmed all manual
checks. No account switching, login refresh, permission/provider changes,
or fallback subscription selection was performed. No real identifiers or account
usernames are recorded here.

## Live paging correction and reruns — 2026-11-06

The next operator run validated the subscription and completed inventory, but
returned exit 3 after the first Resource Health page with `unsafe_continuation`.
The saved page evidence and explicit gap were preserved.

A single read-only diagnostic request inspected only the continuation structure.
The service returned the selected subscription's exact operation on public ARM
with the expected API version, lowercase `$skiptoken`, and a `$top` page size.
The allowlist previously accepted only `$skipToken` for Resource Health and no
page-size parameter. It now accepts either token spelling and one positive
31-bit page size for Resource Health. Ambiguous/duplicate token parameters,
invalid/duplicate page sizes, filters/expansions, wrong hosts/subscriptions/API
versions, redirects, and mutations remain rejected. Token aliases normalize to
the same repetition key. Inventory's query allowance was not broadened.

The service-shaped regression reproduced the rejection before the fix and passes
afterward. The full CLI fixture now exercises lowercase pagination and page size
through native SDK pagers, checkpointing, normalization, and report publication.
All required repository checks passed again, including uncached race tests and
the vulnerability audit.

Using the rebuilt executable, the agent reran the selected subscription by its
resolved name and UUID. Both exited 0 with completed inventory/Resource Health
collectors; Resource Health consumed two pages. The scope matched between runs,
run directories were distinct, and Markdown matched each saved run ID/revision.
There were no live issue findings, so live finding-reference usability remains
unexercised; synthetic finding-reference tests cover that case. Both live run
directories are ignored by Git and no run files are tracked. Actual evidence,
identifiers, and pagination tokens were not copied into this record.

These successful executions were followed by the operator's confirmation that
all manual tests passed. The manual outcomes below are operator-reported; the
agent independently observed the successful name/UUID runs and Git exclusion.

## Live acceptance

| Progress row | Status | Acceptance evidence |
| --- | --- | --- |
| 4.4 | PASS | Successful name/resolved-UUID runs observed by the agent; operator confirmed intended scope and representative live Resource Health comparison on 2026-11-06. |
| 4.5 | PASS | Operator confirmed acceptance of the incomplete report, evidence references, identifiers/tags, and local layout on 2026-11-06. |
| 4.6 | PASS | Operator confirmed the graceful Ctrl+C manual check on 2026-11-06: cancelled/incomplete saved output, retained evidence, and cancellation exit behavior. This is operator-reported acceptance. |
| 4.7 | PASS | Operator confirmed all manual checks on 2026-11-06; sanitized outcomes/limits are recorded here, local artifacts are ignored, and guest access is not claimed live-verified. |

Keep actual subscription names/IDs, tenant/resource identifiers, source evidence,
and generated reports in ignored local storage. Report only pass/fail, acceptance,
and limitations here. Do not paste tokens or copy live evidence into fixtures.

To repeat the accepted checks using an externally established CLI session, build
and run from the repository tree (replace placeholders locally):

```sh
az login
# Use this instead for guest access to the intended tenant:
az login --tenant "<tenant-id>"
az account list --all --query "[].{Name:name,SubscriptionID:id}" --output table
go build -o /tmp/devops-ai-agents ./cmd/devops-ai-agents
/tmp/devops-ai-agents health check --subscription "<subscription-id>"
# Inspect the shell's exit status immediately; repeat with the unique name:
/tmp/devops-ai-agents health check --subscription "<unique-subscription-name>"
# Run again and press Ctrl+C while collection is in progress:
/tmp/devops-ai-agents health check --subscription "<subscription-id>"
git check-ignore -v -- runs/<run-id>/run.json runs/<run-id>/report.md
git ls-files -- runs/
git status --short --untracked-files=all
```

For each run, compare run ID/revision between the two files and trace finding
`ev-...` references into `run.json`. Inspect states/timestamps against current
Resource Health evidence and review missing coverage. Do not create outages,
resources, role changes, or duplicate names to manufacture cases. Document absent
scenarios as not live-verified and cite the controlled tests above where relevant.

Guest-tenant login, unhealthy-resource cases, and adverse API failures beyond the
observed continuation rejection remain **synthetic-only**: the general manual
confirmation does not establish that these scenarios occurred in this private
subscription. No live issue findings were observed in the agent's runs, so live
finding-reference usability was not exercised there. Successful live collection
and safe retention after the rejected continuation were observed; source/report
review and interruption passed according to the operator's confirmation.
No real subscription evidence is committed, and the work remains uncommitted by
request. This acceptance completes S-01; overall four-source coverage is still
intentionally incomplete until later slices.

# S-02 verification and operator acceptance

Automated verification: PASS (2026-11-15). Operator acceptance: ACCEPTED
(2026-11-18). All Progress items 4.1–4.7 are complete. The operator confirmed:
"all tests are good, S-02 accepted" after receiving the manual checklist.

## Automated evidence

All application tests used synthetic identities and in-memory ARM transports.
Integration tests run actual discovery and subscription-bound credential
subprocesses against an isolated fake `az`; production collector defaults use
native SDK pipelines/pagers. Process signal tests use injected synthetic
credentials and native collectors. No test reads the operator's Azure session or
uses live ARM access. The agent has not performed a live S-02 run.

Commands ran from the repository with writable temporary caches:

```sh
export GOCACHE=/private/tmp/active-alert-report-cache/go-build
export GOMODCACHE=/private/tmp/active-alert-report-cache/go-mod
export STATICCHECK_CACHE=/private/tmp/active-alert-report-cache/staticcheck

go test -race -count=1 ./internal/cli ./cmd/devops-ai-agents
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

Every command exited 0. The targeted race run passed CLI and command packages
(39.326s and 1.704s). The final uncached full race/coverage run passed all six
real packages:

| Package | Duration | Statement coverage |
| --- | --- | --- |
| cmd/devops-ai-agents | 1.815s | 57.1% |
| internal/azure | 1.750s | 91.7% |
| internal/cli | 43.302s | 95.9% |
| internal/health | 2.315s | 97.4% |
| internal/report | 5.785s | 96.5% |
| internal/runstore | 2.103s | 82.3% |

Local tools: Go 1.27.4 darwin/arm64, Staticcheck 2026.2.2 (0.8.2),
govulncheck v1.9.0, actionlint 1.7.12, ShellCheck 0.11.0. The vulnerability audit
used network-enabled access to the public Go vulnerability database and found
zero reachable and zero imported-package vulnerabilities. Three advisories exist
in required modules outside called code. The Go directive remains 1.27.3 and
module dependencies are unchanged; these results describe the installed local
1.27.4 toolchain, not a GitHub CI execution on the directive's version.

The built binary at
`/private/tmp/active-alert-report-acceptance/devops-ai-agents` was produced with:

```sh
go build -o /private/tmp/active-alert-report-acceptance/devops-ai-agents ./cmd/devops-ai-agents
/private/tmp/active-alert-report-acceptance/devops-ai-agents --help
```

Both exited 0. Help caused no Azure access. Rebuild the binary if source changes
before manual acceptance.

## Scenario coverage

[Native full-flow tests](../../../internal/cli/operational_integration_test.go)
exercise the default command orchestration, fake CLI discovery/credentials,
validation, all four native collectors, normalization, checkpoint storage and
Markdown publication together:

- Three pages from every collector, 360 inventory resources, 360 Resource Health
  records, 12 operational records and seven active operational findings. All four
  Service Health categories and New/Acknowledged/Closed Fired alerts are present.
- A separate 24-page-per-source stress scenario. Each subsequent request observes
  the committed page count; operational query provenance is saved before reads.
  No timing SLA is inferred from test durations.
- Each new source failing, both new sources failing with old evidence retained,
  both old sources failing with operational evidence retained, all four failing,
  and all four successfully empty. HTTP 401 in Service Health remains a local
  failure and Monitor still runs. Exits are 0/3/1 according to execution results;
  successful empty enumeration still has incomplete coverage.
- After a committed page in either new source: 401/403/404, exhausted 429/timeout
  retries, foreign-host/subscription continuations, redirects, repeated links,
  malformed envelopes and mixed valid/malformed/foreign records. Other sources
  continue and the completed page survives; unsafe requests never reach the wire.
- Unknown domain facts, duplicates across pages, tied contradictions and uniquely
  ordered superseding updates. Unknown/conflicting facts remain traceable without
  active findings; valid enumeration can complete with coverage gaps.
- Native operational checkpoint failures stop further reads and retain the first
  page; final snapshot/publication failures retain useful evidence and disclose
  the saved snapshot without claiming a report.
- Subscription/service/region evidence does not become an invented resource
  fault. Missing/foreign Monitor targets remain unattributed; workspace targets
  remain explicit. Actual JSON, Markdown, stdout and debug stderr reject private
  source prose, query/results, usernames, tokens, error bodies and foreign scope
  canaries. Source markup remains literal.

[Process tests](../../../cmd/devops-ai-agents/main_test.go) synchronize through a
child-process marker emitted only after a native pager has returned and committed
a page, before interrupting the next request or retry delay. All four existing
modes run at inventory, Service Health and Monitor stages: first interruption,
retry interruption, failed final write, and second interrupt during blocked
finalization. Graceful cases exit 130 with a cancelled/incomplete report matching
the saved revision. Failed final writes disclose retained artifacts. Forced
termination preserves the last committed checkpoint and claims no report.
Decoded evidence remains unchanged, and later collectors stay unattempted. There are no synchronization sleeps. These tests run on macOS;
Windows skips POSIX fake-CLI and process-signal cases.

Existing health, Azure literal-fixture/read-boundary, report, store, output-delivery
and schema-1 regressions remain enabled. Synthetic compatibility is separate from
actual service compatibility and operator acceptance.

## Artifact and Git checks

Full-flow assertions verify the report's run ID/revision/outcome against committed
JSON, unique evidence IDs, finding/resource/observation/gap references, literal
source text, Advisor unchecked and overall INCOMPLETE coverage. Both checked-in
report examples have regression checks following their sibling snapshot links.
The schema-1 fixture bytes, go.mod and go.sum remain unchanged.

`TestGeneratedRunFilesAreIgnored` generates actual snapshot/report files in an
isolated Git repository using the project's real ignore rules. Both files are
ignored, absent from `ls-files` and ordinary status; source and synthetic fixture
paths remain eligible for tracking. Workspace checks also passed:

```sh
git check-ignore -v -- runs/synthetic-active-alert-check/run.json runs/synthetic-active-alert-check/report.md
git ls-files -- runs/
```

Both synthetic paths matched the root `/runs/` rule; `ls-files` returned no run
files. No existing run directory or archived document was modified. All changes
remain uncommitted; no staging, commit, upload or live identifiers were added.

## Manual procedure — accepted by the operator on 2026-11-18

Use the same authorized environment as S-01 with login established externally.
Run from the repository, keep live artifacts under ignored `runs/`, and record
only sanitized outcomes here. Do not create events, outages, alert rules or
permission failures to manufacture scenarios.

1. Run the built binary and inspect its actual exit status:

   ```sh
   /private/tmp/active-alert-report-acceptance/devops-ai-agents health check --subscription "<subscription-id>"
   printf 'Exit: %s\n' "$?"
   ```

   Repeat with the intended unique full subscription name if useful. Confirm the
   selected scope locally; do not paste its identifiers into tracked documents.
2. Open the printed report beside its sibling run.json. Check run ID, revision,
   source states and query/request times. Compare Service Health with all four
   categories and the saved exact update cutoff/API date. Portal filters may
   differ from that cutoff. Compare Monitor with a 30-day view and Fired across
   New/Acknowledged/Closed responses, preserving suppression as a separate fact.
3. Trace available findings through their saved evidence IDs and native source
   identities. Evaluate generated summaries, category/workflow separation,
   future notices, sensitive-detail limits, service/region uncertainty, missing
   resource attribution and workspace limitations. Confirm Advisor unchecked and
   no overall OK. Record absent examples as not live-verified.
4. Interrupt an in-progress run once. Record the stage actually reached, retained
   committed pages, actual exit, and whether cancelled/incomplete finalization
   succeeded. If new-source pages cannot naturally be interrupted, disclose that
   manual limitation; the synthetic process tests remain separate evidence.
5. Check the generated private paths locally:

   ```sh
   git check-ignore -v -- runs/<run-id>/run.json runs/<run-id>/report.md
   git ls-files -- runs/
   ```

6. Report sanitized pass/fail outcomes, unavailable scenarios and explicit
   acceptance or rejection. Do not supply raw payloads, real IDs or credentials.

## Operator result record

The operator confirmed on 2026-11-18: "all tests are good, S-02 accepted".
This closes the supplied manual checklist and the final acceptance gate.

| Progress item | Status | Confirmation |
| --- | --- | --- |
| 4.4 Scope and portal comparison | CONFIRMED | Operator's checklist confirmation and explicit S-02 acceptance |
| 4.5 Report usefulness and evidence | CONFIRMED | Operator's checklist confirmation and explicit S-02 acceptance |
| 4.6 Interruption | CONFIRMED | Operator's checklist confirmation and explicit S-02 acceptance |
| 4.7 Sanitized acceptance and private artifacts | CONFIRMED | Acceptance recorded here without live identifiers or payloads |

The operator supplied an overall confirmation rather than per-scenario logs.
Exact live exit codes, source counts, the interrupted stage and which category,
workflow, target or paging variants were available were not supplied. Those
details remain unspecified; this record does not claim every synthetic variant
was reproduced live. Earlier automated results retain their recorded date and
toolchain. No new automated or live Azure run was performed by the agent while
recording acceptance. The work remained uncommitted at that checkpoint.

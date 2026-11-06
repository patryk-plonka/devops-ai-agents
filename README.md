# Azure health checks

A local Go CLI for one manually selected public Azure subscription. Azure access
is read-only and uses the operator's existing Azure CLI user login.

**S-01 is implemented and accepted:** subscription selection,
live subscription access validation, independent inventory and Resource Health collection, checkpointed
evidence in `run.json`, and Markdown reports with graceful interruption.
Ticket drafts are not implemented.
Advisor, Service Health, and Azure Monitor are explicitly not checked; overall
coverage remains incomplete and no resource is classified overall OK.
The synthetic complete-flow suite covers scale and adverse scenarios; it does
not establish access to any real subscription. Acceptance status and the local
checklist are in [verification.md](context/changes/subscription-resource-health-report/verification.md).

Install the Go version in `go.mod` and Azure CLI. Sign in separately:

```sh
az login
# For guest access, use the intended client's tenant:
az login --tenant <tenant-id>
```

The CLI reads cached subscriptions without refreshing them or switching accounts.
If the target is missing from the session, sign in or refresh the session externally.
Only AzureCloud user accounts are supported. Live validation needs permission to
read the selected subscription's metadata, ARM resource inventory, and subscription
Resource Health availability statuses. The program never registers providers,
repairs permissions, or falls back to another identity.

Run from this repository or one of its subdirectories:

```sh
go run ./cmd/devops-ai-agents health check --subscription "<subscription-id-or-full-name>"
go run ./cmd/devops-ai-agents health check --subscription "<subscription-id>" --non-interactive
go run ./cmd/devops-ai-agents health check --subscription "<subscription-id>" --debug
go build -o /tmp/devops-ai-agents ./cmd/devops-ai-agents
/tmp/devops-ai-agents health check --subscription "<subscription-id>"
```

Names match in full without regard to case. Multiple matching subscriptions prompt
on terminal input, showing their names, IDs, and tenant IDs. With piped input or
`--non-interactive`, supply a UUID to resolve ambiguity. Blank input or EOF cancels
selection without choosing a default. `--help` performs no Azure or filesystem work.

Use `--debug` during development to print filtered Azure SDK request attempts,
response status, durations, and retry delays to stderr. SDK logging is disabled
by default, including when `AZURE_SDK_GO_LOGGING=all` is inherited. Debug output
excludes raw messages, response bodies, headers, URLs, and tokens; unrecognized SDK
log formats are omitted. It does not change the run outcome or saved record.

Each accepted invocation creates `runs/<UTC-timestamp>-<random-id>/run.json` before
Azure access. The directory is Git-ignored and owner-only on supported filesystems;
snapshots use atomic replacements. Failed scope validation is recorded as failed.
An unavailable filesystem prevents Azure access. No automatic deletion occurs.
Each successfully collected page is filtered and checkpointed before the next
request. A later failure retains earlier evidence; either collector can fail
without suppressing the other. Each page has a 60-second deadline including
authentication and up to three SDK retries. Collection stops at 10,000 pages per
source, repeated or unsafe continuations, cancellation, or a checkpoint failure.
Schema-invalid records are rejected individually with an explicit count and gap;
valid siblings on the page are retained. Invalid page envelopes fail the page.
After finalizing `run.json`, the CLI atomically publishes `report.md` from that
saved revision. Stdout contains only the published report path; stderr identifies
the outcome, snapshot, source status, and incomplete overall coverage. Failed
scope validation also produces a diagnostic report when storage is available.
Both artifacts carry the same run ID and revision. The two file replacements
are separate operations: if rendering/publication fails, stdout stays empty,
the command fails, and stderr identifies the retained checkpoint and whether
an older report exists. An older report is explicitly marked stale in that
diagnostic; compare its revision with `run.json`.
If stdout cannot accept the published path, the command returns 1 (130 if
interrupted) and stderr identifies the saved report; the artifacts remain intact.

```text
runs/                              # Git-ignored; private local output
  <UTC-timestamp>-<random-id>/
    run.json                       # normalized evidence and coverage
    report.md                      # derived from the saved revision
```

The normalized record retains the requested scope and resolved subscription/tenant
identifiers, with defined credential patterns redacted and terminal controls removed.
It retains resource IDs/names/types/locations and ordinary tags, redacting values
for credential-like keys and recognized credential patterns. It excludes account
usernames, tokens, full CLI profiles, arbitrary resource properties/source prose,
and raw SDK errors. Do not commit local run records or put real subscription data
in test fixtures. Credential-bearing URLs are filtered across URI schemes,
including Redis and PostgreSQL connection URLs.

Inspect `resources`, `evidence`, `observations`, `findings`, and `coverage_gaps` in
`run.json`. Findings reference stable evidence IDs with source/API version and
collection/source timestamps. Exact duplicate observations collapse; conflicting
states remain undetermined unless source timestamps identify a unique current
state. Available is a Resource Health observation only.
Historical statuses remain evidence with a `historical` status role; only
`current` statuses determine current health and findings. Historical evidence
alone leaves current health undetermined with an explicit coverage gap.
Unknown, event-only, missing, invalid, and inaccessible evidence remain explicit
gaps. Inventory completion describes ARM resource enumeration, without promising every child
resource is listed; health observations absent from inventory remain visible.
Empty results mean no resources observed. Defined text filtering does not certify
arbitrary free text as secret-free.

The report separates source findings, current Resource Health issue states,
Available observations, unknown/not-checked resources, and failed checks. All
observed inventory resources and evidence records remain enumerable, with sorted
tags and links to the sibling `run.json`; untrusted strings render as literal
text. A finding retains its source observation even if conflicting or newer
evidence changes the current state. Use the observation tables and source times
to interpret it.
Open `report.md` beside `run.json`. Check their run ID/revision first, then find
each finding's `ev-...` reference in the JSON `evidence` array. Use `resource_id`,
`status_id`, source/API version, and source/collection timestamps to locate and
compare the observation in Azure. Keep the files together so sibling links work.
For a synthetic example, see [mixed-report.md](internal/report/testdata/mixed-report.md)
and its sibling [run.json](internal/report/testdata/run.json).

The first Ctrl+C stops discovery/collection and finalizes a cancelled, incomplete
report using local storage, retaining completed pages. Selection prompts also
respond to interruption. No new Azure evidence is requested during finalization.
A second Ctrl+C may force termination; the last committed `run.json` checkpoint
is the recovery artifact. Forced termination or an unwritable filesystem can
prevent the final report. The operator confirmed the manual checks, including
interruption and report acceptance, on 2026-11-06; the verification record retains
the limits of that live coverage.

Exit codes from the built binary:

| Code | Meaning |
| --- | --- |
| 0 | Help, or both collectors completed and the report was saved; overall coverage is still incomplete |
| 1 | Authentication, live scope-access, execution/storage failure, or both collectors failed without useful evidence |
| 2 | Invalid input, missing/ambiguous CLI-session subscription, unsupported cloud/account, or missing repository context |
| 3 | Partial collection with useful evidence or a completed collector retained |
| 130 | Interrupted execution; cancelled/incomplete output is saved when finalization succeeds; any storage failure is also disclosed |

`go run` itself returns a wrapper error for nonzero application codes; use the built
binary when inspecting exact exit codes.

Development checks and exact tool pins are in
[development-tools.md](context/foundation/development-tools.md). CI uses synthetic
Azure CLI and HTTP responses, with no Azure credentials or live run uploads.

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

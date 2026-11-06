# Implementation review fixes

## F1 — SDK logging bypasses diagnostic sanitization

Source: [phase 1 implementation review](../reviews/impl-review-phase-1.md).

Decision on 2026-11-06: fix using the operator's preferred `--debug` flag, keeping
useful development diagnostics within the existing privacy contract.

Status: implemented and verified. The CLI disables inherited SDK logging before
Azure operations. `--debug` enables filtered request attempts, HTTP status,
response timing, fixed outcomes, and retry delays on stderr. Raw messages and
unrecognized formats are excluded. Listener configuration is cleared after the
run. No changes were made to the approved plan's Progress rows.

Files: `internal/azure/logging.go`, `internal/azure/logging_test.go`,
`internal/cli/cli.go`, `internal/cli/debug_test.go`, and `README.md`.

Fresh-process regression cases reproduce SDK initialization from environment
settings and verify both quiet and debug modes using synthetic credentials and
HTTP responses. They prove raw response-body, header, and token canaries do not
reach process output. Targeted package tests, build, race/coverage, vet,
staticcheck, govulncheck, formatting, ShellCheck, and actionlint all pass. Full
results and the preserved original reproduction are in the linked review.

No pending fixes, commits, or live acceptance claims result from this triage.


## Phase 2 — F1, F2 and F3

Source: [phase 2 implementation review](../reviews/impl-review-phase-2.md).

Decision on 2026-11-06: the operator requested applying all three fixes after
combined triage. Status: all implemented and verified; zero pending decisions.

- **F1 — Credential URLs:** recognize userinfo-bearing URLs across URI schemes,
  including Redis/PostgreSQL, before and after control stripping. Normal text,
  tag values, and saved JSON regression cases pass.
- **F2 — Malformed records:** validate list envelopes and individual SDK record
  shapes below the native pager's payload cache. Replace malformed entries with
  null placeholders, then use existing normalization to count rejections and
  preserve valid siblings. Malformed envelopes remain non-retriable page failures.
  Real pager/runstore tests prove page checkpoints precede further reads and
  survive later retries or unsafe continuations.
- **F3 — Historical states:** save explicit current/historical roles, retain history
  as evidence, and use only current statuses for current-state selection and
  findings. Historical-only results remain undetermined with a missing-current
  coverage gap; evidence references remain intact.

Files: `internal/health/privacy.go`, `privacy_test.go`, `evidence.go`,
`normalize.go`, `normalize_test.go`; `internal/azure/paging.go`,
`record_decode.go`, `record_decode_test.go`; `README.md`; the linked review.

The review retains the original findings/reproductions and records the passing
regressions, build, race/coverage, vet, staticcheck, govulncheck, formatting,
ShellCheck and actionlint results. No plan Progress changes, commits, or live
acceptance claims were made.


## Phase 3 — F1

Source: [phase 3 implementation review](../reviews/impl-review-phase-3.md).

Decision on 2026-11-06: the operator selected fixing F1 during individual triage.
Status: implemented and verified. F2 was subsequently selected and fixed below.

Both finalization write paths now recheck the cancellation context before
returning a storage-failure exit code. An interruption arriving during either
write returns 130 and discloses incomplete finalization while identifying the
retained checkpoint. The stdout-delivery finding is unchanged.

Files: `internal/cli/cli.go`, `internal/cli/report_test.go`, and the linked review.
Permanent regressions reproduced the error before the fix and verify cancellation
inside a failed final snapshot write and inside failed/successful publication-
diagnostic writes, with retained evidence and byte-identical prior checkpoints
when writes fail. Build, race/coverage tests, vet, staticcheck, govulncheck,
formatting, ShellCheck, and actionlint pass. No plan Progress changes, commits,
or live acceptance claims were made.


## Phase 3 — F2

Source: [phase 3 implementation review](../reviews/impl-review-phase-3.md).

Decision on 2026-11-06: the operator selected fixing F2 during individual triage.
Status: implemented and verified. Phase 3 triage is complete: F1 and F2 are fixed.

The final report-path stdout write now handles I/O errors, emits sanitized stderr
feedback identifying the already saved report, and returns 1 (130 if interrupted).
Matching JSON and Markdown artifacts remain unchanged; delivery failure does not
invalidate publication or expose raw writer errors.

Files: `internal/cli/cli.go`, `internal/cli/report_test.go`, and the linked review.
Permanent regressions cover completed/partial runs, cancellation before/during
output delivery, and partial writes with errors. All five cases failed before
the fix and pass afterward, proving byte-identical artifacts and retained
run/revision/evidence references. Build, race/coverage tests, vet, staticcheck,
govulncheck, formatting, ShellCheck, and actionlint pass. No plan Progress changes,
commits, or live acceptance claims were made.


## Phase 4 — F1

Source: [phase 4 implementation review](../reviews/impl-review-phase-4.md).

Decision on 2026-11-06: the operator requested fixing F1. Status: implemented and
verified; no pending phase 4 finding decisions remain.

The integration privacy assertion reads the committed `run.json` bytes directly
instead of re-marshaling a decoded `health.Run`. Unknown JSON fields are included
in the canary scan alongside Markdown and stderr; typed semantic assertions remain.
The controlled probe that previously passed with an inserted raw JSON canary now
fails at the expected privacy assertion for both ID and name runs. The normal
uncached race/coverage suite passes, as do build, vet, staticcheck, govulncheck,
formatting, ShellCheck, and actionlint.

Files: `internal/cli/integration_test.go`, the linked review, and `change.md`.
Production behavior, plan Progress, and live acceptance remain unchanged. No
staging or commits were performed.


## Live preflight — Azure CLI account schema

Source: the operator's failed manual UUID/name attempts on 2026-11-06, recorded
without subscription details in [verification.md](../verification.md).

Status: corrected and verified; live acceptance rerun remains pending.
`az account list` emits `cloudName`, but `Account.Environment` decoded
`environmentName`, incorrectly rejecting a public-cloud user account. Both the
production JSON tag and the synthetic CLI fixture now use `cloudName`.

`TestDiscoverDecodesAzureCLIAccountSchema` uses literal CLI-shaped JSON independent
of the application's serialization tags. It reproduced the rejection before the
fix and now passes for UUID/name selection, while retaining rejection of missing
cloud metadata, sovereign clouds, and service-principal accounts. The full
uncached race/coverage suite, build, vet, staticcheck, govulncheck, formatting,
ShellCheck, and actionlint pass. The executable at `/tmp/devops-ai-agents` was
rebuilt. The independently observed UUID mismatch requires the operator to verify
their intended scope; no alternate subscription was selected automatically.

Files: `internal/azure/session.go`, `internal/azure/session_test.go`,
`internal/cli/testdata/fake-az.sh`, `verification.md`, and `change.md`.
No live ARM collection, manual Progress changes, staging, or commits were performed.


## Live collection — Resource Health continuation shape

Source: the operator's partial run on 2026-11-06; sanitized results and reruns are
recorded in [verification.md](../verification.md).

Status: corrected and verified. Resource Health returned lowercase `$skiptoken`
and `$top` in a valid same-operation/public-ARM continuation. The previous query
allowlist incorrectly rejected that shape. Both token spellings are now accepted
for Resource Health, with at most one nonempty token and one positive 31-bit
page size. Token aliases share a repetition key. Unknown query parameters,
ambiguous/duplicate token keys, invalid page sizes, wrong scope/API/host, and
mutations remain rejected. Inventory's query allowance is unchanged.

The independent service-shaped boundary regression failed before the correction
and passes afterward. Full CLI fixtures now use the observed pagination shape.
The full uncached race/coverage suite, build, vet, staticcheck, govulncheck,
formatting, ShellCheck, and actionlint pass. The rebuilt executable completed
live runs by resolved name and UUID with exit 0, two Resource Health pages,
matching scope/revisions, distinct directories, and ignored artifacts.

Files: `internal/azure/transport.go`, `internal/azure/paging.go`,
`internal/azure/transport_test.go`, `internal/cli/integration_test.go`,
`verification.md`, and `change.md`.
No real evidence/tokens were copied into tracked documents. Operator source
comparison, report acceptance, and live interruption remain pending; no manual
Progress changes, staging, or commits were performed.


## Operator acceptance — 2026-11-06

The operator confirmed "all manual tests are ok" after the live corrections and
tenant/subscription ID clarification. All manual Progress rows are now complete;
the plan is 18/18 and the change is marked `implemented`. Sanitized acceptance
results and scenario limits are recorded in `verification.md`. No pending review
fixes remain. Work stays uncommitted by request; no phase or epilogue commit was
created.


## Full S-01 review — F1 and F2

Source: [full implementation review](../reviews/impl-review.md).

Decision on 2026-11-06: the operator requested "fix both". Status: implemented
and verified; no pending full-review finding decisions remain.

F1: after successful report-path delivery, finalization rechecks cancellation
and returns 130 with accurate saved-report feedback. Previously published
JSON/Markdown remain unchanged, including their matching run ID and revision.
The existing delivery regression now covers completed and partial runs cancelled
inside a successful writer, alongside the prior failed-write scenarios.

F2: `ReportExists` recognizes regular files only. Non-regular entries return an
inspection error without following symlinks, so the existing CLI indeterminate-
report message replaces the incorrect older-report claim for a directory.
Regressions cover missing/regular/directory/symlink/dangling-symlink entries and
real publication failure with a report-path directory and retained JSON diagnostic.

The new regressions failed before the fixes: successful writes returned 0/3
instead of 130, directory-obstructed publication claimed an older report, and
non-regular entries were classified as reports. All pass after the fixes.

Post-fix checks: `go build ./...`, `go test -race -cover -count=1 ./...`,
`go vet ./...`, `staticcheck ./...`, `govulncheck ./...`,
`./scripts/check-format.sh`,
`shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh`, and
`actionlint` all pass. CLI coverage is 93.8%, runstore 82.3%; all six packages
pass without races. The vulnerability scan required a permitted network retry
after sandbox DNS failure; zero called-code/imported-package vulnerabilities
were found, with three advisories in required modules outside called code.

Files: `internal/cli/{cli.go,report_test.go}`,
`internal/runstore/{store.go,store_test.go}`, the linked full review, and
`change.md`. Plan Progress stays 18/18; accepted manual outcomes remain intact.
No live Azure calls, staging, commits, or archive changes were performed.

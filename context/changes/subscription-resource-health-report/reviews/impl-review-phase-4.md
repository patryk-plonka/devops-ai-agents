<!-- IMPL-REVIEW-REPORT -->
# Implementation Review: Subscription Resource Health Report

- **Plan**: context/changes/subscription-resource-health-report/plan.md
- **Scope**: Phase 4 of 4 — automated complete-flow implementation and documentation; live acceptance was pending at review time
- **Date**: 2026-11-06
- **Verdict**: APPROVED
- **Acceptance**: F1 resolved; operator confirmed all manual checks on 2026-11-06 after this review
- **Findings**: 0 critical, 0 warnings, 1 observation

## Verdicts

| Dimension | Verdict |
|-----------|---------|
| Plan Adherence | PASS |
| Scope Discipline | PASS |
| Safety & Quality | PASS |
| Architecture | PASS |
| Pattern Consistency | PASS |
| Success Criteria | PASS — F1 resolved; automated commands pass; manual checks subsequently confirmed by the operator |

## Findings

### F1 — Privacy assertion omits unknown fields in saved JSON

- **Severity**: ℹ️ OBSERVATION
- **Impact**: 🏃 LOW — quick decision; fix is obvious and narrowly scoped
- **Dimension**: Success Criteria
- **Location**: internal/cli/integration_test.go:162
- **Detail**: `assertIntegrationArtifacts` checks canaries in `json.Marshal(r)`, where `r` was decoded from the saved snapshot into `health.Run`. Decoding silently drops unknown JSON fields, so the assertion does not inspect all bytes actually persisted in `run.json`. A review-only overlay appended `"raw_response":"token-canary"` to the completed snapshot before this helper ran; both scale subtests still passed. This is a limited regression-test blind spot, not a demonstrated production leak: unchanged `Store.Save` serializes the typed `Run` with `json.MarshalIndent`. The phase's privacy verification should inspect the committed artifact itself.
- **Fix**: Read the sibling committed `run.json` bytes and scan them alongside Markdown and stderr; retain the typed snapshot for semantic assertions.
- **Decision**: FIXED — scan the committed snapshot bytes directly, verified on 2026-11-06.

## Scope and plan comparison

The repository has no commits (`git log` exits 128: current branch has no commits),
and all implementation files remain untracked by the operator's instruction.
An empty `git diff` is not evidence of an empty implementation. Phase scope was
reconstructed from the immediately preceding implementation's touched-file set:

- `internal/cli/integration_test.go`
- `internal/cli/testdata/fake-az.sh`
- `README.md`
- `AGENTS.md`
- `context/foundation/development-tools.md`
- `.github/workflows/checks.yml`
- `context/changes/subscription-resource-health-report/verification.md`
- `context/changes/subscription-resource-health-report/change.md`
- `context/changes/subscription-resource-health-report/plan.md`, Progress only

All planned phase 4 changes match intent. Generating synthetic pages in Go and
keeping the fake CLI under CLI `testdata/` are justified fixture choices; no
additional Azure/health fixture directory is required. Existing process, report,
and transport tests provide the signal/read-boundary checks described in the
plan. The previously approved `--debug` flag is baseline behavior, not phase 4
scope creep.

Two independent reviewers checked plan adherence and safety/pattern compliance.
Both found no substantive production, architecture, or scope defect. Comparisons
included existing CLI/report tests, Azure credential/collection tests, and the
process signal suite. No lessons register exists. No live account was accessed.

## Automated verification

Commands ran against real packages with writable temporary `GOCACHE`,
`GOMODCACHE`, and `STATICCHECK_CACHE` directories as appropriate.

| Command | Result | Actual output / evidence |
| --- | --- | --- |
| `go test -race -cover -count=1 ./...` | PASS | All six packages `ok`; process 57.1%, Azure 87.3%, CLI 93.1%, health 94.2%, report 98.6%, runstore 81.5%; no races. This includes the required uncached full synthetic suite. |
| `go build ./...` | PASS | Exit 0, no output. |
| `go vet ./...` | PASS | Exit 0, no output. |
| `staticcheck ./...` | PASS | Exit 0, no output. |
| `govulncheck ./...` | PASS after permitted network retry | Zero vulnerabilities affecting code; zero imported-package vulnerabilities; three advisories in required modules outside called code. Initial sandbox attempt could not resolve `vuln.go.dev`; it was not counted as a successful scan. |
| `./scripts/check-format.sh` | PASS | Exit 0, no output. |
| `shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh` | PASS | Exit 0, no output. |
| `actionlint` | PASS | Exit 0, no output. |
| `git check-ignore -v -- runs/phase4-synthetic-check/run.json runs/phase4-synthetic-check/report.md` | PASS | Both paths match `.gitignore:1:/runs/`. These are synthetic path probes, not live artifacts. |
| `git ls-files -- runs/` | PASS | Exit 0, empty output; no run artifacts tracked. |
| `git check-ignore --no-index --` with README, AGENTS, integration source, fake CLI, report fixtures, and verification document paths | PASS | Expected exit 1, empty output: none of these source/documentation paths is ignored. |
| `TestGeneratedRunFilesAreIgnored` in the full suite | PASS | Real synthetic JSON/report files generated in an isolated Git repository using this project's ignore rules; absent from ordinary status and tracked files; no workspace index changes. |

Synthetic full-flow results include 360 inventory resources, 359 health records,
719 unique evidence records, six issue/event findings, three pages per collector,
matching saved revisions, finding references, ID/name selection of a non-default
guest target, explicit gaps, independent failures, bounded retries, rejected
continuations/records, and retained checkpoints. The test boundaries do not add
production endpoints or token flags. CI matches local checks and has no Azure
credentials, live uploads, or releases. A GitHub-hosted run is not claimed.

## Finding probe

The independent safety reviewer created an isolated Go overlay, without changing
repository source. The primary reviewer inspected and reproduced it:

```sh
go test -overlay=/tmp/phase4-privacy-review-h1t0b9ri/overlay.json \
  -count=1 -run '^TestFullFlowScaleByIDAndName$' ./internal/cli
```

Original result: PASS for both UUID/name subtests despite the raw committed JSON canary.
The overlay changes only a temporary test copy to append the unknown field after
the normal run and before the artifact assertion. Test-generated artifacts are
temporary and synthetic. This demonstrates the assertion gap only; it does not
establish that production storage currently leaks raw source fields.

## Manual verification and limits

At review time, Progress was **14/18 complete**. Automated rows 4.1–4.3 were
checked and manual rows 4.4–4.7 were unchecked; none was rubber-stamped:

- 4.4: private-subscription runs by ID/name and representative live Resource Health comparison.
- 4.5: operator acceptance of report, evidence references, retained identifiers/tags, and local layout.
- 4.6: live graceful Ctrl+C acceptance or an explicitly recorded live limitation.
- 4.7: sanitized acceptance results/limitations and Git exclusion; no untested guest-access claim.

At review time, no target or operator confirmation had been supplied, and
`verification.md` labeled guest access, unhealthy-resource cases, adverse API
failures, and live interruption synthetic-only. This review approved the
implemented automated work after F1 resolution, without declaring live acceptance.
The initial review made no application/test fixes, Progress edits, staging, or
commits. F1 triage is complete; subsequent live acceptance is recorded below.

## F1 resolution

On 2026-11-06, the operator requested fixing F1. The artifact assertion now reads
the sibling committed `run.json` bytes and checks them alongside decoded Markdown
and stderr. The typed snapshot remains available for all semantic assertions;
unknown JSON fields can no longer disappear before the canary check. No production
code or output behavior changed.

The same review-only mutation was reapplied to the updated test through
`/tmp/phase4-f1-fixed-jj6sdndr/overlay.json`. Both UUID/name scale subtests now fail
with `sensitive source/profile/token/error escaped into saved output`, as expected.
The probe harness confirms both failures are the privacy assertion, rather than
a compile or unrelated execution error. The normal uncached race/coverage suite
passes across all six packages; CLI coverage remains 93.1%.

Build, vet, staticcheck, formatting, ShellCheck, and actionlint also pass.
Govulncheck passed after the permitted network retry, reporting zero reachable
or imported-package vulnerabilities and three required-module advisories outside
called code. All finding decisions are complete: F1 FIXED, zero pending.
Progress rows, manual acceptance, staging, and commits were not changed.

## Subsequent operator acceptance — 2026-11-06

After live validation and its recorded schema/pagination corrections, the
operator confirmed "all manual tests are ok". The implement workflow marked
manual rows 4.4–4.7 complete on that confirmation; the plan is now 18/18 and
`change.md` is `implemented`. This is a later acceptance update, not a new code
review. See `verification.md` for observed runs, operator-reported results, and
scenarios that remain synthetic-only. Changes remain uncommitted by request.

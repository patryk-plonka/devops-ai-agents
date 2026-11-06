# Local development tools

Installed and verified on 2026-11-02 on macOS Apple Silicon using Homebrew.
Executables are available through the existing `/opt/homebrew/bin` PATH entry.

| Tool | Installed version | Purpose |
| --- | --- | --- |
| Go | 1.27.3 | Build, test, formatting, coverage, race detection, and vet |
| govulncheck | 1.9.0 | Check reachable known Go vulnerabilities |
| staticcheck | 2026.2.2 (0.8.2) | Additional Go static analysis |
| gopls | 0.24.0 | Go language server for editor integration |
| Delve (`dlv`) | 1.27.2 | Go debugger |
| Azure CLI (`az`) | 2.91.0 | Interactive work-identity login and Azure CLI access |
| GitHub CLI (`gh`) | 2.103.3 | Repository, pull request, and workflow operations |
| actionlint | 1.7.12 | GitHub Actions workflow validation |
| ShellCheck | 0.11.0 | Shell-script analysis; installed as an actionlint dependency |

Git was already available at `/usr/bin/git`. No additional test framework is
needed for the initial Go CLI. Editor integration may require configuring the
editor to use gopls and Delve; installing the tools does not configure an editor.

## Installation and maintenance

To install the same tool set on a Mac with Homebrew:

```sh
brew install go govulncheck staticcheck gopls delve gh actionlint shellcheck
brew install --cask azure-cli
```

These commands install the versions currently supplied by Homebrew, not pinned
copies of the versions above. Update selected tools with `brew upgrade <name>`.

## Authentication

When ready to connect your accounts, run these interactively:

```sh
az login
gh auth login
```

For a client tenant, use `az login --tenant <tenant-id>` as appropriate.
Installation did not sign in, select a subscription, or change cloud resources.
The CLI restricts ARM requests to the selected subscription's approved GET routes
and disables automatic provider registration.

## Pinned checks

The repository has real packages and `.github/workflows/checks.yml`. CI runs on pull
requests and pushes to main, with `contents: read`, no Azure credentials, and no
report uploads. `setup-go` reads the Go directive in `go.mod` (currently 1.27.3).

| Component | CI pin |
| --- | --- |
| actions/checkout v5.0.0 | `08c6903cd8c0fde910a37f88322edcfb5dd907a8` |
| actions/setup-go v6.0.0 | `44694675825211faa026b3c33043df3e48a5fa00` |
| staticcheck 2026.2.2 | `honnef.co/go/tools/cmd/staticcheck@v0.8.2` |
| govulncheck 1.9.0 | `golang.org/x/vuln/cmd/govulncheck@v1.9.0` |
| actionlint 1.7.12 | `github.com/rhysd/actionlint/cmd/actionlint@v1.7.12` |
| ShellCheck | Ubuntu 24.04 package; local validation uses 0.11.0 |

The checkout and setup-go SHA pins were verified against their official release
tags. Tool source and action contracts: [checkout](https://github.com/actions/checkout/tree/v5.0.0),
[setup-go](https://github.com/actions/setup-go/tree/v6.0.0),
[staticcheck](https://github.com/dominikh/go-tools/tree/v0.8.2),
[govulncheck](https://github.com/golang/vuln/tree/v1.9.0), and
[actionlint](https://github.com/rhysd/actionlint/tree/v1.7.12).

To install matching Go analysis tools locally:

```sh
go install honnef.co/go/tools/cmd/staticcheck@v0.8.2
go install golang.org/x/vuln/cmd/govulncheck@v1.9.0
go install github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
```

Selected application SDKs are `azidentity v1.15.0`, `armsubscriptions/v2 v2.0.0`,
`armresources/v4 v4.0.0`, `armresourcehealth v1.4.0`, and compatible
`azcore v1.23.3`; `go.mod` and `go.sum` record exact resolved
versions and checksums. Tests use synthetic accounts, a fake Azure CLI process,
and fake HTTP transports without reading the operator's real session.

The first source audit detected five reachable HTTP/2 advisories in the SDK's
initial `golang.org/x/net v0.58.0` selection. Phase 1 updates it to the reported
fixed `v0.60.0` and accepts its compatible `x/crypto`, `x/sync`, `x/sys`, `x/term`,
and `x/text` requirements. The SDK selections and Go directive remain unchanged.

```sh
./scripts/check-format.sh
go vet ./...
go test -race -cover -count=1 ./...
go build ./...
staticcheck ./...
govulncheck ./...
actionlint
shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh
```

The formatting script reads all Go files under `cmd/` and `internal/`, including
untracked files, and fails without modifying them. Generated `runs/` data is
excluded. To apply formatting deliberately, use `gofmt -w cmd internal`.

If the agent sandbox blocks user caches, set writable temporary `GOCACHE` and
`STATICCHECK_CACHE` paths. Set `GOMODCACHE` to a writable temporary directory when
resolving modules under that sandbox. Network access is needed for dependency
downloads and the vulnerability database. Empty package sets never establish a
passing test or audit.

## Phase 2 verification

On 2026-11-06, collection/privacy tests, build, race/coverage tests, vet,
staticcheck, govulncheck, formatting, ShellCheck, and actionlint passed. The
vulnerability scan found zero reachable vulnerabilities and three module
advisories outside called code. Synthetic tests cover native pagers, bounded
retries/deadlines, unsafe continuations and redirects, independent source failures,
committed page checkpoints, invalid scope, health conflicts, and tag filtering.
No live subscription was accessed; report and live acceptance work remain pending.

## Phase 3 verification

On 2026-11-06, report/orchestration tests, build, race/coverage tests, vet,
staticcheck, govulncheck, formatting, ShellCheck, and actionlint passed. The
vulnerability scan reported zero reachable or imported-package vulnerabilities
and three advisories in required modules outside called code. Coverage was
98.6% for report, 90.6% for CLI, 94.2% for health, 81.5% for runstore, 87.3% for
Azure, and 57.1% for the process entry package.

Synthetic tests check literal Markdown/tag rendering, complete enumeration of
350 inventory resources, resolvable evidence references, matching saved revisions,
render/JSON/report write failures, and explicit stale-report diagnostics. Process
tests synchronize after a committed page and exercise first Ctrl+C during the next
native SDK request or retry delay, cancellation with a failed final write, and a
second interrupt during blocked finalization. Graceful cases return 130 without
further Azure calls; forced termination retains the last checkpoint. No live
subscription was accessed. Phase 4 integration and operator acceptance are pending.

## Complete-flow checks

Local and CI checks above match, including uncached race/coverage tests and
ShellCheck for the synthetic Azure CLI script. `internal/cli/integration_test.go`
uses real session discovery, Azure CLI credential subprocesses, native ARM SDK
pagers, health normalization/orchestration, local storage, and Markdown rendering.
The strict synthetic `az` only supports discovery and subscription-bound token
requests; it cannot log in, switch accounts, or refresh the session. ARM responses
are in memory and any fallback to the default HTTP transport fails locally.

The suite covers 360 inventory resources, 359 health observations over three
pages per source, six issue/event findings, evidence references and saved
revisions, name/ID selection, independent failures, retry exhaustion, unsafe
continuations, invalid records, and storage failures. Existing process tests
exercise synchronized interrupts after a checkpoint. An isolated Git repository
checks the project's actual ignore rules against generated private artifacts
and eligible synthetic fixtures without changing the workspace index.

On 2026-11-06, build, uncached race/coverage tests, vet, staticcheck,
govulncheck, formatting, ShellCheck, and actionlint passed. The vulnerability scan
reported zero reachable or imported-package vulnerabilities and three advisories
in required modules outside called code. Subsequent live acceptance exposed and
corrected CLI-schema and Resource Health continuation assumptions; both now have
independent regressions. Successful live name/UUID runs followed, and the operator
confirmed all manual checks on 2026-11-06. CI has not been executed on GitHub.
Live outcomes and scenario limits are in
[the change verification record](../changes/subscription-resource-health-report/verification.md);
real subscription evidence remains in ignored local storage.

## Historical bootstrap verification

Phase 1 source verification on 2026-11-05 passed build, package tests,
race/coverage tests, vet, staticcheck, govulncheck, the formatting script,
ShellCheck, and actionlint. The final vulnerability scan reported zero reachable
vulnerabilities and three module advisories outside called code. Built-binary
help/usage checks and formatting checks with untracked paths containing spaces
also passed. Azure behavior was tested with synthetic inputs; no live access or
phase 4 acceptance is claimed. CI has been validated locally, not executed on
GitHub yet.

All tool version commands succeeded. Azure CLI verification used a temporary
configuration directory without accessing account credentials.

The first `govulncheck -json ./...` attempt could not create the default Go build
cache under the agent's filesystem sandbox. Retrying with a temporary `GOCACHE`
started the scanner successfully and returned exit code 2 with
`no packages matched the provided patterns`. There is no Go source to scan yet;
this is not a completed vulnerability audit or a zero-findings result. The
earlier bootstrap logs remain unchanged as historical records.

## References

- [Go vulnerability tooling](https://go.dev/doc/security/vuln/)
- [Staticcheck installation and usage](https://staticcheck.dev/docs/getting-started/)
- [Azure CLI installation on macOS](https://learn.microsoft.com/en-us/cli/azure/install-azure-cli-macos)

---
bootstrapped_at: 2026-11-02T17:05:44+01:00
starter_id: go
starter_name: "Go (standard library)"
project_name: devops-ai-agents
language_family: go
package_manager: go-modules
cwd_strategy: subdir-then-move
bootstrapper_confidence: first-class
phase_3_status: ok
audit_command: "govulncheck ./..."
---

## Hand-off

```markdown
---
starter_id: go
project_name: devops-ai-agents
hints:
  language_family: go
  team_size: solo
  deployment_target: self-host
  ci_provider: github-actions
  ci_default_flow: manual-promotion
  bootstrapper_confidence: first-class
  path_taken: standard
  quality_override: false
  self_check_answers: null
  has_auth: true
  has_payments: false
  has_realtime: false
  has_ai: false
  has_background_jobs: false
---

## Why this stack

A solo consultant building a manually invoked CLI over eight weeks after hours chose the recommended Go standard-library starter with Go modules. It satisfies all four registry quality gates and supports the local, one-subscription-at-a-time workflow. The self-host value means execution on the consultant's own computer; no hosted service or container is required. Existing-work-identity authentication and read-only integrations for Azure Advisor, Service Health, Resource Health, and Azure Monitor will be added during implementation. Reports and explicitly selected ticket or recommendation drafts are Markdown, with traceable evidence and explicit coverage gaps. The current PRD does not require an LLM, payments, realtime updates, or scheduled jobs, so those feature flags are false. GitHub Actions will run automated checks, with releases triggered explicitly by the user. The starter is registered but has not been verified end to end by the bootstrapper.
```

## Pre-scaffold verification

| Signal | Value | Severity | Notes |
| --- | --- | --- | --- |
| npm package | not run | not applicable | Go starter; no npm package. |
| GitHub repo | not run | not applicable | Registry documentation URL is https://go.dev/doc/, not a GitHub repository URL. No recency signal available. |

Go is available: `go version go1.27.3 darwin/arm64`. No existing application scaffold was detected. The saved hand-off matches the confirmed stack and explicit retry request. The previous failure left an empty temporary directory; it was removed with `rmdir` before retrying.

## Scaffold log

**Registry template**: `mkdir {name} && cd {name} && go mod init github.com/user/{name}`
**Resolved invocation**: `mkdir .bootstrap-scaffold && cd .bootstrap-scaffold && go mod init github.com/patryk-plonka/devops-ai-agents`
**Strategy**: subdir-then-move
**Exit code**: 0
**Files moved**: 1
**Conflicts (.scaffold siblings)**: none
**.gitignore handling**: absent in scaffold
**.bootstrap-scaffold cleanup**: deleted

The module identity was corrected from the template's placeholder owner and temporary directory name to the existing origin remote, `https://github.com/patryk-plonka/devops-ai-agents.git`. The registry and saved hand-off were not edited.

**CLI stdout**: empty
**CLI stderr**:

```text
go: creating new go.mod: module github.com/patryk-plonka/devops-ai-agents
```

| Source | Destination | Result |
| --- | --- | --- |
| .bootstrap-scaffold/go.mod | go.mod | Moved; no existing destination. |

**Verification**: `go mod edit -json` succeeded and reported module `github.com/patryk-plonka/devops-ai-agents`, Go `1.27.3`, with no dependencies. `git diff --check` exited 0. SHA-256 comparisons confirmed all seven pre-existing context files were unchanged, including the earlier failure log. The generated module contains no source packages, so no application build or tests were run.

## Post-scaffold audit

**Configured tool**: `govulncheck ./...`
**Actual invocation**: `govulncheck -json ./...`
**Status**: failed to run
**Exit code**: 127
**Reason**: audit tool unavailable
**Stdout**: empty
**Stderr**:

```text
zsh:1: command not found: govulncheck
```

Findings and severity counts are unavailable; this is not a clean security-audit result.

## Hints recorded but not acted on

| Hint | Value |
| --- | --- |
| team_size | solo |
| deployment_target | self-host |
| ci_provider | github-actions |
| ci_default_flow | manual-promotion |
| bootstrapper_confidence | first-class |
| path_taken | standard |
| quality_override | false |
| self_check_answers | null |
| has_auth | true |
| has_payments | false |
| has_realtime | false |
| has_ai | false |
| has_background_jobs | false |

## Next steps

The Go module is initialized. Implement the CLI and Azure integrations through the planning workflow. Agent instructions and GitHub Actions workflows remain separate setup tasks. Install and run govulncheck when preparing executable source for verification.

This log is saved as `verification-v2.md` to preserve `verification.md`, which records the earlier failed attempt. No commits or remote changes were made.

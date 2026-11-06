---
bootstrapped_at: 2026-11-01T19:25:10+01:00
starter_id: go
starter_name: "Go (standard library)"
project_name: devops-ai-agents
language_family: go
package_manager: go-modules
cwd_strategy: subdir-then-move
bootstrapper_confidence: first-class
phase_3_status: failed
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
| GitHub repo | not run | not applicable | Registry docs URL is https://go.dev/doc/, not a GitHub repository URL. No recency signal available. |

The hand-off matches the user's confirmed choices and explicit bootstrap request. No scaffold-shaped files or existing verification log were present. Go modules is the built-in tooling; the registry toolchain map and hand-off package-manager field are empty/omitted.

## Scaffold log

**Resolved invocation**: `mkdir .bootstrap-scaffold && cd .bootstrap-scaffold && go mod init github.com/user/.bootstrap-scaffold`
**Strategy**: subdir-then-move
**Exit code**: 127
**Stdout**: empty
**Stderr**:

```text
zsh:1: command not found: go
```

**Files moved**: 0
**Conflicts (.scaffold siblings)**: none
**.gitignore handling**: absent in scaffold
**.bootstrap-scaffold left in place at**: `.bootstrap-scaffold/` (empty)

No merge was attempted. Existing context documents and Git metadata were not modified. The module path in the invocation is the literal registry substitution; no module was created. Before retry, resolve a real module path instead of retaining the registry's placeholder GitHub owner and temporary directory name.

**Audit not run**: scaffold halted because the Go command was unavailable; no project to audit.

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

Make the Go toolchain available on PATH, verify `go version`, then re-invoke `/10x-bootstrapper`. The retained empty temporary directory must be handled before retry because the registry command starts with `mkdir`. Preserve this failure log when recording a subsequent attempt. No application, Azure integrations, CI workflows, or agent instruction files were generated.

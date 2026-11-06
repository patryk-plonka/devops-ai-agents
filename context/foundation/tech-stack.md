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

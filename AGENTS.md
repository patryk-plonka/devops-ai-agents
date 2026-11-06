# Repository Guidelines

This repository contains a local Go CLI for a consultant's Azure health checks. The S-01 implementation collects inventory and Resource Health and publishes an explicitly incomplete Markdown report. The operator confirmed all manual checks on 2026-11-06; see @context/changes/subscription-resource-health-report/verification.md for acceptance and scenario limits.

## Critical rules

Keep Azure access read-only and ticket submission manual. Generate drafts only after the consultant selects findings. Every finding needs traceable evidence without secrets or unnecessary sensitive details. Report missing evidence as unknown/not checked; partial failures must remain visibly incomplete. “OK” requires complete applicable coverage, not merely absent alerts. See @context/foundation/prd.md.

Bind Azure CLI credentials to the selected subscription; leave credential TenantID unset. Disable ARM provider registration and preserve the GET-only public-ARM route/subscription boundary, including continuations and redirects. Do not repair login, permissions, or provider registration automatically.

Filter accepted text/tags before checkpointing and render source strings literally. Never persist raw SDK responses, profile usernames, tokens, or error bodies. Preserve completed pages on failure/cancellation. Publish report.md only from committed run.json; both must identify the same run/revision. No overall OK classification exists in S-01.

## Validation

Run `go build ./...`, `go test -race -cover -count=1 ./...`, `go vet ./...`, `staticcheck ./...`, `govulncheck ./...`, `./scripts/check-format.sh`, `shellcheck scripts/check-format.sh internal/cli/testdata/fake-az.sh`, and `actionlint`. Exact tool pins and matching CI commands are in @context/foundation/development-tools.md and @.github/workflows/checks.yml.

Tests use synthetic CLI processes and ARM transports; they must not use the operator's Azure session or live network. Keep dependency injection in Go test boundaries, never production token/endpoint flags. Signal tests synchronize after a committed page, without sleeps. Synthetic success does not complete manual acceptance. Do not report “no packages matched” as a successful test or vulnerability audit. When the sandbox blocks user-cache writes, use temporary writable `GOCACHE`, `GOMODCACHE`, and `STATICCHECK_CACHE` directories; see @context/changes/bootstrap-verification/setup-check.md.

## Product scope

Follow @context/foundation/prd.md for current requirements and @context/foundation/tech-stack.md for stack decisions. Shaping notes retain historical answers; the PRD contains the later workload and timeline decisions.

V1 checks one subscription per manually initiated run using the existing work identity. Cover Azure Advisor, Service Health, Resource Health, and existing Azure Monitor alerts. Resolve ambiguous subscription names interactively and report denied subscription access as unsuccessful. Produce the Markdown report before selected drafts.

The current S-01 flow is `go run ./cmd/devops-ai-agents health check --subscription "<id-or-full-name>"`; operate from this repository tree after external `az login`. Advisor, Service Health, Azure Monitor, and drafts remain later slices. See @README.md for guest login, evidence inspection, exit codes, and interruption behavior.

Baseline comparisons, custom proactive checks, scheduling, concurrent subscription runs, other agents, and a web interface are excluded. Production write access would require the token-burn guard and self-security scanner; it is outside v1.

## Repository layout and conventions

@go.mod defines the module identity and Go version. `cmd/devops-ai-agents` owns process signals/exit; `internal/cli` coordinates interaction; `internal/azure` owns external adapters and the read boundary; `internal/health` owns normalization/coverage/orchestration; `internal/runstore` checkpoints local artifacts; `internal/report` renders Markdown. Use standard Go formatting and package tests; no coverage threshold is imposed. GitHub Actions checks are configured; releases remain unconfigured.

`runs/<UTC-timestamp>-<random-id>/` holds private local `run.json` and `report.md`, excluded by the root `/runs/` ignore rule. Never force-add output or copy live identifiers/resource data into tracked fixtures/docs. Keep fixtures synthetic under package `testdata/`. Do not delete existing runs as cleanup. Git has no commits; keep work uncommitted unless the user authorizes a commit.

Edit living foundation documents in place; follow @context/foundation/README.md. Put change-specific plans and research under `context/changes/<change-id>/` per @context/changes/README.md. Completed changes belong in `context/archive/`, which is read-only by convention; see @context/archive/README.md.

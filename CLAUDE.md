# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`gccli` is a Go CLI (`gccli`) for interacting with the Garmin Connect API. It covers authentication, activities, workouts, health data, body composition, devices, gear, goals, badges, challenges, profile, hydration, training, and wellness data. Module: `github.com/bpauli/gccli`.

## Build & Quality Commands

```bash
make build              # Build binary to ./bin/gccli
make run -- <args>      # Build and run with arguments
make fmt                # Format code (goimports + gofumpt)
make lint               # Run golangci-lint
make test               # Unit tests with race detector
make test-e2e           # E2E tests (requires .env with GARMIN_EMAIL/GARMIN_PASSWORD)
make ci                 # Full CI gate: fmt-check + lint + test
make tools              # Install dev tools to .tools/
```

Run a single test:
```bash
go test -run TestFuncName ./internal/pkg/...
```

Run E2E tests selectively:
```bash
go test -tags=e2e -run TestName -v -count=1 ./internal/e2e/...
```

All three checks (`make fmt`, `make lint`, `make test`) must pass before committing.

## Architecture

**Layered structure:** `cmd/gccli/main.go` → `internal/cmd/` (CLI commands) → `internal/garminapi/` (API client) → `internal/garminauth/` (authentication).

### Key packages

- **`internal/cmd/`** — Kong-based CLI commands. Each command is a struct with `Run(g *Globals) error`. `Globals` holds context, UI, and account info. `resolveClient()` loads tokens from the OS keyring and creates an API client.
- **`internal/garminapi/`** — HTTP client for Garmin Connect. Returns `json.RawMessage` from API calls (no Go structs for every endpoint). Includes `RetryTransport` (429/5xx retry with backoff) and `CircuitBreaker` for resilience. Auto-refreshes OAuth2 tokens on 401.
- **`internal/garminauth/`** — Garmin SSO authentication. Supports browser-based SSO (local callback server) and headless email/password login with MFA. OAuth1 → OAuth2 exchange flow.
- **`internal/secrets/`** — OS keyring storage (macOS Keychain, Linux Secret Service, file fallback). Tokens keyed by `gccli:token:{email}`.
- **`internal/config/`** — Config file paths and env var overrides.
- **`internal/outfmt/`** — Output formatting (JSON/Table/Plain). Mode passed via context.
- **`internal/ui/`** — Terminal UI (colors, prompts) via termenv. Passed via context.
- **`internal/errfmt/`** — Maps typed errors to user-friendly messages.
- **`internal/fit/`** — FIT file binary encoding for weight data uploads.
- **`internal/testutil/`** — Test helpers: mock HTTP servers, test token fixtures.
- **`internal/e2e/`** — End-to-end tests (build tag `e2e`) against real Garmin Connect API.

### Patterns

- **Context-based DI:** Output mode and UI are injected via `context.Context` using `outfmt.NewContext()`/`ui.NewContext()` and retrieved with `outfmt.IsJSON(ctx)`/`ui.FromContext(ctx)`.
- **Function variable mocking:** External dependencies are stored as package-level function variables (e.g., `newClientFn`, `loginHeadlessFn`, `refreshTokensFn`) that tests override. No interface-heavy DI.
- **Output convention:** Parseable output (JSON, TSV) goes to stdout. Human messages (errors, info, warnings) go to stderr via `UI`.
- **Typed errors:** `garminapi` defines `TokenExpiredError`, `RateLimitError` and `GarminAPIError`. `errfmt.Format()` adds hints for the first two.

## Code Style

- **Formatting:** `goimports` (local prefix `github.com/bpauli/gccli`) + `gofumpt`. Run `make fmt`.
- **Import order:** stdlib → external → local.
- **Linting:** govet, errcheck, staticcheck, unused (see `.golangci.yml`).
- **Commits:** Conventional Commits — `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`.
- **No AI co-author:** NEVER add `Co-Authored-By` lines mentioning Claude or any AI assistant in commit messages.
- **No AI attribution in PRs:** NEVER include "Generated with Claude Code" or any similar AI attribution in pull request descriptions.

## Testing

- Unit tests use stdlib `testing` and `httptest`. Co-located `*_test.go` files.
- `testutil.NewClientWithServer(t, handler)` creates a mock server + configured API client.
- E2E tests use build tag `e2e` and require `.env` with credentials. Created resources are prefixed `E2E_TEST_` and cleaned up via `t.Cleanup()`.

## Documentation Updates

For every feature enhancement (new command, new flag, new subcommand), you MUST update all three documentation sources:

- **`README.md`** — features list and command examples section
- **`skills/gccli/SKILL.md`** — command reference for the skill
- **`docs/index.html`** — feature cards and command reference on the docs site

## Shell Completion

Shell completions are **auto-generated** from the Kong parser tree (`internal/cmd/completion*.go`). When you add a new command or subcommand to the `CLI` struct in `root.go`, it is automatically included in all completion scripts (bash, zsh, fish, powershell). However, you MUST verify that the new command appears in `TestCollectCommands` in `completion_test.go` — update the expected top-level commands list if you add a new top-level command group.

## Releases

After goreleaser creates a new release (triggered by pushing a tag), you MUST manually edit the release notes on GitHub:

1. Replace the auto-generated commit list with a proper changelog (Added/Fixed/Docs sections)
2. Add a **New Contributors** section at the bottom mentioning any external contributors (e.g. `* @username made their first contribution in #PR`)
3. Add a **Full Changelog** comparison link (e.g. `https://github.com/bpauli/gccli/compare/v1.1.0...v1.2.0`)

Use `gh release edit <tag> --notes "..."` to update the release notes.

## Changelog

For every new release, you MUST update `CHANGELOG.md` with all changes since the last release. The changelog follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format:

1. Add a new `## [version] - YYYY-MM-DD` section at the top (below the header)
2. Group changes under `### Added`, `### Fixed`, `### Changed`, `### Docs` as appropriate
3. Reference relevant PR/issue numbers (e.g. `(#30)`)
4. Add a `[version]` link reference at the bottom of the file
5. Commit the changelog update to main after the release is created

## Environment Variables

- `GCCLI_ACCOUNT` — Default account email (overrides `default_account` in config.json, which is auto-set by `gccli auth login`)
- `GCCLI_DOMAIN` — Garmin domain (garmin.com/garmin.cn)
- `GCCLI_JSON` / `GCCLI_PLAIN` — Output mode
- `GCCLI_COLOR` — Color mode (auto/always/never)
- `GCCLI_KEYRING_BACKEND` — Keyring backend (auto/keychain/secret-service/file)

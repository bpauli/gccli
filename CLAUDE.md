# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

Read `AGENTS.md` before you verify or merge a PR, and before you change command output.

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
- **`internal/garminapi/`** - HTTP client for Garmin Connect. Returns `json.RawMessage` from API calls (no Go structs for every endpoint). All API client requests and the exercise catalog fetch go through `RetryTransport` (429 retried for every method within `Retry-After`, 5xx retried only for GET and HEAD with backoff, because writes may already be saved) over a stall watch per attempt (`stallTransport`: a request fails with `StallError` when no data moves for 60 s; a transfer that keeps moving has no time limit) over a clone of `http.DefaultTransport`. Auto-refreshes OAuth2 tokens on 401.
- **`internal/garminauth/`** — Garmin SSO authentication. Supports browser-based SSO (local callback server) and headless email/password login with MFA. OAuth1 → OAuth2 exchange flow.
- **`internal/secrets/`** — OS keyring storage (macOS Keychain, Linux Secret Service, file fallback). Tokens keyed by `gccli:token:{email}`.
- **`internal/config/`** — Config file paths and env var overrides.
- **`internal/outfmt/`** — Output formatting (JSON/Table/Plain). Mode passed via context.
- **`internal/ui/`** - Terminal UI (colors, prompts) via termenv. Passed to commands in `Globals.UI`.
- **`internal/errfmt/`** — Maps typed errors to user-friendly messages.
- **`internal/fit/`** — FIT file binary encoding for weight data uploads.
- **`internal/testutil/`** - Test helpers. `TempConfigDir` points the config directory at a temp directory.
- **`internal/e2e/`** — End-to-end tests (build tag `e2e`) against real Garmin Connect API.

### Patterns

- **Context-based DI:** The output mode is injected via `context.Context` using `outfmt.NewContext()` and read with `outfmt.IsJSON(ctx)`/`outfmt.IsPlain(ctx)`. Commands get the UI from `Globals.UI`.
- **Function variable mocking:** External dependencies are stored as package-level function variables (e.g., `newClientFn`, `loginHeadlessFn`, `refreshTokensFn`) that tests override. No interface-heavy DI.
- **Output convention:** Parseable output (JSON, TSV) goes to `g.Stdout`. Human messages (errors, info, warnings) go to stderr via `g.UI`, and prompts go to `g.Stderr`. Commands never write to `os.Stdout` or `os.Stderr` directly. `Execute` sets both writers from the os files, so a test can set a `bytes.Buffer` on `Globals` instead.
- **Command table:** The Kong tree is `commands` in `internal/cmd/root.go`, a `[]Node` of `Group`, `Custom` and `Leaf` rows. A command that sends one GET and prints the JSON, or prints a list of objects as columns, is a `Leaf` row: its Kong args struct in `Spec: argsOf(&struct{...}{})`, the endpoint name, `Params` that bind each placeholder to `In("Field")`, `DisplayName`, `ProfilePK` or `Today` (with `.Or(...)` as the fallback), the `Verb` that wraps a request error, and `Output: JSON{}` or `Output: Rows{...}` with its `Noun`, the `Items` wrapper key and the `Columns`, each with a `Path` and a `Format` from the closed set in `rows.go`. `leaf.go` runs every row. A command with logic, a body, an output that no `Output` type prints, or more than one data request (a `DisplayName` or `ProfilePK` lookup does not count, and neither does the units lookup of a `Rows` row) is a `Custom` row with its own struct and `Run`. `checkTable` rejects a bad row before `kong.New`, and `TestCommandTable` reports every error. `TestGolden` pins the output of every leaf, `TestGoldenErrorCases` requires a golden file per leaf that ends in an error from the data request (usually `<leaf>@error.golden`), which pins the `Verb`, `TestLeafRowSadPaths` runs every `Leaf` row through a failed lookup and an empty body, and `TestLeafRun` covers the runner on a fixture tree.
- **Endpoint table:** To add a Garmin JSON request, add a row to the `<area>Endpoints` table at the top of its service file in `internal/garminapi/` (for example `gearEndpoints` in `gear.go`). A new table also goes into `mustIndex` in `endpoint.go`, or `TestEveryTableIsIndexed` fails. The command sends the row with `client.Call(ctx, "<area>.<action>", args, body)`. Pass the name as a string literal, and pass `nil` as the body unless the row has `Body: true`. `Call` escapes each `{name}` placeholder, and a `Param` makes it `Optional`, gives it a `Default` or limits it to an `Enum`. Write a `Client` method only when it adds logic: it builds a body, reads the answer, joins requests, downloads or uploads. `TestNoNewPassthroughMethods` fails on a method that only forwards one request. `TestCallSitesMatchTable` checks the name, the placeholders and the body argument of each `Call`. Each service test (`TestGearEndpoints`) needs one case per row.
- **Typed errors:** `garminapi` defines `TokenExpiredError`, `RateLimitError`, `GarminAPIError` and `StallError`. `errfmt.Format()` adds hints for the first two.

## Code Style

- **Formatting:** `goimports` (local prefix `github.com/bpauli/gccli`) + `gofumpt`. Run `make fmt`.
- **Import order:** stdlib → external → local.
- **Linting:** govet, errcheck, staticcheck, unused (see `.golangci.yml`).
- **Commits:** Conventional Commits — `feat:`, `fix:`, `refactor:`, `test:`, `docs:`, `chore:`.
- **No AI co-author:** NEVER add `Co-Authored-By` lines mentioning Claude or any AI assistant in commit messages.
- **No AI attribution in PRs:** NEVER include "Generated with Claude Code" or any similar AI attribution in pull request descriptions.

## Testing

- Unit tests use stdlib `testing` and `httptest`. Co-located `*_test.go` files.
- `garminapi` tests call `testServer(t, handler)`. It starts an `httptest` server and returns a client that points at it.
- `cmd` tests call `overrideNewClient(t, server)`. It replaces `newClientFn`, so the command under test calls the test server.
- `TestGolden` runs every leaf command in table, plain and json mode. Each command has one file in `internal/cmd/testdata/golden/` with its args, the mock routes it calls, and the expected stdout, stderr and exit code per mode. The generated `<mode> requests` section records each request in the order sent, as a `METHOD URI` line with the body indented below it (JSON indented, a multipart part as its size and hash), so a golden also pins the order, the count and the bodies of the requests. A new leaf command fails the test until it has a file. To pin one more input for a command, such as an empty body or another response shape, add a case file `<leaf>@<case>.golden` next to the main file. A case file has the same sections and checks as the main file, and its `<leaf>` part must name a leaf command. Write the input sections by hand: `args` (one argument per line), `stdin`, `file <name>` and `route <METHOD> <uri> [status]` (status 200 by default). A section name that ends in `noeol` marks data without a final newline. Then write the outputs with `go test ./internal/cmd -run TestGolden -update` and review the diff. Every leaf that sends a request also needs a golden in which the data request answers with an error status, such as a 400, so that the verb of the error message is pinned. Any golden of the leaf that ends in an error from its data route counts; the usual file is `<leaf>@error.golden`. `TestGoldenErrorCases` fails until such a golden exists. A leaf that sends no request goes into `errorCaseExempt` with a reason.
- `TestGroupHelp` pins the `--help` output of the root and of every command group in `internal/cmd/testdata/help.golden`. After you add a group or change a group `Help`, run `go test ./internal/cmd -run TestGroupHelp -update` and review the diff.
- E2E tests use build tag `e2e` and require `.env` with credentials or `GARMIN_TOKEN`. Created resources are prefixed `E2E_TEST_` and cleaned up via `t.Cleanup()`.
- `GARMIN_TOKEN` replaces the SSO login of the E2E suite. It holds the base64 output of `gccli auth export`. All tests share one token, so one refresh serves the later tests too. When the access token has expired, the refresh runs in the first test that needs the token. When Garmin rejects an unexpired token with a 401, the refresh runs in the first test that gets the 401. A failed refresh fails that test, and the later tests skip.
- The export and your keyring hold the same refresh token, and Garmin may reject a refresh token that was used before. So a refresh in the suite can break the stored login. `gccli auth export` does not refresh, so the export of an expired login is also expired, and the suite then spends the keyring's refresh token. To keep the stored login, export right after `gccli auth login`, or export a login that no keyring you use holds.

## Documentation Updates

The command reference in `README.md`, `skills/gccli/SKILL.md` and `docs/index.html` is generated from the Kong command tree. It is between the `<!-- gccli:commands:begin -->` and `<!-- gccli:commands:end -->` lines. Do not edit it by hand.

- After you add or change a command, a flag or a `help:` tag, run `go test ./internal/cmd -run TestDocsUpToDate -update`.
- `make test` fails while a block is stale.
- A new top-level command also needs a place in `referenceGroups` in `internal/cmd/docs_gen.go`. If it has no place, `TestDocsUpToDate` fails.
- Text outside the markers is hand-written. Update it by hand when a feature needs it:
  - `README.md`: the features list and the examples.
  - `skills/gccli/SKILL.md`: Setup, Output, Date shortcuts, Worked examples, the strength training workflow and Notes.
  - `docs/index.html`: the rest of the page, such as the feature cards, the Commands intro above the reference and the exit code table. Its CSS also styles the generated rows.

## Shell Completion

Shell completions are **auto-generated** from the Kong parser tree (`internal/cmd/completion*.go`). The tree is the `commands` value in `root.go`: a top-level command is a `Group` or `Custom` node in it, and a subcommand is a child of its area's `Group` literal (for example `healthGroup` in `health.go`). A node in the tree is in every completion script (bash, zsh, fish, powershell). The same tree also generates the command reference in the docs (see Documentation Updates).

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

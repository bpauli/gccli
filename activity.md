# gccli - Garmin Connect CLI Tool - Activity Log

## Current Status
**Last Updated:** 2026-02-12
**Tasks Completed:** 36 / 36
**Current Task:** All tasks complete ✓

---

## Session Log

<!--
The Ralph Wiggum loop will append dated entries here.
Each entry should include:
- Date and time
- Task worked on
- Changes made
- Commands run
- Screenshot filename (if applicable)
- Any issues and resolutions
-->

### 2026-02-11 — Task 1: Project scaffolding

**Changes:**
- Created `go.mod` (module `github.com/bpauli/gccli`, Go 1.23)
- Created `cmd/gc/main.go` with entry point calling `cmd.Execute()` with version/commit/date via ldflags
- Created `internal/cmd/root.go` with stub `Execute()` function
- Created `Makefile` with targets: `build`, `run`, `fmt`, `fmt-check`, `lint`, `test`, `test-e2e`, `ci`, `tools`, `clean`
- Created `.goreleaser.yaml` for darwin/linux amd64/arm64 builds (CGO_ENABLED=1 for macOS)
- Created `.golangci.yml` with linters: govet, errcheck, staticcheck, unused; formatter: gofumpt

**Commands run:**
- `go mod init github.com/bpauli/gccli`
- `go mod tidy`
- `make tools` (installed golangci-lint v2.1.6, gofumpt v0.7.0, goimports v0.31.0)
- `make build` — success, binary at `./bin/gc`
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — passes (no test files yet)
- `make ci` — all green
- `./bin/gc` — prints version info

**Issues:**
- golangci-lint v2 moved `gofumpt` to formatters-only (not a linter); removed from `linters.enable`
- golangci-lint v2 merged `gosimple` into `staticcheck`; removed from config

### 2026-02-11 — Task 2: Config package

**Changes:**
- Created `internal/config/paths.go` with `ConfigDir()`, `ConfigFilePath()`, `CredentialsDir()` using `os.UserConfigDir()`
- Created `internal/config/config.go` with `File` struct (`KeyringBackend`, `DomainName`, `DefaultFormat`), `Read()`/`Write()` functions, `ReadFrom()`/`WriteTo()` for testability
- Added env var support: `GC_DOMAIN`, `GC_COLOR`, `GC_JSON`, `GC_PLAIN`, `GC_KEYRING_BACKEND`
- Helper methods: `Domain()` (env → config → default), `KeyringBackendValue()`, `IsJSON()`, `IsPlain()`, `ColorMode()`
- Created `internal/config/config_test.go` with tests for paths, read/write roundtrip, missing file handling, invalid JSON, file permissions, env var overrides, and truthy parsing

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass
- `make build` — success

**Issues:**
- `Domain` field name conflicted with `Domain()` method on the struct; renamed field to `DomainName` (JSON key stays `"domain"`)

### 2026-02-11 — Task 3: UI package

**Changes:**
- Added `github.com/muesli/termenv` dependency
- Created `internal/ui/ui.go` with `UI` struct wrapping termenv
- `New(colorMode)` constructor writing to stderr, `NewWithWriter(w, colorMode)` for testability
- Color modes: `"auto"` (detect terminal), `"always"` (force TrueColor), `"never"` (Ascii/no color)
- Methods: `Successf()` (green), `Error()` (bold red), `Warnf()` (yellow), `Infof()` (blue)
- Context helpers: `NewContext(ctx, ui)` and `FromContext(ctx)` with auto-mode fallback
- Created `internal/ui/ui_test.go` with tests for all methods, color mode switching (ANSI codes present/absent), and context round-trip

**Commands run:**
- `go get github.com/muesli/termenv@latest`
- `make fmt` — clean
- `make lint` — 0 issues (after fixing errcheck on `fmt.Fprintln`)
- `make test` — all pass
- `make build` — success

**Issues:**
- `errcheck` linter flagged unchecked `fmt.Fprintln` returns; fixed with `_, _ =` discard pattern

### 2026-02-11 — Task 4: Output formatting package

**Changes:**
- Created `internal/outfmt/outfmt.go` with `Mode` type (`Table`, `JSON`, `Plain`)
- `WriteJSON()` with pretty-print and no HTML escaping
- `WriteTable()` using `tabwriter` for aligned column output
- `WritePlain()` for TSV output with no alignment
- `NewTabWriter()` helper for custom table usage
- Context helpers: `NewContext(ctx, mode)`, `ModeFromContext(ctx)`, `IsJSON(ctx)`, `IsPlain(ctx)`
- Convenience `Write(ctx, v, header, rows)` that dispatches to the correct format based on context
- Created `internal/outfmt/outfmt_test.go` with tests for all modes, JSON pretty-print/no-HTML-escaping, table alignment, plain TSV, empty inputs, and context round-trip

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass
- `make build` — success

**Issues:**
- None

### 2026-02-11 — Task 5: Error formatting package

**Changes:**
- Created `internal/errfmt/errfmt.go` with typed error types and user-facing formatter
- Error types: `AuthRequiredError`, `TokenExpiredError`, `RateLimitError`, `GarminAPIError`
- Each type has an `Error()` method for programmatic use and optional fields (Email, RetryAfter, StatusCode, Message)
- `Format(err error)` function maps known error types to actionable user messages using `errors.As` for wrapped error support
- Created `internal/errfmt/errfmt_test.go` with tests for all error types' `Error()` methods, all `Format()` branches including wrapped errors, and interface compliance checks

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass
- `make build` — success

**Issues:**
- None

### 2026-02-11 — Task 6: Secrets/keyring store

**Changes:**
- Added `github.com/99designs/keyring` dependency
- Created `internal/secrets/errors.go` with `ErrNotFound` sentinel error
- Created `internal/secrets/store.go` with `Store` struct wrapping `keyring.Keyring`
- `Open(backend)` constructor with backend selection: `"keychain"`, `"secret-service"`, `"file"`, or `""` (auto)
- Linux D-Bus fallback: if auto-detect fails on Linux, falls back to file backend
- `OpenWithKeyring(ring)` constructor for injecting test keyrings
- `Get(email)`, `Set(email, data)`, `Delete(email)` methods with `gc:token:{email}` key format
- Created `internal/secrets/store_test.go` with in-memory keyring mock and tests for set/get, not-found, delete, overwrite, multiple emails, key format, and backend resolution

**Commands run:**
- `go get github.com/99designs/keyring@latest`
- `go mod tidy`
- `make fmt` — clean
- `make lint` — 0 issues (after removing unused `dbusTimeout` constant)
- `make test` — all pass
- `make build` — success

**Issues:**
- `unused` linter flagged `dbusTimeout` constant; removed since D-Bus fallback is handled by catching Open errors directly

### 2026-02-11 — Task 7: CLI root

**Changes:**
- Added `github.com/alecthomas/kong` v1.14.0 dependency
- Created `internal/cmd/root_flags.go` with `RootFlags` struct (`JSON`, `Plain`, `Color`, `Account`) using Kong struct tags with env var support
- Created `internal/cmd/exit.go` with `ExitError` type (wraps error with exit code), `exitCode()` helper, and `exitSignal` type for intercepting Kong's help/version exit calls
- Created `internal/cmd/output_helpers.go` with `TableWriter()` (tabwriter for stdout) and `PrintNextPageHint()` (pagination hint)
- Created `internal/cmd/help_printer.go` with `colorHelpPrinter` that wraps Kong's `DefaultHelpPrinter` and adds terminal colors — bold section headers ("Flags:", "Commands:", "Usage:") and dimmed hint text
- Rewrote `internal/cmd/root.go` with full Kong CLI structure:
  - `CLI` struct embedding `RootFlags` with `--version` flag and `auth` placeholder command
  - `Globals` struct for runtime DI into command Run methods (context, UI, account)
  - `Execute()` sets up Kong parser with custom help printer, overrides Exit to use panic/recover pattern for clean exit code capture
  - `run()` parses args, builds context with output mode and UI, injects `Globals` into commands, centralizes error formatting via `errfmt.Format()`
- No changes needed to `cmd/gc/main.go` — it already calls `cmd.Execute(os.Args[1:], version, commit, date)`
- Created `internal/cmd/root_test.go` with tests for: `--version` (exit 0), `--help` (exit 0), `auth --help` (exit 0), `auth` (exit 0), unknown command (non-zero), no args (non-zero), `exitCode()` function, `ExitError` type, `PrintNextPageHint`, `TableWriter`

**Commands run:**
- `go get github.com/alecthomas/kong@latest`
- `go mod tidy`
- `make fmt` — clean
- `make lint` — 0 issues (after exporting helpers to satisfy `unused` linter)
- `make test` — all pass
- `make build` — success
- `make ci` — all green
- `./bin/gc --help` — shows colorized help with flags, commands, usage
- `./bin/gc --version` — shows version string
- `./bin/gc auth` — prints "auth: not yet implemented"

**Issues:**
- `unused` linter flagged unexported `tableWriter` and `printNextPageHint`; exported them as `TableWriter` and `PrintNextPageHint` since they're in an `internal` package and will be used by future commands

### 2026-02-11 — Task 8: Garmin auth endpoints, token types, and SSO constants

**Changes:**
- Created `internal/garminauth/endpoints.go` with `Endpoints` struct and `NewEndpoints(domain)` constructor
  - SSO base URLs for garmin.com (global) and garmin.cn (China)
  - Paths for embed, signin, MFA verification, OAuth preauthorized, and OAuth2 exchange
  - `PreauthorizedURL(ticket, loginURL)` and `ExchangeURL()` helper methods
  - OAuth consumer URL constant and User-Agent constant
- Created `internal/garminauth/tokens.go` with `Tokens` struct
  - OAuth1 credentials: `OAuth1Token`, `OAuth1Secret`
  - OAuth2 credentials: `OAuth2AccessToken`, `OAuth2RefreshToken`, `OAuth2ExpiresAt`
  - Optional fields: `MFAToken`, `DisplayName` (omitempty)
  - Required fields: `Domain`, `Email`
  - `IsExpired()` method with 60-second grace period
  - `HasOAuth1()` method for refresh eligibility check
  - `Marshal()` and `UnmarshalTokens()` for JSON serialization
- Created `internal/garminauth/options.go` with `LoginOptions` struct
  - `Domain`, `MFACode`, `PromptMFA` callback, `HTTPClient` fields
  - `domain()` helper with DomainGlobal default
- Created comprehensive unit tests across 3 test files:
  - `endpoints_test.go`: global/China domain URLs, PreauthorizedURL parsing, ExchangeURL, domain constants
  - `tokens_test.go`: IsExpired (zero/past/grace period/future), HasOAuth1 combos, marshal roundtrip, omitempty, invalid JSON
  - `options_test.go`: default domain, custom domain, zero value fields

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass
- `make build` — success

**Issues:**
- None

### 2026-02-11 — Task 9: Headless SSO login flow

**Changes:**
- Added `golang.org/x/net` dependency for HTML parsing
- Created `internal/garminauth/sso_headless.go` with the full headless SSO login flow:
  - `LoginHeadless(ctx, email, password, opts)` — public API
  - `loginHeadless(ctx, email, password, opts, ep)` — internal implementation with explicit endpoints for testability
  - Step 1: GET SSO embed page to establish cookies via `net/http/cookiejar`
  - Step 2: GET SSO signin page, extract `_csrf` token using `golang.org/x/net/html` parser
  - Step 3: POST credentials (username, password, csrf, embed=true), extract service ticket via regex
  - Step 4: Fetch OAuth consumer credentials from S3 (`oauthConsumerURL` is a package var for testability)
  - Step 5: Exchange service ticket for OAuth1 token via preauthorized endpoint (OAuth1 HMAC-SHA1 signed)
  - Step 6: Exchange OAuth1 token for OAuth2 access+refresh tokens (OAuth1 HMAC-SHA1 signed)
  - MFA detection: returns error when MFA title detected (full MFA handling deferred to task 10)
  - HTML parsing helpers: `getCSRFToken()` (x/net/html tree walk), `getTitle()` (x/net/html), `getTicket()` (regex)
  - OAuth1 signing: `signOAuth1()` with HMAC-SHA1, RFC 3986 percent-encoding, proper parameter normalization
  - Helper functions: `percentEncode()`, `sortedPercentEncode()`, `generateNonce()`
- Created `internal/garminauth/sso_headless_test.go` with comprehensive tests:
  - `mockSSO()` and `ssoMux()` test helpers creating a full mock SSO server
  - `TestLoginHeadless_Success` — full flow end-to-end with mock server
  - `TestLoginHeadless_BadCredentials` — authentication failure
  - `TestLoginHeadless_MFARequired` — MFA detection
  - `TestLoginHeadless_AccountLocked` — locked account detection
  - `TestLoginHeadless_CustomHTTPClient` — custom HTTP client support
  - `TestLoginHeadless_ContextCancelled` — context cancellation
  - `TestGetCSRFToken` — 5 cases (standard, extra attrs, missing, empty, nested)
  - `TestGetTitle` — 5 cases (standard, garmin, MFA, missing, empty)
  - `TestGetTicket` — 4 cases (standard, full page, missing, empty)
  - `TestFetchOAuthConsumer` — success, server error, invalid JSON
  - `TestExchangePreauthorized` — success, no MFA token, missing token, server error
  - `TestExchangeOAuth2` — success, with MFA token, server error
  - `TestSignOAuth1_Format` — OAuth1 header format validation
  - `TestSignOAuth1_WithToken` — OAuth1 with resource owner token
  - `TestPercentEncode` — RFC 3986 percent-encoding (7 cases)
  - `TestEndpointsDomain` — domain detection from endpoint URL

**Commands run:**
- `go get golang.org/x/net@latest` — added dependency
- `go mod tidy`
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (garminauth: 2.8s)
- `make build` — success
- `make ci` — all green

**Issues:**
- `errcheck` linter flagged unchecked `resp.Body.Close()` returns; fixed with `_ = resp.Body.Close()` for inline and `defer func() { _ = resp.Body.Close() }()` for deferred
- `staticcheck` S1031 flagged unnecessary nil check around `range bodyParams`; removed since range on nil map is a no-op
- `go get golang.org/x/net@latest` upgraded go.mod from Go 1.23 to Go 1.24 (machine has Go 1.25, compatible)

### 2026-02-11 — Task 10: MFA handling

**Changes:**
- Created `internal/garminauth/mfa.go` with MFA detection, prompting, and submission
  - `isMFARequired(htmlBody)` — detects MFA challenge by checking if page title contains "MFA"
  - `PromptMFA()` — interactive terminal prompt reading MFA code from stdin, prompts on stderr
  - `promptMFAFrom(w, r)` — testable version accepting writer/reader for prompt and input
  - `submitMFA(ctx, client, ep, csrf, mfaCode)` — POSTs MFA code to `/sso/verifyMFA/loginEnterMfaCode` with CSRF token, extracts service ticket from response, detects invalid codes
  - `resolveMFACode(opts)` — resolves MFA code from `LoginOptions.MFACode` (pre-supplied) or `LoginOptions.PromptMFA` (interactive), errors if neither available
- Modified `internal/garminauth/sso_headless.go` to integrate MFA into the login flow:
  - Replaced `fmt.Errorf("MFA verification required")` with full MFA handling
  - When MFA is detected: extracts CSRF from MFA page → resolves MFA code → submits MFA → extracts ticket → continues with OAuth exchange
  - Non-MFA path unchanged (extract ticket directly from response)
- Created `internal/garminauth/mfa_test.go` with comprehensive tests:
  - `TestIsMFARequired` — 6 cases (MFA challenge, lowercase, success, no title, empty, error page)
  - `TestPromptMFAFrom` — 4 cases (valid code, whitespace trimming, empty input, EOF)
  - `TestResolveMFACode` — 5 cases (pre-supplied, priority over prompt, prompt function, no code/prompt, prompt error)
  - `TestSubmitMFA` — 3 cases (success, invalid code, server error)
  - `TestLoginHeadless_MFAWithCode` — full end-to-end flow with pre-supplied MFA code
  - `TestLoginHeadless_MFAWithPrompt` — full flow using PromptMFA callback
  - `TestLoginHeadless_MFANoCodeOrPrompt` — error when MFA required but nothing configured
  - `TestLoginHeadless_MFAWrongCode` — error on invalid MFA code
  - `ssoMuxWithMFA()` helper — mock SSO server with MFA challenge flow
- Updated `TestLoginHeadless_MFARequired` in `sso_headless_test.go` to match new behavior (now expects "MFA required" error instead of "MFA verification required" since MFA handling is integrated)

**Commands run:**
- `make fmt` — clean (gofumpt adjusted map literal alignment)
- `make lint` — 0 issues
- `make test` — all pass (garminauth: 1.7s)
- `make build` — success

**Issues:**
- None

### 2026-02-11 — Task 11: Browser SSO flow

**Changes:**
- Created `internal/garminauth/sso.go` with browser-based SSO login flow:
  - `LoginBrowser(ctx, email, opts)` — public API for browser SSO login
  - `loginBrowser(ctx, email, opts, ep)` — internal implementation with explicit endpoints for testability
  - `startCallbackServer()` — starts local HTTP server on `127.0.0.1:0` (random port), returns callback URL and ticket channel
  - Callback handler: captures service ticket from `?ticket=` query parameter, ignores requests without ticket (favicon, etc.)
  - Uses `select { case ch <- ...: default: }` to safely handle duplicate callbacks
  - `buildSSOURL(ep, callbackURL)` — constructs Garmin SSO login URL with `service`, `gauthHost`, `source`, and redirect parameters pointing to local callback
  - `openBrowser(rawURL)` — opens URL in default browser (`open` on macOS, `xdg-open` on Linux)
  - `openBrowserFn` package variable for test injection
  - Applies default 2-minute timeout if context has no deadline
  - Reuses OAuth exchange logic from headless flow (`fetchOAuthConsumer`, `exchangePreauthorized`, `exchangeOAuth2`)
- Created `internal/garminauth/sso_test.go` with comprehensive tests:
  - `TestLoginBrowser_Success` — full end-to-end flow with mock browser callback simulating SSO redirect
  - `TestLoginBrowser_Timeout` — verifies timeout error when browser callback never arrives
  - `TestLoginBrowser_BrowserOpenError` — verifies error when browser fails to open
  - `TestLoginBrowser_ContextCancelled` — verifies behavior with pre-cancelled context
  - `TestStartCallbackServer` — verifies server starts on localhost with random port
  - `TestCallbackServer_WithTicket` — verifies ticket is received from callback
  - `TestCallbackServer_NoTicket` — verifies requests without ticket are ignored
  - `TestBuildSSOURL` — verifies SSO URL construction with all parameters
  - `TestBuildSSOURL_ChinaDomain` — verifies China domain URL construction

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (garminauth: 2.2s)
- `make build` — success

**Issues:**
- None

### 2026-02-11 — Task 12: Auth commands

**Changes:**
- Created `internal/cmd/auth.go` with `AuthCmd` struct grouping four subcommands: Login, Status, Remove, Token
  - `AuthStatusCmd` — shows auth state (email, domain, token expiry) for `--account` email; supports `--json` output with structured map
  - `AuthRemoveCmd` — deletes stored credentials from keyring for `--account` email; warns if not found
  - `AuthTokenCmd` — prints raw OAuth2 access token to stdout; warns if expired
  - `loadSecretsStore` function variable (overrideable in tests) that reads config for keyring backend and opens the OS keyring
- Created `internal/cmd/auth_login.go` with `AuthLoginCmd` implementing `gc auth login <email>`
  - Default: browser SSO via `garminauth.LoginBrowser`
  - `--headless` flag: email/password login via `garminauth.LoginHeadless` with secure password prompt (`golang.org/x/term.ReadPassword`)
  - `--mfa-code` flag: pre-supplied MFA code for non-interactive headless login
  - Interactive MFA prompt via `garminauth.PromptMFA` when MFA is required during headless login
  - Stores tokens in OS keyring after successful login
  - Function variables (`loginBrowserFn`, `loginHeadlessFn`, `readPasswordFn`) for test injection
- Updated `internal/cmd/root.go`: replaced placeholder `authCmd` with real `AuthCmd` (subcommand group)
- Updated `internal/cmd/root_test.go`: changed `TestExecute_Auth` to expect non-zero exit (auth without subcommand is now a usage error)
- Moved `golang.org/x/term` from indirect to direct dependency
- Created `internal/cmd/auth_test.go` with comprehensive tests:
  - `memKeyring` in-memory keyring implementation for isolated test store
  - Helper functions: `newTestSecretsStore`, `testGlobals`, `overrideLoadSecrets`, `testTokens`, `storeTestTokens`
  - Execute-level tests: `TestExecute_AuthLoginHelp`, `TestExecute_AuthStatusHelp`, `TestExecute_AuthRemoveHelp`, `TestExecute_AuthTokenHelp`
  - Status tests: NoAccount, NotFound, Success, Expired
  - Remove tests: NoAccount, NotFound, Success (verifies deletion)
  - Token tests: NoAccount, NotFound, Success, Expired (with warning)
  - Login tests: Browser (mock + verify storage), Headless (mock password + login + verify storage), HeadlessBadPassword, BrowserError

**Commands run:**
- `go mod tidy`
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (cmd: 1.9s)
- `make build` — success
- `./bin/gc auth --help` — shows login, status, remove, token subcommands
- `./bin/gc auth login --help` — shows email arg, --headless, --mfa-code flags
- `make ci` — all green

**Issues:**
- None

### 2026-02-11 — Task 13: API client base

**Changes:**
- Created `internal/garminapi/errors.go` with typed error types for the API client layer:
  - `AuthRequiredError` — no auth credentials found
  - `TokenExpiredError` — token expired and could not be refreshed
  - `RateLimitError` — API returned 429 with optional Retry-After
  - `GarminAPIError` — non-2xx API response with status code and message
  - `InvalidFileFormatError` — unsupported file format requested
- Created `internal/garminauth/refresh.go` with exported `RefreshOAuth2()` function
  - Exchanges existing OAuth1 credentials for new OAuth2 tokens
  - Reuses unexported `fetchOAuthConsumer()` and `exchangeOAuth2()` from garminauth
  - Preserves all token metadata (domain, email, display name, MFA token)
- Created `internal/garminapi/client.go` with `Client` struct and API methods:
  - `NewClient(tokens, ...ClientOption)` constructor with functional options pattern
  - `WithHTTPClient()` and `WithBaseURL()` options for DI and testing
  - `ConnectAPI(ctx, method, path, body)` for JSON API calls with Bearer token auth
  - `Download(ctx, path)` for binary downloads
  - Automatic token refresh on 401: buffers request body, attempts OAuth2 refresh via OAuth1 credentials, retries once on success
  - Rate limit handling: returns `RateLimitError` with Retry-After header on 429
  - Error handling: 4xx/5xx responses return typed `GarminAPIError`
  - 204 No Content handling for delete operations
  - `refreshTokensFn` package variable for test injection of refresh behavior
  - `Tokens()` accessor for current token state
- Created `internal/garminapi/client_test.go` with comprehensive tests:
  - `testTokens()` and `testServer()` test helpers
  - NewClient tests: default config, China domain URL, custom options, Tokens accessor
  - ConnectAPI tests: GET success, POST with body, 204 No Content, server error, 404, rate limit, context cancelled
  - Auth tests: 401 with no OAuth1, 401 with refresh success (verifies retry and new token), 401 with refresh failure, POST retry preserves body
  - Download tests: success (binary data), server error, rate limit, 401 with refresh, 401 without OAuth1
  - Error type tests: all five error types with edge cases
  - RefreshToken tests: no OAuth1 creds, success, failure

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (garminapi: 1.9s)
- `make build` — success

**Issues:**
- None

### 2026-02-11 — Task 14: Retry transport and circuit breaker

**Changes:**
- Created `internal/garminapi/transport.go` with `RetryTransport` wrapping `http.RoundTripper`:
  - `NewRetryTransport(base)` constructor with sensible defaults (nil base → `http.DefaultTransport`)
  - Retries on 429 (Too Many Requests) up to `MaxRetries429` (default 3), respecting `Retry-After` header
  - Retries on 5xx server errors up to `MaxRetries5xx` (default 2) with exponential backoff + jitter
  - Request body buffered for retries (reads body once, replays via `bytes.NewReader`)
  - Context-aware sleep via `contextSleep()` — returns early on cancellation
  - `parseRetryAfter()` helper parsing `Retry-After` header as seconds (minimum 1s fallback)
  - `exponentialBackoff(attempt)` — base 500ms doubling each attempt, with up to 25% random jitter
  - `sleepFn` field for test injection of sleep behavior
- Created `internal/garminapi/circuitbreaker.go` with `CircuitBreaker` pattern:
  - `NewCircuitBreaker(threshold, resetTimeout)` constructor (defaults: 5 failures, 30s reset)
  - Three states: `closed` (normal), `open` (blocking), `half-open` (single probe allowed)
  - `Allow()` — checks if requests are permitted; returns `ErrCircuitOpen` when open
  - `RecordSuccess()` — resets failures, closes circuit from half-open
  - `RecordFailure()` — increments failures, opens circuit at threshold
  - `State()` — returns string state for diagnostics
  - `Failures()` — returns current consecutive failure count
  - Thread-safe via `sync.Mutex`
  - `nowFn` field for test time control
- Created `internal/garminapi/transport_test.go` with comprehensive tests:
  - No retry on success (single call)
  - 429 retry with eventual success, 429 exhausts retries
  - 5xx retry with eventual success, 5xx exhausts retries
  - No retry on 4xx (non-retryable)
  - Body preservation across retries
  - Retry-After header parsing (5s delay verified)
  - Context cancellation during retry sleep
  - Mixed 429 and 5xx sequence
  - Nil base transport defaults
  - `parseRetryAfter` — 7 cases (empty, 0, negative, non-numeric, valid values)
  - `exponentialBackoff` — verifies range for 3 attempts
  - `contextSleep` — completion and cancellation
- Created `internal/garminapi/circuitbreaker_test.go` with comprehensive tests:
  - Starts closed, allows requests
  - Opens after threshold failures
  - Transitions to half-open after reset timeout
  - Success in half-open resets to closed
  - Failure in half-open reopens circuit
  - Success during closed resets failure count
  - Default values for zero threshold/timeout
  - Error message verification

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (garminapi: 1.6s)
- `make build` — success

**Issues:**
- None

### 2026-02-11 — Task 15: Activities API methods

**Changes:**
- Created `internal/garminapi/activities.go` with all activity API methods:
  - `ActivityDownloadFormat` type with constants: `FormatFIT`, `FormatTCX`, `FormatGPX`, `FormatKML`, `FormatCSV`
  - `CountActivities(ctx)` — GET `/activitylist-service/activities/count`, returns int
  - `GetActivities(ctx, start, limit, activityType)` — GET with pagination and optional type filter
  - `GetActivity(ctx, activityID)` — GET single activity by ID
  - `GetActivityDetails(ctx, activityID, maxChart, maxPoly)` — GET with chart/polyline size params
  - `GetActivitySplits`, `GetActivityTypedSplits`, `GetActivitySplitSummaries` — split data endpoints
  - `GetActivityWeather`, `GetActivityHRZones`, `GetActivityPowerZones` — activity metrics
  - `GetActivityExerciseSets` — exercise sets (strength training)
  - `GetActivityGear` — gear linked to activity via gear-service
  - `SearchActivities(ctx, start, limit, startDate, endDate)` — date-range search
  - `DownloadActivity(ctx, activityID, format)` — download in FIT/TCX/GPX/KML/CSV
  - `UploadActivity(ctx, filePath)` — multipart file upload (FIT/GPX/TCX)
  - `CreateManualActivity(ctx, name, type, distance, duration, startTime)` — POST manual entry
  - `RenameActivity(ctx, activityID, name)` — PUT to rename
  - `RetypeActivity(ctx, activityID, typeID, typeKey, parentTypeID)` — PUT to change type
  - `DeleteActivity(ctx, activityID)` — DELETE activity
  - `downloadPath()` helper mapping format to URL path
  - `doUpload()`/`doUploadBytes()` internal methods with 401 retry support
- Created `internal/garminapi/activities_test.go` with comprehensive tests:
  - CountActivities: success, invalid JSON, server error
  - GetActivities: success, with type filter
  - GetActivity: success with field verification
  - GetActivityDetails: success with maxChartSize/maxPolylineSize query params
  - All sub-resource methods: splits, typed splits, split summaries, weather, HR zones, power zones, exercise sets, gear (path and query param verification)
  - SearchActivities: with end date, without end date
  - DownloadActivity: all 5 formats (FIT/TCX/GPX/KML/CSV), invalid format error
  - downloadPath: all 5 format paths, invalid format
  - UploadActivity: FIT success (multipart form verification), GPX success, invalid format, file not found, server error, 401 with token refresh
  - CreateManualActivity: success with full JSON payload verification
  - RenameActivity: success, invalid ID
  - RetypeActivity: success with typeId/typeKey/parentTypeId, invalid ID
  - DeleteActivity: success (204 No Content), server error (404)

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (garminapi: 1.7s)
- `make build` — success

**Issues:**
- None

### 2026-02-11 — Task 16: Activities list/count/search commands

**Changes:**
- Created `internal/cmd/client.go` with `resolveClient()` helper to load tokens from keyring and create API client
  - `newClientFn` function variable for test injection of API client creation
  - Error messages guide user to `gc auth login` when credentials are missing
- Created `internal/cmd/activities.go` with `ActivitiesCmd` struct grouping three subcommands:
  - `ActivitiesListCmd` — `gc activities` (default) with `--limit` (default 20), `--start`, `--type` flags
  - `ActivitiesCountCmd` — `gc activities count` showing total activity count
  - `ActivitiesSearchCmd` — `gc activities search --start-date --end-date` for date range queries
- Table output with columns: ID, DATE, TYPE, NAME, DISTANCE, DURATION, CALORIES
  - Distance formatted as km (meters / 1000), duration as H:MM:SS or M:SS
  - Activity type extracted from nested `activityType.typeKey` field
- JSON output: raw Garmin API JSON passthrough
- Plain output: TSV format via `outfmt.WritePlain()`
- Formatting helpers: `parseActivities()`, `formatActivityRows()`, `jsonString()`, `jsonFloat()`, `activityTypeKey()`, `formatDate()`, `formatDistance()`, `formatDuration()`, `formatCalories()`
- Wired `ActivitiesCmd` into CLI struct in `root.go`
- Created `internal/cmd/activities_test.go` with comprehensive tests:
  - Execute-level tests: `activities --help`, `activities count --help`, `activities search --help`
  - ActivitiesList: NoAccount, NotFound, Table, JSON, WithType (query param verification)
  - ActivitiesCount: NoAccount, Success, JSON
  - ActivitiesSearch: NoAccount, Success (date param verification), JSON
  - resolveClient: NoAccount, NotFound, Success (token verification)
  - Formatting: parseActivities (valid, invalid, empty), formatActivityRows (field extraction), formatDistance (4 cases), formatDuration (5 cases), formatCalories (3 cases), formatDate (3 cases), activityTypeKey (4 cases), jsonString (6 cases), jsonFloat (4 cases)
  - Mock HTTP server with activity list and count endpoints

**Commands run:**
- `make build` — success
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (cmd: 1.9s)
- `./bin/gc activities --help` — shows list, count, search subcommands
- `./bin/gc activities list --help` — shows limit, start, type flags
- `./bin/gc activities search --help` — shows start-date, end-date flags

**Issues:**
- None

### 2026-02-11 — Task 17: Activity detail commands

**Changes:**
- Created `internal/cmd/activity.go` with `ActivityCmd` struct grouping 10 subcommands:
  - `ActivitySummaryCmd` — `gc activity <id>` (default) shows key-value summary: name, type, date, distance, duration, avg HR, max HR, calories
  - `ActivityDetailsCmd` — `gc activity <id> details` with `--max-chart` and `--max-poly` flags for chart/polyline data point limits
  - `ActivitySplitsCmd` — `gc activity <id> splits`
  - `ActivityTypedSplitsCmd` — `gc activity <id> typed-splits`
  - `ActivitySplitSummariesCmd` — `gc activity <id> split-summaries`
  - `ActivityWeatherCmd` — `gc activity <id> weather`
  - `ActivityHRZonesCmd` — `gc activity <id> hr-zones`
  - `ActivityPowerZonesCmd` — `gc activity <id> power-zones`
  - `ActivityExerciseSetsCmd` — `gc activity <id> exercise-sets`
  - `ActivityGearCmd` — `gc activity <id> gear`
- Summary command supports all three output modes:
  - JSON: raw API passthrough via `outfmt.WriteJSON`
  - Table: label-value pairs (NAME, TYPE, DATE, DISTANCE, DURATION, AVG HR, MAX HR, CALORIES) via `outfmt.WriteTable`
  - Plain: TSV label-value pairs via `outfmt.WritePlain`
- Sub-resource commands output JSON via `outfmt.WriteJSON` (complex nested structures)
- Added `formatActivitySummary()` and `formatHeartRate()` helper functions
- Wired `ActivityCmd` into CLI struct in `root.go`
- Created `internal/cmd/activity_test.go` with comprehensive tests:
  - Execute-level tests: `activity --help`, `activity details --help`, `activity splits --help`, `activity weather --help`, `activity hr-zones --help`, `activity gear --help`
  - Summary tests: NoAccount, NotFound, Table, JSON, Plain
  - Details tests: NoAccount, Success, QueryParams (maxChartSize/maxPolylineSize verification)
  - Sub-resource tests: Splits, TypedSplits, SplitSummaries, Weather, HRZones, PowerZones, ExerciseSets, Gear (all success cases)
  - Gear activity ID verification test
  - formatActivitySummary: full fields, missing fields (verifies `-` placeholders)
  - formatHeartRate: 4 cases (zero, normal, decimal, low)
  - Mock HTTP server with handlers for all 10 activity sub-resource endpoints

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (cmd: 2.0s)
- `make build` — success
- `./bin/gc activity --help` — shows all 10 subcommands

**Issues:**
- None

### 2026-02-11 — Task 18: Activity download command

**Changes:**
- Created `internal/cmd/activity_download.go` with `ActivityDownloadCmd`:
  - `gc activity <id> download` with `--format` flag (fit, gpx, tcx, kml, csv; default fit)
  - `--output` / `-o` flag for custom output filename
  - Default filename: `activity_{id}.{format}` via `defaultActivityFilename()` helper
  - FIT download handling: extracts `.fit` file from zip archive via `extractFIT()`
  - `extractFIT()` uses `archive/zip` to read zip in memory, finds first `.fit` entry by extension (case-insensitive)
  - Prints download path and size on success via `UI.Successf()`
- Wired `ActivityDownloadCmd` into `ActivityCmd` struct in `activity.go` (now 11 subcommands)
- Created `internal/cmd/activity_download_test.go` with comprehensive tests:
  - Test helpers: `makeZipWithFIT()`, `makeZipWithoutFIT()`, `downloadTestServer()` (mock endpoints for all 5 formats)
  - Execute-level: `activity download --help`
  - Download tests: NoAccount, FIT (verifies zip extraction), GPX (verifies passthrough), TCX, DefaultFilename (verifies `activity_{id}.{format}`), ServerError
  - extractFIT tests: Success, NoFitFile, InvalidZip, EmptyZip
  - defaultActivityFilename tests: all 5 formats

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (cmd: 2.1s)
- `make build` — success
- `./bin/gc activity download --help` — shows format, output flags
- `./bin/gc activity --help` — shows all 11 subcommands including download

**Issues:**
- None

### 2026-02-11 — Task 19: Activity upload, create, and modify commands

**Changes:**
- Created `internal/cmd/confirm.go` with confirmation prompt helper for destructive actions:
  - `confirm(w, prompt, force)` — prompts user for `[y/N]` confirmation, returns true on `y`/`yes`
  - `--force` bypass skips the prompt entirely
  - `confirmReader` package variable for test injection (defaults to `os.Stdin`)
  - Case-insensitive, whitespace-trimmed input handling
- Created `internal/cmd/activity_upload.go` with `ActivityUploadCmd`:
  - `gc activity <id> upload <file>` with Kong `type:"existingfile"` validation
  - Supports FIT, GPX, TCX file formats (validated by API client)
  - JSON mode outputs raw API response; default prints success message
- Created `internal/cmd/activity_create.go` with `ActivityCreateCmd`:
  - `gc activity create --name --type --distance --duration` for manual activity entries
  - `--name` and `--type` (activity type key) and `--duration` are required flags
  - `--distance` in meters is optional (default 0)
  - Duration uses Go's `time.Duration` parsing (e.g. `30m`, `1h15m`)
  - Generates start time as current UTC time
  - Extracts and displays created activity ID from API response
- Created `internal/cmd/activity_modify.go` with three subcommands:
  - `ActivityRenameCmd` — `gc activity <id> rename <name>` to rename an activity
  - `ActivityRetypeCmd` — `gc activity <id> retype --type-id --type-key [--parent-type-id]` to change activity type
  - `ActivityDeleteCmd` — `gc activity <id> delete [-f]` with confirmation prompt (skippable with `--force`)
- Wired all 5 new subcommands into `ActivityCmd` in `activity.go` (now 16 total subcommands)
- Created `internal/cmd/confirm_test.go` with 8 tests:
  - Force bypass, yes, yes (full word), no, empty input, EOF, case insensitive, whitespace trimming
- Created `internal/cmd/activity_upload_test.go` with tests:
  - Execute-level: `activity upload --help`
  - Upload tests: NoAccount, FIT success (multipart verification), JSON mode, ServerError
  - Mock upload server with multipart/form-data validation
- Created `internal/cmd/activity_create_test.go` with tests:
  - Execute-level: `activity create --help`
  - Create tests: NoAccount, Success (activity ID in message), JSON mode, VerifyPayload (field-level validation of activityName, typeKey, distance, duration), ServerError
- Created `internal/cmd/activity_modify_test.go` with tests:
  - Execute-level: `activity rename --help`, `activity retype --help`, `activity delete --help`
  - Rename tests: NoAccount, Success, JSON, InvalidID
  - Retype tests: NoAccount, Success, JSON, VerifyPayload (typeId/typeKey/parentTypeId)
  - Delete tests: NoAccount, Success (with --force), Cancelled (user says no), ConfirmYes (user says yes), ServerError

**Commands run:**
- `make fmt` — clean (after fixing `errcheck` on `fmt.Fprintf`)
- `make lint` — 0 issues
- `make test` — all pass (cmd: 2.0s)
- `make build` — success
- `make ci` — all green
- `./bin/gc activity --help` — shows all 16 subcommands

**Issues:**
- `errcheck` linter flagged unchecked `fmt.Fprintf` return in confirm.go; fixed with `_, _ =` discard pattern

### 2026-02-11 — Task 20: Workouts API and commands

**Changes:**
- Created `internal/garminapi/workouts.go` with all workout API methods:
  - `GetWorkouts(ctx, start, limit)` — GET `/workout-service/workouts` with pagination
  - `GetWorkout(ctx, workoutID)` — GET `/workout-service/workout/{id}` for single workout
  - `DownloadWorkout(ctx, workoutID)` — GET `/workout-service/workout/FIT/{id}` for FIT download
  - `UploadWorkout(ctx, workoutJSON)` — POST `/workout-service/workout` with JSON body
  - `GetScheduledWorkout(ctx, scheduleID)` — GET `/workout-service/schedule/{id}`
  - `DeleteWorkout(ctx, workoutID)` — DELETE `/workout-service/workout/{id}`
- Created `internal/cmd/workouts.go` with `WorkoutsCmd` struct grouping 6 subcommands:
  - `WorkoutsListCmd` — `gc workouts` (default) with `--limit` and `--start` flags
  - `WorkoutDetailCmd` — `gc workouts detail <id>` for full workout JSON
  - `WorkoutDownloadCmd` — `gc workouts download <id>` with `--output` flag, default `workout_{id}.fit`
  - `WorkoutUploadCmd` — `gc workouts upload <file>` reading JSON file, validates JSON before upload
  - `WorkoutScheduleCmd` — `gc workouts schedule <id>` for scheduled workout view
  - `WorkoutDeleteCmd` — `gc workouts delete <id> [-f]` with confirmation prompt
- Table output for list: ID, NAME, TYPE, OWNER columns
  - `parseWorkouts()` handles both array and object-wrapped responses
  - `workoutSportType()` extracts nested `sportType.sportTypeKey`
  - `workoutOwner()` extracts nested `owner.displayName`
- Wired `WorkoutsCmd` into CLI struct in `root.go`
- Created `internal/garminapi/workouts_test.go` with comprehensive tests:
  - GetWorkouts: success (path/params verification), server error
  - GetWorkout: success (field verification), not found
  - DownloadWorkout: success, server error, 401 with token refresh
  - UploadWorkout: success (payload verification), server error
  - GetScheduledWorkout: success
  - DeleteWorkout: success (204 No Content), not found
- Created `internal/cmd/workouts_test.go` with comprehensive tests:
  - Execute-level: 7 help tests (workouts, list, detail, download, upload, schedule, delete)
  - List: NoAccount, Table, JSON, Plain
  - Detail: NoAccount, Success
  - Download: NoAccount, Success (file content verification), DefaultFilename, ServerError
  - Upload: NoAccount, Success, JSON mode, InvalidJSON
  - Schedule: NoAccount, Success
  - Delete: NoAccount, Success (with --force), Cancelled, ConfirmYes, ServerError
  - parseWorkouts: Array, Wrapper, InvalidJSON, EmptyArray
  - formatWorkoutRows: full and partial data
  - workoutSportType: 4 cases
  - workoutOwner: 4 cases

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (garminapi: 2.0s, cmd: 2.2s)
- `make build` — success
- `./bin/gc workouts --help` — shows all 6 subcommands
- `./bin/gc --help` — shows workouts in top-level commands

**Issues:**
- `confirm()` returns `(bool, error)` not just `bool`; fixed WorkoutDeleteCmd to handle both return values matching ActivityDeleteCmd pattern

### 2026-02-12 — Task 21: Health API methods

**Changes:**
- Created `internal/garminapi/health.go` with all health API methods:
  - `GetDailySummary(ctx, displayName, date)` — GET `/usersummary-service/usersummary/daily/{displayName}?calendarDate={date}`
  - `GetSteps(ctx, displayName, date)` — GET `/wellness-service/wellness/dailySummaryChart/{displayName}?date={date}`
  - `GetDailySteps(ctx, startDate, endDate)` — GET `/usersummary-service/stats/steps/daily/{start}/{end}`
  - `GetWeeklySteps(ctx, endDate, weeks)` — GET `/usersummary-service/stats/steps/weekly/{end}/{weeks}`
  - `GetHeartRate(ctx, displayName, date)` — GET `/wellness-service/wellness/dailyHeartRate/{displayName}?date={date}`
  - `GetRestingHeartRate(ctx, displayName, date)` — GET `/userstats-service/wellness/daily/{displayName}?fromDate=&untilDate=&metricId=60`
  - `GetFloors(ctx, date)` — GET `/wellness-service/wellness/floorsChartData/daily/{date}`
  - `GetSleep(ctx, displayName, date)` — GET `/wellness-service/wellness/dailySleepData/{displayName}?date=&nonSleepBufferMinutes=60`
  - `GetStress(ctx, date)` — GET `/wellness-service/wellness/dailyStress/{date}`
  - `GetWeeklyStress(ctx, endDate, weeks)` — GET `/usersummary-service/stats/stress/weekly/{end}/{weeks}`
  - `GetRespiration(ctx, date)` — GET `/wellness-service/wellness/daily/respiration/{date}`
  - `GetSPO2(ctx, date)` — GET `/wellness-service/wellness/daily/spo2/{date}`
  - `GetHRV(ctx, date)` — GET `/hrv-service/hrv/{date}`
  - `GetBodyBattery(ctx, startDate, endDate)` — GET `/wellness-service/wellness/bodyBattery/reports/daily?startDate=&endDate=`
  - `GetIntensityMinutes(ctx, date)` — GET `/wellness-service/wellness/daily/im/{date}`
  - `GetWeeklyIntensityMinutes(ctx, startDate, endDate)` — GET `/usersummary-service/stats/im/weekly/{start}/{end}`
  - `GetTrainingReadiness(ctx, date)` — GET `/metrics-service/metrics/trainingreadiness/{date}`
  - `GetTrainingStatus(ctx, date)` — GET `/metrics-service/metrics/trainingstatus/aggregated/{date}`
  - `GetFitnessAge(ctx, date)` — GET `/fitnessage-service/fitnessage/{date}`
  - `GetMaxMetrics(ctx, date)` — GET `/metrics-service/metrics/maxmet/daily/{date}/{date}`
  - `GetLactateThreshold(ctx)` — GET `/biometric-service/biometric/latestLactateThreshold`
  - `GetCyclingFTP(ctx)` — GET `/biometric-service/biometric/latestFunctionalThresholdPower/CYCLING`
  - `GetRacePredictions(ctx, startDate, endDate)` — current or range with daily type
  - `GetEnduranceScore(ctx, date)` — GET `/metrics-service/metrics/endurancescore?calendarDate={date}`
  - `GetHillScore(ctx, date)` — GET `/metrics-service/metrics/hillscore?calendarDate={date}`
  - `GetAllDayEvents(ctx, date)` — GET `/wellness-service/wellness/dailyEvents?calendarDate={date}`
  - `GetLifestyleLogging(ctx, date)` — GET `/lifestylelogging-service/dailyLog/{date}`
- Created `internal/garminapi/health_test.go` with comprehensive tests:
  - GetDailySummary: success (path + query param verification), server error
  - GetSteps: success (path + date param verification)
  - GetDailySteps: success (path with date range verification)
  - GetWeeklySteps: success (path with weeks param verification)
  - GetHeartRate: success (path + date param verification)
  - GetRestingHeartRate: success (path + fromDate/untilDate/metricId verification)
  - GetFloors: success (date in path verification)
  - GetSleep: success (path + date + nonSleepBufferMinutes verification)
  - GetStress: success (date in path verification)
  - GetWeeklyStress: success (path with weeks param)
  - GetRespiration: success (date in path)
  - GetSPO2: success (value verification)
  - GetHRV: success (date in path)
  - GetBodyBattery: success (startDate/endDate params), date range (multi-entry)
  - GetIntensityMinutes: success (date in path)
  - GetWeeklyIntensityMinutes: success (date range in path)
  - GetTrainingReadiness: success (score verification)
  - GetTrainingStatus: success (date in path)
  - GetFitnessAge: success (fitnessAge value verification)
  - GetMaxMetrics: success (repeated date in path)
  - GetLactateThreshold: success (value verification)
  - GetCyclingFTP: success (value verification)
  - GetRacePredictions: current (no dates), date range (path + type param)
  - GetEnduranceScore: success (calendarDate param verification)
  - GetHillScore: success (calendarDate param verification)
  - GetAllDayEvents: success (calendarDate param verification)
  - GetLifestyleLogging: success (date in path), server error

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (garminapi: 1.9s)
- `make build` — success

**Issues:**
- None

### 2026-02-12 — Task 22: Health basic commands

**Changes:**
- Created `internal/cmd/health.go` with `HealthCmd` struct grouping 8 subcommands:
  - `HealthSummaryCmd` — `gc health [date]` (default) shows daily health summary
  - `HealthStepsCmd` — `gc health steps [date]` with `daily --start --end` and `weekly --end --weeks` subcommands
  - `HealthHRCmd` — `gc health hr [date]` shows heart rate data
  - `HealthRHRCmd` — `gc health rhr [date]` shows resting heart rate data
  - `HealthFloorsCmd` — `gc health floors [date]` shows floors climbed data
  - `HealthSleepCmd` — `gc health sleep [date]` shows sleep data
  - `HealthStressCmd` — `gc health stress [date]` with `weekly --end --weeks` subcommand
  - `HealthRespirationCmd` — `gc health respiration [date]` shows respiration data
- Implemented `resolveDate(s string)` helper with relative date support:
  - Empty or "today" → today's date
  - "yesterday" → yesterday's date
  - "Nd" (e.g. "3d", "7d") → N days ago
  - "YYYY-MM-DD" → explicit date with validation
  - Case-insensitive, whitespace-trimmed
- `nowFn` package variable for test injection of current time
- `writeHealthJSON()` helper for consistent JSON output across all health subcommands
- Health commands extract `displayName` from client tokens for API calls that require it
- Weekly commands default `endDate` to today when not specified
- Wired `HealthCmd` into CLI struct in `root.go`
- Created `internal/cmd/health_test.go` with comprehensive tests:
  - Execute-level tests: 11 help tests (health, steps, hr, sleep, stress, floors, rhr, respiration, steps daily, steps weekly, stress weekly)
  - Summary tests: NoAccount, Success, WithDate (date param verification), DisplayNameInPath, ServerError, InvalidDate
  - Steps tests: NoAccount, Success, Daily (date range path verification), Weekly (success), WeeklyDefaultEndDate (nowFn verification)
  - HR tests: NoAccount, Success
  - RHR tests: NoAccount, Success
  - Floors tests: NoAccount, Success
  - Sleep tests: NoAccount, Success
  - Stress tests: NoAccount, Success, Weekly, WeeklyDefaultEndDate
  - Respiration tests: NoAccount, Success
  - resolveDate tests: 14 cases (empty, today, yesterday, relative, explicit, whitespace, invalid)
  - writeHealthJSON nil data test
  - `overrideNowFn()` test helper for fixed time injection

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (cmd: 1.9s)
- `make build` — success
- `./bin/gc health --help` — shows all 11 subcommands
- `./bin/gc --help` — shows health in top-level commands

**Issues:**
- None

### 2026-02-12 — Task 23: Health advanced commands

**Changes:**
- Created `internal/cmd/health_advanced.go` with 16 advanced health subcommands:
  - `HealthSPO2Cmd` — `gc health spo2 [date]` shows SpO2 (blood oxygen) data
  - `HealthHRVCmd` — `gc health hrv [date]` shows heart rate variability data
  - `HealthBodyBatteryCmd` — `gc health body-battery [date]` with `range --start --end` subcommand
  - `HealthIntensityMinutesCmd` — `gc health intensity-minutes [date]` with `weekly --start --end` subcommand
  - `HealthTrainingReadinessCmd` — `gc health training-readiness [date]`
  - `HealthTrainingStatusCmd` — `gc health training-status [date]`
  - `HealthFitnessAgeCmd` — `gc health fitness-age [date]`
  - `HealthMaxMetricsCmd` — `gc health max-metrics [date]` for VO2max data
  - `HealthLactateThresholdCmd` — `gc health lactate-threshold` (no date, returns latest)
  - `HealthCyclingFTPCmd` — `gc health cycling-ftp` (no date, returns latest)
  - `HealthRacePredictionsCmd` — `gc health race-predictions` with `range --start --end` subcommand
  - `HealthEnduranceScoreCmd` — `gc health endurance-score [date]`
  - `HealthHillScoreCmd` — `gc health hill-score [date]`
  - `HealthEventsCmd` — `gc health events [date]` for daily wellness events
  - `HealthLifestyleCmd` — `gc health lifestyle [date]` for lifestyle logging data
- Updated `HealthCmd` in `health.go` to include all 16 new subcommands (total: 24 health subcommands)
- All commands follow existing patterns: `resolveDate()` for date parsing, `writeHealthJSON()` for output
- Body battery and race predictions support range subcommands with `--start`/`--end` flags
- Lactate threshold and cycling FTP take no date argument (always return latest)
- Created `internal/cmd/health_advanced_test.go` with comprehensive tests:
  - 18 Execute-level help tests for all new subcommands
  - NoAccount and Success tests for all 16 command types
  - Range command tests with date parameter verification (body battery range, intensity minutes weekly, race predictions range)
  - Date parameter verification tests (endurance score, hill score, events)
  - Server error test (lifestyle)
  - Mock HTTP server covering all 16 health API endpoints

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (cmd: 2.6s)
- `make build` — success
- `./bin/gc health --help` — shows all 30 subcommands (8 basic + 16 advanced + 6 sub-subcommands)

**Issues:**
- None

### 2026-02-12 — Task 24: Body composition API and commands

**Changes:**
- Created `internal/garminapi/body.go` with all body composition API methods:
  - `GetBodyComposition(ctx, startDate, endDate)` — GET `/weight-service/weight/dateRange?startDate=&endDate=`
  - `GetWeighIns(ctx, startDate, endDate)` — GET `/weight-service/weight/range/{start}/{end}?includeAll=true`
  - `GetDailyWeighIns(ctx, date)` — GET `/weight-service/weight/dayview/{date}?includeAll=true`
  - `AddWeight(ctx, weight, unitKey, dateTimestamp, gmtTimestamp)` — POST `/weight-service/user-weight`
  - `DeleteWeight(ctx, date, version)` — DELETE `/weight-service/weight/{date}/byversion/{version}`
  - `GetBloodPressure(ctx, startDate, endDate)` — GET `/bloodpressure-service/bloodpressure/range/{start}/{end}?includeAll=true`
  - `AddBloodPressure(ctx, systolic, diastolic, pulse, timestampLocal, timestampGMT, notes)` — POST `/bloodpressure-service/bloodpressure`
  - `DeleteBloodPressure(ctx, date, version)` — DELETE `/bloodpressure-service/bloodpressure/{date}/{version}`
- Created `internal/cmd/body.go` with `BodyCmd` struct grouping 8 subcommands:
  - `BodyCompositionCmd` — `gc body [date]` (default) with `--start/--end` for range queries
  - `BodyWeighInsCmd` — `gc body weigh-ins --start --end` for weigh-in date range
  - `BodyDailyWeighInsCmd` — `gc body daily-weigh-ins [date]` for single day view
  - `BodyAddWeightCmd` — `gc body add-weight <value>` with `--unit` (kg/lbs, default kg)
  - `BodyDeleteWeightCmd` — `gc body delete-weight <pk> --date` with confirmation prompt
  - `BodyBloodPressureCmd` — `gc body blood-pressure --start --end` for BP date range
  - `BodyAddBPCmd` — `gc body add-blood-pressure --systolic --diastolic --pulse [--notes]`
  - `BodyDeleteBPCmd` — `gc body delete-blood-pressure <version> --date` with confirmation prompt
- Wired `BodyCmd` into CLI struct in `root.go`
- Created `internal/garminapi/body_test.go` with comprehensive tests:
  - GetBodyComposition: success (startDate/endDate param verification), server error
  - GetWeighIns: success (path + includeAll param verification)
  - GetDailyWeighIns: success (path + includeAll param verification)
  - AddWeight: success (POST payload verification: value, unitKey, sourceType), server error
  - DeleteWeight: success (DELETE path verification), not found
  - GetBloodPressure: success (path + includeAll param verification), server error
  - AddBloodPressure: success (POST payload verification: systolic, diastolic, pulse, sourceType, notes), server error
  - DeleteBloodPressure: success (DELETE path verification), not found
- Created `internal/cmd/body_test.go` with comprehensive tests:
  - 9 Execute-level help tests (body, composition, weigh-ins, daily-weigh-ins, add-weight, delete-weight, blood-pressure, add-blood-pressure, delete-blood-pressure)
  - Composition: NoAccount, Success, WithDateRange (date param verification), InvalidDate
  - WeighIns: NoAccount, Success
  - DailyWeighIns: NoAccount, Success
  - AddWeight: NoAccount, Success, JSON, VerifyPayload (value/unitKey/sourceType/timestamps)
  - DeleteWeight: NoAccount, Success (with --force), Cancelled, ConfirmYes
  - BloodPressure: NoAccount, Success
  - AddBP: NoAccount, Success, JSON, VerifyPayload (systolic/diastolic/pulse/notes/sourceType)
  - DeleteBP: NoAccount, Success (with --force), Cancelled, ConfirmYes, ServerError

**Commands run:**
- `make fmt` — clean (gofumpt adjusted struct alignment)
- `make lint` — 0 issues
- `make test` — all pass (garminapi: 1.6s, cmd: 2.5s)
- `make build` — success
- `make ci` — all green
- `./bin/gc body --help` — shows all 8 subcommands
- `./bin/gc --help` — shows body in top-level commands

**Issues:**
- None

### 2026-02-12 — Task 25: FIT file encoder

**Changes:**
- Created `internal/fit/encoder.go` porting Python `FitEncoderWeight` class to Go:
  - FIT protocol binary encoding with 12-byte file header (header size, protocol version 1.0, profile version 108, `.FIT` signature)
  - CRC-16 calculation over entire file using nibble-based lookup table from FIT specification
  - `baseType` system matching FIT protocol: `enum`, `uint8`, `uint16`, `uint32`, `uint32z` with correct field IDs, sizes, and invalid sentinels
  - `fieldDef` structs with field number, base type, optional value pointer, and scale factor
  - Record headers with definition bit (bit 6) and local message type (bits 0-3)
  - Definition messages: reserved(1) + architecture(1, little-endian) + global message number(2) + field count(1) + field definitions
  - `NewEncoder()` constructor writing initial header
  - `WriteFileInfo(t)` — file_id message (serial number, time created, manufacturer, product, number, type=9/weight)
  - `WriteFileCreator()` — file_creator message (software version, hardware version)
  - `WriteDeviceInfo(t)` — device_info message (12 fields, definition written once)
  - `WriteWeightScale(data)` — weight_scale message (13 fields: timestamp, weight, body fat %, hydration %, visceral fat mass, bone mass, muscle mass, basal/active met, physique rating, metabolic age, visceral fat rating, BMI; definition written once)
  - `Finish()` — rewrites header with correct data size, appends CRC
  - `EncodeWeightScale(data)` convenience function for complete file generation
  - `WeightScaleData` struct with all optional fields as `*float64` pointers
  - FIT timestamp conversion: seconds since UTC 1989-12-31 00:00:00 (epoch 631065600)
  - Scale factors matching Python implementation: weight×100, fat×100, met×4, BMI×10, etc.
  - Values rounded via `math.Round` before integer encoding
  - Nil values encoded as base type invalid sentinels (0xFFFF for uint16, 0xFF for uint8, etc.)
- Added `UploadBodyComposition(ctx, fitData)` method to `internal/garminapi/body.go`:
  - Uploads FIT-encoded body composition data via multipart/form-data POST to `/upload-service/upload`
  - Filename: `body_composition.fit` (matching Python library)
  - Reuses existing `doUpload()` infrastructure with 401 retry support
- Added `BodyAddCompositionCmd` to `internal/cmd/body.go`:
  - `gc body add-composition <weight>` with 11 optional flags: `--body-fat`, `--percent-hydration`, `--visceral-fat-mass`, `--bone-mass`, `--muscle-mass`, `--basal-met`, `--active-met`, `--physique-rating`, `--metabolic-age`, `--visceral-fat-rating`, `--bmi`
  - Encodes body composition data into FIT format using `fit.EncodeWeightScale()`
  - Uploads via `client.UploadBodyComposition()`
  - Supports `--json` mode for raw API response output
- Created `internal/fit/encoder_test.go` with comprehensive tests:
  - Timestamp conversion, CRC calculation, record headers, field definitions
  - Byte-level verification of all weight scale fields with scale factors
  - Full file validation (header, data size, CRC, determinism)
  - All base type invalid sentinels and scaled value encoding
- Added tests to `internal/garminapi/body_test.go`:
  - `TestUploadBodyComposition_Success` — POST verification, multipart form, filename
  - `TestUploadBodyComposition_ServerError` — GarminAPIError on 500
- Added tests to `internal/cmd/body_test.go`:
  - Help, NoAccount, Success, JSON, WithAllFields (FIT signature check), ServerError
  - Updated `bodyTestServer()` with `/upload-service/upload` handler

**Commands run:**
- `make fmt` — clean (gofumpt adjusted struct alignment)
- `make lint` — 0 issues
- `make test` — all pass (fit: 1.6s, garminapi: cached, cmd: 2.6s)
- `make build` — success
- `make ci` — all green
- `./bin/gc body --help` — shows all 9 subcommands including add-composition
- `./bin/gc body add-composition --help` — shows weight arg and 11 optional flags

**Issues:**
- Test timestamp calculation was initially wrong (off by 1200 seconds); fixed by computing correct FIT timestamp for test date

### 2026-02-12 — Task 26: Devices API and commands

**Changes:**
- Created `internal/garminapi/devices.go` with all device API methods:
  - `GetDevices(ctx)` — GET `/device-service/deviceregistration/devices` for all registered devices
  - `GetDeviceSettings(ctx, deviceID)` — GET `/device-service/deviceservice/device-info/settings/{deviceID}` for device settings
  - `GetPrimaryTrainingDevice(ctx)` — GET `/web-gateway/device-info/primary-training-device` for primary training device info
  - `GetDeviceSolar(ctx, deviceID, startDate, endDate)` — GET `/web-gateway/solar/{deviceID}/{startDate}/{endDate}` for solar charging data
  - `GetDeviceAlarms(ctx)` — composite method: fetches all devices, iterates settings for each, collects alarms; gracefully skips devices where settings fail
  - `GetLastUsedDevice(ctx)` — GET `/device-service/deviceservice/mylastused` for most recently used device
- Created `internal/cmd/devices.go` with `DevicesCmd` struct grouping 6 subcommands:
  - `DevicesListCmd` — `gc devices` (default) lists all registered devices
  - `DeviceSettingsCmd` — `gc devices settings <device-id>` shows device settings
  - `DevicePrimaryCmd` — `gc devices primary` shows primary training device
  - `DeviceSolarCmd` — `gc devices solar <device-id>` with `--start`/`--end` flags (defaults to today)
  - `DeviceAlarmsCmd` — `gc devices alarms` shows alarms from all devices
  - `DeviceLastUsedCmd` — `gc devices last-used` shows last used device
- Wired `DevicesCmd` into CLI struct in `root.go`
- Created `internal/garminapi/devices_test.go` with comprehensive tests:
  - GetDevices: success (path verification, field check), server error
  - GetDeviceSettings: success (path + field verification), not found (404)
  - GetPrimaryTrainingDevice: success (path + field verification)
  - GetDeviceSolar: success (path verification), date range (multi-entry)
  - GetDeviceAlarms: success (3 alarms from 2 devices), no devices, no alarms, settings error (graceful skip)
  - GetLastUsedDevice: success (field verification), server error
- Created `internal/cmd/devices_test.go` with comprehensive tests:
  - 7 Execute-level help tests (devices, list, settings, primary, solar, alarms, last-used)
  - List: NoAccount, Success, ServerError
  - Settings: NoAccount, Success (path verification)
  - Primary: NoAccount, Success
  - Solar: NoAccount, Success, DefaultDates (today default verification)
  - Alarms: NoAccount, Success
  - LastUsed: NoAccount, Success, ServerError

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (garminapi: 1.9s, cmd: 2.6s)
- `make build` — success
- `./bin/gc devices --help` — shows all 6 subcommands
- `./bin/gc --help` — shows devices in top-level commands

**Issues:**
- None

### 2026-02-12 — Task 27: Gear API and commands

**Changes:**
- Created `internal/garminapi/gear.go` with all gear API methods:
  - `GetSocialProfile(ctx)` — GET `/userprofile-service/usersocialprofile` for user profile data (needed to get userProfilePk for gear API calls)
  - `GetGear(ctx, userProfilePK)` — GET `/gear-service/gear/filterGear?userProfilePk={pk}` for all gear items
  - `GetGearStats(ctx, gearUUID)` — GET `/gear-service/gear/stats/{uuid}` for gear usage statistics
  - `GetGearActivities(ctx, gearUUID, limit)` — GET `/activitylist-service/activities/{uuid}/gear?limit={limit}` for activities linked to gear
  - `GetGearDefaults(ctx, userProfileNumber)` — GET `/gear-service/gear/user/{number}/activityTypes` for default gear per activity type
  - `LinkGear(ctx, gearUUID, activityID)` — PUT `/gear-service/gear/link/{uuid}/activity/{id}` to link gear to activity
  - `UnlinkGear(ctx, gearUUID, activityID)` — PUT `/gear-service/gear/unlink/{uuid}/activity/{id}` to unlink gear from activity
- Created `internal/cmd/gear.go` with `GearCmd` struct grouping 6 subcommands:
  - `GearListCmd` — `gc gear` (default) lists all gear for the authenticated user
  - `GearStatsCmd` — `gc gear stats <uuid>` shows gear usage statistics
  - `GearActivitiesCmd` — `gc gear activities <uuid>` with `--limit` flag (default 20)
  - `GearDefaultsCmd` — `gc gear defaults` shows default gear per activity type
  - `GearLinkCmd` — `gc gear link <uuid> <activity-id>` links gear to an activity
  - `GearUnlinkCmd` — `gc gear unlink <uuid> <activity-id>` unlinks gear from an activity
- `getUserProfilePK()` helper fetches social profile and extracts userProfileNumber using `json.Decoder` with `UseNumber()` to preserve numeric precision
- Wired `GearCmd` into CLI struct in `root.go`
- Created `internal/garminapi/gear_test.go` with comprehensive tests:
  - GetSocialProfile: success (path and field verification)
  - GetGear: success (path + userProfilePk param verification), server error
  - GetGearStats: success (path + field verification), not found (404)
  - GetGearActivities: success (path + limit param verification), server error
  - GetGearDefaults: success (path verification)
  - LinkGear: success (PUT method + path verification), not found
  - UnlinkGear: success (PUT method + path verification), not found
- Created `internal/cmd/gear_test.go` with comprehensive tests:
  - 7 Execute-level help tests (gear, list, stats, activities, defaults, link, unlink)
  - List: NoAccount, Success, ServerError, ProfileError
  - Stats: NoAccount, Success (path verification)
  - Activities: NoAccount, Success, LimitParam (limit=50 verification)
  - Defaults: NoAccount, Success
  - Link: NoAccount, Success (PUT method + success message verification), ServerError
  - Unlink: NoAccount, Success (PUT method + success message verification), ServerError
  - getUserProfilePK: Success (numeric precision), MissingField

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (garminapi: 1.9s, cmd: 2.4s)
- `make build` — success
- `./bin/gc gear --help` — shows all 6 subcommands
- `./bin/gc --help` — shows gear in top-level commands

**Issues:**
- JSON number precision: `json.Unmarshal` into `map[string]any` parses numbers as `float64`, causing `12345678` to format as `1.2345678e+07`; fixed by using `json.Decoder` with `UseNumber()` to preserve the original number representation

### 2026-02-12 — Task 28: Goals, badges, challenges, records

**Changes:**
- Created `internal/garminapi/goals.go` with all goals/badges/challenges/records API methods:
  - `GetGoals(ctx, status)` — GET `/goal-service/goal/goals` with optional `?status=` filter
  - `GetBadgesEarned(ctx)` — GET `/badge-service/badge/earned` for earned badges
  - `GetBadgesAvailable(ctx)` — GET `/badge-service/badge/available` for available badges
  - `GetBadgesInProgress(ctx)` — GET `/badge-service/badge/in-progress` for in-progress badges
  - `GetChallenges(ctx)` — GET `/challenge-service/challenge/joined` for joined challenges
  - `GetBadgeChallenges(ctx)` — GET `/badge-service/badge/challenges` for badge challenges
  - `GetPersonalRecords(ctx, ownerDisplayName)` — GET `/personalrecord-service/personalrecord/prs/{name}` for personal records
- Created `internal/cmd/goals.go` with 4 top-level command groups:
  - `GoalsCmd` — `gc goals` with `list` subcommand (default) and `--status` filter flag
  - `BadgesCmd` — `gc badges` with `earned` (default), `available`, `in-progress` subcommands
  - `ChallengesCmd` — `gc challenges` with `list` (default) and `badge` subcommands
  - `RecordsCmd` — `gc records` showing personal records, uses displayName from tokens (falls back to email)
- Wired `GoalsCmd`, `BadgesCmd`, `ChallengesCmd`, `RecordsCmd` into CLI struct in `root.go`
- Created `internal/garminapi/goals_test.go` with comprehensive tests:
  - GetGoals: success, with status filter (query param verification), server error
  - GetBadgesEarned: success (field verification), server error
  - GetBadgesAvailable: success (field verification)
  - GetBadgesInProgress: success (field verification)
  - GetChallenges: success (field verification), server error
  - GetBadgeChallenges: success (field verification)
  - GetPersonalRecords: success (path verification), server error
- Created `internal/cmd/goals_test.go` with comprehensive tests:
  - 9 Execute-level help tests (goals, goals list, badges, badges earned/available/in-progress, challenges, challenges badge, records)
  - GoalsList: NoAccount, Success, WithStatus (query param verification), ServerError
  - BadgesEarned: NoAccount, Success
  - BadgesAvailable: NoAccount, Success
  - BadgesInProgress: NoAccount, Success
  - ChallengesList: NoAccount, Success, ServerError
  - ChallengesBadge: NoAccount, Success
  - Records: NoAccount, Success, DisplayNamePath (path verification), FallbackToEmail (email fallback), ServerError

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass (garminapi: 1.9s, cmd: 2.4s)
- `make build` — success
- `./bin/gc goals --help` — shows list subcommand with --status flag
- `./bin/gc badges --help` — shows earned, available, in-progress subcommands
- `./bin/gc challenges --help` — shows list and badge subcommands
- `./bin/gc records --help` — shows records command
- `./bin/gc --help` — shows goals, badges, challenges, records in top-level commands

**Issues:**
- None

### 2026-02-12 — Task 29: Profile, hydration, training, wellness commands

**Changes:**
- Created `internal/garminapi/profile.go` with profile API methods:
  - `GetProfile(ctx)` — GET `/userprofile-service/usersocialprofile` for user social profile
  - `GetUserSettings(ctx)` — GET `/userprofile-service/userprofile/user-settings` for user settings
- Created `internal/garminapi/hydration.go` with hydration API methods:
  - `GetHydration(ctx, date)` — GET `/usersummary-service/usersummary/hydration/daily/{date}` for daily hydration data
  - `AddHydration(ctx, date, amountML)` — PUT `/usersummary-service/usersummary/hydration/log` with `valueInML` and `calendarDate` payload
- Created `internal/garminapi/training.go` with training API methods:
  - `GetTrainingPlans(ctx, locale)` — GET `/trainingplan-service/trainingplan/plans?locale={locale}` (defaults to "en")
  - `GetTrainingPlan(ctx, planID)` — GET `/trainingplan-service/trainingplan/plan/{planID}`
- Created `internal/garminapi/wellness.go` with wellness and reload API methods:
  - `GetMenstrualCycleData(ctx, startDate, endDate)` — GET `/periodichealth-service/menstrualcycle/dayview/{start}/{end}`
  - `GetMenstrualCycleSummary(ctx, startDate, endDate)` — GET `/periodichealth-service/menstrualcycle/summary/{start}/{end}`
  - `GetPregnancySummary(ctx)` — GET `/periodichealth-service/pregnancy/summary`
  - `RequestReload(ctx, date)` — GET `/wellness-service/wellness/epoch/request/{date}`
- Created `internal/cmd/profile.go` with `ProfileCmd` struct grouping 2 subcommands:
  - `ProfileViewCmd` — `gc profile` (default) shows user profile
  - `ProfileSettingsCmd` — `gc profile settings` shows user settings
- Created `internal/cmd/hydration.go` with `HydrationCmd` struct grouping 2 subcommands:
  - `HydrationViewCmd` — `gc hydration [date]` (default) shows hydration data with date support
  - `HydrationAddCmd` — `gc hydration add <amount_ml>` with optional `--date` flag
- Created `internal/cmd/training.go` with `TrainingCmd` struct grouping 2 subcommands:
  - `TrainingPlansCmd` — `gc training` (default) lists training plans with `--locale` flag
  - `TrainingPlanCmd` — `gc training plan <id>` shows a specific training plan
- Created `internal/cmd/wellness.go` with `WellnessCmd` struct grouping 3 subcommands:
  - `WellnessMenstrualCycleCmd` — `gc wellness menstrual-cycle --start-date --end-date`
  - `WellnessMenstrualSummaryCmd` — `gc wellness menstrual-summary --start-date --end-date`
  - `WellnessPregnancySummaryCmd` — `gc wellness pregnancy-summary`
- Created `internal/cmd/reload.go` with `ReloadCmd`:
  - `gc reload [date]` requests data reload for a date (defaults to today)
- Wired `ProfileCmd`, `HydrationCmd`, `TrainingCmd`, `WellnessCmd`, `ReloadCmd` into CLI struct in `root.go`
- Created API test files with comprehensive tests:
  - `profile_test.go`: GetProfile (success, server error), GetUserSettings (success, server error)
  - `hydration_test.go`: GetHydration (success, server error), AddHydration (success with payload verification, server error)
  - `training_test.go`: GetTrainingPlans (success, default locale, custom locale, server error), GetTrainingPlan (success, not found)
  - `wellness_test.go`: GetMenstrualCycleData (success, server error), GetMenstrualCycleSummary (success), GetPregnancySummary (success, server error), RequestReload (success, server error)
- Created command test files with comprehensive tests:
  - `profile_test.go`: 2 help tests, ProfileView (NoAccount, Success, ServerError), ProfileSettings (NoAccount, Success)
  - `hydration_test.go`: 2 help tests, HydrationView (NoAccount, Success, WithDate, InvalidDate, ServerError), HydrationAdd (NoAccount, Success, WithDate, ServerError)
  - `training_test.go`: 3 help tests, TrainingPlans (NoAccount, Success, WithLocale, ServerError), TrainingPlan (NoAccount, Success, PathVerification)
  - `wellness_test.go`: 4 help tests, MenstrualCycle (NoAccount, Success, PathVerification, ServerError), MenstrualSummary (NoAccount, Success), PregnancySummary (NoAccount, Success, ServerError)
  - `reload_test.go`: 1 help test, Reload (NoAccount, Success, WithDate, InvalidDate, ServerError)

**Commands run:**
- `make build` — success
- `make fmt` — clean (gofumpt adjusted struct alignment in wellness.go)
- `make lint` — 0 issues
- `make test` — all pass (garminapi: 1.6s, cmd: 2.8s)
- `make ci` — all green
- `./bin/gc profile --help` — shows view, settings subcommands
- `./bin/gc hydration --help` — shows view, add subcommands
- `./bin/gc training --help` — shows plans, plan subcommands
- `./bin/gc wellness --help` — shows menstrual-cycle, menstrual-summary, pregnancy-summary subcommands
- `./bin/gc reload --help` — shows date arg
- `./bin/gc --help` — shows all top-level commands including new ones

**Issues:**
- None

### 2026-02-12 — Task 30: Unit test coverage and test utilities

**Changes:**
- Created `internal/testutil/` package with shared test helpers:
  - `tokens.go`: `TestTokens()`, `ExpiredTokens()`, `TokensWithoutOAuth1()` — test token fixtures with realistic fake values
  - `server.go`: `NewServer(t, handler)`, `NewClientWithServer(t, handler)`, `JSONHandler(status, body)` — mock HTTP server factory and API client creator
  - `config.go`: `TempConfigDir(t)`, `TempConfigFile(t, cfg)` — temp config directory helpers that redirect HOME for isolated testing
  - `keyring.go`: `MemKeyring` (exported), `ErrKeyring` (always-erroring), `NewTestSecretsStore(t)`, `StoreTestTokens(t, store, email)`, `NewErrSecretsStore(t)` — in-memory keyring mock and secrets store helpers
- Improved `internal/config` coverage from 72.0% to **86.0%**:
  - Added `TestRead_ReturnsFileOrZeroValue` — exercises the real `Read()` path
  - Added `TestWrite_AndRead_RoundTrip` — round-trip through `Write()`/`Read()` with temp HOME
  - Added `TestReadFrom_PermissionError` — tests unreadable file error path
  - Added `TestWriteTo_MarshalError` — tests deep nested directory creation
- Improved `internal/outfmt` coverage from 69.6% to **80.4%**:
  - Added `captureStdout(t, fn)` helper using `os.Pipe()` to capture stdout output
  - Added `TestWrite_JSON`, `TestWrite_Plain`, `TestWrite_Table` — tests the `Write()` dispatch function in all three modes
  - Added `TestWriteTable_SingleColumn`, `TestWritePlain_SingleColumn`, `TestWriteTable_EmptyRows` — edge cases
- Improved `internal/secrets` coverage from 50.0% to **85.0%**:
  - Added `openKeyringFn` function variable for dependency injection of `keyring.Open`
  - Added `errKeyring` mock that returns errors for all operations
  - Added `TestGetError`, `TestSetError`, `TestDeleteError` — generic keyring error path tests
  - Added `TestOpen_Success`, `TestOpen_WithBackend`, `TestOpen_Error`, `TestOpen_ConfigFields` — tests for the `Open()` function using injected keyring opener

**Coverage summary (all packages >80%):**
| Package | Before | After |
|---------|--------|-------|
| cmd | 85.6% | 85.6% |
| config | 72.0% | 86.0% |
| errfmt | 97.6% | 97.6% |
| fit | 100.0% | 100.0% |
| garminapi | 92.0% | 92.0% |
| garminauth | 83.4% | 83.4% |
| outfmt | 69.6% | 80.4% |
| secrets | 50.0% | 85.0% |
| ui | 100.0% | 100.0% |

**Commands run:**
- `make fmt` — clean (gofumpt adjusted `ErrKeyring` method alignment)
- `make lint` — 0 issues
- `make test` — all pass
- `make ci` — all green

**Issues:**
- None

### 2026-02-12 — Task 31: E2E test infrastructure

**Changes:**
- Added `github.com/joho/godotenv` v1.5.1 dependency for `.env` file loading
- Created `internal/e2e/` directory for end-to-end tests with `//go:build e2e` build tag
- Created `internal/e2e/testenv.go` with `LoadEnv(t)` helper:
  - Loads `.env` file from project root using `godotenv.Load()` (best-effort, ignores missing file)
  - Reads `GARMIN_EMAIL` and `GARMIN_PASSWORD` from environment
  - Calls `t.Skip()` if either variable is missing, so tests degrade gracefully in CI
  - `findEnvFile()` walks up from source file directory to locate `.env` in project root
- Created `internal/e2e/client.go` with `AuthenticatedClient(t)` helper:
  - Returns a `*garminapi.Client` authenticated against real Garmin Connect API
  - Uses headless SSO login via `garminauth.LoginHeadless()` with credentials from `.env`
  - Caches tokens in package-level `cachedTokens` variable protected by `sync.Mutex`
  - Multiple tests reuse the same login session within a test binary run
  - Checks `IsExpired()` before reusing cached tokens
- Created `internal/e2e/cleanup.go` with `RegisterCleanup(t, fn)` helper:
  - Thin wrapper around `t.Cleanup()` for consistent naming in E2E tests
  - Guarantees cleanup runs even on test failure or panic
- Created `internal/e2e/e2e_test.go` with `TestLoadEnv` smoke test:
  - Verifies the `.env` loading and skip logic works end-to-end
  - Skipped when credentials are not available
- Verified existing `test-e2e` Makefile target (`go test -tags=e2e -v -count=1 ./internal/e2e/...`) works correctly
- Verified existing `.env.example` file documents `GARMIN_EMAIL` and `GARMIN_PASSWORD`

**Commands run:**
- `go get github.com/joho/godotenv@latest` — added dependency (v1.5.1)
- `go mod tidy`
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass
- `make test-e2e` — passes (with .env present) / skips gracefully (without .env)
- `make build` — success
- `make ci` — all green

**Issues:**
- None

### 2026-02-12 — Task 32: E2E tests: authentication flow

**Changes:**
- Created `internal/e2e/auth_test.go` with `//go:build e2e` tag containing 3 tests:
  - `TestHeadlessLogin` — performs a full headless SSO login using credentials from `.env`, verifies all token fields are non-empty: OAuth2AccessToken, OAuth2RefreshToken, OAuth1Token, OAuth1Secret
  - `TestTokenFields` — verifies token metadata fields after login: Email matches `GARMIN_EMAIL`, Domain is `garmin.com` or `garmin.cn`, OAuth2ExpiresAt is non-zero and in the future, `IsExpired()` returns false for a fresh token
  - `TestAuthStatus` — creates an authenticated client via `AuthenticatedClient(t)`, makes a real API call to `/userprofile-service/usersocialprofile` to verify tokens are valid and the client works end-to-end
- Tests use the cached token mechanism from `client.go` — only one SSO login is performed per test binary run
- All tests call `LoadEnv(t)` or `AuthenticatedClient(t)` which gracefully skip via `t.Skip()` when `.env` credentials are not available

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass
- `make build` — success
- `go test -tags=e2e -run=^$ -count=0 ./internal/e2e/...` — compiles successfully

**Issues:**
- None

### 2026-02-12 — Task 33: E2E tests: activities (read-only operations)

**Changes:**
- Created `internal/e2e/activities_read_test.go` with `//go:build e2e` tag containing 4 tests:
  - `TestListActivities` — calls `GetActivities(ctx, 0, 5, "")`, verifies the response unmarshals as a valid JSON array, logs the count
  - `TestCountActivities` — calls `CountActivities(ctx)`, verifies count >= 0
  - `TestActivityDetails` — fetches the first activity via `GetActivities`, extracts `activityId`, calls `GetActivity(ctx, id)`, verifies the detail response contains `activityId` field; skips if no activities exist
  - `TestActivitySubResources` — fetches the first activity ID, then runs sub-tests for `GetActivitySplits`, `GetActivityWeather`, and `GetActivityHRZones`, verifying each returns a non-empty response; skips if no activities exist
- Added `formatID()` helper function to convert JSON number (float64) values to string activity IDs
- All tests use `AuthenticatedClient(t)` for cached token reuse and graceful skip when credentials are unavailable

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass
- `go test -tags=e2e -run=^$ -count=0 ./internal/e2e/...` — compiles successfully

**Issues:**
- None

### 2026-02-12 — Task 34: E2E tests: activity create/upload and cleanup

**Changes:**
- Created `internal/e2e/activities_write_test.go` with `//go:build e2e` tag containing:
  - `TestCreateManualActivity` — full lifecycle test:
    1. Runs `cleanupOrphanActivities` safety-net sweep at test start
    2. Creates a manual activity via `CreateManualActivity` with `E2E_TEST_` name prefix, type `running`, 1km, 600s
    3. Extracts the returned `activityId` and verifies it's non-zero
    4. Registers `t.Cleanup()` to delete the activity via `DeleteActivity` — guaranteed to run even on failure/panic
    5. Verifies the created activity appears in `GetActivities` listing via `findActivityByID` helper
    6. Renames the activity to `E2E_TEST_RENAMED` via `RenameActivity`, then verifies the name changed by fetching the activity detail
    7. Cleanup handler deletes the activity when the test finishes
  - `findActivityByID(t, client, id)` helper — searches recent 50 activities for a matching activityId
  - `cleanupOrphanActivities(t, client)` — safety-net that lists all activities, finds any with `E2E_TEST_` name prefix, and deletes them (leftover from prior failed runs)
- All test-created activities use `E2E_TEST_` prefix for easy identification
- Reuses `formatID()` helper from `activities_read_test.go` for JSON number conversion

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass
- `go test -tags=e2e -run=^$ -count=0 ./internal/e2e/...` — compiles successfully

**Issues:**
- None

### 2026-02-12 — Task 35: E2E tests: workouts create/upload and cleanup

**Changes:**
- Created `internal/e2e/workouts_write_test.go` with `//go:build e2e` tag containing:
  - `TestUploadWorkout` — full lifecycle test:
    1. Runs `cleanupOrphanWorkouts` safety-net sweep at test start
    2. Uploads a minimal workout JSON with `E2E_TEST_WORKOUT_` name prefix (running type, 1km interval step)
    3. Extracts the returned `workoutId` and verifies it's non-zero
    4. Registers `t.Cleanup()` to delete the workout via `DeleteWorkout` — guaranteed to run even on failure/panic
    5. Verifies the created workout appears in `GetWorkouts` listing via `findWorkoutByID` helper
    6. Downloads the workout as FIT via `DownloadWorkout`, verifies non-empty bytes returned
    7. Cleanup handler deletes the workout when the test finishes
  - `findWorkoutByID(t, client, id)` helper — searches recent 50 workouts for a matching workoutId, handles both array and wrapper-object response formats
  - `cleanupOrphanWorkouts(t, client)` — safety-net that lists all workouts, finds any with `E2E_TEST_` name prefix, and deletes them (leftover from prior failed runs)
- All test-created workouts use `E2E_TEST_` prefix for easy identification

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass
- `go test -tags=e2e -run=^$ -count=0 ./internal/e2e/...` — compiles successfully

**Issues:**
- None

### 2026-02-12 — Task 36: E2E tests: health, body, devices, profile (read-only)

**Changes:**
- Created `internal/e2e/health_test.go` with `//go:build e2e` tag containing 4 tests:
  - `TestHealthSummary` — calls `GetDailySummary` for today using `displayName` from tokens, verifies valid JSON response
  - `TestHealthSteps` — calls `GetSteps` for today, verifies valid JSON response
  - `TestHealthHeartRate` — calls `GetHeartRate` for today, verifies valid JSON response
  - `TestHealthSleep` — calls `GetSleep` for today, verifies valid JSON response
- Created `internal/e2e/devices_test.go` with `//go:build e2e` tag containing 1 test:
  - `TestGetDevices` — calls `GetDevices`, verifies the response unmarshals as a valid JSON array
- Created `internal/e2e/profile_test.go` with `//go:build e2e` tag containing 1 test:
  - `TestGetProfile` — calls `GetProfile`, verifies the response contains `displayName` field
- All tests are read-only operations — no cleanup required
- All tests use `AuthenticatedClient(t)` for cached token reuse and graceful skip when credentials are unavailable

**Commands run:**
- `make fmt` — clean
- `make lint` — 0 issues
- `make test` — all pass
- `make build` — success
- `go test -tags=e2e -run=^$ -count=0 ./internal/e2e/...` — compiles successfully

**Issues:**
- None
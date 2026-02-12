# gccli - Garmin Connect CLI Tool

## Overview

`gc` is a Go CLI tool that wraps the Garmin Connect API, modeled after the architecture of [gogcli](https://github.com/steipete/gogcli). The goal is to cover all 113 API methods from the [python-garminconnect](https://github.com/cyberjunky/python-garminconnect) library. Phase 1 focuses on authentication and activities -- the most common use case.

**Binary**: `gc`
**Module**: `github.com/bpauli/gccli`
**Auth**: Browser SSO + headless email/password with MFA
**Reference**: gogcli for architecture, python-garminconnect for API coverage

## Target Audience

Developers, power users, and automation enthusiasts who use Garmin devices and want programmatic access to their Garmin Connect data via the command line -- for scripting, data export, integration with other tools, or personal analytics.

## Core Features

1. **Authentication** -- Browser SSO and headless login with MFA support, OAuth1-to-OAuth2 token exchange, OS keyring storage
2. **Activities** -- List, search, view details, download (FIT/TCX/GPX/KML/CSV), upload, create manual entries, rename/retype/delete
3. **Health Data** -- Daily summaries, steps, heart rate, sleep, stress, HRV, SpO2, body battery, training readiness/status, VO2max, race predictions
4. **Body Composition** -- Weight, body fat, muscle mass, blood pressure, FIT file encoding for uploads
5. **Devices & Gear** -- Device management, gear tracking, link/unlink to activities
6. **Goals & Badges** -- Active goals, earned/available badges, challenges, personal records
7. **Workouts & Training** -- Workout management, training plans
8. **Profile** -- User profile, settings, hydration tracking

## Tech Stack

- **Language**: Go 1.23+
- **CLI Framework**: github.com/alecthomas/kong (struct-tag based)
- **Terminal UI**: github.com/muesli/termenv (cross-platform colors)
- **Secrets**: github.com/99designs/keyring (macOS Keychain, Linux SecretService, file fallback)
- **Auth**: golang.org/x/oauth2, golang.org/x/net (HTML parsing for SSO)
- **Build**: Make, goreleaser
- **Lint**: golangci-lint, gofumpt, goimports
- **Testing**: stdlib testing, httptest for mocking

## Architecture

Layered CLI architecture following gogcli patterns:

```
cmd/gc/main.go                  -- Entry point (minimal)
internal/cmd/                   -- CLI commands (Kong)
internal/config/                -- Config management
internal/garminauth/            -- Garmin SSO authentication
internal/garminapi/             -- API client layer
internal/secrets/               -- OS keyring token storage
internal/fit/                   -- FIT file encoding
internal/errfmt/                -- User-facing error formatting
internal/outfmt/                -- Output mode management (JSON/plain/table)
internal/ui/                    -- Terminal UI (colors, prompts)
```

## Data Model

- **Tokens**: OAuth1 + OAuth2 tokens stored in OS keyring, keyed by email
- **Config**: JSON file at `~/.config/gccli/config.json` with keyring backend, domain, default format
- **API Data**: Passed through as `json.RawMessage` from Garmin API; formatted at the command layer

## UI/UX Requirements

- `--json` flag for machine-readable JSON output
- `--plain` flag for TSV/plain text (no alignment)
- Default: tabwriter-aligned tables
- `--color auto|always|never` for terminal colors
- `--account` flag for multi-account support
- Colorized help output via custom Kong help printer
- Destructive actions require confirmation (overridable with `--force`)

## Security Considerations

- OAuth tokens stored in OS keyring (not plaintext files)
- Headless login prompts for password via terminal (no echo)
- MFA support for accounts with two-factor enabled
- File backend fallback when no D-Bus on Linux (with warning)
- No credentials stored in config files

## Third-Party Integrations

- **Garmin Connect API** (connect.garmin.com / connect.garmin.cn)
- **Garmin SSO** (sso.garmin.com / sso.garmin.cn) for authentication

## Constraints & Assumptions

- Go 1.23+ required
- CGO_ENABLED=1 on macOS for Keychain access
- Garmin SSO is non-standard (not OAuth2 authorization code flow)
- Rate limiting handled via retry transport with backoff
- API responses treated as opaque JSON (no Go struct definitions for all 113 endpoints)

## E2E Testing

End-to-end tests run against the real Garmin Connect API using credentials from a `.env` file:

```
GARMIN_EMAIL=your-test-account@example.com
GARMIN_PASSWORD=your-test-password
```

- **Build tag**: `//go:build e2e` -- only runs with `make test-e2e` or `go test -tags=e2e`
- **Cleanup contract**: Any activity, workout, or other mutable resource created during E2E tests MUST be deleted via `t.Cleanup()`, ensuring cleanup runs even on test failure or panic
- **Naming convention**: All test-created resources use an `E2E_TEST_` name prefix for identification
- **Orphan cleanup**: `TestMain` includes a safety-net sweep that deletes any `E2E_TEST_`-prefixed resources left over from prior failed runs
- **Graceful skip**: Tests call `t.Skip()` if `.env` is missing or credentials are not set, so `make test-e2e` does not fail in CI without credentials
- **Read-only tests**: Health, devices, profile tests are read-only and require no cleanup

## Success Criteria

- All Phase 1 commands (auth + activities) fully functional
- `make ci` passes (fmt-check + lint + test)
- Binary builds for macOS and Linux via goreleaser
- All 113 API methods covered across all phases

---

## Task List

```json
[
  {
    "id": 1,
    "category": "setup",
    "description": "Project scaffolding: go.mod, cmd/gc/main.go, Makefile, .goreleaser.yaml, .golangci.yml",
    "steps": [
      "Create go.mod with module github.com/bpauli/gccli, Go 1.23+",
      "Create cmd/gc/main.go with minimal entry point calling cmd.Execute(os.Args[1:])",
      "Create Makefile with targets: build (go build -ldflags with version/commit/date), test (go test ./...), lint (golangci-lint), fmt (goimports + gofumpt), fmt-check, tools (install dev tools to .tools/), ci (fmt-check + lint + test). Binary output to ./bin/gc",
      "Create .goreleaser.yaml for multi-platform builds (darwin/linux, amd64/arm64, CGO_ENABLED=1 for macOS)",
      "Create .golangci.yml with linters: gofumpt, govet, errcheck, staticcheck, unused, gosimple",
      "Add go.sum by running go mod tidy",
      "Verify: make build compiles (will need stub cmd package), make fmt works, make lint works"
    ],
    "passes": true
  },
  {
    "id": 2,
    "category": "setup",
    "description": "Config package: internal/config/paths.go and config.go",
    "steps": [
      "Create internal/config/paths.go with ConfigDir() returning ~/.config/gccli/ via os.UserConfigDir(), ConfigFilePath(), CredentialsDir()",
      "Create internal/config/config.go with File struct (KeyringBackend, Domain, DefaultFormat), Read() and Write() functions",
      "Support env vars: GC_DOMAIN, GC_COLOR, GC_JSON, GC_PLAIN, GC_KEYRING_BACKEND",
      "Write unit tests in internal/config/config_test.go using temp directories",
      "Verify: go test ./internal/config/... passes"
    ],
    "passes": true
  },
  {
    "id": 3,
    "category": "setup",
    "description": "UI package: internal/ui/ui.go with termenv colors",
    "steps": [
      "Create internal/ui/ui.go with UI struct wrapping termenv",
      "Implement Successf(), Error(), Warnf(), Infof() methods with color support",
      "Add --color auto|always|never support via New(colorMode string)",
      "Implement context-based access: NewContext(ctx, ui), FromContext(ctx)",
      "Write unit tests for color mode switching and output formatting",
      "Verify: go test ./internal/ui/... passes"
    ],
    "passes": true
  },
  {
    "id": 4,
    "category": "setup",
    "description": "Output formatting package: internal/outfmt/outfmt.go",
    "steps": [
      "Create internal/outfmt/outfmt.go with Mode type (JSON, Plain, Table)",
      "Implement WriteJSON() with pretty-print, no HTML escaping",
      "Implement context helpers: NewContext(ctx, mode), IsJSON(ctx), IsPlain(ctx), ModeFromContext(ctx)",
      "Add table output helper using tabwriter",
      "Write unit tests for each output mode",
      "Verify: go test ./internal/outfmt/... passes"
    ],
    "passes": true
  },
  {
    "id": 5,
    "category": "setup",
    "description": "Error formatting package: internal/errfmt/errfmt.go",
    "steps": [
      "Create internal/errfmt/errfmt.go with typed error types: AuthRequiredError, RateLimitError, GarminAPIError, TokenExpiredError",
      "Implement Format(err error) string that maps errors to actionable user messages",
      "AuthRequiredError -> 'No auth found. Run: gc auth login <email>'",
      "TokenExpiredError -> 'Token expired. Run: gc auth login <email>'",
      "RateLimitError -> 'Rate limited. Wait and retry.'",
      "GarminAPIError -> HTTP status + message",
      "Write unit tests verifying all error format strings",
      "Verify: go test ./internal/errfmt/... passes"
    ],
    "passes": true
  },
  {
    "id": 6,
    "category": "setup",
    "description": "Secrets/keyring store: internal/secrets/store.go",
    "steps": [
      "Add github.com/99designs/keyring dependency",
      "Create internal/secrets/store.go with Store struct wrapping keyring",
      "Implement OpenDefault(backend string) that selects keyring backend (auto, keychain, file)",
      "Implement Get(email string), Set(email string, data []byte), Delete(email string) methods",
      "Key format: gc:token:{email}",
      "Handle Linux D-Bus timeout with file backend fallback (adopt gogcli pattern)",
      "Write unit tests using file backend for portability",
      "Verify: go test ./internal/secrets/... passes"
    ],
    "passes": true
  },
  {
    "id": 7,
    "category": "setup",
    "description": "CLI root: internal/cmd/root.go, root_flags.go, exit.go, help_printer.go, output_helpers.go",
    "steps": [
      "Add github.com/alecthomas/kong dependency",
      "Create internal/cmd/root_flags.go with RootFlags struct (JSON, Plain, Color, Account)",
      "Create internal/cmd/exit.go with ExitError type and exit code mapping",
      "Create internal/cmd/output_helpers.go with tableWriter() and printNextPageHint() helpers",
      "Create internal/cmd/help_printer.go with custom colorized help output for Kong",
      "Create internal/cmd/root.go with CLI struct embedding RootFlags and placeholder command groups (Auth only for now), Execute(args []string) function with Kong setup, context-based DI (output mode, UI, account), centralized error formatting",
      "Update cmd/gc/main.go to call cmd.Execute(os.Args[1:])",
      "Verify: make build succeeds, ./bin/gc --help shows usage, ./bin/gc --version shows version"
    ],
    "passes": true
  },
  {
    "id": 8,
    "category": "feature",
    "description": "Garmin auth endpoints, token types, and SSO constants",
    "steps": [
      "Create internal/garminauth/endpoints.go with SSO base URLs (garmin.com + garmin.cn), signin paths, OAuth exchange paths, connect API base URLs",
      "Create internal/garminauth/tokens.go with Tokens struct (OAuth1Token, OAuth1TokenSecret, OAuth2AccessToken, OAuth2RefreshToken, OAuth2ExpiresAt, Domain, DisplayName, Email)",
      "Add IsExpired() method, JSON serialization",
      "Add LoginOptions struct for headless login config (domain, MFA code, prompt function)",
      "Write unit tests for token expiry checks and serialization",
      "Verify: go test ./internal/garminauth/... passes"
    ],
    "passes": true
  },
  {
    "id": 9,
    "category": "feature",
    "description": "Headless SSO login flow: internal/garminauth/sso_headless.go",
    "steps": [
      "Create internal/garminauth/sso_headless.go with LoginHeadless(ctx, email, password string, opts LoginOptions) (*Tokens, error)",
      "Step 1: GET SSO signin page, parse HTML for _csrf token, collect cookies using net/http cookie jar",
      "Step 2: POST credentials to SSO signin with username, password, _csrf, follow redirect, extract service ticket from URL",
      "Step 3: POST to OAuth1 exchange endpoint (oauth-service/oauth/preauthorized) to get OAuth1 consumer token",
      "Step 4: POST to OAuth2 exchange endpoint (oauth-service/oauth/exchange/user/2.0) to get OAuth2 access+refresh tokens",
      "Use golang.org/x/net/html for HTML parsing",
      "Write unit tests with httptest mock SSO server for each step",
      "Verify: go test ./internal/garminauth/... passes"
    ],
    "passes": true
  },
  {
    "id": 10,
    "category": "feature",
    "description": "MFA handling: internal/garminauth/mfa.go",
    "steps": [
      "Create internal/garminauth/mfa.go with MFA detection and submission",
      "Detect MFA challenge from SSO response HTML (check for MFA form)",
      "Implement PromptMFA() for interactive terminal MFA code input",
      "Submit MFA code to SSO and continue login flow",
      "Support --mfa-code flag for scripted/non-interactive use",
      "Integrate MFA handling into LoginHeadless flow",
      "Write unit tests with mock MFA challenge HTML responses",
      "Verify: go test ./internal/garminauth/... passes"
    ],
    "passes": true
  },
  {
    "id": 11,
    "category": "feature",
    "description": "Browser SSO flow: internal/garminauth/sso.go",
    "steps": [
      "Create internal/garminauth/sso.go with LoginBrowser(ctx, email string, opts LoginOptions) (*Tokens, error)",
      "Start local HTTP server on 127.0.0.1:0 (random port)",
      "Generate SSO login URL with redirect to local server callback",
      "Open browser to SSO login page using os/exec (xdg-open on Linux, open on macOS)",
      "Capture callback with service ticket, exchange for tokens (reuse exchange logic from headless flow)",
      "Timeout after configurable duration if no callback received",
      "Write unit tests for local server and callback handling",
      "Verify: go test ./internal/garminauth/... passes"
    ],
    "passes": true
  },
  {
    "id": 12,
    "category": "feature",
    "description": "Auth commands: internal/cmd/auth.go, auth_login.go",
    "steps": [
      "Create internal/cmd/auth.go with AuthCmd struct containing Login, Status, Remove, Token subcommands",
      "Create internal/cmd/auth_login.go with login command: gc auth login <email> with --headless and --mfa-code flags",
      "Implement auth status command showing current auth state (email, domain, token expiry)",
      "Implement auth remove command to delete stored tokens from keyring",
      "Implement auth token command to print current access token (for debugging)",
      "Prompt for password securely (no echo) during headless login",
      "Wire auth commands into CLI struct in root.go",
      "Write unit tests for command parsing and output formatting",
      "Verify: make build succeeds, ./bin/gc auth --help shows subcommands"
    ],
    "passes": true
  },
  {
    "id": 13,
    "category": "feature",
    "description": "API client base: internal/garminapi/client.go and errors.go",
    "steps": [
      "Create internal/garminapi/errors.go with typed errors: AuthRequiredError, RateLimitError, GarminAPIError, TokenExpiredError, InvalidFileFormatError",
      "Create internal/garminapi/client.go with Client struct (httpClient, baseURL, tokens)",
      "Implement NewClient(tokens, opts) constructor",
      "Implement connectapi(ctx, method, path, body) (json.RawMessage, error) for JSON API calls with Bearer token auth",
      "Implement download(ctx, path) ([]byte, error) for binary downloads",
      "Add automatic token refresh on 401 response",
      "Add function variables for dependency injection in tests (openSecretsStore, loginGarmin pattern)",
      "Write unit tests with httptest mock server",
      "Verify: go test ./internal/garminapi/... passes"
    ],
    "passes": true
  },
  {
    "id": 14,
    "category": "feature",
    "description": "Retry transport and circuit breaker: internal/garminapi/transport.go, circuitbreaker.go",
    "steps": [
      "Create internal/garminapi/transport.go with RetryTransport wrapping http.RoundTripper",
      "Retry on 429 with Retry-After header respect, max 3 retries",
      "Retry on 5xx with exponential backoff + jitter, max 2 retries",
      "Buffer request body for retries",
      "Context-aware sleep (cancellable)",
      "Create internal/garminapi/circuitbreaker.go with CircuitBreaker (threshold: 5 consecutive failures, reset: 30s, half-open via timeout)",
      "Write unit tests for retry logic, backoff, and circuit breaker states",
      "Verify: go test ./internal/garminapi/... passes"
    ],
    "passes": true
  },
  {
    "id": 15,
    "category": "feature",
    "description": "Activities API methods: internal/garminapi/activities.go",
    "steps": [
      "Create internal/garminapi/activities.go with all activity API methods",
      "CountActivities(ctx) (int, error)",
      "GetActivities(ctx, start, limit int, activityType string) (json.RawMessage, error)",
      "GetActivity(ctx, activityID string) (json.RawMessage, error)",
      "GetActivityDetails(ctx, activityID string, maxChart, maxPoly int) (json.RawMessage, error)",
      "GetActivitySplits(ctx, activityID string) and GetActivityTypedSplits, GetActivitySplitSummaries",
      "GetActivityWeather(ctx, activityID string), GetActivityHRZones, GetActivityPowerZones",
      "GetActivityExerciseSets(ctx, activityID string), GetActivityGear",
      "SearchActivities(ctx, start, limit int, startDate, endDate string) (json.RawMessage, error)",
      "DownloadActivity(ctx, activityID string, format ActivityDownloadFormat) ([]byte, error) with format enum (FIT/TCX/GPX/KML/CSV)",
      "UploadActivity(ctx, filepath string) (json.RawMessage, error)",
      "CreateManualActivity(ctx, name, activityType string, distance float64, duration time.Duration) (json.RawMessage, error)",
      "RenameActivity, RetypeActivity, DeleteActivity methods",
      "Write unit tests with mock HTTP responses for key methods",
      "Verify: go test ./internal/garminapi/... passes"
    ],
    "passes": true
  },
  {
    "id": 16,
    "category": "feature",
    "description": "Activities list/count/search commands: internal/cmd/activities.go",
    "steps": [
      "Create internal/cmd/activities.go with ActivitiesCmd struct",
      "Implement gc activities (list recent, default 20) with --limit, --start, --type flags",
      "Implement gc activities count to show total activity count",
      "Implement gc activities search with --start-date and --end-date flags",
      "Table output: ID, DATE, TYPE, NAME, DISTANCE, DURATION, CALORIES columns",
      "JSON output: activities array with count and total",
      "Plain output: TSV format",
      "Wire into CLI struct in root.go",
      "Create helper to resolve authenticated client from context (load tokens from keyring, create API client)",
      "Write unit tests for command parsing and output formatting",
      "Verify: make build succeeds, ./bin/gc activities --help shows usage"
    ],
    "passes": true
  },
  {
    "id": 17,
    "category": "feature",
    "description": "Activity detail commands: internal/cmd/activity.go",
    "steps": [
      "Create internal/cmd/activity.go with ActivityCmd struct",
      "Implement gc activity <id> showing activity summary (name, type, date, distance, duration, pace, HR, calories)",
      "Implement gc activity <id> details for full activity data",
      "Implement gc activity <id> splits, typed-splits, split-summaries",
      "Implement gc activity <id> weather for weather conditions",
      "Implement gc activity <id> hr-zones and power-zones",
      "Implement gc activity <id> exercise-sets for strength training",
      "Implement gc activity <id> gear for linked gear",
      "Support --json and --plain output modes for all subcommands",
      "Wire into CLI struct in root.go",
      "Write unit tests for command structure and output",
      "Verify: make build succeeds, ./bin/gc activity --help shows subcommands"
    ],
    "passes": true
  },
  {
    "id": 18,
    "category": "feature",
    "description": "Activity download command: internal/cmd/activity_download.go",
    "steps": [
      "Create internal/cmd/activity_download.go",
      "Implement gc activity <id> download with --format flag (fit, gpx, tcx, kml, csv; default fit)",
      "Implement --output flag for custom output filename",
      "Default filename: activity_{id}.{format}",
      "Handle FIT download (comes as zip -- extract .fit file from zip)",
      "Print download path on success",
      "Write unit tests for filename generation and format validation",
      "Verify: make build succeeds, ./bin/gc activity download --help shows usage"
    ],
    "passes": true
  },
  {
    "id": 19,
    "category": "feature",
    "description": "Activity upload, create, and modify commands: internal/cmd/activity_upload.go, activity_modify.go, confirm.go",
    "steps": [
      "Create internal/cmd/confirm.go with confirmation prompt helper for destructive actions (with --force bypass)",
      "Create internal/cmd/activity_upload.go with gc activity upload <file> (FIT/GPX/TCX)",
      "Implement gc activity create --name --type --distance --duration for manual activities",
      "Create internal/cmd/activity_modify.go with rename, retype, delete subcommands",
      "gc activity <id> rename 'New Name'",
      "gc activity <id> retype --type-id 1 --type-key running",
      "gc activity <id> delete with confirmation prompt (skippable with --force)",
      "Write unit tests for all commands",
      "Verify: make build succeeds, ./bin/gc activity --help shows all subcommands"
    ],
    "passes": true
  },
  {
    "id": 20,
    "category": "feature",
    "description": "Workouts API and commands: internal/garminapi/workouts.go, internal/cmd/workouts.go",
    "steps": [
      "Create internal/garminapi/workouts.go with workout API methods: GetWorkouts, GetWorkout, DownloadWorkout, UploadWorkout, GetScheduledWorkout",
      "Create internal/cmd/workouts.go with WorkoutsCmd struct",
      "Implement gc workouts (list), gc workouts <id> (details), gc workouts <id> download (FIT)",
      "Implement gc workouts upload <file.json> for workout JSON upload",
      "Implement gc workouts schedule <id> for scheduled workout view",
      "Wire WorkoutsCmd into CLI struct in root.go",
      "Write unit tests for API methods and commands",
      "Verify: make build succeeds, ./bin/gc workouts --help shows subcommands"
    ],
    "passes": true
  },
  {
    "id": 21,
    "category": "feature",
    "description": "Health API methods: internal/garminapi/health.go",
    "steps": [
      "Create internal/garminapi/health.go with all health API methods",
      "GetDailySummary, GetSteps, GetDailySteps, GetWeeklySteps",
      "GetHeartRate, GetRestingHeartRate, GetFloors",
      "GetSleep, GetStress, GetWeeklyStress",
      "GetRespiration, GetSPO2, GetHRV",
      "GetBodyBattery, GetBodyBatteryRange",
      "GetIntensityMinutes, GetWeeklyIntensityMinutes",
      "GetTrainingReadiness, GetTrainingStatus",
      "GetFitnessAge, GetMaxMetrics, GetLactateThreshold, GetCyclingFTP",
      "GetRacePredictions, GetEnduranceScore, GetHillScore",
      "GetAllDayEvents, GetLifestyleLogging",
      "Write unit tests for key methods with mock HTTP",
      "Verify: go test ./internal/garminapi/... passes"
    ],
    "passes": true
  },
  {
    "id": 22,
    "category": "feature",
    "description": "Health basic commands: internal/cmd/health.go",
    "steps": [
      "Create internal/cmd/health.go with HealthCmd struct",
      "Implement gc health summary [date] for daily summary",
      "Implement gc health steps [date] with gc health steps daily --start --end and gc health steps weekly --end --weeks",
      "Implement gc health hr [date] and gc health rhr [date]",
      "Implement gc health floors [date]",
      "Implement gc health sleep [date]",
      "Implement gc health stress [date] with gc health stress weekly --end --weeks",
      "Implement gc health respiration [date]",
      "Support date positional arg defaulting to today, including relative dates (today, yesterday, 3d)",
      "Wire HealthCmd into CLI struct in root.go",
      "Write unit tests for date parsing and command structure",
      "Verify: make build succeeds, ./bin/gc health --help shows subcommands"
    ],
    "passes": true
  },
  {
    "id": 23,
    "category": "feature",
    "description": "Health advanced commands: internal/cmd/health_advanced.go",
    "steps": [
      "Create internal/cmd/health_advanced.go with advanced health subcommands",
      "Implement gc health spo2 [date], gc health hrv [date]",
      "Implement gc health body-battery [date] with --start/--end range option",
      "Implement gc health intensity-minutes [date] with weekly --start/--end option",
      "Implement gc health training-readiness [date], gc health training-status [date]",
      "Implement gc health fitness-age [date], gc health max-metrics [date]",
      "Implement gc health lactate-threshold, gc health cycling-ftp",
      "Implement gc health race-predictions [date]",
      "Implement gc health endurance-score [date], gc health hill-score [date]",
      "Implement gc health events [date], gc health lifestyle [date]",
      "Write unit tests for command structure",
      "Verify: make build succeeds, ./bin/gc health --help shows all subcommands"
    ],
    "passes": true
  },
  {
    "id": 24,
    "category": "feature",
    "description": "Body composition API and commands: internal/garminapi/body.go, internal/cmd/body.go",
    "steps": [
      "Create internal/garminapi/body.go with body composition API methods: GetBodyComposition, GetBodyCompositionRange, GetWeighIns, GetDailyWeighIns, AddWeight, AddBodyComposition, DeleteWeight, GetBloodPressure, AddBloodPressure, DeleteBloodPressure",
      "Create internal/cmd/body.go with BodyCmd struct",
      "Implement gc body composition [date] with --start/--end range",
      "Implement gc body weigh-ins --start/--end and gc body weigh-ins daily [date]",
      "Implement gc body add-weight <value> with --unit (kg default, lbs)",
      "Implement gc body add-composition --weight --body-fat --muscle-mass",
      "Implement gc body delete-weight <pk> --date with confirmation",
      "Implement gc body blood-pressure --start/--end",
      "Implement gc body add-blood-pressure --systolic --diastolic --pulse",
      "Implement gc body delete-blood-pressure <version> --date with confirmation",
      "Wire BodyCmd into CLI struct in root.go",
      "Write unit tests",
      "Verify: make build succeeds, ./bin/gc body --help shows subcommands"
    ],
    "passes": true
  },
  {
    "id": 25,
    "category": "feature",
    "description": "FIT file encoder: internal/fit/encoder.go",
    "steps": [
      "Create internal/fit/encoder.go porting Python FitEncoderWeight class to Go",
      "Implement FIT protocol binary encoding with CRC checks",
      "Support encoding body composition data (weight, body fat %, muscle mass, etc.) into FIT format",
      "Integrate with gc body add-composition command to encode and upload FIT data",
      "Write unit tests comparing output against known-good FIT file bytes",
      "Verify: go test ./internal/fit/... passes"
    ],
    "passes": true
  },
  {
    "id": 26,
    "category": "feature",
    "description": "Devices API and commands: internal/garminapi/devices.go, internal/cmd/devices.go",
    "steps": [
      "Create internal/garminapi/devices.go with device API methods: GetDevices, GetDeviceSettings, GetPrimaryTrainingDevice, GetDeviceSolar, GetDeviceAlarms, GetLastUsedDevice",
      "Create internal/cmd/devices.go with DevicesCmd struct",
      "Implement gc devices (list), gc devices <id> settings, gc devices primary",
      "Implement gc devices <id> solar, gc devices alarms, gc devices last-used",
      "Wire DevicesCmd into CLI struct in root.go",
      "Write unit tests",
      "Verify: make build succeeds, ./bin/gc devices --help shows subcommands"
    ],
    "passes": true
  },
  {
    "id": 27,
    "category": "feature",
    "description": "Gear API and commands: internal/garminapi/gear.go, internal/cmd/gear.go",
    "steps": [
      "Create internal/garminapi/gear.go with gear API methods: GetGear, GetGearStats, GetGearActivities, GetGearDefaults, LinkGear, UnlinkGear",
      "Create internal/cmd/gear.go with GearCmd struct",
      "Implement gc gear (list), gc gear <uuid> stats, gc gear <uuid> activities",
      "Implement gc gear defaults for gear defaults per activity type",
      "Implement gc gear <uuid> link <activity-id> and gc gear <uuid> unlink <activity-id>",
      "Wire GearCmd into CLI struct in root.go",
      "Write unit tests",
      "Verify: make build succeeds, ./bin/gc gear --help shows subcommands"
    ],
    "passes": true
  },
  {
    "id": 28,
    "category": "feature",
    "description": "Goals, badges, challenges, records: internal/garminapi/goals.go, internal/cmd/goals.go",
    "steps": [
      "Create internal/garminapi/goals.go with API methods: GetGoals, GetBadgesEarned, GetBadgesAvailable, GetBadgesInProgress, GetChallenges, GetBadgeChallenges, GetPersonalRecords",
      "Create internal/cmd/goals.go with GoalsCmd, BadgesCmd, ChallengesCmd, RecordsCmd",
      "Implement gc goals with --status filter (active, completed)",
      "Implement gc badges earned, gc badges available, gc badges in-progress",
      "Implement gc challenges and gc challenges badge",
      "Implement gc records for personal records",
      "Wire all into CLI struct in root.go",
      "Write unit tests",
      "Verify: make build succeeds, ./bin/gc goals --help shows subcommands"
    ],
    "passes": true
  },
  {
    "id": 29,
    "category": "feature",
    "description": "Profile, hydration, training, wellness commands",
    "steps": [
      "Create internal/garminapi/profile.go with API methods: GetProfile, GetUserSettings",
      "Create internal/cmd/profile.go with gc profile and gc profile settings",
      "Create internal/cmd/hydration.go with gc hydration [date] and gc hydration add <amount_ml>",
      "Create internal/cmd/training.go with gc training plans and gc training plan <id>",
      "Create internal/cmd/wellness.go for menstrual/pregnancy data if applicable",
      "Add gc reload [date] command for requesting data reload",
      "Wire all commands into CLI struct in root.go",
      "Write unit tests for all new commands",
      "Verify: make build succeeds, ./bin/gc --help shows all top-level commands"
    ],
    "passes": true
  },
  {
    "id": 30,
    "category": "testing",
    "description": "Unit test coverage and test utilities",
    "steps": [
      "Add testutil helpers in internal/testutil/: mock HTTP server factory, test token fixtures, temp config helpers",
      "Ensure all packages have >80% unit test coverage",
      "Run make ci to verify all tests pass, linting clean, formatting correct",
      "Verify: make ci passes with zero errors"
    ],
    "passes": true
  },
  {
    "id": 31,
    "category": "testing",
    "description": "E2E test infrastructure: .env loading, test helpers, Makefile targets",
    "steps": [
      "Add github.com/joho/godotenv dependency for .env file loading",
      "Create internal/e2e/ directory for end-to-end tests with //go:build e2e tag",
      "Create internal/e2e/testenv.go with LoadEnv() helper that reads .env file (GARMIN_EMAIL, GARMIN_PASSWORD), skips test with t.Skip() if vars are missing",
      "Create internal/e2e/client.go with helper to authenticate and return a ready-to-use *garminapi.Client (login via headless SSO using .env credentials, cache tokens in temp keyring for test session reuse)",
      "Create internal/e2e/cleanup.go with cleanup registry: RegisterCleanup(t, cleanupFn) that uses t.Cleanup() to guarantee cleanup runs even on test failure or panic",
      "Add Makefile target: test-e2e that runs go test -tags=e2e -v -count=1 ./internal/e2e/...",
      "Add .env.example to repo (already created) documenting GARMIN_EMAIL and GARMIN_PASSWORD",
      "Verify: make test-e2e runs (skips gracefully if .env is missing), make ci still passes"
    ],
    "passes": true
  },
  {
    "id": 32,
    "category": "testing",
    "description": "E2E tests: authentication flow",
    "steps": [
      "Create internal/e2e/auth_test.go with //go:build e2e tag",
      "Test headless login: authenticate with GARMIN_EMAIL/GARMIN_PASSWORD from .env, verify tokens are returned with non-empty OAuth2 access token",
      "Test token fields: verify Email, Domain, OAuth2ExpiresAt are populated correctly",
      "Test auth status: after login, verify auth status reports authenticated for the test email",
      "Verify: make test-e2e passes with valid .env credentials"
    ],
    "passes": true
  },
  {
    "id": 33,
    "category": "testing",
    "description": "E2E tests: activities (read-only operations)",
    "steps": [
      "Create internal/e2e/activities_read_test.go with //go:build e2e tag",
      "Test list activities: call GetActivities and verify response is valid JSON array",
      "Test count activities: call CountActivities and verify count >= 0",
      "Test activity details: if activities exist, fetch first activity by ID and verify response contains activityId field",
      "Test activity splits/weather/hr-zones: fetch detail subresources for an existing activity (read-only, no cleanup needed)",
      "Verify: make test-e2e passes"
    ],
    "passes": true
  },
  {
    "id": 34,
    "category": "testing",
    "description": "E2E tests: activity create/upload and cleanup",
    "steps": [
      "Create internal/e2e/activities_write_test.go with //go:build e2e tag",
      "Test create manual activity: call CreateManualActivity with test data (name prefixed 'E2E_TEST_', type running, 1km, 10min), capture returned activity ID",
      "Register t.Cleanup() to delete the created activity via DeleteActivity -- cleanup MUST run even if subsequent assertions fail",
      "Verify created activity appears in GetActivities listing",
      "Test rename: rename the created activity to 'E2E_TEST_RENAMED', verify name changed",
      "Test delete: the t.Cleanup() handler deletes the activity; verify it is removed from listing",
      "All created test activities use 'E2E_TEST_' name prefix so they are identifiable",
      "Add a safety-net cleanup in TestMain that lists all activities with 'E2E_TEST_' prefix and deletes any orphaned ones from prior failed runs",
      "Verify: make test-e2e passes, no test activities remain in the account after tests complete"
    ],
    "passes": true
  },
  {
    "id": 35,
    "category": "testing",
    "description": "E2E tests: workouts create/upload and cleanup",
    "steps": [
      "Create internal/e2e/workouts_write_test.go with //go:build e2e tag",
      "Test upload workout: upload a minimal test workout JSON (name prefixed 'E2E_TEST_'), capture returned workout ID",
      "Register t.Cleanup() to delete the uploaded workout -- cleanup MUST run even on failure",
      "Verify uploaded workout appears in GetWorkouts listing",
      "Test download workout: download the created workout as FIT, verify non-empty bytes returned",
      "Add safety-net cleanup in TestMain to delete any 'E2E_TEST_' prefixed workouts left from prior failed runs",
      "Verify: make test-e2e passes, no test workouts remain in the account after tests complete"
    ],
    "passes": true
  },
  {
    "id": 36,
    "category": "testing",
    "description": "E2E tests: health, body, devices, profile (read-only)",
    "steps": [
      "Create internal/e2e/health_test.go with //go:build e2e tag",
      "Test health summary: call GetDailySummary for today, verify valid JSON response",
      "Test health endpoints: call GetSteps, GetHeartRate, GetSleep for today, verify responses",
      "Create internal/e2e/devices_test.go: call GetDevices, verify response is valid JSON array",
      "Create internal/e2e/profile_test.go: call GetProfile, verify response contains displayName",
      "These are all read-only operations -- no cleanup required",
      "Verify: make test-e2e passes"
    ],
    "passes": true
  }
]
```

---

## Agent Instructions

1. Read `activity.md` first to understand current state
2. Find next task with `"passes": false`
3. Complete all steps for that task
4. Verify with `make build && make lint && make test`
5. Update task to `"passes": true`
6. Log completion in `activity.md`
7. Commit with conventional commit message
8. Repeat until all tasks pass

**Important:** Only modify the `passes` field. Do not remove or rewrite tasks.

---

## Completion Criteria
All tasks marked with `"passes": true`

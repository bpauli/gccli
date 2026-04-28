# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.8.0] - 2026-04-28

### Added

- **Activity types** — New `gccli activities types` command lists every activity type Garmin exposes via `/activity-service/activity/activityTypes`. Output is sorted by `typeId` and includes `TRIMMABLE`, `HIDDEN`, and `RESTRICTED` flags. (#46, #48)

### Fixed

- **DI OAuth login** — Headless authentication on Garmin's Digital Identity (DI) tenant now exchanges SSO tickets for OAuth tokens correctly, fixing login failures on DI accounts. The OAuth `client_id` is also validated more strictly to fail fast on misconfiguration. (#45, #47)

## [1.7.3] - 2026-04-21

### Changed

- Bump `golang.org/x/net`, `golang.org/x/term`, `golang.org/x/sys` dependencies (#42)

## [1.7.2] - 2026-04-07

### Added

- **AUR package update** — Release workflow now automatically triggers `gccli-bin` AUR package update alongside Homebrew (#40)

## [1.7.1] - 2026-04-07

### Fixed

- **SSO rate limit retry** — Headless login now retries on HTTP 429/5xx from Garmin SSO with exponential backoff (2s, 5s, 10s), fixing intermittent "service ticket not found" failures

## [1.7.0] - 2026-04-07

### Fixed

- **Token auto-persist** — Refreshed OAuth2 tokens are now persisted back to the OS keyring, so authentication survives across CLI invocations without re-login (#35, #39)
  - Proactively refresh expired tokens before API calls to avoid wasted 401 round-trips
  - Persist tokens after 401-triggered refresh as a fallback for clock skew or early revocation

### Changed

- Bump `github.com/alecthomas/kong` dependency (#38)

## [1.6.0] - 2026-03-31

### Added

- **`nutrition` command group** — View daily food logs, meal summaries, and nutrition settings by date (#36)
  - `gccli nutrition [date]` — View daily food log
  - `gccli nutrition meals [date]` — View meal summaries
  - `gccli nutrition settings [date]` — View nutrition settings

### Tests

- Added missing nutrition command test coverage: NoAccount, InvalidDate, and ServerError paths (#37)

## [1.5.0] - 2026-03-19

### Added

- **Per-set duration and rest time** — exercise sets now support individual `:dSECS` and `:rSECS` suffixes for per-set duration and rest control (#34)

### Changed

- **Breaking:** removed global `--rest` flag from `activity exercise-sets set` — use per-set `:rSECS` suffix instead

### Docs

- Updated README, skill reference, and docs site with new exercise set format examples

## [1.4.0] - 2026-03-19

### Added

- **`activity exercise-sets set` command** — Log exercise sets (reps, weight, rest) for strength training activities (#32)
- **`exercises list` command** — Browse Garmin's exercise categories and exercises for strength training
- **`activity create --date` flag** — Set the activity start time when creating manual entries
- **`auth export` command** — Export credentials as a portable base64 string for transfer to another machine (#30)
- **`auth import` command** — Import credentials from another machine (accepts argument or stdin) (#30)

### Fixed

- **Auth rate limiting** — Login requests now automatically retry on HTTP 429 and 5xx responses with exponential backoff, fixing headless login failures on rate-limited environments (#30)

## [1.3.1] - 2026-03-11

### Changed

- Restructured `skill/` into `skills/` directory with subdirectories (`skills/gccli/`, `skills/garmin-trainer/`)
- Added `garmin-trainer` skill for AI-assisted training plan generation

## [1.3.0] - 2026-03-11

### Changed

- **`events add` — Typed CLI flags replace JSON payload** (#27):
  - `--name`, `--date`, `--type` (required), plus `--race`, `--location`, `--distance` (e.g. `10km`, `26.2mi`), `--goal` (e.g. `50m`, `1h30m`), `--time`, `--timezone`, `--private`, `--note`, `--url`
  - `--primary` and `--training` flags are mutually exclusive for training priority
  - Distance strings (`10km`, `26.2mi`, `400m`) and goal durations (`50m`, `1h30m`, `2400s`) are parsed and validated at input time

## [1.2.0] - 2026-03-11

### Added

- **`events add` command** — Create calendar events via raw JSON payload:
  - `gccli events add --params '{"eventName":"Race","date":"2026-09-27","eventType":"running"}'`
  - Supports all Garmin Connect event fields: time, location, distance, privacy, race flag, notes, URL
  - Set custom goals and training priority directly when creating: `eventCustomization` with `customGoal` (time-based) and `isPrimaryEvent`/`isTrainingEvent`
- **`events delete` command** — Delete a calendar event by ID with confirmation prompt (`-f` to skip)
- **Events API methods** — `AddEvent`, `DeleteEvent` in the API client
- **E2E test for events** — Full lifecycle test: create event with goal → list → delete → verify removed
- **MFA test coverage** — Tests for `ticketFromURL`, 429 rate limiting, ticket extraction from redirect URL, and signin params forwarding

### Fixed

- **Headless MFA login** — Forward signin query params to the MFA verification endpoint, matching garth behaviour. Without them Garmin SSO silently rejects the MFA submission. Also handle 429 rate limiting and extract tickets from redirect URLs. (Thanks to [@derbauer97](https://github.com/derbauer97) — #23)
- **Zsh completion** — Fix parse errors from empty case blocks and missing closing brace when sourced with `compinit`

### Docs

- Added Events section to website (feature card, command reference with goal/priority examples)
- Added events commands to skill reference (`skills/gccli/SKILL.md`)
- Added Buy Me a Coffee badge to README

## [1.1.0] - 2026-03-06

### Added

- **`courses import` command** — Import GPX files as new courses on Garmin Connect:
  - `courses import route.gpx` — Import with default settings (cycling, private)
  - `--name "My Route"` — Override the course name
  - `--type hiking` — Set activity type (running, cycling, hiking, etc.)
  - `--privacy 1` — Set course privacy (1=public, 2=private, 4=group)
- **`courses delete` command** — Delete a course by ID with confirmation prompt (`-f` to skip)
- **Courses API methods** — `ImportCourseGPX`, `GetCourseElevation`, `SaveCourse`, `DeleteCourse` in the API client
- **E2E test for courses import+delete** — Full lifecycle test: import GPX → enrich elevation → save → verify → delete

### Fixed

- **Courses import save payload** — Filter import response to only API-accepted fields, set required defaults (`coursePrivacy`, `rulePK`, `coordinateSystem`, `sourceTypeId`, `startPoint`), and handle `latitude`/`longitude` field names correctly for elevation enrichment

## [1.0.0] - 2026-03-02

### Added

- **`courses` command group** — New commands for managing Garmin Connect courses:
  - `courses list` — List all user courses with table/JSON/plain output
  - `courses favorites` — List favorite courses
  - `courses detail <id>` — View full course detail (JSON)
  - `courses send <course-id> <device-id>` — Send a course to a device via the device message API
- **Courses API methods** — `GetCourses`, `GetCourse`, `GetCourseFavorites`, `SendCourseToDevice` in the API client
- **E2E tests for courses** — End-to-end tests against the real Garmin Connect API

### Changed

- Updated Go module dependencies (`golang.org/x/net`)
- Updated CI dependencies (`actions/checkout` 4→6, `actions/setup-go` 5→6, `goreleaser/goreleaser-action` 6→7)

### Docs

- Added Courses section to website (hero badge, feature card, command reference)
- Added courses commands to skill reference (`skills/gccli/SKILL.md`)
- Added shell completions card to install section on website

## [0.3.0] - 2026-02-17

### Added

- **`completion` command** — Generate shell completions for bash, zsh, fish, and PowerShell (`gccli completion <shell>`)
- **E2E workflow** — GitHub Actions workflow to run end-to-end tests on push to `main`, daily at 06:00 UTC, and via manual dispatch
- **Dependabot** — Automated dependency update checks via Dependabot configuration

### Fixed

- **Garmin Connect API compatibility** — Adapted to endpoint and payload changes in the Garmin Connect API

### Changed

- Updated outdated Go module dependencies

### Docs

- Added community health files (contributing guide, code of conduct, etc.)
- Added architecture diagram to docs page
- Linked author name to GitHub profile in docs footer

## [0.2.0] - 2026-02-12

### Added

- **`workouts schedule` subcommands** — Restructured `workouts schedule` from a single command into a command group with three subcommands:
  - `workouts schedule add <id> <date>` — Schedule a workout on a calendar date (replaces `workouts schedule <id> <date>`)
  - `workouts schedule list <date>` — List scheduled workouts for a date, with table/JSON/plain output
  - `workouts schedule remove <id> [-f]` — Remove a scheduled workout from the calendar with confirmation prompt
- **Calendar API** — New `GetCalendarWeek` API method to fetch calendar data from Garmin Connect
- **Unschedule API** — New `UnscheduleWorkout` API method to delete scheduled workouts

### Changed

- `workouts schedule` is now a command group instead of a leaf command; use `workouts schedule add` for the previous scheduling behavior

## [0.1.0] - 2026-02-12

Initial release of gccli — a fast, script-friendly CLI for Garmin Connect.

### Added

- **Authentication** — Browser SSO and headless email/password login with MFA support, OAuth1-to-OAuth2 token exchange
- **Secure credential storage** — OS keyring (macOS Keychain, Linux Secret Service, encrypted file fallback), tokens keyed by `gccli:token:<email>`
- **Activities** — List, search, count, view details/splits/weather/HR zones/power zones/exercise sets/gear, download (FIT/GPX/TCX/KML/CSV), upload, create manual entries, rename, retype, delete
- **Health data** — Daily summaries, steps (daily/weekly), heart rate, resting HR, floors, sleep, stress (daily/weekly), respiration, SpO2, HRV, body battery, training readiness/status, fitness age, VO2max/max metrics, lactate threshold, cycling FTP, race predictions, endurance score, hill score, intensity minutes, wellness events
- **Body composition** — Weight tracking, body fat, muscle mass, weigh-in history, blood pressure, FIT file encoding for composition uploads
- **Workouts** — List, view details, download as FIT, upload from JSON, create with pace/HR/power/cadence targets, schedule, delete
- **Devices** — List registered devices, view settings, solar data, alarms, primary/last-used device
- **Gear** — List gear, usage stats, linked activities, defaults per activity type, link/unlink to activities
- **Goals & badges** — Active goals, earned/available/in-progress badges, challenges, personal records
- **Training plans** — Browse and view training plans
- **Profile** — User profile and settings
- **Hydration** — View and log daily water intake
- **Wellness** — Menstrual cycle data, pregnancy summary
- **Output formats** — JSON (`--json`), plain/TSV (`--plain`), and human-friendly tables (default)
- **Multiple accounts** — `--account` flag and `GCCLI_ACCOUNT` env var
- **Resilient networking** — Automatic retry on 429/5xx with exponential backoff, circuit breaker for fault tolerance, auto-refresh OAuth2 tokens on 401
- **CI pipeline** — GitHub Actions for fmt-check, lint, and test
- **Cross-platform builds** — macOS (amd64/arm64) and Linux (amd64/arm64) via goreleaser

[1.8.0]: https://github.com/bpauli/gccli/releases/tag/v1.8.0
[1.7.3]: https://github.com/bpauli/gccli/releases/tag/v1.7.3
[1.7.2]: https://github.com/bpauli/gccli/releases/tag/v1.7.2
[1.7.1]: https://github.com/bpauli/gccli/releases/tag/v1.7.1
[1.7.0]: https://github.com/bpauli/gccli/releases/tag/v1.7.0
[1.6.0]: https://github.com/bpauli/gccli/releases/tag/v1.6.0
[1.5.0]: https://github.com/bpauli/gccli/releases/tag/v1.5.0
[1.4.0]: https://github.com/bpauli/gccli/releases/tag/v1.4.0
[1.3.0]: https://github.com/bpauli/gccli/releases/tag/v1.3.0
[1.2.0]: https://github.com/bpauli/gccli/releases/tag/v1.2.0
[1.1.0]: https://github.com/bpauli/gccli/releases/tag/v1.1.0
[1.0.0]: https://github.com/bpauli/gccli/releases/tag/v1.0.0
[0.3.0]: https://github.com/bpauli/gccli/releases/tag/v0.3.0
[0.2.0]: https://github.com/bpauli/gccli/releases/tag/v0.2.0
[0.1.0]: https://github.com/bpauli/gccli/releases/tag/v0.1.0

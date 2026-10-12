# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.13.0] - 2026-10-12

### Changed

- **API requests retry and time out** - All API calls now go through a retry transport. Before, only `auth login` retried, and a request had no timeout. (#95, #109)
  - A 429 is retried up to 3 times when `Retry-After` is 30 s or less. A longer wait fails at once with the "Rate limited" hint and exit code 5. `Retry-After` can be seconds or an HTTP date.
  - A 5xx is retried up to 2 times with exponential backoff, but only for read requests (GET). A write that gets a 5xx fails at once, because the backend may already have saved it. For example, `hydration add` is a PUT that adds to the day's total, so a retry would log the water twice.
  - Each request waits at most 60 s for the response headers. Before, a server that accepted the connection and never answered kept the command waiting forever. The body read has no limit, so large downloads still finish.
  - `exercises list` also gets the retry and the timeout.
  - A token refresh times out after 60 s.
- `auth status` now exits with 3 when no credentials are stored for the account. In `--json` mode it prints the JSON error object on stderr. Before, it printed a warning and exited with 0, so a script saw success. A logged-in account still exits with 0, also when the token has expired. (#105, #106)

### Removed

- The unused circuit breaker in `internal/garminapi`. Nothing called it, and in a one-shot CLI its state reset on every run. (#109)

### Docs

- **Token storage** - The docs said tokens are "never stored in plaintext files". The file backend encrypts tokens with a fixed passphrase that is in the gccli source, so anyone who can read the files can read the tokens. The README now says this and recommends the macOS Keychain or the Linux Secret Service. (#98, #108)
- `--plain` prints TSV in only 11 commands, with no header row. The docs list them. Most other commands print JSON in all modes. (#98, #108)
- The colors, units, date-flag and Go version (1.26) text now matches the code. Two skill examples that the parser rejected are fixed. 25 commands that were missing from the skill and the docs site are added. (#98, #108)
- The README, the skill and the docs site describe the retry rules and the timeout, and say that `auth status` exits with 3 when no credentials are stored. (#106, #109)

## [1.12.0] - 2026-10-12

### Changed

- **Exit codes by error kind** - gccli now exits with 2 for a usage error, 3 when you must log in (no account, no stored credentials or an expired token), 4 when Garmin returns 404, 5 when Garmin returns 429, and 1 for all other errors. Before, every runtime error exited with 1 and every usage error exited with 80. A script that checks for a non-zero exit code keeps working. A script that checks for exactly 1 or 80 must change. (#96, #103)
- Usage errors, such as an unknown flag, now print the usage text to stderr, not stdout. `--help` still prints to stdout and exits with 0. (#96, #103)
- In `--json` mode, a runtime error now prints one JSON line on stderr, such as `{"error":{"code":"rate_limited","status":429,"message":"Rate limited. Wait 30 and retry."}}`. `code` is one of `error`, `usage`, `auth_required`, `not_found` or `rate_limited`, and it agrees with the exit code. `status` is the HTTP status, when there is one. Usage errors stay plain text. (#96, #104)
- Delete commands without `--force` now fail with exit code 1 when stdin is not a terminal. Before, they read EOF, printed "Cancelled" and exited with 0, so a script saw success but nothing was deleted. If you answer "n" at the prompt, gccli still exits with 0. (#97, #101)

### Fixed

- `goals list` without `--status` failed with HTTP 400, because the goal-service now requires a status. `--status` now defaults to `active` and accepts only `active`, `future` and `past`. (#67, #100)
- The hints for an expired token ("Token expired. Run: gccli auth login <email>") and for a rate limit ("Rate limited. Wait 30 and retry.") never appeared. gccli checked for its own copies of these error types, not the ones the API client returns. (#96, #102)

### Docs

- The README, the `gccli` skill and the docs site list the exit codes, describe the JSON error line and the default goal status, and say that delete commands need `--force` in scripts. (#100, #101, #103, #104)

## [1.11.0] - 2026-10-05

### Added

- **Unit system** - Tables and `--plain` output show distance, speed, pace and elevation in the unit system from your Garmin Connect profile. `metric` shows km, km/h, /km and m. `statute_us` shows mi, mph, /mi and ft. `statute_uk` shows mi, mph and /mi, with elevation in m. This applies to `activities list`, `activities search`, `activity <id>` and the `courses` tables. The new global flag `--units`, the env var `GCCLI_UNITS` and the config key `units` pin a system and skip the profile lookup. JSON output does not change. (#65)

### Changed

- Elevation in tables rounds to the nearest metre or foot, as Garmin Connect does. Before, gccli truncated it, so some values go up by 1. (#65)
- Bump dependencies in the minor-and-patch group: `golang.org/x/net` (0.58.0 -> 0.59.0) and `golang.org/x/term` (0.45.0 -> 0.46.0). Building from source now needs Go 1.26. (#63)

### Docs

- The README, the `gccli` skill and the docs site describe `--units`, `GCCLI_UNITS`, the precedence order, and why scripts that parse `--plain` output should pin a system. (#65)

### Tests

- E2E coverage for `GetMeasurementSystem` against the live API. (#65)
- Tests isolate the config dir through both `HOME` and `XDG_CONFIG_HOME`. Before, `make test` on Linux with `XDG_CONFIG_HOME` set overwrote the developer's real `~/.config/gccli/config.json`. (#64, #66)

## [1.10.0] - 2026-08-31

### Added

- **Workout updates** - `gccli workouts update <id> <file>` replaces an existing workout through `PUT /workout-service/workout/{id}`. The workout keeps its ID and its calendar schedule, so editing no longer means delete plus upload plus re-schedule. (#60)

### Fixed

- `workouts update` writes the ID argument into the payload before sending it. A file written for `workouts upload` carries no `workoutId` and used to reach the server unchanged; a file copied from another workout named that other workout in the body. The ID on the command line now wins, and gccli warns when the file disagrees. A non-numeric ID fails up front instead of returning a 404 from Garmin, and a payload that is not a JSON object is rejected with the file name. (#61)
- Prevent horizontal overflow on the docs site on mobile.

### Changed

- Bump dependencies in the minor-and-patch group: `github.com/alecthomas/kong` (1.16.0 -> 1.16.1) and `golang.org/x/net` (0.57.0 -> 0.58.0). (#59)

### Docs

- The README, the `gccli` skill and the docs site say that an update is a full replace, not a patch, and point at `gccli workouts detail <id>` for a complete payload. (#61)

### Tests

- E2E coverage for `workouts update` against the live API: the changed payload lands under the original ID, and a scheduled workout keeps its calendar entry across an update. (#61)
- The mock servers for `PUT /workout-service/workout/{id}` answer 204 with no body, which is what Garmin does. They used to echo the payload back. (#62)

## [1.9.2] - 2026-08-03

### Changed

- Bump `github.com/alecthomas/kong` from 1.15.0 to 1.16.0 in the minor-and-patch group. (#58)
- Bump `actions/setup-go` from 6 to 7 in GitHub Actions workflows. (#57)

### Docs

- Highlight agent integration via the bundled `gccli` skill in the README.

## [1.9.1] - 2026-07-14

### Changed

- Bump dependencies in the minor-and-patch group: `golang.org/x/net` (0.56.0 → 0.57.0) and `golang.org/x/term` (0.44.0 → 0.45.0). (#56)
- Bump `actions/checkout` from 6 to 7 in GitHub Actions workflows. (#55)

### Docs

- Replace the retired Go Report Card badge with Go version, release, and license badges.

## [1.9.0] - 2026-06-15

### Added

- **Course downloads** — `gccli courses download <id>` saves a course as a FIT or GPX file. Use `--format fit|gpx` (defaults to `fit`) and `-o/--output` to set the path. (#51, #54)
- **Expanded activity summary fields** — Activity summaries can now show elevation loss, aerobic training effect, anaerobic training effect, workout feel, and RPE. Feel maps Garmin's five-level scale to readable labels, and RPE is normalized from Garmin's 0-100 representation to a 0-10 scale. All are selectable through `activity_summary` config. (#52)

### Changed

- Bump dependencies in the minor-and-patch group, including `golang.org/x/net`. (#49, #50, #53)

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

[1.13.0]: https://github.com/bpauli/gccli/releases/tag/v1.13.0
[1.12.0]: https://github.com/bpauli/gccli/releases/tag/v1.12.0
[1.11.0]: https://github.com/bpauli/gccli/releases/tag/v1.11.0
[1.10.0]: https://github.com/bpauli/gccli/releases/tag/v1.10.0
[1.9.2]: https://github.com/bpauli/gccli/releases/tag/v1.9.2
[1.9.1]: https://github.com/bpauli/gccli/releases/tag/v1.9.1
[1.9.0]: https://github.com/bpauli/gccli/releases/tag/v1.9.0
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

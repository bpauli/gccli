# gccli — Garmin Connect in your terminal.

[![ci](https://github.com/bpauli/gccli/actions/workflows/ci.yml/badge.svg)](https://github.com/bpauli/gccli/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/bpauli/gccli)](go.mod)
[![Release](https://img.shields.io/github/v/release/bpauli/gccli)](https://github.com/bpauli/gccli/releases)
[![License](https://img.shields.io/github/license/bpauli/gccli)](LICENSE)
[![Docs](https://img.shields.io/badge/docs-gccli.sh-blue)](https://gccli.sh)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-support-yellow?logo=buymeacoffee)](https://buymeacoffee.com/mail7j)

Fast, script-friendly CLI for Garmin Connect. Access activities, health data, body composition, workouts, courses, devices, gear, goals, badges, and more. JSON-first output, multiple accounts, and credential storage in the OS keyring.

## Features

- **Activities** - list, search, view details, download (FIT/GPX/TCX/KML/CSV), upload, create manual entries (with date), rename, retype, describe, change privacy, mark as favorite, list favorites, trim the start or end and restore the original, change the privacy of all past activities, delete, set exercise sets for strength training, list activity types, list pending activities, list, add, edit and delete comments, like and unlike activities and comments
- **Exercise Catalog** — browse Garmin's exercise categories and exercises for strength training
- **Health Data** - daily summaries, steps, heart rate, resting HR, sleep, stress, HRV, SpO2, respiration, body battery (with daily low and high), floors, training readiness, daily and weekly training status, VO2max, fitness age, race predictions, endurance and hill scores with statistics, intensity minutes, lactate threshold, cycling FTP, critical swim speed, power curve, running economy, heat and altitude acclimation, running tolerance, cycling ability, activity progress totals per period
- **Body Composition** - weight tracking, latest weigh-in, weekly weight summaries, body fat, muscle mass, blood pressure, glucose, FIT file encoding for composition uploads
- **Workouts** - list, view, download as FIT, upload from JSON, update in place from JSON, create with sport types and targets (pace/HR/power/cadence), schedule (add/list/remove/export), send to a device, list workout types and sport types, view, set and clear strength benchmarks, delete
- **Courses** — list courses, view favorites, get full course detail, download as FIT/GPX, import GPX files as new courses, send courses directly to a device, delete courses
- **Segments** - list your segments and favorites, search segments by name or map area, view a segment, its leaderboard and your rank, list the segments that an activity crossed, create a segment from a part of an activity, add and remove favorites, delete segments
- **Devices** - list registered devices, view settings, solar data, alarms, primary/last-used device, set the primary activity tracker, paired sensors, backup summary and device backups, historical devices, device messages, cancel a pending device message
- **Gear** — list gear, usage stats, linked activities, defaults per activity type and set or unset them, link/unlink to activities, gear types, search gear brands and models, add, update, retire, unretire and delete gear, list, create, rename and delete gear collections, add gear to a collection and remove it
- **Goals & Badges** - list, add, update and delete activity goals, set daily goals for steps, floors, intensity minutes and hydration, set and delete the weight goal, earned/available/in-progress badges, badge details, challenges (show, join and leave badge and virtual challenges; create, view, answer, leave and delete ad-hoc challenges and invite people to them), personal records and record candidates, hide or show a record type, delete a personal record
- **Training** - browse, search and view training plans, show a plan's schedule, join and leave a plan, move and delete its workouts, show your active plans, list adaptive coaching workout types
- **Profile** - user profile and settings, view and change privacy settings, list, add and delete privacy zones, heart rate zones, power zones for all sports or one sport
- **Hydration** — view and log daily water intake
- **Nutrition** - view daily food logs, meals, and nutrition settings, search the Garmin food database, list recent and favorite foods, log foods and quick entries in a meal, delete log entries, and add and remove favorite foods (all but the first three need Garmin Connect+)
- **Events** — list, list upcoming, add, and delete calendar events (races, group rides, etc.) with goals and training priority support
- **Calendar** - month view of activities, workouts, events and notes, add, update, move and delete calendar notes, turn the iCalendar feed on and off, and change the web calendar preferences
- **Wellness** - menstrual cycle data and calendar, menstrual daily log (view, set, delete), pregnancy summary, jet lag trips (list, add, update, delete) and status
- **Social** - notifications (page with a cursor, dismiss and undismiss, unread state), view and change notification preferences, connections, connection requests, followers, following, follow and unfollow, follow requests, mute, user search, newsfeed, groups
- **Insights** - compare your steps, sleep, floors, VO2 max, speed, power and more with people of your age group and gender (needs Insights turned on in Garmin Connect)
- **Golf** - scorecards, the courses you played, golf and shot stats (drive, approach, chip, putt), the clubs in your bag, club types and leaderboards
- **Agent-ready** - ships a bundled agent skill (`skills/gccli/`) that plugs into AI assistants like Claude Code, Codex, Hermes, and OpenClaw so you can drive Garmin Connect in plain language
- **Multiple accounts** — manage multiple Garmin accounts via `--account` flag
- **Credential storage** in the OS keyring (macOS Keychain, Linux Secret Service), with a file fallback (see [Credential Storage](#credential-storage))
- **Auto-refreshing tokens** — authenticate once, tokens refresh automatically on 401
- **Resilient networking** - API calls retry on 429 (waits for `Retry-After` up to 30 s, 3 retries max) and read requests (GET) retry on 5xx (2 retries max, exponential backoff); writes are not resent on 5xx; an API request fails when no data moves for 60 s, and a transfer that keeps moving has no time limit
- **Shell completion** — generated scripts for bash, zsh, fish, and PowerShell
- **Parseable output** - JSON and plain/TSV modes for scripting and automation. `--json` output stays stable within a major version. Table and `--plain` output may change in a minor release
- **Your unit system** - tables show km or miles, pace per km or per mile, elevation in metres or feet, and weight in kg or lbs, as set in your Garmin Connect profile; pin one with `--units`

## Installation

### Homebrew (macOS / Linux)

```bash
brew install bpauli/tap/gccli
```

### Build from Source

Requires Go 1.26 or later.

```bash
git clone https://github.com/bpauli/gccli.git
cd gccli
make build
```

The binary is output to `./bin/gccli`.

Run:

```bash
./bin/gccli --help
```

Help:

- `gccli --help` shows top-level command groups.
- Drill down with `gccli <group> --help` (and deeper subcommands).
- Make shortcut: `make run -- --help` (or `make run -- activities --help`).

## Quick Start

### 1. Log In (Browser SSO)

```bash
gccli auth login you@example.com
```

This opens your browser for Garmin SSO authentication. The DI OAuth tokens are stored in your system keyring.

### 2. Log In (Headless)

For servers or environments without a browser:

```bash
gccli auth login you@example.com --headless
```

You'll be prompted for your password. If your account has two-factor authentication:

```bash
gccli auth login you@example.com --headless --mfa-code 123456
```

Login automatically saves your email as the default account, so you don't need to pass `--account` or set `GCCLI_ACCOUNT` on every command.

### 3. Test Authentication

```bash
gccli auth status
gccli activities list --limit 5
```

## Agent & AI Assistant Integration

`gccli` ships a ready-to-use **agent skill** at [`skills/gccli/SKILL.md`](skills/gccli/SKILL.md). It teaches an AI assistant the full command surface - auth, activities, health, body composition, workouts, courses, devices, gear, goals, events, and more - so you can ask in plain language and let the agent drive the CLI:

- "How did I sleep last week?"
- "Download my last three runs as GPX."
- "Log today's strength session: bench press 3x12@20kg, lat pulldowns 3x12@41kg."
- "Build me a tempo run and schedule it for Saturday."

The skill uses the portable `SKILL.md` format (frontmatter + Markdown) and works with any agent runtime that supports skills, including:

- **[Claude Code](https://claude.com/claude-code)** - Anthropic's terminal-native coding agent
- **Codex** - OpenAI's coding agent
- **Hermes** - agent runtime
- **OpenClaw** - the skill's frontmatter carries install metadata (Homebrew / build-from-source) so OpenClaw can provision the `gccli` binary automatically

### Setup

Clone the repo and point your agent at the bundled skill directory (copy or symlink it into wherever your agent discovers skills):

```bash
git clone https://github.com/bpauli/gccli.git
# then copy or symlink gccli/skills/gccli into your agent's skills directory
```

Make sure the `gccli` binary is installed (see [Installation](#installation)) and you have authenticated once with `gccli auth login`. Set `GCCLI_ACCOUNT` so the agent doesn't need to pass `--account` on every call. The skill instructs the agent to use `--json` output when parsing results and to confirm before destructive actions.

### Strength-training workflow (LLM-assisted)

The skill includes a guided workflow for logging free-text strength sessions: the agent fetches Garmin's exercise catalog (`gccli exercises list --json`), maps your described exercises to Garmin's `CATEGORY/EXERCISE_NAME` pairs, confirms the mapping with you, creates the activity, and records each set via `gccli activity exercise-sets set`.

## Authentication & Secrets

### Accounts and Tokens

`gccli` stores your OAuth tokens in a keyring backend. Default is auto-detection (best available backend for your OS/environment).

List current auth state:

```bash
gccli auth status
```

Print the raw OAuth2 access token (useful for scripting). If the token is expired, the command refreshes it first and prints the new token. If the refresh fails, it prints nothing and exits 3:

```bash
gccli auth token
```

Remove stored credentials:

```bash
gccli auth remove
```

### Keyring Backend

Backends:

- **auto** (default) — picks the best backend for the platform
- **keychain** — macOS Keychain
- **secret-service** — Linux D-Bus (GNOME Keyring, KWallet)
- **file** - on-disk files, encrypted with a fixed passphrase that is built into gccli (see [Credential Storage](#credential-storage))

Set backend via environment variable:

```bash
export GCCLI_KEYRING_BACKEND=file
```

Or in the config file (`config.json`):

```json
{
  "keyring_backend": "file"
}
```

On Linux, if D-Bus is unavailable, `gccli` automatically falls back to the file backend.

### Garmin Domain (China)

For Garmin China accounts (garmin.cn):

```bash
export GCCLI_DOMAIN=garmin.cn
```

Or in `config.json`:

```json
{
  "domain": "garmin.cn"
}
```

## Configuration

### Account Selection

The account is resolved in this order:

1. `--account` flag
2. `GCCLI_ACCOUNT` environment variable
3. Default account saved by `gccli auth login` (stored in `config.json`)

```bash
# Via flag
gccli activities list --account you@example.com

# Via environment
export GCCLI_ACCOUNT=you@example.com
gccli activities list

# Via login default (no flag or env var needed)
gccli auth login you@example.com
gccli activities list
```

### Output

- Default: human-friendly tables on stdout.
- `--json` / `-j`: JSON on stdout (best for scripting).
- `--plain`: tab-separated values on stdout, with no header row unless you add `--header`. Only these commands print TSV: `activities list`, `activities search`, `activities favorites`, `activities types`, `activity summary`, `body weekly`, `courses list`, `courses favorites`, `courses import`, `events list`, `nutrition search`, `nutrition recent`, `nutrition favorites`, `workouts list`, `workouts schedule list` and `exercises list` (without `--category`). Most other commands that return data print the same JSON in all modes, also without `--json`. `activities count` prints a single number. Use `--json` when a script parses the output.
- `--header`: with `--plain`, the first line holds the column names. `activity summary` has no column names, so it prints no header line. `--header` without `--plain`, or with `--json`, is a usage error (exit code 2). `GCCLI_HEADER` is not an error: table and JSON output ignore it.
- Table and `--plain` output escape the characters that would break a row or a column. A backslash prints as `\\`, a tab as `\t`, a newline as `\n`, a carriage return as `\r`, a vertical tab as `\v` and a form feed as `\f`. So each record is one line, and in `--plain` output only real tabs separate the fields. `--json` output does not change.
- Write commands (the commands that add, create, upload, import, update, set, rename, retype, describe, change, favorite, unfavorite, schedule, send, link, unlink, remove, reload or delete data in Garmin Connect) all follow one rule:
  - With `--json`, they print the Garmin response on stdout, indented like all JSON output. When Garmin sends an empty response, they print `{"ok": true}`.
  - Without `--json`, they print one message on stderr and nothing on stdout. `courses import` is the only exception. It prints the saved course as a table on stdout (TSV with `--plain`).
- An empty answer is not an error. When Garmin sends no body, the exit code is 0 and the output follows the shape of the Garmin answer: a command whose answer is a list prints `[]` with `--json`, and a command whose answer is one object prints `{}` in every mode. A list command that prints columns (`activities list`, `activities search`, `activities favorites`, `activities types`, `body weekly`, `courses list`, `courses favorites`, `events list`, `nutrition search`, `nutrition recent`, `nutrition favorites`, `workouts list` and `workouts schedule list`) prints the header only as a table and nothing with `--plain` (the header line with `--header`); every other list command prints `[]` in every mode. `activity summary` and `exercises list` show one object, so they print `{}` with `--json` and no row otherwise. [`internal/cmd/testdata/answer_shapes.tsv`](internal/cmd/testdata/answer_shapes.tsv) records the shape of the Garmin answer of every command that prints it (`array`, `object` or `unknown`, with the evidence for each; a shape that no Garmin answer confirmed is `unknown` and prints `{}`), and a test keeps the commands in step with it. `activities count` prints `0` (`{"count": 0}` with `--json`). A list answer without its list key gives no rows, and `--json` prints the answer unchanged.
- Human-facing messages (errors, warnings, info) go to stderr.
- Delete commands ask for confirmation in a terminal. Without a terminal (scripts, pipes, agents), they exit with an error unless you pass `--force`. If you answer no, the command prints `Cancelled` on stderr, prints nothing on stdout and exits with 0.
- Tables, TSV and JSON on stdout have no colors. The messages on stderr use colors. The `--help` text uses bold and faint text when stdout is a terminal, and `--color` does not change this.
- With `--color auto` (the default), gccli uses colors when stderr is a color terminal and neither `NO_COLOR` nor `CI` is set. `--color always` and `--color never` override this detection. `GCCLI_COLOR` sets the mode when you do not give `--color`. `--json` and `--plain` do not change the colors.

### Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Other error. A command that needs Garmin Connect+ also exits 1 on an account without it, and the message says so. |
| 2 | Usage error (unknown command, flag or value). The usage text goes to stderr. |
| 3 | Login needed: no account, no stored credentials, or the token expired and could not be refreshed |
| 4 | Garmin returned 404 Not Found |
| 5 | Garmin rate limit (HTTP 429) |

With `--json`, a runtime error goes to stderr as one JSON line. Usage errors stay plain text, because they happen before the flags are parsed. `code` is `error`, `auth_required`, `not_found` or `rate_limited`. `status` is the HTTP status, when there is one:

```json
{"error":{"code":"rate_limited","status":429,"message":"Rate limited. Wait 30 and retry."}}
```

### Units

Eight commands convert units in table and plain output: `activities list`, `activities search`, `activities favorites`, `activity summary`, `body weekly`, `courses list`, `courses favorites` and `courses import`. They show distances, speeds, paces, elevations and weights in the unit system from your Garmin Connect profile. To read the profile, each of these commands makes one extra request. It does this only in table or plain mode, and only when the unit system is `auto`.

| Value | Distance | Speed | Pace | Elevation | Weight |
| --- | --- | --- | --- | --- | --- |
| `metric` | km | km/h | /km | m | kg |
| `statute_us` | mi | mph | /mi | ft | lbs |
| `statute_uk` | mi | mph | /mi | m | lbs |
| `auto` (default) | from the Garmin profile | | | | |

The unit system is resolved in this order:

1. `--units` flag
2. `GCCLI_UNITS` environment variable
3. `"units"` in `config.json`
4. Garmin profile setting
5. `metric`

A pinned system (steps 1 to 3) skips the Garmin request. If gccli cannot read the profile setting, it prints a warning and uses metric.

JSON output is never converted. It is the raw Garmin data in meters, meters per second and grams.

Pin a system in scripts that parse `--plain` output. Otherwise the output changes when the Garmin profile setting changes:

```bash
gccli --units statute_us activities list
GCCLI_UNITS=metric gccli --plain activities list
```

### Config File

Config path:

- macOS: `~/Library/Application Support/gccli/config.json`
- Linux: `~/.config/gccli/config.json` (or `$XDG_CONFIG_HOME/gccli/config.json`)

Example:

```json
{
  "default_account": "you@example.com",
  "keyring_backend": "file",
  "domain": "garmin.com",
  "units": "statute_us"
}
```

`units` is optional. Leave it out (or set `auto`) to use the Garmin profile setting.

### Environment Variables

| Variable | Description |
| --- | --- |
| `GCCLI_ACCOUNT` | Default account email (overrides config file default) |
| `GCCLI_DOMAIN` | Garmin domain (`garmin.com` or `garmin.cn`) |
| `GCCLI_JSON` | Enable JSON output (`1`, `true`, `yes`) |
| `GCCLI_PLAIN` | Enable plain/TSV output (`1`, `true`, `yes`) |
| `GCCLI_HEADER` | Print the column names as the first line of plain output (`1`, `true`, `yes`). Table and JSON output ignore it |
| `GCCLI_COLOR` | Color mode: `auto`, `always`, `never` |
| `GCCLI_UNITS` | Unit system: `auto`, `metric`, `statute_us`, `statute_uk` (see [Units](#units)) |
| `GCCLI_KEYRING_BACKEND` | Keyring backend: `keychain`, `secret-service`, `file` |

## Security

### Credential Storage

OAuth tokens are stored in a keyring backend:

- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Fallback**: on-disk file store, encrypted with a fixed passphrase

The CLI uses [github.com/99designs/keyring](https://github.com/99designs/keyring) for storage. Tokens are keyed by email (`gccli:token:<email>`).

**The file backend does not protect your tokens.** gccli encrypts each token file with a fixed passphrase that is part of the gccli source code. You cannot set your own passphrase. Anyone who can read the file can decrypt the token. The files are in the `credentials` directory next to `config.json`, with file mode `0600`. Use the macOS Keychain or the Linux Secret Service when you can. If you must use the file backend, protect the `credentials` directory as you would protect a plaintext password file.

[docs/state.md](docs/state.md) describes every file and keyring item that gccli stores: the `config.json` keys, the lock files, the token fields and the token version.

## Commands

### Authentication

```bash
gccli auth login <email>               # Log in via browser SSO
gccli auth login <email> --headless    # Log in via email/password
gccli auth login <email> --headless --mfa-code <code>  # With MFA
gccli auth status                      # Show auth state and token expiry (exit 3 if no credentials are stored)
gccli auth token                       # Print OAuth2 access token (refresh it first if expired; exit 3 if the refresh fails)
gccli auth export                      # Export credentials (for transfer to another machine)
gccli auth import <token>              # Import credentials from another machine
gccli auth remove                      # Remove stored credentials
```

### Activities

```bash
# List and search
gccli activities list --limit 20 --offset 20
gccli activities list --type running
gccli activities count
gccli activities search --start 2024-01-01 --end 2024-12-31
gccli activities pending
gccli activities favorites --limit 5

# Change the privacy of all past activities, or of past running and cycling activities, and show the progress
gccli activities privacy set connections
gccli activities privacy set private --type running,cycling --force
gccli activities privacy status
gccli activities stats
gccli activities stats monthly

# View details
gccli activity summary <id>
gccli activity details <id>
gccli activity splits <id>
gccli activity typed-splits <id>
gccli activity split-summaries <id>
gccli activity weather <id>
gccli activity hr-zones <id>
gccli activity power-zones <id>
gccli activity exercise-sets <id>
gccli activity exercise-sets set <id> -e "BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60" -e "BENCH_PRESS/BARBELL_BENCH_PRESS:10@25:d25"
gccli activity gear <id>
gccli activity comments <id>
gccli activity likes <id>

# Like an activity, and remove the like
gccli activity likes add <id>
gccli activity likes remove <id>

# Comment on an activity, change the comment, like it, and delete it
gccli activity comments add <id> "Nice run"
gccli activity comments edit <id> <comment-id> "Great run"
gccli activity comments likes add <comment-id>
gccli activity comments likes remove <comment-id>
gccli activity comments delete <id> <comment-id> --force

# Trim an activity to the part from 2 minutes after the start to its end, and undo it
gccli activity trim <id> --from 2m
gccli activity trim <id> --from 2m --to 45m
gccli activity restore <id>

# Download and upload
gccli activity download <id> --format fit
gccli activity download <id> --format gpx --output track.gpx
gccli activity upload ./activity.fit

# Create and modify
gccli activity create --name "Morning Run" --type running --duration 30m --distance 5000
gccli activity create --name "Upper Body" --type strength_training --duration 34m --date 2026-03-13T19:01:00
gccli activity rename <id> "New Name"
gccli activity retype <id> --type-id 1 --type-key running
gccli activity delete <id>
gccli activity delete <id> --force

# Activity types
gccli activities types                   # List all Garmin activity types
```

### Health Data

```bash
# Daily summary
gccli health summary                   # Today
gccli health summary yesterday
gccli health summary 2024-06-15
gccli health summary 3d                # 3 days ago

# Vitals
gccli health steps                     # Step chart for today
gccli health steps daily --start 2024-01-01 --end 2024-01-31
gccli health steps weekly --weeks 4
gccli health steps monthly --months 6
gccli health calories weekly --weeks 4
gccli health pushes weekly --weeks 4    # Wheelchair pushes
gccli health pushes monthly --months 6
gccli health lifetime-totals
gccli health hr [date]                 # Heart rate
gccli health hr daily --start 2024-01-01 --end 2024-01-31
gccli health hr weekly --weeks 4
gccli health rhr [date]                # Resting heart rate
gccli health floors [date]             # Floors climbed
gccli health floors daily --start 2024-01-01 --end 2024-01-31
gccli health floors weekly --weeks 4
gccli health sleep [date]              # Sleep data
gccli health sleep naps [date]
gccli health sleep naps add 13:30 13:50   # Add a nap for today (times in your Garmin time zone)
gccli health sleep naps delete today 13:30  # Delete the nap that starts at 13:30 (with confirmation)
gccli health sleep daily --start 2024-01-01 --end 2024-01-28
gccli health sleep weekly
gccli health sleep overall --start 2024-01-01 --end 2024-01-31
gccli health respiration [date]
gccli health respiration daily --start 2024-01-01 --end 2024-01-31
gccli health spo2 [date]               # Blood oxygen
gccli health hrv [date]                # Heart rate variability

# Stress and recovery
gccli health stress                    # Stress for today
gccli health stress daily --start 2024-01-01 --end 2024-01-31
gccli health stress weekly --weeks 4
gccli health body-battery [date]
gccli health body-battery range --start 2024-01-01 --end 2024-01-07
gccli health body-battery daily --start 2024-01-01 --end 2024-01-28   # Daily low and high
gccli health training-readiness [date]
gccli health training-status [date]
gccli health training-status daily [date]
gccli health training-status weekly --start 2024-01-01 --end 2024-06-30

# Fitness metrics
gccli health fitness-age [date]
gccli health max-metrics [date]        # VO2max and more
gccli health lactate-threshold
gccli health cycling-ftp
gccli health critical-swim-speed [date]
gccli health critical-swim-speed range --start 2024-01-01 --end 2024-06-30
gccli health power-to-weight [date]
gccli health thresholds cycling-ftp --start 2024-01-01 --end 2024-06-30
gccli health thresholds lactate-heart-rate --start 2024-01-01 --end 2024-06-30
gccli health thresholds lactate-speed --start 2024-01-01 --end 2024-06-30
gccli health thresholds power-to-weight --start 2024-01-01 --end 2024-06-30
gccli health race-predictions
gccli health race-predictions range --start 2024-01-01 --end 2024-06-30
gccli health endurance-score [date]
gccli health endurance-score stats --start 2024-01-01 --end 2024-06-30 --aggregation monthly
gccli health hill-score [date]
gccli health hill-score stats --start 2024-01-01 --end 2024-06-30
gccli health running-tolerance --start 2024-01-01 --end 2024-06-30 --aggregation weekly
gccli health cycling-ability --start 2024-01-01 --end 2024-06-30
gccli health intensity-minutes [date]
gccli health intensity-minutes weekly --start 2024-01-01 --end 2024-01-31
gccli health acclimation --start 2024-01-01 --end 2024-01-07
gccli health acclimation daily [date]  # SpO2 acclimation of one day
gccli health one-rep-max [date]

# Activity progress (metric names come from health progress metrics)
gccli health progress metrics
gccli health progress --start 2024-01-01 --end 2024-06-30 --aggregation monthly --metric distance,duration --activity-type running
gccli health progress activities --start 2024-06-01 --end 2024-06-30 --metric distance --metric avgHr

# Wellness events
gccli health events [date]
gccli health lifestyle [date]
```

### Body Composition

```bash
# View data
gccli body composition                 # Today
gccli body composition --start 2024-01-01 --end 2024-01-31
gccli body weigh-ins --start 2024-01-01 --end 2024-01-31
gccli body daily-weigh-ins [date]
gccli body latest [date]
gccli body weekly [date]               # 52 weeks that end on the date (default today)

# Add data
gccli body add-weight 75.5 --unit kg
gccli body add-weight 166.4 --unit lbs
gccli body add-composition 75.5 --body-fat 15.2 --muscle-mass 35.0

# Blood pressure
gccli body blood-pressure --start 2024-01-01 --end 2024-01-31
gccli body add-blood-pressure --systolic 120 --diastolic 80 --pulse 65

# Glucose
gccli body glucose [date]

# Delete entries
gccli body delete-weight <pk> --date 2024-01-15
gccli body delete-blood-pressure <version> --date 2024-01-15
```

### Workouts

```bash
gccli workouts list --limit 20
gccli workouts detail <id>
gccli workouts download <id> --output workout.fit
gccli workouts upload ./workout.json   # See JSON structure below
gccli workouts update <id> ./workout.json   # Replace an existing workout, keeping its ID
gccli workouts schedule add <id> <YYYY-MM-DD>
gccli workouts schedule list <YYYY-MM-DD>
gccli workouts schedule list --start 2024-06-01 --end 2024-06-30
gccli workouts schedule remove <schedule-id>
gccli workouts schedule remove <schedule-id> --force
gccli workouts schedule export --start 2024-06-01
gccli workouts types
gccli workouts sport-types
gccli workouts benchmarks list
gccli workouts benchmarks set BARBELL_ROW --weight 50 --reps 5   # 1RM 56.3 kg
gccli workouts benchmarks set BARBELL_ROW --weight 110 --reps 5 --unit lbs
gccli workouts benchmarks clear BARBELL_ROW --force
gccli workouts send <id>                       # Works when one device can receive it
gccli workouts send <id> --device <device-id>
gccli workouts delete <id>

# Create a running workout with pace targets
gccli workouts create "Easy 30min Run" --type run \
  --step "warmup:5m@pace:5:30-6:00" \
  --step "run:20m@pace:5:00-5:30" \
  --step "cooldown:5m"

# Running with heart rate targets
gccli workouts create "HR Zone Run" --type run \
  --step "warmup:10m" \
  --step "run:20m@hr:140-160" \
  --step "cooldown:10m"

# Cycling with power targets
gccli workouts create "FTP Intervals" --type bike \
  --step "warmup:10m" \
  --step "run:5m@power:250-280" \
  --step "recovery:3m" \
  --step "run:5m@power:250-280" \
  --step "cooldown:10m"

# Imperial paces (miles)
gccli workouts create "Easy 30min Run" --type run \
  --step "warmup:5m@pace:8:51-9:39" \
  --step "run:20m@pace:8:03-8:51" \
  --step "cooldown:5m" \
  --unit mi

# Strength workout (no targets)
gccli workouts create "Full Body" --type strength \
  --step "warmup:5m" \
  --step "run:30m" \
  --step "cooldown:5m"
```

**Updating a workout in place.** `gccli workouts update` sends a full replacement, not a patch: whatever the file leaves out is gone from the workout. Start from the current server copy rather than from a hand-written fragment:

```bash
gccli workouts detail 123456 > workout.json
# edit workout.json
gccli workouts update 123456 workout.json
```

The workout keeps its ID and any calendar entries that point at it. `workoutId` in the file is set from the ID argument before the request goes out, so a payload written for `upload` works as-is; if the file already names a different workout, gccli warns and still updates the one given on the command line.

**Sending a workout to a device.** `gccli workouts send <id>` puts the workout in the message queue of a device. The device downloads it at its next sync. gccli keeps the devices that can receive the workout, as the Garmin Connect web app does: the device must be workout-capable and support the sport of the workout. When exactly one device is left, gccli sends to it. When more are left, the command fails before any write, and the error lists their IDs for `--device`. The send prints the message ID. Until the device syncs, `gccli devices messages cancel <message-id>` removes the message. After the sync, the cancel fails with exit code 1.

**Workout JSON structure** for `gccli workouts upload`:

```json
{
  "workoutName": "Tempo Run",
  "sportType": {
    "sportTypeId": 1,
    "sportTypeKey": "running"
  },
  "workoutSegments": [
    {
      "segmentOrder": 1,
      "sportType": {
        "sportTypeId": 1,
        "sportTypeKey": "running"
      },
      "workoutSteps": [
        {
          "type": "ExecutableStepDTO",
          "stepOrder": 1,
          "stepType": { "stepTypeId": 1, "stepTypeKey": "warmup" },
          "endCondition": { "conditionTypeId": 2, "conditionTypeKey": "time" },
          "endConditionValue": 600,
          "targetType": { "workoutTargetTypeId": 1, "workoutTargetTypeKey": "no.target" }
        },
        {
          "type": "ExecutableStepDTO",
          "stepOrder": 2,
          "stepType": { "stepTypeId": 3, "stepTypeKey": "interval" },
          "endCondition": { "conditionTypeId": 2, "conditionTypeKey": "time" },
          "endConditionValue": 1200,
          "targetType": { "workoutTargetTypeId": 6, "workoutTargetTypeKey": "pace.zone" },
          "targetValueOne": 3.333333,
          "targetValueTwo": 3.030303
        },
        {
          "type": "ExecutableStepDTO",
          "stepOrder": 3,
          "stepType": { "stepTypeId": 2, "stepTypeKey": "cooldown" },
          "endCondition": { "conditionTypeId": 2, "conditionTypeKey": "time" },
          "endConditionValue": 600,
          "targetType": { "workoutTargetTypeId": 1, "workoutTargetTypeKey": "no.target" }
        }
      ]
    }
  ]
}
```

Key reference values:
- **Sport types**: running (1), cycling (2), other (3), swimming (4), strength\_training (5), cardio\_training (6), yoga (7), pilates (8), hiit (9), multi\_sport (10), mobility (11)
- **Step types**: warmup (1), cooldown (2), interval (3), recovery (4), rest (5), other (7)
- **Target types**: no.target (1), power.zone (2), cadence (3), heart.rate.zone (4), pace.zone (6)
- **End condition**: time (2) — value in seconds
- **Pace values**: in m/s (e.g., 5:00/km = 1000/300 = 3.333 m/s)

### Exercise Catalog

```bash
# List all exercise categories
gccli exercises list

# List exercises in a category
gccli exercises list -c BENCH_PRESS

# Full catalog as JSON
gccli exercises list --json
```

### Courses

```bash
gccli courses list                             # List all courses
gccli courses favorites                        # List favorite courses
gccli courses detail <id>                      # View full course detail
gccli courses download <id>                    # Download course as FIT (default)
gccli courses download <id> --format gpx       # Download course as GPX
gccli courses download <id> -o ride.fit        # Download to a custom path
gccli courses import route.gpx                 # Import GPX as new course (default: cycling, private)
gccli courses import route.gpx --name "Ride"   # Import with custom name
gccli courses import route.gpx --type hiking   # Override activity type
gccli courses import route.gpx --privacy 1     # Import as public course
gccli courses send <course-id> <device-id>     # Send course to a device
gccli courses delete <id>                      # Delete a course
gccli courses delete <id> -f                   # Delete without confirmation
```

### Devices

```bash
gccli devices list
gccli devices settings <device-id>
gccli devices primary
gccli devices last-used
gccli devices alarms
gccli devices solar <device-id> --start 2024-06-01 --end 2024-06-30
gccli devices sensors
gccli devices backups                         # Backup summary (same as devices backups summary)
gccli devices backups list
gccli devices set-primary-tracker <device-id> # Not the training device of devices primary
gccli devices historical
gccli devices messages
gccli devices messages cancel <message-id>
```

### Gear

```bash
gccli gear list
gccli gear stats <uuid>
gccli gear activities <uuid> --limit 20
gccli gear defaults
gccli gear set-default <uuid> running     # also unset-default; a type can have more than one default gear
gccli gear link <uuid> <activity-id>
gccli gear unlink <uuid> <activity-id>
gccli gear types
gccli gear models "brooks ghost"   # Garmin's brands and models, for gear add --brand and --model
gccli gear add "Pegasus 41" --brand Nike --model Pegasus --type shoes
gccli gear update <uuid> --name "Pegasus 41 (old)"
gccli gear retire <uuid>          # gear unretire <uuid> undoes it
gccli gear delete <uuid> --force
gccli gear collections list
gccli gear collections create "Road shoes"
gccli gear collections add <collection-uuid> <gear-uuid>
gccli gear collections remove <collection-uuid> <gear-uuid>
```

### Goals, Badges & Challenges

```bash
gccli goals list                  # active goals (default)
gccli goals list --status past    # active, future or past
gccli goals add "Run 50 km" --period seven-days --activity-type running --distance 50km
gccli goals add "Ride 10 h" --period one-month --duration 10h   # starts on the 1st of this month
gccli goals update <id> --calories 3000   # a new target replaces the old one
gccli goals delete <id>           # (with confirmation)
gccli goals effective steps --date 2026-11-01   # also floors-climbed, intensity-minutes, hydration, weight-grams
gccli goals weight --start 2026-01-01 --end 2026-06-30
gccli goals weight set 72.5       # kg, or --unit lbs
gccli goals weight delete         # (with confirmation) an earlier weight goal applies again
gccli goals set floors-climbed 12   # also intensity-minutes (minutes) and hydration (ml)
gccli goals set hydration 2500 --date 2026-11-01
gccli goals steps set 9000
gccli goals steps auto            # Garmin's automatic steps goal
gccli badges earned
gccli badges available
gccli badges in-progress
gccli badges detail <id>
gccli badges compare <display-name>
gccli badges leaderboard
gccli challenges list
gccli challenges badge
gccli challenges available
gccli challenges completed
gccli challenges virtual available
gccli challenges virtual completed
gccli challenges virtual in-progress
gccli challenges show <uuid>      # a badge or virtual challenge
gccli challenges join <uuid>
gccli challenges leave <uuid>     # no prompt: join undoes it
gccli challenges adhoc            # ad-hoc challenges that are not finished (same as adhoc list)
gccli challenges adhoc history
gccli challenges adhoc show <uuid>
gccli challenges adhoc invites <uuid>       # each invite has its adHocChallengeInviteId
gccli challenges adhoc accept <invite-id>
gccli challenges adhoc decline <invite-id>
gccli challenges adhoc leave <uuid>         # no undo: only a new invite gets you back in
gccli challenges adhoc delete <uuid>        # only a challenge that you created
gccli challenges adhoc create "Step up" --type steps --invite ann   # a week from today
gccli challenges adhoc create "Ride" --type cycling --duration custom --start +1d --end +14d
gccli challenges adhoc invite <uuid> ann bob   # only people who follow you and whom you follow
gccli records                     # personal records (same as records list)
gccli records types
gccli records activity <activityId>
gccli records candidates          # records that Garmin did not accept yet
gccli records hide <typeId>       # hide a record type on the records page
gccli records unhide <typeId>
gccli records delete <recordId>   # asks first; --force skips the prompt
```

### Profile

```bash
gccli profile
gccli profile settings
gccli profile privacy
gccli profile privacy set --activities connections --badges private
gccli profile privacy set --profile public --wellness groups
gccli profile privacy-zones
gccli profile privacy-zones add --lat 52.52 --lon 13.405 --radius 500
gccli profile privacy-zones add --lat 40.7 --lon=-74.0 --radius 1000   # A negative value needs the = form
gccli profile privacy-zones delete <id>          # Delete a privacy zone (with confirmation)
gccli profile hr-zones
gccli profile power-zones                 # All sports
gccli profile power-zones sport cycling   # One sport: cycling, running, rowing or cross-country-skiing
```

### Hydration

```bash
gccli hydration [date]                 # View hydration data
gccli hydration daily --start 2024-01-01 --end 2024-01-31
gccli hydration weekly --weeks 4
gccli hydration add 500                # Log 500ml of water
gccli hydration add 500 --date 2024-06-15
```

### Nutrition

```bash
gccli nutrition [date]                 # View daily food log
gccli nutrition meals [date]           # View meal summaries
gccli nutrition settings [date]        # View nutrition settings
gccli nutrition search "greek yogurt"  # Search the food database (needs Garmin Connect+)
gccli nutrition recent                 # Foods logged recently (needs Garmin Connect+)
gccli nutrition favorites              # Favorite foods (needs Garmin Connect+)

# Writes (need Garmin Connect+ and nutrition set up in Garmin Connect)
gccli nutrition add today 35718 --query apple --meal breakfast     # Log a food that search found
gccli nutrition add today 35718 -q apple -m lunch --serving oz --servings 3
gccli nutrition quick-add today --meal snacks --calories 120 --carbs 15 --name "Coffee with milk"
gccli nutrition delete today <log-id> --force                       # Delete log entries
gccli nutrition favorites add 35718 --query apple                  # Save a favorite food
gccli nutrition favorites remove 35718
```

`nutrition add` and `nutrition favorites add` need `--query`, the search text with which `nutrition search` lists the food, because Garmin has no read of one food by ID. `--serving` picks a serving by its unit or ID (default the first serving of the food). `--meal` takes the meals of the day: `breakfast`, `lunch`, `dinner`, `snacks`, or a custom meal you enabled. The meal time defaults to the start time of the meal, or to the current time for a meal without one, such as snacks. The add commands print the log ID of the new entry, and `nutrition delete` takes it.

### Training Plans

```bash
gccli training plans --locale en
gccli training plan <id>
gccli training active
gccli training active --start 2026-01-01
gccli training search --sport Running --goal 5k --level Beginner   # Find a catalog plan
gccli training schedule <id>                                       # Weeks, days, workouts and task IDs
gccli training join <id> --start 2026-11-02                        # Prints the ID of your copy of the plan
gccli training task move <plan-id> <task-id> 2026-11-04
gccli training task delete <plan-id> <task-id> --force
gccli training leave <plan-id> --force
gccli training coach-workouts
```

`training join` copies a catalog plan into your account under a new ID and puts its workouts in your calendar. The other plan commands take that new ID. `training schedule` prints a task ID for each workout of a joined plan, and `training task move` and `training task delete` take it. Garmin gives a moved workout a new task ID, which the command prints. A deleted workout becomes a rest day.

### Events

```bash
gccli events list                      # List upcoming events
gccli events list --start 2024-01-01 --limit 50
gccli events list --limit 50 --offset 50  # Next page (--offset must be a multiple of --limit)
# On every paginated command --limit must be at least 1 and --offset must not be negative (exit code 2).
gccli events list --sort eventDate_desc
gccli events upcoming
gccli events delete <id>               # Delete an event (with confirmation)
gccli events delete <id> -f            # Delete without confirmation

# Add a race event
gccli events add --name "Berlin Marathon" --date 2026-09-27 --type running \
  --race --location "Berlin, Germany" --distance 42.195km \
  --time 09:15 --timezone Europe/Berlin

# Add an event with a goal and training priority
gccli events add --name "Spring 10K" --date 2026-05-10 --type running \
  --race --distance 10km --goal 40m --training
```

**Event flags:**
- **Required:** `--name`, `--date` (YYYY-MM-DD, today, +30d), `--type` (running, trail_running, cycling, gravel_cycling, mountain_biking, swimming, triathlon, multi_sport, hiking, walking, fitness_equipment, motorcycling, winter_sport, other)
- **Optional:** `--race`, `--location`, `--time` (HH:MM), `--timezone`, `--distance` (e.g. 10km, 26.2mi, 400m), `--private`, `--note`, `--url`
- **Goal & training priority:** `--goal` (duration, e.g. 50m, 1h30m, 2400s), `--primary` or `--training` (mutually exclusive)

### Calendar notes

```bash
gccli calendar notes add 2026-10-20 "Long run" --content "Easy pace, 25 km"
gccli calendar notes update <id> --name "Tempo run"
gccli calendar notes update <id> --content ""   # Clear the note text
gccli calendar notes move <id> +1d             # Move the note to tomorrow
gccli calendar notes delete <id>               # Delete a note (with confirmation)
```

`calendar notes add` prints the note ID. `calendar notes update` reads the note first and sends it back with your changes, so the fields you do not give stay as they are.

### Calendar month and feed

```bash
gccli calendar month                  # Items of this month
gccli calendar month 2026 10          # Items of October 2026
gccli calendar export enable          # Turn on the iCalendar feed and print its URL
gccli calendar export show --plain    # Print the feed URL alone
gccli calendar export disable         # Turn off the feed (with confirmation)
gccli calendar preferences set showWeeklyTotal false
```

`calendar month` lists the date, type, ID and title of each item. Like the web calendar, it also shows the days before and after the month that fill its weeks. Add the feed URL to a calendar app as a subscription. After `calendar export disable` the URL gives an empty answer. `calendar export enable` keeps a feed that is already on, so the URL stays the same.

### Wellness

```bash
gccli wellness menstrual-cycle                    # Cycle day view of today ({} for a day outside a tracked cycle)
gccli wellness menstrual-summary 2026-10-01       # Cycle summary of a date
gccli wellness menstrual-calendar --start 2024-01-01 --end 2024-03-31
gccli wellness menstrual-log 2026-10-01           # Daily log ({} for a day without a log)
gccli wellness menstrual-log set 2026-10-01 --flow light --symptoms cramps,headache
gccli wellness menstrual-log set 2026-10-01 --notes ""   # Clear the notes, keep the other fields
gccli wellness menstrual-log delete 2026-10-01    # Delete the daily log (with confirmation)
gccli wellness pregnancy-summary                  # Pregnancy snapshot ({} when there is none)
gccli wellness jet-lag list
gccli wellness jet-lag add --from "Frankfurt, Germany" --to "New York, USA" \
  --depart 2026-11-02T10:00+01:00 --arrive 2026-11-02T13:00-05:00 \
  --return-depart 2026-11-09T18:00-05:00 --return-arrive 2026-11-10T08:00+01:00
gccli wellness jet-lag update <id> --return-depart 2026-11-10T18:00-05:00 --return-arrive 2026-11-11T08:00+01:00
gccli wellness jet-lag delete <id>                # Delete a trip (with confirmation)
gccli wellness jet-lag status 2024-03-15
```

Each flight time is the local time at its place, with the UTC offset there. The trip runs from the day of the first departure to the day of the last arrival. For a flight with a stop, give the first departure and the last arrival. `jet-lag update` changes only the fields you give: `--from` and `--to` also change the return flight, so it goes back the same way.

### Social

```bash
gccli notifications list               # List your notifications (50 per page)
gccli notifications list --cursor <next-cursor>   # The next page: pagination.nextCursor of the last page
gccli notifications unread             # Show whether there are unread notifications
gccli notifications dismiss <id>       # Dismiss a notification
gccli notifications undismiss <id>     # Bring it back
gccli notifications preferences        # Show your notification preferences
gccli notifications preferences set allowEmailWeeklySummary false   # Turn one preference off
gccli social connections               # List your connections
gccli social connections --limit 50 --offset 50
gccli social followers                 # List your followers
gccli social followers <display-name>  # List the followers of another user
gccli social following                 # List the users you follow
gccli social following <display-name>  # List the users that another user follows
gccli social requests                  # List pending connection requests
gccli social feed                      # Show your newsfeed
gccli social search anna               # Search users by name
gccli social follow <display-name>     # Follow a user (a private profile gets a follow request)
gccli social unfollow <display-name>   # Stop following a user
gccli social follow-requests           # List the users who asked to follow you
gccli social accept <display-name>     # Accept a follow request
gccli social decline <display-name>    # Decline a follow request
gccli social remove-follower <display-name>
gccli social mute <display-name>       # Mute a user that you follow
gccli social unmute <display-name>
gccli social muted                     # List the users that you muted
gccli groups list                      # List your groups
```

`unfollow`, `remove-follower` and `unmute` read the follow status or the muted list first, because Garmin answers 204 whether there is something to remove or not. They exit 1 when there is nothing to remove. `unfollow` and `remove-follower` ask first (`--force` skips the question). Garmin no longer takes new connection requests (410 `API_DEPRECATED`), so gccli has no command to send one.

### Data Reload

```bash
gccli reload                           # Reload today's data
gccli reload 2024-06-15                # Reload specific date
```

### Shell Completion

Generate completion scripts for your shell:

```bash
gccli completion bash
gccli completion zsh
gccli completion fish
gccli completion powershell
```

To enable completions, add the appropriate line to your shell profile:

**Bash** (`~/.bashrc`):
```bash
source <(gccli completion bash)
```

**Zsh** (`~/.zshrc`):
```bash
source <(gccli completion zsh)
```

**Fish** (`~/.config/fish/config.fish`):
```fish
gccli completion fish | source
```

**PowerShell** (`$PROFILE`):
```powershell
gccli completion powershell | Invoke-Expression
```

Completions cover all commands, subcommands, flags, and enum flag values (e.g. `--color auto|always|never`).

<!-- gccli:commands:begin -->
<!-- Generated from the command tree. Do not edit. Run: go test ./internal/cmd -run TestDocsUpToDate -update -->

## Command Reference

### Authentication

- `gccli auth login <email> [flags]`: Log in to Garmin Connect.
  - `<email>`: Garmin account email address.
  - `--headless`: Use headless login (email/password) instead of browser SSO.
  - `--mfa-code=STRING`: MFA code for two-factor authentication.
- `gccli auth status`: Show authentication status.
- `gccli auth remove`: Remove stored credentials.
- `gccli auth token`: Print current access token, refreshing it first when expired.
- `gccli auth export`: Export credentials for transfer to another machine.
- `gccli auth import [<token>]`: Import credentials from another machine.
  - `[<token>]`: Base64-encoded token string from 'auth export'. Reads from stdin if omitted.

### Activities

- `gccli activities list [flags]`: List recent activities. Also runs as `gccli activities [flags]`.
  - `-l, --limit=20`: Maximum number of activities to return.
  - `--offset=0`: Number of activities to skip.
  - `-t, --type=STRING`: Filter by activity type (e.g. running, cycling).
- `gccli activities count`: Show total activity count.
- `gccli activities search --start=DATE [flags]`: Search activities by date range.
  - `-l, --limit=20`: Maximum number of activities to return.
  - `--offset=0`: Number of activities to skip.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli activities favorites [flags]`: List your favorite activities.
  - `-l, --limit=20`: Maximum number of activities to return.
- `gccli activities types`: List all activity types supported by Garmin.
- `gccli activities pending`: List pending activities.
- `gccli activities stats view`: Show activity totals by activity type. Also runs as `gccli activities stats`.
- `gccli activities stats monthly`: Show monthly activity totals by activity type.
- `gccli activities privacy status`: Show the last privacy change of past activities. Also runs as `gccli activities privacy`.
- `gccli activities privacy set <level> [flags]`: Change the privacy of all past activities, or of past activities of some types.
  - `<level>`: Who can see the activities: public, private or connections.
  - `--type=KEY,...`: Top-level activity type key (e.g. running, cycling). Repeat the flag or separate with commas. Default: all activity types.
  - `-f, --force`: Skip confirmation prompt.
- `gccli activity summary <id>`: Show activity summary. Also runs as `gccli activity <id>`.
  - `<id>`: Activity ID.
- `gccli activity details <id> [flags]`: Show full activity details.
  - `<id>`: Activity ID.
  - `--max-chart=2000`: Maximum chart data points.
  - `--max-poly=4000`: Maximum polyline data points.
- `gccli activity splits <id>`: Show activity splits.
  - `<id>`: Activity ID.
- `gccli activity typed-splits <id>`: Show typed splits.
  - `<id>`: Activity ID.
- `gccli activity split-summaries <id>`: Show split summaries.
  - `<id>`: Activity ID.
- `gccli activity weather <id>`: Show weather conditions.
  - `<id>`: Activity ID.
- `gccli activity hr-zones <id>`: Show heart rate time in zones.
  - `<id>`: Activity ID.
- `gccli activity power-zones <id>`: Show power time in zones.
  - `<id>`: Activity ID.
- `gccli activity segments <id>`: List the segments that an activity crossed.
  - `<id>`: Activity ID.
- `gccli activity exercise-sets show <id>`: Show exercise sets. Also runs as `gccli activity exercise-sets <id>`.
  - `<id>`: Activity ID.
- `gccli activity exercise-sets set --exercise=EXERCISE,... <id>`: Set exercise sets for a strength training activity.
  - `<id>`: Activity ID.
  - `-e, --exercise=EXERCISE,...`: Exercise set in format CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS] (e.g. BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60).
- `gccli activity gear <id>`: Show linked gear.
  - `<id>`: Activity ID.
- `gccli activity comments list <id>`: List the comments on an activity. Also runs as `gccli activity comments <id>`.
  - `<id>`: Activity ID.
- `gccli activity comments add <id> <text>`: Add a comment to an activity.
  - `<id>`: Activity ID.
  - `<text>`: Comment text.
- `gccli activity comments edit <id> <comment-id> <text>`: Change the text of your comment on an activity.
  - `<id>`: Activity ID.
  - `<comment-id>`: Comment ID (conversationCommentUuid in activity comments list).
  - `<text>`: New comment text.
- `gccli activity comments delete <id> <comment-id> [flags]`: Delete a comment on an activity.
  - `<id>`: Activity ID.
  - `<comment-id>`: Comment ID (conversationCommentUuid in activity comments list).
  - `-f, --force`: Skip confirmation prompt.
- `gccli activity comments likes list <comment-id>`: List the likes of a comment. Also runs as `gccli activity comments likes <comment-id>`.
  - `<comment-id>`: Comment ID (conversationCommentUuid in activity comments list).
- `gccli activity comments likes add <comment-id>`: Like a comment.
  - `<comment-id>`: Comment ID (conversationCommentUuid in activity comments list).
- `gccli activity comments likes remove <comment-id>`: Remove your like from a comment.
  - `<comment-id>`: Comment ID (conversationCommentUuid in activity comments list).
- `gccli activity likes list <id>`: List the likes of an activity. Also runs as `gccli activity likes <id>`.
  - `<id>`: Activity ID.
- `gccli activity likes add <id>`: Like an activity.
  - `<id>`: Activity ID.
- `gccli activity likes remove <id>`: Remove your like from an activity.
  - `<id>`: Activity ID.
- `gccli activity download <id> [flags]`: Download activity file.
  - `<id>`: Activity ID.
  - `--format="fit"`: Download format: fit, gpx, tcx, kml, csv.
  - `-o, --output=STRING`: Output file path (default: activity_{id}.{format}).
- `gccli activity upload <file>`: Upload an activity file (FIT/GPX/TCX).
  - `<file>`: Path to activity file (FIT, GPX, or TCX).
- `gccli activity create --name=FREE-TEXT --type=FREE-TEXT --duration=DURATION [flags]`: Create a manual activity.
  - `--name=FREE-TEXT`: Activity name.
  - `--type=FREE-TEXT`: Activity type key (e.g. running, cycling, swimming).
  - `--distance=FLOAT-64`: Distance in meters.
  - `--duration=DURATION`: Duration (e.g. 30m, 1h15m).
  - `--date=DATE-TIME`: Start date and time in local time: YYYY-MM-DDTHH:MM:SS, YYYY-MM-DDTHH:MM or YYYY-MM-DD, with T or a space. Defaults to now.
- `gccli activity rename <id> <name>`: Rename an activity.
  - `<id>`: Activity ID.
  - `<name>`: New activity name.
- `gccli activity retype --type-id=INT --type-key=FREE-TEXT <id> [flags]`: Change activity type.
  - `<id>`: Activity ID.
  - `--type-id=INT`: Activity type ID.
  - `--type-key=FREE-TEXT`: Activity type key (e.g. running, cycling).
  - `--parent-type-id=0`: Parent activity type ID.
- `gccli activity describe <id> <text>`: Set or clear the description of an activity.
  - `<id>`: Activity ID.
  - `<text>`: New description. An empty text ("") clears it.
- `gccli activity privacy <id> <level>`: Change who can see an activity.
  - `<id>`: Activity ID.
  - `<level>`: Who can see the activity: public, private or connections.
- `gccli activity trim <id> [flags]`: Trim the start or end of an activity.
  - `<id>`: Activity ID.
  - `--from=FROM`: New start, as the time since the activity start (e.g. 2m, 1h5m). Default: the activity start.
  - `--to=TO`: New end, as the time since the activity start (e.g. 50m, 1h5m). Default: the activity end.
- `gccli activity restore <id>`: Restore the original of a trimmed activity.
  - `<id>`: Activity ID.
- `gccli activity favorite <id>`: Mark an activity as a favorite.
  - `<id>`: Activity ID.
- `gccli activity unfavorite <id>`: Remove the favorite mark from an activity.
  - `<id>`: Activity ID.
- `gccli activity delete <id> [flags]`: Delete an activity.
  - `<id>`: Activity ID.
  - `-f, --force`: Skip confirmation prompt.
- `gccli exercises list [flags]`: List available exercise categories and exercises. Also runs as `gccli exercises [flags]`.
  - `-c, --category=STRING`: Filter by category (e.g. BENCH_PRESS).

### Health Data

- `gccli health summary [<date>]`: Show daily health summary. Also runs as `gccli health [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health steps view [<date>]`: Show step chart data for a date. Also runs as `gccli health steps [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health steps daily --start=DATE --end=DATE`: Show daily step totals for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health steps weekly [flags]`: Show weekly step totals.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--weeks=4`: Number of weeks.
- `gccli health steps monthly [flags]`: Show monthly step totals.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--months=6`: Number of months.
- `gccli health calories weekly [flags]`: Show weekly calorie totals.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--weeks=4`: Number of weeks.
- `gccli health pushes weekly [flags]`: Show weekly wheelchair push totals.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--weeks=4`: Number of weeks.
- `gccli health pushes monthly [flags]`: Show monthly wheelchair push totals.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--months=6`: Number of months.
- `gccli health lifetime-totals`: Show lifetime totals.
- `gccli health hr view [<date>]`: Show heart rate data for a date. Also runs as `gccli health hr [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health hr daily --start=DATE --end=DATE`: Show daily heart rate statistics for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health hr weekly [flags]`: Show weekly heart rate statistics.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--weeks=4`: Number of weeks.
- `gccli health rhr [<date>]`: Show resting heart rate data.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health floors view [<date>]`: Show floors climbed for a date. Also runs as `gccli health floors [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health floors daily --start=DATE --end=DATE`: Show daily floors climbed for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health floors weekly [flags]`: Show weekly floors climbed.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--weeks=4`: Number of weeks.
- `gccli health sleep view [<date>]`: Show sleep data for a date. Also runs as `gccli health sleep [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health sleep naps view [<date>]`: Show naps for a date. Also runs as `gccli health sleep naps [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli health sleep naps add <start> <end>`: Add a nap for today. Garmin takes naps only for the current day.
  - `<start>`: Start time (HH:MM) in your Garmin time zone.
  - `<end>`: End time (HH:MM) in your Garmin time zone. Must not be later than now.
- `gccli health sleep naps delete <date> <start> [flags]`: Delete a nap.
  - `<date>`: Date of the nap (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `<start>`: Start time of the nap (HH:MM) in your Garmin time zone.
  - `-f, --force`: Skip confirmation prompt.
- `gccli health sleep daily --start=DATE --end=DATE`: Show daily sleep statistics for a date range (at most 28 days).
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health sleep weekly [flags]`: Show weekly sleep statistics for the 52 weeks up to a date.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli health sleep overall --start=DATE --end=DATE`: Show overall sleep statistics for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health stress view [<date>]`: Show stress data for a date. Also runs as `gccli health stress [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health stress daily --start=DATE --end=DATE`: Show daily stress statistics for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health stress weekly [flags]`: Show weekly stress data.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--weeks=4`: Number of weeks.
- `gccli health respiration view [<date>]`: Show respiration data for a date. Also runs as `gccli health respiration [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health respiration daily --start=DATE --end=DATE`: Show daily respiration statistics for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health spo2 [<date>]`: Show SpO2 (blood oxygen) data.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health hrv [<date>]`: Show heart rate variability data.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health body-battery view [<date>]`: Show body battery data for a date. Also runs as `gccli health body-battery [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health body-battery range --start=DATE --end=DATE`: Show body battery data for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health body-battery daily --start=DATE --end=DATE`: Show the daily low and high body battery for a date range (at most 28 days).
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health intensity-minutes view [<date>]`: Show intensity minutes for a date. Also runs as `gccli health intensity-minutes [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health intensity-minutes weekly --start=DATE --end=DATE`: Show weekly intensity minutes for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health training-readiness [<date>]`: Show training readiness data.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health training-status view [<date>]`: Show aggregated training status for a date. Also runs as `gccli health training-status [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health training-status daily [<date>]`: Show daily training status for a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli health training-status weekly --start=DATE --end=DATE`: Show weekly training status for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health fitness-age [<date>]`: Show fitness age data.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health max-metrics [<date>]`: Show VO2max and max metrics data.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health lactate-threshold`: Show latest lactate threshold.
- `gccli health cycling-ftp`: Show latest cycling FTP.
- `gccli health critical-swim-speed view [<date>]`: Show latest critical swim speed as of a date. Also runs as `gccli health critical-swim-speed [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli health critical-swim-speed range --start=DATE --end=DATE`: Show critical swim speed for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health power-to-weight [<date>]`: Show latest power-to-weight data as of a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli health thresholds cycling-ftp --start=DATE --end=DATE`: Show cycling FTP for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health thresholds lactate-heart-rate --start=DATE --end=DATE`: Show lactate threshold heart rate for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health thresholds lactate-speed --start=DATE --end=DATE`: Show lactate threshold speed for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health thresholds power-to-weight --start=DATE --end=DATE`: Show power-to-weight data for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health race-predictions view`: Show current race predictions. Also runs as `gccli health race-predictions`.
- `gccli health race-predictions range --start=DATE --end=DATE`: Show race predictions for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health endurance-score view [<date>]`: Show endurance score data for a date. Also runs as `gccli health endurance-score [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health endurance-score stats --start=DATE --end=DATE [flags]`: Show endurance score statistics for a date range (at most 400 days).
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--aggregation="weekly"`: Period of each group: weekly or monthly.
- `gccli health hill-score view [<date>]`: Show hill score data for a date. Also runs as `gccli health hill-score [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health hill-score stats --start=DATE --end=DATE [flags]`: Show hill score statistics for a date range (at most 400 days).
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--aggregation="weekly"`: Period of each value: daily, weekly or monthly.
- `gccli health power-curve --start=DATE --end=DATE`: Show power curve for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health running-tolerance --start=DATE --end=DATE [flags]`: Show running tolerance for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--aggregation="weekly"`: Period of each value: daily, weekly or monthly.
- `gccli health cycling-ability --start=DATE --end=DATE [flags]`: Show cycling ability for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--aggregation="weekly"`: Period of each value: daily (at most 30 days), weekly or monthly (at most 400 days).
- `gccli health progress summary --start=DATE --end=DATE --metric=METRIC,... [flags]`: Show activity totals per period for a date range. Also runs as `gccli health progress --start=DATE --end=DATE --metric=METRIC,... [flags]`.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--aggregation="weekly"`: Period of each total: daily, weekly, monthly, yearly or all.
  - `--metric=METRIC,...`: Metric to total, as health progress metrics prints it (e.g. distance, duration). Repeat the flag or separate with commas.
  - `--activity-type=ACTIVITY-TYPE,...`: Activity type, as health progress metrics prints it (e.g. running, cycling). Repeat the flag or separate with commas. Defaults to all types.
- `gccli health progress activities --start=DATE --end=DATE --metric=METRIC,... [flags]`: Show metric values of each activity in a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--metric=METRIC,...`: Metric to show, as health progress metrics prints it (e.g. distance, avgHr). Repeat the flag or separate with commas.
  - `--activity-type=ACTIVITY-TYPE,...`: Activity type, as health progress metrics prints it (e.g. running, cycling). Repeat the flag or separate with commas. Defaults to all types.
- `gccli health progress metrics [flags]`: List the metrics of each activity type.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). With --end, only types with activities in the range.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Needs --start.
- `gccli health running-economy [<date>]`: Show latest running economy as of a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli health acclimation range --start=DATE --end=DATE`: Show acclimation data for a date range. Also runs as `gccli health acclimation --start=DATE --end=DATE`.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health acclimation daily [<date>]`: Show SpO2 acclimation for a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli health one-rep-max [<date>]`: Show current estimated one-rep max values as of a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli health events [<date>]`: Show daily wellness events.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health lifestyle [<date>]`: Show lifestyle logging data.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.

### Body Composition

- `gccli body composition [<date>] [flags]`: Show body composition data. Also runs as `gccli body [<date>] [flags]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
  - `--start=DATE`: Start date for range query (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date for range query (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli body weigh-ins --start=DATE --end=DATE`: Show weigh-in data for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli body daily-weigh-ins [<date>]`: Show weigh-in data for a single day.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli body latest [<date>]`: Show the latest weigh-in as of a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli body weekly [<end>]`: Show weekly weight summaries for the 52 weeks that end on a date.
  - `[<end>]`: Last day of the 52 weeks (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli body add-weight <value> [flags]`: Add a weigh-in entry.
  - `<value>`: Weight value.
  - `--unit="kg"`: Unit: kg or lbs.
- `gccli body add-composition <weight> [flags]`: Add body composition data (weight, body fat, muscle mass, etc.) via FIT upload.
  - `<weight>`: Weight in kg.
  - `--body-fat=BODY-FAT`: Body fat percentage.
  - `--percent-hydration=PERCENT-HYDRATION`: Hydration percentage.
  - `--visceral-fat-mass=VISCERAL-FAT-MASS`: Visceral fat mass in kg.
  - `--bone-mass=BONE-MASS`: Bone mass in kg.
  - `--muscle-mass=MUSCLE-MASS`: Muscle mass in kg.
  - `--basal-met=BASAL-MET`: Basal metabolic rate (kcal/day).
  - `--active-met=ACTIVE-MET`: Active metabolic rate (kcal/day).
  - `--physique-rating=PHYSIQUE-RATING`: Physique rating.
  - `--metabolic-age=METABOLIC-AGE`: Metabolic age.
  - `--visceral-fat-rating=VISCERAL-FAT-RATING`: Visceral fat rating.
  - `--bmi=BMI`: Body mass index.
- `gccli body delete-weight --date=DATE <pk> [flags]`: Delete a weigh-in entry.
  - `<pk>`: Weigh-in version (primary key).
  - `--date=DATE`: Date of the weigh-in (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `-f, --force`: Skip confirmation prompt.
- `gccli body blood-pressure --start=DATE --end=DATE`: Show blood pressure data.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli body add-blood-pressure --systolic=INT --diastolic=INT --pulse=INT [flags]`: Add a blood pressure measurement.
  - `--systolic=INT`: Systolic pressure (mmHg).
  - `--diastolic=INT`: Diastolic pressure (mmHg).
  - `--pulse=INT`: Pulse rate (bpm).
  - `--notes=STRING`: Optional notes.
- `gccli body delete-blood-pressure --date=DATE <version> [flags]`: Delete a blood pressure entry.
  - `<version>`: Blood pressure entry version.
  - `--date=DATE`: Date of the measurement (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `-f, --force`: Skip confirmation prompt.
- `gccli body glucose [<date>]`: Show glucose readings for a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.

### Workouts

- `gccli workouts list [flags]`: List workouts. Also runs as `gccli workouts [flags]`.
  - `-l, --limit=20`: Maximum number of workouts to return.
  - `--offset=0`: Number of workouts to skip.
- `gccli workouts detail <id>`: View workout details.
  - `<id>`: Workout ID.
- `gccli workouts download <id> [flags]`: Download workout as FIT file.
  - `<id>`: Workout ID.
  - `-o, --output=STRING`: Output file path (default: workout_{id}.fit).
- `gccli workouts upload <file>`: Upload workout from JSON file.
  - `<file>`: Path to workout JSON file.
- `gccli workouts update <id> <file>`: Update an existing workout from a JSON file.
  - `<id>`: Workout ID to update.
  - `<file>`: Path to workout JSON file.
- `gccli workouts create --type=STRING --step=STEP,... <name> [flags]`: Create a workout with sport type and optional targets.
  - `<name>`: Workout name.
  - `--type=STRING`: Sport type. One of: run, bike, swim, strength, cardio, hiit, yoga, pilates, mobility, multisport, custom.
  - `-s, --step=STEP,...`: Workout step (type:duration[@target:values]).
  - `--unit="km"`: Pace unit: km or mi.
- `gccli workouts copy <id> [<name>]`: Copy a workout.
  - `<id>`: Workout ID.
  - `[<name>]`: Name of the copy. Without it, the copy gets the name of the original.
- `gccli workouts send <id> [flags]`: Send a workout to a device at its next sync. Undo: devices messages cancel.
  - `<id>`: Workout ID.
  - `--device=ID`: Device ID (from 'devices list'). Needed when more than one device can receive the workout.
- `gccli workouts types`: List workout types.
- `gccli workouts sport-types`: List the sport types a workout can have.
- `gccli workouts schedule add <id> <date>`: Add a workout to the schedule.
  - `<id>`: Workout ID.
  - `<date>`: Date to schedule (YYYY-MM-DD, today, yesterday, 3d).
- `gccli workouts schedule list [<date>] [flags]`: List scheduled workouts for a date.
  - `[<date>]`: Date to list (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
  - `--start=DATE`: Start date for range (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date for range (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli workouts schedule remove <id> [flags]`: Remove a scheduled workout from the calendar.
  - `<id>`: Schedule ID (from 'workouts schedule list').
  - `-f, --force`: Skip confirmation prompt.
- `gccli workouts schedule export [flags]`: Export the workout schedule from a start date.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli workouts benchmarks list`: List strength benchmarks. Also runs as `gccli workouts benchmarks`.
- `gccli workouts benchmarks set --weight=WEIGHT --reps=N <key> [flags]`: Set a strength benchmark from a weight and reps.
  - `<key>`: Benchmark key, such as BARBELL_ROW (from 'workouts benchmarks list').
  - `--weight=WEIGHT`: Heaviest weight lifted for the reps, 0.1 to 453.5 kg or 0.1 to 999.9 lbs.
  - `--reps=N`: Number of reps lifted with the weight, 1 to 20.
  - `--unit="kg"`: Unit of the weight: kg or lbs.
- `gccli workouts benchmarks clear <key> [flags]`: Clear a strength benchmark.
  - `<key>`: Benchmark key, such as BARBELL_ROW (from 'workouts benchmarks list').
  - `-f, --force`: Skip confirmation prompt.
- `gccli workouts delete <id> [flags]`: Delete a workout.
  - `<id>`: Workout ID.
  - `-f, --force`: Skip confirmation prompt.

### Courses & Segments

- `gccli courses list`: List courses. Also runs as `gccli courses`.
- `gccli courses favorites`: List favorite courses.
- `gccli courses detail <id>`: View course details.
  - `<id>`: Course ID.
- `gccli courses download <id> [flags]`: Download course file.
  - `<id>`: Course ID.
  - `--format="fit"`: Download format: fit, gpx.
  - `-o, --output=STRING`: Output file path (default: course_{id}.{format}).
- `gccli courses import <file> [flags]`: Import a GPX file as a new course.
  - `<file>`: Path to GPX file.
  - `-n, --name=STRING`: Course name (overrides name from GPX).
  - `-t, --type="cycling"`: Activity type key: cycling, e_bike_fitness, e_bike_mountain, gravel_cycling, hiking, mountain_biking, other, road_biking, running, street_running, trail_running, ultra_run or walking.
  - `-p, --privacy=2`: Course privacy: 1=public, 2=private, 4=group.
- `gccli courses send <course-id> <device-id>`: Send a course to a device.
  - `<course-id>`: Course ID.
  - `<device-id>`: Device ID.
- `gccli courses delete <id> [flags]`: Delete a course.
  - `<id>`: Course ID.
  - `-f, --force`: Skip confirmation prompt.
- `gccli segments list [flags]`: List the segments you created, or your favorites with --favorites. Also runs as `gccli segments [flags]`.
  - `--favorites`: List your favorite segments instead.
  - `-l, --limit=20`: Maximum number of segments to return.
  - `--offset=0`: Number of segments to skip.
- `gccli segments search [flags]`: Search segments by name or map area.
  - `--name=TEXT`: Text in the segment name.
  - `--bbox=MINLAT,MINLON,MAXLAT,MAXLON`: Map area as MINLAT,MINLON,MAXLAT,MAXLON in degrees.
  - `--activity-type=TYPE`: Activity type: cycling or running.
  - `--classification=KEY`: Classification as segments classifications prints it: downhill, hill_climb, hills, other or sprint.
  - `--surface=KEY`: Surface as segments surface-types prints it: dirt, gravel, mixed or paved.
  - `--min-distance=KM`: Minimum distance in kilometers.
  - `--max-distance=KM`: Maximum distance in kilometers.
  - `--min-grade=PCT`: Minimum average grade in percent.
  - `--max-grade=PCT`: Maximum average grade in percent.
  - `-l, --limit=20`: Maximum number of segments to return.
  - `--offset=0`: Number of segments to skip.
- `gccli segments show <uuid>`: Show a segment with its track points.
  - `<uuid>`: Segment UUID.
- `gccli segments leaderboard <uuid> [flags]`: Show the leaderboard of a segment.
  - `<uuid>`: Segment UUID.
  - `--from-rank=1`: Rank to start at. One page has 26 rows.
  - `--scope="all"`: Whose times: all, connections, following or me.
  - `--group=ID`: Show only the members of this group. Goes only with --scope all.
  - `--gender=GENDER`: Gender: female or male.
  - `--age=AGES`: Age group: 0-18, 18-24, 25-34, 35-44, 45-54, 55-64 or 65-120.
- `gccli segments rank <uuid>`: Show your best time and the top three on a segment.
  - `<uuid>`: Segment UUID.
- `gccli segments classifications`: List the segment classifications.
- `gccli segments surface-types`: List the segment surface types.
- `gccli segments create --from-activity=ID --start-km=KM --end-km=KM --name=NAME --activity-type=TYPE [flags]`: Create a segment from a part of an activity track.
  - `--from-activity=ID`: ID of the activity whose track the segment follows.
  - `--start-km=KM`: Start of the segment, in kilometers from the start of the activity.
  - `--end-km=KM`: End of the segment, in kilometers from the start of the activity.
  - `--name=NAME`: Segment name.
  - `--activity-type=TYPE`: Activity type: cycling or running.
  - `--classification="other"`: Classification as segments classifications prints it: downhill, hill_climb, hills, other or sprint.
  - `--surface="paved"`: Surface as segments surface-types prints it: dirt, gravel, mixed or paved.
- `gccli segments favorite <uuid>`: Add a segment to your favorites.
  - `<uuid>`: Segment UUID.
- `gccli segments unfavorite <uuid>`: Remove a segment from your favorites.
  - `<uuid>`: Segment UUID.
- `gccli segments delete <uuid> [flags]`: Delete a segment that you created.
  - `<uuid>`: Segment UUID.
  - `-f, --force`: Skip confirmation prompt.

### Devices

- `gccli devices list`: List registered devices. Also runs as `gccli devices`.
- `gccli devices settings <device-id>`: Show settings for a device.
  - `<device-id>`: Device ID.
- `gccli devices primary`: Show primary training device.
- `gccli devices set-primary-tracker <device-id>`: Set the primary activity tracker (not the 'devices primary' training device).
  - `<device-id>`: Device ID (from 'devices list').
- `gccli devices solar <device-id> [flags]`: Show solar charging data for a device.
  - `<device-id>`: Device ID.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to start date.
- `gccli devices alarms`: Show alarms from all devices.
- `gccli devices last-used`: Show last used device.
- `gccli devices sensors`: List paired sensors.
- `gccli devices backups summary`: Show device backup summary. Also runs as `gccli devices backups`.
- `gccli devices backups list`: List device backups.
- `gccli devices historical`: List historical devices.
- `gccli devices messages list`: List device messages. Also runs as `gccli devices messages`.
- `gccli devices messages cancel <id>`: Cancel a device message, such as a sent workout, before the device syncs.
  - `<id>`: Message ID (from 'devices messages').

### Gear

- `gccli gear list`: List all gear. Also runs as `gccli gear`.
- `gccli gear stats <uuid>`: Show usage statistics for a gear item.
  - `<uuid>`: Gear UUID.
- `gccli gear activities <uuid> [flags]`: Show activities linked to a gear item.
  - `<uuid>`: Gear UUID.
  - `--limit=20`: Maximum number of activities to return.
- `gccli gear defaults`: Show default gear per activity type.
- `gccli gear link <uuid> <activity-id>`: Link gear to an activity.
  - `<uuid>`: Gear UUID.
  - `<activity-id>`: Activity ID.
- `gccli gear unlink <uuid> <activity-id>`: Unlink gear from an activity.
  - `<uuid>`: Gear UUID.
  - `<activity-id>`: Activity ID.
- `gccli gear set-default <uuid> <activity-type>`: Make a gear item a default gear for an activity type.
  - `<uuid>`: Gear UUID, with or without dashes.
  - `<activity-type>`: Activity type key, as activities types prints it (e.g. running, trail_running).
- `gccli gear unset-default <uuid> <activity-type>`: Stop a gear item being a default gear for an activity type.
  - `<uuid>`: Gear UUID, with or without dashes.
  - `<activity-type>`: Activity type key, as activities types prints it (e.g. running, trail_running).
- `gccli gear types`: List gear types.
- `gccli gear models <search>`: Search the Garmin list of gear brands and models.
  - `<search>`: Brand or model words, such as "brooks ghost".
- `gccli gear add --brand=FREE-TEXT --model=FREE-TEXT <name> [flags]`: Add a gear item.
  - `<name>`: Gear name.
  - `--type="shoes"`: Gear type. One of: shoes, bike, other.
  - `--brand=FREE-TEXT`: Brand.
  - `--model=FREE-TEXT`: Model.
  - `--notes=STRING`: Notes (1 to 2000 characters). Defaults to the name, because Garmin requires notes.
  - `--first-use="today"`: Date of first use (YYYY-MM-DD, today, yesterday, 3d).
- `gccli gear update <uuid> [flags]`: Change the name, brand, model or notes of a gear item.
  - `<uuid>`: Gear UUID, with or without dashes.
  - `--name=STRING`: New name.
  - `--brand=STRING`: New brand.
  - `--model=STRING`: New model.
  - `--notes=STRING`: New notes (1 to 2000 characters).
- `gccli gear retire <uuid>`: Retire a gear item.
  - `<uuid>`: Gear UUID, with or without dashes.
- `gccli gear unretire <uuid>`: Make a retired gear item active again.
  - `<uuid>`: Gear UUID, with or without dashes.
- `gccli gear delete <uuid> [flags]`: Delete a gear item.
  - `<uuid>`: Gear UUID, with or without dashes.
  - `-f, --force`: Skip confirmation prompt.
- `gccli gear collections list`: List gear collections. Also runs as `gccli gear collections`.
- `gccli gear collections create <name> [flags]`: Create a gear collection.
  - `<name>`: Collection name.
  - `--first-use="today"`: Date of first use (YYYY-MM-DD, today, yesterday, 3d).
- `gccli gear collections rename <uuid> <name>`: Rename a gear collection.
  - `<uuid>`: Collection UUID.
  - `<name>`: New name.
- `gccli gear collections add <uuid> <gear-uuid>`: Add a gear item to a gear collection.
  - `<uuid>`: Collection UUID.
  - `<gear-uuid>`: Gear UUID, with or without dashes.
- `gccli gear collections remove <uuid> <gear-uuid> [flags]`: Remove a gear item from a gear collection. The gear item stays.
  - `<uuid>`: Collection UUID.
  - `<gear-uuid>`: Gear UUID, with or without dashes.
  - `-f, --force`: Skip confirmation prompt.
- `gccli gear collections delete <uuid> [flags]`: Delete a gear collection. The gear items stay.
  - `<uuid>`: Collection UUID.
  - `-f, --force`: Skip confirmation prompt.

### Goals, Badges & Challenges

- `gccli goals list [flags]`: List goals. Also runs as `gccli goals [flags]`.
  - `--status="active"`: Goal status: active, future, past.
- `gccli goals add --period=STRING <name> [flags]`: Add an activity goal.
  - `<name>`: Goal name, 1 to 49 characters.
  - `--period=STRING`: Goal period: seven-days, thirty-days, one-month or annual.
  - `--activity-type="all"`: Activity type key, as activities types prints it (e.g. running, cycling), or all.
  - `--start=DATE`: First day of the goal (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today, and to the first day of this month for a one-month goal.
  - `--distance=DISTANCE`: Distance to cover (e.g. 50km, 30mi, 5000m).
  - `--duration=DURATION`: Time to spend (e.g. 10h, 90m).
  - `--calories=KCAL`: Kilocalories to burn.
  - `--activities=N`: Number of activities to do.
- `gccli goals update <id> [flags]`: Change an activity goal.
  - `<id>`: Goal ID, as goals list prints it.
  - `--name=FREE-TEXT`: New name, 1 to 49 characters.
  - `--period=PERIOD`: New period: seven-days, thirty-days, one-month or annual.
  - `--activity-type=FREE-TEXT`: New activity type key, as activities types prints it (e.g. running, cycling), or all.
  - `--start=DATE`: New first day of the goal (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--distance=DISTANCE`: Distance to cover (e.g. 50km, 30mi, 5000m).
  - `--duration=DURATION`: Time to spend (e.g. 10h, 90m).
  - `--calories=KCAL`: Kilocalories to burn.
  - `--activities=N`: Number of activities to do.
- `gccli goals delete <id> [flags]`: Delete an activity goal.
  - `<id>`: Goal ID, as goals list prints it.
  - `-f, --force`: Skip confirmation prompt.
- `gccli goals effective <type> [flags]`: Show the daily goal of one type that applies on a date.
  - `<type>`: Goal type: floors-climbed, intensity-minutes, hydration, steps or weight-grams.
  - `--date=DATE`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli goals weight view [flags]`: Show the effective weight goal for a date range. Also runs as `gccli goals weight [flags]`.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli goals weight set <weight> [flags]`: Set the weight goal.
  - `<weight>`: Goal weight.
  - `--unit="kg"`: Unit of the weight: kg or lbs.
  - `--date="today"`: First day of the goal (YYYY-MM-DD, today, yesterday, 3d). The goal stays for the days after it.
- `gccli goals weight delete [flags]`: Delete the current weight goal. An earlier weight goal then applies again.
  - `-f, --force`: Skip confirmation prompt.
- `gccli goals set <type> <value> [flags]`: Set the daily goal for floors climbed, intensity minutes or hydration.
  - `<type>`: Goal type: floors-climbed, intensity-minutes or hydration.
  - `<value>`: Daily goal: floors for floors-climbed, minutes for intensity-minutes, milliliters for hydration.
  - `--date="today"`: First day of the goal (YYYY-MM-DD, today, yesterday, 3d). The goal stays for the days after it.
- `gccli goals steps set <steps> [flags]`: Set a fixed daily steps goal.
  - `<steps>`: Steps per day.
  - `--date="today"`: First day of the goal (YYYY-MM-DD, today, yesterday, 3d). The goal stays for the days after it.
- `gccli goals steps auto [flags]`: Set Garmin's automatic steps goal.
  - `--date="today"`: First day of the goal (YYYY-MM-DD, today, yesterday, 3d). The goal stays for the days after it.
- `gccli badges earned`: Show earned badges. Also runs as `gccli badges`.
- `gccli badges available`: Show available badges.
- `gccli badges in-progress`: Show badges in progress.
- `gccli badges detail <id>`: Show details of a badge.
  - `<id>`: Badge ID.
- `gccli badges compare <display-name>`: Compare your badges with a connection.
  - `<display-name>`: Display name of the connection.
- `gccli badges leaderboard`: Show the badge points leaderboard of your connections.
- `gccli challenges list`: Show current challenges. Also runs as `gccli challenges`.
- `gccli challenges badge`: Show joined badge challenges that are not completed.
- `gccli challenges available`: Show available badge challenges.
- `gccli challenges completed`: Show completed badge challenges.
- `gccli challenges virtual available`: Show available virtual challenges.
- `gccli challenges virtual completed`: Show completed virtual challenges.
- `gccli challenges virtual in-progress`: Show virtual challenges in progress.
- `gccli challenges show <uuid>`: Show a badge or virtual challenge.
  - `<uuid>`: Challenge UUID, as the list commands print it.
- `gccli challenges join <uuid>`: Join a badge or virtual challenge.
  - `<uuid>`: Challenge UUID, as the list commands print it.
- `gccli challenges leave <uuid>`: Leave a badge or virtual challenge.
  - `<uuid>`: Challenge UUID, as the list commands print it.
- `gccli challenges adhoc list`: Show your ad-hoc challenges that are not finished. Also runs as `gccli challenges adhoc`.
- `gccli challenges adhoc history`: Show your finished ad-hoc challenges.
- `gccli challenges adhoc show <uuid>`: Show an ad-hoc challenge.
  - `<uuid>`: Challenge UUID, as the list commands print it.
- `gccli challenges adhoc create --type=STRING <name> [flags]`: Create an ad-hoc challenge and invite people who follow you and whom you follow.
  - `<name>`: Challenge name.
  - `--type=STRING`: Activity: walking, running, cycling, swimming, steps, pushes or pushing-distance.
  - `--duration="week"`: Length: day, weekend (Saturday and Sunday), week (7 days), month (to the end of the month) or custom (to --end).
  - `--start=DATE`: First day (YYYY-MM-DD, today, +3d). Default: today, the Saturday of this week for weekend, the first day of this month for month.
  - `--end=DATE`: Last day. Only with --duration custom, at most 60 days after the first day.
  - `--description=FREE-TEXT`: Description. Defaults to the name, because Garmin requires one.
  - `--invite=DISPLAY-NAME`: Display name of a person who follows you and whom you follow. Repeat it to invite more people.
- `gccli challenges adhoc invites <uuid>`: List the open invites of an ad-hoc challenge with their adHocChallengeInviteId.
  - `<uuid>`: Challenge UUID, as the list commands print it.
- `gccli challenges adhoc invite <uuid> <display-name> ...`: Invite people who follow you and whom you follow to an ad-hoc challenge.
  - `<uuid>`: Challenge UUID, as the list commands print it.
  - `<display-name> ...`: Display name of a person who follows you and whom you follow.
- `gccli challenges adhoc accept <invite-id>`: Accept an invite to an ad-hoc challenge.
  - `<invite-id>`: Invite ID (adHocChallengeInviteId).
- `gccli challenges adhoc decline <invite-id>`: Decline an invite to an ad-hoc challenge.
  - `<invite-id>`: Invite ID (adHocChallengeInviteId).
- `gccli challenges adhoc leave <uuid> [flags]`: Leave an ad-hoc challenge. Only a new invite gets you back in.
  - `<uuid>`: Challenge UUID, as the list commands print it.
  - `-f, --force`: Skip confirmation prompt.
- `gccli challenges adhoc delete <uuid> [flags]`: Delete an ad-hoc challenge that you created.
  - `<uuid>`: Challenge UUID, as the list commands print it.
  - `-f, --force`: Skip confirmation prompt.
- `gccli records list`: View personal records. Also runs as `gccli records`.
- `gccli records types`: List personal record types.
- `gccli records activity <id>`: List the personal records set in an activity.
  - `<id>`: Activity ID.
- `gccli records candidates`: List the personal record candidates that wait for your review.
- `gccli records hide <type-id>`: Hide a personal record type on the records page.
  - `<type-id>`: Record type ID, as records types prints it.
- `gccli records unhide <type-id>`: Show a hidden personal record type on the records page again.
  - `<type-id>`: Record type ID, as records types prints it.
- `gccli records delete <id> [flags]`: Delete a personal record. This cannot be undone.
  - `<id>`: Personal record ID, as records list prints it.
  - `-f, --force`: Skip confirmation prompt.

### Events

- `gccli events list [flags]`: List calendar events. Also runs as `gccli events [flags]`.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `-l, --limit=20`: Maximum number of events to return.
  - `--offset=0`: Number of events to skip.
  - `--sort="eventDate_asc"`: Sort order. One of: eventDate_asc, eventDate_desc.
- `gccli events upcoming`: List upcoming events.
- `gccli events add --name=FREE-TEXT --date=DATE --type=STRING [flags]`: Add a calendar event.
  - `-n, --name=FREE-TEXT`: Event name.
  - `-d, --date=DATE`: Event date (YYYY-MM-DD, today, +30d).
  - `-t, --type=STRING`: Event type. One of: running, trail_running, cycling, gravel_cycling, mountain_biking, swimming, triathlon, multi_sport, hiking, walking, fitness_equipment, motorcycling, winter_sport, other.
  - `--race`: Mark as a race.
  - `--location=STRING`: Event location.
  - `--time=HH:MM`: Start time (HH:MM).
  - `--timezone=STRING`: Time zone ID (e.g. Europe/Berlin).
  - `--distance=DISTANCE`: Target distance (e.g. 10km, 26.2mi, 400m).
  - `--goal=GOAL`: Time goal duration (e.g. 50m, 1h30m, 2400s).
  - `--primary`: Set as primary training event.
  - `--training`: Set as supporting training event.
  - `--private`: Make event private.
  - `--note=STRING`: Event note.
  - `--url=STRING`: Event URL.
- `gccli events delete <id> [flags]`: Delete a calendar event.
  - `<id>`: Event ID.
  - `-f, --force`: Skip confirmation prompt.

### Calendar

- `gccli calendar month [<year>] [<month>]`: List the items of a calendar month, such as activities and notes.
  - `[<year>]`: Year (e.g. 2026). Give it with MONTH. Without both, the command shows this month.
  - `[<month>]`: Month (1-12).
- `gccli calendar notes add <date> <name> [flags]`: Add a note to a calendar day.
  - `<date>`: Date of the note (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `<name>`: Note name.
  - `--content=STRING`: Note text.
- `gccli calendar notes update <id> [flags]`: Change the name or text of a calendar note.
  - `<id>`: Note ID.
  - `--name=STRING`: New name.
  - `--content=STRING`: New text. An empty text clears it.
- `gccli calendar notes move <id> <date>`: Move a calendar note to another day.
  - `<id>`: Note ID.
  - `<date>`: New date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli calendar notes delete <id> [flags]`: Delete a calendar note.
  - `<id>`: Note ID.
  - `-f, --force`: Skip confirmation prompt.
- `gccli calendar export show`: Show the calendar feed URL.
- `gccli calendar export enable`: Turn on the calendar feed and show its URL.
- `gccli calendar export disable [flags]`: Turn off the calendar feed. Its URL then gives an empty answer.
  - `-f, --force`: Skip confirmation prompt.
- `gccli calendar preferences view`: Show the calendar preferences. Also runs as `gccli calendar preferences`.
- `gccli calendar preferences set <key> <value>`: Turn a calendar preference on or off.
  - `<key>`: Preference key, such as showWeeklyTotal. Run gccli calendar preferences to see the keys.
  - `<value>`: true turns the preference on, false turns it off.

### Social

- `gccli social connections [flags]`: List your connections.
  - `-l, --limit=20`: Maximum number of connections to return.
  - `--offset=0`: Number of connections to skip.
- `gccli social followers [<display-name>]`: List the followers of a user.
  - `[<display-name>]`: Display name of the user. Defaults to your display name.
- `gccli social following [<display-name>]`: List the users that a user follows.
  - `[<display-name>]`: Display name of the user. Defaults to your display name.
- `gccli social requests`: List pending connection requests.
- `gccli social feed`: Show your newsfeed.
- `gccli social follow <display-name>`: Follow a user. A user with a private profile gets a follow request.
  - `<display-name>`: Display name of the user.
- `gccli social unfollow <display-name> [flags]`: Stop following a user.
  - `<display-name>`: Display name of the user.
  - `-f, --force`: Skip confirmation prompt.
- `gccli social follow-requests`: List the users who asked to follow you.
- `gccli social accept <display-name>`: Accept the follow request of a user.
  - `<display-name>`: Display name of the user, as social follow-requests lists it.
- `gccli social decline <display-name>`: Decline the follow request of a user.
  - `<display-name>`: Display name of the user, as social follow-requests lists it.
- `gccli social remove-follower <display-name> [flags]`: Remove a user from your followers.
  - `<display-name>`: Display name of the user.
  - `-f, --force`: Skip confirmation prompt.
- `gccli social mute <display-name>`: Mute a user that you follow.
  - `<display-name>`: Display name of the user.
- `gccli social unmute <display-name>`: Unmute a user.
  - `<display-name>`: Display name of the user.
- `gccli social muted`: List the users that you muted.
- `gccli social search <query> [flags]`: Search Garmin Connect users by name.
  - `<query>`: Name to search for, at least 2 characters.
  - `-l, --limit=20`: Maximum number of users to return.
  - `--offset=0`: Number of users to skip.
- `gccli groups list`: List your groups. Also runs as `gccli groups`.
- `gccli notifications list [flags]`: List notifications. Also runs as `gccli notifications [flags]`.
  - `--limit=50`: Maximum number of notifications to return.
  - `--cursor=STRING`: Start after this cursor. Use pagination.nextCursor from the JSON output of the last page.
- `gccli notifications unread`: Show whether there are unread notifications.
- `gccli notifications dismiss <id>`: Dismiss a notification. Use 'notifications undismiss' to bring it back.
  - `<id>`: Notification ID (from 'notifications list').
- `gccli notifications undismiss <id>`: Bring back a dismissed notification.
  - `<id>`: Notification ID (from 'notifications list').
- `gccli notifications preferences view`: Show notification preferences. Also runs as `gccli notifications preferences`.
- `gccli notifications preferences set <key> <value>`: Turn a notification preference on or off.
  - `<key>`: Preference key, such as allowEmailWeeklySummary. Run gccli notifications preferences to see the keys.
  - `<value>`: true turns the preference on, false turns it off.

### Insights

- `gccli insights options [flags]`: List the genders, ages, dates, activities and fields for insights compare. Also runs as `gccli insights [flags]`.
  - `--locale-unit="METRIC"`: Units of the field names: METRIC or STATUTE.
- `gccli insights compare --activity=STRING --field=STRING --age=FREE-TEXT --year=INT [flags]`: Compare a metric with your age group. Needs Insights on in Garmin Connect.
  - `--activity=STRING`: Activity as insights options prints it: STEPS, SLEEP, FLOORS, RUNNING, CYCLING, LAP_SWIMMING or OPEN_WATER_SWIMMING.
  - `--field=STRING`: Field of the activity as insights options prints it: AVG_SPEED, VO2_MAX_VALUE, AVG_POWER, WEEKLY_AVG_DISTANCE, WEEKLY_AVG_DURATION, STEPS, SLEEP, FLOORS or COUNT.
  - `--age=FREE-TEXT`: Age group as insights options prints it (e.g. 40-44, 75+).
  - `--gender="ALL"`: Gender: ALL, M or F.
  - `--year=INT`: Year (e.g. 2025).
  - `--month=MONTH`: Month (1-12). Omit it for the whole year.
  - `--locale-unit="METRIC"`: Units of the answer: METRIC or STATUTE.
- `gccli insights preferences`: Show your saved Insights comparisons.

### Golf

- `gccli golf scorecards [flags]`: List your golf scorecards. Also runs as `gccli golf [flags]`.
  - `-l, --limit=20`: Maximum number of scorecards to return.
  - `--offset=0`: Number of scorecards to skip.
- `gccli golf courses [flags]`: List the golf courses you played.
  - `-l, --limit=20`: Maximum number of courses to return.
  - `--offset=0`: Number of courses to skip.
- `gccli golf stats`: Show your golf statistics.
- `gccli golf shot-stats <kind>`: Show your golf shot statistics of one shot type.
  - `<kind>`: Shot type: drive, approach, chip or putt.
- `gccli golf clubs`: List the golf clubs in your bag.
- `gccli golf club-types`: List the golf club types with their loft and lie angles.
- `gccli golf leaderboards [flags]`: List the golf leaderboards you played in.
  - `-l, --limit=20`: Maximum number of leaderboards to return.
  - `--offset=0`: Number of leaderboards to skip.

### Hydration, Nutrition, Profile, Training, Wellness & Utilities

- `gccli hydration view [<date>]`: Show hydration data for a date. Also runs as `gccli hydration [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli hydration daily --start=DATE --end=DATE`: Show daily hydration statistics for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli hydration weekly [flags]`: Show weekly hydration statistics.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--weeks=4`: Number of weeks.
- `gccli hydration add <amount> [flags]`: Add a hydration log entry.
  - `<amount>`: Amount of water in milliliters.
  - `--date=DATE`: Date to log for (YYYY-MM-DD). Defaults to today.
- `gccli nutrition food-log [<date>]`: Show daily food log for a date. Also runs as `gccli nutrition [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli nutrition meals [<date>]`: Show meal summaries for a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli nutrition settings [<date>]`: Show nutrition settings for a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli nutrition search <query> [flags]`: Search the Garmin food database. Needs Garmin Connect+.
  - `<query>`: Text to search for, such as a food name.
  - `-l, --limit=20`: Maximum number of foods to return.
  - `--offset=0`: Number of foods to skip.
- `gccli nutrition recent [<date>] [flags]`: List the foods you logged recently. Needs Garmin Connect+.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
  - `-l, --limit=20`: Maximum number of foods to return.
  - `--offset=0`: Number of foods to skip.
- `gccli nutrition add --query=TEXT --meal=MEAL <date> <food-id> [flags]`: Log a food of the Garmin food database in a meal. Needs Garmin Connect+.
  - `<date>`: Date of the meal (YYYY-MM-DD, today, yesterday, 3d).
  - `<food-id>`: Food ID, as nutrition search prints it.
  - `-q, --query=TEXT`: Search text that finds the food, as you gave it to nutrition search. Garmin has no read of one food by ID.
  - `-m, --meal=MEAL`: Meal: breakfast, lunch, dinner, snacks, or a custom meal you enabled.
  - `--serving=UNIT`: Serving unit, such as g or oz, or serving ID, as nutrition search --json lists them under nutritionContents. Defaults to the first serving of the food.
  - `--servings=1`: Number of servings. A serving can hold more than one unit, such as 100 g.
  - `--time=HH:MM`: Meal time (HH:MM). Defaults to the start time of the meal, or to the current time for a meal without one, such as snacks.
- `gccli nutrition quick-add --meal=MEAL --calories=KCAL <date> [flags]`: Log calories and macros in a meal without a food. Needs Garmin Connect+.
  - `<date>`: Date of the meal (YYYY-MM-DD, today, yesterday, 3d).
  - `-m, --meal=MEAL`: Meal: breakfast, lunch, dinner, snacks, or a custom meal you enabled.
  - `--calories=KCAL`: Calories (kcal).
  - `--carbs=GRAMS`: Carbohydrates in grams.
  - `--protein=GRAMS`: Protein in grams.
  - `--fat=GRAMS`: Fat in grams.
  - `--name=NAME`: Name of the entry, such as Coffee with milk.
  - `--time=HH:MM`: Meal time (HH:MM). Defaults to the start time of the meal, or to the current time for a meal without one, such as snacks.
- `gccli nutrition delete <date> <log-id> ... [flags]`: Delete food log entries of a day. Needs Garmin Connect+.
  - `<date>`: Date of the entries (YYYY-MM-DD, today, yesterday, 3d).
  - `<log-id> ...`: Log ID of an entry, as the add commands print it or nutrition food-log --json lists it.
  - `-f, --force`: Skip confirmation prompt.
- `gccli nutrition favorites list [flags]`: List your favorite foods. Needs Garmin Connect+. Also runs as `gccli nutrition favorites [flags]`.
  - `-l, --limit=20`: Maximum number of foods to return.
  - `--offset=0`: Number of foods to skip.
- `gccli nutrition favorites add --query=TEXT <food-id> [flags]`: Add a food to your favorite foods. Needs Garmin Connect+.
  - `<food-id>`: Food ID, as nutrition search prints it.
  - `-q, --query=TEXT`: Search text that finds the food, as you gave it to nutrition search. Garmin has no read of one food by ID.
  - `--serving=UNIT`: Serving unit, such as g or oz, or serving ID, as nutrition search --json lists them under nutritionContents. Defaults to the first serving of the food.
  - `--servings=1`: Number of servings. A serving can hold more than one unit, such as 100 g.
- `gccli nutrition favorites remove <food-id>`: Remove a food from your favorite foods. Needs Garmin Connect+.
  - `<food-id>`: Food ID, as nutrition favorites prints it.
- `gccli profile view`: Show user profile. Also runs as `gccli profile`.
- `gccli profile settings`: Show user settings.
- `gccli profile privacy view`: Show privacy settings. Also runs as `gccli profile privacy`.
- `gccli profile privacy set [flags]`: Change who can see your profile, activities, wellness data and badges.
  - `--profile=LEVEL`: Who can see your profile. One of: public, private, connections, groups.
  - `--activities=LEVEL`: Who can see your activities by default. Past activities keep their privacy; change one with activity privacy. One of: public, private, connections, groups.
  - `--wellness=LEVEL`: Who can see your wellness data, such as steps. One of: public, private, connections, groups.
  - `--badges=LEVEL`: Who can see your badges. One of: public, private, connections, groups.
- `gccli profile privacy-zones list`: List privacy zones. Also runs as `gccli profile privacy-zones`.
- `gccli profile privacy-zones add --lat=DEGREES --lon=DEGREES --radius=METERS`: Add a privacy zone around a point.
  - `--lat=DEGREES`: Latitude of the center, -90 to 90. Give a negative value as --lat=-33.9.
  - `--lon=DEGREES`: Longitude of the center, -180 to 180. Give a negative value as --lon=-30.
  - `--radius=METERS`: Radius in meters, 500 to 1500.
- `gccli profile privacy-zones delete <id> [flags]`: Delete a privacy zone.
  - `<id>`: Privacy zone ID.
  - `-f, --force`: Skip confirmation prompt.
- `gccli profile hr-zones`: Show heart rate zones.
- `gccli profile power-zones list`: Show power zones for all sports. Also runs as `gccli profile power-zones`.
- `gccli profile power-zones sport <sport>`: Show power zones for one sport.
  - `<sport>`: Sport: cycling, running, rowing or cross-country-skiing.
- `gccli training plans [flags]`: List training plans. Also runs as `gccli training [flags]`.
  - `--locale="en"`: Locale for training plans (e.g. en, de, fr).
- `gccli training plan <id>`: Show a specific training plan.
  - `<id>`: Training plan ID.
- `gccli training active [flags]`: Show active training plans.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli training search [flags]`: Search the Garmin catalog of training plans.
  - `--sport=STRING`: Sport as Garmin prints it in the SPORT column, such as Running.
  - `--goal=STRING`: Goal as Garmin prints it in the GOAL column, such as 5k, 10k, HalfMarathon or Marathon.
  - `--level=STRING`: Level as Garmin prints it in the LEVEL column, such as Beginner, Intermediate or Advanced.
  - `--target=STRING`: Workout target as Garmin prints it in the TARGET column, such as HeartRate or Time.
  - `--locale="en"`: Locale for plan names (e.g. en, de, fr).
- `gccli training schedule <id> [flags]`: List the workouts and notes of a training plan by week and day.
  - `<id>`: Training plan ID.
  - `--locale="en"`: Locale for workout and note names (e.g. en, de, fr).
- `gccli training join <id> [flags]`: Join a catalog training plan and schedule its workouts in the calendar.
  - `<id>`: Catalog training plan ID, as training search prints it.
  - `--start=DATE`: Day of the first workout (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli training leave <id> [flags]`: Leave a joined training plan and remove its workouts from the calendar.
  - `<id>`: ID of the joined training plan, as training join or training active prints it.
  - `-f, --force`: Skip confirmation prompt.
- `gccli training task move <plan-id> <task-id> <date>`: Move a workout of a joined training plan to another day.
  - `<plan-id>`: ID of the joined training plan.
  - `<task-id>`: Task ID of the workout, as training schedule prints it.
  - `<date>`: New date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli training task delete <plan-id> <task-id> [flags]`: Delete a workout of a joined training plan. Garmin puts a rest day in its place.
  - `<plan-id>`: ID of the joined training plan.
  - `<task-id>`: Task ID of the workout, as training schedule prints it.
  - `-f, --force`: Skip confirmation prompt.
- `gccli training coach-workouts`: List adaptive coaching workout types.
- `gccli wellness menstrual-cycle [<date>]`: Show the menstrual cycle day view of a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli wellness menstrual-summary [<date>]`: Show the menstrual cycle summary of a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli wellness menstrual-calendar --start=DATE --end=DATE`: Show the menstrual cycle calendar for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli wellness menstrual-log view [<date>]`: Show the menstrual daily log of a date. Also runs as `gccli wellness menstrual-log [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli wellness menstrual-log set <date> [flags]`: Change the menstrual daily log of a date. Other fields keep their values.
  - `<date>`: Date of the log (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--flow=FLOW`: Flow: light, medium or heavy. An empty value clears it.
  - `--symptoms=SYMPTOM,...`: Comma-separated symptoms: acne, backache, bloating, body-aches, constipation, cramps, cravings, diarrhea, dizziness, fatigue, headache, insomnia, nausea, tender-breasts. An empty value clears them.
  - `--moods=MOOD,...`: Comma-separated moods: energetic, emotional, fine, frustrated, happy, irritable, mood-swings, motivated, overwhelmed, sad. An empty value clears them.
  - `--discharge=DISCHARGE,...`: Comma-separated discharge: creamy, egg-white, no-discharge, spotting, sticky, unusual. no-discharge takes no other value. An empty value clears it.
  - `--sex-drive=LEVEL`: Sex drive: average, high or low. An empty value clears it.
  - `--sexual-activity=KIND`: Sexual activity: protected or unprotected. An empty value clears it.
  - `--notes=TEXT`: Notes. An empty value clears them.
  - `--ovulation`: Mark the day as the ovulation day. --no-ovulation removes the mark.
- `gccli wellness menstrual-log delete <date> [flags]`: Delete the menstrual daily log of a date.
  - `<date>`: Date of the log (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `-f, --force`: Skip confirmation prompt.
- `gccli wellness pregnancy-summary`: Show pregnancy summary.
- `gccli wellness jet-lag list`: List jet lag trips. Also runs as `gccli wellness jet-lag`.
- `gccli wellness jet-lag add --from=PLACE --to=PLACE --depart=TIME --arrive=TIME [flags]`: Add a jet lag trip of a flight and an optional return flight.
  - `--from=PLACE`: Place of departure, such as "Frankfurt, Germany".
  - `--to=PLACE`: Place of arrival, such as "New York, USA".
  - `--depart=TIME`: Local departure time with the UTC offset of --from, such as 2026-11-02T10:00+01:00.
  - `--arrive=TIME`: Local arrival time with the UTC offset of --to, such as 2026-11-02T13:00-05:00.
  - `--return-depart=TIME`: Local departure time of the return flight from --to, with its UTC offset.
  - `--return-arrive=TIME`: Local arrival time of the return flight at --from, with its UTC offset.
- `gccli wellness jet-lag update <id> [flags]`: Change the places or times of a jet lag trip.
  - `<id>`: Trip ID.
  - `--from=PLACE`: New place of departure. The return flight arrives there.
  - `--to=PLACE`: New place of arrival. The return flight departs there.
  - `--depart=TIME`: New local departure time with the UTC offset, such as 2026-11-02T10:00+01:00.
  - `--arrive=TIME`: New local arrival time with the UTC offset.
  - `--return-depart=TIME`: New local departure time of the return flight, with the UTC offset.
  - `--return-arrive=TIME`: New local arrival time of the return flight, with the UTC offset.
- `gccli wellness jet-lag delete <id> [flags]`: Delete a jet lag trip.
  - `<id>`: Trip ID.
  - `-f, --force`: Skip confirmation prompt.
- `gccli wellness jet-lag status [<date>]`: Show jet lag status for a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli reload [<date>]`: Request data reload for a date.
  - `[<date>]`: Date to reload (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli completion <shell>`: Generate shell completion script.
  - `<shell>`: Shell type (bash, zsh, fish, powershell).
<!-- gccli:commands:end -->

## Output Formats

### Table (default)

Human-readable aligned columns:

```bash
$ gccli activities list --limit 3
ID         DATE        TYPE     NAME             DISTANCE  DURATION  CALORIES
123456789  2024-06-15  running  Morning Run      10.02 km  52:31     712
123456780  2024-06-14  walking  Evening Walk     3.10 km   38:05     180
123456771  2024-06-13  cycling  Cycling Workout  42.50 km  1:25:10   950
```

### JSON

Machine-readable output for scripting and automation:

```bash
$ gccli --json activities list --limit 3
[
  {
    "activityId": 123456789,
    "activityName": "Morning Run",
    "activityType": {"typeKey": "running"},
    "startTimeLocal": "2024-06-15 07:30:00"
  },
  ...
]
```

Data goes to stdout, errors and progress to stderr for clean piping:

```bash
gccli --json activities list --limit 100 | jq '.[] | select(.activityType.typeKey == "running")'
```

### Plain (TSV)

Tab-separated values with no header row. Add `--header` to print the column names first. A tab, newline or backslash in a value is escaped (see [Output](#output)), so each record is one line. Only the commands listed under [Output](#output) print TSV. Most other commands print JSON, also with `--plain`.

```bash
$ gccli --plain activities list --limit 3
123456789	2024-06-15	running	Morning Run	10.02 km	52:31	712
123456780	2024-06-14	walking	Evening Walk	3.10 km	38:05	180
123456771	2024-06-13	cycling	Cycling Workout	42.50 km	1:25:10	950
```

```bash
$ gccli --plain --header courses list
ID	NAME	TYPE	DISTANCE	CREATED
111	Morning Loop	running	5.10 mi	2026-09-01T10:00:00.0
112	Hill Repeats	cycling	26.10 mi	2026-09-20T10:00:00.0
113	Tab\there\nNewline\\back\rCR	hiking	0.62 mi	2026-09-21T10:00:00.0
```

## Examples

### Export all runs from the last month

```bash
gccli --json activities search --start 2024-05-15 --end 2024-06-15 --limit 100 | \
  jq -r '.[] | select(.activityType.typeKey == "running") | .activityId' | \
  while read id; do
    gccli activity download "$id" --format gpx --output "run_${id}.gpx"
  done
```

### Daily health check script

```bash
gccli --json health summary | jq '{
  steps: .totalSteps,
  calories: .totalKilocalories,
  distance_km: (.totalDistanceMeters / 1000),
  active_minutes: .moderateIntensityMinutes + .vigorousIntensityMinutes
}'
```

### Track weight over time

```bash
gccli --json body weigh-ins --start 2024-01-01 --end 2024-06-30 | \
  jq -r '.dailyWeightSummaries[] | [.summaryDate, .weight.value] | @tsv'
```

### Download the latest activity

```bash
latest=$(gccli --json activities list --limit 1 | jq -r '.[0].activityId')
gccli activity download "$latest" --format fit
```

### Compare your VO2 max with your age group

`insights options` lists the age groups, dates, activities and fields. Copy the values from it as they are.

```bash
gccli insights options | jq '{ages, activities: [.activityMapping[].activity]}'
gccli insights compare --activity RUNNING --field VO2_MAX_VALUE --age 40-44 --gender M --year 2025 | \
  jq '.chart.options.percentile'
```

### Name the clubs in your golf bag

`golf clubs` gives each club a `clubTypeId`. `golf club-types` gives the name of each type ID as `value`.

```bash
types=$(gccli golf club-types)
gccli golf clubs | jq --argjson types "$types" \
  'map(select(.retired | not) | . as $club | {id, type: ($types[] | select(.value == $club.clubTypeId) | .name)})'
```

### Find the segments near you and see who is fastest

`segments search` needs `--name` or `--bbox`. The box is `MINLAT,MINLON,MAXLAT,MAXLON` in degrees.

```bash
gccli segments search --bbox 52.45,13.30,52.55,13.45 --activity-type running --classification hill_climb
gccli segments leaderboard <uuid> --scope connections
gccli --json segments rank <uuid> | jq '{best: .bestElapsedTime, leader: .leaderFullName}'
```

`segments create` takes the part of an activity track from `--start-km` to `--end-km`. Garmin refuses a segment shorter than 0.08 km.

```bash
gccli segments create --from-activity <activity-id> --start-km 1.2 --end-km 2.0 \
  --name "Park Hill" --activity-type running --classification hill_climb
gccli segments favorite <uuid>
```

### Monthly running distance

`health progress metrics` lists the metric names of each activity type. Distances are in meters and durations in seconds.

```bash
gccli health progress --start 2026-01-01 --end 2026-06-30 --aggregation monthly \
  --metric distance --activity-type running | \
  jq -r '.[] | [.date, .countOfActivities, (.stats.all.distance.sum // 0) / 1000] | @tsv'
```

## Global Flags

All commands support these flags:

| Flag | Description |
| --- | --- |
| `--account <email>` | Account to use (overrides `GCCLI_ACCOUNT` and config default) |
| `--json`, `-j` | Output JSON to stdout |
| `--plain` | Output TSV to stdout (only some commands, see [Output](#output)) |
| `--header` | With `--plain`, print the column names as the first line |
| `--color <mode>` | Color mode: `auto`, `always`, `never` (default: `auto`) |
| `--units <system>` | Unit system for table and plain output: `auto`, `metric`, `statute_us`, `statute_uk` (default: `auto`, from the Garmin profile) |
| `--version` | Print version information |
| `--help` | Show help for any command |

## Date Format Reference

Many commands accept flexible date formats. Every `--start` and `--end` flag accepts all of them, and so does the optional `[<date>]` argument of the daily views, such as `health sleep`, `health steps`, `body daily-weigh-ins`, `hydration`, `nutrition food-log` and `reload`. A bad or empty date in one of these flags or arguments is a usage error (exit code 2), before the login.

| Format | Example | Description |
| --- | --- | --- |
| `YYYY-MM-DD` | `2024-06-15` | Explicit date |
| `today` | | Today's date |
| `yesterday` | | Yesterday's date |
| `Nd` | `3d`, `7d` | N days ago |
| `+Nd` | `+30d` | N days ahead |

## Development

After cloning, install tools:

```bash
make tools
```

Build, test, and lint:

```bash
make build          # Build to ./bin/gccli
make test           # Unit tests with race detector
make lint           # golangci-lint
make fmt            # Format with goimports + gofumpt
make ci             # Full CI gate (fmt-check + lint + test)
```

Run individual tests:

```bash
go test -run TestFuncName ./internal/pkg/...
```

### E2E Tests (Live Garmin API)

End-to-end tests hit the real Garmin Connect API. They require a `.env` file with test credentials:

```bash
cp .env.example .env
# Edit .env with your test account credentials
make test-e2e
```

**Warning:** E2E tests create and delete activities/workouts. Use a dedicated test account.

## Architecture

Platforms: macOS (amd64, arm64), Linux (amd64, arm64). macOS builds with CGO enabled for Keychain access.

## License

MIT

## Links

- [Garmin Connect](https://connect.garmin.com)

---

<sub>Not affiliated with Garmin. Garmin and Garmin Connect are trademarks of Garmin Ltd. or its subsidiaries.</sub>

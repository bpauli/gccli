# gccli — Garmin Connect in your terminal.

[![ci](https://github.com/bpauli/gccli/actions/workflows/ci.yml/badge.svg)](https://github.com/bpauli/gccli/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/bpauli/gccli)](go.mod)
[![Release](https://img.shields.io/github/v/release/bpauli/gccli)](https://github.com/bpauli/gccli/releases)
[![License](https://img.shields.io/github/license/bpauli/gccli)](LICENSE)
[![Docs](https://img.shields.io/badge/docs-gccli.sh-blue)](https://gccli.sh)
[![Buy Me A Coffee](https://img.shields.io/badge/Buy%20Me%20A%20Coffee-support-yellow?logo=buymeacoffee)](https://buymeacoffee.com/mail7j)

Fast, script-friendly CLI for Garmin Connect. Access activities, health data, body composition, workouts, courses, devices, gear, goals, badges, and more. JSON-first output, multiple accounts, and credential storage in the OS keyring.

## Features

- **Activities** — list, search, view details, download (FIT/GPX/TCX/KML/CSV), upload, create manual entries (with date), rename, retype, describe, change privacy, mark as favorite, delete, set exercise sets for strength training, list activity types, list pending activities
- **Exercise Catalog** — browse Garmin's exercise categories and exercises for strength training
- **Health Data** — daily summaries, steps, heart rate, resting HR, sleep, stress, HRV, SpO2, respiration, body battery, floors, training readiness/status, VO2max, fitness age, race predictions, endurance/hill scores, intensity minutes, lactate threshold, cycling FTP, critical swim speed, power curve, running economy
- **Body Composition** — weight tracking, latest weigh-in, body fat, muscle mass, blood pressure, glucose, FIT file encoding for composition uploads
- **Workouts** — list, view, download as FIT, upload from JSON, update in place from JSON, create with sport types and targets (pace/HR/power/cadence), schedule (add/list/remove/export), list workout types, strength benchmarks, delete
- **Courses** — list courses, view favorites, get full course detail, download as FIT/GPX, import GPX files as new courses, send courses directly to a device, delete courses
- **Devices** - list registered devices, view settings, solar data, alarms, primary/last-used device, paired sensors, backup summary, historical devices, device messages
- **Gear** — list gear, usage stats, linked activities, defaults per activity type, link/unlink to activities, gear types, add, update, retire, unretire and delete gear, list, create, rename and delete gear collections, add gear to a collection
- **Goals & Badges** — active goals, set daily goals for steps, floors, intensity minutes and hydration, earned/available/in-progress badges, badge details, challenges, personal records
- **Training** - browse and view training plans, show your active plans, list adaptive coaching workout types
- **Profile** — user profile and settings, privacy settings, list, add and delete privacy zones, heart rate and power zones
- **Hydration** — view and log daily water intake
- **Nutrition** — view daily food logs, meals, and nutrition settings
- **Events** — list, list upcoming, add, and delete calendar events (races, group rides, etc.) with goals and training priority support
- **Calendar** - add, update, move and delete calendar notes
- **Wellness** - menstrual cycle data and calendar, menstrual daily log (view, set, delete), pregnancy summary, jet lag trips and status
- **Social** - notifications, view and change notification preferences, connections, connection requests, followers, following, newsfeed, groups
- **Agent-ready** - ships a bundled agent skill (`skills/gccli/`) that plugs into AI assistants like Claude Code, Codex, Hermes, and OpenClaw so you can drive Garmin Connect in plain language
- **Multiple accounts** — manage multiple Garmin accounts via `--account` flag
- **Credential storage** in the OS keyring (macOS Keychain, Linux Secret Service), with a file fallback (see [Credential Storage](#credential-storage))
- **Auto-refreshing tokens** — authenticate once, tokens refresh automatically on 401
- **Resilient networking** - API calls retry on 429 (waits for `Retry-After` up to 30 s, 3 retries max) and read requests (GET) retry on 5xx (2 retries max, exponential backoff); writes are not resent on 5xx; an API request fails when no data moves for 60 s, and a transfer that keeps moving has no time limit
- **Shell completion** — generated scripts for bash, zsh, fish, and PowerShell
- **Parseable output** - JSON and plain/TSV modes for scripting and automation. `--json` output stays stable within a major version. Table and `--plain` output may change in a minor release
- **Your unit system** - tables show km or miles, pace per km or per mile, and elevation in metres or feet, as set in your Garmin Connect profile; pin one with `--units`

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
- `--plain`: tab-separated values on stdout, with no header row unless you add `--header`. Only these commands print TSV: `activities list`, `activities search`, `activities types`, `activity summary`, `courses list`, `courses favorites`, `courses import`, `events list`, `workouts list`, `workouts schedule list` and `exercises list` (without `--category`). Most other commands that return data print the same JSON in all modes, also without `--json`. `activities count` prints a single number. Use `--json` when a script parses the output.
- `--header`: with `--plain`, the first line holds the column names. `activity summary` has no column names, so it prints no header line. `--header` without `--plain`, or with `--json`, is a usage error (exit code 2). `GCCLI_HEADER` is not an error: table and JSON output ignore it.
- Table and `--plain` output escape the characters that would break a row or a column. A backslash prints as `\\`, a tab as `\t`, a newline as `\n`, a carriage return as `\r`, a vertical tab as `\v` and a form feed as `\f`. So each record is one line, and in `--plain` output only real tabs separate the fields. `--json` output does not change.
- Write commands (the commands that add, create, upload, import, update, set, rename, retype, describe, change, favorite, unfavorite, schedule, send, link, unlink, remove or delete data in Garmin Connect) all follow one rule:
  - With `--json`, they print the Garmin response on stdout, indented like all JSON output. When Garmin sends an empty response, they print `{"ok": true}`.
  - Without `--json`, they print one message on stderr and nothing on stdout. `courses import` is the only exception. It prints the saved course as a table on stdout (TSV with `--plain`).
- Human-facing messages (errors, warnings, info) go to stderr.
- Delete commands ask for confirmation in a terminal. Without a terminal (scripts, pipes, agents), they exit with an error unless you pass `--force`. If you answer no, the command prints `Cancelled` on stderr, prints nothing on stdout and exits with 0.
- Tables, TSV and JSON on stdout have no colors. The messages on stderr use colors. The `--help` text uses bold and faint text when stdout is a terminal, and `--color` does not change this.
- With `--color auto` (the default), gccli uses colors when stderr is a color terminal and neither `NO_COLOR` nor `CI` is set. `--color always` and `--color never` override this detection. `GCCLI_COLOR` sets the mode when you do not give `--color`. `--json` and `--plain` do not change the colors.

### Exit Codes

| Code | Meaning |
|---|---|
| 0 | Success |
| 1 | Other error |
| 2 | Usage error (unknown command, flag or value). The usage text goes to stderr. |
| 3 | Login needed: no account, no stored credentials, or the token expired and could not be refreshed |
| 4 | Garmin returned 404 Not Found |
| 5 | Garmin rate limit (HTTP 429) |

With `--json`, a runtime error goes to stderr as one JSON line. Usage errors stay plain text, because they happen before the flags are parsed. `code` is `error`, `auth_required`, `not_found` or `rate_limited`. `status` is the HTTP status, when there is one:

```json
{"error":{"code":"rate_limited","status":429,"message":"Rate limited. Wait 30 and retry."}}
```

### Units

Six commands convert units in table and plain output: `activities list`, `activities search`, `activity summary`, `courses list`, `courses favorites` and `courses import`. They show distances, speeds, paces and elevations in the unit system from your Garmin Connect profile. To read the profile, each of these commands makes one extra request. It does this only in table or plain mode, and only when the unit system is `auto`.

| Value | Distance | Speed | Pace | Elevation |
| --- | --- | --- | --- | --- |
| `metric` | km | km/h | /km | m |
| `statute_us` | mi | mph | /mi | ft |
| `statute_uk` | mi | mph | /mi | m |
| `auto` (default) | from the Garmin profile | | | |

The unit system is resolved in this order:

1. `--units` flag
2. `GCCLI_UNITS` environment variable
3. `"units"` in `config.json`
4. Garmin profile setting
5. `metric`

A pinned system (steps 1 to 3) skips the Garmin request. If gccli cannot read the profile setting, it prints a warning and uses metric.

JSON output is never converted. It is the raw Garmin data in meters and meters per second.

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
gccli health training-readiness [date]
gccli health training-status [date]
gccli health training-status daily [date]

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
gccli health hill-score [date]
gccli health intensity-minutes [date]
gccli health intensity-minutes weekly --start 2024-01-01 --end 2024-01-31
gccli health acclimation --start 2024-01-01 --end 2024-01-07
gccli health one-rep-max [date]

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
gccli workouts benchmarks list
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
gccli devices backups
gccli devices historical
gccli devices messages
```

### Gear

```bash
gccli gear list
gccli gear stats <uuid>
gccli gear activities <uuid> --limit 20
gccli gear defaults
gccli gear link <uuid> <activity-id>
gccli gear unlink <uuid> <activity-id>
gccli gear types
gccli gear add "Pegasus 41" --brand Nike --model Pegasus --type shoes
gccli gear update <uuid> --name "Pegasus 41 (old)"
gccli gear retire <uuid>          # gear unretire <uuid> undoes it
gccli gear delete <uuid> --force
gccli gear collections list
gccli gear collections create "Road shoes"
gccli gear collections add <collection-uuid> <gear-uuid>
```

### Goals, Badges & Challenges

```bash
gccli goals list                  # active goals (default)
gccli goals list --status past    # active, future or past
gccli goals weight --start 2026-01-01 --end 2026-06-30
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
gccli records                     # personal records (same as records list)
gccli records types
gccli records activity <activityId>
```

### Profile

```bash
gccli profile
gccli profile settings
gccli profile privacy
gccli profile privacy-zones
gccli profile privacy-zones add --lat 52.52 --lon 13.405 --radius 500
gccli profile privacy-zones add --lat 40.7 --lon=-74.0 --radius 1000   # A negative value needs the = form
gccli profile privacy-zones delete <id>          # Delete a privacy zone (with confirmation)
gccli profile hr-zones
gccli profile power-zones
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
```

### Training Plans

```bash
gccli training plans --locale en
gccli training plan <id>
gccli training active
gccli training active --start 2026-01-01
gccli training coach-workouts
```

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

### Wellness

```bash
gccli wellness menstrual-cycle --start 2024-01-01 --end 2024-03-31
gccli wellness menstrual-summary --start 2024-01-01 --end 2024-03-31
gccli wellness menstrual-calendar --start 2024-01-01 --end 2024-03-31
gccli wellness menstrual-log 2026-10-01           # Daily log ({} for a day without a log)
gccli wellness menstrual-log set 2026-10-01 --flow light --symptoms cramps,headache
gccli wellness menstrual-log set 2026-10-01 --notes ""   # Clear the notes, keep the other fields
gccli wellness menstrual-log delete 2026-10-01    # Delete the daily log (with confirmation)
gccli wellness pregnancy-summary
gccli wellness jet-lag list
gccli wellness jet-lag status 2024-03-15
```

### Social

```bash
gccli notifications list               # List your notifications
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
gccli groups list                      # List your groups
```

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
- `gccli activities types`: List all activity types supported by Garmin.
- `gccli activities pending`: List pending activities.
- `gccli activities stats view`: Show activity totals by activity type. Also runs as `gccli activities stats`.
- `gccli activities stats monthly`: Show monthly activity totals by activity type.
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
- `gccli activity exercise-sets show <id>`: Show exercise sets. Also runs as `gccli activity exercise-sets <id>`.
  - `<id>`: Activity ID.
- `gccli activity exercise-sets set --exercise=EXERCISE,... <id>`: Set exercise sets for a strength training activity.
  - `<id>`: Activity ID.
  - `-e, --exercise=EXERCISE,...`: Exercise set in format CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS] (e.g. BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60).
- `gccli activity gear <id>`: Show linked gear.
  - `<id>`: Activity ID.
- `gccli activity comments <id>`: List the comments on an activity.
  - `<id>`: Activity ID.
- `gccli activity likes <id>`: List the likes of an activity.
  - `<id>`: Activity ID.
- `gccli activity download <id> [flags]`: Download activity file.
  - `<id>`: Activity ID.
  - `--format="fit"`: Download format: fit, gpx, tcx, kml, csv.
  - `-o, --output=STRING`: Output file path (default: activity_{id}.{format}).
- `gccli activity upload <file>`: Upload an activity file (FIT/GPX/TCX).
  - `<file>`: Path to activity file (FIT, GPX, or TCX).
- `gccli activity create --name=STRING --type=STRING --duration=DURATION [flags]`: Create a manual activity.
  - `--name=STRING`: Activity name.
  - `--type=STRING`: Activity type key (e.g. running, cycling, swimming).
  - `--distance=FLOAT-64`: Distance in meters.
  - `--duration=DURATION`: Duration (e.g. 30m, 1h15m).
  - `--date=STRING`: Start date/time (e.g. 2024-06-15T07:30:00). Defaults to now.
- `gccli activity rename <id> <name>`: Rename an activity.
  - `<id>`: Activity ID.
  - `<name>`: New activity name.
- `gccli activity retype --type-id=INT --type-key=STRING <id> [flags]`: Change activity type.
  - `<id>`: Activity ID.
  - `--type-id=INT`: Activity type ID.
  - `--type-key=STRING`: Activity type key (e.g. running, cycling).
  - `--parent-type-id=0`: Parent activity type ID.
- `gccli activity describe <id> <text>`: Set or clear the description of an activity.
  - `<id>`: Activity ID.
  - `<text>`: New description. An empty text ("") clears it.
- `gccli activity privacy <id> <level>`: Change who can see an activity.
  - `<id>`: Activity ID.
  - `<level>`: Who can see the activity: public, private or connections.
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
- `gccli health race-predictions view [<date>]`: Show current race predictions. Also runs as `gccli health race-predictions [<date>]`.
  - `[<date>]`: Unused. Current predictions are always returned.
- `gccli health race-predictions range --start=DATE --end=DATE`: Show race predictions for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health endurance-score [<date>]`: Show endurance score data.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health hill-score [<date>]`: Show hill score data.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli health power-curve --start=DATE --end=DATE`: Show power curve for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli health running-economy [<date>]`: Show latest running economy as of a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli health acclimation --start=DATE --end=DATE`: Show acclimation data for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
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
- `gccli body delete-weight --date=STRING <pk> [flags]`: Delete a weigh-in entry.
  - `<pk>`: Weigh-in version (primary key).
  - `--date=STRING`: Date of the weigh-in (YYYY-MM-DD).
  - `-f, --force`: Skip confirmation prompt.
- `gccli body blood-pressure --start=DATE --end=DATE`: Show blood pressure data.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli body add-blood-pressure --systolic=INT --diastolic=INT --pulse=INT [flags]`: Add a blood pressure measurement.
  - `--systolic=INT`: Systolic pressure (mmHg).
  - `--diastolic=INT`: Diastolic pressure (mmHg).
  - `--pulse=INT`: Pulse rate (bpm).
  - `--notes=STRING`: Optional notes.
- `gccli body delete-blood-pressure --date=STRING <version> [flags]`: Delete a blood pressure entry.
  - `<version>`: Blood pressure entry version.
  - `--date=STRING`: Date of the measurement (YYYY-MM-DD).
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
- `gccli workouts types`: List workout types.
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
- `gccli workouts delete <id> [flags]`: Delete a workout.
  - `<id>`: Workout ID.
  - `-f, --force`: Skip confirmation prompt.

### Courses

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
  - `-t, --type="cycling"`: Activity type key (e.g. running, cycling, hiking).
  - `-p, --privacy=2`: Course privacy: 1=public, 2=private, 4=group.
- `gccli courses send <course-id> <device-id>`: Send a course to a device.
  - `<course-id>`: Course ID.
  - `<device-id>`: Device ID.
- `gccli courses delete <id> [flags]`: Delete a course.
  - `<id>`: Course ID.
  - `-f, --force`: Skip confirmation prompt.

### Devices

- `gccli devices list`: List registered devices. Also runs as `gccli devices`.
- `gccli devices settings <device-id>`: Show settings for a device.
  - `<device-id>`: Device ID.
- `gccli devices primary`: Show primary training device.
- `gccli devices solar <device-id> [flags]`: Show solar charging data for a device.
  - `<device-id>`: Device ID.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to start date.
- `gccli devices alarms`: Show alarms from all devices.
- `gccli devices last-used`: Show last used device.
- `gccli devices sensors`: List paired sensors.
- `gccli devices backups`: Show device backup summary.
- `gccli devices historical`: List historical devices.
- `gccli devices messages`: List device messages.

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
- `gccli gear types`: List gear types.
- `gccli gear add --brand=STRING --model=STRING <name> [flags]`: Add a gear item.
  - `<name>`: Gear name.
  - `--type="shoes"`: Gear type. One of: shoes, bike, other.
  - `--brand=STRING`: Brand.
  - `--model=STRING`: Model.
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
- `gccli gear collections delete <uuid> [flags]`: Delete a gear collection. The gear items stay.
  - `<uuid>`: Collection UUID.
  - `-f, --force`: Skip confirmation prompt.

### Goals, Badges & Challenges

- `gccli goals list [flags]`: List goals. Also runs as `gccli goals [flags]`.
  - `--status="active"`: Goal status: active, future, past.
- `gccli goals weight [flags]`: Show the effective weight goal for a date range.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
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
- `gccli challenges list`: Show joined challenges. Also runs as `gccli challenges`.
- `gccli challenges badge`: Show badge challenges.
- `gccli challenges available`: Show available badge challenges.
- `gccli challenges completed`: Show completed badge challenges.
- `gccli challenges virtual available`: Show available virtual challenges.
- `gccli challenges virtual completed`: Show completed virtual challenges.
- `gccli challenges virtual in-progress`: Show virtual challenges in progress.
- `gccli records list`: View personal records. Also runs as `gccli records`.
- `gccli records types`: List personal record types.
- `gccli records activity <id>`: List the personal records set in an activity.
  - `<id>`: Activity ID.

### Events

- `gccli events list [flags]`: List calendar events. Also runs as `gccli events [flags]`.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
  - `-l, --limit=20`: Maximum number of events to return.
  - `--offset=0`: Number of events to skip.
  - `--sort="eventDate_asc"`: Sort order. One of: eventDate_asc, eventDate_desc.
- `gccli events upcoming`: List upcoming events.
- `gccli events add --name=STRING --date=STRING --type=STRING [flags]`: Add a calendar event.
  - `-n, --name=STRING`: Event name.
  - `-d, --date=STRING`: Event date (YYYY-MM-DD, today, +30d).
  - `-t, --type=STRING`: Event type. One of: running, trail_running, cycling, gravel_cycling, mountain_biking, swimming, triathlon, multi_sport, hiking, walking, fitness_equipment, motorcycling, winter_sport, other.
  - `--race`: Mark as a race.
  - `--location=STRING`: Event location.
  - `--time=STRING`: Start time (HH:MM).
  - `--timezone=STRING`: Time zone ID (e.g. Europe/Berlin).
  - `--distance=STRING`: Target distance (e.g. 10km, 26.2mi, 400m).
  - `--goal=STRING`: Time goal duration (e.g. 50m, 1h30m, 2400s).
  - `--primary`: Set as primary training event.
  - `--training`: Set as supporting training event.
  - `--private`: Make event private.
  - `--note=STRING`: Event note.
  - `--url=STRING`: Event URL.
- `gccli events delete <id> [flags]`: Delete a calendar event.
  - `<id>`: Event ID.
  - `-f, --force`: Skip confirmation prompt.

### Calendar

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
- `gccli groups list`: List your groups. Also runs as `gccli groups`.
- `gccli notifications list`: List notifications. Also runs as `gccli notifications`.
- `gccli notifications preferences view`: Show notification preferences. Also runs as `gccli notifications preferences`.
- `gccli notifications preferences set <key> <value>`: Turn a notification preference on or off.
  - `<key>`: Preference key, such as allowEmailWeeklySummary. Run gccli notifications preferences to see the keys.
  - `<value>`: true turns the preference on, false turns it off.

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
  - `--date=""`: Date to log for (YYYY-MM-DD). Defaults to today.
- `gccli nutrition food-log [<date>]`: Show daily food log for a date. Also runs as `gccli nutrition [<date>]`.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli nutrition meals [<date>]`: Show meal summaries for a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli nutrition settings [<date>]`: Show nutrition settings for a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli profile view`: Show user profile. Also runs as `gccli profile`.
- `gccli profile settings`: Show user settings.
- `gccli profile privacy view`: Show privacy settings. Also runs as `gccli profile privacy`.
- `gccli profile privacy-zones list`: List privacy zones. Also runs as `gccli profile privacy-zones`.
- `gccli profile privacy-zones add --lat=DEGREES --lon=DEGREES --radius=METERS`: Add a privacy zone around a point.
  - `--lat=DEGREES`: Latitude of the center, -90 to 90. Give a negative value as --lat=-33.9.
  - `--lon=DEGREES`: Longitude of the center, -180 to 180. Give a negative value as --lon=-30.
  - `--radius=METERS`: Radius in meters, 500 to 1500.
- `gccli profile privacy-zones delete <id> [flags]`: Delete a privacy zone.
  - `<id>`: Privacy zone ID.
  - `-f, --force`: Skip confirmation prompt.
- `gccli profile hr-zones`: Show heart rate zones.
- `gccli profile power-zones`: Show power zones for all sports.
- `gccli training plans [flags]`: List training plans. Also runs as `gccli training [flags]`.
  - `--locale="en"`: Locale for training plans (e.g. en, de, fr).
- `gccli training plan <id>`: Show a specific training plan.
  - `<id>`: Training plan ID.
- `gccli training active [flags]`: Show active training plans.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli training coach-workouts`: List adaptive coaching workout types.
- `gccli wellness menstrual-cycle --start=DATE --end=DATE`: Show menstrual cycle data.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
- `gccli wellness menstrual-summary --start=DATE --end=DATE`: Show menstrual cycle summary.
  - `--start=DATE`: Start date (YYYY-MM-DD, today, yesterday, 3d, +30d).
  - `--end=DATE`: End date (YYYY-MM-DD, today, yesterday, 3d, +30d).
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

Many commands accept flexible date formats. Every `--start` and `--end` flag accepts all of them. A bad or empty date in one of these flags is a usage error (exit code 2).

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

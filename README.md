# gccli — Garmin Connect in your terminal.

[![ci](https://github.com/bpauli/gccli/actions/workflows/ci.yml/badge.svg)](https://github.com/bpauli/gccli/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/bpauli/gccli)](https://goreportcard.com/report/github.com/bpauli/gccli)
[![Docs](https://img.shields.io/badge/docs-bpauli.de%2Fgccli-blue)](http://bpauli.de/gccli/)

Fast, script-friendly CLI for Garmin Connect. Access activities, health data, body composition, workouts, devices, gear, goals, badges, and more. JSON-first output, multiple accounts, and secure credential storage built in.

## Features

- **Activities** — list, search, view details, download (FIT/GPX/TCX/KML/CSV), upload, create manual entries, rename, retype, delete
- **Health Data** — daily summaries, steps, heart rate, resting HR, sleep, stress, HRV, SpO2, respiration, body battery, floors, training readiness/status, VO2max, fitness age, race predictions, endurance/hill scores, intensity minutes, lactate threshold, cycling FTP
- **Body Composition** — weight tracking, body fat, muscle mass, blood pressure, FIT file encoding for composition uploads
- **Workouts** — list, view, download as FIT, upload from JSON, create with sport types and targets (pace/HR/power/cadence), schedule, delete
- **Devices** — list registered devices, view settings, solar data, alarms, primary/last-used device
- **Gear** — list gear, usage stats, linked activities, defaults per activity type, link/unlink to activities
- **Goals & Badges** — active goals, earned/available/in-progress badges, challenges, personal records
- **Training** — browse and view training plans
- **Profile** — user profile and settings
- **Hydration** — view and log daily water intake
- **Wellness** — menstrual cycle data, pregnancy summary
- **Multiple accounts** — manage multiple Garmin accounts via `--account` flag
- **Secure credential storage** using OS keyring (macOS Keychain, Linux Secret Service, encrypted file fallback)
- **Auto-refreshing tokens** — authenticate once, tokens refresh automatically on 401
- **Resilient networking** — automatic retry on 429/5xx with exponential backoff, circuit breaker for fault tolerance
- **Parseable output** — JSON and plain/TSV modes for scripting and automation

## Installation

### Homebrew (macOS / Linux)

```bash
brew install bpauli/tap/gccli
```

### Build from Source

Requires Go 1.24+.

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

This opens your browser for Garmin SSO authentication. The OAuth tokens are stored securely in your system keyring.

### 2. Log In (Headless)

For servers or environments without a browser:

```bash
gccli auth login you@example.com --headless
```

You'll be prompted for your password. If your account has two-factor authentication:

```bash
gccli auth login you@example.com --headless --mfa-code 123456
```

### 3. Set Your Default Account

```bash
export GCCLI_ACCOUNT=you@example.com
```

### 4. Test Authentication

```bash
gccli auth status
gccli activities list --limit 5
```

## Authentication & Secrets

### Accounts and Tokens

`gccli` stores your OAuth tokens in a keyring backend. Default is auto-detection (best available backend for your OS/environment).

List current auth state:

```bash
gccli auth status
```

Print the raw OAuth2 access token (useful for scripting):

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
- **file** — encrypted on-disk keyring (fallback)

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

Specify the account using either a flag or environment variable:

```bash
# Via flag
gccli activities list --account you@example.com

# Via environment
export GCCLI_ACCOUNT=you@example.com
gccli activities list
```

### Output

- Default: human-friendly tables on stdout.
- `--plain`: stable TSV on stdout (best for piping).
- `--json` / `-j`: JSON on stdout (best for scripting).
- Human-facing messages (errors, warnings, info) go to stderr.
- Colors are enabled by default in TTY mode and disabled for `--json` and `--plain`.

### Config File

Config path:

- macOS: `~/Library/Application Support/gccli/config.json`
- Linux: `~/.config/gccli/config.json` (or `$XDG_CONFIG_HOME/gccli/config.json`)

Example:

```json
{
  "keyring_backend": "file",
  "domain": "garmin.com"
}
```

### Environment Variables

| Variable | Description |
| --- | --- |
| `GCCLI_ACCOUNT` | Default account email |
| `GCCLI_DOMAIN` | Garmin domain (`garmin.com` or `garmin.cn`) |
| `GCCLI_JSON` | Enable JSON output (`1`, `true`, `yes`) |
| `GCCLI_PLAIN` | Enable plain/TSV output (`1`, `true`, `yes`) |
| `GCCLI_COLOR` | Color mode: `auto`, `always`, `never` |
| `GCCLI_KEYRING_BACKEND` | Keyring backend: `keychain`, `secret-service`, `file` |

## Security

### Credential Storage

OAuth tokens are stored securely in your system's keyring:

- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Fallback**: Encrypted on-disk file store

The CLI uses [github.com/99designs/keyring](https://github.com/99designs/keyring) for secure storage.

Tokens are keyed by email (`gccli:token:<email>`) and never stored in plaintext files.

## Commands

### Authentication

```bash
gccli auth login <email>               # Log in via browser SSO
gccli auth login <email> --headless    # Log in via email/password
gccli auth login <email> --headless --mfa-code <code>  # With MFA
gccli auth status                      # Show auth state and token expiry
gccli auth token                       # Print OAuth2 access token
gccli auth remove                      # Remove stored credentials
```

### Activities

```bash
# List and search
gccli activities list --limit 20 --start 0
gccli activities list --type running
gccli activities count
gccli activities search --start-date 2024-01-01 --end-date 2024-12-31

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
gccli activity gear <id>

# Download and upload
gccli activity download <id> --format fit
gccli activity download <id> --format gpx --output track.gpx
gccli activity upload ./activity.fit

# Create and modify
gccli activity create --name "Morning Run" --type running --duration 30m --distance 5000
gccli activity rename <id> "New Name"
gccli activity retype <id> --type-id 1 --type-key running
gccli activity delete <id>
gccli activity delete <id> --force
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
gccli health hr [date]                 # Heart rate
gccli health rhr [date]                # Resting heart rate
gccli health floors [date]             # Floors climbed
gccli health sleep [date]              # Sleep data
gccli health respiration [date]
gccli health spo2 [date]               # Blood oxygen
gccli health hrv [date]                # Heart rate variability

# Stress and recovery
gccli health stress                    # Stress for today
gccli health stress weekly --weeks 4
gccli health body-battery [date]
gccli health body-battery range --start 2024-01-01 --end 2024-01-07
gccli health training-readiness [date]
gccli health training-status [date]

# Fitness metrics
gccli health fitness-age [date]
gccli health max-metrics [date]        # VO2max and more
gccli health lactate-threshold
gccli health cycling-ftp
gccli health race-predictions
gccli health race-predictions range --start 2024-01-01 --end 2024-06-30
gccli health endurance-score [date]
gccli health hill-score [date]
gccli health intensity-minutes [date]
gccli health intensity-minutes weekly --start 2024-01-01 --end 2024-01-31

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

# Add data
gccli body add-weight 75.5 --unit kg
gccli body add-weight 166.4 --unit lbs
gccli body add-composition 75.5 --body-fat 15.2 --muscle-mass 35.0

# Blood pressure
gccli body blood-pressure --start 2024-01-01 --end 2024-01-31
gccli body add-blood-pressure --systolic 120 --diastolic 80 --pulse 65

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
gccli workouts schedule <id> <YYYY-MM-DD>
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

### Devices

```bash
gccli devices list
gccli devices settings <device-id>
gccli devices primary
gccli devices last-used
gccli devices alarms
gccli devices solar <device-id> --start 2024-06-01 --end 2024-06-30
```

### Gear

```bash
gccli gear list
gccli gear stats <uuid>
gccli gear activities <uuid> --limit 20
gccli gear defaults
gccli gear link <uuid> <activity-id>
gccli gear unlink <uuid> <activity-id>
```

### Goals, Badges & Challenges

```bash
gccli goals list
gccli goals list --status active
gccli badges earned
gccli badges available
gccli badges in-progress
gccli challenges list
gccli challenges badge
gccli records
```

### Profile

```bash
gccli profile
gccli profile settings
```

### Hydration

```bash
gccli hydration [date]                 # View hydration data
gccli hydration add 500                # Log 500ml of water
gccli hydration add 500 --date 2024-06-15
```

### Training Plans

```bash
gccli training plans --locale en
gccli training plan <id>
```

### Wellness

```bash
gccli wellness menstrual-cycle --start-date 2024-01-01 --end-date 2024-03-31
gccli wellness menstrual-summary --start-date 2024-01-01 --end-date 2024-03-31
gccli wellness pregnancy-summary
```

### Data Reload

```bash
gccli reload                           # Reload today's data
gccli reload 2024-06-15                # Reload specific date
```

## Output Formats

### Table (default)

Human-readable output with colors:

```bash
$ gccli activities list --limit 3
ID          NAME                TYPE       DATE
123456789   Morning Run         running    2024-06-15
123456780   Evening Walk        walking    2024-06-14
123456771   Cycling Workout     cycling    2024-06-13
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

Stable, tab-separated output for scripting:

```bash
$ gccli --plain activities list --limit 3
123456789	Morning Run	running	2024-06-15
123456780	Evening Walk	walking	2024-06-14
123456771	Cycling Workout	cycling	2024-06-13
```

## Examples

### Export all runs from the last month

```bash
gccli --json activities search --start-date 2024-05-15 --end-date 2024-06-15 --limit 100 | \
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
| `--account <email>` | Account to use (overrides `GCCLI_ACCOUNT`) |
| `--json`, `-j` | Output JSON to stdout |
| `--plain` | Output stable TSV to stdout |
| `--color <mode>` | Color mode: `auto`, `always`, `never` (default: `auto`) |
| `--version` | Print version information |
| `--help` | Show help for any command |

## Date Format Reference

Many commands accept flexible date formats:

| Format | Example | Description |
| --- | --- | --- |
| `YYYY-MM-DD` | `2024-06-15` | Explicit date |
| `today` | | Today's date |
| `yesterday` | | Yesterday's date |
| `Nd` | `3d`, `7d` | N days ago |

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

Inspired by [gogcli](https://github.com/steipete/gogcli). API coverage based on [python-garminconnect](https://github.com/cyberjunky/python-garminconnect) (113 API methods).

Platforms: macOS (amd64, arm64), Linux (amd64, arm64). macOS builds with CGO enabled for Keychain access.

## License

MIT

## Links

- [Garmin Connect](https://connect.garmin.com)
- [python-garminconnect](https://github.com/cyberjunky/python-garminconnect) — API reference
- [gogcli](https://github.com/steipete/gogcli) — architecture inspiration

## Credits

Architecture inspired by [gogcli](https://github.com/steipete/gogcli) by Peter Steinberger. API coverage modeled after [python-garminconnect](https://github.com/cyberjunky/python-garminconnect) by Ron Klinkien.

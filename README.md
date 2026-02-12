# gccli — Garmin Connect in your terminal.

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

### Build from Source

Requires Go 1.24+.

```bash
git clone https://github.com/bpauli/gccli.git
cd gccli
make build
```

The binary is output to `./bin/gc`.

Run:

```bash
./bin/gc --help
```

Help:

- `gc --help` shows top-level command groups.
- Drill down with `gc <group> --help` (and deeper subcommands).
- Make shortcut: `make run -- --help` (or `make run -- activities --help`).

## Quick Start

### 1. Log In (Browser SSO)

```bash
gc auth login you@example.com
```

This opens your browser for Garmin SSO authentication. The OAuth tokens are stored securely in your system keyring.

### 2. Log In (Headless)

For servers or environments without a browser:

```bash
gc auth login you@example.com --headless
```

You'll be prompted for your password. If your account has two-factor authentication:

```bash
gc auth login you@example.com --headless --mfa-code 123456
```

### 3. Set Your Default Account

```bash
export GC_ACCOUNT=you@example.com
```

### 4. Test Authentication

```bash
gc auth status
gc activities list --limit 5
```

## Authentication & Secrets

### Accounts and Tokens

`gc` stores your OAuth tokens in a keyring backend. Default is auto-detection (best available backend for your OS/environment).

List current auth state:

```bash
gc auth status
```

Print the raw OAuth2 access token (useful for scripting):

```bash
gc auth token
```

Remove stored credentials:

```bash
gc auth remove
```

### Keyring Backend

Backends:

- **auto** (default) — picks the best backend for the platform
- **keychain** — macOS Keychain
- **secret-service** — Linux D-Bus (GNOME Keyring, KWallet)
- **file** — encrypted on-disk keyring (fallback)

Set backend via environment variable:

```bash
export GC_KEYRING_BACKEND=file
```

Or in the config file (`config.json`):

```json
{
  "keyring_backend": "file"
}
```

On Linux, if D-Bus is unavailable, `gc` automatically falls back to the file backend.

### Garmin Domain (China)

For Garmin China accounts (garmin.cn):

```bash
export GC_DOMAIN=garmin.cn
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
gc activities list --account you@example.com

# Via environment
export GC_ACCOUNT=you@example.com
gc activities list
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
| `GC_ACCOUNT` | Default account email |
| `GC_DOMAIN` | Garmin domain (`garmin.com` or `garmin.cn`) |
| `GC_JSON` | Enable JSON output (`1`, `true`, `yes`) |
| `GC_PLAIN` | Enable plain/TSV output (`1`, `true`, `yes`) |
| `GC_COLOR` | Color mode: `auto`, `always`, `never` |
| `GC_KEYRING_BACKEND` | Keyring backend: `keychain`, `secret-service`, `file` |

## Security

### Credential Storage

OAuth tokens are stored securely in your system's keyring:

- **macOS**: Keychain Access
- **Linux**: Secret Service (GNOME Keyring, KWallet)
- **Fallback**: Encrypted on-disk file store

The CLI uses [github.com/99designs/keyring](https://github.com/99designs/keyring) for secure storage.

Tokens are keyed by email (`gc:token:<email>`) and never stored in plaintext files.

## Commands

### Authentication

```bash
gc auth login <email>               # Log in via browser SSO
gc auth login <email> --headless    # Log in via email/password
gc auth login <email> --headless --mfa-code <code>  # With MFA
gc auth status                      # Show auth state and token expiry
gc auth token                       # Print OAuth2 access token
gc auth remove                      # Remove stored credentials
```

### Activities

```bash
# List and search
gc activities list --limit 20 --start 0
gc activities list --type running
gc activities count
gc activities search --start-date 2024-01-01 --end-date 2024-12-31

# View details
gc activity summary <id>
gc activity details <id>
gc activity splits <id>
gc activity typed-splits <id>
gc activity split-summaries <id>
gc activity weather <id>
gc activity hr-zones <id>
gc activity power-zones <id>
gc activity exercise-sets <id>
gc activity gear <id>

# Download and upload
gc activity download <id> --format fit
gc activity download <id> --format gpx --output track.gpx
gc activity upload ./activity.fit

# Create and modify
gc activity create --name "Morning Run" --type running --duration 30m --distance 5000
gc activity rename <id> "New Name"
gc activity retype <id> --type-id 1 --type-key running
gc activity delete <id>
gc activity delete <id> --force
```

### Health Data

```bash
# Daily summary
gc health summary                   # Today
gc health summary yesterday
gc health summary 2024-06-15
gc health summary 3d                # 3 days ago

# Vitals
gc health steps                     # Step chart for today
gc health steps daily --start 2024-01-01 --end 2024-01-31
gc health steps weekly --weeks 4
gc health hr [date]                 # Heart rate
gc health rhr [date]                # Resting heart rate
gc health floors [date]             # Floors climbed
gc health sleep [date]              # Sleep data
gc health respiration [date]
gc health spo2 [date]               # Blood oxygen
gc health hrv [date]                # Heart rate variability

# Stress and recovery
gc health stress                    # Stress for today
gc health stress weekly --weeks 4
gc health body-battery [date]
gc health body-battery range --start 2024-01-01 --end 2024-01-07
gc health training-readiness [date]
gc health training-status [date]

# Fitness metrics
gc health fitness-age [date]
gc health max-metrics [date]        # VO2max and more
gc health lactate-threshold
gc health cycling-ftp
gc health race-predictions
gc health race-predictions range --start 2024-01-01 --end 2024-06-30
gc health endurance-score [date]
gc health hill-score [date]
gc health intensity-minutes [date]
gc health intensity-minutes weekly --start 2024-01-01 --end 2024-01-31

# Wellness events
gc health events [date]
gc health lifestyle [date]
```

### Body Composition

```bash
# View data
gc body composition                 # Today
gc body composition --start 2024-01-01 --end 2024-01-31
gc body weigh-ins --start 2024-01-01 --end 2024-01-31
gc body daily-weigh-ins [date]

# Add data
gc body add-weight 75.5 --unit kg
gc body add-weight 166.4 --unit lbs
gc body add-composition 75.5 --body-fat 15.2 --muscle-mass 35.0

# Blood pressure
gc body blood-pressure --start 2024-01-01 --end 2024-01-31
gc body add-blood-pressure --systolic 120 --diastolic 80 --pulse 65

# Delete entries
gc body delete-weight <pk> --date 2024-01-15
gc body delete-blood-pressure <version> --date 2024-01-15
```

### Workouts

```bash
gc workouts list --limit 20
gc workouts detail <id>
gc workouts download <id> --output workout.fit
gc workouts upload ./workout.json   # See JSON structure below
gc workouts schedule <id> <YYYY-MM-DD>
gc workouts delete <id>

# Create a running workout with pace targets
gc workouts create "Easy 30min Run" --type run \
  --step "warmup:5m@pace:5:30-6:00" \
  --step "run:20m@pace:5:00-5:30" \
  --step "cooldown:5m"

# Running with heart rate targets
gc workouts create "HR Zone Run" --type run \
  --step "warmup:10m" \
  --step "run:20m@hr:140-160" \
  --step "cooldown:10m"

# Cycling with power targets
gc workouts create "FTP Intervals" --type bike \
  --step "warmup:10m" \
  --step "run:5m@power:250-280" \
  --step "recovery:3m" \
  --step "run:5m@power:250-280" \
  --step "cooldown:10m"

# Imperial paces (miles)
gc workouts create "Easy 30min Run" --type run \
  --step "warmup:5m@pace:8:51-9:39" \
  --step "run:20m@pace:8:03-8:51" \
  --step "cooldown:5m" \
  --unit mi

# Strength workout (no targets)
gc workouts create "Full Body" --type strength \
  --step "warmup:5m" \
  --step "run:30m" \
  --step "cooldown:5m"
```

**Workout JSON structure** for `gc workouts upload`:

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
gc devices list
gc devices settings <device-id>
gc devices primary
gc devices last-used
gc devices alarms
gc devices solar <device-id> --start 2024-06-01 --end 2024-06-30
```

### Gear

```bash
gc gear list
gc gear stats <uuid>
gc gear activities <uuid> --limit 20
gc gear defaults
gc gear link <uuid> <activity-id>
gc gear unlink <uuid> <activity-id>
```

### Goals, Badges & Challenges

```bash
gc goals list
gc goals list --status active
gc badges earned
gc badges available
gc badges in-progress
gc challenges list
gc challenges badge
gc records
```

### Profile

```bash
gc profile
gc profile settings
```

### Hydration

```bash
gc hydration [date]                 # View hydration data
gc hydration add 500                # Log 500ml of water
gc hydration add 500 --date 2024-06-15
```

### Training Plans

```bash
gc training plans --locale en
gc training plan <id>
```

### Wellness

```bash
gc wellness menstrual-cycle --start-date 2024-01-01 --end-date 2024-03-31
gc wellness menstrual-summary --start-date 2024-01-01 --end-date 2024-03-31
gc wellness pregnancy-summary
```

### Data Reload

```bash
gc reload                           # Reload today's data
gc reload 2024-06-15                # Reload specific date
```

## Output Formats

### Table (default)

Human-readable output with colors:

```bash
$ gc activities list --limit 3
ID          NAME                TYPE       DATE
123456789   Morning Run         running    2024-06-15
123456780   Evening Walk        walking    2024-06-14
123456771   Cycling Workout     cycling    2024-06-13
```

### JSON

Machine-readable output for scripting and automation:

```bash
$ gc --json activities list --limit 3
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
gc --json activities list --limit 100 | jq '.[] | select(.activityType.typeKey == "running")'
```

### Plain (TSV)

Stable, tab-separated output for scripting:

```bash
$ gc --plain activities list --limit 3
123456789	Morning Run	running	2024-06-15
123456780	Evening Walk	walking	2024-06-14
123456771	Cycling Workout	cycling	2024-06-13
```

## Examples

### Export all runs from the last month

```bash
gc --json activities search --start-date 2024-05-15 --end-date 2024-06-15 --limit 100 | \
  jq -r '.[] | select(.activityType.typeKey == "running") | .activityId' | \
  while read id; do
    gc activity download "$id" --format gpx --output "run_${id}.gpx"
  done
```

### Daily health check script

```bash
gc --json health summary | jq '{
  steps: .totalSteps,
  calories: .totalKilocalories,
  distance_km: (.totalDistanceMeters / 1000),
  active_minutes: .moderateIntensityMinutes + .vigorousIntensityMinutes
}'
```

### Track weight over time

```bash
gc --json body weigh-ins --start 2024-01-01 --end 2024-06-30 | \
  jq -r '.dailyWeightSummaries[] | [.summaryDate, .weight.value] | @tsv'
```

### Download the latest activity

```bash
latest=$(gc --json activities list --limit 1 | jq -r '.[0].activityId')
gc activity download "$latest" --format fit
```

## Global Flags

All commands support these flags:

| Flag | Description |
| --- | --- |
| `--account <email>` | Account to use (overrides `GC_ACCOUNT`) |
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
make build          # Build to ./bin/gc
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

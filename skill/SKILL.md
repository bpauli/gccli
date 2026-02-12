---
name: garmin-connect
description: Garmin Connect CLI for activities, health, body composition, workouts, devices, gear, goals, and more.
homepage: https://github.com/bpauli/gccli
metadata:
  {
    "openclaw":
      {
        "emoji": "⌚",
        "os": ["darwin", "linux"],
        "requires": { "bins": ["gccli"] },
        "install":
          [
            {
              "id": "homebrew",
              "kind": "homebrew",
              "formula": "bpauli/tap/gccli",
              "bins": ["gccli"],
              "label": "Homebrew (recommended)",
            },
            {
              "id": "source",
              "kind": "source",
              "url": "https://github.com/bpauli/gccli",
              "bins": ["gccli"],
              "label": "Build from source (Go 1.24+)",
            },
          ],
      },
  }
---

# gccli

Use `gccli` for Garmin Connect health, fitness, and activity data. Requires Garmin SSO authentication.

Setup (once)

- `gccli auth login you@example.com` (opens browser for Garmin SSO)
- Headless: `gccli auth login you@example.com --headless` (with `--mfa-code <code>` for MFA)
- Set default account: `export GCCLI_ACCOUNT=you@example.com`
- Verify: `gccli auth status`

Output

- Default: human-friendly tables. Use `--json` / `-j` for JSON, `--plain` for TSV.
- Data goes to stdout, messages/errors to stderr.
- Always use `--json` when parsing output programmatically.

Date shortcuts

- `today`, `yesterday`, `3d` (N days ago), `YYYY-MM-DD` (specific date).
- Use `--start`/`--end` flags for date ranges.

Common commands

- Auth status: `gccli auth status`
- Auth token (for scripting): `gccli auth token`
- Remove credentials: `gccli auth remove`
- List activities: `gccli activities list --limit 20`
- List activities by type: `gccli activities list --type running`
- Activity count: `gccli activities count`
- Search activities: `gccli activities search --start-date 2024-01-01 --end-date 2024-12-31`
- Activity summary: `gccli activity summary <id>`
- Activity details: `gccli activity details <id>`
- Activity splits: `gccli activity splits <id>`
- Activity weather: `gccli activity weather <id>`
- Activity HR zones: `gccli activity hr-zones <id>`
- Activity power zones: `gccli activity power-zones <id>`
- Activity exercise sets: `gccli activity exercise-sets <id>`
- Activity gear: `gccli activity gear <id>`
- Download activity (FIT): `gccli activity download <id> --format fit`
- Download activity (GPX): `gccli activity download <id> --format gpx --output track.gpx`
- Upload activity: `gccli activity upload ./activity.fit`
- Create manual activity: `gccli activity create --name "Morning Run" --type running --date 2024-06-15T07:30:00 --duration 1800 --distance 5000`
- Rename activity: `gccli activity rename <id> "New Name"`
- Retype activity: `gccli activity retype <id> running`
- Delete activity: `gccli activity delete <id> --force`
- Health summary: `gccli health summary [date]`
- Steps chart: `gccli health steps [date]`
- Daily steps range: `gccli health steps daily --start 2024-01-01 --end 2024-01-31`
- Weekly steps: `gccli health steps weekly --weeks 4`
- Heart rate: `gccli health hr [date]`
- Resting HR: `gccli health rhr [date]`
- Floors climbed: `gccli health floors [date]`
- Sleep: `gccli health sleep [date]`
- Respiration: `gccli health respiration [date]`
- SpO2: `gccli health spo2 [date]`
- HRV: `gccli health hrv [date]`
- Stress: `gccli health stress [date]`
- Weekly stress: `gccli health stress weekly --weeks 4`
- Body battery: `gccli health body-battery [date]`
- Body battery range: `gccli health body-battery range --start 2024-01-01 --end 2024-01-07`
- Training readiness: `gccli health training-readiness [date]`
- Training status: `gccli health training-status [date]`
- Fitness age: `gccli health fitness-age [date]`
- VO2max / max metrics: `gccli health max-metrics [date]`
- Lactate threshold: `gccli health lactate-threshold`
- Cycling FTP: `gccli health cycling-ftp`
- Race predictions: `gccli health race-predictions [date]`
- Race predictions range: `gccli health race-predictions range --start 2024-01-01 --end 2024-06-30`
- Endurance score: `gccli health endurance-score [date]`
- Hill score: `gccli health hill-score [date]`
- Intensity minutes: `gccli health intensity-minutes [date]`
- Weekly intensity minutes: `gccli health intensity-minutes weekly --start 2024-01-01 --end 2024-01-31`
- Wellness events: `gccli health events [date]`
- Body composition: `gccli body composition [date]`
- Body composition range: `gccli body composition --start 2024-01-01 --end 2024-01-31`
- Weigh-ins: `gccli body weigh-ins --start 2024-01-01 --end 2024-01-31`
- Add weight: `gccli body add-weight 75.5 --unit kg`
- Add composition: `gccli body add-composition 75.5 --body-fat 15.2 --muscle-mass 35.0`
- Blood pressure: `gccli body blood-pressure --start 2024-01-01 --end 2024-01-31`
- Add blood pressure: `gccli body add-blood-pressure --systolic 120 --diastolic 80 --pulse 65`
- List workouts: `gccli workouts list --limit 20`
- Workout detail: `gccli workouts detail <id>`
- Download workout (FIT): `gccli workouts download <id> --output workout.fit`
- Upload workout (JSON): `gccli workouts upload ./workout.json`
- Schedule workout: `gccli workouts schedule add <id> 2024-06-20`
- List scheduled workouts: `gccli workouts schedule list 2024-06-20`
- Remove scheduled workout: `gccli workouts schedule remove <schedule-id>` (use `--force` to skip confirmation)
- Delete workout: `gccli workouts delete <id>`
- Create running workout with pace: `gccli workouts create "Easy Run" --type run --step "warmup:5m" --step "run:20m@pace:5:00-5:30" --step "cooldown:5m"`
- Create workout with HR targets: `gccli workouts create "HR Run" --type run --step "warmup:10m" --step "run:20m@hr:140-160" --step "cooldown:10m"`
- Create cycling workout with power: `gccli workouts create "FTP Intervals" --type bike --step "warmup:10m" --step "run:5m@power:250-280" --step "recovery:3m" --step "run:5m@power:250-280" --step "cooldown:10m"`
- List devices: `gccli devices list`
- Device settings: `gccli devices settings <device-id>`
- Primary device: `gccli devices primary`
- Last used device: `gccli devices last-used`
- Device alarms: `gccli devices alarms`
- Solar data: `gccli devices solar <device-id> --start 2024-06-01 --end 2024-06-30`
- List gear: `gccli gear list`
- Gear stats: `gccli gear stats <uuid>`
- Gear activities: `gccli gear activities <uuid> --limit 20`
- Gear defaults: `gccli gear defaults`
- Link gear: `gccli gear link <uuid> <activity-id>`
- Unlink gear: `gccli gear unlink <uuid> <activity-id>`
- Goals: `gccli goals list --status active`
- Earned badges: `gccli badges earned`
- Available badges: `gccli badges available`
- In-progress badges: `gccli badges in-progress`
- Challenges: `gccli challenges list`
- Badge challenges: `gccli challenges badge`
- Personal records: `gccli records`
- Profile: `gccli profile`
- Profile settings: `gccli profile settings`
- Hydration: `gccli hydration [date]`
- Log water: `gccli hydration add 500`
- Training plans: `gccli training plans --locale en`
- Training plan detail: `gccli training plan <id>`
- Menstrual cycle: `gccli wellness menstrual-cycle --start-date 2024-01-01 --end-date 2024-03-31`
- Pregnancy summary: `gccli wellness pregnancy-summary`
- Reload data: `gccli reload [date]`

Notes

- Set `GCCLI_ACCOUNT=you@example.com` to avoid repeating `--account`.
- For scripting, use `--json` for JSON output or `--plain` for TSV.
- Dates support `today`, `yesterday`, `3d`, or `YYYY-MM-DD`.
- Tokens are stored securely in the OS keyring (macOS Keychain, Linux Secret Service, file fallback).
- Tokens auto-refresh on 401; automatic retry on 429/5xx with exponential backoff.
- For Garmin China accounts: `export GCCLI_DOMAIN=garmin.cn`.
- Confirm before deleting activities/workouts (or use `--force`).
- Download formats: FIT (default), GPX, TCX, KML, CSV.
- Workout step format: `type:duration[@target:low-high]` — types: warmup, run, recovery, cooldown; targets: pace (min:sec), hr (bpm), power (watts), cadence.

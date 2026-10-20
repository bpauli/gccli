---
name: garmin-connect
description: Garmin Connect CLI for activities, health, body composition, workouts, devices, gear, goals, and more.
homepage: https://github.com/bpauli/gccli
metadata: {"clawdbot":{"emoji":"⌚","os":["darwin","linux"],"requires":{"bins":["gccli"]},"install":[{"id":"homebrew","kind":"brew","formula":"bpauli/tap/gccli","bins":["gccli"],"label":"Homebrew (recommended)"},{"id":"source","kind":"source","url":"https://github.com/bpauli/gccli","bins":["gccli"],"label":"Build from source (Go 1.26+)"}]}}
---

# gccli

Use `gccli` for Garmin Connect health, fitness, and activity data. Requires Garmin SSO authentication.

Setup (once)

- `gccli auth login you@example.com` (opens browser for Garmin SSO)
- Headless: `gccli auth login you@example.com --headless` (with `--mfa-code <code>` for MFA)
- Set default account: `export GCCLI_ACCOUNT=you@example.com`
- Verify: `gccli auth status` (exits 3 when no credentials are stored)

Output

- Default: human-friendly tables. Use `--json` / `-j` for JSON.
- `--plain` prints TSV with no header row (add `--header` for the column names as the first line), but only for `activities list`, `activities search`, `activities types`, `activity summary`, `body weekly`, `courses list`, `courses favorites`, `courses import`, `events list`, `workouts list`, `workouts schedule list` and `exercises list` (without `-c`). Most other commands print JSON in all modes.
- `--header` works only with `--plain`. With table or `--json` output it is a usage error (exit code 2). `GCCLI_HEADER=1` does the same as `--header` in plain mode, and table and JSON output ignore it.
- Table and `--plain` output escape a backslash as `\\`, a tab as `\t`, a newline as `\n`, a carriage return as `\r`, a vertical tab as `\v` and a form feed as `\f`, so each record is one line. `--json` is not escaped this way.
- Data goes to stdout, messages/errors to stderr.
- Write commands (add, create, upload, import, update, set, rename, retype, describe, change, favorite, unfavorite, schedule, send, link, unlink, remove, reload, delete) print the Garmin response with `--json`, or `{"ok": true}` when the response is empty. Without `--json` they print a message on stderr and nothing on stdout. Only `courses import` prints a table on stdout instead (TSV with `--plain`).
- An empty answer is not an error. When Garmin sends no body, the exit code is 0 and the output follows the shape of the Garmin answer: a command whose answer is a list prints `[]` with `--json`, and a command whose answer is one object prints `{}` in every mode. A list command that prints columns (`activities list`, `activities search`, `activities types`, `body weekly`, `courses list`, `courses favorites`, `events list`, `workouts list` and `workouts schedule list`) prints the header only as a table and nothing with `--plain` (the header line with `--header`); every other list command prints `[]` in every mode. `activity summary` and `exercises list` show one object, so they print `{}` with `--json` and no row otherwise. The file `internal/cmd/testdata/answer_shapes.tsv` in the gccli repository records the shape of the Garmin answer of every command that prints it (`array`, `object` or `unknown`, with the evidence for each; a shape that no Garmin answer confirmed is `unknown` and prints `{}`), and a test keeps the commands in step with it. `activities count` prints `0` (`{"count": 0}` with `--json`). A list answer without its list key gives no rows, and `--json` prints the answer unchanged.
- Always use `--json` when parsing output programmatically.
- Tables show distance, speed, pace, elevation and weight in the user's Garmin unit system (`metric`, `statute_us`, or `statute_uk`, which is miles with elevation in metres). Weight is kg in `metric` and lbs in both statute systems. Override with `--units <system>` or `GCCLI_UNITS`. JSON is never converted (meters, m/s, grams).

Date shortcuts

- `today`, `yesterday`, `3d` (N days ago), `+30d` (N days ahead), `YYYY-MM-DD` (specific date).
- All date range commands use `--start`/`--end`, and these flags accept every form above. So does the optional `[<date>]` argument of the daily views, such as `health sleep [<date>]` or `health steps [<date>]`. A bad or empty date is a usage error (exit code 2), before the login; omit the argument for today.
- `activities list`, `activities search`, `workouts list` and `events list` page with `--limit`/`-l` and `--offset` (number of items to skip). `--limit` must be at least 1 and `--offset` must not be negative; other values are a usage error (exit code 2). In `events list`, `--offset` must be 0 or a multiple of `--limit`.

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
- `gccli activity comments list <id>`: List the comments on an activity. Also runs as `gccli activity comments <id>`.
  - `<id>`: Activity ID.
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
- `gccli health race-predictions view`: Show current race predictions. Also runs as `gccli health race-predictions`.
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
  - `-t, --type="cycling"`: Activity type key. One of: cycling, e_bike_fitness, e_bike_mountain, gravel_cycling, hiking, mountain_biking, other, road_biking, running, street_running, trail_running, ultra_run, walking.
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
  - `--date=DATE`: Date to log for (YYYY-MM-DD). Defaults to today.
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
- `gccli wellness jet-lag status [<date>]`: Show jet lag status for a date.
  - `[<date>]`: Date (YYYY-MM-DD, today, yesterday, 3d, +30d). Defaults to today.
- `gccli reload [<date>]`: Request data reload for a date.
  - `[<date>]`: Date to reload (YYYY-MM-DD, today, yesterday, 3d). Defaults to today.
- `gccli completion <shell>`: Generate shell completion script.
  - `<shell>`: Shell type (bash, zsh, fish, powershell).
<!-- gccli:commands:end -->

Worked examples

- Running workout with a pace target: `gccli workouts create "Easy Run" --type run --step "warmup:5m" --step "run:20m@pace:5:00-5:30" --step "cooldown:5m"`
- Running workout with a heart rate target: `gccli workouts create "HR Run" --type run --step "warmup:10m" --step "run:20m@hr:140-160" --step "cooldown:10m"`
- Cycling workout with a power target: `gccli workouts create "FTP Intervals" --type bike --step "warmup:10m" --step "run:5m@power:250-280" --step "recovery:3m" --step "run:5m@power:250-280" --step "cooldown:10m"`
- Race event: `gccli events add --name "Berlin Marathon" --date 2026-09-27 --type running --race --location "Berlin, Germany" --distance 42.195km --time 09:15 --timezone Europe/Berlin`
- Calendar note: `gccli calendar notes add 2026-10-20 "Long run" --content "Easy pace"` prints the note ID. Then `gccli calendar notes move <id> 2026-10-21`, `gccli calendar notes update <id> --content ""` (clears the text) and `gccli calendar notes delete <id> --force`.
- Menstrual daily log: `gccli wellness menstrual-log set 2026-10-01 --flow light --symptoms cramps,tender-breasts --ovulation` sets fields of the log of a day. `gccli wellness menstrual-log set 2026-10-01 --symptoms ""` clears the symptoms and keeps the other fields. `gccli wellness menstrual-log 2026-10-01` shows the log, and `gccli wellness menstrual-log delete 2026-10-01 --force` deletes it.
- Nap: `gccli health sleep naps add 13:30 13:50` adds a nap for today. Garmin takes only naps that end today, and the times are in the time zone of the Garmin profile, not of the machine. `gccli health sleep naps delete today 13:30 --force` deletes the nap that starts at 13:30.
- Activity like: `gccli activity likes add <id>` likes an activity, and a second add changes nothing. `gccli activity likes <id>` lists the likes. `gccli activity likes remove <id>` removes your like with no prompt. It fails with exit code 1 (`activity <id> is not liked`) when there is no like to remove.
- Notification preference: `gccli notifications preferences` shows the keys. `gccli notifications preferences set allowEmailWeeklySummary false` turns one off. A key that is not in the preferences fails before any write, and the error lists the keys.
- Privacy zone: `gccli profile privacy-zones add --lat 51.5 --lon=-0.12 --radius 500` prints the zone ID. A negative value needs the `=` form. The radius is 500 to 1500 meters. `gccli profile privacy-zones delete <id> --force` deletes it.
- Race event with a goal time and training priority: `gccli events add --name "Spring 10K" --date 2026-05-10 --type running --race --distance 10km --goal 40m --training`
- Access token for a script: `gccli auth token`. It refreshes an expired token first. If the refresh fails, stdout is empty and the exit code is 3.

Strength training workflow (LLM-assisted exercise matching)

When a user wants to log a strength training activity with exercises described in free text (e.g. "Bench Press 3x12@20kg, Lat Pulldowns 3x12@41kg"), follow this workflow:

1. Fetch the Garmin exercise catalog: `gccli exercises list --json`
2. Map each user-described exercise to the best matching Garmin CATEGORY/EXERCISE_NAME pair. The catalog uses SCREAMING_SNAKE_CASE (e.g. BENCH_PRESS/BARBELL_BENCH_PRESS, PULL_UP/WIDE_GRIP_LAT_PULLDOWN, LATERAL_RAISE/DUMBBELL_LATERAL_RAISE, LUNGE/DUMBBELL_LUNGE, CALF_RAISE/STANDING_BARBELL_CALF_RAISE). Use your best judgement to find the closest match.
3. Show the user the proposed mapping and ask for confirmation before proceeding.
4. Create the activity: `gccli activity create --name "Upper Body" --type strength_training --duration 34m --date 2024-06-15T19:01:00`
5. Extract the activity ID from the output (JSON: activityId field, table: "Created activity <id>").
6. Add exercise sets: `gccli activity exercise-sets set <id> -e "CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS]" [-e ...]`

Exercise set format: `CATEGORY/NAME:reps@weightkg[:dSECS][:rSECS]`
- CATEGORY: exercise category (e.g. BENCH_PRESS, PULL_UP, LATERAL_RAISE, LUNGE, CALF_RAISE)
- NAME: specific exercise name within category (e.g. BARBELL_BENCH_PRESS, WIDE_GRIP_LAT_PULLDOWN)
- reps: number of repetitions
- weightkg: weight in kg (e.g. 20 for 20kg, 41.5 for 41.5kg, 0 for bodyweight)
- :dSECS: optional set duration in seconds (e.g. :d30 for 30s)
- :rSECS: optional rest duration in seconds after this set (e.g. :r60 for 60s)

Each -e flag is one set. For 3 sets of 12 reps, use three -e flags with the same exercise.

Example full workflow:
```
gccli activity create --name "Upper Body" --type strength_training --duration 34m --date 2026-03-13T19:01:00 --json
# Extract activityId from JSON response
gccli activity exercise-sets set <id> \
  -e "PULL_UP/WIDE_GRIP_LAT_PULLDOWN:12@41:d30:r60" \
  -e "PULL_UP/WIDE_GRIP_LAT_PULLDOWN:12@41:d30:r60" \
  -e "PULL_UP/WIDE_GRIP_LAT_PULLDOWN:12@41:d30:r90" \
  -e "LATERAL_RAISE/DUMBBELL_LATERAL_RAISE:12@8:d25:r60" \
  -e "LATERAL_RAISE/DUMBBELL_LATERAL_RAISE:12@8:d25:r60" \
  -e "LATERAL_RAISE/DUMBBELL_LATERAL_RAISE:12@8:d25:r90" \
  -e "BENCH_PRESS/BARBELL_BENCH_PRESS:12@20:d30:r60" \
  -e "BENCH_PRESS/BARBELL_BENCH_PRESS:8@20:d35:r60" \
  -e "BENCH_PRESS/BARBELL_BENCH_PRESS:5@20:d40"
```

Notes

- Set `GCCLI_ACCOUNT=you@example.com` to avoid repeating `--account`.
- For scripting, use `--json`. `--plain` gives TSV only for the commands listed under Output. Pin `--units` when parsing `--plain` output so it does not follow the Garmin profile.
- Dates support `today`, `yesterday`, `3d`, `+30d`, or `YYYY-MM-DD`.
- `body weekly [DATE]` always covers the 52 weeks that end on DATE (default today), because Garmin accepts no other window length. Weeks without a weigh-in are left out, so the list can be empty. Weights in `--json` are grams.
- Tokens are stored in the OS keyring (macOS Keychain, Linux Secret Service). The file fallback encrypts tokens with a fixed passphrase built into gccli, so anyone who can read the files can read the tokens.
- Tokens auto-refresh on 401. A 429 is retried up to 3 times when `Retry-After` is 30 s or less; a longer wait is reported as exit code 5. A 5xx is retried up to 2 times for read requests (GET) only, with exponential backoff. A write that gets a 5xx fails at once; check the state before you run it again. An API request fails when no data moves for 60 s; a transfer that keeps moving has no time limit.
- For Garmin China accounts: `export GCCLI_DOMAIN=garmin.cn`.
- Exit codes: 1 other error, 2 usage error, 3 login needed (ask the user to run `gccli auth login`), 4 not found, 5 rate limited (wait, then retry). With `--json`, errors are a JSON object on stderr: `{"error":{"code":"...","status":429,"message":"..."}}`.
- Delete commands fail without a terminal unless `--force` is set. Confirm with the user first, then pass `--force`.
- Download formats: FIT (default), GPX, TCX, KML, CSV.
- Workout step format: `type:duration[@target:low-high]`. Types: warmup, run, recovery, cooldown; targets: pace (min:sec), hr (bpm), power (watts), cadence.
- `goals set` and `goals steps` change the goal from `--date` (default today) onward; the days before keep their goal. `gccli health summary <date> --json` shows the steps, floors and intensity minutes goals (`dailyStepGoal`, `userFloorsAscendedGoal`, `intensityMinutesGoal`).
- `wellness menstrual-log set` changes only the fields you give. Garmin replaces the whole day on a write, so gccli reads the log first and sends the other fields again. The values are lower-case kebab (`light`, `tender-breasts`, `no-discharge`), the lists are comma-separated, and an empty value (`""`) clears a field. `--no-ovulation` removes the ovulation mark. The writes need cycle tracking set up in Garmin Connect; without it, Garmin answers 400 "MCT settings were not found". `wellness menstrual-log` prints `{}` for a day without a log, and `delete` of such a day fails with exit code 1. `wellness menstrual-cycle` and `wellness menstrual-summary` take one date (default today) and print `{}` for a day outside a tracked cycle. `wellness pregnancy-summary` prints `{}` when there is no pregnancy.
- `workouts update` replaces the whole workout, so build the file from `gccli workouts detail <id>` and edit it. `workoutId` is set from the ID argument, so an `upload` payload works unchanged. The workout keeps its ID and its calendar entries.

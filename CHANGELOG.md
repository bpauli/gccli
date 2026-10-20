# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Breaking changes

In `--json` mode, a bad date or time in the commands below used to print the JSON error object on stderr (`{"error":{...}}`) after the login. It now prints the plain usage block on stderr with exit 2, like every other usage error: a usage error never prints the JSON error object, in any mode.

| Command(s) | Mode | Before | After |
| --- | --- | --- | --- |
| `body composition`, `body daily-weigh-ins`, `health body-battery view`, `health endurance-score`, `health events`, `health fitness-age`, `health floors view`, `health hill-score`, `health hr view`, `health hrv`, `health intensity-minutes view`, `health lifestyle`, `health max-metrics`, `health respiration view`, `health rhr`, `health sleep view`, `health spo2`, `health steps view`, `health stress view`, `health summary`, `health training-readiness`, `health training-status view`, `hydration view`, `nutrition food-log`, `nutrition meals`, `nutrition settings`, `reload` (#263) | all | bad date: exit 1 (or 3 without a login), after the login | bad date: exit 2 with the usage block, before the login |
| the same 27 commands (#263) | all | an empty positional date (`""`) means today | an empty positional date is a usage error (exit 2); omit the date for today |
| `workouts schedule add <id> <date>`, `workouts schedule list [<date>]`, `events add --date` (#263) | all | bad date: exit 1 with one error line | bad date: exit 2 with the usage block; the help placeholder of `events add --date` is `DATE` in place of `STRING` |
| `hydration add --date` (#263) | all | bad date: exit 1 (or 3 without a login), after the login | bad date: exit 2 with the usage block, before the login; the help placeholder is `DATE` in place of `""` |
| the same 4 commands (#263) | all | an empty date (`""`) means today | an empty date is a usage error (exit 2); omit the date for today |
| `activity create --date` (#263) | all | bad date or time: exit 1 (or 3 without a login), after the login | bad date or time: exit 2 with the usage block, before the login; the help names the formats, and its placeholder is `DATE-TIME` in place of `STRING` |
| `activity create --date` (#263) | all | an empty `--date` (`""`) means now | an empty `--date` is a usage error (exit 2); omit the flag for now |
| `body delete-weight --date`, `body delete-blood-pressure --date` (#263) | all | the date went into the request path unchecked, so a bad date was a Garmin error after the login (exit 1), and only `YYYY-MM-DD` worked; an empty `--date` (`""`) was `date: required` (exit 1, or 3 without a login) | bad or empty date: exit 2 with the usage block, before the login; `today`, `yesterday`, `3d` and `+30d` are now accepted; the help placeholder is `DATE` in place of `STRING` |
| `events add --time` (#263) | all | bad time: exit 1 with one error line, before the login | bad time: exit 2 with the usage block; the help placeholder is `HH:MM` in place of `STRING`, and a time such as `7:30` is sent as `07:30` |
| `events add --time` (#263) | all | an empty `--time` (`""`) means no start time | an empty `--time` is a usage error (exit 2); omit the flag for no start time |
| `health race-predictions view [<date>]` (#263) | all | took an unused date argument and ignored it | takes no argument; a date is a usage error (exit 2, `unexpected argument`). `health race-predictions` without a child still runs `view` |
| `events add --goal`, `events add --distance` (#263) | all | bad value: exit 1 with one error line, before the login | bad value: exit 2 with the usage block; the help placeholders are `GOAL` and `DISTANCE` in place of `STRING` |
| the same 2 flags (#263) | all | an empty value (`""`) means no goal or no target distance | an empty value is a usage error (exit 2); omit the flag for no goal or no target distance |
| `workouts create --step` (#263) | all | bad or empty step: exit 1 with one error line, before the login | bad or empty step: exit 2 with the usage block |
| `activity exercise-sets set --exercise` (#263) | all | bad exercise: exit 1 with one error line (`exercise N ("..."): ...`), before the login | bad exercise: exit 2 with the usage block (`--exercise: invalid exercise "...": ...`) |
| `activity exercise-sets set <id>`, `workouts update <id>` (#263) | all | non-numeric ID: exit 1 with one error line, before the login | non-numeric ID: exit 2 with the usage block; an ID such as `007` or `+7` is sent as `7` |
| `activity describe`, `activity rename`, `activity retype`, `activity privacy`, `activity favorite` and `activity unfavorite` `<id>` (#263) | all | non-numeric ID: exit 1 (or 3 without a login), after the login | non-numeric ID: exit 2 with the usage block, before the login; an ID such as `007` or `+7` is sent as `7` |
| `courses send <course-id> <device-id>` (#263) | all | non-numeric ID: exit 1 after the login and the course lookup request | non-numeric ID: exit 2 with the usage block, before the login, with no request; an ID such as `007` or `+7` is sent as `7` |
| `courses import --type` (#263) | all | unknown type: exit 1 with one error line that lists the types, before the login | unknown type: exit 2 with the usage block (`--type must be one of ...`); the help lists the types |

### Added

- **Challenges** - `challenges available` lists the badge challenges that you can join. `challenges completed` lists the badge challenges that you finished. (#210, #79)
- **Health statistics** - `health stress daily --start DATE --end DATE` shows daily stress statistics for a date range. `health steps monthly` and `health pushes monthly` show monthly step and wheelchair push totals (`--end`, default today, and `--months`, default 6). `health calories weekly` and `health pushes weekly` show weekly calorie and wheelchair push totals (`--end`, default today, and `--weeks`, default 4). `health lifetime-totals` shows your lifetime totals. (#72)
- **Heart rate, floors and respiration statistics** - `health hr daily`, `health floors daily` and `health respiration daily` show daily statistics for a date range (`--start` and `--end`, both required). `health hr weekly` and `health floors weekly` show weekly statistics (`--end`, default today, and `--weeks`, default 4). (#72)
- **Sleep** - `health sleep naps [DATE]` shows the naps of a date. `health sleep daily` shows daily sleep statistics for a date range of at most 28 days, and `health sleep overall` shows overall sleep statistics for a date range (`--start` and `--end`, both required). `health sleep weekly` shows weekly sleep statistics for the 52 weeks up to `--end` (default today). (#70)
- **Training status, acclimation and one-rep max** - `health training-status daily [DATE]` shows the daily training status of a date. `health acclimation` shows acclimation data for a date range (`--start` and `--end`, both required). `health one-rep-max [DATE]` shows the current estimated one-rep max values as of a date. (#73)
- **Power-to-weight and thresholds** - `health power-to-weight [DATE]` shows the latest power-to-weight data as of a date. `health thresholds cycling-ftp`, `health thresholds lactate-heart-rate`, `health thresholds lactate-speed` and `health thresholds power-to-weight` show each value for a date range, and `health critical-swim-speed range` shows critical swim speed for a date range (`--start` and `--end`, both required). (#74)
- **Hydration statistics** - `hydration daily` shows daily hydration statistics for a date range (`--start` and `--end`, both required). `hydration weekly` shows weekly hydration statistics (`--end`, default today, and `--weeks`, default 4). (#72)
- **Activity statistics** - `activities stats` shows your activity totals by activity type, and `activities stats monthly` shows them per month. (#72)
- **Activity comments and likes** - `activity comments <id>` lists the comments on an activity. It always sends `start=0&limit=20`, the values of the live check. `activity likes <id>` lists the likes of an activity. (#75)
- **Personal records** - `records types` lists the personal record types. `records activity <id>` lists the personal records set in an activity. (#87)
- **Badges** - `badges compare <display-name>` compares your badges with those of a connection. `badges leaderboard` shows the badge points leaderboard of your connections. (#92)
- **Virtual challenges** - `challenges virtual available`, `challenges virtual completed` and `challenges virtual in-progress` list the available, completed and in-progress virtual challenges. (#79)
- **Weekly weight** - `body weekly [DATE]` shows weekly weight summaries (minimum, maximum and average weight per week) for the 52 weeks that end on the date (default today). Garmin accepts only this window length. Table and `--plain` output show the weights in kg or lbs per `--units`, and `--json` prints the Garmin answer in grams. (#93)
- **Social** - `social requests` lists pending connection requests. `social following [<display-name>]` lists the users that a user follows (default: you). `social feed` shows your newsfeed. (#77)
- **Devices** - `devices historical` lists historical devices. `devices messages` lists device messages. (#85)
- **Wellness** - `wellness jet-lag status [DATE]` shows the jet lag status of a date (default today). `wellness menstrual-calendar` shows the menstrual cycle calendar for a date range (`--start` and `--end`, both required). (#94)
- **Training** - `training coach-workouts` lists adaptive coaching workout types. (#83)
- **Notifications** - `notifications preferences` shows your notification preferences. (#86)
- **Gear writes** - `gear add <name> --brand --model` adds a gear item (`--type` shoes, bike or other, default shoes; `--notes` defaults to the name, because Garmin requires notes). `gear update <uuid>` changes `--name`, `--brand`, `--model` or `--notes`. `gear retire <uuid>` and `gear unretire <uuid>` change the gear status, and `gear delete <uuid>` deletes a gear item after a prompt (`--force` skips it). These commands take the UUID with or without dashes. (#69)
- **Gear collection writes** - `gear collections create <name>` creates a gear collection (`--first-use`, default today). `gear collections rename <uuid> <name>` renames it, and `gear collections add <uuid> <gear-uuid>` adds a gear item to it (the gear UUID with or without dashes). `gear collections delete <uuid>` deletes a collection after a prompt (`--force` skips it); its gear items stay. (#69)
- **Activity edits** - `activity describe <id> <text>` sets the description of an activity, and an empty text (`""`) clears it. `activity privacy <id> <level>` changes who can see the activity (`public`, `private` or `connections`). `activity favorite <id>` and `activity unfavorite <id>` add the activity to your favorites and remove it. (#76)
- **Workout copy** - `workouts copy <id> [<name>]` copies a workout and prints the ID of the copy. Without a name, the copy gets the name of the original. (#81)
- **Goal writes** - `goals set <type> <value>` sets the daily goal for `floors-climbed` (floors), `intensity-minutes` (minutes) or `hydration` (milliliters). `goals steps set <steps>` sets a fixed daily steps goal, and `goals steps auto` sets Garmin's automatic steps goal. Each goal starts on `--date` (default today) and stays for the days after it. (#68)
- **Calendar notes** - a new `calendar` group. `calendar notes add <date> <name> [--content TEXT]` adds a note to a calendar day and prints its ID. `calendar notes update <id>` changes `--name` or `--content` (an empty `--content` clears the text). `calendar notes move <id> <date>` moves a note to another day, and `calendar notes delete <id>` deletes a note after a prompt (`--force` skips it). (#82)
- **Nap writes** - `health sleep naps add <start> <end>` adds a nap for today. The times are HH:MM in the time zone of your Garmin profile, and the nap must end by now, because Garmin takes only naps that end today. `health sleep naps delete <date> <start>` deletes the nap that starts at that minute after a prompt (`--force` skips it). (#70)
- **Notification preference writes** - `notifications preferences set <key> <true|false>` turns one notification preference on or off. It reads the preferences, changes the key and sends the full object back. A key that is not in the preferences, or that is not true or false, fails before any write, and the error lists the keys. (#86)
- **Privacy zone writes** - `profile privacy-zones add --lat --lon --radius` adds a privacy zone and prints its ID. The radius is 500 to 1500 meters, and gccli checks it, the latitude and the longitude before it sends the request. `profile privacy-zones delete <id>` deletes a zone after a prompt (`--force` skips it). (#88)
- **Menstrual daily log** - `wellness menstrual-log [<date>]` shows the menstrual daily log of a day (default today), and `{}` for a day without a log. `wellness menstrual-log set <date>` changes `--flow`, `--symptoms`, `--moods`, `--discharge`, `--sex-drive`, `--sexual-activity`, `--notes` or `--ovulation` (`--no-ovulation` removes the mark). The values are lower-case kebab, such as `light` or `tender-breasts`, the lists are comma-separated, and an empty value clears a field. Garmin replaces the whole day on a write, so `set` reads the log first and sends the fields that you do not give unchanged. `wellness menstrual-log delete <date>` deletes the log after a prompt (`--force` skips it), and fails when the day has no log. The writes need cycle tracking set up in Garmin Connect. (#94)

### Changed

- **Health** - `health hr`, `health floors` and `health respiration` are now command groups. The old command is the default child `view`, so `health hr [<date>]` still works and gives the same output and exit codes. The help and the shell completion now list the children. (#72)
- **Sleep** - `health sleep` is now a command group. The old command is the default child `view`, so `health sleep [<date>]` still works and gives the same output and exit codes. The help and the shell completion now list the children. (#70)
- **Training status** - `health training-status` is now a command group. The old command is the default child `view`, so `health training-status [<date>]` still works and gives the same output and exit codes. The help and the shell completion now list the children. (#73)
- **Critical swim speed** - `health critical-swim-speed` is now a command group. The old command is the default child `view`, so `health critical-swim-speed [<date>]` still works and gives the same output and exit codes. The help and the shell completion now list the children. For a bad date, the usage block above the error line now names `health critical-swim-speed view`. (#74)
- **Personal records** - `records` is now a command group. The old command is the default child `list`, so `records` still works and gives the same output and exit codes. The help and the shell completion now list the children. For an extra argument, the usage line above the error now names `records list`. (#87)
- **Naps** - `health sleep naps` is now a command group. The old command is the default child `view`, so `health sleep naps [<date>]` still works and gives the same output and exit codes. The help and the shell completion now list the children. For a bad date, the usage block above the error line now names `health sleep naps view`. (#70)
- **Notification preferences** - `notifications preferences` is now a command group. The old command is the default child `view`, so `notifications preferences` still works and gives the same output and exit codes. The help and the shell completion now list the children. For an extra argument, the usage line above the error now names `notifications preferences view`. (#86)

## [2.3.0] - 2026-10-19

### Added

26 new read commands. Each one prints the Garmin response as JSON in table, `--plain` and `--json` mode. A date argument or a date flag accepts `YYYY-MM-DD`, `today`, `yesterday`, `3d` and `+30d`.

- **Activities** - `activities pending` lists the pending activities. (#219, #76)
- **Badges** - `badges detail <id>` shows the details of one badge. (#209, #92)
- **Body** - `body latest [<date>]` shows the latest weigh-in as of a date. `body glucose [<date>]` shows the glucose readings for a date. Both dates default to today. (#228, #93, #74)
- **Devices** - `devices sensors` lists the paired sensors. `devices backups` shows the device backup summary. (#225, #85)
- **Events** - `events upcoming` lists the upcoming events. (#230, #82)
- **Gear** - `gear types` lists the gear types. `gear collections list` lists the gear collections, and `gear collections` runs it too. (#226, #69)
- **Goals** - `goals weight` shows the effective weight goal for a date range. `--start` and `--end` default to today. (#219, #68)
- **Groups** - The new `groups` command group. `groups list` lists your groups, and `groups` runs it too. (#224, #78)
- **Health** - `health power-curve --start DATE --end DATE` shows the power curve for a date range. Both flags are required. `health running-economy [<date>]` shows the latest running economy as of a date, and `health critical-swim-speed [<date>]` shows the latest critical swim speed as of a date. Both dates default to today. (#219, #73, #74)
- **Notifications** - The new `notifications` command group. `notifications list` lists the notifications, and `notifications` runs it too. (#222, #86)
- **Profile** - `profile privacy view` shows the privacy settings, and `profile privacy` runs it too. `profile privacy-zones list` lists the privacy zones, and `profile privacy-zones` runs it too. `profile hr-zones` shows the heart rate zones. `profile power-zones` shows the power zones for all sports. (#227, #88, #74)
- **Social** - The new `social` command group. `social connections` lists your connections, 20 at a time. Use `--limit` (`-l`) and `--offset` to get more. `social followers [<display-name>]` lists the followers of a user. Without a display name, it lists your followers. (#223, #77)
- **Training** - `training active` shows the active training plans. `--start` sets the start date and defaults to today. (#219, #83)
- **Wellness** - `wellness jet-lag list` lists the jet lag trips, and `wellness jet-lag` runs it too. (#219, #89)
- **Workouts** - `workouts types` lists the workout types. `workouts schedule export` exports the workout schedule from the `--start` date, today by default. `workouts benchmarks list` lists the strength benchmarks, and `workouts benchmarks` runs it too. (#229, #81)

## [2.2.0] - 2026-10-18

### Added

- **`--header` prints the column names in plain output** - With `--plain`, the first line of a list holds the column names, the same names as the table header. `GCCLI_HEADER=1` does the same in plain mode, and table and JSON output ignore it. `activity summary` has no column names, so it prints no header line. `--header` on the command line without `--plain`, or with `--json`, fails with `--header works only with --plain` and exit code 2. (#188, #99)

### Changed

- **Table and plain output escape values** - A backslash prints as `\\`, a tab as `\t`, a newline as `\n`, a carriage return as `\r`, a vertical tab as `\v` and a form feed as `\f`. Before, a course or activity name with a tab or a newline split its record into more fields or more lines, and a script that read `--plain` output got the wrong columns. A value that holds a backslash also changes, because the backslash is now doubled. `--json` output does not change. (#188, #99)
- **IDs in the request path are escaped** - Every `activities`, `activity`, `courses`, `workouts`, `body`, `devices`, `gear`, `events`, `hydration`, `nutrition` and `profile` command escapes the IDs and values that it puts in the URL, as `records` and `training` do since v2.1.0. A value with `/`, `?`, `#`, `&` or `%` now reaches Garmin as one value. For example, `activity summary a/b` sends `/activity/a%2Fb`, and `gear stats 'a#b'` sends `/stats/a%23b`. Before, such a value changed the URL or cut it short. An empty ID, `.` and `..` fail with exit code 1 before any request, and the error names the missing or bad parameter, for example `activityID: required`. Before, the request went to Garmin, and the exit code came from Garmin's answer. Numeric IDs, dates, page sizes and activity types send the same request as before. (#183, #186)

### Fixed

- `devices alarms` printed an empty list for a device with a real Garmin device ID. The device ID went through a float and reached Garmin as `3.44203243e+09`, so the settings request failed and the loop skipped the device. The ID now keeps its digits, and the request goes to `.../device-info/settings/3442032430`. (#185)

## [2.1.0] - 2026-10-18

v2.1 keeps the stored login safe when several gccli processes run at the same time, and it checks command input before it sends a request. No flag or command name changes. Read the `Changed` list if a script checks the exit code of `auth token` or passes `events add --time` with seconds.

### Changed

- **Token refresh waits for other gccli processes** - When two or more commands start with an expired token, only one of them refreshes it. The others wait for it, then use the refreshed token. Before, the second command could fail with exit code 3, because Garmin accepts a refresh token only once. A wait longer than one second prints `Waiting for another gccli process to refresh the token` on stderr. The lock is the new empty file `token.lock` in the config directory. (#158, #179)
- **`auth token` refreshes an expired token** - It prints the new token. When the refresh fails, it exits with 3 and prints nothing on stdout. Before, it printed the expired token, wrote a warning and exited with 0, so a script used a token that Garmin rejects. (#161)
- **The stored token has a version** - `auth export` output gains `"version": 1`. A token with another version fails before any request, and the error names both versions. Tokens from v2.0.0 still import and refresh, and v2.0.0 can read a token from v2.1.0. The unused `display_name` field is removed. (#171)
- **Refresh errors name the cause** - A failed refresh now prints `Token expired: <cause>. Run: gccli auth login <email>`. Before, a refused connection and a Garmin 503 both printed only `Token expired`. The cause quotes at most 512 bytes of the Garmin response and says how many bytes it cut. Before, a long HTML error page went into the `--json` error object. The error code and exit code do not change. (#179, #182)
- **Command input is checked before the request** - `workouts create`, `activity exercise-sets set` and `events add` reject NaN, Inf, numbers that overflow and empty values, with exit code 1 and no request. The error names the value as you typed it. Before, some of these values went to Garmin, and others failed with `json: unsupported value`. (#175)
  - `workouts create` rejects an empty step list.
  - `activity exercise-sets set` rejects a negative rep count and an empty category or name.
  - `events add --time` accepts only `H:MM` or `HH:MM`, from `00:00` to `23:59`. Values such as `07:30:00`, `0730` and `7:30pm` now fail. A script that passes seconds must remove them.
- **Values in the request path are escaped** - `records`, `training plan` and `training plans --locale` escape the values that they put in the URL. A value with `/`, `?`, `#`, `&` or `%` now reaches Garmin as one value. Before, such a value changed the URL or cut it short. An empty plan ID, `.` and `..` fail with exit code 1 before any request. (#170, #178)

### Fixed

- A `garmin.cn` account could not refresh its token, because every refresh went to `diauth.garmin.com`. Login, refresh and API calls now use one domain: the token domain, then the config, then `garmin.com`. `auth login` rejects an unknown domain with exit code 2 before the password prompt, and `auth import` rejects a token with an unknown domain. (#150)
- A stored token with an unknown domain now fails with `stored token for <email> has unknown Garmin domain "<domain>"; run: gccli auth login <email>`. Before, the message did not name the token, and `auth status` and `auth token` showed the token as valid. (#177, #179)
- A refreshed token that gccli cannot save no longer fails the command. The command uses the token and prints `Could not save the refreshed token: <error>` as a warning. A read-only config directory no longer stops a refresh. (#179)
- With the `file` keyring, a command that read the token while another process wrote it could fail with `jwt.DecodeBytes() expects token of 3 or 5 parts`. gccli now writes the token file in a temporary directory and renames it over the old file. A temporary directory that a killed write leaves behind is removed by the next write and by `auth remove`. On Windows gccli does not remove it. (#179, #182)
- `auth login` and `auth import` rewrote `config.json` and lost the keys that they do not know. A write that stopped midway left a cut file. Config writes now keep every key, write a temporary file and rename it, and hold a lock on `config.lock`. (#143)
- `courses import` without `--json` exited with 1 when Garmin saved the course but sent an empty response. It now prints `Imported course from <file>` on stderr and exits with 0. (#169)
- `gear list` and `gear defaults` sent any `profileId` value to Garmin, also `null` or an object. A `profileId` that is not an unsigned integer now fails with exit code 1 before the request. (#174)

### Docs

- The command reference in the README, the skill and the docs site is now generated from the command tree, so it matches the binary. `make test` fails when it is out of date. (#163)
- `docs/state.md` describes every file and keyring item that gccli stores, the lock files and the token format. The README links it. (#171, #179, #182)
- The README no longer has the Credits section. (#155)

## [2.0.0] - 2026-10-15

v2.0 gives each flag one name and each write command one output rule. Scripts that use the old flag names or parse write output must change. A command with an old flag name fails with exit code 2 and names the flag, so a broken script fails loudly. The tables below list every change.

### Changed (breaking)

- **One name for each date and pagination flag** - Date ranges use `--start` and `--end` in every command. Paginated commands use `--limit` (`-l`) and `--offset`. The old names are removed, with no aliases. (#99, #133)

  | Command | Old flag | New flag |
  |---|---|---|
  | `activities list`, `activities search`, `workouts list` | `--start N`, `-s N` | `--offset N` |
  | `events list` | `--page P`, `-p P` | `--offset (P-1)*LIMIT` |
  | `events list` | `-s ORDER` | `--sort ORDER` |
  | `activities search`, `wellness menstrual-cycle`, `wellness menstrual-summary` | `--start-date`, `--end-date` | `--start`, `--end` |

  - `--offset` counts items from 0, as `--start` did. In `events list`, `--offset` must be 0 or a multiple of `--limit`, because Garmin returns events in pages of `--limit`.
  - Each short flag has one meaning. `-s` means only `--step` (`workouts create`), and `-p` means only `--privacy` (`courses import`). `--offset`, `--sort` and the date flags have no short flag.
- **One `--json` output rule for write commands** - With `--json`, every write command prints the Garmin response on stdout, or `{"ok": true}` when the response is empty. Without `--json`, it prints one message on stderr and nothing on stdout. `courses import` keeps its table and TSV row on stdout. (#99, #135, #138)

  | Command | Mode | Before | After |
  |---|---|---|---|
  | `activity rename`, `activity retype`, `workouts update`, `body add-weight`, and any write that gets an empty response | `--json` | `null` | `{"ok": true}` |
  | `activity delete`, `body delete-weight`, `body delete-blood-pressure`, `workouts delete`, `workouts schedule remove`, `courses delete`, `events delete` | `--json` | nothing on stdout, a message on stderr | the Garmin response on stdout, nothing on stderr |
  | `activity exercise-sets set`, `gear link`, `gear unlink` | `--json` | nothing on stdout, a message on stderr | the Garmin response on stdout, nothing on stderr |
  | `hydration add` | table, `--plain` | the Garmin response as JSON on stdout | `Added <ml> ml of water on <date>` on stderr, nothing on stdout |
  | every write command | `--json`, a 2xx response that is not JSON | exit 1 with `json: error calling MarshalJSON ...` | exit 1 with `garmin saved the change, but its response is not JSON: ...` |

  A script that reads the result of a write command should pass `--json` and read stdout. The delete prompt, a declined prompt and a delete without a terminal and without `--force` do not change.
- **Bad flag values are usage errors** - These inputs now exit with code 2 before gccli sends a request. (#133, #134, #136)
  - A bad date in any `--start` or `--end` flag. Before, most range commands sent the value to Garmin, and `events list` and `workouts schedule list` exited with 1.
  - An empty date value, required or optional, such as `--start ""`. Before, it was sent as today. A positional date with no value, such as `health summary`, still means today.
  - `--offset` below 0 or `--limit` below 1 in any paginated command. Before, only `events list` checked `--offset`.

### Added

- Every `--start` and `--end` flag accepts `today`, `yesterday`, `3d` (3 days ago) and `+30d` (30 days from today). Before, only `events list` and `workouts schedule list` did. (#133)

### Fixed

- `health rhr` failed with 404 and `records` failed with 403. No login stored the Garmin display name, so the commands sent an empty name or the email. The six commands that need it (`health summary`, `steps`, `hr`, `rhr`, `sleep` and `records`) now get it from the profile. (#129, #137)
- An API request now fails when no data moves for 60 s, also while it reads the response body or sends an upload. Before, a body that stalled after the headers kept the command waiting forever. A transfer that keeps moving has no time limit. The idle time uses the monotonic clock, so a change of the system clock cannot fail a transfer or hide a stall. (#110, #114, #117, #121)
- `auth remove` on the file keyring exited with 1 when no credentials were stored, so a script that ran it twice failed. It now acts as it does on the other keyrings. (#107, #112)
- `activity create` printed a large activity ID as `2.123456789e+10`. It now prints the digits. (#138)
- Bash completion: (#113, #115, #116, #118, #119, #120, #122, #123, #124, #125)
  - It completes after a flag value and after a positional argument.
  - `--flag=value` completes, and values with `@` or `=` no longer break it.
  - Each flag offers the values of its own command.
  - Quoted words complete, and non-ASCII text works on bash 3.2.
  - An escaped space no longer deletes typed text.
  - TAB stays fast on long lines.

### Removed

- Unused code: `outfmt.Write`, `ui.FromContext`, the table helpers in `internal/cmd`, most of `internal/testutil` and the unused OAuth1 preauthorized exchange. (#99, #130)

### Docs

- README, the skill and the docs site use the new flag names and state the output rule for write commands. (#133, #138, #140)

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

[Unreleased]: https://github.com/bpauli/gccli/compare/v2.3.0...HEAD
[2.3.0]: https://github.com/bpauli/gccli/releases/tag/v2.3.0
[2.2.0]: https://github.com/bpauli/gccli/releases/tag/v2.2.0
[2.1.0]: https://github.com/bpauli/gccli/releases/tag/v2.1.0
[2.0.0]: https://github.com/bpauli/gccli/releases/tag/v2.0.0
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

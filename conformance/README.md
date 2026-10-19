# Conformance suite

This suite runs a built `gccli` binary as a black box. It compares stdout, stderr, the exit code and the requests that the binary sends with golden files, and it checks the token and config state that the binary leaves behind. Use it to check a release build, or any binary that claims to behave like `gccli`.

## Run it

Build the binary, then give its path in `GCCLI_BIN`:

```bash
make build
make conformance GCCLI_BIN=bin/gccli
```

`make conformance` builds the tests with `-tags=conformance`. With that tag, the suite fails when `GCCLI_BIN` is not set or does not name an executable file. Without the tag, as in `make test`, the suite skips when `GCCLI_BIN` is not set.

The suite runs only on Linux. The binary must trust the mock CA through `SSL_CERT_FILE`, and Go on macOS does not read that variable reliably. On other systems the suite skips.

## How it works

Each binary run gets its own environment, built from nothing:

- `HOME` and `XDG_CONFIG_HOME` point to the same new temp directory, so the run never reads or writes your real config or keyring.
- `GCCLI_KEYRING_BACKEND=file` keeps the token in that directory.
- `HTTPS_PROXY` and `HTTP_PROXY` point to a CONNECT proxy in the test process. The proxy never dials out. It ends TLS for every Garmin host itself, with a certificate from a CA that the suite makes for each run.
- `SSL_CERT_FILE` names that CA, and `SSL_CERT_DIR` names a directory that holds only that CA. The binary does not trust the system roots, so a binary that ignores the proxy cannot reach the real Garmin. `TestSandboxDistrustsSystemRoots` checks this.
- `TZ=UTC` fixes the time zone.

The binary uses its real URLs. Nothing overrides a base URL.

## Golden cases

`TestGolden` reads every file in `internal/cmd/testdata/golden`, the same files that the in-process test in `internal/cmd/golden_test.go` uses. Each file holds the args, the stdin, the input files, the routes that the mock answers, and the expected output in table, plain and JSON mode. To add or change a case, edit those files as `internal/cmd/golden_test.go` describes. This suite has no cases of its own. It also runs the case files `<leaf>@<case>.golden`, and a skip for a leaf also skips its case files.

The suite runs each case with `--account test@example.com --units metric`, like the in-process harness. It leaves `--color` at its default, because output to a pipe must have no color. When a case has stdin, the suite gives it a pseudo-terminal, so a confirmation prompt reads the answer as typed input.

A case fails when the output differs, when the requests differ from the `<mode> requests` section, or when the binary sends a request that the file has no route for. The suite and the in-process harness write the requests with the same function, `requestlog.Format` in `internal/testutil/requestlog`. A route matches the method, the host and the URI. The golden files do not name a host, so the suite expects `connect.garmin.com` for the `/web-data/` routes and `connectapi.garmin.com` for all other routes. A call that moves to another host fails.

The suite skips these cases:

- `auth login`, because the in-process harness stubs the browser login.
- `body glucose`, `body latest`, `devices solar`, `events list`, `goals weight`, `health critical-swim-speed`, `health running-economy`, `health steps weekly`, `health stress weekly` and `training active`. Their routes ask for a date that the args do not give. The binary takes that date from its clock, and the in-process harness pins its clock to 2026-10-15.

The suite compares the output of `body add-blood-pressure`, `body add-composition` and `body add-weight`, but not their requests. Their request bodies hold a time that the binary takes from its clock. The `clockBodies` map holds this list, and a case in it fails when its requests match, because then the entry is stale.

The `skipped` map in `conformance_test.go` holds the list of skipped cases. `TestGoldenSkips` fails when a golden case asks for a date from the clock and is not in the list, when a case in the list no longer asks for one, and when the list names a case that has no golden file. So a new skip cannot pass unseen. This test does not need `GCCLI_BIN`.

## State cases

`state_test.go` checks state that only a real process can show. The token and config files for these cases are in `testdata/state`.

| Test | What it checks |
| --- | --- |
| `TestRefreshRace` | Two processes start with the same expired token. Only one refresh reaches Garmin, and both processes use the refreshed token. |
| `TestAuthTokenRefresh` | `auth token` refreshes an expired token, prints it, and saves it. |
| `TestDomainChinaRefresh` | A `garmin.cn` token refreshes at `diauth.garmin.cn`, calls `connectapi.garmin.cn`, and keeps its domain. |
| `TestDomainFromConfig` | A token without a domain takes the domain from `config.json`. |
| `TestConfigWriteKeepsKeys` | `auth import` keeps the `config.json` keys that it does not set, also unknown keys. |

The mock token service rotates refresh tokens like Garmin: a refresh token that comes back a second time gets `400 invalid_grant`.

## Recorded responses

The golden routes are written by hand. `testdata/recorded` holds Garmin responses in the shape that the e2e suite records, so the suite can also check that the binary still reads what Garmin sends.

### Record

When `GCCLI_RECORD_DIR` is set, the e2e suite writes each API response to its own file in that directory, one file per request, as a `Record` (`scrub.go`): the method, the host, the URI, the status and the JSON body. The recorder writes no request header and no request body. A response that is not JSON keeps only its size. Login and token refresh do not go through the recording client, so they are never recorded.

```bash
GCCLI_RECORD_DIR=/tmp/rec make test-e2e
```

### Scrub

`go run ./conformance/cmd/scrub <record dir> <out dir>` replaces every value in each record with a fixed one and writes the result to a new directory. Records that are the same after the scrub are written once.

The scrubber keeps no value, so it never has to guess which value is safe:

- A string becomes `tester`. A string of digits becomes `1`, and a date or time keeps its format with fixed digits, such as `2026-01-01` or `2026-01-01T00:00:00.0`. The format of the string decides, not its key.
- A number becomes `1`, or `0.5` when it has a fraction. A bool becomes `true`. `null` stays.
- An array keeps at most one element, which merges the scrubbed form of every element: it holds every key that any element has, and a value where any element has one. `[]` stays `[]`, and an array that holds only `null` becomes `[null]`. So the length of a series, such as one entry per minute of sleep, the position of each `null`, such as a gap while the watch was off, and which element has a key do not survive. This also holds for an array of scalars, such as a `[time, value]` pair, which becomes `[1]`: no command reads such a pair by index, and one that does fails the replay instead of leaking the length.
- An object key stays when it is an identifier, lower camelCase such as `calendarDate` or UPPER_SNAKE such as `WELLNESS_RESTING_HEART_RATE`. A key that is a date, such as `2026-10-17`, becomes `2026-01-01`, so a map keyed by date keeps its shape. Any other key fails the scrub.
- The path must match a template in `scrubPaths`. A `{name}` segment is a value and is replaced like a string. A query key must be an identifier, and its value is replaced like a string.

Two records that differ only in their values, in the length of an array, in the position of a null or in which element of a series has a key scrub to the same bytes. The replay checks shapes, so the fixed values and one element per array are enough for it. What survives is the set of keys in each response and the JSON type of each value, with two residuals: a field whose value is `null` in every element stays `null`, and a number keeps whether it had a fraction, because a command that reads an integer field must still get an integer.

The scrubber fails closed. A key that is not an identifier or a date, an unknown path or a host that is not a Garmin API host fails the scrub, and nothing is written. The error names the route and the position of the key, never the key or a value, because the job log is public. The position holds only structure: `$` is the body, `[]` an element of an array, `<key 2>` the third key of an object in byte order, and `?<key 1>` the second query pair. To fix it, record the same route locally, find the key at that position in any element, and extend the key rule or the path table with a case in `scrub_test.go`.

### Replay

These tests read `testdata/recorded`, or the directory that `GCCLI_RECORDED_DIR` names:

| Test | What it checks |
| --- | --- |
| `TestRecordedScrubbed` | Each file is already scrubbed: the scrubber does not change it. |
| `TestRecordedShapes` | For each golden case in `caseReads`, every recorded route answers 2xx with what the command reads from it in table and plain mode: the fields and their JSON types, or the top-level kind for a command that prints the body as it is. The test fails when a replayable case has no entry. It does not need `GCCLI_BIN`. |
| `TestRecordedReplay` | Each golden leaf case whose routes all have a 2xx recording checks its `caseReads` entry, then runs in table mode against the recorded responses and must exit 0. A case file `<leaf>@<case>.golden` is not replayed: it runs the leaf's command on other mock data, and the recording replaces that data. |

A request matches a recording by method, host and path. The query is not compared, and a number or date segment matches any number or date.

### The scheduled job

The daily scheduled e2e run records, scrubs and replays inside the runner, so a change in what Garmin sends turns the job red. It uploads nothing and commits nothing. The scrubbed set still shows which keys each response of the account holds, and the repository is public, so nothing derived from the account leaves the runner. The job log names only structure: a route, a position, a file name, a command and an exit code.

### Refresh the recorded set

A maintainer refreshes `testdata/recorded` on their own machine:

```bash
GCCLI_RECORD_DIR=/tmp/rec make test-e2e
go run ./conformance/cmd/scrub /tmp/rec /tmp/scrubbed
rm -r conformance/testdata/recorded && mv /tmp/scrubbed conformance/testdata/recorded
make build && make conformance GCCLI_BIN=bin/gccli
```

Review the diff file by file before you commit. The repository is public, and the scrubber keeps the keys of each response, so a key that names something about you must not go in. Commit the set in a pull request.

The committed set comes from the e2e suite against a mock with synthetic responses. It holds no data from a real account.

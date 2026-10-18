# Conformance suite

This suite runs a built `gccli` binary as a black box. It compares stdout, stderr and the exit code with golden files, and it checks the token and config state that the binary leaves behind. Use it to check a release build, or any binary that claims to behave like `gccli`.

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

`TestGolden` reads every file in `internal/cmd/testdata/golden`, the same files that the in-process test in `internal/cmd/golden_test.go` uses. Each file holds the args, the stdin, the input files, the routes that the mock answers, and the expected output in table, plain and JSON mode. To add or change a case, edit those files as `internal/cmd/golden_test.go` describes. This suite has no cases of its own.

The suite runs each case with `--account test@example.com --units metric`, like the in-process harness. It leaves `--color` at its default, because output to a pipe must have no color. When a case has stdin, the suite gives it a pseudo-terminal, so a confirmation prompt reads the answer as typed input.

A case fails when the output differs, or when the binary sends a request that the file has no route for. A route matches the method, the host and the URI. The golden files do not name a host, so the suite expects `connect.garmin.com` for the `/web-data/` routes and `connectapi.garmin.com` for all other routes. A call that moves to another host fails.

The suite skips these cases:

- `auth login`, because the in-process harness stubs the browser login.
- `devices solar`, `events list`, `health steps weekly` and `health stress weekly`. Their routes ask for a date that the args do not give. The binary takes that date from its clock, and the in-process harness pins its clock to 2026-10-15.

The `skipped` map in `conformance_test.go` holds this list. `TestGoldenSkips` fails when a golden case asks for a date from the clock and is not in the list, when a case in the list no longer asks for one, and when the list names a case that has no golden file. So a new skip cannot pass unseen. This test does not need `GCCLI_BIN`.

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

`go run ./conformance/cmd/scrub <record dir> <out dir>` replaces the personal data in each record and writes the result to a new directory. Records that are the same after the scrub are written once.

Two tables in `scrub.go` decide what happens:

- `scrubPaths` lists every request path that the scrubber accepts. A `{name}` segment holds a value, such as a display name or an id, that the rules classify under that name. A path that matches no template fails.
- `scrubRules` classifies every value in the body, the query and the path. The first rule that matches decides. Every number becomes `1`, or `0.5` when it has a fraction, so ids, coordinates and health values do not stay. Emails, display names, names, string ids, serials, tokens, URLs, free text, time zones and dates become fixed test values. Enum values stay.

The scrubber fails closed. A string that no rule classifies, an unknown path or a host that is not a Garmin API host fails the scrub, and nothing is written. The error names each failing value by its route and JSON path, such as `GET /userprofile-service/userprofile/settings: userData.nickname`, and never prints the value. To fix it, add a rule or a path template, and add a case to `scrub_test.go`.

### Replay

These tests read `testdata/recorded`, or the directory that `GCCLI_RECORDED_DIR` names:

| Test | What it checks |
| --- | --- |
| `TestRecordedScrubbed` | Each file is already scrubbed: the scrubber does not change it. |
| `TestRecordedShapes` | Each route in `readShapes` has a 2xx recording with the JSON shape that the commands which read it need. A failure names those commands, such as `goals list`. It does not need `GCCLI_BIN`. |
| `TestRecordedReplay` | Each golden case whose routes all have a 2xx recording runs in table mode against the recorded responses and exits 0. |

A request matches a recording by method, host and path. The query is not compared, and a number or date segment matches any number or date.

### Refresh the recorded set

The daily scheduled e2e run records, scrubs, replays and uploads the scrubbed set as the workflow artifact `recorded-fixtures`, which holds `recorded-fixtures.tgz`. It never commits. To refresh `testdata/recorded`, download the artifact, read every file, then replace the directory with its `recorded` folder in a pull request.

The committed set comes from the e2e suite against a mock with synthetic responses. It holds no data from a real account.

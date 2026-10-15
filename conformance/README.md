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

A case fails when the output differs, when the binary sends a request that the file has no route for, or when a request goes to a host outside `garmin.com`.

The suite skips these cases:

- `auth login`, because the in-process harness stubs the browser login.
- Each case whose routes ask for a date that the args do not give. The binary takes that date from its clock, and the in-process harness pins its clock to 2026-10-15.

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

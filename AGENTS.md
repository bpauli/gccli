# AGENTS.md

Rules for agents that change this repository. Build, style, test and documentation rules are in `CLAUDE.md`.

## Verify every PR before it merges

Each PR gets one independent verifier: an agent that did not write the change. The verdict decides the merge. CI and passing tests do not replace it.

The verifier:

1. Checks out the exact head SHA in its own worktree, and the base SHA in a second one when it compares behavior.
2. Runs the gate on the head: `make fmt-check`, `make lint`, `go vet -tags e2e ./...` and `go test -race ./...`.
3. Runs the real binary from base and head (`go build -o bin/gccli ./cmd/gccli`) against a local HTTPS mock. Point `HTTPS_PROXY` and `SSL_CERT_FILE` at the mock, and use an isolated `HOME` with `GCCLI_KEYRING_BACKEND=file` and a token from `auth import`. For a refactor, stdout, stderr and the exit code must match base byte for byte. For a behavior change, every difference must be one the PR names.
4. Injects the defect that each new test claims to catch, and confirms the test fails.
5. Reads every changed line and reruns the claims in the PR's Verification section that are cheap to rerun.
6. Makes live Garmin calls only when the account owner approves them for that PR.
7. Posts one PR comment that starts with `**Independent verification: PASS**`, `PASS+NOTES` or `FAIL`, and names the head and base SHA it checked.

Merge only on PASS or PASS+NOTES, with `gh pr merge <n> --merge --match-head-commit <verified head>`. When `main` moved after the verdict, build and test `main` plus the PR before the merge. A new head needs a new verdict. A FAIL goes back to a fresh fix round and then a fresh verifier.

## Keep the stable output contract

- `--json` output, exit codes and the JSON error object on stderr are the stable contract. Change them only in a major release, with a migration table in `CHANGELOG.md`.
- Default table and `--plain` output may change in a minor release. List each change under `### Changed` in `CHANGELOG.md`. Scripts use `--json`.

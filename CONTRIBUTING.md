# Contributing to gccli

Thank you for your interest in contributing to gccli! This guide will help you get started.

## Prerequisites

- [Go](https://go.dev/dl/) 1.26 or later
- [Make](https://www.gnu.org/software/make/)
- Git

## Getting Started

1. Fork the repository and clone your fork:

   ```bash
   git clone https://github.com/<your-username>/gccli.git
   cd gccli
   ```

2. Install development tools:

   ```bash
   make tools
   ```

3. Verify everything works:

   ```bash
   make ci
   ```

## Development Workflow

1. Create a feature branch from `main`:

   ```bash
   git checkout -b feat/my-feature
   ```

2. Make your changes.

3. Format, lint, and test before committing:

   ```bash
   make fmt
   make lint
   make test
   ```

   All three checks must pass before submitting a PR.

4. Commit your changes using [Conventional Commits](#commit-messages).

5. Push and open a pull request.

## Code Style

### Formatting

Code is formatted with `goimports` and `gofumpt`. Run `make fmt` to auto-format.

### Import Order

Imports must follow this order, separated by blank lines:

1. Standard library
2. External packages
3. Local packages (`github.com/bpauli/gccli`)

### Linting

The project uses `golangci-lint` with the configuration in `.golangci.yml`. Run `make lint` to check.

## Commit Messages

This project follows [Conventional Commits](https://www.conventionalcommits.org/):

- `feat:` — New feature
- `fix:` — Bug fix
- `refactor:` — Code refactoring (no behavior change)
- `test:` — Adding or updating tests
- `docs:` — Documentation changes
- `chore:` — Maintenance tasks (CI, dependencies, tooling)

Examples:

```
feat: add support for training status endpoint
fix: handle expired OAuth2 tokens during retry
docs: update README with hydration command examples
```

## Pull Request Process

1. Fork the repo and create your branch from `main`.
2. Ensure `make ci` passes (formatting, linting, and tests).
3. Use a descriptive PR title following Conventional Commits format.
4. Fill out the PR template with a summary and checklist.
5. Keep PRs focused — one logical change per PR.

## Architecture

The codebase follows a layered structure:

- `cmd/gccli/` — Entry point
- `internal/cmd/` — CLI commands (Kong-based)
- `internal/garminapi/` — Garmin Connect API client
- `internal/garminauth/` — Authentication (SSO, OAuth)

See `CLAUDE.md` for a detailed architecture overview.

## Testing

- Unit tests use the standard `testing` package and `httptest` for HTTP mocking.
- Test files are co-located with source files (`*_test.go`).
- Use `testutil.NewClientWithServer(t, handler)` for mock API client tests.
- Run tests: `make test`
- Run a single test: `go test -run TestFuncName ./internal/pkg/...`

### E2E Tests

E2E tests run against the real Garmin Connect API and require credentials:

```bash
# Create .env with GARMIN_EMAIL and GARMIN_PASSWORD
make test-e2e
```

E2E test resources are prefixed with `E2E_TEST_` and cleaned up automatically.

## Questions?

Open an [issue](https://github.com/bpauli/gccli/issues) for questions or discussion.

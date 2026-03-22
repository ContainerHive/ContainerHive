# ContainerHive

Go CLI tool for managing container image builds with dependency resolution.

## Build & Test

- **Run all tests:** `make coverage`
- **Run specific package tests:** `go test ./internal/dependency/...`
- **Run architecture tests:** `go test -run TestPackageDirections -v .`
- **Run best practices tests:** `go test -run TestContextParameterBestPractices -v .`
- **Skip integration tests:** `go test -short ./...`
- **Generate embedded resources:** `make generate` (required before build)
- **Build:** `make build` (uses `-tags prod`)
- **Generate JSON schemas:** `make generate-json-schema`

## Project Structure

- `cmd/ch/` — CLI entry point (`urfave/cli/v3`), thin glue code
- `pkg/` — Public API packages
- `internal/` — Private implementation
- `test-project/` — E2E test fixture for CI provider integration testing
- Strict layering: `cmd` -> `pkg` -> `internal`. No backwards reference. Enforced by architecture tests.

## Workflow

- GitHub PRs for all changes
- CI runs on CircleCI

## Code Style

- Always use early returns to reduce nesting and improve readability.
- No external testing libraries. Tests must use only the standard `testing` package.
- Use `if` + `t.Errorf`/`t.Fatalf` for assertions — no testify, gomega, or similar.
- Use table-driven tests and `t.Run` subtests where appropriate.
- Integration tests must guard with `testing.Short()` and `t.Skip` so `-short` skips them.

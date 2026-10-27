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

## Documentation

- Documentation site built with properdocs + mkdocs-material (`docs/`)
- Preview locally: `pip install -r requirements-docs.txt && properdocs serve`
- Verify before pushing: `properdocs build --strict` (catches broken links and nav problems)
- Navigation managed via `docs/.nav.yml` (awesome-nav plugin). Pages are listed explicitly, so add every new page there.
- ADRs managed via mkdocs-decision-records plugin
- Hosted at [container-hive.eu](https://container-hive.eu/)
- JSON schemas in `schemas/`, published to the docs site

### Keep docs in sync with the code

Documentation is part of the change, not a follow-up. In the same PR:

- **CLI changes** (commands, flags, env vars, defaults, output formats): update `docs/usage/cli.md`, and `docs/configuration/environment.md` for env vars. Check flags against `ch <command> --help`.
- **Config changes** (`hive.yml`, `image.yml`, `test.yml` fields): update `docs/configuration/*`, and regenerate the JSON schemas with `make generate-json-schema`.
- **Feature changes or removals**: update the matching page in `docs/usage/` or `docs/how-it-works/`, and any example in `docs/getting-started.md` or the README that is affected.
- **New features**: add a page (or section) and register it in `docs/.nav.yml`. Explain behavior and when to use it, not just the flags.
- **Design decisions** with real trade-offs get an ADR in `docs/adr/` (copy `000-TEMPLATE.md`).
- Keep entry pages (`README.md`, `docs/index.md`, `docs/getting-started.md`) short and working. Details belong in the reference pages. Never duplicate long text between them, link instead.

## Workflow

- GitHub PRs for all changes
- CI runs on CircleCI

### Commits and pushing

- Never force-push. Do not amend or rebase commits that are already pushed. Add a follow-up commit instead.
- Keep commits granular: one logical change per commit. A review fix, a new test, and a documentation update each get their own commit.
- Answer review feedback with new commits on the same branch, so every round stays reviewable.
- Add the `Co-Authored-By` trailer that names the model used to write the commit.

## Code Style

- Always use early returns to reduce nesting and improve readability.
- No external testing libraries. Tests must use only the standard `testing` package.
- Use `if` + `t.Errorf`/`t.Fatalf` for assertions — no testify, gomega, or similar.
- Use table-driven tests and `t.Run` subtests where appropriate.
- Integration tests must guard with `testing.Short()` and `t.Skip` so `-short` skips them.

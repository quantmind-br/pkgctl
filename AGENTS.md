# Repository Guidelines

## Project Structure & Module Organization
- `cmd/pkgctl` hosts the Cobra CLI entrypoint and version injection logic.
- `internal/core` orchestrates installs, while `internal/backends` handles AppImage/DEB/RPM and archive flows.
- `internal/security`, `internal/logging`, and `internal/config` centralize guards, structured logging, and Viper-based settings.
- `internal/ui` and `internal/helpers` provide CLI UX utilities; share new prompts and detection helpers here.
- `pkg-test/` keeps sample packages for manual drills; `testdata/` stores deterministic fixtures; builds land in `bin/`.

## Build, Test, and Development Commands
- `make build` compiles `cmd/pkgctl` into `bin/pkgctl` with the current git metadata.
- `make run` rebuilds then executes the CLI; use for smoke checks.
- `make test` runs `go test -race ./...`.
- `make lint` invokes `golangci-lint run ./...`.
- `make validate` chains fmt, vet, lint, and test before review.
- `make tidy` refreshes module deps; `make clean` purges build and coverage artifacts.

## Coding Style & Naming Conventions
- Use `go fmt` rules (`make fmt`)—tabs for indentation and gofmt-managed spacing.
- Stick to Go naming: exported types/functions PascalCase with doc comments, internals camelCase, constants ALL_CAPS only when required.
- Keep Cobra command names lowercase with hyphenated aliases (`pkgctl list`, `pkgctl doctor`).
- Log through `internal/logging` to ensure zerolog formatting and rotation.

## Testing Guidelines
- Place unit tests beside code (`*_test.go`, `TestXxx`) mirroring package boundaries.
- Run `make test` before pushes; add `-run` filters for targeted suites when iterating quickly.
- Generate coverage via `make test-coverage`; attach `coverage.html` snippets when diagnosing gaps.
- Prefer fixtures under `testdata/`; use `pkg-test/` assets only for manual scenarios and document reproduction steps.

## Commit & Pull Request Guidelines
- Follow Conventional Commits (`feat:`, `fix:`, `docs:`, `chore:`) as reflected in history.
- Squash work into focused commits describing scope and surface area.
- Summaries should state behavior changes, list validation commands (`make validate`), and link issues; include screenshots or terminal captures for UX changes.
- Request review when all CI-equivalent checks pass and configuration migrations are documented.

## Security & Configuration Tips
- Reuse validators from `internal/security` for filesystem or name inputs; do not bypass path sanitization.
- Configuration loads from `config.toml` in `$HOME/.config/pkgctl` or the repo; keep defaults aligned with `internal/config`.
- Avoid committing binaries from `pkg-test/` or generated artifacts; add new samples to `.gitignore` when necessary.

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

pkgctl is a modern, type-safe package manager for Linux written in Go. It provides a unified interface for installing and managing applications from multiple package formats (AppImage, DEB, RPM, Tarball, ZIP, Binary) with full desktop integration, Wayland/Hyprland support, and SQLite-based tracking.

**Tech Stack:**
- Go 1.25.3
- SQLite (modernc.org/sqlite - pure Go implementation)
- Cobra (CLI framework)
- Viper (configuration management)
- zerolog (structured logging)
- afero (filesystem abstraction for testing)

## Development Commands

### Building & Running
```bash
make build              # Build binary to bin/pkgctl
make install            # Install to $GOBIN or $GOPATH/bin
make run                # Build and run
./bin/pkgctl --help     # Run built binary directly
```

### Testing
```bash
make test               # Run all tests with race detector
make test-coverage      # Generate coverage report (coverage.html)
make coverage           # Show coverage in terminal
```

### Code Quality
```bash
make fmt                # Format code with gofmt
make vet                # Run go vet
make lint               # Run golangci-lint (requires golangci-lint installed)
make validate           # Run fmt + vet + lint + test (full validation)
make quick-check        # Run fmt + vet + lint (skip tests)
make tidy               # Tidy go modules
```

**After any code modification, run:** `make validate` to ensure all checks pass.

## Code Architecture

### Backend Registry Pattern

pkgctl uses a **priority-ordered backend registry** for package format detection and handling. This is the core architectural pattern.

**Key files:**
- `internal/backends/backend.go` - Backend interface and registry
- `internal/backends/{appimage,deb,rpm,tarball,binary}/` - Format-specific backends
- `internal/core/interfaces.go` - Core domain models

**Backend Interface:** (internal/backends/backend.go:19-31)
```go
type Backend interface {
    Name() string
    Detect(ctx context.Context, packagePath string) (bool, error)
    Install(ctx context.Context, packagePath string, opts core.InstallOptions) (*core.InstallRecord, error)
    Uninstall(ctx context.Context, record *core.InstallRecord) error
}
```

**Registration Order (CRITICAL):**
The order backends are registered in `NewRegistry()` matters:
1. DEB and RPM - Specific format detection first
2. AppImage - MUST come before Binary (AppImages are ELF executables too)
3. Binary - Generic ELF detection
4. Tarball/ZIP - Archive formats last

### Installation Flow

1. **Detection**: `Registry.DetectBackend()` iterates backends in priority order
2. **Installation**: Backend-specific `Install()` method
3. **Desktop Integration**: `internal/desktop/` handles .desktop file generation/updates
4. **Icon Management**: `internal/icons/` extracts and installs icons (PNG, SVG, ICO, XPM)
5. **Database**: `internal/db/` records installation in SQLite
6. **Cache Updates**: `internal/cache/` updates desktop database and icon cache

### Key Components

**Configuration:**
- `internal/config/` - TOML-based config (~/.config/pkgctl/config.toml)
- Default paths: `~/.local/share/pkgctl/` (data), `~/.local/share/pkgctl/installed.db` (SQLite)

**Desktop Integration:**
- `internal/desktop/` - Generates/modifies .desktop files
- Wayland support: Auto-injects environment variables (GDK_BACKEND, QT_QPA_PLATFORM, MOZ_ENABLE_WAYLAND, ELECTRON_OZONE_PLATFORM_HINT)
- Located in: `~/.local/share/applications/`

**Database Schema:**
- SQLite table: `installs` with columns: install_id, package_type, name, version, install_date, original_file, install_path, desktop_file, metadata (JSON)
- Indexes on: name, package_type

**Helpers:**
- `internal/helpers/detection.go` - File type detection, executable scoring heuristics
- `internal/helpers/exec.go` - Command execution utilities
- `internal/security/` - Path validation, traversal attack prevention

## Code Style & Conventions

**From .editorconfig:**
- Go files: Tabs for indentation (size 4), max line length 120
- charset: utf-8
- trim trailing whitespace, insert final newline

**Naming:**
- Interface types in `internal/core/` and `internal/backends/`
- Backend implementations in separate packages
- Use `ctx context.Context` as first parameter for I/O operations
- Error handling: Always check errors, use `zerolog` for logging

**Testing:**
- Use `afero` filesystem mocking for all filesystem operations
- Race detector enabled in tests (`-race` flag)
- Test files: `*_test.go`
- Coverage target: Generate HTML reports with `make test-coverage`

**Linting:** (from .golangci.yml)
- Enabled linters: errcheck, gosimple, govet, ineffassign, staticcheck, unused, gosec, gofmt, goimports, misspell, unparam, unconvert, goconst, gocyclo, revive
- Max cyclomatic complexity: 15
- Security: G204 (subprocess with variable) excluded - needed for package installation
- Test files: Relaxed linting (gocyclo, errcheck, gosec, unparam excluded)

## Adding a New Package Format

1. Create backend directory: `internal/backends/<format>/`
2. Implement `Backend` interface (Detect, Install, Uninstall)
3. Register in `internal/backends/backend.go` `NewRegistry()` - **ORDER MATTERS**
4. Add comprehensive tests with `afero` mocking
5. Update README.md documentation

**Example:** See `internal/backends/appimage/` for reference implementation

## Special Considerations

**AppImage Detection:**
- AppImages are ELF executables, so AppImage backend MUST be registered before Binary backend
- Detection checks for magic bytes and `.AppImage` file patterns

**Executable Scoring (Tarball/Binary):**
- `internal/helpers/detection.go` uses heuristics: filename matching, directory depth, file size
- Bonuses for: matching package name, being in bin/, appropriate size
- Penalties for: test files, deep nesting, very large files

**Wayland Support:**
- Config flag: `desktop.wayland_env_vars` (default: true)
- Injects environment variables into Exec= line of .desktop files
- Ensures compatibility with Wayland/Hyprland compositors

**Security:**
- All paths validated in `internal/security/validation.go`
- Prevents directory traversal attacks
- Rollback on installation failures

## Database Operations

- SQLite in WAL mode for reliability
- Schema migrations handled in `internal/db/`
- Metadata stored as JSON blob for flexibility
- Always use transactions for multi-step operations

## Logging

- Structured logging via `zerolog`
- Log levels: debug, info, warn, error
- Log file: `~/.local/share/pkgctl/pkgctl.log`
- Color output configurable: always, never, auto

## Dependencies to Know

**Required for functionality:**
- tar - Extract tarballs
- unsquashfs - Extract AppImage filesystems

**Optional (checked by `pkgctl doctor`):**
- debtap - DEB package conversion on Arch
- rpmextract.sh - RPM extraction
- gtk4-update-icon-cache - Icon cache updates
- update-desktop-database - Desktop database updates
- npx - Electron ASAR icon extraction

## Entry Point

`cmd/pkgctl/main.go:15` - Loads config, initializes logger, executes root command

Commands defined in: `internal/cmd/` (install.go, list.go, info.go, uninstall.go, doctor.go)

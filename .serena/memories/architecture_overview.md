# pkgctl Architecture Overview

## Purpose
Modern, type-safe package manager for Linux supporting multiple package formats (AppImage, DEB, RPM, Tarball, ZIP, Binary) with desktop integration and SQLite tracking.

## Core Architecture Pattern: Backend Registry

### Backend Interface (internal/backends/backend.go:19-31)
All package formats implement the `Backend` interface:
- `Name()` - Backend identifier
- `Detect(ctx, packagePath)` - Detects if backend can handle package
- `Install(ctx, packagePath, opts)` - Installs package
- `Uninstall(ctx, record)` - Removes package

### Registry (internal/backends/backend.go)
Priority-ordered backend registry with **CRITICAL ordering**:

1. **DEB/RPM** - Specific format detection first
2. **AppImage** - MUST come before Binary (AppImages are ELF too)
3. **Binary** - Generic ELF detection
4. **Tarball/ZIP** - Archive formats last

**Why order matters:** AppImages are ELF executables, so if Binary comes first, AppImages will be misdetected.

### Installation Flow

1. **Detection**: `Registry.DetectBackend()` - Iterates backends until match
2. **Installation**: Backend-specific `Install()` method
3. **Desktop Integration**: `internal/desktop/` - Generate/update .desktop files
4. **Icon Management**: `internal/icons/` - Extract icons (PNG, SVG, ICO, XPM)
5. **Database Recording**: `internal/db/` - SQLite record creation
6. **Cache Updates**: `internal/cache/` - Update desktop database & icon cache

## Directory Structure

```
internal/
├── backends/          # Package format handlers
│   ├── appimage/      # AppImage backend
│   ├── binary/        # ELF binary backend
│   ├── deb/           # DEB package backend
│   ├── rpm/           # RPM package backend
│   ├── tarball/       # Tarball/ZIP backend
│   └── backend.go     # Registry & interface
├── cache/             # Desktop database & icon cache updates
├── cmd/               # CLI commands (install, list, uninstall, doctor)
├── config/            # TOML configuration management
├── core/              # Domain models & interfaces
├── db/                # SQLite database layer
├── desktop/           # .desktop file generation/modification
├── helpers/           # Utilities (detection, exec, etc.)
├── icons/             # Icon extraction & installation
├── logging/           # Structured logging (zerolog)
├── security/          # Path validation, traversal prevention
└── ui/                # CLI UI components
cmd/pkgctl/            # Entry point (main.go)
```

## Key Components

### Configuration (internal/config/)
- TOML-based: `~/.config/pkgctl/config.toml`
- Paths: data_dir, db_file, log_file
- Desktop: wayland_env_vars, custom_env_vars
- Logging: level, color

### Database (internal/db/)
- SQLite with WAL mode
- Table: `installs` (install_id, package_type, name, version, install_date, original_file, install_path, desktop_file, metadata)
- Metadata: JSON blob (icons, wrapper scripts, wayland support)
- Indexes: name, package_type

### Desktop Integration (internal/desktop/)
- Generates .desktop files in `~/.local/share/applications/`
- Wayland support: Injects env vars (GDK_BACKEND, QT_QPA_PLATFORM, MOZ_ENABLE_WAYLAND, ELECTRON_OZONE_PLATFORM_HINT)
- Icon references

### Helpers (internal/helpers/)
- **detection.go**: File type detection, executable scoring heuristics
  - Scoring: filename match, directory depth, file size
  - Bonuses: name match, bin/ location, appropriate size
  - Penalties: test files, deep nesting, huge files
- **exec.go**: Command execution utilities

### Security (internal/security/)
- Path validation
- Directory traversal prevention
- Rollback on installation failures

## Tech Stack
- Go 1.25.3
- SQLite (modernc.org/sqlite - pure Go)
- Cobra (CLI framework)
- Viper (configuration)
- zerolog (structured logging)
- afero (filesystem abstraction for testing)

## Entry Point
`cmd/pkgctl/main.go:15` - Loads config → Initializes logger → Executes root command

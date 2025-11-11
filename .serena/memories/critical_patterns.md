# Critical Patterns & Gotchas

## Backend Registration Order (CRITICAL)

**Location:** `internal/backends/backend.go` - `NewRegistry()` function

**MUST maintain this order:**
1. DEB and RPM (specific format detection)
2. **AppImage** - MUST come before Binary
3. Binary (generic ELF detection)
4. Tarball/ZIP (archive formats)

**Why:** AppImages are ELF executables. If Binary backend is registered before AppImage, it will match first and misidentify AppImages as generic binaries.

**Code reference:** `internal/backends/backend.go` - `NewRegistry()` function

## Filesystem Abstraction

**Always use `afero.Fs` interface for filesystem operations.**

**Why:** Enables testing without real filesystem operations. All backends and components use afero mocks in tests.

**Pattern:**
```go
// Production
fs := afero.NewOsFs()

// Testing
fs := afero.NewMemMapFs()
```

## Context Propagation

**Always pass `context.Context` as first parameter for I/O operations.**

**Pattern:**
```go
func Install(ctx context.Context, packagePath string, opts InstallOptions) error {
    // Use ctx for cancellation, timeouts, values
}
```

## Error Handling with Logging

**Pattern:**
```go
if err != nil {
    log.Error().
        Err(err).
        Str("package", packagePath).
        Msg("failed to install package")
    return fmt.Errorf("install failed: %w", err)
}
```

**Use:**
- `%w` for error wrapping
- Structured logging with context
- Descriptive error messages

## Executable Detection Heuristics

**Location:** `internal/helpers/detection.go`

**Scoring system** for finding main executable in tarballs:
- **Bonuses:** Filename matches package name, in bin/ directory, reasonable size (1KB-100MB)
- **Penalties:** Contains "test", deep directory nesting, very large files

**Gotcha:** Sometimes picks wrong executable. Review scoring if detection fails.

## Path Validation

**Location:** `internal/security/validation.go`

**Always validate paths before filesystem operations:**
```go
if err := security.ValidatePath(packagePath); err != nil {
    return fmt.Errorf("invalid path: %w", err)
}
```

**Prevents:** Directory traversal attacks (`../../../etc/passwd`)

## Desktop Entry Wayland Support

**Location:** `internal/desktop/`

**Pattern:** When `wayland_env_vars = true`, inject environment variables:
```
GDK_BACKEND=wayland,x11
QT_QPA_PLATFORM=wayland;xcb
MOZ_ENABLE_WAYLAND=1
ELECTRON_OZONE_PLATFORM_HINT=auto
```

**Gotcha:** Must inject into Exec= line, not as separate variables

## SQLite Transactions

**Always use transactions for multi-step database operations:**
```go
tx, err := db.Begin()
if err != nil {
    return err
}
defer tx.Rollback() // Safe even if committed

// ... operations ...

if err := tx.Commit(); err != nil {
    return err
}
```

## Rollback on Installation Failure

**Pattern:** Track installation state, rollback on errors:
```go
// Track what was created
var installedFiles []string

// On error
defer func() {
    if err != nil {
        for _, f := range installedFiles {
            os.Remove(f)
        }
    }
}()
```

## Icon Extraction Priority

**Order (internal/icons/):**
1. PNG (preferred)
2. SVG (scalable)
3. ICO (Windows format)
4. XPM (legacy)

**Gotcha:** AppImages may have `.DirIcon` or embedded icons in .desktop files

## Command Execution

**Location:** `internal/helpers/exec.go`

**Pattern:**
```go
cmd := exec.CommandContext(ctx, "tar", "xf", archivePath)
cmd.Env = append(os.Environ(), customEnvVars...)
output, err := cmd.CombinedOutput()
```

**Security:** G204 (gosec) excluded because subprocess with variables is required for package installation

## DEB/RPM Installation

**DEB (Arch):** Uses `debtap` to convert to Arch package, installs via `pacman`
**RPM:** Two methods:
1. `rpmextract.sh` (preferred) - Manual extraction
2. `debtap` (fallback) - Conversion to Arch package

**Gotcha:** `debtap` may produce malformed dependencies, handle gracefully

## Electron Apps (Tarballs)

**Pattern:** Detect ASAR files, extract icons with `npx asar extract`

**Gotcha:** Invalid build paths in packaged apps may cause icon extraction to fail

## Testing with Race Detector

**All tests run with `-race` flag.**

**Common race conditions:**
- Shared logger instances
- Concurrent database access
- Filesystem operations without proper locking

**Pattern:** Use proper synchronization or isolated test instances

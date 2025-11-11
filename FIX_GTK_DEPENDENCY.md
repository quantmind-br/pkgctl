# Fix: DEB Package Installation - GTK Dependency Resolution

## Problem Description

When installing DEB packages (e.g., `goose_1.13.0_amd64.deb`) using `pkgctl`, the installation was failing during the `pacman` installation step with the following error:

```
atenção: não é possível resolver "gtk", uma dependência de "goose"
erro: falha ao preparar a transação (não foi possível satisfazer as dependências)
```

**Translation:** "Warning: cannot resolve 'gtk', a dependency of 'goose'. Error: failed to prepare transaction (could not satisfy dependencies)"

## Root Cause Analysis

### The Issue
1. **DEB packages** from Debian/Ubuntu ecosystems declare dependencies using their distribution-specific package names
2. **debtap** converts DEB packages to Arch packages but performs literal translation of dependency names
3. **Arch Linux** uses different package naming conventions than Debian/Ubuntu
4. Specifically, Arch doesn't have a package called "gtk" - it has "gtk2", "gtk3", and "gtk4"

### Why This Happens
- **Debian/Ubuntu naming:** Uses generic names like "gtk", "python3", "libssl", etc.
- **Arch Linux naming:** Uses specific versions like "gtk3", "python", "openssl", etc.
- **debtap limitation:** Doesn't have comprehensive package name mapping for all cross-distro differences
- **Result:** pacman fails to resolve dependencies that don't exist in Arch repositories

## Solution Implemented

### Debian→Arch Package Name Mapping

Added a comprehensive dependency mapping system in `internal/backends/deb/deb.go` that translates Debian/Ubuntu package names to their Arch Linux equivalents during the dependency fixing phase.

### Key Changes

#### 1. Enhanced `fixDependencyLine()` Function
- **Location:** `internal/backends/deb/deb.go:793`
- **Added:** Debian→Arch package name mapping table
- **Functionality:** Automatically translates package names while preserving version constraints

#### 2. Dependency Mapping Table
```go
debianToArchMap := map[string]string{
    "gtk":          "gtk3",          // Generic GTK → GTK3 (most compatible)
    "gtk2.0":       "gtk2",          // Debian GTK2 naming
    "gtk-3.0":      "gtk3",          // Debian GTK3 naming variant
    "python3":      "python",        // Arch uses "python" for Python 3
    "libssl":       "openssl",       // SSL library naming
    "libssl1.1":    "openssl",       // Specific SSL version
    "libssl3":      "openssl",       // OpenSSL 3.x
    "libjpeg":      "libjpeg-turbo", // JPEG library
    "libpng16":     "libpng",        // Specific version to generic
    "zlib1g":       "zlib",          // Debian zlib naming
    "libcurl":      "curl",          // Curl library
    "libcurl4":     "curl",          // Curl 4.x
    "libglib2.0":   "glib2",         // GLib naming difference
    "libnotify4":   "libnotify",     // Remove version suffix
}
```

#### 3. Version Constraint Preservation
- Extracts version constraints (>=, <=, =, >, <) from original dependency
- Applies mapping to package name
- Re-attaches version constraint to mapped package name
- Example: `gtk>=3.0` → `gtk3>=3.0`

#### 4. Enhanced Logging
- Added debug logging for all dependency mappings
- Logs original dependency name and mapped name
- Helps troubleshoot future dependency issues
- Example output:
  ```
  dependency mapping applied original=gtk fixed=gtk3
  ```

### Technical Implementation Details

**Modified Functions:**
1. `fixMalformedDependencies(pkgPath string, logger *zerolog.Logger)`
   - Added logger parameter for tracking
   - Enhanced logging for removed/fixed dependencies

2. `fixDependencyLine(line string, logger *zerolog.Logger)`
   - Added logger parameter
   - Added version constraint extraction and preservation
   - Implemented Debian→Arch mapping logic
   - Applied mapping before other pattern-based fixes

**Execution Flow:**
1. `debtap` converts DEB → Arch package (.pkg.tar.zst)
2. `pkgctl` extracts package metadata (.PKGINFO)
3. **NEW:** Maps Debian dependency names to Arch equivalents
4. **NEW:** Logs all mappings for transparency
5. Repacks modified package
6. `pacman` installs package with corrected dependencies

## Testing

### Build & Test Results
```bash
make build    # ✓ Successful compilation
make test     # ✓ All tests pass
```

### Expected Behavior After Fix

**Before:**
```
→ Installing package...
09:30:16 INF installing converted package with pacman...
Error: installation failed: pacman installation failed
stderr: atenção: não é possível resolver "gtk"
```

**After:**
```
→ Installing package...
09:30:16 DBG dependency mapping applied original=gtk fixed=gtk3
09:30:16 INF installing converted package with pacman...
✓ Package installed successfully via pacman
```

## Benefits

1. **Automatic Resolution:** No user intervention required
2. **Extensible:** Easy to add more package mappings
3. **Safe:** Preserves version constraints
4. **Transparent:** Logs all mappings for debugging
5. **Backward Compatible:** Doesn't affect existing functionality

## Future Enhancements

### Potential Additions to Mapping Table
- More GTK-related packages (gdk, glib variants)
- Python library mappings (python-X vs python3-X)
- Qt library mappings (qt5-base, qt6-base)
- Database libraries (libmysql, libpq)
- Multimedia libraries (libav, ffmpeg variants)

### Dynamic Mapping
- Could query Arch repos to validate mapped package exists
- Could suggest alternatives if mapped package not found
- Could use package provides/conflicts data for smarter mapping

## Related Files
- `internal/backends/deb/deb.go` - Main implementation
- `internal/helpers/exec.go` - Command execution utilities
- `internal/logging/logger.go` - Structured logging

## References
- **Issue:** GTK dependency resolution failure in DEB package installation
- **Arch Packages:** https://archlinux.org/packages/
- **debtap:** https://github.com/helixarch/debtap
- **Package Naming Conventions:** 
  - Debian: https://www.debian.org/doc/debian-policy/ch-relationships.html
  - Arch: https://wiki.archlinux.org/title/PKGBUILD

## Author & Date
- **Fixed:** 2025-11-07
- **Component:** DEB Backend
- **Impact:** All DEB package installations on Arch Linux

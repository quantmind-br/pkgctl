# Optimization Findings

## Stream Command Output to Reduce Memory Pressure
- **Location:** `internal/helpers/exec.go:25`, `internal/helpers/exec.go:43`
- **Issue:** `RunCommand`, `RunCommandInDir`, and `RunCommandWithOutput` buffer entire stdout/stderr in memory even when callers ignore them. Long-running operations such as `sudo pacman -U` or `npx asar extract` can emit tens of megabytes, bloating RSS and blocking the child while Go copies buffers.
- **Recommendation:** Provide streaming variants that accept `io.Writer` sinks (logger, `os.Stdout`, temp files) or expose the `*exec.Cmd` so callers can wire pipes. Default to streaming for large/long commands, reserving buffering for short queries.

## Fast-Path ELF Detection in Tarball Backend
- **Location:** `internal/backends/tarball/tarball.go:289`
- **Issue:** `findExecutables` opens every candidate with `helpers.IsELF`, which calls `elf.Open`—an expensive parse of headers, sections, and symbols. Large archives with numerous binaries trigger repeated disk seeks and decoding.
- **Recommendation:** Start with a cheap magic-number probe (read first 4 bytes for `\x7fELF`). Only escalate to `elf.Open` when deeper validation is required (e.g., to distinguish shared objects). This avoids heavy I/O for clear non-ELF files.

## Avoid Repeated `npx` Boots for ASAR Extraction
- **Location:** `internal/backends/tarball/tarball.go:604-690`
- **Issue:** Each Electron bundle extraction spawns `npx --yes asar`, forcing Node startup and package resolution on every call and potentially hitting the network. Installs pay seconds of overhead per ASAR.
- **Recommendation:** Prefer a Go-native ASAR reader, vendor a minimal extractor, or require the `asar` CLI pre-installed and invoke it directly. Cache a temp workspace per install rather than per archive to reuse the resolved tooling.

## Release Resources Between ASAR Iterations
- **Location:** `internal/backends/tarball/tarball.go:639`, `internal/backends/tarball/tarball.go:647`
- **Issue:** `defer os.RemoveAll(tempDir)` and `defer cancel()` live inside the loop that processes each ASAR. Deferred cleanup only runs after all iterations, leaving temporary directories and active contexts until the end.
- **Recommendation:** Replace the defers with explicit cleanup at the end of each loop iteration. This frees disk space and timers immediately, which is noticeable when multiple ASAR archives are present.

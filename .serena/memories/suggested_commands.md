# Suggested Commands for pkgctl

## Build & Run
```bash
make build              # Build binary to bin/pkgctl
make install            # Install to $GOBIN or $GOPATH/bin
make run                # Build and run
./bin/pkgctl --help     # Run built binary directly
```

## Testing
```bash
make test               # Run all tests with race detector
make test-coverage      # Generate coverage report (coverage.html)
make coverage           # Show coverage in terminal
```

## Code Quality & Validation
```bash
make fmt                # Format code with gofmt
make vet                # Run go vet
make lint               # Run golangci-lint
make validate           # Run fmt + vet + lint + test (FULL VALIDATION)
make quick-check        # Run fmt + vet + lint (skip tests)
make tidy               # Tidy go modules
```

## Task Completion Workflow

**After ANY code modification:**
```bash
make validate
```

This ensures:
1. Code is formatted (`gofmt`)
2. No suspicious constructs (`go vet`)
3. All linters pass (`golangci-lint`)
4. All tests pass with race detector

## Running the Application

```bash
# After building
./bin/pkgctl install <package>
./bin/pkgctl list
./bin/pkgctl doctor
```

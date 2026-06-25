# Task 1 Report — Go module scaffold + internal/serial

## Status: DONE_WITH_CONCERNS

## Commits
- `402e087` — feat(go): scaffold Go module and internal/serial

## Tests
4/4 PASS (TestFindPortEnvOverride, TestFindPortEmptyWithNoDevice, TestWriteStateAppendsNewline, TestWriteStateReturnsErrorOnWriteFailure)

## What was done
1. Removed Python host files: `host/beacon/`, `host/tests/`, `host/pyproject.toml`
2. Initialized Go module (`module beacon`) in `host/`
3. Added `go.bug.st/serial@v1.7.1` as the only external dependency
4. Created `host/internal/serial/`, `host/internal/socket/`, `host/internal/daemon/` directories
5. Wrote failing tests first (TDD red phase confirmed: `undefined: FindPort`)
6. Implemented `serial.go` with `FindPort`, `SendDirect`, `writeState`, and `portWriter` interface
7. All 4 tests pass (TDD green phase confirmed)

## Concerns

**Go version mismatch in GVM setup:**
- The brief specified Go 1.24, but `go.bug.st/serial@v1.7.1` requires `go >= 1.25.0`
- Running `go get go.bug.st/serial@latest` automatically upgraded go.mod to `go 1.25.0`
- GVM has go1.25.4 installed but has a misconfigured GOROOT (points to go1.24.0 tree)
- Workaround: tests must be run with `GOROOT=/Users/jorocha/.gvm/gos/go1.25.4 /Users/jorocha/.gvm/gos/go1.25.4/bin/go test ./internal/serial/...`
- The `go.mod` now declares `go 1.25.0` (not 1.24 as originally specified)
- **Action needed for Task 2+:** either fix GVM GOROOT or use a consistent Go invocation alias. A Makefile or `.envrc` with the correct GOROOT export would resolve this for all future tasks.

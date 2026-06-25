# Task 4 Report: Wire config resolution into state commands

## Status: DONE

## Commits
- `efcce39` feat(config): wire config resolution into state commands

## Changes Made

### host/internal/socket/socket.go
- Updated `SendToSocket` to send `cmd` verbatim via `fmt.Fprint` (was appending `\n` internally).
- Caller is now responsible for including the trailing newline — aligns with `buildCommand` output.

### host/internal/socket/socket_test.go
- Updated `TestSendToSocketWritesStateLine` and `TestSendToSocketErrorWhenNoServer` to pass `"thinking\n"` (full cmd string) instead of `"thinking"`.

### host/internal/serial/serial.go
- Added `SendDirectRaw(cmd, port string, timeout time.Duration) error` — public API, opens serial port and writes cmd verbatim.
- Added `sendDirectRaw(cmd string, p portWriter, _ time.Duration) error` — unexported testable core, same pattern as `writeState`.

### host/internal/serial/serial_test.go
- Added `TestSendDirectRawWritesVerbatim` — verifies full cmd with params is written verbatim.
- Added `TestSendDirectRawReturnsErrorOnWriteFailure` — covers error path.

### host/main.go
- Added `"strings"` import.
- Added `buildCommand(state string, cfg config.Config) string` — calls `cfg.Resolve(state)`, returns bare `"state\n"` when no params, else `"state key=val ...\n"`.
- Updated `runSocket` — loads config, calls `buildCommand`, passes full cmd to `SendToSocket`.
- Updated `runDirect` — loads config, calls `buildCommand`, passes full cmd to `SendDirectRaw`.

## Tests

```
?     beacon              [no test files]
ok    beacon/internal/config   (cached)
ok    beacon/internal/daemon   (cached)
ok    beacon/internal/serial   0.821s
ok    beacon/internal/socket   0.816s
```

All packages pass.

## Concerns
None. Backward compatibility preserved: when `~/.beacon/config.toml` is absent or has no params for the given state, `buildCommand` returns bare `"state\n"`, identical to the previous behavior.

---

# Task 4 Addendum: Code Quality Fixes (config.go)

## Status: DONE

## Commit
- `b266a2b` fix(config): use errors.Is, surface TOML parse error, run go mod tidy

## Changes Made

### host/internal/config/config.go
- Added `"errors"` to import block.
- `Load()`: replaced `os.IsNotExist(err)` with `errors.Is(err, os.ErrNotExist)` — idiomatic and wrapping-safe.
- `Reset()`: replaced `os.IsNotExist(err)` with `errors.Is(err, os.ErrNotExist)` — same fix.
- `Set()`: replaced `_ = toml.Unmarshal(data, &raw)` with a proper error check that returns `fmt.Errorf("existing config is invalid TOML: %w", err)` — prevents silently destroying config on malformed TOML.

### host/go.mod
- `go mod tidy` promoted `github.com/pelletier/go-toml/v2` from `// indirect` to a direct dependency (it is imported directly by config.go).

## Tests

```
?     beacon                       [no test files]
ok    beacon/internal/config       0.797s
ok    beacon/internal/daemon       (cached)
ok    beacon/internal/serial       (cached)
ok    beacon/internal/socket       (cached)
```

All packages pass.

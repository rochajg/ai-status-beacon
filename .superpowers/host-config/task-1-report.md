# Task 1 Report — internal/config: Load, Path, Resolve

## Status: DONE

## Commits
- `dc8fecd` feat(config): add internal/config — Load, Resolve per-state params

## Tests
11/11 PASS — `go test ./internal/config/... -v`

| Test | Result |
|------|--------|
| TestLoadMissingFileReturnsEmpty | PASS |
| TestLoadBrightness | PASS |
| TestLoadColor | PASS |
| TestLoadBuzzer | PASS |
| TestResolveEmptyConfigBareCommand | PASS |
| TestResolveColorOnly | PASS |
| TestResolveFullThinking | PASS |
| TestResolveTimingForDone | PASS |
| TestResolveTimingNotAddedForThinking | PASS |
| TestResolveBuzzerEnabled | PASS |
| TestDefaultPath | PASS |

## Files Created/Modified
- `host/internal/config/config.go` — Config struct, RGB TextUnmarshaler/MarshalText, Load(), DefaultPath(), Resolve()
- `host/internal/config/config_test.go` — 11 tests
- `host/go.mod` — added github.com/pelletier/go-toml/v2 v2.4.2
- `host/go.sum` — updated

## Notes
- GOPROXY was set to `direct` for fetching go-toml/v2 (the Fury proxy returned 403)
- No encoding/json import used
- All Config fields are pointer types; Load() returns empty Config on file-not-found
- timing only emitted for "done" and "waiting" states in Resolve()

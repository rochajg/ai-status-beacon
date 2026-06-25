# Task 3 Report — beacon config subcommand

**Status:** DONE

## Changes

**File modified:** `host/main.go`

- Added `"beacon/internal/config"` import
- Added `case "config": os.Exit(runConfig(args[1:]))` to main switch
- Added `runConfig(args []string) int` — dispatches get/set/reset/path
- Added `printConfig(cfg config.Config, path string)` — prints all categories with `(firmware default)` for nil fields
- Updated `printUsage()` to include all four config subcommands

## Commits

- `44e3533` — feat(config): add beacon config get|set|reset|path subcommand

## Tests

Build: `go build -o /tmp/beacon-test .` — success

Manual verification:
- `beacon config path` → prints `~/.beacon/config.toml`
- `beacon config set color.thinking 0,100,255` → "Set color.thinking = 0,100,255", exit 0
- `beacon config get` → shows set key + (firmware default) for all others
- `beacon config reset` → "Config reset to defaults.", exit 0
- `beacon config set color.invalid 1,2,3` → error message, exit 1

`go test ./... -timeout 30s`:
```
ok  beacon/internal/config   0.206s
ok  beacon/internal/daemon   1.178s
ok  beacon/internal/serial   0.891s
ok  beacon/internal/socket   1.050s
```

Install: `go build -o ~/.local/bin/beacon .` → `beacon config path` prints correct path.

## Concerns

None.

# Task 4 Report — main.go CLI dispatch

## Status: DONE

## Commits
- `9193e60` feat(go): add main.go — CLI dispatch for beacon binary

## Tests
All 3 internal packages pass: daemon (1.336s), serial (0.390s), socket (0.688s). main package has no test files (by design).

## What was implemented

`host/main.go` — 127 lines implementing the full CLI dispatcher:

- **`init()`** — resolves `socketPath` via `os.UserHomeDir()` → `~/.beacon/beacon.sock`
- **`beacon <state>`** — sends via socket, exits 0 always; unknown states exit 0 silently
- **`beacon <state> --direct`** — finds serial port, sends directly, exits 0 always
- **`beacon daemon`** — blocks with `signal.NotifyContext` wired to `SIGTERM`/`SIGINT`
- **`beacon status`** — pings socket + checks `FindPort()`, prints formatted lines, exits 0 (both up) or 1 (either down)
- **`beacon --help` / no args** — prints usage, exits 0

## Verification
- `go build -o /tmp/beacon-test .` — clean build
- `--help` output matches spec
- `beacon unknownflag` → exit 0, no output (hook-safe)
- `beacon status` with no daemon running → exits 1 (device was detected on /dev/cu.usbmodem1101)

## Concerns
None.

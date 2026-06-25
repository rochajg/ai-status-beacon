# Status Beacon — Go Host Design

**Version:** 1.0  
**Date:** 2026-06-25  
**Replaces:** `host/` Python package (beacon + beacon-daemon)

---

## 1. Goal

Replace the Python host package with a single Go binary that compiles to a
self-contained executable with no runtime dependencies. The binary exposes four
subcommands under one UX entry point: `beacon`.

---

## 2. User-facing interface

```
beacon <state>             # send state via daemon socket; exit 0 always
beacon <state> --direct    # send state directly to serial; exit 0 always
beacon daemon              # run the daemon (blocking; use & or launchd)
beacon status              # print daemon + device health; exit 0 or 1
beacon --help              # print usage
```

Valid states: `thinking`, `waiting`, `done`, `idle`, `error`, `ping`

### Exit codes

| Command | Success | Failure |
|---------|---------|---------|
| `beacon <state>` | 0 | 0 (silent — never block hooks) |
| `beacon <state> --direct` | 0 | 0 (silent) |
| `beacon daemon` | — (blocks) | 1 (startup failure) |
| `beacon status` | 0 (all up) | 1 (daemon or device absent) |

---

## 3. Architecture

Single Go module at `host/`, one binary built from `main.go`.

```
host/
  go.mod                        # module beacon, go 1.24
  go.sum
  main.go                       # CLI dispatch (os.Args)
  internal/
    serial/
      serial.go                 # FindPort, SendDirect
      serial_test.go
    socket/
      socket.go                 # SendToSocket, PingSocket
      socket_test.go
    daemon/
      daemon.go                 # Run, handleConn, connectSerial
      daemon_test.go
```

### Dependencies

| Package | Purpose |
|---------|---------|
| `go.bug.st/serial` | Cross-platform serial port I/O |
| stdlib only | Everything else |

---

## 4. Component contracts

### `internal/serial`

```go
// FindPort returns the first /dev/cu.usbmodem* device, or "" if none.
// Respects BEACON_SERIAL_PORT env override.
func FindPort() string

// SendDirect opens port, writes "state\n" at 115200 baud, closes.
// Timeout applies to both open and write. Returns error on failure.
// Always closes the port, even on error.
func SendDirect(state, port string, timeout time.Duration) error
```

### `internal/socket`

```go
// SendToSocket connects to the Unix socket and writes "state\n".
// Returns error on failure. Always closes the connection.
func SendToSocket(state, socketPath string, timeout time.Duration) error

// PingSocket connects to the Unix socket, sends "ping\n",
// and waits for "PONG\n". Used by `beacon status`.
func PingSocket(socketPath string, timeout time.Duration) error
```

### `internal/daemon`

```go
// Run is the daemon entry point. Blocks until ctx is cancelled or SIGTERM/SIGINT.
// Creates ~/.beacon/ if absent. Removes stale socket on start.
// Graceful shutdown: closes socket, closes serial, removes socket file.
// main.go passes context.Background() wired to signal.NotifyContext.
// Tests pass a cancellable context for clean shutdown.
func Run(ctx context.Context, socketPath string, baud int)
```

**Daemon behaviour:**
- Listens on Unix socket (backlog 8, accept timeout 500ms)
- Each accepted connection handled in a goroutine
- `sync.Mutex` protects all `serial.Port.Write` calls
- `ping` received on socket → replies `PONG\n` on that connection, does NOT forward to serial
- Any other valid state → forwards `state\n` to serial
- Invalid state → silently discarded
- Serial disconnect → reconnect loop every 2s (interruptible on shutdown)

### `main.go`

Minimal dispatch — no third-party CLI framework:

```go
func main() {
    args := os.Args[1:]
    switch {
    case len(args) == 0 || args[0] == "--help" || args[0] == "-h":
        printUsage()
    case args[0] == "daemon":
        ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
        defer stop()
        daemon.Run(ctx, socketPath, 115200)
    case args[0] == "status":
        os.Exit(runStatus())
    case isValidState(args[0]) && contains(args, "--direct"):
        runDirect(args[0])
    case isValidState(args[0]):
        runSocket(args[0])
    default:
        // unknown state or flag — silent exit 0 (hook safety)
    }
}
```

---

## 5. Constants

| Constant | Value | Location |
|----------|-------|----------|
| Socket path | `~/.beacon/beacon.sock` | `main.go` (passed down) |
| Baud rate | `115200` | `main.go` |
| Timeout (socket/serial) | `300ms` | `main.go` |
| Reconnect interval | `2s` | `daemon/daemon.go` |
| Serial port pattern | `/dev/cu.usbmodem*` | `serial/serial.go` |
| Valid states | `thinking waiting done idle error ping` | `main.go` |

---

## 6. `beacon status` detail

```
daemon   ● running    (~/.beacon/beacon.sock)
device   ● connected  (/dev/cu.usbmodem1101)
```

1. Call `PingSocket` → if `PONG` received within 300ms → daemon UP
2. Call `FindPort` → if non-empty → device connected
3. Print one line per component with ● (up) or ○ (down)
4. Exit 0 if both up; exit 1 if either down

---

## 7. Build and install

```bash
# Build
cd host && go build -o beacon .

# Install to ~/.local/bin (already in PATH)
go build -o ~/.local/bin/beacon .

# Update hooks reference (no change needed — hooks call "beacon <state>")
# Update daemon launch: beacon-daemon → beacon daemon
```

The `beacon-daemon` pipx installation is removed. `beacon daemon` replaces it.

---

## 8. Testing strategy

- **`internal/serial`**: `FindPort` tested via `BEACON_SERIAL_PORT` env override; `SendDirect` accepts a `portWriter` interface (`Write([]byte)(int,error)` + `Close() error`) injected in tests with `io.Pipe()`
- **`internal/socket`**: real Unix socket in `t.TempDir()`; both `SendToSocket` and `PingSocket` tested end-to-end with a goroutine server
- **`internal/daemon`**: real Unix socket in `t.TempDir()` + serial mocked via `portWriter` interface; covers forwarding, ping reply, invalid state discard, graceful shutdown via `context.WithCancel`
- **No hardware required** for any test

---

## 9. Migration steps (after Go binary works)

1. Remove `host/beacon/`, `host/tests/`, `host/pyproject.toml`
2. `pipx uninstall beacon`
3. `go build -o ~/.local/bin/beacon ./host`
4. Update `~/.claude/settings.json` — no hook changes needed; only `beacon daemon` replaces `beacon-daemon` in documentation
5. Update `scripts/install-daemon.sh` if created in M5

---

## 10. Out of scope

- Windows support (macOS-only: `cu.usbmodem*` glob)
- Multiple simultaneous RP2040 devices
- TUI or interactive output
- Configuration file (all config via env vars and CLI flags)

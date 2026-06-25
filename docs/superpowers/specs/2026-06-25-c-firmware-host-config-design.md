# Status Beacon — C Firmware + Host-Side Config Design

**Version:** 1.0
**Date:** 2026-06-25
**Replaces:** MicroPython firmware (`firmware/*.py`) + external MicroPython UF2 dependency

---

## 1. Goal

Two coupled changes:

1. **Migrate the firmware from MicroPython to C** (pico-sdk) so the release ships a
   single self-contained `.uf2` with zero external runtime dependency. No more
   "flash MicroPython, then copy the `.py` files" — one flash, done.

2. **Move all customisation to the host**, keyed per machine. Colors, timings,
   buzzer and brightness live in `~/.beacon/config.toml` on each host. The same
   physical RP2040 shows different colors depending on which machine sent the
   command. The device stores nothing — it applies the values it receives per
   command, falling back to compiled defaults.

---

## 2. Why host-side config (not flash)

The driving use case: one RP2040 carried between a Mac and a PC, each wanting
different colors. If config lived in the device flash, switching machines would
overwrite it. Host-side config makes the device stateless — each host owns its
own look, and the device is interchangeable.

Secondary benefits: no flash wear, no flash erase/program complexity in C, no
XIP timing hazards. The firmware stays simple.

---

## 3. Architecture

```
Mac: ~/.beacon/config.toml            PC: ~/.beacon/config.toml
   (thinking = 0,100,255)                (thinking = 255,180,0)
        │                                       │
   beacon thinking                         beacon thinking
        │  CLI reads config, resolves           │
        ▼                                       ▼
 "thinking color=0,100,255 brightness=200\n"   "thinking color=255,180,0 brightness=255\n"
        │                                       │
        └───────────── daemon (dumb pipe) ──────┘
                            │ forwards the line verbatim
                            ▼
                  RP2040 (C firmware)
                  parses params, applies, falls back to defaults
```

The CLI does the resolution. The daemon is unchanged — it still forwards a single
line per connection. `--direct` mode inherits config automatically because the
CLI builds the same extended command before sending.

---

## 4. Extended serial protocol

The firmware accepts the existing bare form AND an extended form with `key=value`
parameters, space-separated, one command per line, terminated by `\n`:

```
thinking\n
thinking color=0,100,255 brightness=200\n
done color=0,200,0 timing=30000 buzzer=1 freq=1200 brightness=255\n
waiting color=0,200,0 timing=60000 buzzer=0 brightness=180\n
```

### Parameter keys

| Key | Type | Range | Applies to |
|-----|------|-------|------------|
| `color` | `R,G,B` | each 0–255 | the LED color for this state |
| `brightness` | int | 0–255 | global scale applied to the color |
| `timing` | int (ms) | ≥0 | duration before auto-idle (`done`, `waiting`) |
| `buzzer` | int | 0 or 1 | enable/disable buzzer for this command |
| `freq` | int (Hz) | 100–5000 | buzzer tone frequency |

### Parsing rules

- Parser is **tolerant**: unknown keys are ignored, malformed values fall back to
  the compiled default for that field.
- Any parameter the host omits → firmware uses its compiled default.
- The bare form (`thinking\n`) uses all compiled defaults — the device works
  standalone with any serial terminal.

### Device → host (unchanged)

- On boot: `READY\n`
- Response to `ping`: `PONG\n`
- Unknown command: `ERR unknown <cmd>\n`

---

## 5. C firmware (pico-sdk)

```
firmware/
  CMakeLists.txt
  pico_sdk_import.cmake          # standard pico-sdk bootstrap
  src/
    main.c                       # USB CDC read loop + non-blocking state machine
    parser.c / parser.h         # parse "<state> key=value ..." into a command struct
    states.c / states.h         # compiled defaults + per-tick animation generators
    led.c / led.h               # WS2812 driver via PIO (GP16)
    buzzer.c / buzzer.h         # passive buzzer via PWM (GP15)
    ws2812.pio                   # PIO program for WS2812 timing
```

### Behaviour (mirrors current MicroPython firmware)

- `thinking` — yellow sinusoidal breathing
- `waiting` — fast blink + buzzer pattern every 8s; auto-idle after `timing`
- `done` — green blink + 2 beeps at start; auto-idle after `timing`
- `idle` — LED off, buzzer silent
- `error` — solid red

Non-blocking tick loop (~10ms), `time_us_64()` for phase/duration. A new command
interrupts the current animation immediately.

### Compiled defaults (identical to current values)

```c
#define DEF_THINKING_RGB   {200, 140, 0}
#define DEF_WAITING_RGB    {0, 200, 0}
#define DEF_DONE_RGB       {0, 200, 0}
#define DEF_ERROR_RGB      {200, 0, 0}
#define DEF_BRIGHTNESS     255
#define DEF_DONE_MS        30000
#define DEF_WAITING_MS     60000
#define DEF_BUZZER_ENABLED 1
#define DEF_BUZZER_FREQ    1200
#define LED_PIN            16
#define BUZZER_PIN         15
```

### Pin configuration

Pins are `#define`d in one header (`states.h` or a `config.h`). Changing the
board (e.g. Pico LED on GP25) means editing one line and rebuilding. Documented
in the README.

---

## 6. Host config system

### File: `~/.beacon/config.toml`

```toml
brightness = 255

[color]
thinking = "200,140,0"
waiting  = "0,200,0"
done     = "0,200,0"
error    = "200,0,0"

[timing]
done    = 30000
waiting = 60000

[buzzer]
enabled = true
freq    = 1200
```

- The file is optional. When absent, every value falls back to the firmware's
  compiled defaults (the CLI sends bare commands or omits the missing keys).
- Partial files are valid — only set keys override; unset keys fall through.

### `beacon config` subcommands

```
beacon config get                          # print effective config (file ∪ defaults)
beacon config set <key> <value>            # write one key to the file
beacon config reset                        # delete the file → all defaults
beacon config path                         # print the config file path
```

Keys use dotted notation matching the TOML structure:

```
beacon config set brightness 200
beacon config set color.thinking 0,100,255
beacon config set timing.done 60000
beacon config set buzzer.enabled false
beacon config set buzzer.freq 880
```

### Validation

- `color.*`: three ints 0–255 separated by commas, else error to stderr, exit 1.
- `brightness`, `freq`, `timing.*`: integer range check, else error, exit 1.
- `buzzer.enabled`: `true`/`false`/`1`/`0`.
- Unknown key: error to stderr, exit 1, list valid keys.

`config set/get/reset/path` are the only commands besides `status` that may exit
non-zero. State-sending commands remain exit 0 always.

---

## 7. CLI resolution

When the CLI handles a state command (`beacon thinking`, with or without
`--direct`):

1. Load `~/.beacon/config.toml` if present (tolerant: missing file → empty config).
2. Resolve the state to its parameters, merging file values over nothing (absent
   keys are simply not sent — the firmware applies its compiled default).
3. Build the extended command line:
   `thinking color=0,100,255 brightness=200\n`
   - Only include keys the config actually sets.
   - For `done`/`waiting`, include `timing`, `buzzer`, `freq` when set.
4. Send via socket (default) or serial (`--direct`).

If the config file is absent or empty, the CLI sends the bare command
(`thinking\n`) and the device uses compiled defaults — identical to today's
behaviour.

### Resolution stays in the CLI, not the daemon

The daemon remains a verbatim line forwarder (no protocol change, no
request/response). Each hook invocation reads the tiny TOML file fresh (<1ms), so
config changes take effect on the next command with no daemon reload.

---

## 8. Internal Go packages

```
host/internal/
  config/
    config.go        # Load, Get, Set, Reset, Path, Resolve(state) -> params
    config_test.go
  serial/  (unchanged)
  socket/  (unchanged)
  daemon/  (unchanged)
```

### `config` package contract

```go
// Config is the parsed ~/.beacon/config.toml (all fields optional).
type Config struct { ... }

// Load reads the config file; returns an empty Config if the file is absent.
func Load(path string) (Config, error)

// Set writes one dotted key to the file, creating it if needed.
func Set(path, key, value string) error

// Reset deletes the config file.
func Reset(path string) error

// Resolve returns the ordered key=value params for a state, omitting unset keys.
func (c Config) Resolve(state string) []string
```

TOML parsing/writing: use `github.com/pelletier/go-toml/v2` (single
well-maintained dependency). A hand-rolled TOML writer is not worth the risk of
subtle quoting bugs for the `config set` path.

---

## 9. CI / release changes

The release workflow gains a firmware build job:

```yaml
firmware:
  runs-on: ubuntu-latest
  steps:
    - checkout (recursive, for pico-sdk submodule)
    - install arm-none-eabi-gcc, cmake
    - build: cmake -B build && cmake --build build
    - upload build/beacon.uf2 as release asset
```

Release assets become:
- `beacon-darwin-arm64`, `beacon-darwin-amd64` (host CLI, unchanged)
- `beacon.uf2` (single self-contained firmware — replaces firmware.zip)
- `checksums.txt`

pico-sdk is pulled as a git submodule pinned to a known-good tag, or fetched in CI
via `PICO_SDK_FETCH_FROM_GIT=1`.

---

## 10. User-facing flow after this change

```bash
# 1. Install CLI (unchanged)
curl -fsSL .../install.sh | bash

# 2. Flash firmware — ONE step now
#    Hold BOOT, plug in, drag beacon.uf2 onto RPI-RP2 drive. Done.

# 3. (optional) customise — per host
beacon config set color.thinking 0,100,255
beacon config set brightness 180

# 4. Run
beacon daemon &
beacon thinking
```

No MicroPython. No `mpremote`. No copying `.py` files.

---

## 11. Migration / cleanup

1. Remove `firmware/main.py`, `firmware/states.py`, `firmware/hardware.py`.
2. `scripts/flash.sh` updated: flash `beacon.uf2` by copying to the `RPI-RP2`
   mass-storage mount (BOOT mode), no `mpremote`.
3. README (EN + PT-BR) updated: single-flash firmware, `beacon config` section,
   pin/board customisation now means editing `firmware/src/states.h` + rebuild
   (for pins) OR `beacon config` (for colors/timings — no rebuild).
4. `docs/` design + plan committed.

---

## 12. Backwards compatibility

- The bare serial protocol (`thinking\n`) still works → any old tooling or manual
  serial terminal still drives the device.
- A host with no `~/.beacon/config.toml` behaves exactly like today (compiled
  defaults).
- `beacon status`, `beacon daemon`, hooks — all unchanged.

---

## 13. Out of scope

- Windows host support (still macOS `cu.usbmodem*`).
- Storing config on the device flash (explicitly rejected — host-side by design).
- Per-state brightness (single global brightness only).
- Multiple simultaneous devices.
- Live re-render on config change (config applies on the next command, which is
  immediate in practice since hooks fire constantly).
```

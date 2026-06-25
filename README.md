# AI Status Beacon

A USB physical status indicator for Claude Code (and other AI agents). A small RP2040 Zero board on your desk shows what the AI is doing via LED color and sound — no need to watch the terminal.

| State | Meaning | LED | Sound |
|-------|---------|-----|-------|
| `thinking` | AI is working | Yellow, slow breathing | — |
| `waiting` | AI needs your input | Fast blink | Beep every 8s |
| `done` | Task complete | Green blink, 30s | 2 beeps |
| `idle` | Session ended | Off | — |
| `error` | Tool error | Solid red | — |

## Hardware

| Component | Where | Notes |
|-----------|-------|-------|
| [Waveshare RP2040 Zero](https://www.waveshare.com/rp2040-zero.htm) | Any electronics store | Has onboard NeoPixel (GP16) |
| Passive buzzer | GP15 + GND | Optional — LEDs work without it |
| USB-C cable | — | Powers the board from your Mac |

> **Passive buzzer only.** Active buzzers produce a fixed tone; passive ones need PWM so you can control the pitch.

## Quick Start

### 1 — Flash MicroPython

Hold BOOT on the RP2040 Zero while plugging it in. It appears as a USB drive (`RPI-RP2`). Download [MicroPython for RP2040](https://micropython.org/download/RPI_PICO/) and copy the `.uf2` file to the drive. The board reboots automatically.

### 2 — Install the beacon CLI

```bash
curl -fsSL https://raw.githubusercontent.com/rochajg/ai-status-beacon/main/scripts/install.sh | bash
```

This downloads the pre-built binary for your Mac (Apple Silicon or Intel) and places it in `~/.local/bin/beacon`.

### 3 — Flash the firmware

```bash
pip install mpremote
beacon-flash    # or run scripts/flash.sh manually
```

### 4 — Configure Claude Code hooks

Add to `~/.claude/settings.json` (merge with existing content):

```json
{
  "hooks": {
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "beacon thinking", "timeout": 1 }] }],
    "Notification":     [{ "hooks": [{ "type": "command", "command": "beacon waiting",  "timeout": 1 }] }],
    "Stop":             [{ "hooks": [{ "type": "command", "command": "beacon done",     "timeout": 1 }] }],
    "SessionEnd":       [{ "hooks": [{ "type": "command", "command": "beacon idle",     "timeout": 1 }] }]
  }
}
```

### 5 — Start the daemon

```bash
beacon daemon
```

The daemon holds the serial port open and routes hook events to the LED. Keep it running in a terminal, or set it up with launchd (see [Advanced](#advanced)).

### 6 — Test it

```bash
beacon status       # shows daemon + device health
beacon thinking     # LED turns yellow
beacon done         # LED turns green + 2 beeps
beacon idle         # LED turns off
```

---

## How It Works

```
Claude Code ──(hooks)──▶ beacon CLI
                              │
                        beacon daemon  ──(USB serial)──▶ RP2040 Zero
                              │                               │
                         Unix socket                    NeoPixel LED
                                                        Buzzer (opt.)
```

The CLI sends the state name to the daemon via a Unix socket (`~/.beacon/beacon.sock`). The daemon holds the serial port open (avoiding USB resets) and forwards the command to the RP2040, which runs the animation.

Hooks always exit `0` — they never block Claude Code, even if the daemon is not running.

---

## CLI Reference

```
beacon <state>             # send via daemon (default)
beacon <state> --direct    # bypass daemon, write serial directly
beacon daemon              # run the daemon (blocking)
beacon status              # daemon + device health check
```

States: `thinking`, `waiting`, `done`, `idle`, `error`

---

## Advanced

### Auto-start daemon on login (launchd)

```bash
cat > ~/Library/LaunchAgents/com.beacon.daemon.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>             <string>com.beacon.daemon</string>
  <key>ProgramArguments</key>  <array><string>/Users/YOUR_USER/.local/bin/beacon</string><string>daemon</string></array>
  <key>RunAtLoad</key>         <true/>
  <key>KeepAlive</key>         <true/>
  <key>StandardOutPath</key>   <string>/tmp/beacon-daemon.log</string>
  <key>StandardErrorPath</key> <string>/tmp/beacon-daemon.log</string>
</dict>
</plist>
EOF
launchctl load ~/Library/LaunchAgents/com.beacon.daemon.plist
```

Replace `YOUR_USER` with your username.

### Using a different board

The RP2040 Zero has its NeoPixel on **GP16**. If you use a standard Raspberry Pi Pico (LED on GP25) or another board, edit `firmware/hardware.py`:

```python
_np = neopixel.NeoPixel(machine.Pin(16), 1)  # ← change pin here
_buzzer = machine.PWM(machine.Pin(15))        # ← change pin here
```

Then reflash: `./scripts/flash.sh`

### Customising colours and timings

Edit `firmware/states.py` and reflash.

**Change colours** (RGB, 0–255):
```python
YELLOW = (200, 140, 0)   # thinking
GREEN  = (0, 200, 0)     # done / waiting
RED    = (200, 0, 0)     # error
```

**Change durations**:
```python
DONE_DURATION_MS    = 30_000   # how long "done" stays green (ms)
WAITING_DURATION_MS = 60_000   # how long "waiting" blinks before auto-idle (ms)
```

**Disable the buzzer** — in `firmware/hardware.py`, replace `beep()` with a no-op:
```python
def beep(freq=1000, duration_ms=80):
    pass  # buzzer disabled
```

**Change buzzer frequency** (pitch) — in `firmware/states.py`, the `done` state calls `hw.beep(1200, 80)`. Adjust the frequency (Hz) and duration (ms):
```python
hw.beep(880, 100)   # lower pitch, longer beep
```

### Override the serial port

If auto-discovery doesn't find your board:

```bash
export BEACON_SERIAL_PORT=/dev/cu.usbmodem1234
beacon thinking
```

---

## Project Structure

```
firmware/      MicroPython code (runs on the RP2040)
host/          Go source for the beacon CLI
scripts/       flash.sh, install.sh
claude/        settings.example.json for Claude Code hooks
```

---

## Contributing

Issues and PRs welcome. The firmware is plain MicroPython; the host is a single Go binary with no CGO dependencies.

## Licence

MIT

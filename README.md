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

### 1 — Install the beacon CLI

```bash
curl -fsSL https://raw.githubusercontent.com/rochajg/ai-status-beacon/main/scripts/install.sh | bash
```

This downloads the pre-built binary for your Mac (Apple Silicon or Intel) and places it in `~/.local/bin/beacon`.

### 2 — Flash the firmware

Download `beacon.uf2` from the [latest release](https://github.com/rochajg/ai-status-beacon/releases/latest).

Hold BOOT on the RP2040 Zero while plugging it in. Drag `beacon.uf2` onto the `RPI-RP2` drive that appears. The board reboots automatically.

That's it — no MicroPython, no extra tools.

### 4 — Configure Claude Code hooks

Add to `~/.claude/settings.json` (merge with existing content):

```json
{
  "hooks": {
    "UserPromptSubmit": [{ "hooks": [{ "type": "command", "command": "ai-beacon thinking", "timeout": 1 }] }],
    "Notification":     [{ "hooks": [{ "type": "command", "command": "ai-beacon waiting",  "timeout": 1 }] }],
    "Stop":             [{ "hooks": [{ "type": "command", "command": "ai-beacon done",     "timeout": 1 }] }],
    "SessionEnd":       [{ "hooks": [{ "type": "command", "command": "ai-beacon idle",     "timeout": 1 }] }]
  }
}
```

### 5 — Start the daemon

```bash
ai-beacon daemon
```

The daemon holds the serial port open and routes hook events to the LED. Keep it running in a terminal, or set it up with launchd (see [Advanced](#advanced)).

### 6 — Test it

```bash
ai-beacon status       # shows daemon + device health
ai-beacon thinking     # LED turns yellow
ai-beacon done         # LED turns green + 2 beeps
ai-beacon idle         # LED turns off
```

---

## How It Works

```
Claude Code ──(hooks)──▶ beacon CLI
                              │
                        ai-beacon daemon  ──(USB serial)──▶ RP2040 Zero
                              │                               │
                         Unix socket                    NeoPixel LED
                                                        Buzzer (opt.)
```

The CLI sends the state name to the daemon via a Unix socket (`~/.ai-beacon/ai-beacon.sock`). The daemon holds the serial port open (avoiding USB resets) and forwards the command to the RP2040, which runs the animation.

Hooks always exit `0` — they never block Claude Code, even if the daemon is not running.

---

## CLI Reference

```
ai-beacon <state>             # send via daemon (default)
ai-beacon <state> --direct    # bypass daemon, write serial directly
ai-beacon daemon              # run the daemon (blocking)
ai-beacon status              # daemon + device health check
```

States: `thinking`, `waiting`, `done`, `idle`, `error`

---

## Advanced

### Auto-start daemon on login (launchd)

```bash
cat > ~/Library/LaunchAgents/com.ai-beacon.daemon.plist << 'EOF'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>             <string>com.ai-beacon.daemon</string>
  <key>ProgramArguments</key>  <array><string>/Users/YOUR_USER/.local/bin/beacon</string><string>daemon</string></array>
  <key>RunAtLoad</key>         <true/>
  <key>KeepAlive</key>         <true/>
  <key>StandardOutPath</key>   <string>/tmp/beacon-daemon.log</string>
  <key>StandardErrorPath</key> <string>/tmp/beacon-daemon.log</string>
</dict>
</plist>
EOF
launchctl load ~/Library/LaunchAgents/com.ai-beacon.daemon.plist
```

Replace `YOUR_USER` with your username.

### Using a different board

The RP2040 Zero has its NeoPixel on **GP16**. If you use a different board, edit `firmware/src/config.h`:

```c
#define LED_PIN     16   /* change to your board's LED pin */
#define BUZZER_PIN  15   /* change to your board's buzzer pin */
```

Rebuild and reflash with `./scripts/flash.sh`.

### Customising colours and timings

Edit `firmware/src/config.h` and rebuild.

**Change colours** (RGB, 0–255):
```c
#define DEF_THINKING_R  200
#define DEF_THINKING_G  140
#define DEF_THINKING_B    0   /* yellow */

#define DEF_DONE_R        0
#define DEF_DONE_G      200
#define DEF_DONE_B        0   /* green */
```

**Change durations**:
```c
#define DEF_DONE_MS     30000   /* how long "done" stays before auto-idle (ms) */
#define DEF_WAITING_MS  60000   /* how long "waiting" blinks before auto-idle (ms) */
```

**Disable the buzzer**:
```c
#define DEF_BUZZER_ENABLED  0
```

**Change buzzer frequency** (pitch):
```c
#define DEF_BUZZER_FREQ     880   /* Hz — lower pitch */
#define DEF_BUZZER_DUR_MS   100   /* ms per beep — longer */
```

### Override the serial port

If auto-discovery doesn't find your board:

```bash
export BEACON_SERIAL_PORT=/dev/cu.usbmodem1234
ai-beacon thinking
```

---

## Project Structure

```
firmware/      C firmware (pico-sdk, produces beacon.uf2)
  src/         Source files (config.h, parser, led, buzzer, states, main)
  test/        Native unit tests for the parser (no RP2040 needed)
  pico-sdk/    pico-sdk submodule (pinned to 2.1.1)
host/          Go source for the beacon CLI
scripts/       flash.sh, install.sh
claude/        settings.example.json for Claude Code hooks
```

---

## Contributing

Issues and PRs welcome. The firmware is C (pico-sdk); the host is a single Go binary with no CGO dependencies.

## Licence

MIT

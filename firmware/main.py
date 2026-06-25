"""Main loop — non-blocking state machine for the Status Beacon."""

import sys
import time
import select

import hardware as hw
import states as st

VALID = {b"thinking", b"waiting", b"done", b"idle", b"error", b"ping"}

_state   = "idle"
_phase   = 0
_entered = time.ticks_ms()


def _set_state(new: str) -> None:
    global _state, _phase, _entered
    _state   = new
    _phase   = 0
    _entered = time.ticks_ms()
    if new == "done":
        hw.beep(1200, 80)
        time.sleep_ms(120)
        hw.beep(1200, 80)
    elif new == "idle":
        hw.led_off()
        hw.buzzer_off()


def _read_line() -> str | None:
    """Non-blocking stdin read. Returns stripped line or None."""
    if select.select([sys.stdin], [], [], 0)[0]:
        line = sys.stdin.readline()
        if line:
            return line.strip()
    return None


def _tick() -> None:
    global _phase
    now = time.ticks_ms()

    if _state == "thinking":
        hw.set_color(*st.thinking_tick(_phase))

    elif _state == "waiting":
        hw.set_color(*st.waiting_tick(_phase))
        # buzzer pattern: one beep every 8 seconds
        if _phase > 0 and (_phase % 800) == 0:
            hw.beep(880, 80)
        # auto-expire after WAITING_DURATION_MS
        if time.ticks_diff(now, _entered) >= st.WAITING_DURATION_MS:
            _set_state("idle")

    elif _state == "done":
        hw.set_color(*st.done_tick(_phase))
        if time.ticks_diff(now, _entered) >= st.DONE_DURATION_MS:
            _set_state("idle")

    elif _state == "error":
        hw.set_color(*st.error_color())

    else:  # idle
        hw.led_off()

    _phase += 1


sys.stdout.write("READY\n")

while True:
    line = _read_line()
    if line:
        cmd = line.encode() if isinstance(line, str) else line
        cmd = cmd.split()[0]
        if cmd == b"ping":
            sys.stdout.write("PONG\n")
        elif cmd in VALID:
            _set_state(cmd.decode())
        else:
            sys.stdout.write(f"ERR unknown {cmd.decode()}\n")

    _tick()
    time.sleep_ms(st.TICK_MS)

"""Serial port discovery and direct I/O."""

import glob
import os

import serial


def find_port() -> str | None:
    override = os.environ.get("BEACON_SERIAL_PORT")
    if override:
        return override
    # Use cu.* not tty.* — on macOS, tty.* blocks waiting for DCD signal
    # that MicroPython's USB CDC never asserts.
    candidates = glob.glob("/dev/cu.usbmodem*")
    return candidates[0] if candidates else None


def send_direct(state: str, port: str, timeout_s: float = 0.3) -> bool:
    """Open port, send state\n, close. Returns True on success."""
    s = None
    try:
        s = serial.Serial(port, 115200, timeout=timeout_s, write_timeout=timeout_s)
        s.write(f"{state}\n".encode())
        return True
    except Exception:
        return False
    finally:
        if s is not None:
            try:
                s.close()
            except Exception:
                pass

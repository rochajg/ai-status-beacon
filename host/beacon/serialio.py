"""Serial port discovery and I/O — stub for M2/M3."""

import glob
import os


def find_port() -> str | None:
    override = os.environ.get("BEACON_SERIAL_PORT")
    if override:
        return override
    # Use cu.* not tty.* — on macOS, tty.* blocks waiting for DCD signal
    # that MicroPython's USB CDC never asserts.
    candidates = glob.glob("/dev/cu.usbmodem*")
    return candidates[0] if candidates else None

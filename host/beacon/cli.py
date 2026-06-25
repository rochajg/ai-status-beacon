"""CLI entry point for the Status Beacon."""

import pathlib
import sys

from beacon import serialio

VALID_STATES = {"thinking", "waiting", "done", "idle", "error", "ping"}
SOCKET_TIMEOUT = 0.3
SOCKET_PATH = str(pathlib.Path.home() / ".beacon" / "beacon.sock")


def main():
    args = sys.argv[1:]

    if not args or args[0] in ("-h", "--help"):
        print("Usage: beacon <state> [--direct]")
        print(f"States: {', '.join(sorted(VALID_STATES))}")
        sys.exit(0)

    state = args[0]
    direct = "--direct" in args

    if state not in VALID_STATES:
        # silent — never block Claude Code
        sys.exit(0)

    if direct:
        _send_direct(state)
    else:
        _send_via_socket(state)

    sys.exit(0)


def _send_direct(state: str) -> None:
    port = serialio.find_port()
    if port is None:
        return
    serialio.send_direct(state, port, timeout_s=SOCKET_TIMEOUT)


def _send_via_socket(state: str) -> None:
    serialio.send_to_socket(state, SOCKET_PATH, timeout_s=SOCKET_TIMEOUT)

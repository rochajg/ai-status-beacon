"""CLI entry point for the Status Beacon."""

import sys

from beacon import serialio

VALID_STATES = {"thinking", "waiting", "done", "idle", "error", "ping"}
SOCKET_TIMEOUT = 0.3


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
        _send_via_socket(state) or _noop()

    sys.exit(0)


def _send_direct(state: str) -> None:
    port = serialio.find_port()
    if port is None:
        return
    serialio.send_direct(state, port, timeout_s=SOCKET_TIMEOUT)


def _send_via_socket(state: str) -> bool:
    """Try to send via daemon socket. Returns True if sent."""
    # M3: implemented in Task 6
    return False


def _noop() -> None:
    pass

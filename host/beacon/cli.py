"""CLI entry point — stub for M2."""

import sys


VALID_STATES = {"thinking", "waiting", "done", "idle", "error", "ping"}


def main():
    if len(sys.argv) < 2 or sys.argv[1] in ("-h", "--help"):
        print("Usage: beacon <state>  [--direct]")
        print(f"States: {', '.join(sorted(VALID_STATES))}")
        sys.exit(0)

    state = sys.argv[1]
    if state not in VALID_STATES:
        print(f"Unknown state: {state}", file=sys.stderr)
        sys.exit(0)  # always exit 0 to never block Claude Code

    # TODO(M2): send via daemon socket or --direct serial
    print(f"[beacon stub] state={state}")
    sys.exit(0)

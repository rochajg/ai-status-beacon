"""State definitions: colors, durations, and tick-based animation generators."""

import math
import time

# Colors (R, G, B)
YELLOW = (200, 140, 0)
GREEN  = (0, 200, 0)
RED    = (200, 0, 0)
OFF    = (0, 0, 0)

DONE_DURATION_MS    = 30_000
WAITING_DURATION_MS = 60_000
TICK_MS             = 10


def thinking_tick(phase: int) -> tuple[int, int, int]:
    """Sinusoidal yellow breathing. phase increments each tick."""
    brightness = 0.3 + 0.7 * (0.5 + 0.5 * math.sin(phase * 0.05))
    return (int(YELLOW[0] * brightness), int(YELLOW[1] * brightness), 0)


def waiting_tick(phase: int) -> tuple[int, int, int]:
    """Fast on/off blink (200ms on, 200ms off)."""
    return GREEN if (phase % 40) < 20 else OFF


def done_tick(phase: int) -> tuple[int, int, int]:
    """Slow green blink (500ms on, 500ms off)."""
    return GREEN if (phase % 100) < 50 else OFF


def error_color() -> tuple[int, int, int]:
    return RED

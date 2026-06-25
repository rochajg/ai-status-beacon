#pragma once
#include <stdbool.h>
#include <stdint.h>

typedef enum {
    STATE_THINKING,
    STATE_WAITING,
    STATE_DONE,
    STATE_IDLE,
    STATE_ERROR,
    STATE_PING,
    STATE_UNKNOWN
} BeaconState;

typedef struct {
    BeaconState state;

    /* color */
    bool     has_color;
    uint8_t  r, g, b;

    /* brightness */
    bool     has_brightness;
    uint8_t  brightness;

    /* timing (ms) — only for done/waiting */
    bool     has_timing;
    uint32_t timing_ms;

    /* buzzer */
    bool     has_buzzer;
    bool     buzzer_enabled;
    bool     has_freq;
    uint16_t buzzer_freq;
} Command;

/**
 * Parse one trimmed line (no newline) into *out.
 * Returns true on success; false if state is unknown.
 * Unknown parameters are silently ignored (tolerant parsing).
 */
bool parse_command(const char *line, Command *out);

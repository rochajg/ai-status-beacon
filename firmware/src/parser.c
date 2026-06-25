#include "parser.h"
#include <string.h>
#include <stdlib.h>
#include <stdio.h>

static BeaconState parse_state(const char *s) {
    if (strcmp(s, "thinking") == 0) return STATE_THINKING;
    if (strcmp(s, "waiting")  == 0) return STATE_WAITING;
    if (strcmp(s, "done")     == 0) return STATE_DONE;
    if (strcmp(s, "idle")     == 0) return STATE_IDLE;
    if (strcmp(s, "error")    == 0) return STATE_ERROR;
    if (strcmp(s, "ping")     == 0) return STATE_PING;
    return STATE_UNKNOWN;
}

bool parse_command(const char *line, Command *out) {
    if (!line || *line == '\0') {
        out->state = STATE_UNKNOWN;
        return false;
    }

    /* Zero-initialise so all has_* flags start false. */
    memset(out, 0, sizeof(*out));

    /* Copy line to avoid modifying the original. */
    char buf[256];
    strncpy(buf, line, sizeof(buf) - 1);
    buf[sizeof(buf) - 1] = '\0';

    /* First token is the state name. */
    char *saveptr = NULL;
    char *tok = strtok_r(buf, " \t", &saveptr);
    if (!tok) {
        out->state = STATE_UNKNOWN;
        return false;
    }

    out->state = parse_state(tok);
    if (out->state == STATE_UNKNOWN) return false;

    /* Parse remaining key=value tokens. */
    while ((tok = strtok_r(NULL, " \t", &saveptr)) != NULL) {
        char *eq = strchr(tok, '=');
        if (!eq) continue;
        *eq = '\0';
        const char *key = tok;
        const char *val = eq + 1;

        if (strcmp(key, "color") == 0) {
            int r, g, b;
            if (sscanf(val, "%d,%d,%d", &r, &g, &b) == 3 &&
                r >= 0 && r <= 255 && g >= 0 && g <= 255 && b >= 0 && b <= 255) {
                out->has_color = true;
                out->r = (uint8_t)r;
                out->g = (uint8_t)g;
                out->b = (uint8_t)b;
            }
        } else if (strcmp(key, "brightness") == 0) {
            int v = atoi(val);
            if (v >= 0 && v <= 255) {
                out->has_brightness = true;
                out->brightness = (uint8_t)v;
            }
        } else if (strcmp(key, "timing") == 0) {
            long v = atol(val);
            if (v >= 0) {
                out->has_timing = true;
                out->timing_ms = (uint32_t)v;
            }
        } else if (strcmp(key, "buzzer") == 0) {
            out->has_buzzer = true;
            out->buzzer_enabled = (atoi(val) != 0);
        } else if (strcmp(key, "freq") == 0) {
            int v = atoi(val);
            if (v >= 100 && v <= 5000) {
                out->has_freq = true;
                out->buzzer_freq = (uint16_t)v;
            }
        }
        /* Unknown keys are silently ignored (tolerant parsing). */
    }
    return true;
}

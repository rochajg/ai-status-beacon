#include "states.h"
#include "led.h"
#include "buzzer.h"
#include "config.h"
#include "pico/stdlib.h"
#include <math.h>
#include <stdint.h>

/* --- Active command (resolved against defaults at set time) --- */
static struct {
    BeaconState state;
    uint8_t  r, g, b, brightness;
    uint32_t timing_ms;
    bool     buzzer_enabled;
    uint16_t buzzer_freq;
} cur;

static uint32_t phase       = 0;
static uint64_t entered_us  = 0;
static uint64_t buzzer_off_us = 0;  /* 0 = buzzer not scheduled */

/* --- Helpers --------------------------------------------------------- */
static uint8_t sc(uint8_t ch, uint8_t br) {
    return (uint8_t)(((uint16_t)ch * br) / 255);
}

static void apply_led(uint8_t r, uint8_t g, uint8_t b) {
    led_set(sc(r, cur.brightness), sc(g, cur.brightness), sc(b, cur.brightness), 255);
}

static void start_buzzer(void) {
    if (!cur.buzzer_enabled) return;
    buzzer_start(BUZZER_PIN, cur.buzzer_freq);
    buzzer_off_us = time_us_64() + (uint64_t)DEF_BUZZER_DUR_MS * 1000;
}

/* --- Public API ------------------------------------------------------ */
void states_init(void) {
    led_init(LED_PIN);
    buzzer_init(BUZZER_PIN);
    /* Start idle */
    cur.state          = STATE_IDLE;
    cur.r = cur.g = cur.b = 0;
    cur.brightness     = DEF_BRIGHTNESS;
    cur.timing_ms      = 0;
    cur.buzzer_enabled = DEF_BUZZER_ENABLED;
    cur.buzzer_freq    = DEF_BUZZER_FREQ;
    phase              = 0;
    entered_us         = time_us_64();
    buzzer_off_us      = 0;
    /* Do NOT call led_off() here — pio_sm_put_blocking stalls if the PIO
     * FIFO is full and no data has been read yet. The LED is dark on power-up
     * by default; the first states_tick() will handle idle state correctly. */
}

void states_set(const Command *cmd) {
    /* Resolve command against compiled defaults */
    cur.state      = cmd->state;
    cur.brightness = cmd->has_brightness ? cmd->brightness  : DEF_BRIGHTNESS;
    cur.buzzer_enabled = cmd->has_buzzer ? cmd->buzzer_enabled : (bool)DEF_BUZZER_ENABLED;
    cur.buzzer_freq    = cmd->has_freq   ? cmd->buzzer_freq    : DEF_BUZZER_FREQ;

    /* Override colour defaults per state when no colour was sent */
    if (cmd->has_color) {
        cur.r = cmd->r;
        cur.g = cmd->g;
        cur.b = cmd->b;
    } else {
        switch (cmd->state) {
        case STATE_THINKING: cur.r=DEF_THINKING_R; cur.g=DEF_THINKING_G; cur.b=DEF_THINKING_B; break;
        case STATE_WAITING:  cur.r=DEF_WAITING_R;  cur.g=DEF_WAITING_G;  cur.b=DEF_WAITING_B;  break;
        case STATE_DONE:     cur.r=DEF_DONE_R;     cur.g=DEF_DONE_G;     cur.b=DEF_DONE_B;     break;
        case STATE_ERROR:    cur.r=DEF_ERROR_R;    cur.g=DEF_ERROR_G;    cur.b=DEF_ERROR_B;     break;
        default: cur.r=0; cur.g=0; cur.b=0; break;
        }
    }

    switch (cmd->state) {
    case STATE_DONE:    cur.timing_ms = cmd->has_timing ? cmd->timing_ms : DEF_DONE_MS;    break;
    case STATE_WAITING: cur.timing_ms = cmd->has_timing ? cmd->timing_ms : DEF_WAITING_MS; break;
    default:            cur.timing_ms = 0; break;
    }

    phase      = 0;
    entered_us = time_us_64();
    buzzer_off_us = 0;
    buzzer_stop(BUZZER_PIN);

    /* Immediate on-enter effects */
    if (cmd->state == STATE_DONE) {
        /* First of two beeps at the start */
        start_buzzer();
    } else if (cmd->state == STATE_IDLE || cmd->state == STATE_PING) {
        led_off();
    }
}

void states_tick(void) {
    uint64_t now = time_us_64();

    /* Stop scheduled buzzer */
    if (buzzer_off_us && now >= buzzer_off_us) {
        buzzer_stop(BUZZER_PIN);
        buzzer_off_us = 0;
    }

    switch (cur.state) {
    case STATE_THINKING: {
        /* Sinusoidal brightness breathing */
        float t = (float)phase * 0.05f;
        float bright_scale = 0.3f + 0.7f * (0.5f + 0.5f * sinf(t));
        uint8_t br = (uint8_t)((float)cur.brightness * bright_scale);
        led_set(sc(cur.r, br), sc(cur.g, br), sc(cur.b, br), 255);
        break;
    }

    case STATE_WAITING: {
        /* Fast blink: 200ms on / 200ms off (20 ticks each at 10ms) */
        if ((phase % 40) < 20) {
            apply_led(cur.r, cur.g, cur.b);
        } else {
            led_off();
        }
        /* Buzzer pattern: one short beep every 8 seconds */
        if (phase > 0 && (phase % 800) == 0) {
            start_buzzer();
        }
        /* Auto-expire */
        if (cur.timing_ms && (now - entered_us) >= (uint64_t)cur.timing_ms * 1000) {
            Command idle = { .state = STATE_IDLE };
            states_set(&idle);
            return;
        }
        break;
    }

    case STATE_DONE: {
        /* Slow blink: 500ms on / 500ms off (50 ticks each) */
        if ((phase % 100) < 50) {
            apply_led(cur.r, cur.g, cur.b);
        } else {
            led_off();
        }
        /* Second beep 120ms after first */
        if (phase == 12 && cur.buzzer_enabled) {
            start_buzzer();
        }
        /* Auto-expire */
        if (cur.timing_ms && (now - entered_us) >= (uint64_t)cur.timing_ms * 1000) {
            Command idle = { .state = STATE_IDLE };
            states_set(&idle);
            return;
        }
        break;
    }

    case STATE_ERROR:
        apply_led(cur.r, cur.g, cur.b);
        break;

    case STATE_IDLE:
    default:
        led_off();
        break;
    }
    phase++;
}

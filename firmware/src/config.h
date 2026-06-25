#pragma once

/* ── Board pin assignments ────────────────────────────── */
#define LED_PIN     16   /* WS2812 NeoPixel (GP16 on Waveshare RP2040 Zero) */
#define BUZZER_PIN  15   /* Passive buzzer  (GP15) */

/* ── Tick interval ────────────────────────────────────── */
#define TICK_MS     10

/* ── Compiled colour defaults (R, G, B each 0-255) ───── */
#define DEF_THINKING_R  200
#define DEF_THINKING_G  140
#define DEF_THINKING_B    0

#define DEF_WAITING_R     0
#define DEF_WAITING_G   200
#define DEF_WAITING_B     0

#define DEF_DONE_R        0
#define DEF_DONE_G      200
#define DEF_DONE_B        0

#define DEF_ERROR_R     200
#define DEF_ERROR_G       0
#define DEF_ERROR_B       0

/* ── Compiled brightness default (0-255) ─────────────── */
#define DEF_BRIGHTNESS  255

/* ── Compiled timing defaults (milliseconds) ─────────── */
#define DEF_DONE_MS     30000
#define DEF_WAITING_MS  60000

/* ── Compiled buzzer defaults ────────────────────────── */
#define DEF_BUZZER_ENABLED  1
#define DEF_BUZZER_FREQ     1200   /* Hz */
#define DEF_BUZZER_DUR_MS     80   /* ms per beep */

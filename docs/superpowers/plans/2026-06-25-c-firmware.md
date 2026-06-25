# Status Beacon — C Firmware Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the MicroPython firmware with a self-contained C firmware (pico-sdk) that produces a single `beacon.uf2` — no external MicroPython runtime required. The firmware accepts the existing bare protocol PLUS the extended `state key=value ...` protocol and falls back to compiled defaults for unset parameters.

**Architecture:** pico-sdk project in `firmware/` (Python files removed). WS2812 driven via PIO (GP16), buzzer via PWM (GP15), USB CDC via TinyUSB. Non-blocking tick loop at ~10ms. Parser is pure C with no RP2040 dependencies, unit-tested natively. GitHub Actions builds the UF2 in CI.

**Tech Stack:** pico-sdk (git submodule, pinned tag), arm-none-eabi-gcc, CMake 3.13+, Unity (single-header test framework for parser unit tests, native compilation).

## Global Constraints

- Board: Waveshare RP2040 Zero (RP2040 chip, 2MB flash)
- LED pin: GP16 (WS2812 NeoPixel, 1 pixel)
- Buzzer pin: GP15 (passive, PWM)
- Baud: 115200 (USB CDC, configurable via `CMakeLists.txt`)
- Tick interval: 10ms
- Protocol: `<state>[ key=value ...]\n`, one command per line
- Valid states: `thinking`, `waiting`, `done`, `idle`, `error`, `ping`
- Bare command (`thinking\n`) uses compiled defaults — must work with any serial terminal
- Compiled defaults match current MicroPython values exactly (see Task 1)
- Output: `READY\n` on boot, `PONG\n` for ping, `ERR unknown <cmd>\n` for unknowns
- pico-sdk pinned to tag `2.1.1` (latest stable at spec date)
- Commits: Conventional Commits in English
- Co-author: `Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>`

---

## File Map

```
firmware/
  CMakeLists.txt                  ← project definition, pico-sdk bootstrap
  pico_sdk_import.cmake           ← standard pico-sdk cmake bootstrap (copied from SDK)
  .gitmodules (repo root)         ← pico-sdk as git submodule at firmware/pico-sdk/
  src/
    config.h                      ← compiled defaults + pin defines (edit here to change board)
    parser.h / parser.c          ← parse command lines into Command struct (no RP2040 deps)
    led.h / led.c                ← WS2812 NeoPixel via PIO
    buzzer.h / buzzer.c          ← passive buzzer via PWM (non-blocking)
    states.h / states.c          ← state machine + per-tick animation functions
    ws2812.pio                   ← PIO program for WS2812 bit timing
    main.c                       ← USB CDC read loop + main() entry point
  test/
    unity.h / unity.c            ← Unity single-header test framework (copy from unity repo)
    test_parser.c                ← unit tests for parser, compiled natively (no RP2040)
    CMakeLists.txt               ← native (host) test build
```

---

### Task 1: pico-sdk submodule + CMakeLists.txt + config.h

**Files:**
- Create: `firmware/CMakeLists.txt`
- Create: `firmware/pico_sdk_import.cmake`
- Create: `firmware/src/config.h`
- Add: `pico-sdk` as git submodule at `firmware/pico-sdk/`

- [ ] **Step 1: Add pico-sdk as git submodule**

```bash
git submodule add -b 2.1.1 https://github.com/raspberrypi/pico-sdk.git firmware/pico-sdk
git submodule update --init --recursive firmware/pico-sdk
```

Expected: `firmware/pico-sdk/` populated, `.gitmodules` updated.

- [ ] **Step 2: Copy pico_sdk_import.cmake**

```bash
cp firmware/pico-sdk/external/pico_sdk_import.cmake firmware/pico_sdk_import.cmake
```

- [ ] **Step 3: Create `firmware/CMakeLists.txt`**

```cmake
cmake_minimum_required(VERSION 3.13)

# Bootstrap the pico-sdk before the project() call.
set(PICO_SDK_PATH "${CMAKE_CURRENT_SOURCE_DIR}/pico-sdk")
include(pico_sdk_import.cmake)

project(beacon C CXX ASM)
set(CMAKE_C_STANDARD 11)
set(CMAKE_CXX_STANDARD 17)

pico_sdk_init()

add_executable(beacon
    src/main.c
    src/parser.c
    src/led.c
    src/buzzer.c
    src/states.c
)

# Generate PIO header from ws2812.pio
pico_generate_pio_header(beacon ${CMAKE_CURRENT_LIST_DIR}/src/ws2812.pio)

target_include_directories(beacon PRIVATE src)

target_link_libraries(beacon
    pico_stdlib
    pico_unique_id
    hardware_pio
    hardware_pwm
    hardware_clocks
    tinyusb_device
    tinyusb_board
)

# Enable USB CDC (replaces UART stdio)
pico_enable_stdio_usb(beacon 1)
pico_enable_stdio_uart(beacon 0)

# Create map/bin/hex/uf2 output files alongside ELF
pico_add_extra_outputs(beacon)
```

- [ ] **Step 4: Create `firmware/src/config.h`**

```c
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
```

- [ ] **Step 5: Verify the project configures (no source files yet)**

```bash
# Install toolchain on macOS if needed:
# brew install arm-none-eabi-gcc cmake

mkdir -p firmware/build
cd firmware/build
cmake .. 2>&1 | tail -10
```

Expected: cmake completes (may warn about missing source files — that's fine at this stage). The key check is that pico-sdk is found.

- [ ] **Step 6: Add `firmware/build/` to .gitignore**

Append to the root `.gitignore`:
```
firmware/build/
```

- [ ] **Step 7: Commit**

```bash
git add .gitmodules firmware/pico-sdk firmware/CMakeLists.txt firmware/pico_sdk_import.cmake firmware/src/config.h .gitignore
git commit -m "$(cat <<'EOF'
feat(firmware): add pico-sdk submodule and CMakeLists scaffold

pico-sdk pinned to tag 2.1.1 as a git submodule.
CMakeLists.txt configures USB CDC, PIO, PWM, TinyUSB.
config.h defines all board pins and compiled defaults.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 2: Parser (TDD, native)

**Files:**
- Create: `firmware/src/parser.h`
- Create: `firmware/src/parser.c`
- Create: `firmware/test/CMakeLists.txt`
- Create: `firmware/test/unity.h` + `firmware/test/unity.c` (copy from Unity repo)
- Create: `firmware/test/test_parser.c`

**Produces:**
- `parse_command(const char *line, Command *out) -> bool`
- `Command` struct with state, color, brightness, timing, buzzer fields + has_* flags

- [ ] **Step 1: Download Unity test framework**

```bash
# Single-file download (no submodule needed)
mkdir -p firmware/test
curl -fsSL https://raw.githubusercontent.com/ThrowTheSwitch/Unity/master/src/unity.h -o firmware/test/unity.h
curl -fsSL https://raw.githubusercontent.com/ThrowTheSwitch/Unity/master/src/unity.c -o firmware/test/unity.c
```

- [ ] **Step 2: Create `firmware/src/parser.h`**

```c
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
```

- [ ] **Step 3: Write failing tests — `firmware/test/test_parser.c`**

```c
#include "unity.h"
#include "../src/parser.h"
#include <string.h>

void setUp(void)   {}
void tearDown(void) {}

void test_bare_thinking(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("thinking", &cmd));
    TEST_ASSERT_EQUAL(STATE_THINKING, cmd.state);
    TEST_ASSERT_FALSE(cmd.has_color);
    TEST_ASSERT_FALSE(cmd.has_brightness);
    TEST_ASSERT_FALSE(cmd.has_timing);
    TEST_ASSERT_FALSE(cmd.has_buzzer);
}

void test_bare_ping(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("ping", &cmd));
    TEST_ASSERT_EQUAL(STATE_PING, cmd.state);
}

void test_unknown_state_returns_false(void) {
    Command cmd;
    TEST_ASSERT_FALSE(parse_command("launch_missiles", &cmd));
    TEST_ASSERT_EQUAL(STATE_UNKNOWN, cmd.state);
}

void test_color_param(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("thinking color=0,100,255", &cmd));
    TEST_ASSERT_EQUAL(STATE_THINKING, cmd.state);
    TEST_ASSERT_TRUE(cmd.has_color);
    TEST_ASSERT_EQUAL(0,   cmd.r);
    TEST_ASSERT_EQUAL(100, cmd.g);
    TEST_ASSERT_EQUAL(255, cmd.b);
}

void test_brightness_param(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("thinking brightness=180", &cmd));
    TEST_ASSERT_TRUE(cmd.has_brightness);
    TEST_ASSERT_EQUAL(180, cmd.brightness);
}

void test_timing_param(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("done timing=60000", &cmd));
    TEST_ASSERT_TRUE(cmd.has_timing);
    TEST_ASSERT_EQUAL(60000, cmd.timing_ms);
}

void test_buzzer_disabled(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("waiting buzzer=0", &cmd));
    TEST_ASSERT_TRUE(cmd.has_buzzer);
    TEST_ASSERT_FALSE(cmd.buzzer_enabled);
}

void test_buzzer_enabled(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("waiting buzzer=1", &cmd));
    TEST_ASSERT_TRUE(cmd.has_buzzer);
    TEST_ASSERT_TRUE(cmd.buzzer_enabled);
}

void test_freq_param(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("done freq=880", &cmd));
    TEST_ASSERT_TRUE(cmd.has_freq);
    TEST_ASSERT_EQUAL(880, cmd.buzzer_freq);
}

void test_multiple_params(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("thinking color=0,100,255 brightness=200", &cmd));
    TEST_ASSERT_TRUE(cmd.has_color);
    TEST_ASSERT_EQUAL(0,   cmd.r);
    TEST_ASSERT_EQUAL(100, cmd.g);
    TEST_ASSERT_EQUAL(255, cmd.b);
    TEST_ASSERT_TRUE(cmd.has_brightness);
    TEST_ASSERT_EQUAL(200, cmd.brightness);
}

void test_unknown_param_ignored(void) {
    Command cmd;
    /* tolerant: unknown key silently ignored */
    TEST_ASSERT_TRUE(parse_command("thinking rainbow=1,2,3", &cmd));
    TEST_ASSERT_EQUAL(STATE_THINKING, cmd.state);
    TEST_ASSERT_FALSE(cmd.has_color);
}

void test_empty_line_returns_false(void) {
    Command cmd;
    TEST_ASSERT_FALSE(parse_command("", &cmd));
}

int main(void) {
    UNITY_BEGIN();
    RUN_TEST(test_bare_thinking);
    RUN_TEST(test_bare_ping);
    RUN_TEST(test_unknown_state_returns_false);
    RUN_TEST(test_color_param);
    RUN_TEST(test_brightness_param);
    RUN_TEST(test_timing_param);
    RUN_TEST(test_buzzer_disabled);
    RUN_TEST(test_buzzer_enabled);
    RUN_TEST(test_freq_param);
    RUN_TEST(test_multiple_params);
    RUN_TEST(test_unknown_param_ignored);
    RUN_TEST(test_empty_line_returns_false);
    return UNITY_END();
}
```

- [ ] **Step 4: Create `firmware/test/CMakeLists.txt`**

```cmake
cmake_minimum_required(VERSION 3.13)
project(beacon_tests C)
set(CMAKE_C_STANDARD 11)

add_executable(test_parser
    test_parser.c
    unity.c
    ../src/parser.c
)
target_include_directories(test_parser PRIVATE ../src)
```

- [ ] **Step 5: Build and run tests natively — expect FAIL (parser.c missing)**

```bash
mkdir -p firmware/test/build
cd firmware/test/build
cmake .. && make 2>&1 | tail -5
```

Expected: compile error — `parser.c` not found or functions undefined.

- [ ] **Step 6: Implement `firmware/src/parser.c`**

```c
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
```

- [ ] **Step 7: Build and run — expect all 12 tests PASS**

```bash
cd firmware/test/build
cmake .. && make && ./test_parser
```

Expected:
```
12 Tests 0 Failures 0 Ignored
OK
```

- [ ] **Step 8: Commit**

```bash
git add firmware/src/parser.h firmware/src/parser.c firmware/test/
git commit -m "$(cat <<'EOF'
feat(firmware): add C parser for extended serial protocol

Tolerant parser: unknown state returns false, unknown params ignored.
Parses color=R,G,B, brightness=N, timing=Ms, buzzer=0|1, freq=Hz.
12/12 unit tests pass natively (no RP2040 required).

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 3: LED driver (WS2812 via PIO)

**Files:**
- Create: `firmware/src/ws2812.pio`
- Create: `firmware/src/led.h`
- Create: `firmware/src/led.c`

**Produces:**
- `led_init(uint pin)` — initialise PIO for WS2812
- `led_set(uint8_t r, uint8_t g, uint8_t b, uint8_t brightness)` — set pixel color

- [ ] **Step 1: Copy `ws2812.pio` from pico-examples**

The official pico-examples WS2812 PIO program is proven and handles all
timing correctly. Copy it directly rather than writing from scratch:

```bash
curl -fsSL \
  https://raw.githubusercontent.com/raspberrypi/pico-examples/master/pio/ws2812/ws2812.pio \
  -o firmware/src/ws2812.pio
```

Verify the file has these key sections (timing defines + `ws2812_program_init`):
- `.define PUBLIC T1 2` / `T2 5` / `T3 3`  
- `ws2812_program_init` in the `% c-sdk { ... %}` block

These defines are what `led.c` uses to calculate `cycles_per_bit`:
```c
int cycles_per_bit = ws2812_T1 + ws2812_T2 + ws2812_T3;  // = 10
```

- [ ] **Step 2: Create `firmware/src/led.h`**

```c
#pragma once
#include <stdint.h>

void led_init(uint pin);

/**
 * Set the NeoPixel color. brightness (0-255) is applied as a global
 * scale before sending to the PIO state machine.
 */
void led_set(uint8_t r, uint8_t g, uint8_t b, uint8_t brightness);

void led_off(void);
```

- [ ] **Step 3: Create `firmware/src/led.c`**

```c
#include "led.h"
#include "ws2812.pio.h"
#include "hardware/pio.h"
#include "hardware/clocks.h"

static PIO  _pio = pio0;
static uint _sm  = 0;

void led_init(uint pin) {
    uint offset = pio_add_program(_pio, &ws2812_program);
    ws2812_program_init(_pio, _sm, offset, pin, 800000, false);
}

static inline uint32_t urgb_u32(uint8_t r, uint8_t g, uint8_t b) {
    /* WS2812 expects GRB order in the top 24 bits */
    return ((uint32_t)g << 24) | ((uint32_t)r << 16) | ((uint32_t)b << 8);
}

static inline uint8_t scale(uint8_t channel, uint8_t brightness) {
    return (uint8_t)(((uint16_t)channel * brightness) / 255);
}

void led_set(uint8_t r, uint8_t g, uint8_t b, uint8_t brightness) {
    uint8_t sr = scale(r, brightness);
    uint8_t sg = scale(g, brightness);
    uint8_t sb = scale(b, brightness);
    pio_sm_put_blocking(_pio, _sm, urgb_u32(sr, sg, sb));
}

void led_off(void) {
    led_set(0, 0, 0, 255);
}
```

- [ ] **Step 4: Build firmware (partial — main.c placeholder)**

Create a minimal `firmware/src/main.c` for build verification only (will be replaced in Task 5):

```c
#include "pico/stdlib.h"
#include "led.h"
#include "config.h"

int main(void) {
    stdio_init_all();
    led_init(LED_PIN);
    led_set(200, 140, 0, DEF_BRIGHTNESS); /* yellow — thinking */
    while (1) tight_loop_contents();
}
```

```bash
mkdir -p firmware/build
cd firmware/build
cmake .. && make beacon 2>&1 | tail -10
```

Expected: `beacon.uf2` produced in `firmware/build/`.

- [ ] **Step 5: Flash and verify on hardware**

Hold BOOT on RP2040 Zero, plug in, copy `firmware/build/beacon.uf2` to `RPI-RP2` drive. LED should turn **solid yellow** after reboot.

- [ ] **Step 6: Commit**

```bash
git add firmware/src/ws2812.pio firmware/src/led.h firmware/src/led.c firmware/src/main.c
git commit -m "$(cat <<'EOF'
feat(firmware): add WS2812 LED driver via PIO

PIO program at 800kHz, GRB byte order, brightness scaling.
led_init/led_set/led_off public API. Placeholder main.c shows
solid yellow on boot for hardware verification.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 4: Buzzer driver (PWM, non-blocking)

**Files:**
- Create: `firmware/src/buzzer.h`
- Create: `firmware/src/buzzer.c`

**Produces:**
- `buzzer_init(uint pin)` — configure PWM slice
- `buzzer_start(uint pin, uint freq_hz)` — enable PWM at frequency
- `buzzer_stop(uint pin)` — disable PWM (buzzer silent)

The calling code (state machine) tracks stop time via `time_us_64()`.

- [ ] **Step 1: Create `firmware/src/buzzer.h`**

```c
#pragma once
#include <stdint.h>

void buzzer_init(uint pin);
void buzzer_start(uint pin, uint freq_hz);
void buzzer_stop(uint pin);
```

- [ ] **Step 2: Create `firmware/src/buzzer.c`**

```c
#include "buzzer.h"
#include "hardware/pwm.h"
#include "hardware/gpio.h"
#include "hardware/clocks.h"

void buzzer_init(uint pin) {
    gpio_set_function(pin, GPIO_FUNC_PWM);
    uint slice = pwm_gpio_to_slice_num(pin);
    pwm_set_enabled(slice, false);
}

void buzzer_start(uint pin, uint freq_hz) {
    uint slice = pwm_gpio_to_slice_num(pin);
    uint chan  = pwm_gpio_to_channel(pin);

    uint32_t sys_hz = clock_get_hz(clk_sys); /* typically 125 000 000 */
    uint32_t wrap   = 4095u;                 /* 12-bit resolution */
    float    divider = (float)sys_hz / ((float)freq_hz * (wrap + 1));
    if (divider < 1.0f) divider = 1.0f;

    pwm_set_clkdiv(slice, divider);
    pwm_set_wrap(slice, wrap);
    pwm_set_chan_level(slice, chan, wrap / 2); /* 50% duty → maximum volume */
    pwm_set_enabled(slice, true);
}

void buzzer_stop(uint pin) {
    pwm_set_enabled(pwm_gpio_to_slice_num(pin), false);
}
```

- [ ] **Step 3: Update placeholder main.c to test the buzzer**

Replace `firmware/src/main.c` with:

```c
#include "pico/stdlib.h"
#include "led.h"
#include "buzzer.h"
#include "config.h"

int main(void) {
    stdio_init_all();
    led_init(LED_PIN);
    buzzer_init(BUZZER_PIN);

    /* Two beeps, then solid green — mirrors the "done" state. */
    led_set(DEF_DONE_R, DEF_DONE_G, DEF_DONE_B, DEF_BRIGHTNESS);
    buzzer_start(BUZZER_PIN, DEF_BUZZER_FREQ);
    sleep_ms(DEF_BUZZER_DUR_MS);
    buzzer_stop(BUZZER_PIN);
    sleep_ms(120);
    buzzer_start(BUZZER_PIN, DEF_BUZZER_FREQ);
    sleep_ms(DEF_BUZZER_DUR_MS);
    buzzer_stop(BUZZER_PIN);

    while (1) tight_loop_contents();
}
```

Build, flash, verify: LED solid green + two beeps.

- [ ] **Step 4: Commit**

```bash
git add firmware/src/buzzer.h firmware/src/buzzer.c firmware/src/main.c
git commit -m "$(cat <<'EOF'
feat(firmware): add passive buzzer driver via PWM

Non-blocking API: buzzer_start/stop let the state machine
schedule beep end times via time_us_64(). 50% duty, 12-bit wrap.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 5: State machine + main.c (USB CDC)

**Files:**
- Create: `firmware/src/states.h`
- Create: `firmware/src/states.c`
- Replace: `firmware/src/main.c` (final version)

**Consumes:**
- `parse_command` — Task 2
- `led_init`, `led_set`, `led_off` — Task 3
- `buzzer_init`, `buzzer_start`, `buzzer_stop` — Task 4
- all `DEF_*` constants from `config.h` — Task 1

**Produces:**
- `states_init()`, `states_set(const Command *cmd)`, `states_tick()` — the state machine

- [ ] **Step 1: Create `firmware/src/states.h`**

```c
#pragma once
#include "parser.h"
#include <stdint.h>

void states_init(void);

/** Apply a new command, interrupting the current animation immediately. */
void states_set(const Command *cmd);

/** Advance animation by one tick (~10ms). Called from the main loop. */
void states_tick(void);
```

- [ ] **Step 2: Create `firmware/src/states.c`**

```c
#include "states.h"
#include "led.h"
#include "buzzer.h"
#include "config.h"
#include "pico/stdlib.h"
#include <math.h>
#include <stdint.h>

/* ─── Active command (resolved against defaults at set time) ─── */
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

/* ─── Helpers ──────────────────────────────────────────────── */
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

/* ─── Public API ───────────────────────────────────────────── */
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
    led_off();
}

void states_set(const Command *cmd) {
    /* Resolve command against compiled defaults */
    cur.state      = cmd->state;
    cur.r          = cmd->has_color      ? cmd->r           : DEF_THINKING_R;
    cur.g          = cmd->has_color      ? cmd->g           : DEF_THINKING_G;
    cur.b          = cmd->has_color      ? cmd->b           : DEF_THINKING_B;
    cur.brightness = cmd->has_brightness ? cmd->brightness  : DEF_BRIGHTNESS;
    cur.buzzer_enabled = cmd->has_buzzer ? cmd->buzzer_enabled : (bool)DEF_BUZZER_ENABLED;
    cur.buzzer_freq    = cmd->has_freq   ? cmd->buzzer_freq    : DEF_BUZZER_FREQ;

    /* Override colour defaults per state when no colour was sent */
    if (!cmd->has_color) {
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
        /* Two beeps at the start */
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
```

- [ ] **Step 3: Create final `firmware/src/main.c`**

```c
#include "pico/stdlib.h"
#include "parser.h"
#include "states.h"
#include "config.h"
#include <string.h>
#include <stdio.h>

#define LINE_BUF_SIZE 256

int main(void) {
    stdio_init_all();
    states_init();

    /* Signal readiness to the host */
    printf("READY\n");

    char    buf[LINE_BUF_SIZE];
    size_t  pos = 0;

    while (1) {
        /* Non-blocking character read */
        int c = getchar_timeout_us(0);
        if (c != PICO_ERROR_TIMEOUT) {
            if (c == '\n' || c == '\r') {
                if (pos > 0) {
                    buf[pos] = '\0';
                    Command cmd;
                    if (!parse_command(buf, &cmd)) {
                        printf("ERR unknown %s\n", buf);
                    } else if (cmd.state == STATE_PING) {
                        printf("PONG\n");
                    } else {
                        states_set(&cmd);
                    }
                    pos = 0;
                }
            } else if (pos < LINE_BUF_SIZE - 1) {
                buf[pos++] = (char)c;
            }
        }

        states_tick();
        sleep_ms(TICK_MS);
    }
}
```

- [ ] **Step 4: Update `CMakeLists.txt` — add math library**

Add `m` to `target_link_libraries` (needed for `sinf`):

```cmake
target_link_libraries(beacon
    pico_stdlib
    pico_unique_id
    hardware_pio
    hardware_pwm
    hardware_clocks
    tinyusb_device
    tinyusb_board
    m
)
```

- [ ] **Step 5: Build**

```bash
cd firmware/build && cmake .. && make beacon 2>&1 | tail -5
```

Expected: `beacon.uf2` produced with no errors.

- [ ] **Step 6: Flash and hardware test**

Hold BOOT, plug in, copy `firmware/build/beacon.uf2` to `RPI-RP2`. After reboot:

```bash
# Start daemon
beacon daemon &

# Test all states
beacon thinking   # LED: yellow breathing
beacon waiting    # LED: fast blink (buzzer every 8s if connected)
beacon done       # LED: green blink + 2 beeps
beacon idle       # LED: off
beacon error      # LED: solid red

# Extended params
beacon thinking color=0,100,255 brightness=150
beacon status
```

- [ ] **Step 7: Commit**

```bash
git add firmware/src/states.h firmware/src/states.c firmware/src/main.c firmware/CMakeLists.txt
git commit -m "$(cat <<'EOF'
feat(firmware): add state machine and USB CDC main loop

Non-blocking tick loop at 10ms. States: thinking (sinusoidal breath),
waiting (fast blink + buzzer 8s), done (slow blink + 2 beeps, auto-idle),
idle (off), error (solid red). Extended params override compiled defaults.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 6: Remove Python firmware + update scripts and README

**Files:**
- Delete: `firmware/main.py`, `firmware/states.py`, `firmware/hardware.py`
- Modify: `scripts/flash.sh`
- Modify: `README.md` and `README.pt-br.md`

- [ ] **Step 1: Remove Python firmware files**

```bash
git rm firmware/main.py firmware/states.py firmware/hardware.py
```

- [ ] **Step 2: Rewrite `scripts/flash.sh`**

```bash
#!/usr/bin/env bash
# Flash the beacon.uf2 firmware to a RP2040 Zero in BOOT mode.
# Usage: ./scripts/flash.sh [path/to/beacon.uf2]
set -euo pipefail

UF2="${1:-firmware/build/beacon.uf2}"

if [ ! -f "$UF2" ]; then
  echo "Error: $UF2 not found." >&2
  echo "Build first: cd firmware/build && cmake .. && make beacon" >&2
  exit 1
fi

VOLUME="/Volumes/RPI-RP2"
if [ ! -d "$VOLUME" ]; then
  echo "Error: RPI-RP2 volume not found." >&2
  echo "Hold BOOT on the RP2040 Zero while plugging in the USB cable." >&2
  exit 1
fi

echo "Flashing $UF2 → $VOLUME ..."
cp "$UF2" "$VOLUME/"
echo "Done. The device will reboot automatically."
```

```bash
chmod +x scripts/flash.sh
```

- [ ] **Step 3: Update README quick-start sections**

In `README.md`, replace the "Flash the firmware" step:

```markdown
### 2 — Install the beacon CLI
...

### 3 — Flash the firmware

Download `beacon.uf2` from the [latest release](https://github.com/rochajg/ai-status-beacon/releases/latest).

Hold BOOT on the RP2040 Zero while plugging it in. Drag `beacon.uf2` onto the `RPI-RP2` drive that appears. The board reboots automatically.

That's it — no MicroPython, no extra tools.
```

Remove the mpremote step entirely. Apply the same change to `README.pt-br.md`.

- [ ] **Step 4: Commit**

```bash
git add firmware/ scripts/flash.sh README.md README.pt-br.md
git commit -m "$(cat <<'EOF'
chore(firmware): remove MicroPython files, update flash script

Python firmware replaced by C. flash.sh now copies the pre-built UF2
to the RPI-RP2 mass-storage mount. No mpremote dependency.

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

### Task 7: CI — build beacon.uf2 in GitHub Actions

**Files:**
- Modify: `.github/workflows/release.yml`

- [ ] **Step 1: Replace the firmware job in `.github/workflows/release.yml`**

Replace the existing `firmware:` job with:

```yaml
  firmware:
    name: Build firmware (UF2)
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
        with:
          submodules: recursive   # pulls pico-sdk and its own submodules

      - name: Install ARM toolchain + CMake
        run: |
          sudo apt-get update -qq
          sudo apt-get install -y --no-install-recommends \
            gcc-arm-none-eabi libnewlib-arm-none-eabi cmake ninja-build

      - name: Build
        run: |
          mkdir -p firmware/build
          cd firmware/build
          cmake .. -G Ninja -DCMAKE_BUILD_TYPE=Release
          ninja beacon

      - name: Upload artifact
        uses: actions/upload-artifact@v4
        with:
          name: firmware
          path: firmware/build/beacon.uf2
```

Also update the `release:` job to change `firmware.zip` → `beacon.uf2`:

```yaml
      - name: Generate checksums
        run: |
          cd dist
          sha256sum beacon-darwin-arm64 beacon-darwin-amd64 beacon.uf2 > checksums.txt

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          files: |
            dist/beacon-darwin-arm64
            dist/beacon-darwin-amd64
            dist/beacon.uf2
            dist/checksums.txt
```

Update the release body in the workflow to mention `beacon.uf2` instead of `firmware.zip`.

- [ ] **Step 2: Verify workflow syntax**

```bash
# Install actionlint if available, or just check YAML is valid:
python3 -c "import yaml; yaml.safe_load(open('.github/workflows/release.yml'))" && echo "YAML valid"
```

- [ ] **Step 3: Commit and push — watch CI**

```bash
git add .github/workflows/release.yml
git commit -m "$(cat <<'EOF'
ci: build beacon.uf2 from C firmware in release workflow

Installs arm-none-eabi-gcc + cmake on ubuntu-latest, checks out
pico-sdk submodule recursively, builds with Ninja. Release asset is
beacon.uf2 (replaces firmware.zip).

Co-Authored-By: Claude Sonnet 4.6 (1M context) <noreply@anthropic.com>
EOF
)"
git push origin main
```

Watch the Actions tab to confirm the firmware build succeeds before tagging.

- [ ] **Step 4: Tag new release**

Once CI passes on `main`:

```bash
git tag v0.2.0
git push origin v0.2.0
```

Expected release assets: `beacon-darwin-arm64`, `beacon-darwin-amd64`,
`beacon.uf2`, `checksums.txt`.

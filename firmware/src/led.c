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

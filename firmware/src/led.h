#pragma once
#include <stdint.h>

void led_init(uint pin);

/**
 * Set the NeoPixel color. brightness (0-255) is applied as a global
 * scale before sending to the PIO state machine.
 */
void led_set(uint8_t r, uint8_t g, uint8_t b, uint8_t brightness);

void led_off(void);

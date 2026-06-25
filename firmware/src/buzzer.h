#pragma once
#include <stdint.h>

void buzzer_init(uint32_t pin);
void buzzer_start(uint32_t pin, uint32_t freq_hz);
void buzzer_stop(uint32_t pin);

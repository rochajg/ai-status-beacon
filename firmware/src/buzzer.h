#pragma once
#include <stdint.h>

void buzzer_init(uint pin);
void buzzer_start(uint pin, uint freq_hz);
void buzzer_stop(uint pin);

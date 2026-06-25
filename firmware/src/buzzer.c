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
    pwm_set_chan_level(slice, chan, wrap / 2); /* 50% duty -> maximum volume */
    pwm_set_enabled(slice, true);
}

void buzzer_stop(uint pin) {
    pwm_set_enabled(pwm_gpio_to_slice_num(pin), false);
}

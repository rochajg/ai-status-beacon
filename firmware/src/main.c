#include "pico/stdlib.h"
#include "led.h"
#include "config.h"

int main(void) {
    stdio_init_all();
    led_init(LED_PIN);
    led_set(200, 140, 0, DEF_BRIGHTNESS); /* yellow — thinking */
    while (1) tight_loop_contents();
}

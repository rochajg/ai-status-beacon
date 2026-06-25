#include "pico/stdlib.h"
#include "led.h"
#include "buzzer.h"
#include "config.h"

int main(void) {
    stdio_init_all();
    led_init(LED_PIN);
    buzzer_init(BUZZER_PIN);

    /* Two beeps, then solid green -- mirrors the "done" state. */
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

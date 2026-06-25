#include "pico/stdlib.h"
#include "parser.h"
#include "states.h"
#include "config.h"
#include <string.h>
#include <stdio.h>

#define LINE_BUF_SIZE 256

int main(void) {
    stdio_init_all();

    /* Give USB 2s to enumerate on the host before hardware init.
     * If hardware init hangs, USB will already be visible for diagnosis. */
    sleep_ms(2000);

    states_init();
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

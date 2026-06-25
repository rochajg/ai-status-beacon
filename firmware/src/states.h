#pragma once
#include "parser.h"
#include <stdint.h>

void states_init(void);

/** Apply a new command, interrupting the current animation immediately. */
void states_set(const Command *cmd);

/** Advance animation by one tick (~10ms). Called from the main loop. */
void states_tick(void);

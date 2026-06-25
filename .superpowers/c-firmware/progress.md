# C Firmware Plan — Progress Ledger
Plan: docs/superpowers/plans/2026-06-25-c-firmware.md
Worktree: /Users/jorocha/pessoal/ai-status-beacon-c-firmware
Branch: feature/c-firmware
Started: 2026-06-25

## Tasks
- [x] Task 1: pico-sdk submodule + CMakeLists + config.h (commits cb3f642..040251a, review clean)
- [x] Task 2: Parser (TDD, native) (commits 040251a..46f08eb, 12/12 tests pass, review clean)
- [x] Task 3: LED driver WS2812 PIO (commits 46f08eb..66bfc38, UF2 build pending ARM toolchain in CI, review clean)
- [x] Task 4: Buzzer driver PWM (commits 66bfc38..e03e52a, UF2 build pending ARM toolchain in CI, review clean)
- [x] Task 5: State machine + main.c USB CDC (commits e03e52a..d5ec295, UF2 build pending ARM toolchain in CI, review clean)
- [x] Task 6: Remove Python firmware + update scripts (commits d5ec295..87546ba, review clean)
- [x] Task 7: CI build UF2 (commits 87546ba..1d5b902, YAML valid, review clean)

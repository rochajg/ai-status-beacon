#include "unity.h"
#include "../src/parser.h"
#include <string.h>

void setUp(void)   {}
void tearDown(void) {}

void test_bare_thinking(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("thinking", &cmd));
    TEST_ASSERT_EQUAL(STATE_THINKING, cmd.state);
    TEST_ASSERT_FALSE(cmd.has_color);
    TEST_ASSERT_FALSE(cmd.has_brightness);
    TEST_ASSERT_FALSE(cmd.has_timing);
    TEST_ASSERT_FALSE(cmd.has_buzzer);
}

void test_bare_ping(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("ping", &cmd));
    TEST_ASSERT_EQUAL(STATE_PING, cmd.state);
}

void test_unknown_state_returns_false(void) {
    Command cmd;
    TEST_ASSERT_FALSE(parse_command("launch_missiles", &cmd));
    TEST_ASSERT_EQUAL(STATE_UNKNOWN, cmd.state);
}

void test_color_param(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("thinking color=0,100,255", &cmd));
    TEST_ASSERT_EQUAL(STATE_THINKING, cmd.state);
    TEST_ASSERT_TRUE(cmd.has_color);
    TEST_ASSERT_EQUAL(0,   cmd.r);
    TEST_ASSERT_EQUAL(100, cmd.g);
    TEST_ASSERT_EQUAL(255, cmd.b);
}

void test_brightness_param(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("thinking brightness=180", &cmd));
    TEST_ASSERT_TRUE(cmd.has_brightness);
    TEST_ASSERT_EQUAL(180, cmd.brightness);
}

void test_timing_param(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("done timing=60000", &cmd));
    TEST_ASSERT_TRUE(cmd.has_timing);
    TEST_ASSERT_EQUAL(60000, cmd.timing_ms);
}

void test_buzzer_disabled(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("waiting buzzer=0", &cmd));
    TEST_ASSERT_TRUE(cmd.has_buzzer);
    TEST_ASSERT_FALSE(cmd.buzzer_enabled);
}

void test_buzzer_enabled(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("waiting buzzer=1", &cmd));
    TEST_ASSERT_TRUE(cmd.has_buzzer);
    TEST_ASSERT_TRUE(cmd.buzzer_enabled);
}

void test_freq_param(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("done freq=880", &cmd));
    TEST_ASSERT_TRUE(cmd.has_freq);
    TEST_ASSERT_EQUAL(880, cmd.buzzer_freq);
}

void test_multiple_params(void) {
    Command cmd;
    TEST_ASSERT_TRUE(parse_command("thinking color=0,100,255 brightness=200", &cmd));
    TEST_ASSERT_TRUE(cmd.has_color);
    TEST_ASSERT_EQUAL(0,   cmd.r);
    TEST_ASSERT_EQUAL(100, cmd.g);
    TEST_ASSERT_EQUAL(255, cmd.b);
    TEST_ASSERT_TRUE(cmd.has_brightness);
    TEST_ASSERT_EQUAL(200, cmd.brightness);
}

void test_unknown_param_ignored(void) {
    Command cmd;
    /* tolerant: unknown key silently ignored */
    TEST_ASSERT_TRUE(parse_command("thinking rainbow=1,2,3", &cmd));
    TEST_ASSERT_EQUAL(STATE_THINKING, cmd.state);
    TEST_ASSERT_FALSE(cmd.has_color);
}

void test_empty_line_returns_false(void) {
    Command cmd;
    TEST_ASSERT_FALSE(parse_command("", &cmd));
}

int main(void) {
    UNITY_BEGIN();
    RUN_TEST(test_bare_thinking);
    RUN_TEST(test_bare_ping);
    RUN_TEST(test_unknown_state_returns_false);
    RUN_TEST(test_color_param);
    RUN_TEST(test_brightness_param);
    RUN_TEST(test_timing_param);
    RUN_TEST(test_buzzer_disabled);
    RUN_TEST(test_buzzer_enabled);
    RUN_TEST(test_freq_param);
    RUN_TEST(test_multiple_params);
    RUN_TEST(test_unknown_param_ignored);
    RUN_TEST(test_empty_line_returns_false);
    return UNITY_END();
}

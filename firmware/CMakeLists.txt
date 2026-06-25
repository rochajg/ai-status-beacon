cmake_minimum_required(VERSION 3.13)

# Bootstrap the pico-sdk before the project() call.
set(PICO_SDK_PATH "${CMAKE_CURRENT_SOURCE_DIR}/pico-sdk")
include(pico_sdk_import.cmake)

project(beacon C CXX ASM)
set(CMAKE_C_STANDARD 11)
set(CMAKE_CXX_STANDARD 17)

pico_sdk_init()

add_executable(beacon
    src/main.c
    src/parser.c
    src/led.c
    src/buzzer.c
    src/states.c
)

# Generate PIO header from ws2812.pio
pico_generate_pio_header(beacon ${CMAKE_CURRENT_LIST_DIR}/src/ws2812.pio)

target_include_directories(beacon PRIVATE src)

target_link_libraries(beacon
    pico_stdlib
    pico_unique_id
    hardware_pio
    hardware_pwm
    hardware_clocks
    m
)

# Enable USB CDC — pico_enable_stdio_usb internally links tinyusb_device
# and tinyusb_board. DO NOT link them manually or USB init conflicts.
pico_enable_stdio_usb(beacon 1)
pico_enable_stdio_uart(beacon 0)

# Create map/bin/hex/uf2 output files alongside ELF
pico_add_extra_outputs(beacon)

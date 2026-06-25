cmake_minimum_required(VERSION 3.13)
project(beacon_tests C)
set(CMAKE_C_STANDARD 11)

add_executable(test_parser
    test_parser.c
    unity.c
    ../src/parser.c
)
target_include_directories(test_parser PRIVATE ../src)

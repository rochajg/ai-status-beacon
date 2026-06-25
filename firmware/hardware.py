"""Hardware abstraction: NeoPixel (GP16) and passive buzzer (GP15)."""

import machine
import neopixel
import time

_np = neopixel.NeoPixel(machine.Pin(16), 1)
_buzzer = machine.PWM(machine.Pin(15))
_buzzer.duty_u16(0)


def set_color(r: int, g: int, b: int) -> None:
    _np[0] = (r, g, b)
    _np.write()


def led_off() -> None:
    set_color(0, 0, 0)


def beep(freq: int = 1000, duration_ms: int = 80) -> None:
    _buzzer.freq(freq)
    _buzzer.duty_u16(32768)  # 50% duty
    time.sleep_ms(duration_ms)
    _buzzer.duty_u16(0)


def buzzer_off() -> None:
    _buzzer.duty_u16(0)

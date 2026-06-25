#!/usr/bin/env bash
# Upload firmware files to the RP2040 via mpremote.
# Usage: ./scripts/flash.sh
set -euo pipefail

FIRMWARE_DIR="$(cd "$(dirname "$0")/../firmware" && pwd)"

echo "Uploading firmware..."
# Use cu.* not tty.* — on macOS, tty.* blocks waiting for DCD
PORT=$(ls /dev/cu.usbmodem* 2>/dev/null | head -1)
if [ -z "$PORT" ]; then
  echo "Error: no RP2040 found on /dev/cu.usbmodem*" >&2
  exit 1
fi
echo "Using port: $PORT"

mpremote connect "$PORT" fs cp "$FIRMWARE_DIR/hardware.py" :hardware.py
mpremote connect "$PORT" fs cp "$FIRMWARE_DIR/states.py"   :states.py
mpremote connect "$PORT" fs cp "$FIRMWARE_DIR/main.py"     :main.py
echo "Done. Resetting device..."
mpremote connect "$PORT" reset
echo "Firmware uploaded successfully."

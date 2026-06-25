#!/usr/bin/env bash
# Upload firmware files to the RP2040 via mpremote.
# Usage: ./scripts/flash.sh
set -euo pipefail

FIRMWARE_DIR="$(cd "$(dirname "$0")/../firmware" && pwd)"

echo "Uploading firmware..."
mpremote connect auto fs cp "$FIRMWARE_DIR/hardware.py" :hardware.py
mpremote connect auto fs cp "$FIRMWARE_DIR/states.py"   :states.py
mpremote connect auto fs cp "$FIRMWARE_DIR/main.py"     :main.py
echo "Done. Resetting device..."
mpremote connect auto reset
echo "Firmware uploaded successfully."

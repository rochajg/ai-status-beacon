#!/usr/bin/env bash
# Flash the beacon.uf2 firmware to a RP2040 Zero in BOOT mode.
# Usage: ./scripts/flash.sh [path/to/beacon.uf2]
set -euo pipefail

UF2="${1:-firmware/build/beacon.uf2}"

if [ ! -f "$UF2" ]; then
  echo "Error: $UF2 not found." >&2
  echo "Build first: cd firmware/build && cmake .. && make beacon" >&2
  exit 1
fi

VOLUME="/Volumes/RPI-RP2"
if [ ! -d "$VOLUME" ]; then
  echo "Error: RPI-RP2 volume not found." >&2
  echo "Hold BOOT on the RP2040 Zero while plugging in the USB cable." >&2
  exit 1
fi

echo "Flashing $UF2 -> $VOLUME ..."
cp "$UF2" "$VOLUME/"
echo "Done. The device will reboot automatically."

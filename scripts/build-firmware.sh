#!/usr/bin/env bash
# Build the C firmware locally. Produces firmware/build/beacon.uf2
# Usage: ./scripts/build-firmware.sh
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
FIRMWARE_DIR="$REPO_ROOT/firmware"
BUILD_DIR="$FIRMWARE_DIR/build"

# ── Check ARM toolchain ────────────────────────────────────────
# Verify the toolchain has the newlib spec files (nosys.specs).
# brew install arm-none-eabi-gcc is NOT enough — it omits newlib.
# Use: brew install --cask gcc-arm-embedded  (full official toolchain)
if ! arm-none-eabi-gcc -print-file-name=nosys.specs 2>/dev/null | grep -q nosys; then
  echo "ERROR: ARM toolchain missing nosys.specs (newlib runtime)."
  echo ""
  echo "Install the full official ARM GNU Embedded Toolchain:"
  echo "  macOS:  brew install --cask gcc-arm-embedded"
  echo "  Ubuntu: sudo apt-get install gcc-arm-none-eabi libnewlib-arm-none-eabi \\"
  echo "            libstdc++-arm-none-eabi-newlib cmake ninja-build"
  echo ""
  echo "NOTE: 'brew install arm-none-eabi-gcc' is a bare compiler and will NOT work."
  exit 1
fi

# ── Init pico-sdk submodule if needed ─────────────────────────
if [ ! -f "$FIRMWARE_DIR/pico-sdk/CMakeLists.txt" ]; then
  echo "Initialising pico-sdk submodule..."
  git -C "$REPO_ROOT" submodule update --init --recursive firmware/pico-sdk
fi

# ── Configure + build ─────────────────────────────────────────
echo "Configuring..."
mkdir -p "$BUILD_DIR"
cmake -S "$FIRMWARE_DIR" -B "$BUILD_DIR" -G Ninja -DCMAKE_BUILD_TYPE=Release --fresh 2>&1 | tail -5

echo "Building..."
cmake --build "$BUILD_DIR" --target beacon -- -j"$(nproc 2>/dev/null || sysctl -n hw.logicalcpu)"

echo ""
echo "✓ Done: $BUILD_DIR/beacon.uf2"
echo ""
echo "To flash: hold BOOT on RP2040, plug in, then:"
echo "  cp $BUILD_DIR/beacon.uf2 /Volumes/RPI-RP2/"

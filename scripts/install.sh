#!/usr/bin/env bash
# Install the beacon CLI for macOS.
# Usage: curl -fsSL <url>/scripts/install.sh | bash
set -euo pipefail

REPO="rochajg/ai-status-beacon"
INSTALL_DIR="${BEACON_INSTALL_DIR:-$HOME/.local/bin}"
BINARY="$INSTALL_DIR/beacon"

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  arm64)  SUFFIX="darwin-arm64" ;;
  x86_64) SUFFIX="darwin-amd64" ;;
  *)
    echo "Unsupported architecture: $ARCH" >&2
    exit 1
    ;;
esac

# Fetch latest release tag
echo "Fetching latest release..."
TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": "\(.*\)".*/\1/')

if [ -z "$TAG" ]; then
  echo "Could not determine latest release. Check your internet connection." >&2
  exit 1
fi

echo "Installing beacon $TAG ($SUFFIX)..."

# Download binary
URL="https://github.com/$REPO/releases/download/$TAG/beacon-$SUFFIX"
mkdir -p "$INSTALL_DIR"
curl -fsSL "$URL" -o "$BINARY"
chmod +x "$BINARY"

echo ""
echo "✓ beacon $TAG installed to $BINARY"

# PATH check
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$INSTALL_DIR"; then
  echo ""
  echo "⚠  $INSTALL_DIR is not in your PATH."
  echo "   Add this to your ~/.zshrc or ~/.bash_profile:"
  echo ""
  echo "   export PATH=\"\$HOME/.local/bin:\$PATH\""
fi

echo ""
echo "Next steps:"
echo "  1. Flash MicroPython to your RP2040 Zero (see README)"
echo "  2. Install firmware: pip install mpremote && ./scripts/flash.sh"
echo "  3. Start the daemon:  beacon daemon &"
echo "  4. Add Claude Code hooks (see README)"
echo "  5. Test: beacon status"

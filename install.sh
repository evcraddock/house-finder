#!/usr/bin/env sh
#
# Install the hf CLI from GitHub releases.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/evcraddock/house-finder/main/install.sh | sh
#
# Options (via env vars):
#   INSTALL_DIR  — where to install (default: /usr/local/bin, or ~/.local/bin if no write access)
#   VERSION      — specific version to install (default: latest)
#
set -e

REPO="evcraddock/house-finder"
BINARY="hf"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux)  OS="linux" ;;
  darwin) OS="darwin" ;;
  *)
    echo "Error: unsupported OS: $OS" >&2
    echo "Supported: linux, darwin (macOS)" >&2
    exit 1
    ;;
esac

# Detect arch
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64)  ARCH="amd64" ;;
  aarch64|arm64)  ARCH="arm64" ;;
  *)
    echo "Error: unsupported architecture: $ARCH" >&2
    echo "Supported: amd64 (x86_64), arm64 (aarch64)" >&2
    exit 1
    ;;
esac

# Determine install directory
if [ -n "$INSTALL_DIR" ]; then
  DIR="$INSTALL_DIR"
elif [ -w /usr/local/bin ]; then
  DIR="/usr/local/bin"
else
  DIR="$HOME/.local/bin"
  mkdir -p "$DIR"
fi

# Determine version
if [ -n "$VERSION" ]; then
  TAG="$VERSION"
else
  echo "Finding latest release..."
  TAG=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
  if [ -z "$TAG" ]; then
    echo "Error: could not determine latest version" >&2
    exit 1
  fi
fi

ARTIFACT="${BINARY}-${OS}-${ARCH}"
URL="https://github.com/$REPO/releases/download/${TAG}/${ARTIFACT}"

echo "Installing $BINARY $TAG ($OS/$ARCH) to $DIR..."

# Download
TMP=$(mktemp)
if ! curl -fsSL -o "$TMP" "$URL"; then
  echo "Error: download failed — $URL" >&2
  echo "Check that $TAG has a binary for $OS/$ARCH" >&2
  rm -f "$TMP"
  exit 1
fi

# Install
chmod +x "$TMP"
mv "$TMP" "$DIR/$BINARY"

echo "Installed $BINARY to $DIR/$BINARY"

# Check if in PATH
if ! echo "$PATH" | tr ':' '\n' | grep -qx "$DIR"; then
  echo ""
  echo "Note: $DIR is not in your PATH. Add it:"
  echo "  export PATH=\"$DIR:\$PATH\""
fi

echo ""
"$DIR/$BINARY" version 2>/dev/null || true

#!/bin/sh
set -e

REPO="fabiobrady/tact"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
BINARY="tact"

# Detect OS and arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
  darwin|linux) ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Get latest release tag
echo "Fetching latest release..."
TAG=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4)
if [ -z "$TAG" ]; then
  echo "Could not determine latest release. Check https://github.com/${REPO}/releases"
  exit 1
fi
echo "Latest version: $TAG"

# Download
URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY}_${OS}_${ARCH}.tar.gz"
echo "Downloading ${URL}..."
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
curl -fsSL "$URL" -o "$TMP/archive.tar.gz"
tar -xzf "$TMP/archive.tar.gz" -C "$TMP"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
else
  echo "Installing to $INSTALL_DIR (requires sudo)..."
  sudo mv "$TMP/$BINARY" "$INSTALL_DIR/$BINARY"
fi
chmod +x "$INSTALL_DIR/$BINARY"

echo "Installed $BINARY $TAG to $INSTALL_DIR/$BINARY"
echo "  Run 'tact' to start the TUI"
echo "  Run 'tact todo --help' for todo management"
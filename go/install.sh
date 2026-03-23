#!/bin/sh
set -e

REPO="Fabio-RibeiroB/tact"
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

fetch_latest_tag() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | cut -d'"' -f4
}

fetch_newest_tag() {
  curl -fsSL "https://api.github.com/repos/${REPO}/tags?per_page=1" | grep '"name"' | head -n1 | cut -d'"' -f4
}

install_binary() {
  src="$1"
  if [ -w "$INSTALL_DIR" ]; then
    mv "$src" "$INSTALL_DIR/$BINARY"
  else
    echo "Installing to $INSTALL_DIR (requires sudo)..."
    sudo mv "$src" "$INSTALL_DIR/$BINARY"
  fi
  chmod +x "$INSTALL_DIR/$BINARY"
}

build_from_source() {
  TAG="$1"
  TARBALL_URL="https://github.com/${REPO}/archive/refs/tags/${TAG}.tar.gz"
  echo "No release asset available for ${TAG}; building from source instead..."
  command -v go >/dev/null 2>&1 || {
    echo "Go is required for source installs. Install Go 1.22+ and retry."
    exit 1
  }
  TMP=$(mktemp -d)
  trap 'rm -rf "$TMP"' EXIT
  curl -fsSL "$TARBALL_URL" -o "$TMP/source.tar.gz"
  tar -xzf "$TMP/source.tar.gz" -C "$TMP"
  SRC_DIR="$TMP/tact-${TAG#v}/go"
  if [ ! -d "$SRC_DIR" ]; then
    SRC_DIR=$(find "$TMP" -maxdepth 3 -type f -name go.mod | sed 's|/go.mod$||' | head -n1)
  fi
  if [ ! -d "$SRC_DIR" ]; then
    echo "Could not find Go source in downloaded archive."
    exit 1
  fi
  (cd "$SRC_DIR" && go build -o "$TMP/$BINARY" ./cmd/tact)
  install_binary "$TMP/$BINARY"
  echo "Installed $BINARY $TAG to $INSTALL_DIR/$BINARY"
  exit 0
}

echo "Fetching latest version..."
TAG=$(fetch_latest_tag)
if [ -z "$TAG" ]; then
  TAG=$(fetch_newest_tag)
fi
if [ -z "$TAG" ]; then
  echo "Could not determine latest tag. Check https://github.com/${REPO}/tags"
  exit 1
fi
echo "Latest version: $TAG"

# Download
URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY}_${OS}_${ARCH}.tar.gz"
echo "Downloading ${URL}..."
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT
if ! curl -fsSL "$URL" -o "$TMP/archive.tar.gz"; then
  build_from_source "$TAG"
fi
if ! tar -xzf "$TMP/archive.tar.gz" -C "$TMP"; then
  build_from_source "$TAG"
fi

# Install
install_binary "$TMP/$BINARY"

echo "Installed $BINARY $TAG to $INSTALL_DIR/$BINARY"
echo "  Run 'tact' to start the TUI"
echo "  Run 'tact todo --help' for todo management"

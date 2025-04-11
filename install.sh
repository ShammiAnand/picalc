#!/bin/bash
set -e

VERSION="0.1.0"
BINARY_NAME="picalc"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

# Map architecture to Go's arch naming
case $ARCH in
x86_64)
  ARCH="amd64"
  ;;
aarch64 | arm64)
  ARCH="arm64"
  ;;
armv7*)
  ARCH="arm"
  ;;
i386 | i686)
  ARCH="386"
  ;;
*)
  echo "Unsupported architecture: $ARCH"
  exit 1
  ;;
esac

echo "Detected OS: $OS, Architecture: $ARCH"

# Check if the binary release exists
RELEASE_URL="https://github.com/yourusername/picalc/releases/download/v${VERSION}/picalc_${VERSION}_${OS}_${ARCH}.tar.gz"
HTTP_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" --head "$RELEASE_URL")

if [ "$HTTP_RESPONSE" != "200" ]; then
  echo "Release not found: $RELEASE_URL"
  echo "Building from source..."

  # Check if Go is installed
  if ! command -v go &>/dev/null; then
    echo "Go is not installed. Please install Go first."
    exit 1
  fi

  # Clone the repository and build
  TMP_DIR=$(mktemp -d)
  git clone https://github.com/yourusername/picalc.git "$TMP_DIR"
  cd "$TMP_DIR"
  go build -o "$BINARY_NAME" ./cmd/picalc
  sudo mv "$BINARY_NAME" "$INSTALL_DIR/"
  cd - >/dev/null
  rm -rf "$TMP_DIR"
else
  # Download and install the binary
  echo "Downloading pre-built binary..."
  TMP_DIR=$(mktemp -d)
  curl -L "$RELEASE_URL" | tar xz -C "$TMP_DIR"
  sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/"
  rm -rf "$TMP_DIR"
fi

echo "Installation complete! The 'picalc' command is now available."
echo "Run 'picalc --help' for usage instructions."

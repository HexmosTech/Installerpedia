#!/usr/bin/env bash
set -e

APPNAME="ipm"
DIST_DIR="ipm_dist"
INSTALL_DIR="$HOME/.local/bin"          # fallback for Windows
BIN_PATH="/usr/local/bin/$APPNAME"      # global path for Unix

echo "==> Cleaning old build..."
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Detect current platform
CUR_OS="$(uname | tr '[:upper:]' '[:lower:]')"
CUR_ARCH="$(uname -m)"

# Normalize architecture
if [ "$CUR_ARCH" = "x86_64" ]; then
    CUR_ARCH="amd64"
elif [ "$CUR_ARCH" = "aarch64" ]; then
    CUR_ARCH="arm64"
fi

EXT=""
if [[ "$CUR_OS" == "mingw"* || "$CUR_OS" == "cygwin"* || "$CUR_OS" == "msys"* ]]; then
    CUR_OS="windows"
    EXT=".exe"
fi

OUT="$DIST_DIR/$APPNAME-$CUR_OS-$CUR_ARCH$EXT"

echo "==> Building Go binary for $CUR_OS-$CUR_ARCH..."
CGO_ENABLED=0 GOOS="$CUR_OS" GOARCH="$CUR_ARCH" go build -ldflags="-s -w" -o "$OUT" ./cmd

# Determine final install directory
if [[ "$CUR_OS" == "windows" ]]; then
    INSTALL_PATH="$INSTALL_DIR/$APPNAME$EXT"
else
    INSTALL_PATH="$BIN_PATH"
fi

echo "==> Preparing install directory..."
if [[ "$CUR_OS" != "windows" ]]; then
    sudo mkdir -p "$(dirname "$INSTALL_PATH")"
else
    mkdir -p "$(dirname "$INSTALL_PATH")"
fi

echo "==> Overwriting binary..."
if [[ "$CUR_OS" != "windows" ]]; then
    sudo cp -f "$OUT" "$INSTALL_PATH"
    sudo chmod +x "$INSTALL_PATH"
else
    cp -f "$OUT" "$INSTALL_PATH"
    chmod +x "$INSTALL_PATH"
fi

echo ""
echo "==> Installation complete!"
echo "Installed binary:"
echo "  $INSTALL_PATH"

if [[ "$CUR_OS" == "windows" ]]; then
    echo ""
    echo "Run from anywhere (add to PATH if needed):"
    echo "  $INSTALL_PATH"
else
    echo ""
    echo "Run from anywhere:"
    echo "  ipm \"install repo-name\""
fi

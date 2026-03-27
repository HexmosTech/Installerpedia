#!/usr/bin/env bash
set -e

APPNAME="ipm"
DIST_DIR="ipm_dist"
INSTALL_DIR="$HOME/.local/bin"

echo "==> Cleaning old build..."
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

# Define the platforms we want to build
PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
)

for platform in "${PLATFORMS[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"

    OUTFILE="$DIST_DIR/${APPNAME}-${GOOS}-${GOARCH}"
    if [[ "$GOOS" == "windows" ]]; then
        OUTFILE="$OUTFILE.exe"
    fi

    echo "==> Building $OUTFILE..."
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
        go build -ldflags="-s -w" -o "$OUTFILE" ./cmd

    # 1. macOS specific handling (IMPORTANT)
    if [[ "$GOOS" == "darwin" ]]; then
        echo "==> Signing $OUTFILE for macOS compatibility..."
        rcodesign sign "$OUTFILE"
    fi
done

echo "✅ Build complete. All binaries are in $DIST_DIR"

#!/usr/bin/env bash
set -e

APPNAME="ipm_darwin"
DIST_DIR="ipm_dist"
# Specifically targeting macOS Intel
GOOS="darwin"
GOARCH="amd64"
OUTFILE="$DIST_DIR/${APPNAME}-${GOOS}-${GOARCH}"

echo "==> Cleaning old build..."
rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "==> Building for macOS (Intel)..."
CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH \
    go build -ldflags="-s -w" -o "$OUTFILE" ./cmd

# Ad-hoc signing for macOS compatibility
if command -v rcodesign &>/dev/null; then
    echo "==> Signing $OUTFILE..."
    rcodesign sign "$OUTFILE"
else
    echo "⚠️  Warning: rcodesign not found. Binary may require manual permissions to run."
fi

echo "✅ Build complete: $OUTFILE"
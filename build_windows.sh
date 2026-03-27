#!/usr/bin/env bash
set -e

APPNAME="ipm"
DIST_DIR="ipm_dist"
# Force architecture to amd64 for Windows (common) or detect it
CUR_ARCH="$(uname -m)"
[ "$CUR_ARCH" = "x86_64" ] && CUR_ARCH="amd64"
[ "$CUR_ARCH" = "aarch64" ] && CUR_ARCH="arm64"

OUT="$DIST_DIR/$APPNAME-windows-$CUR_ARCH.exe"

echo "==> Cleaning old build..."
# Adding sudo here is a safety net if the folder was previously created by root
sudo rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

echo "==> Building Go binary for Windows ($CUR_ARCH)..."
# Added -buildvcs=false to bypass the 'exit status 128' error
CGO_ENABLED=0 GOOS=windows GOARCH="$CUR_ARCH" go build -buildvcs=false -ldflags="-s -w" -o "$OUT" ./cmd

echo ""
echo "Build complete: $OUT"
#!/usr/bin/env bash
set -e

APPNAME="ipm"
DIST_DIR="ipm_dist"
INSTALL_DIR="$HOME/.local/bin"


echo "==> Cleaning old build..."
rm -rf $DIST_DIR
mkdir -p $DIST_DIR

echo "==> Building Go binary..."
go build -o $DIST_DIR/$APPNAME ./cmd

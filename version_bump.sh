#!/usr/bin/env bash

set -e

FILE="version/version.json"

# Read current version
current=$(jq -r '.version' "$FILE")

IFS='.' read -r major minor patch <<< "$current"

case "$1" in
  patch)
    patch=$((patch + 1))
    ;;
  minor)
    minor=$((minor + 1))
    patch=0
    ;;
  major)
    major=$((major + 1))
    minor=0
    patch=0
    ;;
  *)
    echo "Usage: ./version-bump.sh [major|minor|patch]"
    echo "Current version: $current"
    exit 1
    ;;
esac

new="$major.$minor.$patch"

# Update JSON
jq --arg v "$new" '.version = $v' "$FILE" > "$FILE.tmp"
mv "$FILE.tmp" "$FILE"

echo "Version bumped: $current → $new"

#!/usr/bin/env bash
set -e

# === Config ===
REPO="hexmosTech/Installerpedia"
FILE="version/version.json"
DIST_DIR="ipm_dist"

# Visual Helpers
info() { echo -e "\033[1;34m[INFO]\033[0m $1"; }
success() { echo -e "\033[1;32m[SUCCESS]\033[0m $1"; }
working() { echo -e "\033[1;33m[...]\033[0m $1"; }

echo "-----------------------------------------------"
echo "  GitHub Release Automation: $REPO"
echo "-----------------------------------------------"

# Read version
version=$(jq -r '.version' "$FILE")
info "Target Version: $version"

# Create draft release via GitHub API
working "Creating draft release on GitHub..."
response=$(curl -s -X POST \
  -H "Authorization: token $GH_IPM_PUBLISH_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"tag_name\": \"$version\", \"name\": \"ipm $version\", \"body\": \"\", \"draft\": true}" \
  "https://api.github.com/repos/$REPO/releases")

# Extract release ID and HTML URL
release_id=$(echo "$response" | jq -r '.id')
draft_url=$(echo "$response" | jq -r '.html_url')

success "Draft release initialized."
info "Release URL: $draft_url"

# Upload binaries
echo -e "\n--- Uploading Assets ---"
for file in "$DIST_DIR"/*; do
  filename=$(basename "$file")
  working "Uploading: $filename"
  
  curl -s -X POST \
    -H "Authorization: token $GH_IPM_PUBLISH_TOKEN" \
    -H "Content-Type: $(file -b --mime-type "$file")" \
    --data-binary @"$file" \
    "https://uploads.github.com/repos/$REPO/releases/$release_id/assets?name=$filename"
done

echo "-----------------------------------------------"
success "Deployment complete. All binaries uploaded."
info "Finalize your release at: $draft_url"
echo "-----------------------------------------------"
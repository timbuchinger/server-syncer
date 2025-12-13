#!/bin/bash
set -euo pipefail

# This script updates the Homebrew formula with the release version and SHA256 checksums
# Usage: ./scripts/update-brew-formula.sh <version> <dist-dir>

VERSION=$1
DIST_DIR=$2

if [ -z "$VERSION" ] || [ -z "$DIST_DIR" ]; then
  echo "Usage: $0 <version> <dist-dir>"
  echo "Example: $0 2.5.3 dist"
  exit 1
fi

# Remove 'v' prefix if present
VERSION=${VERSION#v}

echo "Updating Homebrew formula for version $VERSION"

# Calculate SHA256 checksums for macOS and Linux binaries
SHA256_DARWIN_AMD64=$(sha256sum "$DIST_DIR/agent-align-darwin-amd64.tar.gz" | awk '{print $1}')
SHA256_DARWIN_ARM64=$(sha256sum "$DIST_DIR/agent-align-darwin-arm64.tar.gz" | awk '{print $1}')
SHA256_LINUX_AMD64=$(sha256sum "$DIST_DIR/agent-align-linux-amd64.tar.gz" | awk '{print $1}')
SHA256_LINUX_ARM64=$(sha256sum "$DIST_DIR/agent-align-linux-arm64.tar.gz" | awk '{print $1}')

echo "Checksums calculated:"
echo "  darwin-amd64: $SHA256_DARWIN_AMD64"
echo "  darwin-arm64: $SHA256_DARWIN_ARM64"
echo "  linux-amd64: $SHA256_LINUX_AMD64"
echo "  linux-arm64: $SHA256_LINUX_ARM64"

# Update the formula file
FORMULA_FILE="Formula/agent-align.rb"

if [ ! -f "$FORMULA_FILE" ]; then
  echo "Error: Formula file $FORMULA_FILE not found"
  exit 1
fi

# Create updated formula
sed -e "s/VERSION_PLACEHOLDER/$VERSION/g" \
    -e "s/SHA256_AMD64_PLACEHOLDER/$SHA256_DARWIN_AMD64/g" \
    -e "s/SHA256_ARM64_PLACEHOLDER/$SHA256_DARWIN_ARM64/g" \
    -e "s/SHA256_LINUX_AMD64_PLACEHOLDER/$SHA256_LINUX_AMD64/g" \
    -e "s/SHA256_LINUX_ARM64_PLACEHOLDER/$SHA256_LINUX_ARM64/g" \
    "$FORMULA_FILE" > "$FORMULA_FILE.tmp"

mv "$FORMULA_FILE.tmp" "$FORMULA_FILE"

echo "Formula updated successfully at $FORMULA_FILE"

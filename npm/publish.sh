#!/bin/bash
set -euo pipefail

# Usage: publish.sh <version> <release-dir>
# release-dir should contain the GoReleaser tar.gz/zip files

VERSION="$1"
RELEASE_DIR="$2"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

# Map GoReleaser archive names to npm package dirs
declare -A PLATFORM_MAP=(
  ["ink_${VERSION}_darwin_arm64.tar.gz"]="ink-cli-darwin-arm64"
  ["ink_${VERSION}_darwin_amd64.tar.gz"]="ink-cli-darwin-x64"
  ["ink_${VERSION}_linux_amd64.tar.gz"]="ink-cli-linux-x64"
  ["ink_${VERSION}_linux_arm64.tar.gz"]="ink-cli-linux-arm64"
  ["ink_${VERSION}_windows_amd64.zip"]="ink-cli-win32-x64"
)

echo "Publishing ink-cli v${VERSION}"

# Extract binaries into platform packages
for archive in "${!PLATFORM_MAP[@]}"; do
  pkg="${PLATFORM_MAP[$archive]}"
  pkg_dir="${SCRIPT_DIR}/${pkg}"
  bin_dir="${pkg_dir}/bin"
  archive_path="${RELEASE_DIR}/${archive}"

  if [ ! -f "$archive_path" ]; then
    echo "  SKIP ${pkg} (${archive} not found)"
    continue
  fi

  mkdir -p "$bin_dir"

  if [[ "$archive" == *.zip ]]; then
    unzip -o -j "$archive_path" "ink.exe" -d "$bin_dir"
  else
    tar -xzf "$archive_path" -C "$bin_dir" ink
  fi

  chmod +x "$bin_dir"/*

  # Update version in package.json
  cd "$pkg_dir"
  npm version "$VERSION" --no-git-tag-version --allow-same-version
  npm publish --access public
  echo "  PUBLISHED ${pkg}@${VERSION}"
  cd "$SCRIPT_DIR"
done

# Publish meta package
META_DIR="${SCRIPT_DIR}/ink-cli"
cd "$META_DIR"

# Update version and optional dependency versions
node -e "
const pkg = require('./package.json');
pkg.version = '${VERSION}';
for (const dep of Object.keys(pkg.optionalDependencies || {})) {
  pkg.optionalDependencies[dep] = '${VERSION}';
}
require('fs').writeFileSync('package.json', JSON.stringify(pkg, null, 2) + '\n');
"

npm publish --access public
echo "  PUBLISHED @mldotink/ink-cli@${VERSION}"

echo "Done!"

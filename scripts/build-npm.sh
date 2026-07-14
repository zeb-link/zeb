#!/usr/bin/env bash
# Assembles the per-platform npm packages under npm/packages/ from the binaries
# in dist/. Run scripts/build-release.sh first.
#
# Each package carries one native binary and declares os/cpu, so npm installs
# only the one matching the host and skips the rest.
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

version="${VERSION:-}"
if [[ -z "$version" ]]; then
  version="$(node -p "require('./npm/package.json').version")"
fi

dist="$root/dist"
packages="$root/npm/packages"

if [[ ! -d "$dist" ]]; then
  echo "dist/ not found — run 'make release-build' first" >&2
  exit 1
fi

rm -rf "$packages"
mkdir -p "$packages"

# goos goarch node_platform node_arch
targets=(
  "darwin amd64 darwin x64"
  "darwin arm64 darwin arm64"
  "linux amd64 linux x64"
  "linux arm64 linux arm64"
  "windows amd64 win32 x64"
  "windows arm64 win32 arm64"
)

for target in "${targets[@]}"; do
  read -r goos goarch node_platform node_arch <<< "$target"

  ext=""
  if [[ "$goos" == "windows" ]]; then
    ext=".exe"
  fi

  src="$dist/zeb-$goos-$goarch$ext"
  if [[ ! -f "$src" ]]; then
    echo "missing $src — run 'make release-build' first" >&2
    exit 1
  fi

  name="zeb-$node_platform-$node_arch"
  pkg="$packages/$name"
  mkdir -p "$pkg/bin"

  cp "$src" "$pkg/bin/zeb$ext"
  chmod +x "$pkg/bin/zeb$ext"
  cp "$root/LICENSE" "$pkg/LICENSE"

  cat > "$pkg/package.json" <<JSON
{
  "name": "@zeb-link/$name",
  "version": "$version",
  "description": "Native zeb binary for $node_platform $node_arch. Installed automatically by @zeb-link/zeb.",
  "repository": {
    "type": "git",
    "url": "git+https://github.com/zeb-link/zeb.git"
  },
  "homepage": "https://github.com/zeb-link/zeb#readme",
  "author": "kerns <david@kerns.dk>",
  "license": "MIT",
  "os": ["$node_platform"],
  "cpu": ["$node_arch"],
  "files": ["bin/zeb$ext", "LICENSE"]
}
JSON

  cat > "$pkg/README.md" <<MD
# @zeb-link/$name

The native \`zeb\` binary for $node_platform $node_arch.

You don't install this directly. It is an optional dependency of
[\`@zeb-link/zeb\`](https://www.npmjs.com/package/@zeb-link/zeb), which npm
resolves to the one package matching your platform.

\`\`\`bash
npm i -g @zeb-link/zeb
\`\`\`
MD

  echo "assembled npm/packages/$name"
done

echo "platform packages for v$version written to $packages"

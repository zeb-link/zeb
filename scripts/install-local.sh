#!/usr/bin/env sh
set -eu

project_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
binary_name="zeb"
build_dir="${ZEB_BUILD_DIR:-"$project_root/bin"}"
install_dir="${ZEB_INSTALL_DIR:-"$HOME/.local/bin"}"
install_mode="${ZEB_INSTALL_MODE:-link}"
binary_path="$build_dir/$binary_name"
install_path="$install_dir/$binary_name"

mkdir -p "$build_dir" "$install_dir"

echo "Building $binary_name -> $binary_path"
go build -o "$binary_path" "$project_root/cmd/zeb"

if [ -e "$install_path" ] || [ -L "$install_path" ]; then
  if [ -L "$install_path" ]; then
    rm "$install_path"
  elif [ "${ZEB_INSTALL_OVERWRITE:-0}" = "1" ]; then
    rm "$install_path"
  else
    echo "Refusing to replace existing non-symlink: $install_path" >&2
    echo "Set ZEB_INSTALL_OVERWRITE=1 to replace it, or remove it manually." >&2
    exit 1
  fi
fi

case "$install_mode" in
  link)
    ln -s "$binary_path" "$install_path"
    ;;
  copy)
    cp "$binary_path" "$install_path"
    ;;
  *)
    echo "Unknown ZEB_INSTALL_MODE: $install_mode. Use link or copy." >&2
    exit 1
    ;;
esac

case ":$PATH:" in
  *":$install_dir:"*) ;;
  *)
    echo "Warning: $install_dir is not on PATH for this shell." >&2
    echo "Add this to your shell config: export PATH=\"$install_dir:\$PATH\"" >&2
    ;;
esac

echo "Installed $binary_name -> $install_path"
echo "Try: zeb --help"

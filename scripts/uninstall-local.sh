#!/usr/bin/env sh
set -eu

project_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
binary_name="zeb"
install_dir="${ZEB_INSTALL_DIR:-"$HOME/.local/bin"}"
install_path="$install_dir/$binary_name"

if [ ! -e "$install_path" ] && [ ! -L "$install_path" ]; then
  echo "No local $binary_name install found at $install_path"
  exit 0
fi

if [ -L "$install_path" ]; then
  target="$(readlink "$install_path")"
  case "$target" in
    "$project_root"/*)
      rm "$install_path"
      echo "Removed $install_path"
      exit 0
      ;;
  esac
fi

if [ "${ZEB_UNINSTALL_FORCE:-0}" = "1" ]; then
  rm "$install_path"
  echo "Removed $install_path"
  exit 0
fi

echo "Refusing to remove $install_path because it is not a symlink to this checkout." >&2
echo "Set ZEB_UNINSTALL_FORCE=1 to remove it anyway." >&2
exit 1

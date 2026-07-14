#!/usr/bin/env bash
# Publishes the platform packages, then the main package that depends on them.
#
# Order matters: the main package's optionalDependencies pin exact versions, so
# the platform packages must exist on the registry first.
#
# Dry run (default):  scripts/publish-npm.sh
# For real:           PUBLISH=1 scripts/publish-npm.sh
#
# The account has 2FA on publish, so a real publish needs one of:
#   - an automation/granular token with "bypass 2FA" in ~/.npmrc (preferred —
#     this script makes seven publishes, and one OTP can expire partway through)
#   - OTP=123456 PUBLISH=1 scripts/publish-npm.sh
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$root"

packages="$root/npm/packages"
version="$(node -p "require('./npm/package.json').version")"
publish="${PUBLISH:-0}"

if [[ ! -d "$packages" ]]; then
  echo "npm/packages/ not found — run 'make npm-build' first" >&2
  exit 1
fi

if [[ "$publish" == "1" ]]; then
  if ! npm whoami >/dev/null 2>&1; then
    echo "not logged in to npm — run 'npm login' first" >&2
    exit 1
  fi
  echo "publishing v$version as $(npm whoami)"
else
  echo "DRY RUN for v$version — set PUBLISH=1 to publish for real"
fi

otp_args=()
if [[ -n "${OTP:-}" ]]; then
  otp_args=(--otp "$OTP")
fi

run_publish() {
  local dir="$1"
  if [[ "$publish" == "1" ]]; then
    (cd "$dir" && npm publish --access public "${otp_args[@]}")
  else
    (cd "$dir" && npm publish --access public --dry-run)
  fi
}

for pkg in "$packages"/*/; do
  echo "--- $(basename "$pkg")"
  run_publish "$pkg"
done

echo "--- @zeb-link/zeb (main)"
run_publish "$root/npm"

if [[ "$publish" == "1" ]]; then
  echo "published v$version — smoke test with: npm i -g @zeb-link/zeb && zeb version"
else
  echo "dry run complete — nothing was published"
fi

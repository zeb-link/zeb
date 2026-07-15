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
  # In CI there is no logged-in user: trusted publishing authenticates via OIDC
  # at publish time, so `npm whoami` correctly fails. Only gate on it locally.
  if [[ -n "${CI:-}" ]]; then
    echo "publishing v$version from CI via trusted publishing (OIDC)"
  elif npm whoami >/dev/null 2>&1; then
    echo "publishing v$version as $(npm whoami)"
  else
    echo "not logged in to npm — run 'npm login' first" >&2
    exit 1
  fi
else
  echo "DRY RUN for v$version — set PUBLISH=1 to publish for real"
fi

# Build the flag list in one place so the dry-run and real-publish paths take
# the same code path — an earlier split let a bug hide from every dry run.
# Keep the array non-empty: macOS bash 3.2 treats expanding an empty array under
# `set -u` as an unbound variable.
run_publish() {
  local dir="$1"
  local -a args=(--access public)
  if [[ -n "${OTP:-}" ]]; then
    args+=(--otp "$OTP")
  fi
  if [[ "$publish" != "1" ]]; then
    args+=(--dry-run)
  fi
  (cd "$dir" && npm publish "${args[@]}")
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

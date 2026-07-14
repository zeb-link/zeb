#!/usr/bin/env node
// Launcher for the zeb binary.
//
// The native binary ships in a per-platform package (@zeb-link/zeb-darwin-arm64
// and friends) listed under optionalDependencies. npm installs only the one
// matching the host, and this shim hands off to it with stdio inherited so the
// Bubble Tea TUI keeps a real TTY.

const { spawnSync } = require("node:child_process");

const PLATFORM_PACKAGES = {
  "darwin arm64": "@zeb-link/zeb-darwin-arm64",
  "darwin x64": "@zeb-link/zeb-darwin-x64",
  "linux arm64": "@zeb-link/zeb-linux-arm64",
  "linux x64": "@zeb-link/zeb-linux-x64",
  "win32 arm64": "@zeb-link/zeb-win32-arm64",
  "win32 x64": "@zeb-link/zeb-win32-x64",
};

function fail(message) {
  console.error(`zeb: ${message}`);
  process.exit(1);
}

function resolveBinary() {
  const key = `${process.platform} ${process.arch}`;
  const pkg = PLATFORM_PACKAGES[key];

  if (!pkg) {
    const supported = Object.keys(PLATFORM_PACKAGES).join(", ");
    fail(
      `unsupported platform: ${key}\n` +
        `Supported: ${supported}\n` +
        `Build from source instead: https://github.com/zeb-link/zeb`
    );
  }

  const binary = process.platform === "win32" ? "zeb.exe" : "zeb";

  try {
    return require.resolve(`${pkg}/bin/${binary}`);
  } catch {
    fail(
      `the ${pkg} package is missing.\n` +
        `It ships the native binary and installs automatically as an optional dependency.\n` +
        `If you installed with --no-optional or --omit=optional, reinstall without it:\n` +
        `  npm i -g @zeb-link/zeb`
    );
  }
}

const result = spawnSync(resolveBinary(), process.argv.slice(2), {
  stdio: "inherit",
});

if (result.error) {
  fail(result.error.message);
}

// Re-raise the child's terminating signal so callers see the real cause
// (Ctrl-C should look like Ctrl-C to the parent shell, not a plain exit 0).
if (result.signal) {
  process.kill(process.pid, result.signal);
}

process.exit(result.status ?? 1);

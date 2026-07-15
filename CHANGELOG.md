# Changelog

## 0.1.1 - 2026-07-15

First release published from CI. Carries a provenance attestation.

### Changed

- The product is **Zebra**, not "Zebra Link" — README and package description
  updated. <https://zeblink.io>

## 0.1.0 - 2026-07-14

First npm release.

### Added

- Distribution as `@zeb-link/zeb` on npm, shipping a prebuilt native binary for
  macOS, Linux, and Windows on x64 and arm64.
- `make release-build`, `make npm-build`, `make npm-publish`, and
  `make release-check` for cross-compiling and publishing.
- MIT license.

- `go install github.com/zeb-link/zeb/cmd/zeb@latest` as an alternative to npm.

### Changed

- `make build` embeds the npm package version into `zeb version` via ldflags.
  Builds without ldflags — `go install`, in particular — now resolve the version
  from module build info instead of reporting a hardcoded fallback.

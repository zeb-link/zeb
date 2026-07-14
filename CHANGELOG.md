# Changelog

## 0.1.0 - unreleased

First npm release.

### Added

- Distribution as `@zeb-link/zeb` on npm, shipping a prebuilt native binary for
  macOS, Linux, and Windows on x64 and arm64.
- `make release-build`, `make npm-build`, `make npm-publish`, and
  `make release-check` for cross-compiling and publishing.
- MIT license.

### Changed

- `make build` now embeds the npm package version into `zeb version` via
  ldflags. A plain `go build` still reports the `0.1.0-dev` fallback.

# Changelog

## 0.3.0 - 2026-07-20

### Fixed

- `zeb tui` was unusable with more than a screen of links: it drew every
  loaded link at once, so the list overflowed the terminal and the command
  input and footer controls were pushed off screen. The list is now a
  scrolling window that follows the selection (arrows, pgup/pgdn, home/end).

### Added

- TUI: the next page of links loads in the background as you scroll toward
  the end of what is loaded, the same cursor paging the command line uses.
- TUI: free-text search from the command input. A URL still creates a link;
  any other text searches all links, with the match total in the header and
  more results loading as you scroll. Esc clears the search.
- TUI: picking a collection in the footer now also scopes the list to that
  collection, in addition to being where new links go.
- TUI: enter on an empty command line (or c while a footer control is
  focused) copies the selected short URL to the clipboard, using OSC 52 plus
  pbcopy on macOS. The footer help shows which enter you are about to get.
- TUI: a third footer control sorts the list the same four ways as the web
  app (created, edited, clicked, total clicks) with an ascending toggle, and
  rows now show click counts.
- TUI: the selected row lights up with a muted background wash and brighter
  text instead of only the gutter tick.
- Bare `zeb` and `zeb help` now mention `zeb tui`.

### Changed

- Inactive links now read as switched off everywhere: tomato dot and label,
  and the short URL drops the live emerald for plain ink.
- Every human-readable response now uses the shared palette and ends with a
  blank line before the prompt. That covers errors and hints (suggested
  commands are highlighted like in `zeb examples`), confirmations from the
  collection, space, domain, config and spec commands, the login prompts,
  the context picker summary, version, and health. Output still strips color
  when piped and follows light and dark terminals.

## 0.2.0 - 2026-07-20

### Changed

- Rebuilt the terminal UI on Charm v2 (bubbletea, bubbles, lipgloss v2).
- Redesigned every screen. One warm palette lives in a single theme, and the
  product's own colors carry meaning: green for links, violet for collections,
  amber for warnings, red for errors. `zeb`, `zeb help`, `zeb examples`,
  `zeb status`, and the rest are styled and aligned instead of raw text.
- Bare `zeb` now shows a short five-command start screen. The full command list
  moved to `zeb help` and the copy-paste cookbook to `zeb examples`.

### Added

- Light and dark terminal support. Zeb detects the terminal background at
  startup and picks the matching palette. Override with `ZEB_THEME=light` or
  `ZEB_THEME=dark`; `NO_COLOR` is honored.
- `links query` finds links by condition (destination, clicks, dates,
  attribution, negation, free text) and `links lookup` resolves a short URL or
  code back to its link.
- `qr` and `qr variants` expose a link's QR image URLs and named designs.

### Fixed

- Every command's human output strips color when piped or captured, so
  `zeb … | …` gives clean text. Machine output stays on `--json` (`--agent`).

## 0.1.3 - 2026-07-15

### Security

- Built against Go 1.25.12, clearing **10 reachable standard-library
  vulnerabilities** in `crypto/tls`, `crypto/x509`, `net/http`, `net/url`, and
  `net/textproto` that affected every earlier binary — several reachable from
  `zeb`'s own API client. `go.mod` previously requested `go 1.25.0`, and
  releases build against whatever that resolves to. Also updates
  `golang.org/x/net` to v0.57.0 (GO-2026-4918). `govulncheck ./...` now reports
  no vulnerabilities.

### Added

- `govulncheck` runs in CI on every push and PR, plus weekly on a schedule —
  a CVE can appear with no code change on our side.
- Dependabot opens weekly PRs for Go modules and GitHub Actions.

## 0.1.2 - 2026-07-15

### Changed

- The CLI's own text now says **Zebra**, matching the README: `zeb --help`,
  `zeb login`, and the TUI welcome line all dropped "Zebra Link".

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

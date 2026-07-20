# Zeb

[![ci](https://github.com/zeb-link/zeb/actions/workflows/ci.yml/badge.svg)](https://github.com/zeb-link/zeb/actions/workflows/ci.yml)
[![npm](https://img.shields.io/npm/v/@zeb-link/zeb)](https://www.npmjs.com/package/@zeb-link/zeb)
[![license](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

The command-line client for [Zebra](https://zeblink.io), the short link
operating system. Create and manage short links, collections, domains, and
spaces from the terminal, or from a script.

Zeb is a Go binary with a Cobra command surface and a Bubble Tea TUI for the
interactive flows. It talks to the Zebra REST API and stores credentials in
`~/.zlink`.

## Install

```bash
npm i -g @zeb-link/zeb
zeb login
```

The npm package ships a prebuilt native binary for your platform (macOS, Linux,
and Windows on x64 or arm64). Node is used to install it, not to run it.

`zeb login` prompts for your Zebra API key and validates it against the API.
That is the only setup step.

If you have Go 1.25 and would rather not go through npm:

```bash
go install github.com/zeb-link/zeb/cmd/zeb@latest
```

To build from a clone:

```bash
make build          # builds ./bin/zeb
make install-local  # symlinks it into ~/.local/bin
```

`make install-local` links rather than copies, so a later `make build` updates
the installed command in place. Set `ZEB_INSTALL_MODE=copy` if you want a copied
binary, and `make uninstall-local` to remove it.

## Quick Start

```bash
zeb https://example.com                 # create a short link
zeb links                               # list links
zeb links query --status active --min-clicks 100   # filter links
zeb links lookup zbrah.link/abc        # short link -> its record
zeb analytics --group-by country --range 7d   # click analytics
zeb tui                                 # interactive browser
zeb --help
```

Every command takes `--json` (alias `--agent`) for machine-readable output,
which is what you want in scripts and agent workflows. See [Agents](#agents).

## Creating links

Both forms create links; the bare form is the fast path:

```bash
zeb https://example.com
zeb https://a.com https://b.com
zeb links create https://example.com --short-code launch
```

`--short-code` is an alias for the API's `path` field.

The domain and collection a new link lands in resolve in this order:

1. Explicit flag: `--domain` / `--collection`
2. Environment: `ZLINK_DOMAIN` / `ZLINK_COLLECTION`
3. Local active config: `zeb domain use ...` / `zeb collection use ...`
4. Server default, for domain only

Use `--no-collection` to ignore an active collection for a single create.

## Finding links

`zeb links query` filters by any combination of dimensions — the same
vocabulary the dashboard search and smart collections use. Dimensions
AND-combine; list flags OR within a dimension; `--not` inverts one:

```bash
zeb links query --status active --not clicked --clicked 30d   # active, stale links
zeb links query --target-host cnn.com,bbc.com --min-clicks 100
zeb links query "newsletter" --created 7d --json
zeb links query --status inactive --save-as "Inactive"        # persist as a smart collection
```

`zeb links lookup <short-url|code>` goes the other way — from a short link to
its record (a full URL, or a code with `--domain`). Both take `--json`.

## Analytics

`zeb analytics` is the click counterpart to `zeb links query`. It uses the
**same object-scope flags** to pick which links to count, then filters and
groups the clicks:

```bash
zeb analytics --group-by country --range 7d              # top countries this week
zeb analytics --status active --not clicked --clicked 30d --group-by browser
zeb analytics --is-bot --group-by botType --json         # the AI/bot traffic mix
zeb analytics --target-host cnn.com                      # one total for a destination
```

Omit `--group-by` for a single total; add it for a breakdown. `--measure`
counts `clicks` (default) or `uniqueClicks`. `--json` returns the raw aggregate.

## QR codes

Every link has a QR code. `zeb qr` gets it two ways:

```bash
zeb qr <link-id>                          # print stable public image URLs (PNG + SVG)
zeb qr <link-id> --download qr.png        # save the rendered image to a file
zeb qr <link-id> --download qr.svg        # SVG (format inferred from the extension)
zeb qr <link-id> --download big.png --size 1024
zeb qr variants <link-id>                 # list the link's named designs
```

The default (no `--download`) returns the design's **stable public URLs** —
key-free, CDN-served, and safe to embed in an `<img>` tag or hand to a third
party. Saving the design in the studio rewrites those same files in place, so
the URL always serves the latest look. `--download` instead fetches the
rendered bytes and writes them to disk (`--format png|svg`, `--size` for PNG,
`--variant <id>` to render a specific named design).

Authoring QR designs — styles, gradients, branding marks, signals — lives in
the web studio. The CLI reads, exports, and renders codes; it does not style
them.

## Commands

```bash
zeb login
zeb auth login
zeb auth logout
zeb auth whoami
zeb health
zeb domains
zeb domain
zeb domain use <hostname>
zeb domain clear
zeb collections
zeb collection
zeb collection show [id-or-name|active]
zeb collection links [id-or-name|active]
zeb collection create <name>
zeb collection update <id-or-name> --name … --description …
zeb collection delete <id-or-name>
zeb collection convert <id-or-name>
zeb collection add <link-id...> [--to <id-or-name>]
zeb collection remove <link-id...> [--from <id-or-name>]
zeb collection use <id-or-name>
zeb collection clear
zeb context
zeb links [--sort …] [--cursor …] [--all] [--status …] [--limit …]
zeb links --collection active
zeb links create <url...>
zeb links query [text] [--status …] [--target-host …] [--created …] [--min-clicks …] [--not …] [--filter '<json>'] [--save-as <name>]
zeb links lookup <short-url|code> [--domain <hostname>]
zeb links get <link-id>
zeb links update <link-id> [--target …] [--title …] [--description …] [--path …] [--active|--inactive]
zeb links delete <link-id...>
zeb analytics [text] [--status …] [--target-host …] [--not …] [--country …] [--browser …] [--is-bot] [--group-by …] [--measure …] [--range …] [--collection <id>] [--filter '<json>']
zeb <url...>
zeb qr <link-id>
zeb qr <link-id> --download <file> [--format png|svg] [--size <px>] [--variant <id>]
zeb qr variants <link-id>
zeb space current
zeb space list
zeb space use <space-id>
zeb config get
zeb config set <key> <value>
zeb config unset <key>
zeb config path
zeb spec sync
zeb spec path
zeb status
zeb status --check
zeb tui
zeb tui --preview
```

## Auth

API key resolution:

1. `--api-key`
2. `ZLINK_API_KEY`
3. `~/.zlink/credentials.json`

Space resolution:

1. `--space`
2. `ZLINK_SPACE`
3. `activeSpace` from `~/.zlink/config.json`

`zeb login` validates the key against `GET /api/v1/me`, stores it, and sets an
active space — automatically when the key has exactly one accessible space,
otherwise from a prompt.

## Agents

Zeb is built to be driven by an agent or script with no interactive steps.

```bash
export ZLINK_API_KEY=zeb_...     # no `zeb login` needed; --api-key also works
export ZLINK_SPACE=spc_...       # find yours with `zeb auth whoami --json`
zeb status --check --json        # preflight: validates the key + space, exits non-zero if either is bad
zeb links --limit 50 --json      # read a page; follow .nextCursor to paginate (or --all)
zeb https://example.com --json   # create; read .created[].link.shortUrl
zeb qr <link-id> --json          # get .export.imageUrls.png / .svg (public, embeddable)
```

The contract:

- **`--json`** (alias **`--agent`**) is available on every command.
- **Both success and failure are JSON on stdout.** A failed command prints
  `{"error":{"code":"…","message":"…","status":…}}` and exits non-zero, so a
  `zeb … --json | jq` pipeline always parses; the exit code tells you which.
  The `code` matches the API's error vocabulary (`PATH_TAKEN`, `INVALID_SORT`,
  `LINK_NOT_FOUND`, …), so you can branch on it.
- **No prompts.** Destructive commands (`links delete`, `collection delete`)
  do not ask for confirmation, so they never block an automated run.
- **Partial-batch results are per-row.** `links create`/`delete` on many URLs
  report `created`/`results` and `failed` arrays; a failed row never hides a
  successful one.

## TUI

```bash
zeb tui
```

Zeb is a command CLI first. The Bubble Tea surface covers focused interactive
flows; it is not a terminal rebuild of the dashboard.

`zeb tui` loads live links, domains, and collections. It opens a link browser
with a command input and a footer context toolbar. Type an HTTP URL and press
enter to create a real short link. Press `tab` to focus the footer controls, then
`d` cycles the create domain, `c` cycles the create collection, and `r` refreshes
links. Context changes are saved back to `~/.zlink/config.json` on exit.

The launch intro is picked at random per session. Preview them with:

```bash
zeb tui --preview --frames 6
zeb tui --preview --intro block-scan --frames 10
```

Variants are `block-boot`, `block-scan`, `block-glitch`, `block-pulse`, and
`block-wipe`. `zeb tui --gallery` opens a side-by-side comparison board that
stays up until `esc` or `q`; `--gallery-frame <n>` compares a different frame.

## OpenAPI snapshot

The client is hand-written against a local snapshot of the API spec:

```text
internal/openapi/openapi.json
```

Refresh it with `zeb spec sync`. The URL defaults to the configured API plus
`/openapi.json`; pass `--url` to sync from a different Core.

A drift test in `internal/api` asserts that every endpoint the client calls
exists in the snapshot, so `zeb spec sync` followed by `go test ./...` catches
client/API drift before it ships.

## Project layout

```text
cmd/zeb/             executable entrypoint
internal/cli/        Cobra command registration and flag handling
internal/api/        HTTP client primitives
internal/config/     ~/.zlink credentials and context
internal/tui/        Bubble Tea models and renderers
internal/ui/         shared brand copy and Lip Gloss styles
internal/openapi/    local Core OpenAPI snapshot
npm/                 npm package that ships the prebuilt binary
```

Command files stay thin: parse flags, resolve config, call an API/config/TUI
primitive, then format the result.

## Development

```bash
make build          # build ./bin/zeb
make test           # go test ./...
make fmt            # go fmt ./...
make vet            # go vet ./...
make spec-sync      # refresh the OpenAPI snapshot
make install-local  # symlink bin/zeb into ~/.local/bin
make release-check  # test, cross-compile, assemble and dry-run the npm packages
```

See `RELEASE.md` for the npm publish process, [docs/ROADMAP.md](docs/ROADMAP.md)
for the live checklist, and `AGENTS.md` for the conventions that govern work in
this repo.

## Feedback

Bug reports and ideas are welcome — open an issue or email <support@zeblink.io>.

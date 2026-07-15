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
zeb tui                                 # interactive browser
zeb --help
```

Every command takes `--json` for machine-readable output, which is what you want
in scripts and agent workflows.

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
zeb links [--sort …] [--cursor …] [--all] [--status …] [--limit …]
zeb links --collection active
zeb links create <url...>
zeb links get <link-id>
zeb links update <link-id> [--target …] [--title …] [--path …] [--active|--inactive]
zeb links delete <link-id...>
zeb <url...>
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

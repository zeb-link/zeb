# Zeb CLI

Go scaffold for Zeb, the next Zebra Link CLI.

This project is intentionally small right now: it wires the command framework,
local config/auth storage, a minimal HTTP client, an OpenAPI sync command, link
listing/creation, and a Bubble Tea TUI for link browsing and create context.

## Stack

- Go
- Cobra for command routing
- Bubble Tea for interactive loops
- Bubbles for reusable TUI components
- Lip Gloss for terminal styling

## Quick Start

```bash
go mod tidy
go run ./cmd/zeb --help
go run ./cmd/zeb login
go run ./cmd/zeb https://example.com
go run ./cmd/zeb tui
go run ./cmd/zeb tui --preview --frames 10
go run ./cmd/zeb tui --preview --intro block-scan --frames 10
go run ./cmd/zeb tui --gallery
```

`zeb login` prompts for your Zebra Link API key and validates it against the
production API — that is the only setup step.

## Local Install

Use this while Zeb is private/local and changing quickly:

```bash
make install-local
```

That builds `bin/zeb` and links it into `~/.local/bin/zeb`, which is already on
the PATH for this machine. After that you can run `zeb` from any directory:

```bash
zeb --help
zeb links
zeb https://example.com
```

During development, rebuild with:

```bash
make build
```

The installed command points at the rebuilt binary. If you want a copied binary
instead of a symlink, run:

```bash
ZEB_INSTALL_MODE=copy make install-local
```

Remove the local install with:

```bash
make uninstall-local
```

## Implemented Commands

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

## Project Layout

```text
cmd/zeb/             executable entrypoint
internal/cli/        Cobra command registration and flag handling
internal/api/        HTTP client primitives
internal/config/     ~/.zlink credentials and context
internal/tui/        Bubble Tea models and renderers
internal/ui/         shared brand copy and Lip Gloss styles
internal/openapi/    local Core OpenAPI snapshot
```

Keep command files thin: parse flags, resolve config, call an API/config/TUI
primitive, then format the result.

## Command Posture

Zeb is still a command CLI first. The Bubble Tea surface is for focused
interactive flows, not for recreating the whole dashboard in the terminal.

`zeb tui` loads live links, domains, and collections from Core. It plays a
launch intro, then opens a link browser with a command input and footer context
toolbar. Type an HTTP URL and press enter to create a real short link. Press
`tab` to focus the footer controls; then `d` cycles the create domain, `c`
cycles the create collection, and `r` refreshes links. Changed context is saved
back to `~/.zlink/config.json` when the TUI exits.

The launch intro is randomly selected for normal `zeb tui` sessions. Preview all
variants with:

```bash
go run ./cmd/zeb tui --preview --frames 6
```

Preview a specific variant with `--intro block-boot`,
`--intro block-scan`, `--intro block-glitch`, `--intro block-pulse`,
or `--intro block-wipe`.

Open a screenshot-friendly comparison board with:

```bash
go run ./cmd/zeb tui --gallery
```

The gallery stays open until `esc` or `q`. Use `--gallery-frame <n>` to compare
a different animation frame.

Create links with either form:

```bash
zeb https://example.com
zeb https://a.com https://b.com
zeb links create https://example.com --short-code launch
```

Creation context resolves like the old CLI:

1. Explicit flag: `--domain` / `--collection`
2. Environment: `ZLINK_DOMAIN` / `ZLINK_COLLECTION`
3. Local active config: `zeb domain use ...` / `zeb collection use ...`
4. Server default, for domain only

Use `--no-collection` to ignore an active collection for a specific create
operation. `--short-code` is an alias for Core's `path` field.

## Auth Model

The new CLI keeps the existing `~/.zlink` storage location so it can share local
credentials with earlier tools during the transition.

API key priority:

1. `--api-key`
2. `ZLINK_API_KEY`
3. `~/.zlink/credentials.json`

Space priority:

1. `--space`
2. `ZLINK_SPACE`
3. `activeSpace` from `~/.zlink/config.json`

The CLI talks to the built-in production API.

`zeb login` validates the key against `GET /api/v1/me`, stores the API key, and
sets an active space when the key has one accessible space or the user chooses
one from the prompt.

## OpenAPI Snapshot

The local snapshot lives at:

```text
internal/openapi/openapi.json
```

Refresh it with:

```bash
go run ./cmd/zeb spec sync
```

The URL defaults to the configured API plus `/openapi.json`; pass `--url` to
sync from a different Core. A drift test in `internal/api` asserts every
hand-written client endpoint exists in the snapshot, so `spec sync` +
`go test ./...` catches client/API drift.

## Next Build Steps

See [docs/ROADMAP.md](docs/ROADMAP.md) for the live checklist and
[docs/HANDOFF.md](docs/HANDOFF.md) for resume context.

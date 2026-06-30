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
go run ./cmd/zeb spec sync
go run ./cmd/zeb auth login --api-url http://localhost:3000 --api-key zeb_...
go run ./cmd/zeb https://example.com
go run ./cmd/zeb tui
go run ./cmd/zeb tui --preview --frames 10
go run ./cmd/zeb tui --preview --intro signal-sweep --frames 10
go run ./cmd/zeb tui --gallery
```

Core usually runs on port `3000` in this repo. The spec sync command uses
`http://localhost:3000/api/v1/openapi.json` by default.

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
zeb auth login
zeb auth logout
zeb auth whoami
zeb domains
zeb domain
zeb domain use <hostname>
zeb domain clear
zeb collections
zeb collection
zeb collection use <id-or-name>
zeb collection clear
zeb links
zeb links --collection active
zeb links create <url...>
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

API URL priority:

1. `--api-url`
2. `ZLINK_API_URL`
3. `apiUrl` from `~/.zlink/config.json`
4. `http://localhost:3000/api/v1`

`auth login` validates the key against `GET /api/v1/me`, stores the API key, and
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

or:

```bash
scripts/fetch-openapi.sh http://localhost:3000/api/v1/openapi.json
```

The placeholder snapshot is valid JSON so future codegen can be wired before a
local Core server is running.

## Next Build Steps

See [docs/ROADMAP.md](docs/ROADMAP.md) for the live checklist and
[docs/HANDOFF.md](docs/HANDOFF.md) for resume context. The short list:

1. Improve `zeb collections` output so it matches the polish of `zeb links`.
2. Make link-list pagination usable; the CLI currently prints a raw
   `Next cursor` without a matching human workflow.
3. Add compact/table output and snapshot tests for human command output.
4. Add OpenAPI client generation while keeping a small CLI wrapper for
   auth/config, pagination behavior, and terminal output.
5. Add more TUI affordances only where they support fast link work.

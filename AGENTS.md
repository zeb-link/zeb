# AGENTS.md

This is the Go CLI for Zebra Link, invoked as `zeb`. It is a separate project
from `zlink-core`.

## Commands

```bash
go run ./cmd/zeb --help
go run ./cmd/zeb auth login --api-url http://localhost:3000 --api-key zeb_...
go run ./cmd/zeb https://example.com
go run ./cmd/zeb links create https://example.com --short-code example
go run ./cmd/zeb spec sync
go test ./...
go build ./cmd/zeb
```

## Project Memory

- `docs/ROADMAP.md` is the live checklist. Update it whenever command behavior
  changes or when a next-step item is completed, added, or made stale.
- `docs/HANDOFF.md` is the resume checkpoint for future agents. Keep it focused
  on current shape, command model, and decisions that should survive context
  resets.
- `README.md` is user-facing orientation and should point to those docs instead
  of duplicating a competing roadmap.

## Shape

- Cobra owns normal command routing in `internal/cli`.
- Bubble Tea, Bubbles, and Lip Gloss power focused interactive flows under
  `zeb tui`, `zeb context`, and future login/space pickers.
- `zeb <url...>` and `zeb links create <url...>` are the primary create-link
  command surfaces. The TUI may support fast link browsing/creation and
  context selection, but it is not a dashboard replacement.
- `internal/api` is the one place that should set auth headers and perform HTTP
  requests.
- `internal/config` owns `~/.zlink/credentials.json` and `~/.zlink/config.json`.
- `internal/ui` owns shared terminal brand copy and Lip Gloss styles.
- `internal/tui` owns Bubble Tea models and renderers.
- `internal/openapi/openapi.json` is a local snapshot of the Core OpenAPI spec.

## Current Auth Contract

API key resolution:

1. `--api-key`
2. `ZLINK_API_KEY`
3. `~/.zlink/credentials.json`

API URL resolution:

1. `--api-url`
2. `ZLINK_API_URL`
3. `~/.zlink/config.json`
4. `http://localhost:3000/api/v1`

Space resolution:

1. `--space`
2. `ZLINK_SPACE`
3. `activeSpace` in `~/.zlink/config.json`

The Core REST paths use `/api/v1/spaces/{spaceId}/...`; the `/me` response
returns `accessibleSpaces`, which `internal/api` exposes as `AccessibleSpaces`.

## Boundaries

Do not edit other Zebra Link projects unless the user explicitly asks for
cross-project work. Use this project as the source of truth for Zeb behavior.

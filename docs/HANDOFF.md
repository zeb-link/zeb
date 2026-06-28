# CLI Handoff Notes

## What Exists

The scaffold is a compilable Go CLI with:

- Cobra root command and global flags.
- Auth commands that validate `GET /api/v1/me` and store local credentials.
- Space commands that read and set active context.
- Domain commands that list available domains and set active link-create domain.
- Collection commands that list collections and set active link-create collection.
- Links command that lists all links or a requested collection.
- Link creation via both `zeb <url...>` and `zeb links create <url...>`.
- Config commands for inspecting `~/.zlink`.
- Status command for inspecting resolved local context.
- Spec sync command for downloading Core's OpenAPI JSON.
- Fake-data playground for links-list and footer context interaction design.
- Minimal Bubble Tea TUI to prove the interactive stack is installed.

## What Is Deliberately Missing

- No update/delete links commands yet.
- No collection create/delete/add/remove commands yet.
- No analytics commands yet.
- No bulk-create endpoint integration yet; multiple URLs are currently created
  sequentially through the single-link API.
- No generated OpenAPI client yet.
- No packaging/release pipeline yet.

## Command Model

Zeb is a normal command CLI first. The TUI should stay focused on narrow
interactive tasks, such as selecting active context or previewing list layouts.
Do not turn it into a full dashboard replica.

Context is shared through `~/.zlink/config.json`: `zeb domain use`,
`zeb collection use`, `zeb collection none`, and the live `zeb context` picker
all write the same `activeDomain` / `activeCollection` values. Create commands
read those defaults automatically. Per-command flags still win:
`--domain`, `--collection`, and `--no-collection` override the saved context for
that one create run.

Bubble Tea owns terminal input while a TUI is running, so normal shell typing is
not available inside that process unless the model includes a command input. The
playground has a fake-data command bar for this reason. It can accept examples
like `https://example.com`, `links`, `domain use zlnk.to`, and `collection
clear`, then reports the real `zeb ...` command shape without calling the API.
Use `tab` or `shift+tab` to move focus between the command input and the context
controls. The `c` and `d` context shortcuts should only be active while context
controls are focused, so they do not steal typed command text.

`zeb context` is the real API-backed context picker. `zeb tui` is currently only
the intro shell/proof of stack. `zeb playground` remains fake-data interaction
design and should not be confused with live product state.

`zeb tui` randomly picks a launch intro from `internal/tui/intro`. Preview all
variants with `zeb tui --preview --frames 6`; preview one with
`--intro signal-sweep` or another slug from `zeb tui --help`.

Link creation has two equivalent entry points:

```text
zeb <url...>
zeb links create <url...>
```

Both use the same precedence:

1. `--domain` / `--collection`
2. `ZLINK_DOMAIN` / `ZLINK_COLLECTION`
3. `activeDomain` / `activeCollection` in `~/.zlink/config.json`
4. Server default for domain

`--no-collection` bypasses the active collection. `--short-code` maps to Core's
`path` field and is kept as an old-CLI muscle-memory alias.

By default, create commands ask Core to probe the destination with
`?verify=true`. Core returns `targetReachable: true | false | null`, and Zeb
prints a status dot under the created link (`● verified` or `● unreachable`).
`--no-verify` skips that network probe. This is
advisory only: Core creates the link either way. A future "only create reachable
links" flag needs a Core contract that gates creation, not just this advisory
post-create probe.

## Core API Notes

The current Core route vocabulary uses `space` in URLs:

```text
/api/v1/spaces/{spaceId}/links
```

The `/me` response returns `accessibleSpaces`. Treat the returned IDs as space
IDs in the CLI. Current IDs are prefixed `spc_`.

## Recommended Next Decision

Choose whether to generate a strict client from OpenAPI or keep a small
hand-written client for v1. Generated is probably better once the spec includes
the whole intended CLI surface, but hand-written is fine while the surface is
still changing quickly.

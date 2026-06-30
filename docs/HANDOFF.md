# CLI Handoff Notes

Use this as the resume checkpoint for future agents. The live to-do list is
`docs/ROADMAP.md`; this file explains the current shape, command model, and
decisions that are easy to lose between sessions.

Last checked against code: 2026-06-29.

## What Exists

The scaffold is a compilable Go CLI with:

- Cobra root command and global flags.
- Auth commands that validate `GET /api/v1/me` and store local credentials.
- Space commands that read and set active context.
- Domain commands that list available domains and set active link-create domain.
- Collection commands that list collections, create collections, and set active
  link-create collection.
- Links command that lists all links or a requested collection.
- Link creation via both `zeb <url...>` and `zeb links create <url...>`.
- Config commands for inspecting `~/.zlink`.
- Status command for inspecting resolved local context.
- Spec sync command for downloading Core's OpenAPI JSON.
- Live context picker for active domain and collection.
- API-backed `zeb tui` with the launch intro, live link list, URL command input,
  and footer controls for active create domain / collection.

## What Is Deliberately Missing

- No update/delete links commands yet.
- No collection update/delete/add/remove commands yet.
- No analytics commands yet.
- No bulk-create endpoint integration yet; multiple URLs are currently created
  sequentially through the single-link API.
- No human-facing pagination workflow yet. The API returns `nextCursor` and the
  client model has a `Cursor` field, but `zeb links` currently only prints the
  raw next cursor.
- No generated OpenAPI client yet.
- No packaging/release pipeline yet.

## Command Model

Zeb is a normal command CLI first. The TUI should stay focused on fast link work:
review recent links, create a link from a pasted URL, and adjust create context.
Do not turn it into a full dashboard replica.

Context is shared through `~/.zlink/config.json`: `zeb domain use`,
`zeb collection use`, `zeb collection none`, and the live `zeb context` picker
all write the same `activeDomain` / `activeCollection` values. Create commands
read those defaults automatically. Per-command flags still win:
`--domain`, `--collection`, and `--no-collection` override the saved context for
that one create run.

Bubble Tea owns terminal input while a TUI is running, so normal shell typing is
not available inside that process unless the model includes a command input.
`zeb tui` has a real command input: type an HTTP URL and press enter to create a
link through Core. Use `tab` or `shift+tab` to move focus between the command
input and the footer context controls. The `c` and `d` context shortcuts should
only be active while context controls are focused, so they do not steal typed
command text.

`zeb context` remains the dedicated API-backed context picker. `zeb tui` is now
the API-backed link browser/create surface; there is no separate experimental
TUI command.

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

Plan the OpenAPI generated-client migration if the Core REST contract is now
stable enough. "Hand-written client" means `internal/api/client.go` currently
defines request/response structs and endpoint methods manually. A generated
client would derive those types and endpoint calls from
`internal/openapi/openapi.json`, then the CLI would keep a small wrapper layer
for auth/config resolution, pagination ergonomics, and terminal output.

The immediate user-facing work is output and navigation polish:

- Make `zeb collections` look as intentional as `zeb links`.
- Turn `Next cursor: ...` into a usable pagination flow, such as `--cursor`,
  `--next`, or a clearer command hint.
- Replace the manual endpoint structs/calls with generated OpenAPI types once
  the current snapshot covers the CLI surface.

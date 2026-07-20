# CLI Handoff Notes

Use this as the resume checkpoint for future agents. The live to-do list is
`docs/ROADMAP.md`; this file explains the current shape, command model, and
decisions that are easy to lose between sessions.

Last checked against code: 2026-07-07.

## What Exists

A Go CLI covering nearly the full Core REST v1 surface:

- Auth commands that validate `GET /api/v1/me` and store local credentials.
- Space commands that read and set active context.
- Domain commands that list available domains and set the active create domain.
- Collection commands: list, show, create, update, delete, convert-to-manual,
  membership add/remove, plus active-collection context (use/clear/none).
- Link commands: list (sort/status/cursor/--all), query (full `LinkFilter` over
  `/links/query`, with `--filter` JSON and `--save-as` to persist as a smart
  collection), resolve (`/links/lookup`, short URL/code → record), create
  (single with reachability probe; multi via the bulk endpoint), get, update,
  delete (bulk, chunked at 250).
- `zeb analytics`: click analytics via `/analytics/query` — the counterpart to
  `links query`. Shares the object-scope flags (`addObjectScopeFlags`, used by
  both) and adds click dims + `--group-by`/`--measure`/`--range`. Gated on the
  ANALYTICS_VIEW feature server-side.
- `zeb <url...>` root shorthand for create.
- Config commands for inspecting `~/.zlink`; `zeb status` for context and
  `zeb status --check` for API-validated context; `zeb health` for a ping.
- Spec sync command plus drift tests binding the hand-written client to the
  vendored snapshot.
- Live context picker (`zeb context`) and the API-backed `zeb tui`.

## Command Model

Zeb is a normal command CLI first. The TUI should stay focused on fast link
work: review recent links, create a link from a pasted URL, and adjust create
context. Do not turn it into a full dashboard replica.

Context is shared through `~/.zlink/config.json`: `zeb domain use`,
`zeb collection use`, `zeb collection none`, and the live `zeb context` picker
all write the same `activeDomain` / `activeCollection` values. Create commands
read those defaults automatically. Per-command flags still win:
`--domain`, `--collection`, and `--no-collection` override the saved context
for that one create run.

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

`--no-collection` bypasses the active collection. `--short-code` maps to
Core's `path` field and is kept as an old-CLI muscle-memory alias.

## Decisions Worth Keeping

- **Production by default; login defines the environment.** The built-in API
  URL lives in `internal/config/config.go` (`defaultAPIURL`) — when the
  dedicated API domain lands, change that line and rebuild;
  `TestDefaultAPIURLIsProduction` is the deliberate tripwire. `zeb login`
  resolves the URL from override > built-in default (NEVER the stored
  config) and only persists it when overridden, so plain logins always track
  the binary's default. The development override is intentionally
  undocumented and its flag hidden from every `--help`
  (`TestAPIURLFlagHiddenButFunctional`); read the comment on
  `defaultAPIURL` for how the owner points at a local Core.

- **Stale ambient context degrades, explicit context fails.** If the saved or
  env collection no longer resolves (e.g. after a database wipe), creates warn
  on stderr and proceed without a collection; an explicit `--collection` that
  does not resolve is an error. `zeb status --check` surfaces dangling context
  with fix hints, and `zeb collection delete` clears the active-collection
  context when it deletes the active one.
- **Per-row failures are output, not command failure.** Bulk create/delete
  report per-row results (mirroring the API's 200/207 semantics) and exit
  non-zero only when NOTHING succeeded — the core marks whole-request
  invariant failures (not a member, read-only) on every row, so the first
  row's error is representative then.
- **Single create verifies, bulk create does not.** `?verify=true` is a
  single-create feature; multi-URL creates trade the reachability dot for one
  round-trip per 250 URLs and visible partial failures. `--no-verify` skips
  the probe for single creates.
- **Errors print once, without the usage block.** The root command sets
  SilenceUsage/SilenceErrors; `Execute` prints `zeb: <error>`. Put recovery
  hints in error messages, not in usage dumps.
- **URL-shorthand flag detection derives from the cobra flag set**
  (`rootFlagFor` in root.go) — never reintroduce a hardcoded flag list.
- **Spec drift is test-enforced.** `internal/api/spec_drift_test.go` pins
  every client endpoint to the snapshot and fails on NEW spec operations that
  are neither wired nor recorded in `knownUnimplemented`.
  `internal/cli/sort_values_test.go` pins the `--sort` help text to the spec.
  After `zeb spec sync`, run `go test ./...`.
- **oapi-codegen deferred**: the Core spec is OpenAPI 3.1, which
  oapi-codegen does not support well; the drift tests carry the contract
  instead.
- **Command context flows from cobra** (`cmd.Context()`, wired to SIGINT/
  SIGTERM in `Execute`) — do not reintroduce `context.Background()` in
  command bodies.

## TUI Notes

Bubble Tea owns terminal input while a TUI is running. `zeb tui` has a real
command input: type an HTTP URL and press enter to create a link through Core.
Use `tab`/`shift+tab` to move focus between the command input and the footer
context controls; `c` and `d` context shortcuts are only active while footer
controls are focused.

`zeb tui` randomly picks a launch intro from `internal/tui/intro` (five
`block-*` variants; the unused variant experiments were deleted 2026-07-07).
Preview with `zeb tui --preview --frames 6` or `--intro block-scan`; compare
variants with `zeb tui --gallery`.

## Core API Notes

The current Core route vocabulary uses `space` in URLs:

```text
/api/v1/spaces/{spaceId}/links
```

The `/me` response returns `accessibleSpaces`. Treat the returned IDs as space
IDs in the CLI. Current IDs are prefixed `spc_`.

List endpoints accept `limit` 1-1000 (server clamps; default 50) and return
`nextCursor` for keyset pagination. The cursor is only valid under the same
`sort` it was issued with.

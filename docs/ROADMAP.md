# Zeb CLI Roadmap

This is the live checklist for Zeb's command surface. Keep it current when
command behavior changes; use `docs/HANDOFF.md` for resume context and
architectural notes, and `RELEASE.md` for how releases ship.

Last checked against code: 2026-07-19. Zeb is no longer local-first — it is
published on npm as `@zeb-link/zeb` and releases from CI on a tag.

## Current Shape

- `zeb <url...>` and `zeb links create <url...>` create links through the same
  command path. One URL uses the single-create endpoint with the reachability
  probe; two or more URLs go through `POST /links/bulk` with per-row results
  (partial failures are reported, never hidden; batches chunk at 250).
- Link creation supports active domain and collection context, per-command
  overrides, custom path / old `--short-code` alias, namespace, title, and
  optional target reachability verification (single URL only).
- A stale AMBIENT collection (saved context or env pointing at a deleted
  collection) downgrades to a warning and the create proceeds without a
  collection; an explicit `--collection` that does not resolve still fails.
- `zeb links` lists links with `--sort` (full API vocabulary, incl. click
  sorts), `--status`, `--limit`, `--cursor`, and `--all` (pagination loop).
  The human output prints a copyable next-page command instead of a raw
  cursor. Same flags on `zeb collection links`.
- `zeb links query` finds links by the full filter vocabulary — the same
  `LinkFilter` the dashboard search and smart collections use — via
  `POST /links/query`: a flag per dimension (`--target-host`, `--short-domain`,
  `--created`/`--edited`/`--clicked`, `--schedule`, `--created-via`,
  `--attribution`, `--status`, `--min-clicks`/`--max-clicks`,
  `--min-unique`/`--max-unique`, `--in-collection`/`--uncollected`), `--not`
  for negation, free-text `[text]`, or a raw `--filter '<json>'`. `--save-as`
  persists the filter as a smart collection. Returns `{links, total}`.
- `zeb links lookup <short-url|code>` maps a short link back to its record via
  `GET /links/lookup` (a full URL, or a code with `--domain`).
- `zeb analytics` queries click analytics via `POST /analytics/query` — the
  counterpart to `zeb links query`. It SHARES the object-scope flags (which
  links) via `addObjectScopeFlags` and adds click dimensions (`--country`,
  `--browser`, `--device-type`, bot dims, …) plus `--group-by`/`--measure`/
  `--range`. `--json` returns the raw aggregate; omit `--group-by` for a single
  total. Gated on the ANALYTICS_VIEW plan feature server-side.
- `zeb links get/update/delete` manage single links; delete accepts many ids
  and runs through the bulk endpoint (chunked at 250, per-row results).
- `zeb qr <link-id>` returns a link's QR code: its stable public image URLs
  (via `/qr/export`) by default, `--download` to save the rendered PNG/SVG
  (via `/qr/image`, with `--format`/`--size`/`--variant`), and `zeb qr
  variants` to list a link's named designs. Authoring designs (variant styles,
  signals) stays in the studio — the variant write/detail ops are recorded in
  the drift test's `knownUnimplemented`.
- `zeb collections` / `zeb collection …` cover list, show, create, update,
  delete (clears the active-collection context if it pointed at the deleted
  one), convert (smart → manual), and membership (`add`/`remove` link ids,
  defaulting to the active collection).
- `zeb domains` / `zeb domain use` manage the default domain for new links.
- `zeb context` is a live API-backed picker for active domain and collection.
- `zeb auth login` validates an API key against `/api/v1/me`, stores it in
  `~/.zlink/credentials.json`, and can set the active space.
- `zeb space`, `zeb config`, and `zeb status` cover local context inspection;
  `zeb status --check` validates the key, space, collection, and domain
  against the API and exits non-zero on dangling context.
- `zeb health` pings the public health endpoint.
- `zeb spec sync` refreshes the vendored Core OpenAPI snapshot from the
  configured API; drift tests in `internal/api` and `internal/cli` fail when
  the hand-written client or the `--sort` help text diverges from the
  snapshot, and when the spec grows operations the CLI has not considered.
- `zeb tui` is an API-backed link browser/create surface with the launch intro,
  a scrolling link list (the window follows the selection and the next page
  loads by cursor as you near the end of what is loaded), free-text search
  through the command input (a URL creates, anything else searches via
  POST /links/query with offset paging), and footer controls for the active
  domain and collection. The selected collection both receives new links and
  scopes the browsed list. A sort control cycles the four API orderings with
  an ascending/descending toggle, and enter on an empty command line (or c
  from the footer controls) copies the selected short URL to the clipboard
  (OSC 52 + pbcopy).

## Local Distribution

Use the local install target while Zeb is not published:

```bash
make install-local
```

That builds `bin/zeb` and links it into `~/.local/bin/zeb`. This machine's
shell `PATH` already includes `~/.local/bin`, so new shells can run `zeb` from
any directory.

Useful variants:

```bash
make build
make install-local INSTALL_DIR="$HOME/bin"
ZEB_INSTALL_MODE=copy make install-local
make uninstall-local
```

The default symlink mode is best during development: rebuild with `make build`,
and the global `zeb` command points at the updated binary.

## Next Product Work

- Add compact/table output for `zeb links` once the default rich list settles.
- Translate more API error codes into user-facing advice where the raw
  message is not actionable.
- TUI pagination, free-text search, and collection filtering shipped; a next
  step is domain-scoped browsing in the TUI (the query endpoint already takes
  shortDomain).
- Broaden analytics output (compact/table rendering; a `--csv` for breakdowns).

## Deliberately Not Wired

Recorded in `internal/api/spec_drift_test.go` (`knownUnimplemented`) so new
spec operations still trip the drift test:

- `PATCH /links/bulk` (bulk update) — no CLI verb needs it yet.
- `POST /collections/bulk` — niche for a CLI; `zeb collection create` covers
  the flow.

## Engineering Cleanup

- OpenAPI client generation (`oapi-codegen`) was evaluated and deferred: the
  Core spec is OpenAPI 3.1, which oapi-codegen does not support well. The
  hand-written client is instead pinned to the snapshot by drift tests.
  Revisit if the client grows past this surface or 3.1 support matures.
- Add snapshot tests for human command output if the formatting churn slows
  down.

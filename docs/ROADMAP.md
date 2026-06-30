# Zeb CLI Roadmap

This is the live checklist for Zeb while it is still local-first and changing
quickly. Keep this file current when command behavior changes; use
`docs/HANDOFF.md` for resume context and architectural notes.

Last checked against code: 2026-06-29.

## Current Shape

- `zeb <url...>` and `zeb links create <url...>` create links through the same
  command path.
- Link creation supports active domain and collection context, per-command
  overrides, custom path / old `--short-code` alias, namespace, title, and
  optional target reachability verification.
- `zeb links` lists links in the active space, optionally scoped to a
  collection, and prints a formatted human view.
- `zeb domains` / `zeb domain use` manage the default domain for new links.
- `zeb collections`, `zeb collection use`, `zeb collection none`, and
  `zeb collection create` manage collection defaults and manual collection
  creation.
- `zeb context` is a live API-backed picker for active domain and collection.
- `zeb auth login` validates an API key against `/api/v1/me`, stores it in
  `~/.zlink/credentials.json`, and can set the active space.
- `zeb space`, `zeb config`, and `zeb status` cover local context inspection.
- `zeb spec sync` refreshes the vendored Core OpenAPI snapshot.
- `zeb tui` is an API-backed link browser/create surface with the launch intro,
  live links, URL command input, and footer controls for active create domain /
  collection.

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

- Improve collection list formatting. The current output is a plain
  `id name count` line, which feels much rougher than the formatted links list.
  It should show active state, type/smart-vs-creatable status, link count, and
  useful metadata with the same terminal polish as `zeb links`.
- Fix link-list pagination ergonomics. The API returns `nextCursor`, and the
  client model supports a cursor, but the human `zeb links` command does not
  expose a `--cursor` flag or a friendly "next page" workflow yet. Do not leave
  users with only a raw cursor and no obvious command to run.
- Add compact/table output for `zeb links` once the default rich list settles.
- Improve bulk-create output. Multiple URLs are created sequentially through
  the single-link API today; the output should include a more deliberate summary
  and clearer partial-failure behavior before this becomes a daily workflow.
- Expose more create options once the Core REST contract settles.
- Translate common API errors into user-facing advice instead of surfacing raw
  API error strings everywhere.
- Add TUI pagination and collection filtering once the command-line pagination
  semantics are settled.

## Later Product Work

- Add link update/delete commands.
- Add collection update/delete commands and link add/remove commands if the CLI
  should manage collection membership directly.
- Add analytics commands.
- Integrate a bulk-create endpoint if Core exposes one.
- Add packaging/release automation.

## Engineering Cleanup

- Add OpenAPI client generation, likely with `oapi-codegen`, now that the Core
  REST surface is closer to stable. Keep a small CLI wrapper around generated
  calls for auth/config resolution, pagination behavior, and output formatting.
- Add tests for config precedence: flags, env, stored config, defaults.
- Add tests for API URL normalization and request path construction.
- Add snapshot tests for human command output, especially link list/create and
  collection list formatting.
- Move shared terminal output primitives into a small presenter package if
  command files keep growing.

# Zeb CLI Roadmap

This document tracks the useful next steps while Zeb is still local-first and
changing quickly.

## Current Shape

- `zeb <url...>` and `zeb links create <url...>` create links.
- `zeb links` lists links in the active space, optionally scoped to a collection.
- `zeb domains` / `zeb domain use` manage the default domain for new links.
- `zeb collections` / `zeb collection use` manage the optional collection for
  new links.
- `zeb auth login` stores an API key in `~/.zlink/credentials.json`.
- `zeb spec sync` refreshes the vendored Core OpenAPI snapshot.
- `zeb tui` and `zeb playground` are exploratory interaction surfaces, not the
  primary command interface.

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

## Missing Product Features

- Bulk create output needs a more deliberate summary when several links are
  created at once.
- `links` needs pagination follow-up ergonomics, not just printing `Next cursor`.
- `links` should support a compact/table mode and maybe a richer default mode.
- Create should expose the important Core options once the REST contract settles.
- Error messages should translate common API errors into user-facing advice.
- TUI context selectors should graduate from fake data to real domains and
  collections.
- The command bar in `playground` should either execute real commands or stay
  clearly labeled as a design playground.

## Engineering Cleanup

- Decide whether to generate the API client from `internal/openapi/openapi.json`
  or keep the hand-written client until the API surface stabilizes.
- Add tests for config precedence: flags, env, stored config, defaults.
- Add tests for API URL normalization and request path construction.
- Add snapshot tests for human command output, especially link list/create.
- Move shared terminal output primitives into a small presenter package if
  command files keep growing.

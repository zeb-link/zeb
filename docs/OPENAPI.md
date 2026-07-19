# OpenAPI Workflow

The API serves its spec at `<api base>/openapi.json`. The sync command
downloads it from the CLI's resolved API base when no URL is supplied:

```bash
go run ./cmd/zeb spec sync
```

Use an explicit source when needed:

```bash
go run ./cmd/zeb spec sync --url <spec url>
```

The snapshot is written to `internal/openapi/openapi.json` (resolved against
the repo root, so the command works from any directory inside the repo).

## Drift Guard

The hand-written client is pinned to the snapshot by tests:

- `internal/api/spec_drift_test.go` asserts every client endpoint exists in
  the snapshot and flags NEW spec operations that are neither wired nor
  recorded in `knownUnimplemented`.
- `internal/cli/sort_values_test.go` pins the `--sort` help text to the
  spec's sort enum.

After a sync, run `go test ./...`. Client generation via `oapi-codegen` was
evaluated and deferred — the spec is OpenAPI 3.1, which it does not support.

## Automatic sync

Nobody should sync the snapshot by hand. Production is the single source of
truth — Core regenerates the spec from code on every deploy and serves it at
`/api/v1/openapi.json`. Consumers stay current two ways:

- **The website docs fetch the live spec** (`cache: no-store`), so they reflect
  production the instant Core deploys. No vendored copy, nothing to maintain.
- **This CLI vendors a snapshot** — it has to, because the hand-written Go
  client is drift-tested offline in CI. A vendored copy is the only thing that
  can go stale, so `.github/workflows/spec-sync.yml` keeps it fresh: it runs
  `zeb spec sync` and, when the snapshot differs from production, opens (or
  updates) a `bot/spec-sync` PR. Routine changes are a one-click merge; a PR
  whose body reports a drift-test failure means production grew an endpoint the
  client hasn't considered — wire it or record `knownUnimplemented`, then push
  to the branch.

The sync workflow runs **daily** (and on demand via Actions → spec-sync → Run
workflow), so the vendored copy is never more than about a day behind
production. On days the spec did not change, the job does nothing. This is
deliberately kept simple — no cross-repo tokens or deploy webhooks to maintain;
a day's lag on an offline test artifact is harmless, and merging the bot PR is
the only manual step.

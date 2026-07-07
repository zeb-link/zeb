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

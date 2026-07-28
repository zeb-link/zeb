# OpenAPI Workflow

Production is the single source of truth. Core regenerates the spec from code
on every deploy and serves it at `/api/v1/openapi.json`; the website docs
fetch it live. This repo vendors nothing and syncs nothing.

## Drift Guard

The hand-written client is pinned to the live spec by tests:

- `internal/api/spec_drift_test.go` asserts every client endpoint exists in
  the spec and flags NEW spec operations that are neither wired nor recorded
  in `knownUnimplemented`.
- `internal/cli/sort_values_test.go` pins the `--sort` help text to the
  spec's sort enum.

Both fetch the spec through `internal/openapi` (one download per test
process, cached). `go test ./...` is the whole workflow — locally, in CI on
every push, and in the release gate before publishing.

Behavior when the spec can't be fetched:

- **Network failure** (offline, DNS, timeout): the tests skip with a notice.
  A plane ride doesn't break `go test ./...`; drift simply isn't checked
  that run.
- **HTTP error** (the server answered, but wrongly): the tests fail. A 404
  from production means the API surface moved — that's drift, not a
  connectivity problem.

`ZEB_SPEC_URL` points the tests at a different Core, for example a local dev
server:

```bash
ZEB_SPEC_URL=http://localhost:3000/api/v1/openapi.json go test ./...
```

When a drift test fails, production grew or changed an endpoint the client
hasn't considered — wire it (add a client method and a `clientEndpoints`
row) or record a `knownUnimplemented` entry with the reason.

Client generation via `oapi-codegen` was evaluated and deferred — the spec is
OpenAPI 3.1, which it does not support.

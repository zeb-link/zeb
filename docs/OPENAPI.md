# OpenAPI Workflow

Core serves the spec at:

```text
http://localhost:3000/api/v1/openapi.json
```

The CLI sync command uses that URL when no URL is supplied:

```bash
go run ./cmd/zeb spec sync
```

Use an explicit source when needed:

```bash
go run ./cmd/zeb spec sync --url http://localhost:3000/api/v1/openapi.json
```

The snapshot is written to `internal/openapi/openapi.json`.

## Future Codegen

Likely path:

1. Add `oapi-codegen` as a tool dependency.
2. Generate types and a client from `internal/openapi/openapi.json`.
3. Keep generated files under `internal/generated`.
4. Wrap generated calls in small command-facing functions where config,
   pagination, and output behavior need CLI-specific handling.

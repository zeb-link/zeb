# Auth And Context

The CLI uses API keys. It does not manage browser sessions.

## Login

`zeb auth login` asks for or accepts an API key, calls `GET /api/v1/me`, and
writes:

```text
~/.zlink/credentials.json
~/.zlink/config.json
```

Credentials:

```json
{
  "apiKey": "zeb_...",
  "storedAt": "2026-06-26T12:00:00Z"
}
```

Config:

```json
{
  "apiUrl": "http://localhost:3000/api/v1",
  "activeSpace": "spc_..."
}
```

## Resolution Order

API key:

1. `--api-key`
2. `ZLINK_API_KEY`
3. stored credentials

API URL:

1. `--api-url`
2. `ZLINK_API_URL`
3. stored config
4. `http://localhost:3000/api/v1`

Space:

1. `--space`
2. `ZLINK_SPACE`
3. `activeSpace`

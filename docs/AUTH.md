# Auth And Context

The CLI uses API keys. It does not manage browser sessions.

## Login

`zeb login` (or `zeb auth login`) asks for or accepts an API key, validates it
against the Zebra Link API (`GET /api/v1/me`), and writes:

```text
~/.zlink/credentials.json
~/.zlink/config.json
```

The CLI always talks to the built-in production API. Login picks the
environment fresh each time — it never inherits a previously stored URL.

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
  "activeSpace": "spc_..."
}
```

## Resolution Order

API key:

1. `--api-key`
2. `ZLINK_API_KEY`
3. stored credentials

Space:

1. `--space`
2. `ZLINK_SPACE`
3. `activeSpace`

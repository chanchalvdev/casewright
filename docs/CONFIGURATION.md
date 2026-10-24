# Configuration

All settings are environment variables, read from `casewright/.env` (copy `casewright/.env.example`). Back to the [README](../README.md).

## Environment Variables

| Variable | Description | Default |
|---|---|---|
| `LLM_PROVIDER` | `gemini`, `ollama`, `openai` (OpenAI-compatible) or `none` (rules only, no AI) | auto: `gemini` if `GEMINI_API_KEY` is set |
| `GEMINI_API_KEY` | Gemini API key | — |
| `GEMINI_MODEL` | Gemini model | `gemini-3.8-flash` |
| `OLLAMA_MODEL` | Local chat model when `LLM_PROVIDER=ollama` | `llama3.2` |
| `KIMCHI_API_KEY` / `OPENAI_API_KEY` | Key for an OpenAI-compatible endpoint | — |
| `KIMCHI_BASE_URL` | OpenAI-compatible base URL | `https://api.kimchi.ai/v1` |
| `CASEWRIGHT_MODEL` | Model for the OpenAI-compatible endpoint | `kimi-k2-5` |
| `CASEWRIGHT_API_KEYS` | Comma-separated API keys for machine access (admin role) | — |
| `CASEWRIGHT_ADMIN_USER` / `CASEWRIGHT_ADMIN_PASSWORD` | Creates the first dashboard admin when no user exists ([docs/USERS.md](USERS.md)) | — |
| `CASEWRIGHT_SESSION_TTL` / `CASEWRIGHT_COOKIE_SECURE` | Login lifetime; Secure cookie behind HTTPS | `12h` / `false` |
| `INCIDENT_WINDOW` / `INCIDENT_MIN_SEVERITY` | Alert correlation window and threshold ([docs/INCIDENTS.md](INCIDENTS.md)) | `1h` / `P3` |
| `PLAYBOOKS_DIR` | Extra response playbooks ([docs/RESPONSE.md](RESPONSE.md)) | — |
| `RESPONSE_WEBHOOK_URL` / `SLACK_WEBHOOK_URL` | Where approved containment actions and notifications go | — |
| `RETENTION_EVENTS_DAYS` / `RETENTION_INCIDENTS_DAYS` / `RETENTION_AUDIT_DAYS` | Days to keep events, resolved incidents and audit entries (Postgres only); `0` keeps forever | `0` |
| `POSTGRES_DSN` | Postgres connection string; empty keeps events in memory | see `.env.example` |
| `DETECTION_MODE` | `rules-first`, `enrich` or `off` ([docs/DETECTION.md](DETECTION.md)) | `rules-first` |
| `SIGMA_RULES_DIR` | Extra Sigma rules folder, searched recursively | — |
| `SYSLOG_UDP_ADDR` / `SYSLOG_TCP_ADDR` | Syslog listener addresses, e.g. `:5514` | disabled |
| `IOC_WATCHLIST_DIR` | Directory of read-only IOC list files ([docs/WATCHLISTS.md](WATCHLISTS.md)) | — |
| `SEED_LOG_FILE` | Log file replayed through the pipeline on first start, when no events are stored (demos) | — |
| `CONDUCTOR_PORT` | HTTP server port | `8080` |
| `ALLOWED_ORIGIN` | CORS origin | `http://localhost:5173` |
| `QDRANT_ADDR` | Qdrant gRPC address | `localhost:6334` |
| `OLLAMA_URL` | Ollama base URL (embeddings, local LLM) | `http://localhost:11434` |
| `ABUSEIPDB_KEY` | AbuseIPDB API key for the agent's IP-reputation tool | optional |
| `OTX_API_KEY` | AlienVault OTX API key for the agent's threat-intel tool | optional |

Dashboard (in `casewright/web/.env.local`): `VITE_CASEWRIGHT_API_KEY` — the key the dashboard sends when `CASEWRIGHT_API_KEYS` is set.

## Sending Logs over Syslog

Enable the listener in `.env`:

```env
SYSLOG_UDP_ADDR=:5514
SYSLOG_TCP_ADDR=:5514
```

Point rsyslog at it (`/etc/rsyslog.d/90-casewright.conf`):

```
*.* @casewright-host:5514      # UDP
*.* @@casewright-host:5514     # TCP (newline-framed)
```

Or test by hand:

```bash
logger -n 127.0.0.1 -P 5514 -d "Failed password for root from 203.0.113.9 port 22 ssh2"
```

Each line is parsed (RFC 5424, RFC 3164 with or without `<PRI>` as written to `/var/log/auth.log`, or rsyslog's RFC 3339 "high-precision" format, falling back to raw text), queued and classified by the worker pool. When the queue is full, lines are dropped rather than slowing senders down; watch `ingest_dropped_total` in `/metrics`.

## Authentication

Three ways to authenticate (details in [docs/USERS.md](USERS.md)):

- **User accounts**: set `CASEWRIGHT_ADMIN_USER` and `CASEWRIGHT_ADMIN_PASSWORD` to create
  the first admin, then sign in to the dashboard and add users as **viewer**,
  **analyst** or **admin**. Sessions use an `HttpOnly`, `SameSite=Strict`
  cookie; every change is written to the audit log.
- **API keys** for scripts and integrations: set `CASEWRIGHT_API_KEYS`
  (comma-separated, so keys can be rotated). A key acts as an admin:

  ```bash
  curl -H "Authorization: Bearer $KEY" http://localhost:8080/api/events
  curl -H "X-API-Key: $KEY"            http://localhost:8080/api/events
  ```

  Browsers cannot set headers on a WebSocket handshake, so `/ws/alerts` also
  accepts `?api_key=<key>` (plain HTTP requests do not).
- **Open mode**: with neither configured, everything is allowed. For local
  development only; the server warns at start.

`/health`, `/health/ready`, `/docs` and `/api/auth/login` are always public.
See [SECURITY.md](../SECURITY.md) for deployment advice.

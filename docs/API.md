# API reference

The server also serves interactive Swagger UI at `/docs` and the OpenAPI spec at `/docs/openapi.yaml`. Back to the [README](../README.md).

All examples assume auth is off; add `-H "X-API-Key: $KEY"` when `CASEWRIGHT_API_KEYS` is set.

## `POST /api/classify`

Classify a single log line.

```bash
curl -X POST http://localhost:8080/api/classify \
  -H "Content-Type: application/json" \
  -d '{"log": "Failed password for root from 192.168.1.100 port 22", "format": "auto"}'
```

**Response:**
```json
{
  "event": { "raw": "...", "hostname": "webserver01", "source": "syslog" },
  "attack_type": "Brute Force",
  "severity": "P3",
  "confidence": 0.90,
  "mitre": { "tactic": "Credential Access", "technique_id": "T1110", "technique": "Brute Force" },
  "iocs": ["192.168.1.100", "root"],
  "summary": "Failed SSH login attempt for root from 192.168.1.100",
  "remediation": "Block IP, enforce key-based auth, disable root login"
}
```

## `POST /api/classify/stream`

Same as `/classify` but streams LLM tokens via SSE.

## `POST /api/ingest`

Batch classify up to 500 log lines and store in Qdrant + Postgres.

```bash
curl -X POST http://localhost:8080/api/ingest \
  -H "Content-Type: application/json" \
  -d '{"logs": ["log line 1", "log line 2"], "format": "auto"}'
```

## `GET /api/events?limit=100`

Most recent stored events, newest first (`limit` 1–500, default 100). With Postgres this includes history from before the last restart.

## `GET /api/detections/rules`

Loaded Sigma rules with level, tags, source, type (`single` or `threshold`),
enabled state and match count since start.

## `PATCH /api/detections/rules/{id}`

Enable or disable a rule: `{"enabled": false}`. Saved in Postgres, so it
survives restarts.

## Incidents

Related alerts are grouped into incidents with status, assignee, comments and
history. See [docs/INCIDENTS.md](INCIDENTS.md).

- `GET /api/incidents?status=open&entity=ip:1.2.3.4`
- `GET /api/incidents/stats`
- `GET /api/soc/metrics?days=30`: MTTD/MTTA/MTTR, volume, workload ([docs/INCIDENTS.md](INCIDENTS.md#soc-metrics))
- `GET /api/incidents/{id}`
- `PATCH /api/incidents/{id}` with `{"status","assignee","severity","resolution"}`
- `POST /api/incidents/{id}/comments` with `{"body": "..."}`
- `POST /api/incidents/{id}/investigate`: AI investigation of the whole incident
- `GET /api/incidents/{id}/report`: Markdown incident report
- `POST /api/incidents/{id}/feedback` with `{"helpful": true}`
- `GET /api/watchlists` · `GET /api/ioc/lookup?value=`: IOC watchlists ([docs/WATCHLISTS.md](WATCHLISTS.md))
- `GET /api/suppressions` · `POST /api/suppressions` · `DELETE /api/suppressions/{id}`: snooze noisy alerts

## Users and audit

See [docs/USERS.md](USERS.md): `POST /api/auth/login`, `GET /api/auth/me`,
`/api/users` (admin) and `GET /api/audit` (admin).

## Response playbooks

Playbooks propose actions (block IP, disable user, isolate host, notify) that
analysts approve. See [docs/RESPONSE.md](RESPONSE.md).

- `GET /api/playbooks`
- `GET /api/response/actions?status=pending`
- `POST /api/response/actions/{id}/approve` · `POST /api/response/actions/{id}/reject`
- `POST /api/incidents/{id}/playbooks/{playbook}/run`

## `GET /api/search?q=brute+force&limit=10`

Semantic vector search over stored events.

## `GET /api/analytics/summary`

Returns attack type counts, severity distribution, 6h timeline, and MITRE tactic breakdown.

## `GET /ws/alerts`

WebSocket endpoint streaming live incident/agent events (JSON frames). Each frame
is a sanitized `AgentEvent` — the UI reduces these into incident cards shown in the
alert ticker and incident overlay.

```bash
websocat ws://localhost:8080/ws/alerts
```

## `GET /health`

```json
{"status": "ok"}
```

## `GET /health/ready`

Checks the LLM, Postgres (when configured) and Qdrant; returns 503 if any is down.

## `GET /docs`

Interactive Swagger UI.

## `GET /metrics`

Prometheus metrics endpoint.

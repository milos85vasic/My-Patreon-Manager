---
title: "Status"
date: 2026-04-11
---

## Service Status

Check the live health endpoint:

```
GET /health
```

Returns:
```json
{"status": "ok"}
```

## Monitoring

- **Prometheus metrics**: `/metrics`
- **Grafana dashboard**: Import `ops/grafana/dashboard.json`
- **Audit log**: `GET /admin/audit` (requires `X-Admin-Key`)

## SLO Targets

| Endpoint | P99 Target |
|----------|-----------|
| /health | < 1ms |
| /webhook/* | < 50ms |
| /download/* | < 100ms |
| /admin/* | < 200ms |

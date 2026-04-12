# Admin Manual

## Webhook Setup

### GitHub
1. Go to your repo Settings > Webhooks > Add webhook
2. Payload URL: `https://your-domain:8080/webhook/github`
3. Content type: `application/json`
4. Secret: same value as `WEBHOOK_HMAC_SECRET` in `.env`
5. Events: select "Push events"

### GitLab
1. Go to Settings > Webhooks
2. URL: `https://your-domain:8080/webhook/gitlab`
3. Secret token: same as `WEBHOOK_HMAC_SECRET`
4. Trigger: Push events

### GitFlic / GitVerse
1. Navigate to repo webhook settings
2. URL: `https://your-domain:8080/webhook/gitflic` or `/gitverse`
3. Secret: same as `WEBHOOK_HMAC_SECRET`
4. Signature header: `X-Webhook-Signature` (HMAC-SHA256)

## Monitoring

### Prometheus metrics
Key metrics exposed at `/metrics`:
- `patreon_sync_repos_total` — repos processed per sync
- `patreon_sync_duration_seconds` — sync duration histogram
- `patreon_webhook_received_total` — webhook events received
- `patreon_llm_requests_total` — LLM API calls
- `patreon_patreon_publish_total` — Patreon posts created/updated

### Grafana
Import `ops/grafana/dashboard.json` for pre-built panels.

### Audit log
```bash
curl -H "X-Admin-Key: $ADMIN_KEY" http://localhost:8080/admin/audit
```

## Rate Limiting
- Default: 100 requests/second, burst 200
- Configure via `RATE_LIMIT_RPS` and `RATE_LIMIT_BURST`
- Stale IP entries evicted every 60s (10-minute TTL)

## Credential Rotation
1. Generate new token on the provider platform
2. Update `.env` with the new value
3. Restart the server (or send SIGHUP to reload `.repoignore`)
4. Verify via `./patreon-manager validate`

## SLO Reference
| Endpoint | P99 target | Notes |
|----------|-----------|-------|
| /health | < 1ms | Always fast |
| /webhook/* | < 50ms | Enqueue only, no sync |
| /download/* | < 100ms | Signed URL verify + serve |
| /admin/* | < 200ms | DB queries |

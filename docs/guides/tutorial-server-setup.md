# Tutorial: Setting Up the HTTP Server

This tutorial walks through starting the server, configuring webhooks, verifying every endpoint, and setting up monitoring.

## Step 1: Configure the server

Ensure these keys are set in `.env`:

```ini
PORT=8080
GIN_MODE=release
LOG_LEVEL=info
ADMIN_KEY=a-strong-random-key-here
WEBHOOK_HMAC_SECRET=another-strong-random-secret
RATE_LIMIT_RPS=100
RATE_LIMIT_BURST=200
```

Generate strong keys:

```bash
openssl rand -hex 32   # Use for ADMIN_KEY
openssl rand -hex 32   # Use for WEBHOOK_HMAC_SECRET
```

## Step 2: Start the server

```bash
./patreon-manager-server
```

Expected:
```
INFO  server started  port=8080  mode=release
```

## Step 3: Verify health endpoint

```bash
curl -s http://localhost:8080/health | jq .
```

Expected:
```json
{"status": "ok"}
```

## Step 4: Verify metrics endpoint

```bash
curl -s http://localhost:8080/metrics | head -5
```

Expected:
```
# HELP go_goroutines Number of goroutines that currently exist.
# TYPE go_goroutines gauge
go_goroutines 12
```

## Step 5: Verify admin endpoints

```bash
# Sync status
curl -s -H "X-Admin-Key: $ADMIN_KEY" http://localhost:8080/admin/sync-status | jq .

# Audit log
curl -s -H "X-Admin-Key: $ADMIN_KEY" http://localhost:8080/admin/audit | jq .

# Without the key → 401
curl -s http://localhost:8080/admin/sync-status
# {"error":"unauthorized"}
```

## Step 6: Verify webhook authentication

```bash
# Without signature → 401
curl -s -X POST http://localhost:8080/webhook/github -d '{}' -H "Content-Type: application/json"
# {"error":"invalid signature"}

# With valid HMAC signature → 200
BODY='{"repository":{"full_name":"org/repo"}}'
SIG=$(echo -n "$BODY" | openssl dgst -sha256 -hmac "$WEBHOOK_HMAC_SECRET" | awk '{print $2}')
curl -s -X POST http://localhost:8080/webhook/github \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=$SIG" \
  -d "$BODY"
```

## Step 7: Verify rate limiting

```bash
# Rapid-fire 300 requests — some should get 429
for i in $(seq 1 300); do
  CODE=$(curl -s -o /dev/null -w '%{http_code}' http://localhost:8080/health)
  if [ "$CODE" = "429" ]; then
    echo "Rate limited at request $i"
    break
  fi
done
```

## Step 8: Configure GitHub webhook

1. Go to your GitHub repository → Settings → Webhooks → Add webhook
2. Payload URL: `https://your-domain:8080/webhook/github`
3. Content type: `application/json`
4. Secret: paste your `WEBHOOK_HMAC_SECRET` value
5. Events: select "Push events"
6. Click "Add webhook"

GitHub sends a ping event. Check your server logs:
```
INFO  webhook received  provider=github  event=ping
```

## Step 9: Configure GitLab webhook

1. Go to Settings → Webhooks
2. URL: `https://your-domain:8080/webhook/gitlab`
3. Secret token: paste your `WEBHOOK_HMAC_SECRET`
4. Trigger: Push events
5. Click "Add webhook"

## Step 10: Set up Grafana monitoring

```bash
# Start Prometheus + Grafana (example docker-compose)
podman-compose up -d prometheus grafana

# Import the dashboard
# Open http://localhost:3000 → Dashboards → Import
# Upload: ops/grafana/dashboard.json
# Select Prometheus as the data source
```

The dashboard shows six panels:
- HTTP requests per second
- HTTP error rate
- P99 latency per route
- Sync repos processed per second
- LLM API calls per second
- Database query rate

## Step 11: Verify pprof (debugging)

```bash
# Goroutine dump
curl -s -H "X-Admin-Key: $ADMIN_KEY" \
  http://localhost:8080/debug/pprof/goroutine?debug=1 | head -30

# Heap profile
curl -s -H "X-Admin-Key: $ADMIN_KEY" \
  http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

## Step 12: Graceful shutdown

Send SIGTERM or SIGINT:

```bash
kill $(pgrep patreon-manager-server)
```

Expected:
```
INFO  shutting down gracefully
INFO  webhook queue drained
INFO  rate limiter stopped
INFO  dedup cleanup stopped
INFO  server stopped
```

The server:
1. Stops accepting new connections
2. Drains the webhook queue
3. Stops the rate limiter sweeper
4. Closes the EventDeduplicator
5. Exits cleanly

## Troubleshooting

| Symptom | Cause | Fix |
|---------|-------|-----|
| `bind: address already in use` | Another process on port 8080 | Change `PORT` in `.env` or `kill $(lsof -ti:8080)` |
| `401` on all webhook calls | Wrong `WEBHOOK_HMAC_SECRET` | Ensure the secret in `.env` matches the one in GitHub/GitLab |
| `429` on legitimate traffic | Rate limit too low | Increase `RATE_LIMIT_RPS` and `RATE_LIMIT_BURST` |
| No metrics data in Grafana | Prometheus not scraping | Check Prometheus config targets at `http://localhost:9090/targets` |

# HTTP Endpoint Contracts

The Gin-based web server provides webhook reception, health monitoring,
content access, and metrics endpoints.

## Health Check

```
GET /health
```

**Response 200**:
```json
{"status": "healthy", "version": "1.0.0", "uptime_seconds": 86400}
```

## Webhook Endpoints

### GitHub Webhook

```
POST /webhook/github
```

**Headers**: `X-Hub-Signature-256` (HMAC-SHA256 validation required)
**Content-Type**: `application/json`

**Handled events**: `push`, `release`, `repository` (archived/deleted)

**Response 200**:
```json
{"status": "queued", "event_id": "abc123", "repository": "owner/repo"}
```

**Response 401**: Invalid or missing signature.

### GitLab Webhook

```
POST /webhook/gitlab
```

**Headers**: `X-Gitlab-Token` (secret token validation)
**Content-Type**: `application/json`

**Handled events**: `push`, `tag_push`, `repository_update`

### Generic Webhook

```
POST /webhook/{service}
```

Services: `gitflic`, `gitverse`. Query parameter `token` for validation.

## Content Access Endpoints

### Download Premium Content

```
GET /download/{content_id}?token=***
```

**Auth**: Signed URL token with expiration and tier validation.

**Response 200**: File content with `Content-Disposition: attachment`.
**Response 403**: Token expired or insufficient tier.
**Response 404**: Content not found.

### Verify Access

```
GET /access/{content_id}
```

**Auth**: Patreon session cookie or Bearer token.

**Response 200**:
```json
{"access": true, "tier": "premium", "expires_at": "2026-04-10T00:00:00Z"}
```

**Response 403**:
```json
{"access": false, "required_tier": "premium", "current_tier": "basic",
 "upgrade_url": "https://patreon.com/..."}
```

## Metrics

```
GET /metrics
```

**Format**: Prometheus exposition format.

**Metrics exposed**:
- `sync_duration_seconds` (histogram)
- `sync_success_rate` (gauge)
- `repos_processed_total` (counter)
- `patreon_api_errors_total` (counter by type)
- `llm_latency_seconds` (histogram)
- `llm_tokens_total` (counter)
- `llm_quality_score` (gauge)
- `webhook_events_total` (counter by service)
- `active_sync_lock` (gauge: 0 or 1)

## Admin Endpoints

### Reload Configuration

```
POST /admin/reload
```

**Auth**: Admin API key in `X-Admin-Key` header.

**Response 200**:
```json
{"status": "reloaded", "config_source": ".env"}
```

### Sync Status

```
GET /admin/sync/status
```

**Auth**: Admin API key.

**Response 200**:
```json
{"active": true, "pid": 12345, "started_at": "2026-04-09T10:00:00Z",
 "repos_completed": 12, "repos_total": 23, "eta_seconds": 180}
```

# Troubleshooting FAQ

## 1. "validate" command reports missing environment variables

**Q:** I run `go run ./cmd/cli validate` and get errors about missing configuration.

**A:** Ensure you have copied `.env.example` to `.env` and filled in all required values:
```sh
cp .env.example .env
# Edit .env with your actual credentials
```
Required variables include `PATREON_CLIENT_ID`, `PATREON_CLIENT_SECRET`, `PATREON_ACCESS_TOKEN`, `PATREON_REFRESH_TOKEN`, `PATREON_CAMPAIGN_ID`, and at least one Git provider token (e.g., `GITHUB_TOKEN`). Run `validate` again after updating.

---

## 2. GitHub API rate limit exceeded

**Q:** Sync fails with a 403 error mentioning rate limits from GitHub.

**A:** GitHub allows 5,000 requests/hour for authenticated users. The application implements rate limiting and exponential backoff, but large repository portfolios can still exhaust the quota.

Mitigation options:
- Use `--org` or `--repo` flags to scope the sync to specific organizations or repositories.
- Configure `GITHUB_TOKEN_SECONDARY` in `.env` for automatic failover when the primary token is rate-limited.
- Wait for the rate limit window to reset (check the `X-RateLimit-Reset` header in logs at DEBUG level).
- Run the sync less frequently via `--schedule`.

---

## 3. Circuit breaker is open for a provider

**Q:** Logs show "circuit breaker is open" and requests to a provider are being rejected immediately.

**A:** The circuit breaker trips after a configurable number of consecutive failures to protect the system from cascading failures. The breaker will transition to half-open state after a cooldown period and attempt a probe request.

Steps:
1. Check the provider's status page for outages.
2. Verify your credentials are still valid: `go run ./cmd/cli validate`
3. Check Prometheus metrics for `circuit_breaker_state_changes_total` to see the breaker history.
4. The breaker will auto-recover once the provider is healthy. No manual intervention is typically needed.
5. If the issue persists, restart the application to reset all circuit breakers.

---

## 4. Duplicate content appearing on Patreon

**Q:** The same repository content is being posted multiple times to Patreon.

**A:** This should not happen under normal operation -- the application uses content fingerprinting to prevent duplicates. If duplicates appear:

1. Check the database for duplicate fingerprint entries: the content fingerprint for each repository should be unique.
2. Verify the database file has not been corrupted: `sqlite3 patreon-manager.db "PRAGMA integrity_check;"`
3. If the database was lost or recreated, the application may re-generate content. Run `sync --dry-run` first to preview what would be published.
4. Manually remove duplicate posts from Patreon and then run a sync to re-establish the local-to-remote mapping.

---

## 5. SQLite "database is locked" errors

**Q:** I see "database is locked" errors in the logs.

**A:** SQLite does not support concurrent writers. This typically happens when:

- Multiple instances of the CLI are running simultaneously.
- The HTTP server is handling webhook-triggered writes while a CLI sync is in progress.

Solutions:
- Ensure only one sync process runs at a time (use `--schedule` with cron rather than multiple parallel invocations).
- For high-concurrency deployments, switch to PostgreSQL by setting `DB_DRIVER=postgres` and configuring the PostgreSQL connection variables.
- Check for zombie processes: `ps aux | grep patreon-manager`

---

## 6. LLM content generation fails or produces low-quality output

**Q:** Content generation returns errors or the quality score is below threshold.

**A:** The LLM pipeline uses quality gates with a configurable threshold (`CONTENT_QUALITY_THRESHOLD`, default 0.75). When content falls below this threshold:

1. The system automatically retries with alternative prompts or models via the fallback chain.
2. If all fallback models fail, the content enters a human review queue.
3. Check `LLM_PROVIDER` and related configuration in `.env`.
4. Review the `llm_quality_score` and `llm_latency_seconds` Prometheus metrics for trends.
5. Verify your LLM provider API key is valid and has sufficient quota.
6. Try lowering `CONTENT_QUALITY_THRESHOLD` temporarily (not recommended for production).

---

## 7. GitFlic or GitVerse API errors

**Q:** Scanning fails specifically for GitFlic or GitVerse repositories.

**A:** These services have different API conventions than GitHub/GitLab:

1. Verify your token is valid by testing the API directly:
   ```sh
   curl -H "Authorization: Bearer $GITFLIC_TOKEN" https://api.gitflic.ru/user
   ```
2. Check if the service is experiencing an outage.
3. GitFlic and GitVerse have different rate limit thresholds -- check logs at DEBUG level for rate limit headers.
4. Ensure the token has the required scopes for repository listing and metadata access.
5. If using secondary tokens, verify those are also valid.

---

## 8. Mirror detection incorrectly links/unlinks repositories

**Q:** Two unrelated repositories are being detected as mirrors of each other, or actual mirrors are not being detected.

**A:** Mirror detection uses multiple signals: exact name matching, README content hashing, and commit SHA comparison. False positives/negatives can occur when:

- Two unrelated repositories share the same name across services.
- A forked repository has diverged significantly from the original.
- The repository was recently created on a new service and has not been pushed yet.

Solutions:
- Use `.repoignore` to exclude specific repositories from scanning.
- Run `scan --json` to inspect the mirror detection output and verify the linkage.
- If the issue persists, check the database for stale mirror state and consider clearing the mirror cache.

---

## 9. Prometheus metrics endpoint returns no data

**Q:** The `/metrics` endpoint on the HTTP server returns empty or minimal data.

**A:** Metrics are populated during sync operations. If no sync has run since the server started, most application-specific metrics will have zero values or be absent.

Steps:
1. Verify the server is running: `curl http://localhost:8080/health`
2. Run a sync: `go run ./cmd/cli sync`
3. Check the metrics endpoint again: `curl http://localhost:8080/metrics`
4. Verify Prometheus is configured to scrape the correct endpoint and port.
5. Application metrics use the `patreon_manager_` prefix -- filter for that in your Prometheus/Grafana queries.

---

## 10. "go build" fails with CGo errors

**Q:** Building the application fails with CGo-related errors, especially on systems without GCC.

**A:** The SQLite driver (`mattn/go-sqlite3`) requires CGo and a C compiler:

- **Linux**: `sudo apt-get install gcc` or `sudo dnf install gcc`
- **macOS**: `xcode-select --install`
- **Windows**: Install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or use MSYS2.
- **Cross-compilation**: CGo complicates cross-compilation. Build on the target platform or use Docker multi-stage builds.

Verify CGo is enabled: `go env CGO_ENABLED` should return `1`.

---

## 11. Webhook handler returns 401 Unauthorized

**Q:** Patreon webhook requests are rejected with 401 errors.

**A:** Webhook requests are verified using HMAC-SHA256 signatures:

1. Verify `HMAC_SECRET` in `.env` matches the webhook secret configured in the Patreon Developer Portal.
2. Check that the `X-Patreon-Signature` header is present in the webhook request (visible in DEBUG logs).
3. If you recently rotated the HMAC secret, update it in both `.env` and the Patreon webhook configuration.
4. Ensure the webhook URL in Patreon points to your server's correct address and port.

---

## 12. Application crashes on startup with "address already in use"

**Q:** The HTTP server fails to start with "bind: address already in use".

**A:** Another process is already listening on port 8080:

1. Find the conflicting process: `lsof -i :8080` or `ss -tlnp | grep 8080`
2. Stop the conflicting process, or configure a different port via the `PORT` environment variable.
3. If running multiple instances, use different ports for each.

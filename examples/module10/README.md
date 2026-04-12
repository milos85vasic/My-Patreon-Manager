# Module 10 Exercise: Observability

1. Start the server and Prometheus (via podman-compose)
2. Trigger a sync: `./patreon-manager sync --dry-run`
3. Query metrics: `curl http://localhost:8080/metrics | grep patreon_sync`
4. Import `ops/grafana/dashboard.json` and observe panels updating
5. Query the audit endpoint:
   ```
   curl -H "X-Admin-Key: $ADMIN_KEY" http://localhost:8080/admin/audit
   ```

# Module 06 Exercise: Administration

1. Start the server: `./patreon-manager-server`
2. Hit endpoints:
   ```
   curl http://localhost:8080/health
   curl -H "X-Admin-Key: $ADMIN_KEY" http://localhost:8080/admin/sync-status
   curl -H "X-Admin-Key: $ADMIN_KEY" http://localhost:8080/admin/audit
   ```
3. Import `ops/grafana/dashboard.json` into Grafana

# Module 08 Exercise: Troubleshooting

1. Set an invalid `GITHUB_TOKEN` and observe token failover in debug logs:
   ```
   LOG_LEVEL=debug ./patreon-manager sync --dry-run 2>&1 | head -50
   ```
2. Start the server and capture a goroutine dump:
   ```
   curl -H "X-Admin-Key: $ADMIN_KEY" http://localhost:8080/debug/pprof/goroutine?debug=1
   ```

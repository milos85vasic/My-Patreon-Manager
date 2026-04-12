# Video Course Exercise Files

Each directory contains starter files for the corresponding video course module.

| Directory | Module | Exercise |
|-----------|--------|----------|
| module01/ | Introduction | Clone, build, validate, dry-run |
| module02/ | Configuration | Fill in .env, run validate |
| module03/ | Sync | Create .repoignore, run scan |
| module04/ | Generate | Modify quality threshold, generate content |
| module05/ | Publish | Dry-run publish, verify idempotency |
| module06/ | Admin | Start server, hit admin endpoints, import Grafana |
| module07/ | Extending | Add a stub provider, add a renderer |
| module08/ | Troubleshooting | Debug token issues, read pprof |
| module09/ | Concurrency | Find and fix a race condition |
| module10/ | Observability | Trigger sync, observe Prometheus metrics |

## Usage

```bash
# Spin out a clean copy for any module:
bash scripts/video/spinout_example.sh module03
```

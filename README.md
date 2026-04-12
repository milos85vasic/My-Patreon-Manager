# My Patreon Manager

My Patreon Manager is a Go application that automates content creation and publishing for Patreon creators. It scans Git repositories across GitHub, GitLab, GitFlic, and GitVerse, generates tier-gated content via an LLM pipeline with quality gates, and publishes posts to Patreon -- all driven by a CLI-first design that supports cron scheduling, dry-run previews, and idempotent operations.

## Features

- **Multi-platform Git scanning** -- GitHub, GitLab, GitFlic, and GitVerse as first-class, interchangeable sources with mirror detection
- **LLM-powered content generation** -- quality-scored model selection with automatic fallback chains and configurable quality thresholds
- **Tier-gated Patreon publishing** -- maps repository content to Patreon tiers with deduplication via content fingerprinting
- **CLI subcommands** -- `sync`, `scan`, `generate`, `validate`, `publish` with `--dry-run`, `--schedule`, `--org`, `--repo`, `--pattern`, `--json`, `--log-level`
- **HTTP server** -- Gin-based server on `:8080` with health checks, Prometheus metrics, and webhook handlers
- **Resilience patterns** -- circuit breakers (gobreaker), exponential backoff, rate limiting per provider
- **Observability** -- structured logging, Prometheus metrics (`sync_duration_seconds`, `repos_processed_total`, `llm_quality_score`, etc.)
- **Idempotent operations** -- content fingerprinting and checkpointing ensure safe re-runs after failures
- **Database flexibility** -- SQLite by default, PostgreSQL for production deployments
- **Security-first** -- twelve-factor credential management, automatic token refresh, credential redaction in logs

## Quickstart

1. **Clone the repository**
   ```sh
   git clone https://github.com/milos85vasic/My-Patreon-Manager.git
   cd My-Patreon-Manager
   ```

2. **Configure credentials**
   ```sh
   cp .env.example .env
   # Edit .env with your Patreon API credentials, Git provider tokens, etc.
   ```

3. **Validate configuration**
   ```sh
   go run ./cmd/cli validate
   ```

4. **Dry-run a sync**
   ```sh
   go run ./cmd/cli sync --dry-run
   ```

5. **Run a full sync**
   ```sh
   go run ./cmd/cli sync
   ```

## Architecture

![Overview](docs/architecture/diagrams/overview.svg)

The codebase follows a provider/service layering where the CLI and server are thin wrappers around a shared orchestration core. See [docs/architecture/overview.md](docs/architecture/overview.md) for details.

**Key layers:**

- **Providers** (`internal/providers/`) -- pluggable external integrations (Git services, LLM, Patreon, renderers) behind Go interfaces
- **Services** (`internal/services/`) -- orchestration logic (sync, content generation, filtering, access control, audit)
- **Entrypoints** (`cmd/cli/`, `cmd/server/`) -- thin wrappers with dependency-injection via package-level function variables

## Documentation

| Document | Description |
|----------|-------------|
| [Quickstart Guide](docs/guides/quickstart.md) | Getting started in 5 minutes |
| [Configuration](docs/guides/configuration.md) | Environment variables and config file reference |
| [Architecture Overview](docs/architecture/overview.md) | System design and component interactions |
| [SQL Schema](docs/architecture/sql-schema.md) | Database schema reference |
| [Git Providers](docs/guides/git-providers.md) | Provider-specific setup and configuration |
| [Content Generation](docs/guides/content-generation.md) | LLM pipeline and quality gates |
| [Deployment](docs/guides/deployment.md) | Production deployment guide |
| [API Reference](docs/api/openapi.yaml) | OpenAPI specification |
| [CLI Reference](docs/api/cli-reference.md) | CLI subcommands and flags |
| [ADRs](docs/adr/) | Architecture Decision Records |
| [Runbooks](docs/runbooks/) | Operational procedures |
| [Troubleshooting FAQ](docs/troubleshooting/faq.md) | Common issues and solutions |
| [Security](docs/security/README.md) | Security policies and baselines |
| [Main Specification](docs/main_specification.md) | Full system specification |

## Development

```sh
go build ./...                                  # build all packages
go test ./internal/... ./cmd/... ./tests/...    # run full test suite
go test -race ./...                             # race detector
go vet ./...                                    # static analysis
bash scripts/coverage.sh                        # coverage (gates at 100%)
go run ./cmd/server                             # run HTTP server
```

## License

See the project license file for terms and conditions.

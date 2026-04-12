# Deployment: Bare Binary

## 1. Build

```bash
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o patreon-manager ./cmd/cli
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o patreon-manager-server ./cmd/server
```

Note: SQLite requires CGO. For SQLite support, use `CGO_ENABLED=1` and ensure `gcc` is available.

## 2. Configure

```bash
cp .env.example .env
# Edit .env with your tokens
```

## 3. Run

```bash
# CLI
./patreon-manager validate
./patreon-manager sync --dry-run

# Server (background)
./patreon-manager-server &
curl http://localhost:8080/health
```

## 4. Cross-compile

```bash
GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -o patreon-manager-linux-arm64 ./cmd/cli
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -o patreon-manager-darwin-amd64 ./cmd/cli
```

## 5. Verify

```bash
./patreon-manager validate && echo "Ready"
```

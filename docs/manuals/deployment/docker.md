# Deployment: Docker

## 1. Build

```bash
docker build -t patreon-manager:latest .
```

## 2. Run the server

```bash
docker run -d --name pm-server \
  -p 8080:8080 \
  --env-file .env \
  -v pm-data:/data \
  patreon-manager:latest
```

## 3. Run CLI commands

```bash
docker run --rm --env-file .env -v pm-data:/data \
  patreon-manager:latest patreon-manager sync --dry-run
```

## 4. Docker Compose

Use the existing `docker-compose.yml`:

```bash
docker-compose up -d
```

## 5. Health check

```bash
curl http://localhost:8080/health
```

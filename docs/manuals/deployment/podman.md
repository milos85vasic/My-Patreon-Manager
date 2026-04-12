# Deployment: Podman

## 1. Build the image

```bash
podman build -t localhost/patreon-manager:latest .
```

## 2. Create a pod

```bash
podman pod create --name patreon -p 8080:8080
```

## 3. Run the server

```bash
podman run -d --pod patreon --name pm-server \
  --env-file .env \
  -v pm-data:/data \
  localhost/patreon-manager:latest
```

## 4. Health check

```bash
curl http://localhost:8080/health
```

Expected: `{"status":"ok"}`

## 5. Systemd integration

```bash
podman generate systemd --name pm-server > ~/.config/systemd/user/pm-server.service
systemctl --user enable --now pm-server.service
```

## 6. Run CLI commands

```bash
podman run --rm --env-file .env \
  -v pm-data:/data \
  localhost/patreon-manager:latest \
  patreon-manager sync --dry-run
```

## 7. Security scanning

```bash
podman-compose -f docker-compose.security.yml run --rm gosec
podman-compose -f docker-compose.security.yml run --rm trivy-fs
```

See docs/security/README.md for the full scanner list.

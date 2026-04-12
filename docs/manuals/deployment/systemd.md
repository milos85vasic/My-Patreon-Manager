# Deployment: systemd

## 1. Build the binary

```bash
go build -o /usr/local/bin/patreon-manager-server ./cmd/server
```

## 2. Create a system user

```bash
sudo useradd -r -s /sbin/nologin patreon-manager
sudo mkdir -p /var/lib/patreon-manager
sudo chown patreon-manager:patreon-manager /var/lib/patreon-manager
```

## 3. Create the unit file

`/etc/systemd/system/patreon-manager.service`:

```ini
[Unit]
Description=My Patreon Manager Server
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=patreon-manager
Group=patreon-manager
EnvironmentFile=/etc/patreon-manager/env
ExecStart=/usr/local/bin/patreon-manager-server
WorkingDirectory=/var/lib/patreon-manager
Restart=on-failure
RestartSec=5
LimitNOFILE=65536

[Install]
WantedBy=multi-user.target
```

## 4. Deploy the env file

```bash
sudo mkdir -p /etc/patreon-manager
sudo cp .env /etc/patreon-manager/env
sudo chmod 600 /etc/patreon-manager/env
sudo chown patreon-manager:patreon-manager /etc/patreon-manager/env
```

## 5. Enable and start

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now patreon-manager
sudo systemctl status patreon-manager
```

## 6. Logs

```bash
journalctl -u patreon-manager -f
```

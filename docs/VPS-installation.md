# Beamdrop VPS Installation Guide

A lightweight, self-hosted file sharing server with S3-compatible API, designed to run on a VPS with persistent background service.

---

## Prerequisites

- Linux VPS (Ubuntu/Debian recommended)
- User with `sudo` privileges
- `curl` or `wget`
- `tar`
- Nginx installed (optional, for reverse proxy & HTTPS)
- Ports: 7777 (Beamdrop) and 80/443 (optional for Nginx)

---

## Step 1 Download Beamdrop

Determine your VPS architecture:

```bash
uname -m
```

- `x86_64` → AMD64
- `aarch64` → ARM64

Download the corresponding release from GitHub:

```bash
# AMD64
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-linux-amd64.tar.gz -o beamdrop.tar.gz

# ARM64
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-linux-arm64.tar.gz -o beamdrop.tar.gz
```

Extract and move to `/usr/local/bin`:

```bash
sudo tar -C /usr/local/bin -xzf beamdrop.tar.gz
rm beamdrop.tar.gz
```

Make sure it’s executable:

```bash
chmod +x /usr/local/bin/beamdrop
```

---

## Step 2 Run Beamdrop (Temporary)

Test Beamdrop:

```bash
beamdrop -p='yourpassword' --qr
```

Open in browser:

```
http://your-vps-ip:7777
```

This confirms Beamdrop runs before setting up systemd.

---

## Step 3 Create a Systemd Service

Create service file:

```bash
sudo nano /etc/systemd/system/beamdrop.service
```

Paste:

```ini
[Unit]
Description=Beamdrop File Server
After=network.target

[Service]
User=forge
WorkingDirectory=/home/forge
ExecStart=/usr/local/bin/beamdrop -p='yourpassword' --qr
Restart=always
RestartSec=5
Environment=BEAMDROP_PORT=7777
Environment=BEAMDROP_API_AUTH=true

[Install]
WantedBy=multi-user.target
```

Save & exit (`CTRL + O`, `CTRL + X`).

Reload systemd:

```bash
sudo systemctl daemon-reload
```

Start and enable service:

```bash
sudo systemctl start beamdrop
sudo systemctl enable beamdrop
```

Check status:

```bash
sudo systemctl status beamdrop
```

Beamdrop now runs **persistently in the background**.

View logs:

```bash
journalctl -u beamdrop -f
```

---

## Step 4 Set Up Nginx Reverse Proxy (Optional)

This allows HTTPS and a proper domain instead of exposing port 7777.

1. Create a new site in Nginx:

```bash
sudo nano /etc/nginx/sites-available/beamdrop.example.com
```

Example config:

```nginx
server {
    listen 80;
    server_name beamdrop.example.com;

    location / {
        proxy_pass http://127.0.0.1:7777;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

Enable site and restart Nginx:

```bash
sudo ln -s /etc/nginx/sites-available/beamdrop.example.com /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl restart nginx
```

Optional: Use **Let's Encrypt** or Forge’s SSL to enable HTTPS.

---

## Step 5 Security Considerations

- Disable public access to port 7777 if using Nginx:

```bash
sudo ufw delete allow 7777
```

- Keep strong password for web UI (`-p` flag)
- Enable API key authentication: `--api-auth` or `BEAMDROP_API_AUTH=true`
- Consider rate limiting with `--rate-limit` or `BEAMDROP_RATE_LIMIT`
- Set a storage limit with `--max-storage` or `BEAMDROP_MAX_STORAGE` (bytes, 0 = unlimited)

---

## Step 6 Optional Environment Variables

You can set environment variables in systemd service for additional config:

| Variable                   | Purpose                              |
| -------------------------- | ------------------------------------ |
| `BEAMDROP_PORT`            | Change default port (7777)           |
| `BEAMDROP_API_AUTH`        | Enable API key authentication        |
| `BEAMDROP_ALLOWED_ORIGINS` | Set allowed CORS origins             |
| `BEAMDROP_RATE_LIMIT`      | Requests per minute per IP           |
| `BEAMDROP_MAX_STORAGE`     | Max storage in bytes (0 = unlimited) |

---

## Step 7 Quick Commands

```bash
# Start service
sudo systemctl start beamdrop

# Stop service
sudo systemctl stop beamdrop

# Restart service
sudo systemctl restart beamdrop

# View logs
journalctl -u beamdrop -f
```

---

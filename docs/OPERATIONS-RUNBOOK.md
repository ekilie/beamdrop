# Beamdrop Operations Runbook

> Day-to-day operations guide for running Beamdrop in production.

---

## Table of Contents

1. [Installation Steps](#1-installation-steps)
2. [Configuration Reference](#2-configuration-reference)
3. [Backup and Restore](#3-backup-and-restore)
4. [Monitoring Setup](#4-monitoring-setup)
5. [Scaling Guidance](#5-scaling-guidance)
6. [Common Troubleshooting](#6-common-troubleshooting)

---

## 1. Installation Steps

### 1.1 Docker Compose (Recommended for Production)

Docker Compose is the simplest and most reliable deployment method.

**Prerequisites:** Docker 24+ and Docker Compose v2.

```bash
# Clone the repository (or copy docker-compose.yml + Caddyfile)
git clone https://github.com/ekilie/beamdrop.git
cd beamdrop

# Create your environment file
cp .env.example .env   # or create one manually (see Section 2)

# Start in the background
docker compose up -d

# Verify startup
docker compose ps
docker compose logs -f beamdrop
```

**With automatic HTTPS via Caddy:**

```bash
# 1. Set your domain
export BEAMDROP_DOMAIN=files.example.com

# 2. Uncomment the caddy service in docker-compose.yml

# 3. Start
docker compose up -d
```

Caddy automatically provisions a Let's Encrypt TLS certificate.  
Data is persisted in `./data/` on the host.

---

### 1.2 Docker (Standalone)

```bash
# Build the image
docker build -t beamdrop .

# Run with a named volume
docker run -d \
  --name beamdrop \
  --restart unless-stopped \
  -p 7777:7777 \
  -v beamdrop-data:/data \
  -e BEAMDROP_PASSWORD="change-me" \
  -e BEAMDROP_API_AUTH=true \
  -e BEAMDROP_RATE_LIMIT=100 \
  beamdrop

# Check health
curl http://localhost:7777/health/live
```

---

### 1.3 Pre-Built Binary on Linux VPS

**Step 1 Determine architecture:**

```bash
uname -m
# x86_64  → AMD64
# aarch64 → ARM64
```

**Step 2 Download and install:**

```bash
# AMD64
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-linux-amd64.tar.gz \
  -o beamdrop.tar.gz

# ARM64
curl -L https://github.com/ekilie/beamdrop/releases/latest/download/beamdrop-linux-arm64.tar.gz \
  -o beamdrop.tar.gz

sudo tar -C /usr/local/bin -xzf beamdrop.tar.gz
rm beamdrop.tar.gz
beamdrop -v   # verify
```

**Step 3 Create a dedicated service user and data directory:**

```bash
sudo useradd --system --no-create-home --shell /usr/sbin/nologin beamdrop
sudo mkdir -p /var/lib/beamdrop
sudo chown beamdrop:beamdrop /var/lib/beamdrop
```

**Step 4 Create the systemd unit:**

```bash
sudo tee /etc/systemd/system/beamdrop.service > /dev/null <<'EOF'
[Unit]
Description=Beamdrop File Server
Documentation=https://github.com/ekilie/beamdrop
After=network.target

[Service]
User=beamdrop
Group=beamdrop
WorkingDirectory=/var/lib/beamdrop

ExecStart=/usr/local/bin/beamdrop \
  -dir /var/lib/beamdrop \
  -db-path /var/lib/beamdrop/.beamdrop/beamdrop.db \
  -log-level info \
  -rate-limit 100

# Environment overrides   keep secrets here, not in command flags
EnvironmentFile=-/etc/beamdrop/beamdrop.env

Restart=on-failure
RestartSec=5s
TimeoutStopSec=30s

# Hardening
NoNewPrivileges=yes
PrivateTmp=yes
ProtectSystem=strict
ReadWritePaths=/var/lib/beamdrop

[Install]
WantedBy=multi-user.target
EOF
```

**Step 5 Create the environment file with secrets:**

```bash
sudo mkdir -p /etc/beamdrop
sudo tee /etc/beamdrop/beamdrop.env > /dev/null <<'EOF'
BEAMDROP_PASSWORD=change-me-strong-password
BEAMDROP_API_AUTH=true
BEAMDROP_PORT=7777
EOF
sudo chmod 600 /etc/beamdrop/beamdrop.env
sudo chown root:root /etc/beamdrop/beamdrop.env
```

**Step 6 Enable and start:**

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now beamdrop
sudo systemctl status beamdrop
```

---

### 1.4 Nginx Reverse Proxy (Optional)

Place Beamdrop behind Nginx for TLS termination and a proper domain name.

```nginx
# /etc/nginx/sites-available/beamdrop
server {
    listen 80;
    server_name files.example.com;
    return 301 https://$host$request_uri;
}

server {
    listen 443 ssl http2;
    server_name files.example.com;

    ssl_certificate     /etc/letsencrypt/live/files.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/files.example.com/privkey.pem;

    # Increase for large file uploads
    client_max_body_size 10G;

    # WebSocket support (required for /ws/stats)
    location / {
        proxy_pass         http://127.0.0.1:7777;
        proxy_http_version 1.1;
        proxy_set_header   Upgrade $http_upgrade;
        proxy_set_header   Connection "upgrade";
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
        proxy_set_header   X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
        proxy_read_timeout 300s;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/beamdrop /etc/nginx/sites-enabled/
sudo nginx -t && sudo systemctl reload nginx

# Obtain TLS certificate with Certbot
sudo certbot --nginx -d files.example.com
```

After enabling Nginx, close direct access to port 7777:

```bash
sudo ufw delete allow 7777
sudo ufw allow 443/tcp
sudo ufw allow 80/tcp
```

---

### 1.5 Build from Source

```bash
# Prerequisites: Go 1.21+, Node.js 18+, bun
git clone https://github.com/ekilie/beamdrop.git
cd beamdrop

# Build frontend + backend
make build

# Binary is at ./cmd/beam/beamdrop
./cmd/beam/beamdrop -v
```

---

## 2. Configuration Reference

### 2.1 Environment Variables

All variables can be set in `.env` (Docker Compose), `/etc/beamdrop/beamdrop.env` (systemd), or exported directly.

| Variable                   | Default                             | Description                                                |
| -------------------------- | ----------------------------------- | ---------------------------------------------------------- |
| `BEAMDROP_PORT`            | `7777`                              | Port to listen on                                          |
| `BEAMDROP_PASSWORD`        | _(none)_                            | Web UI password. Leave unset to disable auth               |
| `BEAMDROP_LOG_LEVEL`       | `info`                              | Log verbosity: `debug`, `info`, `warn`, `error`            |
| `BEAMDROP_RATE_LIMIT`      | `100`                               | General rate limit in requests/min per IP (`0` = disabled) |
| `BEAMDROP_API_AUTH`        | `false`                             | Enable S3 API key authentication                           |
| `BEAMDROP_QR`              | `false`                             | Print a QR code to the terminal at startup                 |
| `BEAMDROP_ALLOWED_ORIGINS` | _(none)_                            | Comma-separated CORS origins. Empty = CORS disabled        |
| `BEAMDROP_DB_PATH`         | `<sharedDir>/.beamdrop/beamdrop.db` | SQLite database path or parent directory                   |
| `BEAMDROP_TLS_CERT`        | _(none)_                            | Path to TLS certificate (PEM)                              |
| `BEAMDROP_TLS_KEY`         | _(none)_                            | Path to TLS private key (PEM)                              |

### 2.2 Command-Line Flags

Flags override environment variables when both are set.

| Flag               | Description                              | Default                   |
| ------------------ | ---------------------------------------- | ------------------------- |
| `-dir`             | Directory to share                       | Current directory         |
| `-port`            | Server port                              | Auto-detect               |
| `-p`               | Web UI password                          | None                      |
| `-api-auth`        | Enable API key authentication            | `false`                   |
| `-tls-cert`        | Path to TLS certificate                  | None                      |
| `-tls-key`         | Path to TLS private key                  | None                      |
| `-allowed-origins` | CORS allowed origins (comma-separated)   | None                      |
| `-db-path`         | SQLite database path or directory        | `~/.beamdrop/beamdrop.db` |
| `-rate-limit`      | Rate limit in req/min per IP (`0` = off) | `100`                     |
| `-log-level`       | `debug` / `info` / `warn` / `error`      | `info`                    |
| `-qr`              | Print QR code at startup                 | `false`                   |
| `-v`               | Print version and exit                   |                           |
| `-h`               | Print help and exit                      |                           |

### 2.3 Rate Limiting Tiers

Beamdrop applies three independent per-IP token-bucket tiers derived from the general rate (`-rate-limit`):

| Tier    | Endpoints                | Rate                           |
| ------- | ------------------------ | ------------------------------ |
| General | All other endpoints      | `BEAMDROP_RATE_LIMIT` req/min  |
| Auth    | `/auth/login`            | 5% of general (min 1) req/min  |
| Upload  | `/upload`, S3 PUT object | 10% of general (min 1) req/min |

### 2.4 TLS / HTTPS

**Self-signed certificate (development/testing):**

```bash
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes \
  -subj "/C=US/ST=State/L=City/O=Org/CN=localhost"

beamdrop -dir /data -tls-cert cert.pem -tls-key key.pem
```

**Let's Encrypt (production, via Caddy):**

```bash
export BEAMDROP_DOMAIN=files.example.com
# Uncomment caddy service in docker-compose.yml, then:
docker compose up -d
```

### 2.5 Production `.env` Example

```bash
# .env   Production configuration
BEAMDROP_PORT=7777
BEAMDROP_PASSWORD=<strong-random-password>
BEAMDROP_LOG_LEVEL=info
BEAMDROP_RATE_LIMIT=100
BEAMDROP_API_AUTH=true
BEAMDROP_QR=false
BEAMDROP_ALLOWED_ORIGINS=https://files.example.com
BEAMDROP_DB_PATH=/data/.beamdrop/beamdrop.db
# TLS is handled by Caddy   leave these blank if using a reverse proxy
BEAMDROP_TLS_CERT=
BEAMDROP_TLS_KEY=
```

---

## 3. Backup and Restore

Beamdrop stores state in two locations:

| Location                                                    | Contents                                        | Must Back Up? |
| ----------------------------------------------------------- | ----------------------------------------------- | ------------- |
| `<sharedDir>/`                                              | Uploaded files, buckets                         | **Yes**       |
| `<sharedDir>/.beamdrop/beamdrop.db` (or `BEAMDROP_DB_PATH`) | API keys, shareable links, starred files, stats | **Yes**       |
| `<sharedDir>/.beamdrop/beamdrop.log`                        | Structured JSON logs                            | Optional      |
| `<sharedDir>/.beamdrop_trash/`                              | Deleted files (recoverable)                     | Optional      |

### 3.1 Manual Backup

```bash
# Stop the service first for a consistent snapshot (or use SQLite hot backup below)
sudo systemctl stop beamdrop

# Create a timestamped archive
BACKUP_DIR=/backups/beamdrop
DATE=$(date +%Y%m%d_%H%M%S)
mkdir -p "$BACKUP_DIR"

tar -czf "$BACKUP_DIR/beamdrop-data-$DATE.tar.gz" /var/lib/beamdrop/

sudo systemctl start beamdrop
echo "Backup saved: $BACKUP_DIR/beamdrop-data-$DATE.tar.gz"
```

### 3.2 Live Database Backup (SQLite No Downtime)

The SQLite `.backup` command creates a consistent copy while the server is running:

```bash
sqlite3 /var/lib/beamdrop/.beamdrop/beamdrop.db \
  ".backup /backups/beamdrop/beamdrop-$(date +%Y%m%d_%H%M%S).db"
```

### 3.3 Automated Daily Backup (cron)

```bash
sudo tee /usr/local/bin/beamdrop-backup.sh > /dev/null <<'SCRIPT'
#!/usr/bin/env bash
set -euo pipefail

DATA_DIR=/var/lib/beamdrop
DB_PATH="$DATA_DIR/.beamdrop/beamdrop.db"
BACKUP_DIR=/backups/beamdrop
KEEP_DAYS=14

mkdir -p "$BACKUP_DIR"
DATE=$(date +%Y%m%d_%H%M%S)

# 1. Live database backup (no downtime)
sqlite3 "$DB_PATH" ".backup $BACKUP_DIR/beamdrop-db-$DATE.db"

# 2. Archive all uploaded files
tar -czf "$BACKUP_DIR/beamdrop-files-$DATE.tar.gz" \
  --exclude="$DATA_DIR/.beamdrop" \
  "$DATA_DIR/"

# 3. Remove backups older than KEEP_DAYS
find "$BACKUP_DIR" -name "beamdrop-*.tar.gz" -mtime +"$KEEP_DAYS" -delete
find "$BACKUP_DIR" -name "beamdrop-*.db"     -mtime +"$KEEP_DAYS" -delete

echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) Backup complete: $BACKUP_DIR"
SCRIPT

sudo chmod +x /usr/local/bin/beamdrop-backup.sh

# Run daily at 2 AM
(crontab -l 2>/dev/null; echo "0 2 * * * /usr/local/bin/beamdrop-backup.sh >> /var/log/beamdrop-backup.log 2>&1") \
  | crontab -
```

### 3.4 Docker Volume Backup

```bash
# Stop the container first
docker compose down

# Archive the volume-mapped data directory
tar -czf "beamdrop-data-$(date +%Y%m%d_%H%M%S).tar.gz" ./data/

# Restart
docker compose up -d
```

For zero-downtime database backup while the container is running:

```bash
docker exec beamdrop sqlite3 /data/.beamdrop/beamdrop.db \
  ".backup /data/.beamdrop/beamdrop-backup.db"
docker cp beamdrop:/data/.beamdrop/beamdrop-backup.db \
  "./beamdrop-db-$(date +%Y%m%d_%H%M%S).db"
```

### 3.5 Restore

**Full restore from archive:**

```bash
sudo systemctl stop beamdrop

# Restore data directory (adjust path to your backup file)
sudo tar -xzf /backups/beamdrop/beamdrop-files-20260101_020000.tar.gz \
  -C /

# Restore database
cp /backups/beamdrop/beamdrop-db-20260101_020000.db \
   /var/lib/beamdrop/.beamdrop/beamdrop.db

sudo chown -R beamdrop:beamdrop /var/lib/beamdrop
sudo systemctl start beamdrop
```

**Database-only restore:**

```bash
sudo systemctl stop beamdrop
cp /backups/beamdrop/beamdrop-db-<TIMESTAMP>.db \
   /var/lib/beamdrop/.beamdrop/beamdrop.db
sudo chown beamdrop:beamdrop /var/lib/beamdrop/.beamdrop/beamdrop.db
sudo systemctl start beamdrop
```

---

## 4. Monitoring Setup

### 4.1 Health Endpoints

Beamdrop provides Kubernetes-compatible health probes. These require no authentication.

| Endpoint              | Purpose                               | Use Case                  |
| --------------------- | ------------------------------------- | ------------------------- |
| `GET /health`         | Full health overview (all components) | Manual checks, dashboards |
| `GET /health/live`    | Liveness process is running (no I/O)  | K8s `livenessProbe`       |
| `GET /health/ready`   | Readiness DB + storage accessible     | K8s `readinessProbe`      |
| `GET /health/startup` | Startup complete                      | K8s `startupProbe`        |

**Example response (`/health`):**

```json
{
  "status": "healthy",
  "service": "beamdrop",
  "version": "1.0.0",
  "timestamp": "2026-01-01T00:00:00Z",
  "components": {
    "process": { "status": "ok", "message": "running" },
    "startup": { "status": "ok", "message": "initialisation complete" },
    "database": { "status": "ok", "message": "connected", "latency": "1.23ms" },
    "storage": { "status": "ok", "message": "writable" },
    "runtime": { "status": "ok", "message": "goroutines: 12" }
  }
}
```

**Quick health check script:**

```bash
#!/usr/bin/env bash
STATUS=$(curl -sf http://localhost:7777/health/live && echo "UP" || echo "DOWN")
echo "Beamdrop: $STATUS"
```

### 4.2 Prometheus Metrics

Beamdrop exposes a `/metrics` endpoint in Prometheus text format (no authentication required).

**Key metrics:**

| Metric                              | Type      | Description                           |
| ----------------------------------- | --------- | ------------------------------------- |
| `beamdrop_requests_total`           | counter   | HTTP requests by method, path, status |
| `beamdrop_request_duration_seconds` | histogram | Request latency (p50/p95/p99)         |
| `beamdrop_auth_failures_total`      | counter   | Auth failures by reason               |
| `beamdrop_uploads_total`            | counter   | Completed uploads                     |
| `beamdrop_downloads_total`          | counter   | Completed downloads                   |
| `beamdrop_upload_size_bytes`        | histogram | Upload file sizes                     |
| `beamdrop_storage_bytes`            | gauge     | Bytes used by stored files            |
| `beamdrop_objects_total`            | gauge     | Number of stored files                |
| `beamdrop_active_connections`       | gauge     | In-flight HTTP requests               |
| `beamdrop_storage_free_bytes`       | gauge     | Free disk space                       |
| `beamdrop_storage_total_bytes`      | gauge     | Total disk capacity                   |
| `beamdrop_goroutines_count`         | gauge     | Go goroutine count                    |

**Add Beamdrop as a Prometheus scrape target:**

```yaml
# prometheus.yml
scrape_configs:
  - job_name: beamdrop
    static_configs:
      - targets: ["localhost:7777"]
    # Optional basic auth if behind a proxy that requires it
    # basic_auth:
    #   username: prometheus
    #   password: secret
```

### 4.3 Grafana Dashboard

A pre-built Grafana dashboard is included at [`docs/grafana-dashboard.json`](grafana-dashboard.json).

**Import steps:**

1. Open Grafana → **Dashboards → Import**
2. Click **Upload JSON file** and select `docs/grafana-dashboard.json`
3. Select your Prometheus data source
4. Click **Import**

### 4.4 Kubernetes Probes

```yaml
livenessProbe:
  httpGet:
    path: /health/live
    port: 7777
  initialDelaySeconds: 5
  periodSeconds: 10
  failureThreshold: 3

readinessProbe:
  httpGet:
    path: /health/ready
    port: 7777
  initialDelaySeconds: 10
  periodSeconds: 15
  failureThreshold: 3

startupProbe:
  httpGet:
    path: /health/startup
    port: 7777
  failureThreshold: 30
  periodSeconds: 2
```

### 4.5 Alerting Rules (Prometheus)

```yaml
# beamdrop-alerts.yml
groups:
  - name: beamdrop
    rules:
      - alert: BeamdropDown
        expr: up{job="beamdrop"} == 0
        for: 1m
        labels:
          severity: critical
        annotations:
          summary: "Beamdrop instance is down"

      - alert: BeamdropHighErrorRate
        expr: |
          rate(beamdrop_requests_total{status=~"5.."}[5m])
            / rate(beamdrop_requests_total[5m]) > 0.05
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Beamdrop error rate > 5%"

      - alert: BeamdropDiskAlmostFull
        expr: |
          beamdrop_storage_free_bytes / beamdrop_storage_total_bytes < 0.10
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Beamdrop storage less than 10% free"

      - alert: BeamdropHighLatency
        expr: |
          histogram_quantile(0.95,
            rate(beamdrop_request_duration_seconds_bucket[5m])
          ) > 2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Beamdrop p95 latency > 2s"
```

### 4.6 Log Monitoring

Beamdrop writes structured JSON logs to `<sharedDir>/.beamdrop/beamdrop.log`.

**Follow logs (systemd):**

```bash
journalctl -u beamdrop -f
```

**Follow logs (Docker):**

```bash
docker compose logs -f beamdrop
```

**Query logs with `jq`:**

```bash
# Show errors only
tail -f /var/lib/beamdrop/.beamdrop/beamdrop.log | jq 'select(.level=="ERROR")'

# Count requests by status in the last hour
jq -r 'select(.msg | test("request")) | .status' \
  /var/lib/beamdrop/.beamdrop/beamdrop.log | sort | uniq -c
```

**Shipping logs to a centralised system (e.g., Loki via Promtail):**

```yaml
# promtail-config.yaml
scrape_configs:
  - job_name: beamdrop
    static_configs:
      - targets: [localhost]
        labels:
          job: beamdrop
          __path__: /var/lib/beamdrop/.beamdrop/beamdrop.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            msg: msg
      - labels:
          level:
```

---

## 5. Scaling Guidance

Beamdrop is a single-process server backed by an embedded SQLite database. The following guidance will help you maximise capacity within that architecture.

### 5.1 Vertical Scaling

The simplest way to handle more load. Beamdrop's Go runtime makes good use of multiple CPU cores for concurrent request handling.

| Resource | Recommendation                                                 |
| -------- | -------------------------------------------------------------- |
| CPU      | 2+ cores for moderate traffic; 4+ cores for high concurrency   |
| RAM      | 512 MB minimum; 2 GB+ for large file transfers                 |
| Disk I/O | Use SSD/NVMe; IOPS matter more than throughput for small files |
| Network  | 1 Gbps for general use; 10 Gbps for sustained large uploads    |

**Monitor goroutine count** (`beamdrop_goroutines_count`) a continuously rising count may indicate a leak; a spike under load is normal.

### 5.2 Storage Scaling

```bash
# Add a larger disk and move the data directory
sudo systemctl stop beamdrop
sudo rsync -a /var/lib/beamdrop/ /mnt/new-disk/beamdrop/
# Update ExecStart -dir flag or BEAMDROP_DB_PATH, then:
sudo systemctl start beamdrop
```

**Recommended filesystem:**

- `ext4` or `xfs` on Linux both handle large directories well
- Enable `noatime` mount option to reduce write overhead:

  ```
  # /etc/fstab
  /dev/sdb1  /var/lib/beamdrop  ext4  defaults,noatime  0 2
  ```

### 5.3 Reverse Proxy Tuning

When Beamdrop is behind Nginx or Caddy, tune the proxy for large files:

**Nginx:**

```nginx
# Increase for large file uploads
client_max_body_size 10G;
proxy_read_timeout   600s;
proxy_send_timeout   600s;
proxy_request_buffering off;   # stream uploads directly to backend
```

**Caddy:**

```caddy
{$BEAMDROP_DOMAIN} {
    request_body {
        max_size 10GB
    }
    reverse_proxy beamdrop:7777 {
        transport http {
            read_timeout  600s
            write_timeout 600s
        }
    }
}
```

### 5.4 Rate Limit Tuning

Adjust the rate limit to balance protection against abuse vs. legitimate user throughput. As a starting point:

| Traffic Level             | `BEAMDROP_RATE_LIMIT` |
| ------------------------- | --------------------- |
| Personal / low            | `50`                  |
| Team (< 20 users)         | `100` (default)       |
| Departmental              | `300`                 |
| CI/CD heavy usage         | `500–1000`            |
| Disable (trusted network) | `0`                   |

### 5.5 Read-Only Replicas (Advanced)

SQLite supports multiple readers. If your workload is read-heavy (many downloads), you can serve reads from a replica by periodically copying the `.db` file:

```bash
# On a read replica node   refresh every 60 seconds
while true; do
  rsync -a beamdrop-primary:/var/lib/beamdrop/.beamdrop/beamdrop.db \
            /var/lib/beamdrop/.beamdrop/beamdrop.db
  sleep 60
done
```

> **Note:** Writes must still go to the primary. A load balancer (e.g., HAProxy) should route mutating requests (`POST`, `PUT`, `DELETE`) to the primary and `GET` requests to replicas.

### 5.6 Multi-Instance (Stateless Files, Shared DB)

For high-availability deployments, run multiple Beamdrop instances pointing to a shared NFS/CIFS volume:

```bash
# On each node
beamdrop \
  -dir /mnt/nfs/beamdrop-data \
  -db-path /mnt/nfs/beamdrop-data/.beamdrop/beamdrop.db \
  -port 7777
```

Place a load balancer (Nginx upstream, Caddy, HAProxy) in front:

```nginx
upstream beamdrop {
    least_conn;
    server node1:7777;
    server node2:7777;
    server node3:7777;
    keepalive 32;
}
```

> **Important:** SQLite in WAL mode handles concurrent writers from multiple processes, but with NFS you must test carefully. For heavy write loads, consider migrating to an external database via a future release.

---

## 6. Common Troubleshooting

### 6.1 Service Will Not Start

**Symptom:** `systemctl status beamdrop` shows `Active: failed`

**Steps:**

```bash
# 1. Check the full error message
journalctl -u beamdrop -n 50 --no-pager

# 2. Verify the binary is executable
ls -la /usr/local/bin/beamdrop

# 3. Test the binary manually as the service user
sudo -u beamdrop /usr/local/bin/beamdrop -dir /var/lib/beamdrop -v

# 4. Check file/directory permissions
ls -la /var/lib/beamdrop
sudo chown -R beamdrop:beamdrop /var/lib/beamdrop

# 5. Check if the port is already in use
sudo ss -tlnp | grep 7777
```

---

### 6.2 Port Conflict

**Symptom:** `listen tcp :7777: bind: address already in use`

```bash
# Find and stop the conflicting process
sudo ss -tlnp | grep 7777
sudo kill <PID>

# Or use a different port
export BEAMDROP_PORT=8888
```

---

### 6.3 Web UI Login Fails

**Symptom:** Correct password rejected; browser shows 401.

```bash
# 1. Confirm password is set in environment file
grep BEAMDROP_PASSWORD /etc/beamdrop/beamdrop.env

# 2. Restart to reload the environment
sudo systemctl restart beamdrop

# 3. Check auth failure metrics
curl -s http://localhost:7777/metrics | grep auth_failures
```

---

### 6.4 API Key Authentication Errors

**Symptom:** `403 Forbidden` or `401 Unauthorized` on S3 API calls.

```bash
# 1. Verify API auth is enabled
curl -s http://localhost:7777/health | jq .

# 2. Check that the access key and secret match what was generated
#    (the secret is shown only once at key creation)

# 3. Verify the clock skew   the HMAC timestamp must be within 15 minutes
date -u

# 4. Regenerate the signature with the correct timestamp format: 2006-01-02T15:04:05Z
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
echo "Timestamp: $TIMESTAMP"
```

---

### 6.5 File Upload Fails

**Symptom:** Upload errors, timeouts, or "413 Request Entity Too Large".

```bash
# 1. Check available disk space
df -h /var/lib/beamdrop

# 2. Verify directory is writable by the service user
sudo -u beamdrop touch /var/lib/beamdrop/test-write && \
  sudo -u beamdrop rm /var/lib/beamdrop/test-write && \
  echo "Writable"

# 3. If behind Nginx, increase client_max_body_size
sudo nano /etc/nginx/sites-available/beamdrop
# Add/update: client_max_body_size 10G;
sudo nginx -t && sudo systemctl reload nginx

# 4. Check upload rate limit
curl -s http://localhost:7777/metrics | grep rate_limit
```

---

### 6.6 Disk Full

**Symptom:** HTTP 500 on uploads; `no space left on device` in logs; or HTTP 507 `STORAGE_FULL` when `-max-storage` limit is reached.

```bash
# 1. Check disk usage
df -h /var/lib/beamdrop
du -sh /var/lib/beamdrop/*

# 2. Check if max-storage limit is set
grep MAX_STORAGE /etc/beamdrop/beamdrop.env

# 3. Increase the limit or set to 0 (unlimited)
BEAMDROP_MAX_STORAGE=10GB   # in /etc/beamdrop/beamdrop.env
sudo systemctl restart beamdrop

# 4. Recover space from trash (permanently delete trashed files)
sudo rm -rf /var/lib/beamdrop/.beamdrop_trash/*

# 3. Compress or archive old files in buckets
du -sh /var/lib/beamdrop/buckets/* | sort -hr | head -20

# 4. Expand the volume (cloud provider) or attach additional storage
```

---

### 6.7 Rate Limiting Issues

**Symptom:** Legitimate users receive `429 Too Many Requests`.

```bash
# 1. Check current limit
beamdrop -v && grep RATE_LIMIT /etc/beamdrop/beamdrop.env

# 2. Increase the limit or disable it
BEAMDROP_RATE_LIMIT=500   # in /etc/beamdrop/beamdrop.env
sudo systemctl restart beamdrop

# 3. Verify the rate limit tier breakdown
curl -s http://localhost:7777/metrics | grep rate
```

---

### 6.8 Database Errors

**Symptom:** `database is locked`, `unable to open database file`, or intermittent 500 errors.

```bash
# 1. Check that only one Beamdrop process is running
pgrep -a beamdrop

# 2. Verify the database file exists and is readable
ls -la /var/lib/beamdrop/.beamdrop/beamdrop.db

# 3. Check SQLite integrity
sqlite3 /var/lib/beamdrop/.beamdrop/beamdrop.db "PRAGMA integrity_check;"

# 4. If corrupt   restore from latest backup (see Section 3.5)

# 5. Enable WAL mode for better concurrent access
sqlite3 /var/lib/beamdrop/.beamdrop/beamdrop.db "PRAGMA journal_mode=WAL;"
```

---

### 6.9 TLS Certificate Issues

**Symptom:** `certificate signed by unknown authority`, `certificate has expired`, or HTTPS not working.

```bash
# 1. Check certificate expiry
openssl x509 -in /etc/beamdrop/cert.pem -noout -dates

# 2. Verify the key matches the certificate
openssl x509 -in /etc/beamdrop/cert.pem -noout -modulus | md5sum
openssl rsa  -in /etc/beamdrop/key.pem  -noout -modulus | md5sum
# Both MD5 hashes must match

# 3. Renew Let's Encrypt certificate (Certbot)
sudo certbot renew --nginx
sudo systemctl reload nginx

# 4. If using Caddy, certificates are renewed automatically
docker compose logs caddy | grep -i cert
```

---

### 6.10 Health Check Failing

**Symptom:** Load balancer marks the instance as unhealthy; `/health/ready` returns 503.

```bash
# 1. Check all components
curl -s http://localhost:7777/health | jq .

# 2. Database component unhealthy
sqlite3 /var/lib/beamdrop/.beamdrop/beamdrop.db "SELECT 1;"

# 3. Storage component unhealthy   check directory permissions
ls -la /var/lib/beamdrop/

# 4. Restart the service
sudo systemctl restart beamdrop

# 5. Check startup probe if the container was just launched
curl -s http://localhost:7777/health/startup | jq .
```

---

### 6.11 Logs Are Not Appearing

**Symptom:** Log file is empty or missing.

```bash
# Default log file location
ls -la /var/lib/beamdrop/.beamdrop/beamdrop.log

# Verify log level is not set too high
grep LOG_LEVEL /etc/beamdrop/beamdrop.env

# Tail the systemd journal as a fallback
journalctl -u beamdrop -f

# Docker logs
docker compose logs -f beamdrop
```

---

## Quick Reference

### Service Commands

```bash
# systemd
sudo systemctl start   beamdrop
sudo systemctl stop    beamdrop
sudo systemctl restart beamdrop
sudo systemctl status  beamdrop
journalctl -u beamdrop -f

# Docker Compose
docker compose up -d
docker compose down
docker compose restart beamdrop
docker compose logs -f beamdrop
```

### Health Check

```bash
curl http://localhost:7777/health/live    # liveness
curl http://localhost:7777/health/ready   # readiness
curl http://localhost:7777/health         # full overview
curl http://localhost:7777/metrics        # Prometheus metrics
```

### Backup

```bash
# Live database backup
sqlite3 /var/lib/beamdrop/.beamdrop/beamdrop.db \
  ".backup /backups/beamdrop/beamdrop-$(date +%Y%m%d).db"

# Full archive
tar -czf /backups/beamdrop/beamdrop-$(date +%Y%m%d).tar.gz /var/lib/beamdrop/
```

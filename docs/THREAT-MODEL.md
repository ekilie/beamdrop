# Security Threat Model

This document describes the security threat model for Beamdrop. It complements the [Security Features](SECURITY.md) guide by identifying what we protect, who may attack the system, and how attacks are mitigated.

---

## 1. Assets to Protect

| Asset                             | Sensitivity | Storage Location                                              | Impact if Compromised                                       |
| --------------------------------- | ----------- | ------------------------------------------------------------- | ----------------------------------------------------------- |
| **User files**                    | High        | `<dir>/` and `<dir>/buckets/` on disk                         | Data breach, data loss, unauthorized disclosure             |
| **Password hash**                 | High        | SQLite DB (`<dir>/.beamdrop/beamdrop.db`, `Config` table)     | Full system access if cracked                               |
| **JWT signing secret**            | Critical    | In-process memory (generated at startup)                      | Token forgery, full authentication bypass                   |
| **API keys & secrets**            | High        | SQLite DB (AES-256-GCM encrypted secrets in `APIKey` table) | Unauthorized API access, data exfiltration                  |
| **Shareable link tokens**         | Medium      | SQLite DB (`ShareableLink` table)                             | Unauthorized file access for specific files                 |
| **Presigned URLs**                | Medium      | SQLite DB (`PresignedURL` table) and client-side HMAC URLs    | Time-limited unauthorized file access                       |
| **Session tokens (JWT)**          | High        | Client cookies (`beamdrop_token`) and `Authorization` headers | Session hijacking, impersonation                            |
| **SQLite database**               | High        | `<dir>/.beamdrop/beamdrop.db`                                 | Full metadata leak, credential theft, audit trail tampering |
| **Application logs**              | Medium      | `<dir>/.beamdrop/beamdrop.log`                                | Information disclosure, audit trail tampering               |
| **TLS private key**               | Critical    | File system (operator-provided path)                          | Traffic decryption, man-in-the-middle attacks               |
| **Server binary & configuration** | Medium      | Host file system                                              | Backdoor injection, configuration tampering                 |

---

## 2. Trust Boundaries

```
┌─────────────────────────────────────────────────────────────────────┐
│                        INTERNET (Untrusted)                         │
│                                                                     │
│   ┌───────────┐   ┌───────────┐   ┌───────────┐                    │
│   │  Browser   │   │ S3 Client │   │   Bot /   │                    │
│   │  (Web UI)  │   │ (API)     │   │  Scanner  │                    │
│   └─────┬─────┘   └─────┬─────┘   └─────┬─────┘                    │
│         │               │               │                           │
└─────────┼───────────────┼───────────────┼───────────────────────────┘
          │               │               │
══════════╪═══════════════╪═══════════════╪═══ BOUNDARY 1: Network ════
          │               │               │
┌─────────┼───────────────┼───────────────┼───────────────────────────┐
│         ▼               ▼               ▼                           │
│  ┌─────────────────────────────────────────────┐                    │
│  │          Reverse Proxy (Caddy/nginx)         │  ◄── TLS          │
│  │          [optional   terminates TLS]         │      termination  │
│  └──────────────────────┬──────────────────────┘                    │
│                         │                                           │
│  ═══════════════════════╪══════ BOUNDARY 2: Application ════════    │
│                         ▼                                           │
│  ┌─────────────────────────────────────────────┐                    │
│  │              Beamdrop Server                 │                    │
│  │  ┌────────────────────────────────────────┐  │                    │
│  │  │        Middleware Chain                 │  │                    │
│  │  │  ┌──────────┐ ┌──────────┐ ┌────────┐ │  │                    │
│  │  │  │Rate Limit│→│  CORS    │→│Security│ │  │                    │
│  │  │  │ (per-IP) │ │ Headers  │ │Headers │ │  │                    │
│  │  │  └──────────┘ └──────────┘ └────────┘ │  │                    │
│  │  │  ┌──────────┐ ┌──────────┐            │  │                    │
│  │  │  │  Auth     │→│ Metrics  │            │  │                    │
│  │  │  │Middleware │ │Collector │            │  │                    │
│  │  │  └──────────┘ └──────────┘            │  │                    │
│  │  └────────────────────────────────────────┘  │                    │
│  │                                              │                    │
│  │  ┌──────────────────┐  ┌──────────────────┐  │                    │
│  │  │   Route Handlers  │  │  WebSocket Stats │  │                    │
│  │  │  (upload, download│  │  (real-time)     │  │                    │
│  │  │   file ops, auth) │  │                  │  │                    │
│  │  └────────┬─────────┘  └──────────────────┘  │                    │
│  └───────────┼──────────────────────────────────┘                    │
│              │                                                       │
│  ════════════╪══════════════ BOUNDARY 3: Data ══════════════════     │
│              ▼                                                       │
│  ┌──────────────────┐  ┌──────────────────────┐                      │
│  │   SQLite Database │  │   File System        │                      │
│  │  (credentials,    │  │  (user files,        │                      │
│  │   API keys,       │  │   buckets, logs,     │                      │
│  │   metadata)       │  │   temp uploads)      │                      │
│  └──────────────────┘  └──────────────────────┘                      │
│                                                                     │
│                          HOST (Trusted)                              │
└─────────────────────────────────────────────────────────────────────┘
```

**Boundary 1 Network**: Separates untrusted internet traffic from the host. TLS terminates here (either at a reverse proxy or at the Beamdrop server itself).

**Boundary 2 Application**: Separates inbound HTTP requests from application logic. The middleware chain enforces rate limiting, authentication, CORS, and security headers before any handler executes.

**Boundary 3 Data**: Separates application logic from persistent storage. All file I/O and database queries flow through the storage and database packages; handlers never access the file system directly.

---

## 3. Threat Actors

| Actor                                 | Motivation                              | Capability                                       | Examples                                                       |
| ------------------------------------- | --------------------------------------- | ------------------------------------------------ | -------------------------------------------------------------- |
| **Unauthenticated external attacker** | Data theft, disruption, ransomware      | Network access, automated tools, public exploits | Port scanners, credential stuffers, opportunistic attackers    |
| **Authenticated malicious user**      | Data exfiltration, privilege escalation | Valid JWT or API key, knowledge of the API       | Disgruntled team member, compromised workstation               |
| **Automated bot / scanner**           | Vulnerability discovery, brute-force    | High request volume, exploit databases           | Shodan crawlers, credential-stuffing botnets                   |
| **Network-level attacker (MitM)**     | Eavesdropping, session hijacking        | Traffic interception on the same network         | Rogue Wi-Fi, ARP spoofing, compromised router                  |
| **Local / insider attacker**          | Direct data access, persistence         | Shell access to the host                         | Compromised container, malicious operator, supply chain attack |

---

## 4. Attack Vectors and Mitigations

| #   | Attack Vector                         | Threat Actor            | Asset at Risk                    | Existing Mitigation                                                                          | Residual Risk / Recommendation                                                         |
| --- | ------------------------------------- | ----------------------- | -------------------------------- | -------------------------------------------------------------------------------------------- | -------------------------------------------------------------------------------------- |
| 1   | **Brute-force password**              | External, Bot           | Password hash, files             | Auth rate limit (5 req/min per IP), bcrypt (cost 10)                                         | Low consider account lockout after N failures                                          |
| 2   | **JWT token theft (XSS)**             | External                | Session token                    | CSP header (no `unsafe-eval`, restricted `script-src 'self'`), `httpOnly` cookies, `X-Content-Type-Options: nosniff`, no `localStorage` token storage, `Permissions-Policy` | Negligible — tokens never exposed to JavaScript                            |
| 3   | **API key leakage**                   | Authenticated, Insider  | API key secrets                  | Keys encrypted with AES-256-GCM at rest; secrets shown once at creation; encryption key rotates on restart | Low rotate keys periodically; monitor for leaked keys                                  |
| 4   | **Path traversal (file access)**      | External, Authenticated | User files                       | Go `filepath.Clean`, storage layer validates paths against root dir                          | Low continuously fuzz file operation endpoints                                         |
| 5   | **Denial of service (request flood)** | Bot, External           | Service availability             | Per-IP token-bucket rate limiting (3 tiers), stale-client eviction                           | Medium distributed attacks bypass per-IP limits; consider a WAF for public deployments |
| 6   | **Upload flooding (disk exhaustion)** | Bot, Authenticated      | Disk, availability               | Upload rate limit (10 req/min per IP), temp dir for in-flight uploads                        | Medium add configurable max file size and disk quota                                   |
| 7   | **SQL injection**                     | External                | SQLite database                  | GORM parameterized queries (no raw SQL interpolation)                                        | Low keep GORM updated; audit any raw queries                                           |
| 8   | **Cross-site request forgery (CSRF)** | External                | Authenticated sessions           | CORS disabled by default; `SameSite=Strict` cookies; double-submit cookie CSRF protection (`beamdrop_csrf` cookie + `X-CSRF-Token` header validation); global fetch interceptor on frontend | Negligible — all state-changing requests require CSRF token                             |
| 9   | **Man-in-the-middle**                 | MitM                    | Data in transit, tokens          | TLS support, HSTS header (when TLS enabled)                                                  | Low when TLS is used enforce TLS in production                                         |
| 10  | **Clickjacking**                      | External                | Web UI sessions                  | `X-Frame-Options: DENY`                                                                      | Negligible                                                                             |
| 11  | **Shareable link enumeration**        | External, Bot           | Shared files                     | Links use cryptographic random tokens; rate limiting                                         | Low tokens are 128-bit random; enumeration is impractical                              |
| 12  | **Presigned URL abuse**               | External                | Specific files                   | 15-min timestamp window (HMAC URLs), configurable expiry, download limits (server-side URLs) | Low use short expiry times; prefer server-side presigned URLs                          |
| 13  | **Log injection**                     | External                | Log integrity                    | Structured JSON logging (slog) escapes values                                                | Negligible                                                                             |
| 14  | **Database file theft**               | Insider, Local          | All metadata, hashed credentials | File permissions set by OS; Docker uses non-root user                                        | Medium encrypt the database at rest for sensitive deployments                          |
| 15  | **Container escape**                  | Insider, Local          | Host system                      | Alpine minimal image, non-root user, no unnecessary capabilities                             | Low add `read_only` filesystem and drop all capabilities in Docker Compose             |
| 16  | **Dependency supply-chain**           | External                | Server binary                    | Go module checksums (`go.sum`), pinned versions                                              | Low enable Dependabot or Renovate for automated updates                                |
| 17  | **IP spoofing (rate limit bypass)**   | External, Bot           | Rate limiter effectiveness       | `X-Forwarded-For` / `X-Real-IP` headers only trusted from configured proxy CIDRs (`--trusted-proxies`); defaults to remote address when no trusted proxy is configured | Low when trusted proxies are configured correctly                                      |

---

## 5. Security Controls

### 5.1 Authentication & Authorization

| Control                 | Implementation                                           | Reference                 |
| ----------------------- | -------------------------------------------------------- | ------------------------- |
| Password hashing        | bcrypt with `DefaultCost` (10)                           | `pkg/auth/password.go`    |
| JWT tokens              | HS256, 24-hour expiry, random 256-bit secret per process, unique JTI per token | `pkg/auth/password.go`    |
| JWT revocation          | In-memory JTI blocklist; tokens revoked on logout; background cleanup every 10 min | `pkg/auth/password.go`    |
| API key authentication  | HMAC-SHA256 request signing, 15-min timestamp window     | `pkg/crypto/signature.go` |
| API key storage         | AES-256-GCM encrypted secrets in database                | `pkg/db/api_keys.go`     |
| CSRF protection         | Double-submit cookie (`beamdrop_csrf` + `X-CSRF-Token` header) | `pkg/middleware/csrf.go`  |
| Shareable link passwords| bcrypt (cost 10) with SHA-256 backward compatibility     | `pkg/db/shareable_links.go` |
| Public route whitelist  | Health, metrics, and login endpoints bypass auth         | `pkg/auth/middleware.go`  |
| HTTP method enforcement | Each route restricts allowed methods; 405 on violation   | `beam/server/routes.go`   |

### 5.2 Transport Security

| Control           | Implementation                                                              | Reference                    |
| ----------------- | --------------------------------------------------------------------------- | ---------------------------- |
| TLS/HTTPS         | Optional cert/key via CLI flags                                             | `cmd/beam/main.go`           |
| HSTS              | `Strict-Transport-Security: max-age=31536000; includeSubDomains` (TLS only) | `pkg/middleware/security.go` |
| Reverse proxy TLS | Caddy auto-HTTPS in Docker Compose                                          | `docker-compose.yml`         |

### 5.3 Input Validation & Output Encoding

| Control                   | Implementation                                            | Reference                    |
| ------------------------- | --------------------------------------------------------- | ---------------------------- |
| Parameterized queries     | GORM ORM no raw SQL interpolation                         | All `pkg/db/` files          |
| Path traversal prevention | `filepath.Clean` + root directory validation              | `pkg/storage/`               |
| CSP header                | Restricts script, style, image, font, and connect sources; no `unsafe-eval` | `pkg/middleware/security.go` |
| `X-Content-Type-Options`  | `nosniff` prevents MIME-type confusion                    | `pkg/middleware/security.go` |
| `Permissions-Policy`      | Disables geolocation, microphone, camera                  | `pkg/middleware/security.go` |

### 5.4 Rate Limiting & Availability

| Control                   | Implementation                                                       | Reference                     |
| ------------------------- | -------------------------------------------------------------------- | ----------------------------- |
| Token-bucket rate limiter | Per-IP, 3 tiers (general, auth, upload); trusted proxy support | `pkg/middleware/ratelimit.go` |
| Stale-client eviction     | Clients unseen for 10+ min purged every 5 min                        | `pkg/middleware/ratelimit.go` |
| Graceful shutdown         | Configurable timeout for in-flight requests                          | `cmd/beam/main.go`            |
| Health probes             | `/health/live`, `/health/ready`, `/health/startup` for orchestrators | `beam/server/routes.go`       |

### 5.5 Logging & Monitoring

| Control            | Implementation                                         | Reference                     |
| ------------------ | ------------------------------------------------------ | ----------------------------- |
| Structured logging | `slog` with JSON file output + colored terminal output | `pkg/logger/`                 |
| Request IDs        | `X-Request-ID` header on every request                 | `pkg/middleware/reqctx/`      |
| Prometheus metrics | Request counts, durations, error rates at `/metrics`   | `pkg/metrics/`                |
| Grafana dashboard  | Pre-built dashboard for visualizing metrics            | `docs/grafana-dashboard.json` |

### 5.6 Deployment Hardening

| Control            | Implementation                          | Reference    |
| ------------------ | --------------------------------------- | ------------ |
| Non-root container | `beamdrop` user in Docker image         | `Dockerfile` |
| Minimal base image | Alpine 3.21 (~39 MB total)              | `Dockerfile` |
| Multi-stage build  | Build tools excluded from runtime image | `Dockerfile` |
| Static binary      | No CGO, no shared library dependencies  | `Makefile`   |
| Health checks      | Docker `HEALTHCHECK` instruction        | `Dockerfile` |

---

## 6. Incident Response Outline

### 6.1 Detection

| Signal                      | Source                             | Example                     |
| --------------------------- | ---------------------------------- | --------------------------- |
| Spike in 401/403 responses  | Prometheus metrics, Grafana alerts | Brute-force attempt         |
| Spike in 429 responses      | Rate limiter logs                  | Bot or DDoS activity        |
| Unusual download volume     | Request logs, Prometheus           | Data exfiltration           |
| Unexpected API key creation | Database audit / application logs  | Insider threat              |
| TLS certificate errors      | Reverse proxy logs                 | MitM or expired certificate |

### 6.2 Containment

1. **Isolate the server** remove from the network or block the offending IP(s) at the firewall/reverse proxy.
2. **Revoke compromised credentials** delete affected API keys, rotate the server password, and restart the server (which regenerates the JWT secret, invalidating all sessions).
3. **Disable public access** if the attack is ongoing, stop the Beamdrop process or restrict access to trusted IPs only.
4. **Preserve evidence** copy `<dir>/.beamdrop/beamdrop.log` and the SQLite database before any remediation that could overwrite audit data.

### 6.3 Eradication

1. **Identify the root cause** review structured logs filtered by the suspect IP, time window, or request ID.
2. **Patch the vulnerability** update Beamdrop, dependencies, or configuration as needed.
3. **Scan for persistence** check for unexpected files in the shared directory and review database records for unauthorized API keys or shareable links.

### 6.4 Recovery

1. **Restore from backup** if files were tampered with, restore from the most recent known-good backup.
2. **Rotate all secrets** change the server password, regenerate API keys, and replace TLS certificates if compromise is suspected.
3. **Re-enable access** bring the server back online with the patched configuration and monitor closely.

### 6.5 Post-Incident

1. **Document the timeline** record what happened, when it was detected, and how it was resolved.
2. **Update the threat model** add any newly discovered attack vectors or threat actors to this document.
3. **Improve detection** add Grafana alerts or log-based alerts for the patterns observed during the incident.
4. **Communicate** notify affected users if any data was accessed or modified without authorization.

---

## Revision History

| Date       | Change                                                |
| ---------- | ----------------------------------------------------- |
| 2026-05-05 | Updated for Phase 1 & 2 security hardening            |
| 2026-03-07 | Initial threat model |

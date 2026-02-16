# ─────────────────────────────────────────────────────────────────────────────
# Stage 1: Build frontend with Bun
# ─────────────────────────────────────────────────────────────────────────────
FROM oven/bun:1 AS frontend

WORKDIR /frontend

# Install dependencies first (layer cache)
COPY static/frontend/package.json static/frontend/bun.lock* static/frontend/bun.lockb* ./
RUN bun install

# Copy source and build
COPY static/frontend/ ./
RUN bun run build

# ─────────────────────────────────────────────────────────────────────────────
# Stage 2: Build Go binary
# ─────────────────────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Build arguments for version injection
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

WORKDIR /src

# Install git (needed for go module resolution if any use git)
RUN apk add --no-cache git

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy full source
COPY . .

# Copy built frontend assets into the embed directory
COPY --from=frontend /frontend/dist ./static/frontend/dist

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w \
    -X 'github.com/tachRoutine/beamdrop-go/config.VERSION=${VERSION}' \
    -X 'github.com/tachRoutine/beamdrop-go/config.Commit=${COMMIT}' \
    -X 'github.com/tachRoutine/beamdrop-go/config.BuildDate=${BUILD_DATE}'" \
    -o /beamdrop ./cmd/beam

# ─────────────────────────────────────────────────────────────────────────────
# Stage 3: Minimal runtime image
# ─────────────────────────────────────────────────────────────────────────────
FROM alpine:3.21

LABEL org.opencontainers.image.title="BeamDrop" \
    org.opencontainers.image.description="Local-first file sharing with S3-compatible API" \
    org.opencontainers.image.source="https://github.com/ekilie/beamdrop" 

# Install ca-certificates for HTTPS and wget for HEALTHCHECK
RUN apk add --no-cache ca-certificates wget \
    && addgroup -S beamdrop \
    && adduser  -S -G beamdrop -h /data beamdrop

# Copy the static binary
COPY --from=builder /beamdrop /usr/local/bin/beamdrop

# Copy the entrypoint script (translates env vars → CLI flags)
COPY docker-entrypoint.sh /usr/local/bin/docker-entrypoint.sh
RUN chmod +x /usr/local/bin/docker-entrypoint.sh

# /data is the shared directory — mount a volume here for persistence.
# The SQLite DB, logs, and uploaded files all live under this path.
RUN mkdir -p /data && chown beamdrop:beamdrop /data
VOLUME /data

# Default port (beamdrop will also try 8080, 8888, … if 7777 is taken)
EXPOSE 7777

# Health check using the lightweight liveness probe
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD ["wget", "--no-verbose", "--tries=1", "--spider", "http://localhost:7777/health/live"]

USER beamdrop:beamdrop

ENTRYPOINT ["docker-entrypoint.sh"]
CMD []

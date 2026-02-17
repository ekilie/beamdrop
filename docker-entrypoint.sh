#!/bin/sh
# docker-entrypoint.sh — Translates BEAMDROP_* environment variables into
# CLI flags so that docker-compose.yml can configure BeamDrop purely through
# the `environment:` block.
#
# Any explicit arguments passed to the container (via `command:`) take
# precedence — this script only adds flags that are not already present.

set -e

args="--dir ${BEAMDROP_DIR:-/data}"

# Always disable QR in containers (no terminal to scan it)
args="$args --no-qr"

[ -n "$BEAMDROP_PORT" ]            && args="$args --port $BEAMDROP_PORT"
[ -n "$BEAMDROP_PASSWORD" ]        && args="$args -p $BEAMDROP_PASSWORD"
[ -n "$BEAMDROP_LOG_LEVEL" ]       && args="$args --log-level $BEAMDROP_LOG_LEVEL"
[ -n "$BEAMDROP_RATE_LIMIT" ]      && args="$args --rate-limit $BEAMDROP_RATE_LIMIT"
[ -n "$BEAMDROP_ALLOWED_ORIGINS" ] && args="$args --allowed-origins $BEAMDROP_ALLOWED_ORIGINS"
[ -n "$BEAMDROP_DB_PATH" ]         && args="$args --db-path $BEAMDROP_DB_PATH"
[ -n "$BEAMDROP_TLS_CERT" ]        && args="$args --tls-cert $BEAMDROP_TLS_CERT"
[ -n "$BEAMDROP_TLS_KEY" ]         && args="$args --tls-key $BEAMDROP_TLS_KEY"

# Boolean flags — only add when explicitly set to a truthy value
case "${BEAMDROP_API_AUTH:-}" in
    1|true|yes) args="$args --api-auth" ;;
esac

# If the caller passed extra arguments (docker run ... beamdrop --flag),
# append them so they can override the env-derived defaults.
exec beamdrop $args "$@"

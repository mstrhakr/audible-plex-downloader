#!/bin/sh
set -eu

APP_BIN="/usr/local/bin/audplexus"

# If container already runs as a non-root user (for example with --user),
# defer to that identity and ignore PUID/PGID.
if [ "$(id -u)" -ne 0 ]; then
    exec "$APP_BIN" "$@"
fi

RUN_UID="${PUID:-}"
RUN_GID="${PGID:-}"

# Unraid-style defaults when only one side is provided.
if [ -z "$RUN_UID" ] && [ -n "$RUN_GID" ]; then
    RUN_UID=99
fi
if [ -z "$RUN_GID" ] && [ -n "$RUN_UID" ]; then
    RUN_GID=100
fi

if [ -n "$RUN_UID" ] || [ -n "$RUN_GID" ]; then
    [ -n "$RUN_UID" ] || RUN_UID=99
    [ -n "$RUN_GID" ] || RUN_GID=100

    if [ "${TAKE_OWNERSHIP:-false}" = "true" ]; then
        for path in /config /audiobooks /downloads; do
            if [ -e "$path" ]; then
                chown -R "$RUN_UID:$RUN_GID" "$path" || true
            fi
        done
    fi

    exec su-exec "$RUN_UID:$RUN_GID" "$APP_BIN" "$@"
fi

exec "$APP_BIN" "$@"


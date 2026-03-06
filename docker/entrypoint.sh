#!/bin/sh
set -e

PUID=${PUID:-1000}
PGID=${PGID:-1000}

# Create group if GID doesn't exist yet
if ! getent group "$PGID" >/dev/null 2>&1; then
    addgroup -g "$PGID" -S seedghost
fi

# Create user if UID doesn't exist yet
if ! getent passwd "$PUID" >/dev/null 2>&1; then
    adduser -u "$PUID" -G "$(getent group "$PGID" | cut -d: -f1)" -S -D -H seedghost
fi

# Timezone
if [ -n "$TZ" ] && [ -f "/usr/share/zoneinfo/$TZ" ]; then
    cp "/usr/share/zoneinfo/$TZ" /etc/localtime
    echo "$TZ" > /etc/timezone
fi

# Fix ownership using numeric IDs
chown -R "$PUID:$PGID" /app/data /app/profiles

exec su-exec "$PUID:$PGID" ./seedghost "$@"

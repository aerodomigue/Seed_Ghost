#!/bin/sh
set -e

PUID=${PUID:-1000}
PGID=${PGID:-1000}

# Create group/user
addgroup -g "$PGID" -S seedghost 2>/dev/null || true
adduser -u "$PUID" -G seedghost -S -D -H seedghost 2>/dev/null || true

# Timezone
if [ -n "$TZ" ] && [ -f "/usr/share/zoneinfo/$TZ" ]; then
    cp "/usr/share/zoneinfo/$TZ" /etc/localtime
    echo "$TZ" > /etc/timezone
fi

# Fix ownership
chown -R seedghost:seedghost /app/data /app/profiles

exec su-exec seedghost:seedghost ./seedghost "$@"

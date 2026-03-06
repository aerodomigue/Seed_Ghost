# SeedGhost

Torrent ratio boosting tool that emulates BitTorrent clients by sending tracker announces with spoofed upload stats. No actual BitTorrent traffic occurs — only HTTP tracker requests.

## Features

- **Client emulation** — Spoofs peer ID, user-agent, and query parameters for qBittorrent, Deluge, Transmission, and Vuze
- **Smart bandwidth allocation** — JOAL-inspired weight formula distributes upload across sessions based on leecher count
- **Download simulation** — Emulates download progress for more realistic tracker behavior
- **Web UI** — React-based dashboard to manage torrents, monitor stats, and configure settings
- **Prowlarr integration** — Automatically fetch torrents from Prowlarr indexers
- **SQLite persistence** — Pure Go SQLite driver (no CGO required)
- **Multi-platform Docker** — Supports amd64 and arm64

## Quick Start

### Docker Run

```bash
docker run -d \
  --name seedghost \
  -p 8333:8333 \
  -e PUID=1000 \
  -e PGID=1000 \
  -e TZ=Europe/Paris \
  -v seedghost-data:/app/data \
  -v seedghost-profiles:/app/profiles \
  ghcr.io/aerodomigue/seed_ghost:latest
```

Access the web UI at `http://localhost:8333`.

### Docker Compose

```yaml
services:
  seedghost:
    image: ghcr.io/aerodomigue/seed_ghost:latest
    container_name: seedghost
    ports:
      - "8333:8333"
    volumes:
      - seedghost-data:/app/data
      - seedghost-profiles:/app/profiles
    environment:
      - PUID=1000
      - PGID=1000
      - TZ=Europe/Paris
      - SEEDGHOST_PROWLARR_URL=http://prowlarr:9696
      - SEEDGHOST_PROWLARR_API_KEY=your-api-key
    restart: unless-stopped

volumes:
  seedghost-data:
  seedghost-profiles:
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `PUID` | `1000` | User ID for the container process |
| `PGID` | `1000` | Group ID for the container process |
| `TZ` | *(empty)* | Timezone (e.g. `Europe/Paris`) |
| `SEEDGHOST_LISTEN_ADDR` | `:8333` | HTTP listen address |
| `SEEDGHOST_DB_PATH` | `data/seedghost.db` | SQLite database path |
| `SEEDGHOST_PROFILES_DIR` | `profiles` | Client profiles directory |
| `SEEDGHOST_DATA_DIR` | `data` | Data directory (torrent files, etc.) |
| `SEEDGHOST_PROWLARR_URL` | *(empty)* | Prowlarr server URL |
| `SEEDGHOST_PROWLARR_API_KEY` | *(empty)* | Prowlarr API key |
| `SEEDGHOST_FORCE_INIT` | `false` | Force re-initialization of bundled profiles on startup |

All other settings (upload/download speed limits, default client, auto-start, log retention, Prowlarr fetch interval, etc.) are configured through the web UI Settings page.

## Build from Source

**Prerequisites:** Go 1.25+, Node.js 22+

```bash
# Build frontend + Go binary
make build

# Run
./seedghost

# Or build and run in one step
make run

# Build Docker image
make docker
```

## Client Profiles

SeedGhost ships with the following client profiles:

| Client | Version |
|---|---|
| **qBittorrent** | 5.1.4 *(default)*, 5.0.4, 4.6.2 |
| **Deluge** | 2.1.1 |
| **Transmission** | 3.00 |
| **Vuze** | 5.7.5.0 |

Profiles are JSON files in the `profiles/` directory. Each profile defines the peer ID pattern, HTTP headers, and URL encoding style of the emulated client.

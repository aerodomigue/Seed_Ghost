# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Development Commands

```bash
make build          # Build frontend + Go binary (output: seedghost)
make run            # Build and run
make dev            # Run frontend dev server + backend concurrently
make test           # Go tests: go test ./...
make test-race      # Go tests with race detector
make test-frontend  # Frontend tests (Vitest)
make test-all       # All tests (backend + frontend)
make test-cover     # Coverage report
make clean          # Remove build artifacts
make docker         # Build Docker image
```

Run a single Go test:
```bash
go test ./internal/engine/ -run TestWeightCalculation
```

Frontend dev (standalone):
```bash
cd frontend && npm run dev    # Vite dev server, proxies API to :8333
cd frontend && npm run build  # Build to ../web/dist/
```

## Architecture

SeedGhost is a torrent ratio boosting tool. It emulates BitTorrent clients by sending tracker announces with spoofed upload stats. No actual BitTorrent traffic occurs.

### Core Flow
1. User uploads `.torrent` files via web UI
2. **Engine Manager** creates a **Session** per torrent, each running an announce loop in its own goroutine
3. Sessions use **Client Profiles** (JSON) to emulate specific BitTorrent clients (peer ID, headers, URL encoding)
4. **Announce** module sends HTTP tracker requests and parses bencode responses
5. Manager runs a **bandwidth dispatcher** goroutine that recalculates upload speed allocation across sessions every ~20 minutes
6. **Weight formula** (JOAL-inspired): `leechersRatio² × 100 × leechers` — sessions with no leechers get zero weight

### Backend (Go 1.25, stdlib router)
- **Web API**: `internal/web/server.go` — all routes under `/api/v1/`, registered on `http.ServeMux` (no external router). SPA fallback serves `index.html` for frontend routes.
- **Config**: `internal/config/service.go` — thread-safe (`RWMutex`) config service. Defaults loaded first, then DB overrides. Changes persisted to SQLite.
- **Database**: `internal/database/` — SQLite via `modernc.org/sqlite` (pure Go, no CGO). WAL mode, `MaxOpenConns=1` to avoid SQLITE_BUSY. Migrations auto-run on startup.
- **Engine**: `internal/engine/` — Manager coordinates sessions, distributes bandwidth by weight. Session state (PeerID, Key, Port, Uploaded) persisted in `announce_state` table.

### Frontend (React 19 + Vite + TypeScript)
- Built to `web/dist/`, embedded into Go binary via `web/embed.go` (`//go:embed all:dist`)
- Dev mode serves from filesystem; production uses embedded files
- Path alias: `@/*` → `src/*`
- API client: Axios with baseURL `/api/v1`
- Styling: Tailwind CSS with dark theme (`ghost-400` custom color)

## Conventions

- **Go errors**: Wrap with `fmt.Errorf("context: %w", err)`
- **Go tests**: Table-driven with `testServer(t)` helper pattern for web handler tests
- **Go logging**: stdlib `log` package with `LstdFlags | Lshortfile`
- **JSON tags**: snake_case field names
- **Frontend types**: Strict TypeScript, no `any`
- **No external Go router** — uses stdlib `http.ServeMux`
- **No `golift.io/starr`** — custom Prowlarr client in `internal/prowlarr/`
- **Default port**: 8333

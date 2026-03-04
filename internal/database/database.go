package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQL database connection.
type DB struct {
	*sql.DB
}

// Open opens a SQLite database at the given path and runs migrations.
func Open(path string) (*DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}

	sqlDB, err := sql.Open("sqlite", path+"?_journal=WAL&_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable foreign keys
	if _, err := sqlDB.Exec("PRAGMA foreign_keys = ON"); err != nil {
		sqlDB.Close()
		return nil, err
	}

	db := &DB{sqlDB}
	if err := db.migrate(); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS torrents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			info_hash TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			total_size INTEGER NOT NULL DEFAULT 0,
			tracker_url TEXT NOT NULL,
			torrent_data BLOB NOT NULL,
			client_profile TEXT NOT NULL DEFAULT '',
			active INTEGER NOT NULL DEFAULT 0,
			added_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			source TEXT NOT NULL DEFAULT 'manual'
		);

		CREATE TABLE IF NOT EXISTS announce_state (
			torrent_id INTEGER PRIMARY KEY REFERENCES torrents(id) ON DELETE CASCADE,
			peer_id TEXT NOT NULL,
			key TEXT NOT NULL,
			port INTEGER NOT NULL,
			uploaded INTEGER NOT NULL DEFAULT 0,
			downloaded INTEGER NOT NULL DEFAULT 0,
			last_announce DATETIME,
			last_interval INTEGER NOT NULL DEFAULT 1800,
			last_leechers INTEGER NOT NULL DEFAULT 0,
			last_seeders INTEGER NOT NULL DEFAULT 0,
			last_delta INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS stats_log (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			torrent_id INTEGER REFERENCES torrents(id) ON DELETE CASCADE,
			timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			uploaded_total INTEGER NOT NULL,
			leechers INTEGER NOT NULL DEFAULT 0,
			seeders INTEGER NOT NULL DEFAULT 0
		);

		CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);

		CREATE TABLE IF NOT EXISTS prowlarr_indexers (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			enabled INTEGER NOT NULL DEFAULT 0,
			max_upload_speed_kbs REAL,
			last_fetch DATETIME
		);

		CREATE TABLE IF NOT EXISTS ratio_targets (
			tracker_host TEXT PRIMARY KEY,
			target_ratio REAL NOT NULL DEFAULT 2.0
		);

		CREATE TABLE IF NOT EXISTS announce_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			torrent_id INTEGER REFERENCES torrents(id) ON DELETE CASCADE,
			timestamp DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			tracker_url TEXT NOT NULL,
			event TEXT NOT NULL DEFAULT '',
			upload_delta INTEGER NOT NULL DEFAULT 0,
			leechers INTEGER NOT NULL DEFAULT 0,
			seeders INTEGER NOT NULL DEFAULT 0,
			interval_secs INTEGER NOT NULL DEFAULT 0,
			status TEXT NOT NULL DEFAULT 'success',
			error_msg TEXT NOT NULL DEFAULT ''
		);

		CREATE INDEX IF NOT EXISTS idx_announce_logs_torrent ON announce_logs(torrent_id);
		CREATE INDEX IF NOT EXISTS idx_announce_logs_timestamp ON announce_logs(timestamp);
		CREATE INDEX IF NOT EXISTS idx_stats_log_torrent ON stats_log(torrent_id);
	`)
	return err
}

// TorrentRow represents a torrent record in the database.
type TorrentRow struct {
	ID            int64
	InfoHash      string
	Name          string
	TotalSize     int64
	TrackerURL    string
	TorrentData   []byte
	ClientProfile string
	Active        bool
	AddedAt       time.Time
	Source        string
}

// AnnounceStateRow represents the announce state for a torrent.
type AnnounceStateRow struct {
	TorrentID     int64
	PeerID        string
	Key           string
	Port          int
	Uploaded      int64
	Downloaded    int64
	LastAnnounce  *time.Time
	LastInterval  int
	LastLeechers  int
	LastSeeders   int
	LastDelta     int64
}

// InsertTorrent inserts a new torrent and returns its ID.
func (db *DB) InsertTorrent(t *TorrentRow) (int64, error) {
	res, err := db.Exec(
		`INSERT INTO torrents (info_hash, name, total_size, tracker_url, torrent_data, client_profile, active, source)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.InfoHash, t.Name, t.TotalSize, t.TrackerURL, t.TorrentData, t.ClientProfile, t.Active, t.Source,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// GetTorrent returns a torrent by ID.
func (db *DB) GetTorrent(id int64) (*TorrentRow, error) {
	row := db.QueryRow(
		`SELECT id, info_hash, name, total_size, tracker_url, torrent_data, client_profile, active, added_at, source
		 FROM torrents WHERE id = ?`, id,
	)
	t := &TorrentRow{}
	err := row.Scan(&t.ID, &t.InfoHash, &t.Name, &t.TotalSize, &t.TrackerURL, &t.TorrentData,
		&t.ClientProfile, &t.Active, &t.AddedAt, &t.Source)
	if err != nil {
		return nil, err
	}
	return t, nil
}

// ListTorrents returns all torrents.
func (db *DB) ListTorrents() ([]*TorrentRow, error) {
	rows, err := db.Query(
		`SELECT id, info_hash, name, total_size, tracker_url, torrent_data, client_profile, active, added_at, source
		 FROM torrents ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var torrents []*TorrentRow
	for rows.Next() {
		t := &TorrentRow{}
		if err := rows.Scan(&t.ID, &t.InfoHash, &t.Name, &t.TotalSize, &t.TrackerURL, &t.TorrentData,
			&t.ClientProfile, &t.Active, &t.AddedAt, &t.Source); err != nil {
			return nil, err
		}
		torrents = append(torrents, t)
	}
	return torrents, rows.Err()
}

// DeleteTorrent deletes a torrent by ID.
func (db *DB) DeleteTorrent(id int64) error {
	_, err := db.Exec("DELETE FROM torrents WHERE id = ?", id)
	return err
}

// SetTorrentActive sets the active flag for a torrent.
func (db *DB) SetTorrentActive(id int64, active bool) error {
	_, err := db.Exec("UPDATE torrents SET active = ? WHERE id = ?", active, id)
	return err
}

// UpsertAnnounceState creates or updates the announce state for a torrent.
func (db *DB) UpsertAnnounceState(s *AnnounceStateRow) error {
	_, err := db.Exec(
		`INSERT INTO announce_state (torrent_id, peer_id, key, port, uploaded, downloaded, last_announce, last_interval, last_leechers, last_seeders, last_delta)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(torrent_id) DO UPDATE SET
			peer_id=excluded.peer_id, key=excluded.key, port=excluded.port,
			uploaded=excluded.uploaded, downloaded=excluded.downloaded,
			last_announce=excluded.last_announce, last_interval=excluded.last_interval,
			last_leechers=excluded.last_leechers, last_seeders=excluded.last_seeders,
			last_delta=excluded.last_delta`,
		s.TorrentID, s.PeerID, s.Key, s.Port, s.Uploaded, s.Downloaded,
		s.LastAnnounce, s.LastInterval, s.LastLeechers, s.LastSeeders, s.LastDelta,
	)
	return err
}

// GetAnnounceState returns the announce state for a torrent.
func (db *DB) GetAnnounceState(torrentID int64) (*AnnounceStateRow, error) {
	row := db.QueryRow(
		`SELECT torrent_id, peer_id, key, port, uploaded, downloaded, last_announce, last_interval, last_leechers, last_seeders, last_delta
		 FROM announce_state WHERE torrent_id = ?`, torrentID,
	)
	s := &AnnounceStateRow{}
	err := row.Scan(&s.TorrentID, &s.PeerID, &s.Key, &s.Port, &s.Uploaded, &s.Downloaded,
		&s.LastAnnounce, &s.LastInterval, &s.LastLeechers, &s.LastSeeders, &s.LastDelta)
	if err != nil {
		return nil, err
	}
	return s, nil
}

// InsertAnnounceLog records an announce event.
func (db *DB) InsertAnnounceLog(torrentID int64, trackerURL, event string, uploadDelta int64, leechers, seeders, interval int, status, errMsg string) error {
	_, err := db.Exec(
		`INSERT INTO announce_logs (torrent_id, tracker_url, event, upload_delta, leechers, seeders, interval_secs, status, error_msg)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		torrentID, trackerURL, event, uploadDelta, leechers, seeders, interval, status, errMsg,
	)
	return err
}

// ListAnnounceLogs returns announce logs with optional filtering.
func (db *DB) ListAnnounceLogs(torrentID int64, limit, offset int) ([]AnnounceLogRow, error) {
	query := `SELECT id, torrent_id, timestamp, tracker_url, event, upload_delta, leechers, seeders, interval_secs, status, error_msg
		 FROM announce_logs`
	var args []interface{}
	if torrentID > 0 {
		query += " WHERE torrent_id = ?"
		args = append(args, torrentID)
	}
	query += " ORDER BY id DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []AnnounceLogRow
	for rows.Next() {
		var l AnnounceLogRow
		if err := rows.Scan(&l.ID, &l.TorrentID, &l.Timestamp, &l.TrackerURL, &l.Event,
			&l.UploadDelta, &l.Leechers, &l.Seeders, &l.IntervalSecs, &l.Status, &l.ErrorMsg); err != nil {
			return nil, err
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

// AnnounceLogRow represents an announce log entry.
type AnnounceLogRow struct {
	ID           int64
	TorrentID    int64
	Timestamp    time.Time
	TrackerURL   string
	Event        string
	UploadDelta  int64
	Leechers     int
	Seeders      int
	IntervalSecs int
	Status       string
	ErrorMsg     string
}

// InsertStatsLog records a stats snapshot.
func (db *DB) InsertStatsLog(torrentID, uploadedTotal int64, leechers, seeders int) error {
	_, err := db.Exec(
		`INSERT INTO stats_log (torrent_id, uploaded_total, leechers, seeders) VALUES (?, ?, ?, ?)`,
		torrentID, uploadedTotal, leechers, seeders,
	)
	return err
}

// CleanOldLogs removes announce logs older than the given number of days.
func (db *DB) CleanOldLogs(days int) error {
	_, err := db.Exec(
		"DELETE FROM announce_logs WHERE timestamp < datetime('now', ? || ' days')",
		fmt.Sprintf("-%d", days),
	)
	return err
}

// GetSetting returns a setting value by key.
func (db *DB) GetSetting(key string) (string, error) {
	var val string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&val)
	return val, err
}

// SetSetting upserts a setting.
func (db *DB) SetSetting(key, value string) error {
	_, err := db.Exec(
		"INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value",
		key, value,
	)
	return err
}

// UpsertRatioTarget sets a ratio target for a tracker host.
func (db *DB) UpsertRatioTarget(trackerHost string, targetRatio float64) error {
	_, err := db.Exec(
		"INSERT INTO ratio_targets (tracker_host, target_ratio) VALUES (?, ?) ON CONFLICT(tracker_host) DO UPDATE SET target_ratio=excluded.target_ratio",
		trackerHost, targetRatio,
	)
	return err
}

// GetRatioTargets returns all ratio targets.
func (db *DB) GetRatioTargets() (map[string]float64, error) {
	rows, err := db.Query("SELECT tracker_host, target_ratio FROM ratio_targets")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targets := make(map[string]float64)
	for rows.Next() {
		var host string
		var ratio float64
		if err := rows.Scan(&host, &ratio); err != nil {
			return nil, err
		}
		targets[host] = ratio
	}
	return targets, rows.Err()
}

// GetActiveTorrents returns all torrents with active=1.
func (db *DB) GetActiveTorrents() ([]*TorrentRow, error) {
	rows, err := db.Query(
		`SELECT id, info_hash, name, total_size, tracker_url, torrent_data, client_profile, active, added_at, source
		 FROM torrents WHERE active = 1 ORDER BY id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var torrents []*TorrentRow
	for rows.Next() {
		t := &TorrentRow{}
		if err := rows.Scan(&t.ID, &t.InfoHash, &t.Name, &t.TotalSize, &t.TrackerURL, &t.TorrentData,
			&t.ClientProfile, &t.Active, &t.AddedAt, &t.Source); err != nil {
			return nil, err
		}
		torrents = append(torrents, t)
	}
	return torrents, rows.Err()
}

package database

import (
	"path/filepath"
	"testing"
	"time"
)

func testDB(t *testing.T) *DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestInsertAndGetTorrent(t *testing.T) {
	db := testDB(t)

	row := &TorrentRow{
		InfoHash:      "abcdef1234567890abcd",
		Name:          "Test Torrent",
		TotalSize:     1024000,
		TrackerURL:    "http://tracker.example.com/announce",
		TorrentData:   []byte("fake torrent data"),
		ClientProfile: "qBittorrent 4.6.2",
		Active:        false,
		Source:        "manual",
	}
	id, err := db.InsertTorrent(row)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if id <= 0 {
		t.Error("expected positive ID")
	}

	got, err := db.GetTorrent(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Test Torrent" {
		t.Errorf("name = %q, want %q", got.Name, "Test Torrent")
	}
	if got.InfoHash != "abcdef1234567890abcd" {
		t.Errorf("info_hash = %q, want %q", got.InfoHash, "abcdef1234567890abcd")
	}
}

func TestListTorrents(t *testing.T) {
	db := testDB(t)

	for i := 0; i < 3; i++ {
		db.InsertTorrent(&TorrentRow{
			InfoHash:    "hash" + string(rune('a'+i)),
			Name:        "Torrent",
			TrackerURL:  "http://tracker.example.com/announce",
			TorrentData: []byte("data"),
			Source:      "manual",
		})
	}

	list, err := db.ListTorrents()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 3 {
		t.Errorf("count = %d, want 3", len(list))
	}
}

func TestDeleteTorrent(t *testing.T) {
	db := testDB(t)

	id, _ := db.InsertTorrent(&TorrentRow{
		InfoHash: "todelete", Name: "Del", TrackerURL: "http://t.co/a",
		TorrentData: []byte("d"), Source: "manual",
	})

	err := db.DeleteTorrent(id)
	if err != nil {
		t.Fatalf("delete: %v", err)
	}

	_, err = db.GetTorrent(id)
	if err == nil {
		t.Error("expected error after delete")
	}
}

func TestSetTorrentActive(t *testing.T) {
	db := testDB(t)

	id, _ := db.InsertTorrent(&TorrentRow{
		InfoHash: "active", Name: "Act", TrackerURL: "http://t.co/a",
		TorrentData: []byte("d"), Source: "manual",
	})

	db.SetTorrentActive(id, true)
	got, _ := db.GetTorrent(id)
	if !got.Active {
		t.Error("expected active=true")
	}
}

func TestAnnounceState(t *testing.T) {
	db := testDB(t)

	id, _ := db.InsertTorrent(&TorrentRow{
		InfoHash: "statetest", Name: "State", TrackerURL: "http://t.co/a",
		TorrentData: []byte("d"), Source: "manual",
	})

	now := time.Now()
	state := &AnnounceStateRow{
		TorrentID:    id,
		PeerID:       "-qB4620-abcdefghijkl",
		Key:          "a1b2c3d4",
		Port:         12345,
		Uploaded:     1024000,
		LastAnnounce: &now,
		LastInterval: 1800,
		LastLeechers: 5,
		LastSeeders:  10,
		LastDelta:    51200,
	}
	if err := db.UpsertAnnounceState(state); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := db.GetAnnounceState(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Uploaded != 1024000 {
		t.Errorf("uploaded = %d, want 1024000", got.Uploaded)
	}
	if got.PeerID != "-qB4620-abcdefghijkl" {
		t.Errorf("peerID = %q", got.PeerID)
	}
}

func TestSettings(t *testing.T) {
	db := testDB(t)

	db.SetSetting("foo", "bar")
	val, err := db.GetSetting("foo")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if val != "bar" {
		t.Errorf("value = %q, want %q", val, "bar")
	}

	// Update
	db.SetSetting("foo", "baz")
	val, _ = db.GetSetting("foo")
	if val != "baz" {
		t.Errorf("value = %q, want %q", val, "baz")
	}
}

func TestRatioTargets(t *testing.T) {
	db := testDB(t)

	db.UpsertRatioTarget("tracker.example.com", 2.5)
	targets, err := db.GetRatioTargets()
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if targets["tracker.example.com"] != 2.5 {
		t.Errorf("ratio = %f, want 2.5", targets["tracker.example.com"])
	}
}

func TestAnnounceLogs(t *testing.T) {
	db := testDB(t)

	id, _ := db.InsertTorrent(&TorrentRow{
		InfoHash: "logtest", Name: "Log", TrackerURL: "http://t.co/a",
		TorrentData: []byte("d"), Source: "manual",
	})

	err := db.InsertAnnounceLog(id, "http://t.co/a", "", 51200, 5, 10, 1800, "success", "")
	if err != nil {
		t.Fatalf("insert log: %v", err)
	}

	logs, err := db.ListAnnounceLogs(id, 10, 0)
	if err != nil {
		t.Fatalf("list logs: %v", err)
	}
	if len(logs) != 1 {
		t.Fatalf("logs count = %d, want 1", len(logs))
	}
	if logs[0].UploadDelta != 51200 {
		t.Errorf("uploadDelta = %d, want 51200", logs[0].UploadDelta)
	}
}

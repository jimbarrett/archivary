package sync

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSyncConfig_MissingFile(t *testing.T) {
	dir := t.TempDir()
	cfg, err := LoadSyncConfig(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Remotes) != 0 {
		t.Fatalf("expected empty remotes, got %d", len(cfg.Remotes))
	}
}

func TestSyncConfig_RoundTrip(t *testing.T) {
	dir := t.TempDir()

	original := &SyncConfig{
		Remotes: []RemoteConfig{
			{
				Path:                "work",
				URL:                 "git@example.com:repo.git",
				Branch:              "main",
				AutoCommit:          true,
				AutoPush:            true,
				PushIntervalMinutes: 5,
			},
			{
				Path:                "personal",
				URL:                 "https://github.com/user/notes.git",
				Branch:              "master",
				AutoCommit:          false,
				AutoPush:            false,
				PushIntervalMinutes: 0,
			},
		},
	}

	if err := SaveSyncConfig(dir, original); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Verify the file was created
	if _, err := os.Stat(filepath.Join(dir, syncConfigFile)); err != nil {
		t.Fatalf("sync.json not created: %v", err)
	}

	loaded, err := LoadSyncConfig(dir)
	if err != nil {
		t.Fatalf("load error: %v", err)
	}

	if len(loaded.Remotes) != 2 {
		t.Fatalf("expected 2 remotes, got %d", len(loaded.Remotes))
	}

	r := loaded.Remotes[0]
	if r.Path != "work" || r.URL != "git@example.com:repo.git" || r.Branch != "main" {
		t.Errorf("remote 0 mismatch: %+v", r)
	}
	if !r.AutoCommit || !r.AutoPush || r.PushIntervalMinutes != 5 {
		t.Errorf("remote 0 settings mismatch: %+v", r)
	}

	r = loaded.Remotes[1]
	if r.Path != "personal" || r.URL != "https://github.com/user/notes.git" {
		t.Errorf("remote 1 mismatch: %+v", r)
	}
}

func TestSaveConfig_NoTmpLeftBehind(t *testing.T) {
	dir := t.TempDir()
	cfg := &SyncConfig{Remotes: []RemoteConfig{{Path: "test"}}}

	if err := SaveSyncConfig(dir, cfg); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// No .tmp file should remain
	if _, err := os.Stat(filepath.Join(dir, syncConfigFile+".tmp")); !os.IsNotExist(err) {
		t.Error("temp file was not cleaned up")
	}
}

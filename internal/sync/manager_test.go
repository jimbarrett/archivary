package sync

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jimbarrett/archivary/internal/index"
	"github.com/jimbarrett/archivary/internal/store"
)

// testEnv sets up a workspace dir, data dir, FileStore, and Indexer for tests.
type testEnv struct {
	workspaceDir string
	dataDir      string
	store        *store.FileStore
	indexer      *index.Indexer
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	workspace := t.TempDir()
	dataDir := t.TempDir()

	fs, err := store.NewFileStore(workspace)
	if err != nil {
		t.Fatalf("NewFileStore: %v", err)
	}

	db, err := index.OpenMemoryDB()
	if err != nil {
		t.Fatalf("OpenMemoryDB: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	idx := index.NewIndexer(db)
	return &testEnv{
		workspaceDir: workspace,
		dataDir:      dataDir,
		store:        fs,
		indexer:      idx,
	}
}

// configureGitUser sets user.name and user.email in a repo so commits work.
func configureGitUser(t *testing.T, repoDir string) {
	t.Helper()
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = repoDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}
	run("config", "user.email", "test@test.com")
	run("config", "user.name", "Test")
}

// createBareRepo creates a bare git repo and returns its path.
func createBareRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	cmd := exec.Command("git", "init", "--bare", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init --bare: %s", out)
	}
	return dir
}

func TestNewSyncManager_NoConfig(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatalf("NewSyncManager: %v", err)
	}
	if len(m.repos) != 0 {
		t.Fatalf("expected 0 repos, got %d", len(m.repos))
	}
	if len(m.config.Remotes) != 0 {
		t.Fatalf("expected 0 remotes, got %d", len(m.config.Remotes))
	}
}

func TestAddRemote_Init(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatal(err)
	}

	rc := RemoteConfig{
		Path:       "notes",
		AutoCommit: true,
		Branch:     "main",
	}
	if err := m.AddRemote(rc); err != nil {
		t.Fatalf("AddRemote: %v", err)
	}

	// Verify the directory was created with a git repo.
	gitDir := filepath.Join(env.workspaceDir, "notes", ".git")
	if _, err := os.Stat(gitDir); err != nil {
		t.Fatalf("expected .git dir at %s: %v", gitDir, err)
	}

	// Verify config was saved.
	if len(m.config.Remotes) != 1 {
		t.Fatalf("expected 1 remote, got %d", len(m.config.Remotes))
	}
	if m.config.Remotes[0].Path != "notes" {
		t.Fatalf("expected path 'notes', got %q", m.config.Remotes[0].Path)
	}

	// Verify repo is tracked.
	if _, ok := m.repos["notes"]; !ok {
		t.Fatal("expected repo in repos map")
	}
}

func TestAddRemote_Clone(t *testing.T) {
	// Create a source repo with a commit to clone from.
	srcDir := t.TempDir()
	cmd := exec.Command("git", "init", srcDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %s", out)
	}
	configureGitUser(t, srcDir)

	// Create a file and commit.
	if err := os.WriteFile(filepath.Join(srcDir, "readme.md"), []byte("# Hello"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"add", "readme.md"},
		{"commit", "-m", "Initial"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = srcDir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %s", args, out)
		}
	}

	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatal(err)
	}

	rc := RemoteConfig{
		Path: "cloned",
		URL:  srcDir,
	}
	if err := m.AddRemote(rc); err != nil {
		t.Fatalf("AddRemote clone: %v", err)
	}

	// Verify cloned file exists and contains the expected content.
	// Note: reindex after clone may add frontmatter, so check with Contains.
	content, err := os.ReadFile(filepath.Join(env.workspaceDir, "cloned", "readme.md"))
	if err != nil {
		t.Fatalf("reading cloned file: %v", err)
	}
	if !strings.Contains(string(content), "# Hello") {
		t.Fatalf("cloned content missing '# Hello': %q", string(content))
	}
}

func TestAddRemote_Duplicate(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatal(err)
	}

	rc := RemoteConfig{Path: "notes"}
	if err := m.AddRemote(rc); err != nil {
		t.Fatal(err)
	}
	if err := m.AddRemote(rc); err == nil {
		t.Fatal("expected error on duplicate AddRemote")
	}
}

func TestNotifyChange_AutoCommit(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatal(err)
	}

	rc := RemoteConfig{
		Path:       "work",
		AutoCommit: true,
	}
	if err := m.AddRemote(rc); err != nil {
		t.Fatal(err)
	}
	configureGitUser(t, filepath.Join(env.workspaceDir, "work"))

	// Write a file into the synced directory.
	filePath := filepath.Join(env.workspaceDir, "work", "note.md")
	if err := os.WriteFile(filePath, []byte("# Note"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Notify the manager.
	m.NotifyChange("work/note.md", "create")

	// Verify a commit was created.
	commits, err := m.Log("work", 1)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].Message != "create note.md" {
		t.Fatalf("commit message = %q, want %q", commits[0].Message, "create note.md")
	}
}

func TestNotifyChange_IgnoresUnsynced(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatal(err)
	}

	// No synced repos — NotifyChange should not panic.
	m.NotifyChange("random/file.md", "create")

	// File in root (no directory) should be ignored.
	m.NotifyChange("file.md", "create")
}

func TestNotifyChange_NoAutoCommit(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatal(err)
	}

	rc := RemoteConfig{
		Path:       "work",
		AutoCommit: false,
	}
	if err := m.AddRemote(rc); err != nil {
		t.Fatal(err)
	}
	configureGitUser(t, filepath.Join(env.workspaceDir, "work"))

	filePath := filepath.Join(env.workspaceDir, "work", "note.md")
	if err := os.WriteFile(filePath, []byte("# Note"), 0o644); err != nil {
		t.Fatal(err)
	}

	m.NotifyChange("work/note.md", "create")

	// No commit should exist because auto-commit is off.
	commits, err := m.Log("work", 10)
	if err != nil {
		// git log fails on empty repo — that's expected.
		return
	}
	if len(commits) != 0 {
		t.Fatalf("expected 0 commits, got %d", len(commits))
	}
}

func TestRemoveRemote(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.AddRemote(RemoteConfig{Path: "notes"}); err != nil {
		t.Fatal(err)
	}
	if err := m.RemoveRemote("notes"); err != nil {
		t.Fatalf("RemoveRemote: %v", err)
	}

	// Config should be empty.
	if len(m.config.Remotes) != 0 {
		t.Fatalf("expected 0 remotes after remove, got %d", len(m.config.Remotes))
	}

	// Repo should be gone from map.
	if _, ok := m.repos["notes"]; ok {
		t.Fatal("expected repo removed from map")
	}

	// .git directory should be removed.
	gitDir := filepath.Join(env.workspaceDir, "notes", ".git")
	if _, err := os.Stat(gitDir); err == nil {
		t.Fatal(".git dir should have been removed")
	}

	// The directory itself should still exist (files kept).
	notesDir := filepath.Join(env.workspaceDir, "notes")
	if _, err := os.Stat(notesDir); err != nil {
		t.Fatalf("notes dir should still exist: %v", err)
	}
}

func TestRemoveRemote_NotFound(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatal(err)
	}

	if err := m.RemoveRemote("nonexistent"); err == nil {
		t.Fatal("expected error removing nonexistent remote")
	}
}

func TestStatus(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatal(err)
	}

	// Empty status.
	status := m.Status()
	if len(status) != 0 {
		t.Fatalf("expected empty status, got %d entries", len(status))
	}

	// Add a remote.
	if err := m.AddRemote(RemoteConfig{Path: "work", Branch: "main"}); err != nil {
		t.Fatal(err)
	}

	status = m.Status()
	if len(status) != 1 {
		t.Fatalf("expected 1 status entry, got %d", len(status))
	}
	ds, ok := status["work"]
	if !ok {
		t.Fatal("expected status for 'work'")
	}
	if ds.Path != "work" || ds.Branch != "main" {
		t.Fatalf("unexpected status: %+v", ds)
	}
	if !ds.Clean {
		t.Fatal("expected clean status for empty repo")
	}
}

func TestConfigPersistence(t *testing.T) {
	env := setupTestEnv(t)
	m, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatal(err)
	}

	rc := RemoteConfig{
		Path:                "notes",
		AutoCommit:          true,
		AutoPush:            true,
		PushIntervalMinutes: 10,
	}
	if err := m.AddRemote(rc); err != nil {
		t.Fatal(err)
	}

	// Create a new manager from the same data dir — config should persist.
	m2, err := NewSyncManager(env.workspaceDir, env.dataDir, env.store, env.indexer)
	if err != nil {
		t.Fatalf("NewSyncManager reload: %v", err)
	}

	if len(m2.config.Remotes) != 1 {
		t.Fatalf("expected 1 remote after reload, got %d", len(m2.config.Remotes))
	}
	loaded := m2.config.Remotes[0]
	if loaded.Path != "notes" || !loaded.AutoCommit || !loaded.AutoPush || loaded.PushIntervalMinutes != 10 {
		t.Fatalf("config mismatch after reload: %+v", loaded)
	}

	// Verify the repo was re-opened.
	if _, ok := m2.repos["notes"]; !ok {
		t.Fatal("expected repo to be re-opened on reload")
	}
}

func TestTopLevelDir(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"work/notes/foo.md", "work"},
		{"personal/journal.md", "personal"},
		{"file.md", ""},
		{"a/b/c/d.md", "a"},
	}
	for _, tt := range tests {
		got := topLevelDir(tt.input)
		if got != tt.want {
			t.Errorf("topLevelDir(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

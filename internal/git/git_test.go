package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// helper: create a temp dir and return its path + cleanup func.
func tmpDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	return dir
}

// helper: write a file in a directory.
func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// helper: init a repo with git user config set (required for commits).
func initRepo(t *testing.T) *GitRepo {
	t.Helper()
	dir := tmpDir(t)
	repo, err := Init(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Set user config so commits work without global gitconfig.
	if _, err := repo.run("config", "user.email", "test@test.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.run("config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}
	return repo
}


func TestInitAndOpen(t *testing.T) {
	dir := tmpDir(t)

	// Open should fail on a non-repo directory.
	_, err := Open(dir)
	if err == nil {
		t.Fatal("expected error opening non-repo dir")
	}

	// Init should create a repo.
	repo, err := Init(dir)
	if err != nil {
		t.Fatalf("Init: %v", err)
	}
	if repo.Dir() != dir {
		t.Fatalf("Dir() = %s, want %s", repo.Dir(), dir)
	}

	// Open should now succeed.
	repo2, err := Open(dir)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if repo2.Dir() != dir {
		t.Fatalf("Dir() = %s, want %s", repo2.Dir(), dir)
	}
}

func TestAddAndCommit(t *testing.T) {
	repo := initRepo(t)

	writeFile(t, repo.Dir(), "hello.md", "# Hello")

	if err := repo.Add("hello.md"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := repo.Commit("Add hello"); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	// Verify the commit exists.
	commits, err := repo.Log(1)
	if err != nil {
		t.Fatalf("Log: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].Message != "Add hello" {
		t.Fatalf("message = %q, want %q", commits[0].Message, "Add hello")
	}
}

func TestCommitNoChanges(t *testing.T) {
	repo := initRepo(t)

	writeFile(t, repo.Dir(), "hello.md", "# Hello")
	if err := repo.Add("hello.md"); err != nil {
		t.Fatal(err)
	}
	if err := repo.Commit("First"); err != nil {
		t.Fatal(err)
	}

	// Commit with nothing staged should be a no-op, not an error.
	if err := repo.Commit("Empty"); err != nil {
		t.Fatalf("expected nil for no-op commit, got: %v", err)
	}

	commits, err := repo.Log(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
}

func TestStatus(t *testing.T) {
	repo := initRepo(t)

	// Empty repo, no files — clean.
	s, err := repo.Status()
	if err != nil {
		t.Fatal(err)
	}
	if !s.Clean {
		t.Fatal("expected clean status for empty repo")
	}

	// Add an untracked file.
	writeFile(t, repo.Dir(), "new.md", "new file")
	s, err = repo.Status()
	if err != nil {
		t.Fatal(err)
	}
	if s.Clean {
		t.Fatal("expected dirty status")
	}
	if len(s.Untracked) != 1 || s.Untracked[0] != "new.md" {
		t.Fatalf("Untracked = %v, want [new.md]", s.Untracked)
	}

	// Stage and commit, then modify.
	repo.Add("new.md")
	repo.Commit("Add new")

	writeFile(t, repo.Dir(), "new.md", "modified content")
	s, err = repo.Status()
	if err != nil {
		t.Fatal(err)
	}
	if s.Clean {
		t.Fatal("expected dirty status after modify")
	}
	if len(s.Modified) != 1 {
		t.Fatalf("Modified = %v, want 1 file", s.Modified)
	}
}

func TestLog(t *testing.T) {
	repo := initRepo(t)

	for i := 0; i < 5; i++ {
		writeFile(t, repo.Dir(), "file.md", string(rune('a'+i)))
		repo.Add("file.md")
		repo.Commit("Commit " + string(rune('0'+i)))
	}

	// Get last 3.
	commits, err := repo.Log(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(commits) != 3 {
		t.Fatalf("expected 3 commits, got %d", len(commits))
	}
	// Most recent first.
	if commits[0].Message != "Commit 4" {
		t.Fatalf("first commit message = %q, want %q", commits[0].Message, "Commit 4")
	}
	if commits[0].Hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if commits[0].Author != "Test" {
		t.Fatalf("author = %q, want %q", commits[0].Author, "Test")
	}
	if commits[0].Time.IsZero() {
		t.Fatal("expected non-zero time")
	}
}

func TestRemote(t *testing.T) {
	repo := initRepo(t)

	has, err := repo.HasRemote()
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("expected no remote")
	}

	// Set a remote.
	if err := repo.SetRemote("/tmp/fake-remote.git"); err != nil {
		t.Fatal(err)
	}

	has, err = repo.HasRemote()
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected remote after SetRemote")
	}

	url, err := repo.RemoteURL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "/tmp/fake-remote.git" {
		t.Fatalf("RemoteURL = %q, want %q", url, "/tmp/fake-remote.git")
	}

	// Update the remote.
	if err := repo.SetRemote("/tmp/other-remote.git"); err != nil {
		t.Fatal(err)
	}
	url, err = repo.RemoteURL()
	if err != nil {
		t.Fatal(err)
	}
	if url != "/tmp/other-remote.git" {
		t.Fatalf("RemoteURL = %q, want %q", url, "/tmp/other-remote.git")
	}
}

func TestPushAndPull(t *testing.T) {
	// Set up a "remote" as a local bare repo.
	repo := initRepo(t)
	bareDir := tmpDir(t)

	// Create bare repo.
	bare := execCommand("git", "init", "--bare", bareDir)
	if out, err := bare.CombinedOutput(); err != nil {
		t.Fatalf("creating bare repo: %s", out)
	}

	repo.SetRemote(bareDir)

	// Create a commit and push.
	writeFile(t, repo.Dir(), "hello.md", "# Hello")
	repo.Add("hello.md")
	repo.Commit("Initial")

	// Need to set upstream for the first push.
	if _, err := repo.run("push", "-u", "origin", "main"); err != nil {
		// Try "master" if "main" fails (depends on git config).
		if _, err2 := repo.run("push", "-u", "origin", "master"); err2 != nil {
			t.Fatalf("initial push failed: %v / %v", err, err2)
		}
	}

	// Clone into a second repo to simulate another device.
	clone2Dir := tmpDir(t)
	repo2, err := Clone(bareDir, clone2Dir)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}
	if _, err := repo2.run("config", "user.email", "test@test.com"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo2.run("config", "user.name", "Test"); err != nil {
		t.Fatal(err)
	}

	// Make a change in repo2, push it.
	writeFile(t, repo2.Dir(), "hello.md", "# Hello World")
	repo2.Add("hello.md")
	repo2.Commit("Update from device 2")
	if err := repo2.Push(); err != nil {
		t.Fatalf("Push from repo2: %v", err)
	}

	// Pull in original repo.
	if err := repo.Pull(); err != nil {
		t.Fatalf("Pull: %v", err)
	}

	// Verify the change landed.
	content, err := os.ReadFile(filepath.Join(repo.Dir(), "hello.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "# Hello World" {
		t.Fatalf("content after pull = %q, want %q", string(content), "# Hello World")
	}

	// Check ahead/behind.
	s, err := repo.Status()
	if err != nil {
		t.Fatal(err)
	}
	if s.Ahead != 0 || s.Behind != 0 {
		t.Fatalf("Ahead=%d Behind=%d, want 0/0", s.Ahead, s.Behind)
	}
}

func TestPushNoRemote(t *testing.T) {
	repo := initRepo(t)

	writeFile(t, repo.Dir(), "hello.md", "# Hello")
	repo.Add("hello.md")
	repo.Commit("Initial")

	// Push with no remote should be a no-op.
	if err := repo.Push(); err != nil {
		t.Fatalf("Push with no remote should be nil, got: %v", err)
	}
}

func TestPullNoRemote(t *testing.T) {
	repo := initRepo(t)

	// Pull with no remote should be a no-op.
	if err := repo.Pull(); err != nil {
		t.Fatalf("Pull with no remote should be nil, got: %v", err)
	}
}

func TestClone(t *testing.T) {
	// Create a source repo with a commit.
	src := initRepo(t)
	writeFile(t, src.Dir(), "page.md", "# Page")
	src.Add("page.md")
	src.Commit("Initial")

	// Clone it.
	dst := tmpDir(t)
	repo, err := Clone(src.Dir(), dst)
	if err != nil {
		t.Fatalf("Clone: %v", err)
	}

	// Verify cloned content.
	content, err := os.ReadFile(filepath.Join(repo.Dir(), "page.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "# Page" {
		t.Fatalf("cloned content = %q, want %q", string(content), "# Page")
	}

	// Verify remote is set.
	has, err := repo.HasRemote()
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("expected remote on cloned repo")
	}
}

func TestAddEmpty(t *testing.T) {
	repo := initRepo(t)
	// Adding no files should be a no-op.
	if err := repo.Add(); err != nil {
		t.Fatalf("Add() with no files should be nil, got: %v", err)
	}
}

// execCommand is a helper that wraps exec.Command for tests.
func execCommand(name string, args ...string) *execCmd {
	return &execCmd{cmd: exec.Command(name, args...)}
}

type execCmd struct {
	cmd *exec.Cmd
}

func (e *execCmd) CombinedOutput() ([]byte, error) {
	return e.cmd.CombinedOutput()
}

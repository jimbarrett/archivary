// Package git wraps the git CLI for syncing workspace directories to remote repositories.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// GitRepo represents a git repository at a specific directory.
type GitRepo struct {
	dir string
}

// RepoStatus holds the current state of a git repository.
type RepoStatus struct {
	Clean      bool
	Ahead      int
	Behind     int
	Modified   []string
	Untracked  []string
	Conflicted []string
}

// Commit represents a single git commit.
type Commit struct {
	Hash    string    `json:"hash"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Time    time.Time `json:"time"`
}

// Open verifies the directory is a git repository and returns a GitRepo.
func Open(dir string) (*GitRepo, error) {
	r := &GitRepo{dir: dir}
	_, err := r.run("rev-parse", "--git-dir")
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %s", dir)
	}
	return r, nil
}

// Init initializes a new git repository in the given directory.
func Init(dir string) (*GitRepo, error) {
	cmd := exec.Command("git", "init", dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git init: %s", bytes.TrimSpace(out))
	}
	return &GitRepo{dir: dir}, nil
}

// Clone clones a remote repository into the given directory.
func Clone(url, dir string) (*GitRepo, error) {
	cmd := exec.Command("git", "clone", url, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("git clone: %s", bytes.TrimSpace(out))
	}
	return &GitRepo{dir: dir}, nil
}

// Dir returns the repository's working directory.
func (r *GitRepo) Dir() string {
	return r.dir
}

// Add stages files for the next commit.
func (r *GitRepo) Add(files ...string) error {
	if len(files) == 0 {
		return nil
	}
	args := append([]string{"add", "--"}, files...)
	_, err := r.run(args...)
	return err
}

// AddAll stages all changes (new, modified, deleted) in the repository.
func (r *GitRepo) AddAll() error {
	_, err := r.run("add", "-A")
	return err
}

// Commit creates a commit with the given message.
// Returns nil if there is nothing to commit.
func (r *GitRepo) Commit(message string) error {
	_, err := r.run("diff", "--cached", "--quiet")
	if err == nil {
		// Exit code 0 means no staged changes — nothing to commit.
		return nil
	}
	_, err = r.run("commit", "-m", message)
	return err
}

// Push pushes to the remote. Returns nil if there is no remote configured.
func (r *GitRepo) Push() error {
	has, err := r.HasRemote()
	if err != nil {
		return err
	}
	if !has {
		return nil
	}
	_, err = r.run("push")
	return err
}

// Pull pulls from the remote with rebase. Returns nil if there is no remote configured.
func (r *GitRepo) Pull() error {
	has, err := r.HasRemote()
	if err != nil {
		return err
	}
	if !has {
		return nil
	}
	// Fetch first to update tracking refs.
	if _, err := r.run("fetch"); err != nil {
		return err
	}
	// Only rebase if the tracking branch exists and has commits.
	upstream, upErr := r.run("rev-parse", "--abbrev-ref", "@{u}")
	if upErr != nil || strings.TrimSpace(upstream) == "" {
		return nil
	}
	_, err = r.run("rebase", strings.TrimSpace(upstream))
	return err
}

// Status returns the current repository status.
func (r *GitRepo) Status() (*RepoStatus, error) {
	out, err := r.run("status", "--porcelain")
	if err != nil {
		return nil, err
	}

	s := &RepoStatus{Clean: true}
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 3 {
			continue
		}
		s.Clean = false
		xy := line[:2]
		file := strings.TrimSpace(line[3:])

		switch {
		case xy == "??" || xy == "!!":
			s.Untracked = append(s.Untracked, file)
		case strings.Contains(xy, "U") || xy == "AA" || xy == "DD":
			s.Conflicted = append(s.Conflicted, file)
		default:
			s.Modified = append(s.Modified, file)
		}
	}

	// Ahead/behind tracking branch.
	abOut, err := r.run("rev-list", "--left-right", "--count", "@{u}...HEAD")
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(abOut))
		if len(parts) == 2 {
			s.Behind, _ = strconv.Atoi(parts[0])
			s.Ahead, _ = strconv.Atoi(parts[1])
		}
	}
	// If rev-list fails (no upstream), ahead/behind stay 0 — that's fine.

	return s, nil
}

// HasRemote returns true if the repository has at least one remote configured.
func (r *GitRepo) HasRemote() (bool, error) {
	out, err := r.run("remote")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// RemoteURL returns the fetch URL of the "origin" remote, or empty string if none.
func (r *GitRepo) RemoteURL() (string, error) {
	out, err := r.run("remote", "get-url", "origin")
	if err != nil {
		return "", nil
	}
	return strings.TrimSpace(out), nil
}

// SetRemote sets (or updates) the "origin" remote URL.
func (r *GitRepo) SetRemote(url string) error {
	has, err := r.HasRemote()
	if err != nil {
		return err
	}
	if has {
		_, err = r.run("remote", "set-url", "origin", url)
	} else {
		_, err = r.run("remote", "add", "origin", url)
	}
	return err
}

// HasConflicts returns true if any tracked files contain conflict markers.
func (r *GitRepo) HasConflicts() (bool, error) {
	out, err := r.run("diff", "--name-only", "--diff-filter=U")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(out) != "", nil
}

// Log returns the last n commits.
func (r *GitRepo) Log(n int) ([]Commit, error) {
	// Use a delimiter that won't appear in commit messages.
	const sep = "---ARCHIVARY_SEP---"
	format := fmt.Sprintf("%%H%s%%s%s%%an%s%%aI", sep, sep, sep)
	out, err := r.run("log", fmt.Sprintf("-%d", n), fmt.Sprintf("--format=%s", format))
	if err != nil {
		return nil, err
	}

	var commits []Commit
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, sep, 4)
		if len(parts) < 4 {
			continue
		}
		t, _ := time.Parse(time.RFC3339, parts[3])
		commits = append(commits, Commit{
			Hash:    parts[0],
			Message: parts[1],
			Author:  parts[2],
			Time:    t,
		})
	}
	return commits, nil
}

// run executes a git command in the repo directory and returns stdout.
func (r *GitRepo) run(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.dir

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = strings.TrimSpace(stdout.String())
		}
		return "", fmt.Errorf("git %s: %s", args[0], msg)
	}
	return stdout.String(), nil
}

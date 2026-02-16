package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	gosync "sync"
	"time"

	"github.com/jimbarrett/archivary/internal/git"
	"github.com/jimbarrett/archivary/internal/index"
	"github.com/jimbarrett/archivary/internal/store"
)

// DirSyncStatus describes the sync state of a single directory.
type DirSyncStatus struct {
	Path     string    `json:"path"`
	URL      string    `json:"url"`
	Branch   string    `json:"branch"`
	Clean    bool      `json:"clean"`
	Ahead    int       `json:"ahead"`
	Behind   int       `json:"behind"`
	LastSync time.Time `json:"last_sync"`
	Error    string    `json:"error,omitempty"`
}

// SyncManager orchestrates git sync for the workspace.
type SyncManager struct {
	repos        map[string]*git.GitRepo
	config       *SyncConfig
	dataDir      string
	workspaceDir string
	store        *store.FileStore
	indexer      *index.Indexer
	stopCh       chan struct{}
	mu           gosync.Mutex

	lastPush  map[string]time.Time
	lastError map[string]string
	lastSync  map[string]time.Time
}

// NewSyncManager creates a SyncManager, loads existing config, and opens the
// workspace repo if configured.
func NewSyncManager(workspaceDir, dataDir string, s *store.FileStore, idx *index.Indexer) (*SyncManager, error) {
	cfg, err := LoadSyncConfig(dataDir)
	if err != nil {
		return nil, fmt.Errorf("loading sync config: %w", err)
	}

	m := &SyncManager{
		repos:        make(map[string]*git.GitRepo),
		config:       cfg,
		dataDir:      dataDir,
		workspaceDir: workspaceDir,
		store:        s,
		indexer:      idx,
		stopCh:       make(chan struct{}),
		lastPush:     make(map[string]time.Time),
		lastError:    make(map[string]string),
		lastSync:     make(map[string]time.Time),
	}

	// Open the workspace root repo if configured.
	for _, rc := range cfg.Remotes {
		if rc.Path == "." {
			repo, err := git.Open(workspaceDir)
			if err != nil {
				log.Printf("sync: could not open workspace repo: %v", err)
				m.lastError["."] = err.Error()
			} else {
				m.repos["."] = repo
			}
		} else {
			log.Printf("sync: ignoring legacy per-directory remote: %s", rc.Path)
		}
	}

	return m, nil
}

// Start launches the background push goroutine.
func (m *SyncManager) Start() {
	go m.backgroundLoop()
}

// Stop signals the background goroutine to stop and does a final push.
func (m *SyncManager) Stop() {
	close(m.stopCh)

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rc := range m.config.Remotes {
		if rc.Path != "." || !rc.AutoPush {
			continue
		}
		repo, ok := m.repos["."]
		if !ok {
			continue
		}
		if err := repo.Push(); err != nil {
			log.Printf("sync: final push failed: %v", err)
		}
	}
}

// backgroundLoop ticks every minute and pushes if the interval has elapsed.
func (m *SyncManager) backgroundLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.mu.Lock()
			for _, rc := range m.config.Remotes {
				if rc.Path != "." || !rc.AutoPush || rc.PushIntervalMinutes <= 0 {
					continue
				}
				repo, ok := m.repos["."]
				if !ok {
					continue
				}
				last := m.lastPush["."]
				if time.Since(last) < time.Duration(rc.PushIntervalMinutes)*time.Minute {
					continue
				}
				if err := repo.Push(); err != nil {
					m.lastError["."] = err.Error()
					log.Printf("sync: auto-push failed: %v", err)
				} else {
					m.lastError["."] = ""
					m.lastPush["."] = time.Now()
					m.lastSync["."] = time.Now()
				}
			}
			m.mu.Unlock()
		}
	}
}

// SyncAll pulls then pushes the workspace repo.
func (m *SyncManager) SyncAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.syncRepo()
}

// SyncDir pulls and pushes the workspace repo. Path must be ".".
func (m *SyncManager) SyncDir(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.syncRepo()
}

// syncRepo does pull + push for the workspace repo. Must be called with mu held.
func (m *SyncManager) syncRepo() error {
	repo, ok := m.repos["."]
	if !ok {
		return fmt.Errorf("no workspace repo configured")
	}

	m.updateRootGitignore()
	if err := repo.AddAll(); err != nil {
		log.Printf("sync: add-all before sync failed: %v", err)
	}
	if err := repo.Commit("sync workspace"); err != nil {
		log.Printf("sync: commit before sync failed: %v", err)
	}

	if err := repo.Pull(); err != nil {
		m.lastError["."] = err.Error()
		return fmt.Errorf("pull: %w", err)
	}

	// Reindex after pull to pick up any changes from remote.
	if err := m.store.RebuildIndex(); err != nil {
		log.Printf("sync: reindex after pull failed: %v", err)
	} else if err := m.indexer.Reindex(context.Background(), m.store); err != nil {
		log.Printf("sync: reindex after pull failed: %v", err)
	}

	if err := repo.Push(); err != nil {
		m.lastError["."] = err.Error()
		return fmt.Errorf("push: %w", err)
	}

	m.lastError["."] = ""
	m.lastPush["."] = time.Now()
	m.lastSync["."] = time.Now()
	return nil
}

// NotifyChange is called by API handlers after a file is saved or deleted.
// It auto-commits the change if auto-commit is enabled and the file's
// top-level directory is not excluded.
func (m *SyncManager) NotifyChange(filePath, action string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	repo, ok := m.repos["."]
	if !ok {
		return
	}

	// Check if the file's top-level directory is excluded.
	topDir := topLevelDir(filePath)
	if topDir != "" && m.config.IsExcluded(topDir) {
		return
	}

	// Find the config for the root repo.
	var rc *RemoteConfig
	for i := range m.config.Remotes {
		if m.config.Remotes[i].Path == "." {
			rc = &m.config.Remotes[i]
			break
		}
	}
	if rc == nil || !rc.AutoCommit {
		return
	}

	filename := filepath.Base(filePath)
	if err := repo.Add(filePath); err != nil {
		log.Printf("sync: auto-add %s failed: %v", filePath, err)
		m.lastError["."] = err.Error()
		return
	}

	msg := fmt.Sprintf("%s %s", action, filename)
	if err := repo.Commit(msg); err != nil {
		log.Printf("sync: auto-commit %s failed: %v", filePath, err)
		m.lastError["."] = err.Error()
	}
}

// Status returns sync status for the workspace repo.
func (m *SyncManager) Status() map[string]DirSyncStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string]DirSyncStatus)
	for _, rc := range m.config.Remotes {
		if rc.Path == "." {
			result["."] = m.dirStatus(rc)
		}
	}
	return result
}

// DirStatus returns sync status for the workspace repo.
func (m *SyncManager) DirStatus(path string) (DirSyncStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, rc := range m.config.Remotes {
		if rc.Path == "." {
			return m.dirStatus(rc), nil
		}
	}
	return DirSyncStatus{}, fmt.Errorf("no workspace repo configured")
}

// dirStatus builds a DirSyncStatus for one remote config. Must be called with mu held.
func (m *SyncManager) dirStatus(rc RemoteConfig) DirSyncStatus {
	ds := DirSyncStatus{
		Path:     rc.Path,
		URL:      rc.URL,
		Branch:   rc.Branch,
		LastSync: m.lastSync[rc.Path],
		Error:    m.lastError[rc.Path],
	}

	repo, ok := m.repos[rc.Path]
	if !ok {
		if ds.Error == "" {
			ds.Error = "repo not open"
		}
		return ds
	}

	status, err := repo.Status()
	if err != nil {
		ds.Error = err.Error()
		return ds
	}
	ds.Clean = status.Clean
	ds.Ahead = status.Ahead
	ds.Behind = status.Behind
	return ds
}

// AddRemote configures the workspace sync. Only path "." is accepted.
func (m *SyncManager) AddRemote(rc RemoteConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	rc.Path = "."

	// Check for duplicate.
	for _, existing := range m.config.Remotes {
		if existing.Path == "." {
			return fmt.Errorf("workspace sync already configured")
		}
	}

	// Init in workspace root (never clone, content already exists).
	repo, err := git.Init(m.workspaceDir)
	if err != nil {
		return fmt.Errorf("initializing workspace repo: %w", err)
	}
	if rc.URL != "" {
		if err := repo.SetRemote(rc.URL); err != nil {
			return fmt.Errorf("setting remote: %w", err)
		}
	}
	if rc.Branch == "" {
		rc.Branch = "main"
	}
	if err := repo.SetBranch(rc.Branch); err != nil {
		log.Printf("sync: failed to set branch to %s: %v", rc.Branch, err)
	}

	m.repos["."] = repo
	m.config.Remotes = append(m.config.Remotes, rc)

	if err := SaveSyncConfig(m.dataDir, m.config); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Build initial .gitignore and commit everything.
	m.updateRootGitignore()
	if err := repo.AddAll(); err != nil {
		log.Printf("sync: initial add-all failed: %v", err)
	}
	if err := repo.Commit("Initial commit"); err != nil {
		log.Printf("sync: initial commit failed: %v", err)
	}
	return nil
}

// RemoveRemote removes the workspace sync. Only path "." is accepted.
func (m *SyncManager) RemoveRemote(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	remotes := make([]RemoteConfig, 0, len(m.config.Remotes))
	for _, rc := range m.config.Remotes {
		if rc.Path == "." {
			found = true
			continue
		}
		remotes = append(remotes, rc)
	}
	if !found {
		return fmt.Errorf("no workspace sync configured")
	}

	m.config.Remotes = remotes
	m.config.ExcludedDirs = nil
	delete(m.repos, ".")
	delete(m.lastPush, ".")
	delete(m.lastError, ".")
	delete(m.lastSync, ".")

	// Remove .git directory.
	gitDir := filepath.Join(m.workspaceDir, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		log.Printf("sync: failed to remove .git: %v", err)
	}

	// Remove .gitignore managed block.
	m.cleanupGitignore()

	return SaveSyncConfig(m.dataDir, m.config)
}

// UpdateRemote updates the configuration for the workspace sync.
func (m *SyncManager) UpdateRemote(path string, rc RemoteConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, existing := range m.config.Remotes {
		if existing.Path == "." {
			rc.Path = "."

			// If URL changed, update the git remote.
			if rc.URL != "" && rc.URL != existing.URL {
				repo, ok := m.repos["."]
				if ok {
					if err := repo.SetRemote(rc.URL); err != nil {
						return fmt.Errorf("updating remote URL: %w", err)
					}
				}
			}

			m.config.Remotes[i] = rc
			return SaveSyncConfig(m.dataDir, m.config)
		}
	}
	return fmt.Errorf("no workspace sync configured")
}

// ManualCommit stages all changes and commits with the given message.
func (m *SyncManager) ManualCommit(path, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	repo, ok := m.repos["."]
	if !ok {
		return fmt.Errorf("no workspace repo configured")
	}

	if err := repo.AddAll(); err != nil {
		return fmt.Errorf("staging changes: %w", err)
	}
	if err := repo.Commit(message); err != nil {
		return fmt.Errorf("committing: %w", err)
	}
	m.lastError["."] = ""
	return nil
}

// Log returns the last n git commits for the workspace repo.
func (m *SyncManager) Log(path string, n int) ([]git.Commit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	repo, ok := m.repos["."]
	if !ok {
		return nil, fmt.Errorf("no workspace repo configured")
	}
	return repo.Log(n)
}

// Remotes returns the current list of configured remotes.
func (m *SyncManager) Remotes() []RemoteConfig {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]RemoteConfig, len(m.config.Remotes))
	copy(out, m.config.Remotes)
	return out
}

// ExcludeDir excludes a directory from git tracking.
func (m *SyncManager) ExcludeDir(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Validate: simple name, no slashes.
	if dir == "" || strings.Contains(dir, "/") || dir == "." || dir == ".." {
		return fmt.Errorf("invalid directory name: %s", dir)
	}

	repo, ok := m.repos["."]
	if !ok {
		return fmt.Errorf("no workspace repo configured")
	}

	m.config.AddExcludedDir(dir)
	if err := SaveSyncConfig(m.dataDir, m.config); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	m.updateRootGitignore()

	// Remove from git index (graceful if not tracked).
	if err := repo.RemoveCached(dir); err != nil {
		log.Printf("sync: RemoveCached %s failed: %v", dir, err)
	}

	if err := repo.Add(".gitignore"); err != nil {
		return fmt.Errorf("staging .gitignore: %w", err)
	}
	if err := repo.Commit("exclude " + dir); err != nil {
		log.Printf("sync: commit exclude %s: %v", dir, err)
	}
	return nil
}

// IncludeDir re-includes a previously excluded directory in git tracking.
func (m *SyncManager) IncludeDir(dir string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if dir == "" || strings.Contains(dir, "/") || dir == "." || dir == ".." {
		return fmt.Errorf("invalid directory name: %s", dir)
	}

	repo, ok := m.repos["."]
	if !ok {
		return fmt.Errorf("no workspace repo configured")
	}

	m.config.RemoveExcludedDir(dir)
	if err := SaveSyncConfig(m.dataDir, m.config); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	m.updateRootGitignore()

	if err := repo.Add(dir); err != nil {
		log.Printf("sync: add %s failed: %v", dir, err)
	}
	if err := repo.Add(".gitignore"); err != nil {
		return fmt.Errorf("staging .gitignore: %w", err)
	}
	if err := repo.Commit("include " + dir); err != nil {
		log.Printf("sync: commit include %s: %v", dir, err)
	}
	return nil
}

// ExcludedDirs returns a copy of the excluded directories list.
func (m *SyncManager) ExcludedDirs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, len(m.config.ExcludedDirs))
	copy(out, m.config.ExcludedDirs)
	return out
}

// NotifyDirDelete is called after a directory has been removed from disk.
// It stages all changes (the deletions) and commits if auto-commit is enabled.
func (m *SyncManager) NotifyDirDelete(dirPath string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	repo, ok := m.repos["."]
	if !ok {
		return
	}

	// Check if excluded — if so, git doesn't track it anyway.
	topDir := topLevelDir(dirPath + "/x") // ensure we get the top-level component
	if topDir == "" {
		topDir = dirPath
	}
	if m.config.IsExcluded(topDir) {
		return
	}

	var rc *RemoteConfig
	for i := range m.config.Remotes {
		if m.config.Remotes[i].Path == "." {
			rc = &m.config.Remotes[i]
			break
		}
	}
	if rc == nil || !rc.AutoCommit {
		return
	}

	if err := repo.AddAll(); err != nil {
		log.Printf("sync: auto-add after dir delete %s failed: %v", dirPath, err)
		m.lastError["."] = err.Error()
		return
	}
	if err := repo.Commit("delete " + dirPath); err != nil {
		log.Printf("sync: auto-commit after dir delete %s failed: %v", dirPath, err)
		m.lastError["."] = err.Error()
	}
}

// topLevelDir extracts the first path component from a relative file path.
// Returns empty string if the file is in the root directory.
func topLevelDir(filePath string) string {
	clean := filepath.ToSlash(filepath.Clean(filePath))
	parts := strings.SplitN(clean, "/", 2)
	if len(parts) < 2 {
		return "" // file is in root, no directory
	}
	return parts[0]
}

// updateRootGitignore refreshes the managed block in the workspace-root
// .gitignore based on the ExcludedDirs config.
func (m *SyncManager) updateRootGitignore() {
	if _, ok := m.repos["."]; !ok {
		return
	}

	// Build sorted list of gitignore entries from excluded dirs.
	var lines []string
	for _, dir := range m.config.ExcludedDirs {
		lines = append(lines, dir+"/")
	}
	sort.Strings(lines)

	managedBlock := "# BEGIN ARCHIVARY MANAGED\n"
	for _, line := range lines {
		managedBlock += line + "\n"
	}
	managedBlock += "# END ARCHIVARY MANAGED"

	// Read existing .gitignore.
	ignorePath := filepath.Join(m.workspaceDir, ".gitignore")
	existing, _ := os.ReadFile(ignorePath)
	content := string(existing)

	const beginMarker = "# BEGIN ARCHIVARY MANAGED"
	const endMarker = "# END ARCHIVARY MANAGED"

	beginIdx := strings.Index(content, beginMarker)
	endIdx := strings.Index(content, endMarker)

	if beginIdx >= 0 && endIdx >= 0 {
		content = content[:beginIdx] + managedBlock + content[endIdx+len(endMarker):]
	} else {
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content != "" {
			content += "\n"
		}
		content += managedBlock + "\n"
	}

	if err := os.WriteFile(ignorePath, []byte(content), 0o644); err != nil {
		log.Printf("sync: failed to write .gitignore: %v", err)
	}
}

// cleanupGitignore removes the managed block from .gitignore when unsyncing.
func (m *SyncManager) cleanupGitignore() {
	ignorePath := filepath.Join(m.workspaceDir, ".gitignore")
	existing, err := os.ReadFile(ignorePath)
	if err != nil {
		return
	}
	content := string(existing)

	const beginMarker = "# BEGIN ARCHIVARY MANAGED"
	const endMarker = "# END ARCHIVARY MANAGED"

	beginIdx := strings.Index(content, beginMarker)
	endIdx := strings.Index(content, endMarker)

	if beginIdx >= 0 && endIdx >= 0 {
		// Remove the managed block and any surrounding blank lines.
		before := strings.TrimRight(content[:beginIdx], "\n")
		after := strings.TrimLeft(content[endIdx+len(endMarker):], "\n")
		content = before
		if after != "" {
			if content != "" {
				content += "\n\n"
			}
			content += after
		}
		if content != "" && !strings.HasSuffix(content, "\n") {
			content += "\n"
		}
		if content == "\n" {
			// File is now empty, remove it.
			os.Remove(ignorePath)
			return
		}
		os.WriteFile(ignorePath, []byte(content), 0o644)
	}
}

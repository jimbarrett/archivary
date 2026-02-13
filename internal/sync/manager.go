package sync

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

// SyncManager orchestrates git sync for workspace directories.
type SyncManager struct {
	repos        map[string]*git.GitRepo
	config       *SyncConfig
	dataDir      string
	workspaceDir string
	store        *store.FileStore
	indexer      *index.Indexer
	stopCh       chan struct{}
	mu           gosync.Mutex

	// lastPush tracks the last push time per repo path.
	lastPush map[string]time.Time
	// lastError tracks the last error per repo path.
	lastError map[string]string
	// lastSync tracks the last successful sync time per repo path.
	lastSync map[string]time.Time
}

// NewSyncManager creates a SyncManager, loads existing config, and opens any
// already-configured repos.
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
		indexer:       idx,
		stopCh:       make(chan struct{}),
		lastPush:     make(map[string]time.Time),
		lastError:    make(map[string]string),
		lastSync:     make(map[string]time.Time),
	}

	// Open repos for existing config entries.
	for _, rc := range cfg.Remotes {
		dir := filepath.Join(workspaceDir, rc.Path)
		repo, err := git.Open(dir)
		if err != nil {
			log.Printf("sync: could not open repo at %s: %v", rc.Path, err)
			m.lastError[rc.Path] = err.Error()
			continue
		}
		m.repos[rc.Path] = repo
	}

	return m, nil
}

// Start launches the background push goroutine.
func (m *SyncManager) Start() {
	go m.backgroundLoop()
}

// Stop signals the background goroutine to stop and does a final push of all
// repos that have auto-push enabled.
func (m *SyncManager) Stop() {
	close(m.stopCh)

	m.mu.Lock()
	defer m.mu.Unlock()
	for _, rc := range m.config.Remotes {
		if !rc.AutoPush {
			continue
		}
		repo, ok := m.repos[rc.Path]
		if !ok {
			continue
		}
		if err := repo.Push(); err != nil {
			log.Printf("sync: final push for %s failed: %v", rc.Path, err)
		}
	}
}

// backgroundLoop ticks every minute and pushes repos whose interval has elapsed.
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
				if !rc.AutoPush || rc.PushIntervalMinutes <= 0 {
					continue
				}
				repo, ok := m.repos[rc.Path]
				if !ok {
					continue
				}
				last := m.lastPush[rc.Path]
				if time.Since(last) < time.Duration(rc.PushIntervalMinutes)*time.Minute {
					continue
				}
				if err := repo.Push(); err != nil {
					m.lastError[rc.Path] = err.Error()
					log.Printf("sync: auto-push %s failed: %v", rc.Path, err)
				} else {
					m.lastError[rc.Path] = ""
					m.lastPush[rc.Path] = time.Now()
					m.lastSync[rc.Path] = time.Now()
				}
			}
			m.mu.Unlock()
		}
	}
}

// SyncAll pulls then pushes all configured repos.
func (m *SyncManager) SyncAll() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var errs []string
	for _, rc := range m.config.Remotes {
		if err := m.syncRepo(rc.Path); err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", rc.Path, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("sync errors: %s", strings.Join(errs, "; "))
	}
	return nil
}

// SyncDir pulls and pushes a single directory.
func (m *SyncManager) SyncDir(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.syncRepo(path)
}

// syncRepo does pull + push for one repo. Must be called with mu held.
func (m *SyncManager) syncRepo(path string) error {
	repo, ok := m.repos[path]
	if !ok {
		return fmt.Errorf("no synced repo at path: %s", path)
	}

	if err := repo.Pull(); err != nil {
		m.lastError[path] = err.Error()
		return fmt.Errorf("pull: %w", err)
	}

	// Reindex after pull to pick up any changes from remote.
	if err := m.store.RebuildIndex(); err != nil {
		log.Printf("sync: reindex after pull for %s failed: %v", path, err)
	} else if err := m.indexer.Reindex(context.Background(), m.store); err != nil {
		log.Printf("sync: reindex after pull for %s failed: %v", path, err)
	}

	if err := repo.Push(); err != nil {
		m.lastError[path] = err.Error()
		return fmt.Errorf("push: %w", err)
	}

	m.lastError[path] = ""
	m.lastPush[path] = time.Now()
	m.lastSync[path] = time.Now()
	return nil
}

// NotifyChange is called by API handlers after a file is saved or deleted.
// It auto-commits the change if the file is inside a synced directory with
// auto-commit enabled.
func (m *SyncManager) NotifyChange(filePath, action string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Extract the top-level directory from the file path.
	// e.g. "work/notes/foo.md" -> "work"
	topDir := topLevelDir(filePath)
	if topDir == "" {
		return // file is in workspace root, not in a synced dir
	}

	repo, ok := m.repos[topDir]
	if !ok {
		return
	}

	// Find the config for this repo.
	var rc *RemoteConfig
	for i := range m.config.Remotes {
		if m.config.Remotes[i].Path == topDir {
			rc = &m.config.Remotes[i]
			break
		}
	}
	if rc == nil || !rc.AutoCommit {
		return
	}

	// Convert workspace-relative path to repo-relative path.
	// e.g. "work/notes/foo.md" with topDir "work" -> "notes/foo.md"
	repoRelPath := strings.TrimPrefix(filePath, topDir+"/")
	filename := filepath.Base(filePath)
	if err := repo.Add(repoRelPath); err != nil {
		log.Printf("sync: auto-add %s failed: %v", filePath, err)
		m.lastError[topDir] = err.Error()
		return
	}

	msg := fmt.Sprintf("%s %s", action, filename)
	if err := repo.Commit(msg); err != nil {
		log.Printf("sync: auto-commit %s failed: %v", filePath, err)
		m.lastError[topDir] = err.Error()
	}
}

// Status returns sync status for all configured directories.
func (m *SyncManager) Status() map[string]DirSyncStatus {
	m.mu.Lock()
	defer m.mu.Unlock()

	result := make(map[string]DirSyncStatus, len(m.config.Remotes))
	for _, rc := range m.config.Remotes {
		result[rc.Path] = m.dirStatus(rc)
	}
	return result
}

// DirStatus returns sync status for a single directory.
func (m *SyncManager) DirStatus(path string) (DirSyncStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, rc := range m.config.Remotes {
		if rc.Path == path {
			return m.dirStatus(rc), nil
		}
	}
	return DirSyncStatus{}, fmt.Errorf("no synced repo at path: %s", path)
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

// AddRemote configures a new synced directory. If URL is provided, it clones
// the repo; otherwise it initializes a new repo in the workspace directory.
func (m *SyncManager) AddRemote(rc RemoteConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for duplicate.
	for _, existing := range m.config.Remotes {
		if existing.Path == rc.Path {
			return fmt.Errorf("remote already configured for path: %s", rc.Path)
		}
	}

	dir := filepath.Join(m.workspaceDir, rc.Path)
	var repo *git.GitRepo
	var err error

	if rc.URL != "" {
		repo, err = git.Clone(rc.URL, dir)
		if err != nil {
			return fmt.Errorf("cloning %s: %w", rc.URL, err)
		}
	} else {
		repo, err = git.Init(dir)
		if err != nil {
			return fmt.Errorf("initializing repo at %s: %w", rc.Path, err)
		}
	}

	if rc.Branch == "" {
		rc.Branch = "main"
	}

	m.repos[rc.Path] = repo
	m.config.Remotes = append(m.config.Remotes, rc)

	if err := SaveSyncConfig(m.dataDir, m.config); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	// Reindex to pick up any files from clone.
	if rc.URL != "" {
		if err := m.store.RebuildIndex(); err != nil {
			log.Printf("sync: reindex after clone for %s failed: %v", rc.Path, err)
		} else if err := m.indexer.Reindex(context.Background(), m.store); err != nil {
			log.Printf("sync: reindex after clone for %s failed: %v", rc.Path, err)
		}
	}

	return nil
}

// RemoveRemote removes a directory from sync config and deletes the .git
// directory, turning it back into a normal unsynced folder. Files are kept.
func (m *SyncManager) RemoveRemote(path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	found := false
	remotes := make([]RemoteConfig, 0, len(m.config.Remotes))
	for _, rc := range m.config.Remotes {
		if rc.Path == path {
			found = true
			continue
		}
		remotes = append(remotes, rc)
	}
	if !found {
		return fmt.Errorf("no remote configured for path: %s", path)
	}

	m.config.Remotes = remotes
	delete(m.repos, path)
	delete(m.lastPush, path)
	delete(m.lastError, path)
	delete(m.lastSync, path)

	// Remove .git directory to fully disconnect from git.
	gitDir := filepath.Join(m.workspaceDir, path, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		log.Printf("sync: failed to remove .git for %s: %v", path, err)
	}

	return SaveSyncConfig(m.dataDir, m.config)
}

// UpdateRemote updates the configuration for an existing synced directory.
func (m *SyncManager) UpdateRemote(path string, rc RemoteConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for i, existing := range m.config.Remotes {
		if existing.Path == path {
			// Preserve the path — it can't be changed.
			rc.Path = path

			// If URL changed, update the git remote.
			if rc.URL != "" && rc.URL != existing.URL {
				repo, ok := m.repos[path]
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
	return fmt.Errorf("no remote configured for path: %s", path)
}

// ManualCommit stages all changes and commits with the given message.
// Returns nil if there is nothing to commit.
func (m *SyncManager) ManualCommit(path, message string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	repo, ok := m.repos[path]
	if !ok {
		return fmt.Errorf("no synced repo at path: %s", path)
	}

	if err := repo.AddAll(); err != nil {
		return fmt.Errorf("staging changes: %w", err)
	}
	if err := repo.Commit(message); err != nil {
		return fmt.Errorf("committing: %w", err)
	}
	m.lastError[path] = ""
	return nil
}

// Log returns the last n git commits for a synced directory.
func (m *SyncManager) Log(path string, n int) ([]git.Commit, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	repo, ok := m.repos[path]
	if !ok {
		return nil, fmt.Errorf("no synced repo at path: %s", path)
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

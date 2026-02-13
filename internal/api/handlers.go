package api

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/jimbarrett/archivary/internal/git"
	"github.com/jimbarrett/archivary/internal/index"
	"github.com/jimbarrett/archivary/internal/store"
	"github.com/jimbarrett/archivary/internal/sync"
)

type handlers struct {
	store   *store.FileStore
	indexer *index.Indexer
	sync    *sync.SyncManager
}

// apiError is a consistent JSON error response.
type apiError struct {
	Error string `json:"error"`
}

func errJSON(c echo.Context, status int, msg string) error {
	return c.JSON(status, apiError{Error: msg})
}

// GET /api/health
func (h *handlers) health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/pages?dir=optional/prefix
func (h *handlers) listPages(c echo.Context) error {
	dir := c.QueryParam("dir")
	pages, err := h.store.ListPages(c.Request().Context(), dir)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}
	if pages == nil {
		pages = []*store.Page{}
	}
	return c.JSON(http.StatusOK, pages)
}

// GET /api/pages/:id
func (h *handlers) getPage(c echo.Context) error {
	id := c.Param("id")
	page, err := h.store.GetPage(c.Request().Context(), id)
	if err != nil {
		return errJSON(c, http.StatusNotFound, "page not found")
	}
	return c.JSON(http.StatusOK, page)
}

// createPageRequest is the JSON body for creating a new page.
type createPageRequest struct {
	Title   string `json:"title"`
	Content string `json:"content"`
	Path    string `json:"path"`
}

// POST /api/pages
func (h *handlers) createPage(c echo.Context) error {
	var req createPageRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}

	if req.Path == "" {
		return errJSON(c, http.StatusBadRequest, "path is required")
	}

	page := &store.Page{
		Content: req.Content,
		Path:    req.Path,
	}

	// SavePage will generate an ID via EnsureID
	if err := h.store.SavePage(c.Request().Context(), page); err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}

	// Read back the full page (with derived title, timestamps, etc.)
	saved, err := h.store.GetPage(c.Request().Context(), page.ID)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}

	// Index the new page
	if err := h.indexer.IndexPage(c.Request().Context(), saved); err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}

	if h.sync != nil {
		h.sync.NotifyChange(saved.Path, "create")
	}

	return c.JSON(http.StatusCreated, saved)
}

// updatePageRequest is the JSON body for updating a page.
type updatePageRequest struct {
	Content string `json:"content"`
	Path    string `json:"path,omitempty"`
}

// PUT /api/pages/:id
func (h *handlers) updatePage(c echo.Context) error {
	id := c.Param("id")

	// Verify the page exists
	existing, err := h.store.GetPage(c.Request().Context(), id)
	if err != nil {
		return errJSON(c, http.StatusNotFound, "page not found")
	}

	var req updatePageRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}

	path := existing.Path
	if req.Path != "" {
		path = req.Path
	}

	page := &store.Page{
		ID:      id,
		Content: req.Content,
		Path:    path,
	}

	if err := h.store.SavePage(c.Request().Context(), page); err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}

	// Read back the full page
	saved, err := h.store.GetPage(c.Request().Context(), id)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}

	// Update the index
	if err := h.indexer.IndexPage(c.Request().Context(), saved); err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}

	if h.sync != nil {
		h.sync.NotifyChange(saved.Path, "update")
	}

	return c.JSON(http.StatusOK, saved)
}

// DELETE /api/pages/:id
func (h *handlers) deletePage(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	// Get the page path before deleting (needed for sync notification).
	var pagePath string
	if h.sync != nil {
		if page, err := h.store.GetPage(ctx, id); err == nil {
			pagePath = page.Path
		}
	}

	if err := h.store.DeletePage(ctx, id); err != nil {
		return errJSON(c, http.StatusNotFound, "page not found")
	}

	// Remove from index
	if err := h.indexer.RemovePage(ctx, id); err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}

	if h.sync != nil && pagePath != "" {
		h.sync.NotifyChange(pagePath, "delete")
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// GET /api/search?q=query
func (h *handlers) search(c echo.Context) error {
	q := c.QueryParam("q")
	if q == "" {
		return c.JSON(http.StatusOK, []index.SearchResult{})
	}

	results, err := h.indexer.Search(c.Request().Context(), q)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}
	if results == nil {
		results = []index.SearchResult{}
	}
	return c.JSON(http.StatusOK, results)
}

// GET /api/tree
func (h *handlers) getTree(c echo.Context) error {
	tree, err := h.store.BuildTree(c.Request().Context())
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, tree)
}

// GET /api/pages/:id/backlinks
func (h *handlers) getBacklinks(c echo.Context) error {
	id := c.Param("id")
	backlinks, err := h.indexer.GetBacklinks(c.Request().Context(), id)
	if err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}
	if backlinks == nil {
		backlinks = []store.Page{}
	}
	return c.JSON(http.StatusOK, backlinks)
}

// renameDirRequest is the JSON body for renaming a directory.
type renameDirRequest struct {
	Name string `json:"name"`
}

// PUT /api/dirs/*
func (h *handlers) renameDir(c echo.Context) error {
	dirPath := c.Param("*")
	if dirPath == "" {
		return errJSON(c, http.StatusBadRequest, "directory path is required")
	}

	var req renameDirRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}
	if req.Name == "" {
		return errJSON(c, http.StatusBadRequest, "name is required")
	}

	if err := h.store.RenameDir(c.Request().Context(), dirPath, req.Name); err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "renamed"})
}

// DELETE /api/dirs/*
func (h *handlers) deleteDir(c echo.Context) error {
	dirPath := c.Param("*")
	if dirPath == "" {
		return errJSON(c, http.StatusBadRequest, "directory path is required")
	}

	if err := h.store.DeleteDir(c.Request().Context(), dirPath); err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "deleted"})
}

// POST /api/reindex
func (h *handlers) reindex(c echo.Context) error {
	ctx := c.Request().Context()

	if err := h.store.RebuildIndex(); err != nil {
		return errJSON(c, http.StatusInternalServerError, "rebuilding file index: "+err.Error())
	}

	if err := h.indexer.Reindex(ctx, h.store); err != nil {
		return errJSON(c, http.StatusInternalServerError, "reindexing: "+err.Error())
	}

	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/pages/check-path?path=some/file.md
func (h *handlers) checkPath(c echo.Context) error {
	path := c.QueryParam("path")
	if path == "" {
		return errJSON(c, http.StatusBadRequest, "path query parameter is required")
	}

	exists := h.store.PathExists(path)
	return c.JSON(http.StatusOK, map[string]bool{"exists": exists})
}

// --- Sync handlers ---

// addRemoteRequest is the JSON body for adding a sync remote.
type addRemoteRequest struct {
	URL                 string `json:"url"`
	Path                string `json:"path"`
	Branch              string `json:"branch"`
	AutoCommit          bool   `json:"auto_commit"`
	AutoPush            bool   `json:"auto_push"`
	PushIntervalMinutes int    `json:"push_interval_minutes"`
}

// updateRemoteRequest is the JSON body for updating a sync remote.
type updateRemoteRequest struct {
	URL                 string `json:"url"`
	Branch              string `json:"branch"`
	AutoCommit          *bool  `json:"auto_commit"`
	AutoPush            *bool  `json:"auto_push"`
	PushIntervalMinutes *int   `json:"push_interval_minutes"`
}

// GET /api/sync/status
func (h *handlers) syncStatus(c echo.Context) error {
	if h.sync == nil {
		return c.JSON(http.StatusOK, map[string]sync.DirSyncStatus{})
	}
	return c.JSON(http.StatusOK, h.sync.Status())
}

// GET /api/sync/status/:path
func (h *handlers) syncDirStatus(c echo.Context) error {
	if h.sync == nil {
		return errJSON(c, http.StatusNotFound, "sync not configured")
	}
	path := c.Param("path")
	status, err := h.sync.DirStatus(path)
	if err != nil {
		return errJSON(c, http.StatusNotFound, err.Error())
	}
	return c.JSON(http.StatusOK, status)
}

// POST /api/sync/now
func (h *handlers) syncNow(c echo.Context) error {
	if h.sync == nil {
		return errJSON(c, http.StatusBadRequest, "sync not configured")
	}
	if err := h.sync.SyncAll(); err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// POST /api/sync/now/:path
func (h *handlers) syncNowDir(c echo.Context) error {
	if h.sync == nil {
		return errJSON(c, http.StatusBadRequest, "sync not configured")
	}
	path := c.Param("path")
	if err := h.sync.SyncDir(path); err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

// GET /api/sync/remotes
func (h *handlers) listRemotes(c echo.Context) error {
	if h.sync == nil {
		return c.JSON(http.StatusOK, []sync.RemoteConfig{})
	}
	remotes := h.sync.Remotes()
	if remotes == nil {
		remotes = []sync.RemoteConfig{}
	}
	return c.JSON(http.StatusOK, remotes)
}

// POST /api/sync/remotes
func (h *handlers) addRemote(c echo.Context) error {
	if h.sync == nil {
		return errJSON(c, http.StatusBadRequest, "sync not configured")
	}
	var req addRemoteRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}
	if req.Path == "" {
		return errJSON(c, http.StatusBadRequest, "path is required")
	}

	rc := sync.RemoteConfig{
		Path:                req.Path,
		URL:                 req.URL,
		Branch:              req.Branch,
		AutoCommit:          req.AutoCommit,
		AutoPush:            req.AutoPush,
		PushIntervalMinutes: req.PushIntervalMinutes,
	}
	if err := h.sync.AddRemote(rc); err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusCreated, rc)
}

// PUT /api/sync/remotes/:path
func (h *handlers) updateRemote(c echo.Context) error {
	if h.sync == nil {
		return errJSON(c, http.StatusBadRequest, "sync not configured")
	}
	path := c.Param("path")

	var req updateRemoteRequest
	if err := c.Bind(&req); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}

	// Get existing config to merge partial updates.
	status, err := h.sync.DirStatus(path)
	if err != nil {
		return errJSON(c, http.StatusNotFound, err.Error())
	}

	rc := sync.RemoteConfig{
		Path:   path,
		URL:    status.URL,
		Branch: status.Branch,
	}
	if req.URL != "" {
		rc.URL = req.URL
	}
	if req.Branch != "" {
		rc.Branch = req.Branch
	}

	// For boolean/int pointer fields, use the value if provided.
	existing := h.sync.Remotes()
	for _, e := range existing {
		if e.Path == path {
			rc.AutoCommit = e.AutoCommit
			rc.AutoPush = e.AutoPush
			rc.PushIntervalMinutes = e.PushIntervalMinutes
			break
		}
	}
	if req.AutoCommit != nil {
		rc.AutoCommit = *req.AutoCommit
	}
	if req.AutoPush != nil {
		rc.AutoPush = *req.AutoPush
	}
	if req.PushIntervalMinutes != nil {
		rc.PushIntervalMinutes = *req.PushIntervalMinutes
	}

	if err := h.sync.UpdateRemote(path, rc); err != nil {
		return errJSON(c, http.StatusBadRequest, err.Error())
	}
	return c.JSON(http.StatusOK, rc)
}

// DELETE /api/sync/remotes/:path
func (h *handlers) removeRemote(c echo.Context) error {
	if h.sync == nil {
		return errJSON(c, http.StatusBadRequest, "sync not configured")
	}
	path := c.Param("path")
	if err := h.sync.RemoveRemote(path); err != nil {
		return errJSON(c, http.StatusNotFound, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "removed"})
}

// POST /api/sync/commit/:path
func (h *handlers) syncCommit(c echo.Context) error {
	if h.sync == nil {
		return errJSON(c, http.StatusBadRequest, "sync not configured")
	}
	path := c.Param("path")
	var body struct {
		Message string `json:"message"`
	}
	if err := c.Bind(&body); err != nil {
		return errJSON(c, http.StatusBadRequest, "invalid request body")
	}
	if body.Message == "" {
		body.Message = "manual commit"
	}
	if err := h.sync.ManualCommit(path, body.Message); err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
	}
	return c.JSON(http.StatusOK, map[string]string{"status": "committed"})
}

// GET /api/sync/log/:path
func (h *handlers) syncLog(c echo.Context) error {
	if h.sync == nil {
		return errJSON(c, http.StatusBadRequest, "sync not configured")
	}
	path := c.Param("path")
	n := 20
	if nStr := c.QueryParam("n"); nStr != "" {
		if parsed, err := strconv.Atoi(nStr); err == nil && parsed > 0 {
			n = parsed
		}
	}
	commits, err := h.sync.Log(path, n)
	if err != nil {
		return errJSON(c, http.StatusNotFound, err.Error())
	}
	if commits == nil {
		commits = []git.Commit{}
	}
	return c.JSON(http.StatusOK, commits)
}

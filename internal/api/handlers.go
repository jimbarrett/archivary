package api

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/jimbarrett/archivary/internal/index"
	"github.com/jimbarrett/archivary/internal/store"
)

type handlers struct {
	store   *store.FileStore
	indexer *index.Indexer
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

	return c.JSON(http.StatusOK, saved)
}

// DELETE /api/pages/:id
func (h *handlers) deletePage(c echo.Context) error {
	id := c.Param("id")
	ctx := c.Request().Context()

	if err := h.store.DeletePage(ctx, id); err != nil {
		return errJSON(c, http.StatusNotFound, "page not found")
	}

	// Remove from index
	if err := h.indexer.RemovePage(ctx, id); err != nil {
		return errJSON(c, http.StatusInternalServerError, err.Error())
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

// GET /api/pages/check-path?path=some/file.md
func (h *handlers) checkPath(c echo.Context) error {
	path := c.QueryParam("path")
	if path == "" {
		return errJSON(c, http.StatusBadRequest, "path query parameter is required")
	}

	exists := h.store.PathExists(path)
	return c.JSON(http.StatusOK, map[string]bool{"exists": exists})
}

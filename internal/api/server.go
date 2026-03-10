package api

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/jimbarrett/archivary/frontend"
	"github.com/jimbarrett/archivary/internal/config"
	"github.com/jimbarrett/archivary/internal/index"
	"github.com/jimbarrett/archivary/internal/store"
	"github.com/jimbarrett/archivary/internal/sync"
)

func StartServer(ctx context.Context, cfg *config.Config, fileStore *store.FileStore, indexer *index.Indexer, syncMgr *sync.SyncManager) error {
	e := echo.New()
	e.HideBanner = true

	e.Use(middleware.Logger())
	e.Use(middleware.Recover())

	// CORS for development (Vite dev server on a different port)
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"http://localhost:5173"},
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete},
	}))

	h := &handlers{
		store:   fileStore,
		indexer: indexer,
		sync:    syncMgr,
	}

	api := e.Group("/api")
	api.GET("/health", h.health)
	api.GET("/pages", h.listPages)
	api.GET("/pages/check-path", h.checkPath)
	api.GET("/pages/:id", h.getPage)
	api.POST("/pages", h.createPage)
	api.PUT("/pages/:id", h.updatePage)
	api.DELETE("/pages/:id", h.deletePage)
	api.POST("/pages/:id/move", h.movePage)
	api.GET("/search", h.search)
	api.GET("/tree", h.getTree)
	api.GET("/pages/:id/backlinks", h.getBacklinks)
	api.POST("/reindex", h.reindex)
	api.POST("/dirs", h.createDir)
	api.PUT("/dirs/*", h.renameDir)
	api.DELETE("/dirs/*", h.deleteDir)

	// Sync routes
	syncAPI := api.Group("/sync")
	syncAPI.GET("/status", h.syncStatus)
	syncAPI.GET("/status/:path", h.syncDirStatus)
	syncAPI.POST("/now", h.syncNow)
	syncAPI.POST("/now/:path", h.syncNowDir)
	syncAPI.GET("/remotes", h.listRemotes)
	syncAPI.POST("/remotes", h.addRemote)
	syncAPI.PUT("/remotes/:path", h.updateRemote)
	syncAPI.DELETE("/remotes/:path", h.removeRemote)
	syncAPI.POST("/commit/:path", h.syncCommit)
	syncAPI.GET("/log/:path", h.syncLog)
	syncAPI.GET("/excluded", h.listExcluded)
	syncAPI.POST("/exclude/:path", h.excludeDir)
	syncAPI.POST("/include/:path", h.includeDir)
	syncAPI.POST("/exclude-file/:path", h.excludeFile)
	syncAPI.POST("/include-file/:path", h.includeFile)

	// Workspace routes
	api.GET("/workspace/entries", h.getWorkspaceEntries)

	// Serve frontend from embedded assets
	serveFrontend(e)

	fmt.Printf("Archivary running at http://localhost:%s\n", cfg.Port)

	// Start server in a goroutine so we can listen for context cancellation
	errCh := make(chan error, 1)
	go func() {
		if err := e.Start(":" + cfg.Port); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*1e9) // 5 seconds
		defer cancel()
		return e.Shutdown(shutdownCtx)
	}
}

func serveFrontend(e *echo.Echo) {
	// The frontend assets are embedded in the binary via go:embed.
	// We sub into the "dist" subdirectory since the embed includes "dist/*".
	distFS, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		// Fallback: this shouldn't happen if the binary was built correctly
		e.GET("/*", func(c echo.Context) error {
			return c.HTML(http.StatusOK, `<!DOCTYPE html>
<html><head><title>Archivary</title></head>
<body><h1>Archivary</h1><p>Frontend assets not found. Rebuild with <code>make build</code>.</p></body>
</html>`)
		})
		return
	}

	fileServer := http.FileServer(http.FS(distFS))

	e.GET("/*", echo.WrapHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "index.html"
		} else if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}

		// If the file doesn't exist, serve index.html for Vue Router
		if _, err := fs.Stat(distFS, path); err != nil {
			r.URL.Path = "/"
		}

		fileServer.ServeHTTP(w, r)
	})))
}

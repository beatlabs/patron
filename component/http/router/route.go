package router

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	patronhttp "github.com/beatlabs/patron/component/http"
)

// NewFileServerRoute returns a route that acts as a file server.
func NewFileServerRoute(path string, assetsDir string, fallbackPath string) (*patronhttp.Route, error) {
	if path == "" {
		return nil, errors.New("path is empty")
	}

	if assetsDir == "" {
		return nil, errors.New("assets path is empty")
	}

	if fallbackPath == "" {
		return nil, errors.New("fallback path is empty")
	}

	_, err := os.Stat(assetsDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("assets directory [%s] doesn't exist", path)
	} else if err != nil {
		return nil, fmt.Errorf("error while checking assets dir: %w", err)
	}

	_, err = os.Stat(fallbackPath)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("fallback file [%s] doesn't exist", fallbackPath)
	} else if err != nil {
		return nil, fmt.Errorf("error while checking fallback file: %w", err)
	}

	absAssetsDir, err := filepath.Abs(assetsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve assets dir: %w", err)
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		filePath := filepath.Join(absAssetsDir, r.PathValue("path"))

		// Prevent directory traversal: filepath.Join resolves ".." so a path
		// escaping absAssetsDir will no longer share its prefix. Adding a
		// separator to filePath before the check handles the edge case where
		// filePath equals absAssetsDir exactly (empty path value).
		if !strings.HasPrefix(filePath+string(filepath.Separator), absAssetsDir+string(filepath.Separator)) {
			http.ServeFile(w, r, fallbackPath)
			return
		}

		info, err := os.Stat(filePath) //nolint:gosec // filePath is validated by the HasPrefix check above
		if err != nil {
			if os.IsNotExist(err) {
				http.ServeFile(w, r, fallbackPath)
				return
			}
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		if info.IsDir() {
			http.ServeFile(w, r, fallbackPath)
			return
		}

		http.ServeFile(w, r, filePath) //nolint:gosec // filePath is validated by the HasPrefix check above
	}

	return patronhttp.NewRoute(path, handler)
}

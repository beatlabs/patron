package httprouter

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	patronhttp "github.com/beatlabs/patron/component/http"
	"github.com/julienschmidt/httprouter"
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

	handler := func(w http.ResponseWriter, r *http.Request) {
		paramPath := ""
		for _, param := range httprouter.ParamsFromContext(r.Context()) {
			if param.Key == "path" {
				paramPath = param.Value
				break
			}
		}

		// get the absolute path to prevent directory traversal
		path := assetsDir + paramPath

		// check whether a file exists at the given path
		info, err := os.Stat(path)
		if os.IsNotExist(err) || info.IsDir() {
			// file does not exist, serve index.html
			http.ServeFile(w, r, fallbackPath)
			return
		} else if err != nil {
			// if we got an error (that wasn't that the file doesn't exist) stating the
			// file, return a 500 internal server error and stop
			http.Error(w, "", http.StatusInternalServerError)
			return
		}

		// otherwise, use server the specific file directly from the filesystem.
		http.ServeFile(w, r, path)
	}

	return patronhttp.NewRoute(http.MethodGet, path, handler)
}

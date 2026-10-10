// Package spa serves the built single-page frontend, falling back to
// index.html for client-side routes and telling the page which URL prefix it
// is mounted under when running behind a reverse proxy.
package spa

import (
	"io/fs"
	"net/http"
	"path"
	"regexp"
	"strings"
)

var baseTag = regexp.MustCompile(`<base\s+href="[^"]*"\s*/?>`)

// NormalizeBasePath returns "" for the root, otherwise a path with a leading
// slash and no trailing slash ("sweep/" -> "/sweep").
func NormalizeBasePath(p string) string {
	p = strings.Trim(strings.TrimSpace(p), "/")
	if p == "" {
		return ""
	}
	return "/" + p
}

// Handler serves root. The reverse proxy is expected to strip basePath from
// request URLs; basePath is only used to rewrite index.html's <base href> so the
// browser builds asset, API and router URLs under the prefix.
func Handler(root fs.FS, basePath string) http.Handler {
	basePath = NormalizeBasePath(basePath)
	files := http.FileServerFS(root)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name != "" {
			if st, err := fs.Stat(root, name); err == nil && !st.IsDir() {
				files.ServeHTTP(w, r)
				return
			}
		}
		serveIndex(w, root, basePath)
	})
}

func serveIndex(w http.ResponseWriter, root fs.FS, basePath string) {
	b, err := fs.ReadFile(root, "index.html")
	if err != nil {
		http.NotFound(w, nil)
		return
	}
	if basePath != "" {
		b = baseTag.ReplaceAll(b, []byte(`<base href="`+basePath+`/" />`))
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(b)
}

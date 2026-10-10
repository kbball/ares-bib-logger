package spa_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/spa"
	"github.com/stretchr/testify/assert"
)

const index = `<html><head><base href="/" /></head><body>app</body></html>`

func newFS() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte(index)},
		"assets/app.js": {Data: []byte("console.log(1)")},
	}
}

func get(h http.Handler, target string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, target, nil))
	return rec
}

func TestNormalizeBasePath(t *testing.T) {
	for in, want := range map[string]string{"": "", "/": "", "bibs": "/bibs", "/bibs/": "/bibs", " /a/b/ ": "/a/b"} {
		assert.Equal(t, want, spa.NormalizeBasePath(in), in)
	}
}

func TestHandler_ServesRealFiles(t *testing.T) {
	rec := get(spa.Handler(newFS(), "/bibs"), "/assets/app.js")
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "console.log(1)", rec.Body.String())
}

func TestHandler_FallsBackToIndexForClientRoutes(t *testing.T) {
	for _, p := range []string{"/", "/events/3", "/assets", "/missing.js"} {
		rec := get(spa.Handler(newFS(), ""), p)
		assert.Equal(t, http.StatusOK, rec.Code, p)
		assert.Equal(t, index, rec.Body.String(), p)
	}
}

func TestHandler_RewritesBaseHref(t *testing.T) {
	rec := get(spa.Handler(newFS(), "bibs/"), "/events/3")
	assert.Contains(t, rec.Body.String(), `<base href="/bibs/" />`)
	assert.NotContains(t, rec.Body.String(), `<base href="/" />`)
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
}

func TestHandler_MissingIndex(t *testing.T) {
	rec := get(spa.Handler(fstest.MapFS{}, ""), "/")
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

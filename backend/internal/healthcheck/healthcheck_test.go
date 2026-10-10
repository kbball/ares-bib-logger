package healthcheck_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/kevinball/ares-bib-logger/backend/internal/healthcheck"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func portOf(t *testing.T, srv *httptest.Server) string {
	t.Helper()
	u, err := url.Parse(srv.URL)
	require.NoError(t, err)
	return u.Port()
}

func TestRun_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()
	assert.NoError(t, healthcheck.Run(context.Background(), portOf(t, srv)))
}

func TestRun_NotOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	assert.ErrorContains(t, healthcheck.Run(context.Background(), portOf(t, srv)), "HTTP 503")
}

func TestRun_Unreachable(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	port := portOf(t, srv)
	srv.Close()
	assert.Error(t, healthcheck.Run(context.Background(), port))
}

func TestRun_BadPort(t *testing.T) {
	assert.Error(t, healthcheck.Run(context.Background(), "bad port"))
}

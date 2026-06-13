package sse_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/sse"
)

func TestBroker_PublishReachesSubscriber(t *testing.T) {
	b := sse.NewBroker()

	req := httptest.NewRequest(http.MethodGet, "/api/stream", nil)
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)

	w := newFlushRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		b.ServeHTTP(w, req)
	}()

	require.True(t, w.waitFor(`"connected"`, 2*time.Second))

	b.Publish("bib_logged", map[string]any{"bib": 42})
	require.True(t, w.waitFor(`"bib_logged"`, 2*time.Second))

	cancel()
	<-done

	body := w.body()
	assert.Contains(t, body, `"type":"bib_logged"`)
	assert.Contains(t, body, `"bib":42`)
}

func TestBroker_PublishToMultipleSubscribers(t *testing.T) {
	b := sse.NewBroker()

	const n = 3
	recorders := make([]*flushRecorder, n)
	cancels := make([]func(), n)
	dones := make([]chan struct{}, n)

	for i := range n {
		req := httptest.NewRequest(http.MethodGet, "/api/stream", nil)
		ctx, cancel := context.WithCancel(req.Context())
		cancels[i] = cancel
		req = req.WithContext(ctx)
		rec := newFlushRecorder()
		recorders[i] = rec
		done := make(chan struct{})
		dones[i] = done
		go func() {
			defer close(done)
			b.ServeHTTP(rec, req)
		}()
		require.True(t, rec.waitFor(`"connected"`, 2*time.Second), "subscriber %d connect timeout", i)
	}

	b.Publish("session_changed", map[string]any{"ok": true})

	for i, rec := range recorders {
		assert.True(t, rec.waitFor(`"session_changed"`, 2*time.Second), "subscriber %d missed event", i)
		cancels[i]()
		<-dones[i]
	}
}

func TestBroker_NonFlusherReturns500(t *testing.T) {
	b := sse.NewBroker()
	req := httptest.NewRequest(http.MethodGet, "/api/stream", nil)
	w := &strictResponseWriter{header: make(http.Header)}
	b.ServeHTTP(w, req)
	assert.Equal(t, http.StatusInternalServerError, w.status)
}

// strictResponseWriter implements http.ResponseWriter but NOT http.Flusher.
type strictResponseWriter struct {
	header http.Header
	status int
}

func (s *strictResponseWriter) Header() http.Header       { return s.header }
func (s *strictResponseWriter) Write(b []byte) (int, error) { return len(b), nil }
func (s *strictResponseWriter) WriteHeader(code int)        { s.status = code }

func TestBroker_PublishPayloadIsValidJSON(t *testing.T) {
	b := sse.NewBroker()

	req := httptest.NewRequest(http.MethodGet, "/api/stream", nil)
	ctx, cancel := context.WithCancel(req.Context())
	defer cancel()
	req = req.WithContext(ctx)

	w := newFlushRecorder()
	done := make(chan struct{})
	go func() {
		defer close(done)
		b.ServeHTTP(w, req)
	}()
	require.True(t, w.waitFor(`"connected"`, 2*time.Second))

	b.Publish("test_event", map[string]any{"key": "value"})
	require.True(t, w.waitFor(`"test_event"`, 2*time.Second))
	cancel()
	<-done

	for _, line := range strings.Split(w.body(), "\n") {
		if strings.HasPrefix(line, "data: ") {
			var env map[string]any
			require.NoError(t, json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &env))
		}
	}
}

// ── helpers ──────────────────────────────────────────────────────────────────

// flushRecorder is a ResponseWriter that implements http.Flusher and buffers output.
type flushRecorder struct {
	header http.Header
	status int
	buf    strings.Builder
	mu     sync.Mutex
	notify chan struct{}
}

func newFlushRecorder() *flushRecorder {
	return &flushRecorder{
		header: make(http.Header),
		notify: make(chan struct{}, 64),
	}
}

func (r *flushRecorder) Header() http.Header { return r.header }
func (r *flushRecorder) WriteHeader(s int)   { r.status = s }
func (r *flushRecorder) Write(b []byte) (int, error) {
	r.mu.Lock()
	n, err := r.buf.Write(b)
	r.mu.Unlock()
	select {
	case r.notify <- struct{}{}:
	default:
	}
	return n, err
}
func (r *flushRecorder) Flush() {}

func (r *flushRecorder) body() string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.buf.String()
}

func (r *flushRecorder) waitFor(substr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if strings.Contains(r.body(), substr) {
			return true
		}
		select {
		case <-r.notify:
		case <-time.After(10 * time.Millisecond):
		}
	}
	return strings.Contains(r.body(), substr)
}

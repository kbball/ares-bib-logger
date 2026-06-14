package handler_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/http/handler"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

// mockSessionPublishErr lets SetEvent succeed but Get fail, so publishSession's
// error branch is reached without the main handler failing.
type mockSessionPublishErr struct{}

func (m *mockSessionPublishErr) Get(_ context.Context) (entity.ActiveSession, error) {
	return entity.ActiveSession{}, errors.New("get failed")
}
func (m *mockSessionPublishErr) SetEvent(_ context.Context, _ int) error         { return nil }
func (m *mockSessionPublishErr) SetCheckpoint(_ context.Context, _, _ int) error { return nil }
func (m *mockSessionPublishErr) ClearCheckpoint(_ context.Context, _ int) error  { return nil }

// --- updateCheckpoint ---

func TestHandler_UpdateCheckpoint_BadID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodPut, "/api/checkpoints/abc")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateCheckpoint_MissingFields(t *testing.T) {
	w := putJSON(t, defaultHandler(), "/api/checkpoints/1", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_UpdateCheckpoint_Success(t *testing.T) {
	w := putJSON(t, defaultHandler(), "/api/checkpoints/1", map[string]string{
		"code": "AS1", "display_name": "Aid 1",
	})
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_UpdateCheckpoint_ServiceError(t *testing.T) {
	cps := &mockCheckpointService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, cps,
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})
	w := putJSON(t, h, "/api/checkpoints/1", map[string]string{
		"code": "AS1", "display_name": "Aid 1",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- deleteCheckpoint ---

func TestHandler_DeleteCheckpoint_BadID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodDelete, "/api/checkpoints/abc")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteCheckpoint_Success(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodDelete, "/api/checkpoints/1")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_DeleteCheckpoint_ServiceError(t *testing.T) {
	cps := &mockCheckpointService{err: domain.ErrLocked}
	h := newHandler(&mockEventService{}, &mockRaceService{}, cps,
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})
	w := doRequest(t, h, http.MethodDelete, "/api/checkpoints/1")
	assert.Equal(t, http.StatusConflict, w.Code)
}

// --- archiveEvent ---

func TestHandler_ArchiveEvent_BadID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodPut, "/api/events/abc/archive")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ArchiveEvent_Success(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodPut, "/api/events/1/archive")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_ArchiveEvent_ServiceError(t *testing.T) {
	events := &mockEventService{err: domain.ErrNotFound}
	h := newHandler(events, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})
	w := doRequest(t, h, http.MethodPut, "/api/events/1/archive")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- listCheckpointLogs ---

func TestHandler_ListCheckpointLogs_BadRaceID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodGet, "/api/races/abc/logs")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListCheckpointLogs_Success(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodGet, "/api/races/1/logs")
	require.Equal(t, http.StatusOK, w.Code)
	var resp []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Empty(t, resp)
}

func TestHandler_ListCheckpointLogs_ServiceError(t *testing.T) {
	logs := &mockCheckpointLogService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, logs, &mockSessionService{}, &mockWinlinkService{})
	w := doRequest(t, h, http.MethodGet, "/api/races/1/logs")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- lockRaceOrder ---

func TestHandler_LockRaceOrder_BadID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodPut, "/api/races/abc/lock-order")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_LockRaceOrder_Success(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodPut, "/api/races/1/lock-order")
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_LockRaceOrder_ServiceError(t *testing.T) {
	races := &mockRaceService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, races, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})
	w := doRequest(t, h, http.MethodPut, "/api/races/1/lock-order")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- LoggingMiddleware + statusWriter.WriteHeader ---

func TestLoggingMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	wrapped := handler.LoggingMiddleware(inner)
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	wrapped.ServeHTTP(w, req)
	assert.Equal(t, http.StatusTeapot, w.Code)
}

// --- parseTSVRoster — 2-column paths ---

func TestHandler_ImportRoster_TwoColumnWithSpace(t *testing.T) {
	w := postJSON(t, defaultHandler(), "/api/races/1/roster", map[string]string{
		"tsv": "100\tAlice Smith",
	})
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(1), resp["imported"])
}

func TestHandler_ImportRoster_TwoColumnNoSpace(t *testing.T) {
	w := postJSON(t, defaultHandler(), "/api/races/1/roster", map[string]string{
		"tsv": "100\tAlice",
	})
	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(1), resp["imported"])
}

// --- publishSession error branch ---

func TestHandler_PublishSession_GetError(t *testing.T) {
	// Use handler.New directly: newHandler is typed to *mockSessionService, but we need
	// a session where SetEvent succeeds and Get fails to exercise publishSession's error branch.
	h := handler.New(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionPublishErr{}, &mockWinlinkService{}, noopPublisher{})
	w := putJSON(t, h, "/api/session/event", map[string]int{"event_id": 1})
	assert.Equal(t, http.StatusNoContent, w.Code)
}

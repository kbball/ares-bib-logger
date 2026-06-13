package handler_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
)

func doRequest(t *testing.T, h interface{ Register(*http.ServeMux) }, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)
	return w
}

func TestHandler_DeleteRace_BadID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodDelete, "/api/races/abc")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteRace_ServiceError(t *testing.T) {
	races := &mockRaceService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, races, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	w := doRequest(t, h, http.MethodDelete, "/api/races/1")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetSession_Error(t *testing.T) {
	sess := &mockSessionService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, sess, &mockWinlinkService{})

	w := doRequest(t, h, http.MethodGet, "/api/session")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_SetSessionEvent_BadBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/session/event", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SetSessionCheckpoint_BadBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/session/checkpoint", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ClearSessionCheckpoint_BadID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodDelete, "/api/session/checkpoint/abc")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TransferRunner_BadBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/runners/transfer", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListRaces_BadID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodGet, "/api/events/abc/races")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListCheckpoints_BadID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodGet, "/api/races/abc/checkpoints")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListRunners_BadID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodGet, "/api/races/abc/runners")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListRunners_ServiceError(t *testing.T) {
	runners := &mockRunnerService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		runners, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})
	w := doRequest(t, h, http.MethodGet, "/api/races/1/runners")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_CreateEvent_ServiceConflict(t *testing.T) {
	events := &mockEventService{err: domain.ErrAlreadyExists}
	h := newHandler(events, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})
	w := postJSON(t, h, "/api/events", map[string]string{"name": "GDR"})
	assert.Equal(t, http.StatusConflict, w.Code)
}

func TestHandler_LogStatus_ServiceError(t *testing.T) {
	logs := &mockCheckpointLogService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, logs, &mockSessionService{}, &mockWinlinkService{})
	w := postJSON(t, h, "/api/log/status", map[string]any{"bib_number": 42, "status": "DNS"})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ExportWinlink_Error(t *testing.T) {
	wl := &mockWinlinkService{err: domain.ErrNoSession}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, wl)

	w := doRequest(t, h, http.MethodGet, "/api/winlink/export/1")
	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

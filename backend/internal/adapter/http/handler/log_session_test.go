package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/http/handler"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

func newHandler(
	events *mockEventService,
	races *mockRaceService,
	cps *mockCheckpointService,
	runners *mockRunnerService,
	logs *mockCheckpointLogService,
	session *mockSessionService,
	winlink *mockWinlinkService,
) *handler.Handler {
	return handler.New(events, races, cps, runners, logs, session, winlink)
}

func defaultHandler() *handler.Handler {
	return newHandler(
		&mockEventService{},
		&mockRaceService{},
		&mockCheckpointService{},
		&mockRunnerService{},
		&mockCheckpointLogService{},
		&mockSessionService{},
		&mockWinlinkService{},
	)
}

func postJSON(t *testing.T, h *handler.Handler, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)
	return w
}

func getReq(t *testing.T, h *handler.Handler, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)
	return w
}

// --- logBib ---

func TestHandler_LogBib_Success(t *testing.T) {
	runner := entity.Runner{ID: 10, BibNumber: 42}
	logs := &mockCheckpointLogService{
		result: portsvc.LogBibResult{Runner: runner, IsDuplicate: false},
	}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, logs, &mockSessionService{}, &mockWinlinkService{})

	w := postJSON(t, h, "/api/log/bib", map[string]int{"bib_number": 42})

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.False(t, resp["is_duplicate"].(bool))
}

func TestHandler_LogBib_Duplicate(t *testing.T) {
	runner := entity.Runner{ID: 10, BibNumber: 42}
	logs := &mockCheckpointLogService{
		result: portsvc.LogBibResult{Runner: runner, IsDuplicate: true},
	}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, logs, &mockSessionService{}, &mockWinlinkService{})

	w := postJSON(t, h, "/api/log/bib", map[string]int{"bib_number": 42})

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp["is_duplicate"].(bool))
}

func TestHandler_LogBib_NoSession(t *testing.T) {
	logs := &mockCheckpointLogService{err: domain.ErrNoSession}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, logs, &mockSessionService{}, &mockWinlinkService{})

	w := postJSON(t, h, "/api/log/bib", map[string]int{"bib_number": 42})

	assert.Equal(t, http.StatusUnprocessableEntity, w.Code)
}

func TestHandler_LogBib_MissingBib(t *testing.T) {
	w := postJSON(t, defaultHandler(), "/api/log/bib", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_LogStatus_Success(t *testing.T) {
	w := postJSON(t, defaultHandler(), "/api/log/status", map[string]any{
		"bib_number": 42, "status": "DNS",
	})
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_LogStatus_InvalidStatus(t *testing.T) {
	w := postJSON(t, defaultHandler(), "/api/log/status", map[string]any{
		"bib_number": 42, "status": "BOGUS",
	})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- session ---

func TestHandler_GetSession(t *testing.T) {
	eventID := 5
	sess := &mockSessionService{session: entity.ActiveSession{EventID: &eventID}}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, sess, &mockWinlinkService{})

	w := getReq(t, h, "/api/session")

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.NotNil(t, resp["EventID"])
}

func TestHandler_SetSessionEvent(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/session/event",
		bytes.NewBufferString(`{"event_id":1}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_SetSessionCheckpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/api/session/checkpoint",
		bytes.NewBufferString(`{"race_id":1,"checkpoint_id":5}`))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_ClearSessionCheckpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/session/checkpoint/1", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

package handler_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
)

func putJSON(t *testing.T, h interface{ Register(*http.ServeMux) }, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPut, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)
	return w
}

// --- listEvents ---

func TestHandler_ListEvents_ServiceError(t *testing.T) {
	events := &mockEventService{err: domain.ErrNotFound}
	h := newHandler(events, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})
	w := getReq(t, h, "/api/events")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- listRaces ---

func TestHandler_ListRaces_ServiceError(t *testing.T) {
	races := &mockRaceService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, races, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})
	w := getReq(t, h, "/api/events/1/races")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- createRace ---

func TestHandler_CreateRace_BadEventID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodPost, "/api/events/abc/races")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateRace_ServiceError(t *testing.T) {
	races := &mockRaceService{err: domain.ErrAlreadyExists}
	h := newHandler(&mockEventService{}, races, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	body, _ := json.Marshal(map[string]string{"name": "GDR 100"})
	req := httptest.NewRequest(http.MethodPost, "/api/events/1/races", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// --- listCheckpoints ---

func TestHandler_ListCheckpoints_ServiceError(t *testing.T) {
	cps := &mockCheckpointService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, cps,
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})
	w := getReq(t, h, "/api/races/1/checkpoints")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- createCheckpoint ---

func TestHandler_CreateCheckpoint_BadRaceID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodPost, "/api/races/abc/checkpoints")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateCheckpoint_ServiceError(t *testing.T) {
	cps := &mockCheckpointService{err: domain.ErrAlreadyExists}
	h := newHandler(&mockEventService{}, &mockRaceService{}, cps,
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	body, _ := json.Marshal(map[string]any{"code": "AS6", "display_name": "AS6", "display_order": 1})
	req := httptest.NewRequest(http.MethodPost, "/api/races/1/checkpoints", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// --- reorderCheckpoints ---

func TestHandler_ReorderCheckpoints_BadRaceID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodPut, "/api/races/abc/checkpoints/order")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReorderCheckpoints_EmptyIDs(t *testing.T) {
	w := putJSON(t, defaultHandler(), "/api/races/1/checkpoints/order", map[string]any{"ordered_ids": []int{}})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- logStatus ---

func TestHandler_LogStatus_MissingBib(t *testing.T) {
	w := postJSON(t, defaultHandler(), "/api/log/status", map[string]any{"status": "DNS"})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_LogStatus_MissingStatus(t *testing.T) {
	w := postJSON(t, defaultHandler(), "/api/log/status", map[string]any{"bib_number": 42})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- importRoster ---

func TestHandler_ImportRoster_BadRaceID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodPost, "/api/races/abc/roster")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ImportRoster_ServiceError(t *testing.T) {
	runners := &mockRunnerService{err: domain.ErrAlreadyExists}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		runners, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	body, _ := json.Marshal(map[string]any{"tsv": "100\tA\tB"})
	req := httptest.NewRequest(http.MethodPost, "/api/races/1/roster", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// --- transferRunner ---

func TestHandler_TransferRunner_ServiceError(t *testing.T) {
	runners := &mockRunnerService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		runners, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	w := postJSON(t, h, "/api/runners/transfer", map[string]any{
		"bib_number": 42, "from_race_id": 1, "to_race_id": 2,
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- session ---

func TestHandler_SetSessionEvent_ServiceError(t *testing.T) {
	sess := &mockSessionService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, sess, &mockWinlinkService{})
	w := putJSON(t, h, "/api/session/event", map[string]int{"event_id": 1})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_SetSessionCheckpoint_ServiceError(t *testing.T) {
	sess := &mockSessionService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, sess, &mockWinlinkService{})
	w := putJSON(t, h, "/api/session/checkpoint", map[string]int{"race_id": 1, "checkpoint_id": 5})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_ClearSessionCheckpoint_ServiceError(t *testing.T) {
	sess := &mockSessionService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, sess, &mockWinlinkService{})

	req := httptest.NewRequest(http.MethodDelete, "/api/session/checkpoint/1", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- exportWinlink ---

func TestHandler_ExportWinlink_BadRaceID(t *testing.T) {
	w := doRequest(t, defaultHandler(), http.MethodGet, "/api/winlink/export/abc")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- importWinlink ---

func TestHandler_ImportWinlink_ServiceError(t *testing.T) {
	wl := &mockWinlinkService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, wl)

	w := postJSON(t, h, "/api/winlink/import", map[string]any{
		"race_id": 1, "checkpoint_id": 5, "text": "AS6\n17:45:00\n",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- errStatus default case ---

func TestHandler_ErrStatus_InternalServerError(t *testing.T) {
	races := &mockRaceService{err: fmt.Errorf("unexpected failure")}
	h := newHandler(&mockEventService{}, races, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})
	w := doRequest(t, h, http.MethodDelete, "/api/races/1")
	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

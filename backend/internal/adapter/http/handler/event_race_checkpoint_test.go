package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

// --- Events ---

func TestHandler_ListEvents_Empty(t *testing.T) {
	w := getReq(t, defaultHandler(), "/api/events")
	require.Equal(t, http.StatusOK, w.Code)
	var resp []any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp, 0)
}

func TestHandler_ListEvents_WithData(t *testing.T) {
	events := &mockEventService{events: []entity.Event{{ID: 1, Name: "GDR"}}}
	h := newHandler(events, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	w := getReq(t, h, "/api/events")
	require.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp, 1)
	assert.Equal(t, "GDR", resp[0]["Name"])
}

func TestHandler_CreateEvent_Success(t *testing.T) {
	w := postJSON(t, defaultHandler(), "/api/events", map[string]string{"name": "GA Jewel"})
	require.Equal(t, http.StatusCreated, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, "GA Jewel", resp["Name"])
}

func TestHandler_CreateEvent_MissingName(t *testing.T) {
	w := postJSON(t, defaultHandler(), "/api/events", map[string]string{})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetEvent_Success(t *testing.T) {
	events := &mockEventService{events: []entity.Event{{ID: 1, Name: "GDR"}}}
	h := newHandler(events, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	w := getReq(t, h, "/api/events/1")
	require.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetEvent_NotFound(t *testing.T) {
	events := &mockEventService{err: domain.ErrNotFound}
	h := newHandler(events, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	w := getReq(t, h, "/api/events/99")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetEvent_BadID(t *testing.T) {
	w := getReq(t, defaultHandler(), "/api/events/abc")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Races ---

func TestHandler_ListRaces(t *testing.T) {
	races := &mockRaceService{races: []entity.Race{{ID: 1, EventID: 1, Name: "GDR 100"}}}
	h := newHandler(&mockEventService{}, races, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	w := getReq(t, h, "/api/events/1/races")
	require.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp, 1)
}

func TestHandler_CreateRace_Success(t *testing.T) {
	body, _ := json.Marshal(map[string]string{"name": "GDR 100"})
	req := httptest.NewRequest(http.MethodPost, "/api/events/1/races", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateRace_MissingName(t *testing.T) {
	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/events/1/races", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteRace(t *testing.T) {
	req := httptest.NewRequest(http.MethodDelete, "/api/races/1", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

// --- Checkpoints ---

func TestHandler_ListCheckpoints(t *testing.T) {
	cps := &mockCheckpointService{checkpoints: []entity.Checkpoint{{ID: 1, Code: "AS6"}}}
	h := newHandler(&mockEventService{}, &mockRaceService{}, cps,
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	w := getReq(t, h, "/api/races/1/checkpoints")
	require.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp, 1)
}

func TestHandler_CreateCheckpoint_Success(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"code": "AS6", "display_name": "Aid Station 6", "display_order": 1,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/races/1/checkpoints", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusCreated, w.Code)
}

func TestHandler_CreateCheckpoint_MissingCode(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"display_name": "Aid Station 6"})
	req := httptest.NewRequest(http.MethodPost, "/api/races/1/checkpoints", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ReorderCheckpoints_Success(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"ordered_ids": []int{3, 1, 2}})
	req := httptest.NewRequest(http.MethodPut, "/api/races/1/checkpoints/order", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestHandler_ReorderCheckpoints_Locked(t *testing.T) {
	cps := &mockCheckpointService{err: domain.ErrLocked}
	h := newHandler(&mockEventService{}, &mockRaceService{}, cps,
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	body, _ := json.Marshal(map[string]any{"ordered_ids": []int{1, 2}})
	req := httptest.NewRequest(http.MethodPut, "/api/races/1/checkpoints/order", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

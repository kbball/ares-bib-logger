package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

// --- correctLog ---

func TestHandler_CorrectLog_Success(t *testing.T) {
	logs := &mockCheckpointLogService{
		correctLog: entity.CheckpointLog{ID: 1, RunnerID: 2, CheckpointID: 5, Source: entity.SourceCorrection},
	}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, logs, &mockSessionService{}, &mockWinlinkService{})

	w := postJSON(t, h, "/api/log/correction", map[string]any{
		"race_id": 1, "checkpoint_id": 5, "bib_number": 100, "time": "14:32",
	})
	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, []any{1, 5, 100, "14:32"}, logs.correctArgs)

	var got entity.CheckpointLog
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &got))
	assert.Equal(t, entity.SourceCorrection, got.Source)
}

func TestHandler_CorrectLog_MissingFields(t *testing.T) {
	w := postJSON(t, defaultHandler(), "/api/log/correction", map[string]any{"race_id": 1})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CorrectLog_ServiceError(t *testing.T) {
	logs := &mockCheckpointLogService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, logs, &mockSessionService{}, &mockWinlinkService{})

	w := postJSON(t, h, "/api/log/correction", map[string]any{
		"race_id": 1, "checkpoint_id": 5, "bib_number": 100, "time": "14:32",
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// --- deleteLog ---

func TestHandler_DeleteLog_Success(t *testing.T) {
	logs := &mockCheckpointLogService{}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, logs, &mockSessionService{}, &mockWinlinkService{})

	w := deleteJSON(t, h, "/api/log/correction", map[string]any{
		"race_id": 1, "checkpoint_id": 5, "bib_number": 100,
	})
	require.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, []any{1, 5, 100}, logs.deleteArgs)
}

func TestHandler_DeleteLog_MissingFields(t *testing.T) {
	w := deleteJSON(t, defaultHandler(), "/api/log/correction", map[string]any{"race_id": 1})
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteLog_ServiceError(t *testing.T) {
	logs := &mockCheckpointLogService{err: domain.ErrNotFound}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, logs, &mockSessionService{}, &mockWinlinkService{})

	w := deleteJSON(t, h, "/api/log/correction", map[string]any{
		"race_id": 1, "checkpoint_id": 5, "bib_number": 100,
	})
	assert.Equal(t, http.StatusNotFound, w.Code)
}

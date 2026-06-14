package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

// --- Winlink ---

func TestHandler_ExportWinlink(t *testing.T) {
	wl := &mockWinlinkService{exportText: "AS6\n17:45:00\nDNS\n"}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, wl)

	req := httptest.NewRequest(http.MethodGet, "/api/winlink/export/1", nil)
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "text/plain; charset=utf-8", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "AS6")
}

func TestHandler_ImportWinlink(t *testing.T) {
	wl := &mockWinlinkService{importRes: portsvc.WinlinkImportResult{Created: 2, Updated: 1}}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		&mockRunnerService{}, &mockCheckpointLogService{}, &mockSessionService{}, wl)

	body := map[string]any{
		"race_id": 1, "checkpoint_id": 5,
		"text": "AS6\n17:45:00\n",
	}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/winlink/import", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Equal(t, float64(2), resp["Created"])
}

func TestHandler_ImportWinlink_MissingFields(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"race_id": 1})
	req := httptest.NewRequest(http.MethodPost, "/api/winlink/import", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// --- Runners / Roster ---

func TestHandler_ListRunners(t *testing.T) {
	runners := &mockRunnerService{
		runners: []entity.Runner{
			{ID: 1, BibNumber: 100, FirstName: "Alice"},
		},
	}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		runners, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	w := getReq(t, h, "/api/races/1/runners")

	require.Equal(t, http.StatusOK, w.Code)
	var resp []map[string]any
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.Len(t, resp, 1)
}

func TestHandler_ImportRoster_Success(t *testing.T) {
	runners := &mockRunnerService{}
	h := newHandler(&mockEventService{}, &mockRaceService{}, &mockCheckpointService{},
		runners, &mockCheckpointLogService{}, &mockSessionService{}, &mockWinlinkService{})

	b, _ := json.Marshal(map[string]any{"tsv": "100\tAlice\tSmith\n101\tBob\tJones"})
	req := httptest.NewRequest(http.MethodPost, "/api/races/1/roster", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	h.Register(mux)
	mux.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, runners.importCalled)
}

func TestHandler_ImportRoster_EmptyRows(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"rows": []any{}})
	req := httptest.NewRequest(http.MethodPost, "/api/races/1/roster", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_TransferRunner(t *testing.T) {
	body, _ := json.Marshal(map[string]any{
		"bib_number": 42, "from_race_id": 1, "to_race_id": 2,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/runners/transfer", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux := http.NewServeMux()
	defaultHandler().Register(mux)
	mux.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
}

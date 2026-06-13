package handler

import (
	"net/http"
)

func (h *Handler) exportWinlink(w http.ResponseWriter, r *http.Request) {
	raceID, ok := pathInt(r, "raceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	column, err := h.winlink.Export(r.Context(), raceID)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(column))
}

func (h *Handler) importWinlink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RaceID       int    `json:"race_id"`
		CheckpointID int    `json:"checkpoint_id"`
		Text         string `json:"text"`
	}
	if err := decode(r, &body); err != nil || body.RaceID == 0 || body.CheckpointID == 0 || body.Text == "" {
		writeError(w, http.StatusBadRequest, "race_id, checkpoint_id, and text are required")
		return
	}

	result, err := h.winlink.Import(r.Context(), body.RaceID, body.CheckpointID, body.Text)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

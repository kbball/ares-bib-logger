package handler

import (
	"net/http"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

func (h *Handler) listCheckpointLogs(w http.ResponseWriter, r *http.Request) {
	raceID, ok := pathInt(r, "raceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	logs, err := h.checkpointLogs.ListByRace(r.Context(), raceID)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	if logs == nil {
		logs = []entity.CheckpointLog{}
	}
	writeJSON(w, http.StatusOK, logs)
}

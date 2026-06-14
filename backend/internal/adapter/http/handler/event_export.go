package handler

import (
	"encoding/json"
	"fmt"
	"net/http"

	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

func (h *Handler) exportEventConfig(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	payload, err := h.eventExport.Export(r.Context(), id)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}

	filename := fmt.Sprintf("event-%d.json", id)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}

func (h *Handler) importEventConfig(w http.ResponseWriter, r *http.Request) {
	var payload portsvc.EventExportPayload
	if err := decode(r, &payload); err != nil {
		writeError(w, http.StatusBadRequest, "invalid payload")
		return
	}
	if payload.Event.Name == "" {
		writeError(w, http.StatusBadRequest, "event name is required")
		return
	}

	eventID, err := h.eventExport.Import(r.Context(), payload)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, map[string]int{"event_id": eventID})
}

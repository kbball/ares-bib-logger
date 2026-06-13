package handler

import (
	"net/http"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

func (h *Handler) listRaces(w http.ResponseWriter, r *http.Request) {
	eventID, ok := pathInt(r, "eventID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	races, err := h.races.List(r.Context(), eventID)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	if races == nil {
		races = []entity.Race{}
	}
	writeJSON(w, http.StatusOK, races)
}

func (h *Handler) createRace(w http.ResponseWriter, r *http.Request) {
	eventID, ok := pathInt(r, "eventID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	var body struct {
		Name string `json:"name"`
	}
	if err := decode(r, &body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	race, err := h.races.Create(r.Context(), eventID, body.Name)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, race)
}

func (h *Handler) deleteRace(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	if err := h.races.Delete(r.Context(), id); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

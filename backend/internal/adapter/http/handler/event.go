package handler

import (
	"net/http"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	events, err := h.events.List(r.Context())
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	if events == nil {
		events = []entity.Event{}
	}
	writeJSON(w, http.StatusOK, events)
}

func (h *Handler) createEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := decode(r, &body); err != nil || body.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}

	event, err := h.events.Create(r.Context(), body.Name)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (h *Handler) getEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	event, err := h.events.Get(r.Context(), id)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h *Handler) archiveEvent(w http.ResponseWriter, r *http.Request) {
	id, ok := pathInt(r, "id")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid event id")
		return
	}

	if err := h.events.Archive(r.Context(), id); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

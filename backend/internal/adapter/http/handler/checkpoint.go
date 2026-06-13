package handler

import (
	"net/http"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

func (h *Handler) listCheckpoints(w http.ResponseWriter, r *http.Request) {
	raceID, ok := pathInt(r, "raceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	cps, err := h.checkpoints.List(r.Context(), raceID)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	if cps == nil {
		cps = []entity.Checkpoint{}
	}
	writeJSON(w, http.StatusOK, cps)
}

func (h *Handler) createCheckpoint(w http.ResponseWriter, r *http.Request) {
	raceID, ok := pathInt(r, "raceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	var body struct {
		Code         string `json:"code"`
		DisplayName  string `json:"display_name"`
		DisplayOrder int    `json:"display_order"`
	}
	if err := decode(r, &body); err != nil || body.Code == "" || body.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "code and display_name are required")
		return
	}

	cp, err := h.checkpoints.Create(r.Context(), entity.Checkpoint{
		RaceID:       raceID,
		Code:         body.Code,
		DisplayName:  body.DisplayName,
		DisplayOrder: body.DisplayOrder,
	})
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cp)
}

func (h *Handler) reorderCheckpoints(w http.ResponseWriter, r *http.Request) {
	raceID, ok := pathInt(r, "raceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	var body struct {
		OrderedIDs []int `json:"ordered_ids"`
	}
	if err := decode(r, &body); err != nil || len(body.OrderedIDs) == 0 {
		writeError(w, http.StatusBadRequest, "ordered_ids is required")
		return
	}

	if err := h.checkpoints.Reorder(r.Context(), raceID, body.OrderedIDs); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

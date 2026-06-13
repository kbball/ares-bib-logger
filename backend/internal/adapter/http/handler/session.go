package handler

import (
	"net/http"
)

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	sess, err := h.session.Get(r.Context())
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

func (h *Handler) setSessionEvent(w http.ResponseWriter, r *http.Request) {
	var body struct {
		EventID int `json:"event_id"`
	}
	if err := decode(r, &body); err != nil || body.EventID == 0 {
		writeError(w, http.StatusBadRequest, "event_id is required")
		return
	}

	if err := h.session.SetEvent(r.Context(), body.EventID); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	h.publishSession(r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) setSessionCheckpoint(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RaceID       int `json:"race_id"`
		CheckpointID int `json:"checkpoint_id"`
	}
	if err := decode(r, &body); err != nil || body.RaceID == 0 || body.CheckpointID == 0 {
		writeError(w, http.StatusBadRequest, "race_id and checkpoint_id are required")
		return
	}

	if err := h.session.SetCheckpoint(r.Context(), body.RaceID, body.CheckpointID); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	h.publishSession(r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) clearSessionCheckpoint(w http.ResponseWriter, r *http.Request) {
	raceID, ok := pathInt(r, "raceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	if err := h.session.ClearCheckpoint(r.Context(), raceID); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	h.publishSession(r)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) publishSession(r *http.Request) {
	sess, err := h.session.Get(r.Context())
	if err != nil {
		return
	}
	h.stream.Publish("session_changed", sess)
}

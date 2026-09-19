package handler

import (
	"net/http"
	"strconv"

	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

func (h *Handler) exportWinlink(w http.ResponseWriter, r *http.Request) {
	raceID, ok := pathInt(r, "raceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	result, err := h.winlink.Export(r.Context(), raceID)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	h.stream.Publish("winlink_reminder_changed", map[string]any{"race_id": raceID})

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	if result.FooterOverflowCount > 0 {
		w.Header().Set("X-Footer-Overflow-Count", strconv.Itoa(result.FooterOverflowCount))
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(result.Text))
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

func (h *Handler) previewWinlink(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RaceID       int    `json:"race_id"`
		CheckpointID int    `json:"checkpoint_id"`
		Text         string `json:"text"`
	}
	if err := decode(r, &body); err != nil || body.RaceID == 0 || body.CheckpointID == 0 || body.Text == "" {
		writeError(w, http.StatusBadRequest, "race_id, checkpoint_id, and text are required")
		return
	}

	result, err := h.winlink.Preview(r.Context(), body.RaceID, body.CheckpointID, body.Text)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func (h *Handler) listWinlinkReminders(w http.ResponseWriter, r *http.Request) {
	reminders, err := h.winlink.Reminders(r.Context())
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	if reminders == nil {
		reminders = []portsvc.WinlinkReminderStatus{}
	}
	writeJSON(w, http.StatusOK, reminders)
}

func (h *Handler) dismissWinlinkReminder(w http.ResponseWriter, r *http.Request) {
	raceID, ok := pathInt(r, "raceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	if err := h.winlink.DismissReminder(r.Context(), raceID); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	h.stream.Publish("winlink_reminder_changed", map[string]any{"race_id": raceID})
	w.WriteHeader(http.StatusNoContent)
}

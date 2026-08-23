package handler

import (
	"net/http"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

func (h *Handler) logBib(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BibNumber int `json:"bib_number"`
	}
	if err := decode(r, &body); err != nil || body.BibNumber == 0 {
		writeError(w, http.StatusBadRequest, "bib_number is required")
		return
	}

	result, err := h.checkpointLogs.LogBib(r.Context(), portsvc.LogBibInput{
		BibNumber: body.BibNumber,
		Source:    entity.SourceManual,
	})
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	payload := map[string]any{
		"runner":       result.Runner,
		"log":          result.Log,
		"is_duplicate": result.IsDuplicate,
	}
	h.stream.Publish("bib_logged", payload)
	writeJSON(w, http.StatusOK, payload)
}

func (h *Handler) logStatus(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BibNumber int    `json:"bib_number"`
		Status    string `json:"status"`
	}
	if err := decode(r, &body); err != nil || body.BibNumber == 0 || body.Status == "" {
		writeError(w, http.StatusBadRequest, "bib_number and status are required")
		return
	}

	status := entity.RunnerStatus(body.Status)
	switch status {
	case entity.StatusDNS, entity.StatusDNF, entity.StatusActive, entity.StatusFinished:
	default:
		writeError(w, http.StatusBadRequest, "status must be DNS, DNF, ACTIVE, or FINISHED")
		return
	}

	if err := h.checkpointLogs.LogStatus(r.Context(), body.BibNumber, status); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) correctLog(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RaceID       int    `json:"race_id"`
		CheckpointID int    `json:"checkpoint_id"`
		BibNumber    int    `json:"bib_number"`
		Time         string `json:"time"`
	}
	if err := decode(r, &body); err != nil || body.RaceID == 0 || body.CheckpointID == 0 ||
		body.BibNumber == 0 || body.Time == "" {
		writeError(w, http.StatusBadRequest, "race_id, checkpoint_id, bib_number, and time are required")
		return
	}

	log, err := h.checkpointLogs.CorrectLog(r.Context(), body.RaceID, body.CheckpointID, body.BibNumber, body.Time)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, log)
}

func (h *Handler) deleteLog(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RaceID       int `json:"race_id"`
		CheckpointID int `json:"checkpoint_id"`
		BibNumber    int `json:"bib_number"`
	}
	if err := decode(r, &body); err != nil || body.RaceID == 0 || body.CheckpointID == 0 || body.BibNumber == 0 {
		writeError(w, http.StatusBadRequest, "race_id, checkpoint_id, and bib_number are required")
		return
	}

	if err := h.checkpointLogs.DeleteLog(r.Context(), body.RaceID, body.CheckpointID, body.BibNumber); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

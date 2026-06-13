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
	writeJSON(w, http.StatusOK, map[string]any{
		"runner":       result.Runner,
		"log":          result.Log,
		"is_duplicate": result.IsDuplicate,
	})
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

package handler

import (
	"net/http"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

func (h *Handler) listRunners(w http.ResponseWriter, r *http.Request) {
	raceID, ok := pathInt(r, "raceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	runners, err := h.runners.ListByRace(r.Context(), raceID)
	if err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	if runners == nil {
		runners = []entity.Runner{}
	}
	writeJSON(w, http.StatusOK, runners)
}

func (h *Handler) importRoster(w http.ResponseWriter, r *http.Request) {
	raceID, ok := pathInt(r, "raceID")
	if !ok {
		writeError(w, http.StatusBadRequest, "invalid race id")
		return
	}

	var body struct {
		Rows []struct {
			BibNumber int    `json:"bib_number"`
			FirstName string `json:"first_name"`
			LastName  string `json:"last_name"`
		} `json:"rows"`
	}
	if err := decode(r, &body); err != nil || len(body.Rows) == 0 {
		writeError(w, http.StatusBadRequest, "rows are required")
		return
	}

	rows := make([]portsvc.RosterRow, len(body.Rows))
	for i, r := range body.Rows {
		rows[i] = portsvc.RosterRow{
			BibNumber: r.BibNumber,
			FirstName: r.FirstName,
			LastName:  r.LastName,
		}
	}

	if err := h.runners.ImportRoster(r.Context(), raceID, rows); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) transferRunner(w http.ResponseWriter, r *http.Request) {
	var body struct {
		BibNumber  int `json:"bib_number"`
		FromRaceID int `json:"from_race_id"`
		ToRaceID   int `json:"to_race_id"`
	}
	if err := decode(r, &body); err != nil || body.BibNumber == 0 || body.FromRaceID == 0 || body.ToRaceID == 0 {
		writeError(w, http.StatusBadRequest, "bib_number, from_race_id, and to_race_id are required")
		return
	}

	if err := h.runners.TransferRace(r.Context(), body.BibNumber, body.FromRaceID, body.ToRaceID); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

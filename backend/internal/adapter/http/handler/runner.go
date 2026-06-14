package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

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
		TSV string `json:"tsv"`
	}
	if err := decode(r, &body); err != nil || strings.TrimSpace(body.TSV) == "" {
		writeError(w, http.StatusBadRequest, "tsv is required")
		return
	}

	rows, err := parseTSVRoster(body.TSV)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.runners.ImportRoster(r.Context(), raceID, rows); err != nil {
		writeError(w, errStatus(err), err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]int{"imported": len(rows)})
}

// parseTSVRoster parses a tab-separated paste (from Excel) into roster rows.
// Accepts 2 columns (bib, combined name) or 3 columns (bib, first, last).
func parseTSVRoster(tsv string) ([]portsvc.RosterRow, error) {
	var rows []portsvc.RosterRow
	for i, line := range strings.Split(strings.TrimSpace(tsv), "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 2 {
			return nil, fmt.Errorf("line %d: expected at least 2 tab-separated columns", i+1)
		}
		bib, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil || bib <= 0 {
			return nil, fmt.Errorf("line %d: invalid bib number %q", i+1, parts[0])
		}
		var first, last string
		if len(parts) == 2 {
			name := strings.TrimSpace(parts[1])
			if idx := strings.Index(name, " "); idx >= 0 {
				first = name[:idx]
				last = name[idx+1:]
			} else {
				first = name
			}
		} else {
			first = strings.TrimSpace(parts[1])
			last = strings.TrimSpace(parts[2])
		}
		rows = append(rows, portsvc.RosterRow{BibNumber: bib, FirstName: first, LastName: last})
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("no valid rows found in TSV")
	}
	return rows, nil
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

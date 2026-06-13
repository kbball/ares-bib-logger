package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
)

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func errStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(err, domain.ErrAlreadyExists):
		return http.StatusConflict
	case errors.Is(err, domain.ErrLocked):
		return http.StatusConflict
	case errors.Is(err, domain.ErrNoSession):
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

func pathInt(r *http.Request, key string) (int, bool) {
	n, err := strconv.Atoi(r.PathValue(key))
	return n, err == nil
}

func decode(r *http.Request, dst any) error {
	return json.NewDecoder(r.Body).Decode(dst)
}

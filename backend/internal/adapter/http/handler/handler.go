package handler

import (
	"net/http"

	"github.com/kevinball/ares-bib-logger/backend/internal/adapter/sse"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

// Handler holds all application services and registers HTTP routes.
type Handler struct {
	events         portsvc.EventService
	races          portsvc.RaceService
	checkpoints    portsvc.CheckpointService
	runners        portsvc.RunnerService
	checkpointLogs portsvc.CheckpointLogService
	session        portsvc.SessionService
	winlink        portsvc.WinlinkService
	stream         sse.Publisher
}

func New(
	events portsvc.EventService,
	races portsvc.RaceService,
	checkpoints portsvc.CheckpointService,
	runners portsvc.RunnerService,
	checkpointLogs portsvc.CheckpointLogService,
	session portsvc.SessionService,
	winlink portsvc.WinlinkService,
	stream sse.Publisher,
) *Handler {
	return &Handler{
		events:         events,
		races:          races,
		checkpoints:    checkpoints,
		runners:        runners,
		checkpointLogs: checkpointLogs,
		session:        session,
		winlink:        winlink,
		stream:         stream,
	}
}

// Register wires all API routes onto mux. broker is registered separately as GET /api/stream.
func (h *Handler) Register(mux *http.ServeMux) {
	// Events
	mux.HandleFunc("GET /api/events", h.listEvents)
	mux.HandleFunc("POST /api/events", h.createEvent)
	mux.HandleFunc("GET /api/events/{id}", h.getEvent)

	// Races
	mux.HandleFunc("GET /api/events/{eventID}/races", h.listRaces)
	mux.HandleFunc("POST /api/events/{eventID}/races", h.createRace)
	mux.HandleFunc("DELETE /api/races/{id}", h.deleteRace)

	// Checkpoints
	mux.HandleFunc("GET /api/races/{raceID}/checkpoints", h.listCheckpoints)
	mux.HandleFunc("POST /api/races/{raceID}/checkpoints", h.createCheckpoint)
	mux.HandleFunc("PUT /api/races/{raceID}/checkpoints/order", h.reorderCheckpoints)

	// Runners / Roster
	mux.HandleFunc("GET /api/races/{raceID}/runners", h.listRunners)
	mux.HandleFunc("POST /api/races/{raceID}/roster", h.importRoster)
	mux.HandleFunc("POST /api/runners/transfer", h.transferRunner)

	// Bib logging
	mux.HandleFunc("POST /api/log/bib", h.logBib)
	mux.HandleFunc("POST /api/log/status", h.logStatus)

	// Session
	mux.HandleFunc("GET /api/session", h.getSession)
	mux.HandleFunc("PUT /api/session/event", h.setSessionEvent)
	mux.HandleFunc("PUT /api/session/checkpoint", h.setSessionCheckpoint)
	mux.HandleFunc("DELETE /api/session/checkpoint/{raceID}", h.clearSessionCheckpoint)

	// Winlink
	mux.HandleFunc("GET /api/winlink/export/{raceID}", h.exportWinlink)
	mux.HandleFunc("POST /api/winlink/import", h.importWinlink)
}

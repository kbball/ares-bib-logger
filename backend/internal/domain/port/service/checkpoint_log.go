package service

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type LogBibInput struct {
	BibNumber  int
	Source     entity.LogSource
	RawMessage string
}

type LogBibResult struct {
	Log         entity.CheckpointLog
	Runner      entity.Runner
	IsDuplicate bool
}

type CheckpointLogService interface {
	LogBib(ctx context.Context, input LogBibInput) (LogBibResult, error)
	// LogStatus records a DNS/DNF/ACTIVE status change for a runner by bib number.
	LogStatus(ctx context.Context, bibNumber int, status entity.RunnerStatus) error
	// ListByRace returns all checkpoint logs for runners in a race.
	ListByRace(ctx context.Context, raceID int) ([]entity.CheckpointLog, error)
	// QueryRunner returns a compact, ready-to-send text summary of a runner's
	// status, last known checkpoint, and pace, for replying to a mesh "query"
	// command. Returns a "not found" message rather than an error when the
	// bib doesn't match any runner in the active event.
	QueryRunner(ctx context.Context, bibNumber int) (string, error)
	// CorrectLog creates or overwrites (by runner+checkpoint) a checkpoint log
	// at an explicit wall-clock time ("HH:MM" or "HH:MM:SS", today's date in
	// the service's configured timezone), with Source set to CORRECTION, for
	// fixing a mis-logged bib after the fact.
	CorrectLog(ctx context.Context, raceID, checkpointID, bibNumber int, dateStr, timeStr string) (entity.CheckpointLog, error)
	// DeleteLog removes a runner's checkpoint log for the given race+checkpoint+bib.
	DeleteLog(ctx context.Context, raceID, checkpointID, bibNumber int) error
	// StationCheckpoints returns a compact summary of this station's active
	// checkpoint for each race in the active event, for replying to a mesh
	// "checkpoint" command.
	StationCheckpoints(ctx context.Context) (string, error)
	// StationCount returns the number of checkpoint logs recorded at this
	// station's active checkpoint(s), for replying to a mesh "count" command —
	// a sanity check against a paper tally.
	StationCount(ctx context.Context) (string, error)
	// SearchRunners returns bib/name/race for runners in the active event whose
	// last name contains the given text (case-insensitive), for replying to a
	// mesh "search <name>" command. Results are capped to keep the reply short.
	SearchRunners(ctx context.Context, lastName string) (string, error)
	// CheckDuplicate reports whether a bib has already been logged at this
	// station's active checkpoint for that runner's race, for replying to a
	// mesh "dup <bib>" command.
	CheckDuplicate(ctx context.Context, bibNumber int) (string, error)
}

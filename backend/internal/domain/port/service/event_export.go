package service

import (
	"context"
	"time"
)

// EventExportPayload is the wire format for event configuration export/import.
// It omits log data — only structural config (event, races, checkpoints, roster).
type EventExportPayload struct {
	Version    int              `json:"version"`
	ExportedAt time.Time        `json:"exported_at"`
	Event      EventExportInfo  `json:"event"`
	Races      []RaceExportData `json:"races"`
}

type EventExportInfo struct {
	Name string `json:"name"`
}

type RaceExportData struct {
	Name        string             `json:"name"`
	Checkpoints []CheckpointExport `json:"checkpoints"`
	Runners     []RunnerExport     `json:"runners"`
}

type CheckpointExport struct {
	Code              string   `json:"code"`
	DisplayName       string   `json:"display_name"`
	DisplayOrder      int      `json:"display_order"`
	DistanceFromStart *float64 `json:"distance_from_start,omitempty"`
}

type RunnerExport struct {
	BibNumber int    `json:"bib_number"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	SortOrder int    `json:"sort_order"`
}

type EventExportService interface {
	Export(ctx context.Context, eventID int) (EventExportPayload, error)
	Import(ctx context.Context, payload EventExportPayload) (int, error) // returns new event ID
}

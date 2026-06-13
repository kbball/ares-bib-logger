package service

import "context"

type WinlinkImportResult struct {
	Created int
	Updated int // status changes (DNS/DNF)
	Skipped int // blank lines
}

type WinlinkService interface {
	// Export generates a Winlink-format column for the active checkpoint of the given race.
	Export(ctx context.Context, raceID int) (string, error)
	// Import parses a pasted Winlink column and records it against the given race+checkpoint.
	Import(ctx context.Context, raceID, checkpointID int, text string) (WinlinkImportResult, error)
}

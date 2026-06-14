package service

import "context"

type WinlinkSkipDetail struct {
	Position  int    // 1-based position in the data rows (after the optional header)
	BibNumber int    // 0 if no runner was found at this position
	Reason    string // "blank" | "no_runner" | "duplicate" | "parse_error"
}

type WinlinkImportResult struct {
	Created        int
	Updated        int // status changes (DNS/DNF)
	Skipped        int
	SkippedDetails []WinlinkSkipDetail
}

type WinlinkService interface {
	// Export generates a Winlink-format column for the active checkpoint of the given race.
	Export(ctx context.Context, raceID int) (string, error)
	// Import parses a pasted Winlink column and records it against the given race+checkpoint.
	Import(ctx context.Context, raceID, checkpointID int, text string) (WinlinkImportResult, error)
}

package service

import "context"

type WinlinkSkipDetail struct {
	Position  int    // 1-based position in the data rows (after the optional header)
	BibNumber int    // 0 if no runner was found at this position
	Reason    string // "blank" | "no_runner" | "parse_error" | "moved"
}

type WinlinkImportResult struct {
	Created        int
	Updated        int // status changes (DNS/DNF)
	Skipped        int
	SkippedDetails []WinlinkSkipDetail
}

// WinlinkRowOutcome describes what Import would do with (or has already done
// with) a single pasted row, used by Preview to show the operator a full
// before-you-commit breakdown.
type WinlinkRowOutcome struct {
	Position  int
	BibNumber int
	Kind      string // "create" | "update" | "skip"
	Value     string // trimmed raw pasted line; empty for blank/no_runner skips
	Reason    string // set only when Kind == "skip": "blank" | "no_runner" | "parse_error" | "moved"
}

type WinlinkPreviewResult struct {
	Created int
	Updated int
	Skipped int
	Rows    []WinlinkRowOutcome
	// HeaderMismatch is true when the pasted text's header line doesn't match
	// the selected checkpoint's expected header (ColumnName, falling back to
	// DisplayName) — a signal the operator may have selected the wrong
	// race/checkpoint or pasted the wrong station's column. False (with both
	// header fields empty) when no header line was detected in the paste.
	HeaderMismatch bool
	PastedHeader   string
	ExpectedHeader string
}

type WinlinkService interface {
	// Export generates a Winlink-format column for the active checkpoint of the given race.
	Export(ctx context.Context, raceID int) (string, error)
	// Import parses a pasted Winlink column and records it against the given race+checkpoint.
	Import(ctx context.Context, raceID, checkpointID int, text string) (WinlinkImportResult, error)
	// Preview classifies each row the same way Import would, without writing anything.
	Preview(ctx context.Context, raceID, checkpointID int, text string) (WinlinkPreviewResult, error)
}

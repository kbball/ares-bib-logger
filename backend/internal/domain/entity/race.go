package entity

import "time"

type Race struct {
	ID           int
	EventID      int
	Name         string
	RosterLocked bool
	OrderLocked  bool
	// RosterCount is the size of the original locked roster (set once, when
	// the roster is locked). Runners with SortOrder <= RosterCount are the
	// original list; runners added afterward (transfers, late adds) sort
	// above it and belong in the Winlink export footer.
	RosterCount int
	// WinlinkFooterRows is how many footer rows the Winlink export pads to
	// (or reports overflow past) for runners added after the roster locked.
	// Editable at any time, independent of RosterLocked, so an RD can raise
	// it mid-race as transfers happen.
	WinlinkFooterRows int
	CreatedAt         time.Time
}

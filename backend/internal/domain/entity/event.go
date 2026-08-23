package entity

import "time"

type Event struct {
	ID                          int
	Name                        string
	Archived                    bool
	WinlinkBlankLineAfterHeader bool
	CreatedAt                   time.Time
}

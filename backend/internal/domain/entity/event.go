package entity

import "time"

type Event struct {
	ID                          int
	Name                        string
	Archived                    bool
	WinlinkBlankLineAfterHeader bool
	// WinlinkReminderMinutes is how long after a Winlink export before an
	// operator is reminded to send another update from the aid station. 0
	// disables the reminder.
	WinlinkReminderMinutes int
	CreatedAt              time.Time
}

package entity

import "time"

type Race struct {
	ID           int
	EventID      int
	Name         string
	RosterLocked bool
	OrderLocked  bool
	CreatedAt    time.Time
}

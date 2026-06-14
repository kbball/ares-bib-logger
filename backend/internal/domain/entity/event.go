package entity

import "time"

type Event struct {
	ID        int
	Name      string
	Archived  bool
	CreatedAt time.Time
}

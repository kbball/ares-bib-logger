package entity

import "time"

type Checkpoint struct {
	ID                int
	RaceID            int
	Code              string
	DisplayName       string
	DisplayOrder      int
	DistanceFromStart *float64
	CreatedAt         time.Time
}

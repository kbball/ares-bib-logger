package entity

import "time"

type Checkpoint struct {
	ID                int
	RaceID            int
	Code              string
	DisplayName       string
	ColumnName        *string
	DisplayOrder      int
	DistanceFromStart *float64
	CutoffTime        *string
	CreatedAt         time.Time
}

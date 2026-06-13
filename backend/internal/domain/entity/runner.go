package entity

import "time"

type RunnerStatus string

const (
	StatusUnknown  RunnerStatus = "UNKNOWN"
	StatusActive   RunnerStatus = "ACTIVE"
	StatusDNS      RunnerStatus = "DNS"
	StatusDNF      RunnerStatus = "DNF"
	StatusFinished RunnerStatus = "FINISHED"
	StatusMoved    RunnerStatus = "MOVED"
)

type Runner struct {
	ID        int
	RaceID    int
	BibNumber int
	FirstName string
	LastName  string
	SortOrder int
	Status    RunnerStatus
	CreatedAt time.Time
	UpdatedAt time.Time
}

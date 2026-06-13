package entity

import "time"

type LogSource string

const (
	SourceMeshtastic    LogSource = "MESHTASTIC"
	SourceManual        LogSource = "MANUAL"
	SourceWinlinkImport LogSource = "WINLINK_IMPORT"
)

type CheckpointLog struct {
	ID           int
	RunnerID     int
	CheckpointID int
	RecordedAt   time.Time
	Source       LogSource
	RawMessage   string
	CreatedAt    time.Time
}

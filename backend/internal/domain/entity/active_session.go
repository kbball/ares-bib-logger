package entity

type ActiveSession struct {
	EventID     *int
	Checkpoints []ActiveSessionCheckpoint
}

type ActiveSessionCheckpoint struct {
	RaceID       int
	CheckpointID int
}

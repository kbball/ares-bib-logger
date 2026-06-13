package service

import (
	"context"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

type LogBibInput struct {
	BibNumber  int
	Source     entity.LogSource
	RawMessage string
}

type LogBibResult struct {
	Log         entity.CheckpointLog
	Runner      entity.Runner
	IsDuplicate bool
}

type CheckpointLogService interface {
	LogBib(ctx context.Context, input LogBibInput) (LogBibResult, error)
	// LogStatus records a DNS/DNF/ACTIVE status change for a runner by bib number.
	LogStatus(ctx context.Context, bibNumber int, status entity.RunnerStatus) error
}

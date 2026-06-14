package service

import (
	"context"
	"fmt"
	"time"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

type CheckpointLogService struct {
	runners        portrepo.RunnerRepository
	checkpointLogs portrepo.CheckpointLogRepository
	session        portrepo.ActiveSessionRepository
}

func NewCheckpointLogService(
	runners portrepo.RunnerRepository,
	checkpointLogs portrepo.CheckpointLogRepository,
	session portrepo.ActiveSessionRepository,
) *CheckpointLogService {
	return &CheckpointLogService{
		runners:        runners,
		checkpointLogs: checkpointLogs,
		session:        session,
	}
}

var _ portsvc.CheckpointLogService = (*CheckpointLogService)(nil)

func (s *CheckpointLogService) LogBib(ctx context.Context, input portsvc.LogBibInput) (portsvc.LogBibResult, error) {
	sess, err := s.session.Get(ctx)
	if err != nil {
		return portsvc.LogBibResult{}, fmt.Errorf("getting session: %w", err)
	}
	if sess.EventID == nil {
		return portsvc.LogBibResult{}, domain.ErrNoSession
	}

	runner, err := s.runners.GetByBibInEvent(ctx, *sess.EventID, input.BibNumber)
	if err != nil {
		return portsvc.LogBibResult{}, fmt.Errorf("bib %d: %w", input.BibNumber, err)
	}

	checkpointID, ok := activeCheckpointForRace(sess, runner.RaceID)
	if !ok {
		return portsvc.LogBibResult{}, fmt.Errorf("no active checkpoint for race %d", runner.RaceID)
	}

	exists, err := s.checkpointLogs.ExistsByRunnerAndCheckpoint(ctx, runner.ID, checkpointID)
	if err != nil {
		return portsvc.LogBibResult{}, fmt.Errorf("checking duplicate: %w", err)
	}
	if exists {
		return portsvc.LogBibResult{Runner: runner, IsDuplicate: true}, nil
	}

	log, err := s.checkpointLogs.Create(ctx, entity.CheckpointLog{
		RunnerID:     runner.ID,
		CheckpointID: checkpointID,
		RecordedAt:   time.Now(),
		Source:       input.Source,
		RawMessage:   input.RawMessage,
	})
	if err != nil {
		return portsvc.LogBibResult{}, fmt.Errorf("creating log: %w", err)
	}

	if runner.Status == entity.StatusUnknown {
		if err := s.runners.UpdateStatus(ctx, runner.ID, entity.StatusActive); err != nil {
			return portsvc.LogBibResult{}, fmt.Errorf("updating runner status: %w", err)
		}
		runner.Status = entity.StatusActive
	}

	return portsvc.LogBibResult{Log: log, Runner: runner}, nil
}

func (s *CheckpointLogService) ListByRace(ctx context.Context, raceID int) ([]entity.CheckpointLog, error) {
	return s.checkpointLogs.ListByRace(ctx, raceID)
}

func (s *CheckpointLogService) LogStatus(ctx context.Context, bibNumber int, status entity.RunnerStatus) error {
	sess, err := s.session.Get(ctx)
	if err != nil {
		return fmt.Errorf("getting session: %w", err)
	}
	if sess.EventID == nil {
		return domain.ErrNoSession
	}

	runner, err := s.runners.GetByBibInEvent(ctx, *sess.EventID, bibNumber)
	if err != nil {
		return fmt.Errorf("bib %d: %w", bibNumber, err)
	}

	return s.runners.UpdateStatus(ctx, runner.ID, status)
}

func activeCheckpointForRace(sess entity.ActiveSession, raceID int) (int, bool) {
	for _, sc := range sess.Checkpoints {
		if sc.RaceID == raceID {
			return sc.CheckpointID, true
		}
	}
	return 0, false
}

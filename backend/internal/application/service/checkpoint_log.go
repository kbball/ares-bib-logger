package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/pace"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

type CheckpointLogService struct {
	runners        portrepo.RunnerRepository
	checkpoints    portrepo.CheckpointRepository
	checkpointLogs portrepo.CheckpointLogRepository
	session        portrepo.ActiveSessionRepository
	loc            *time.Location
}

func NewCheckpointLogService(
	runners portrepo.RunnerRepository,
	checkpoints portrepo.CheckpointRepository,
	checkpointLogs portrepo.CheckpointLogRepository,
	session portrepo.ActiveSessionRepository,
	loc *time.Location,
) *CheckpointLogService {
	if loc == nil {
		loc = time.Local
	}
	return &CheckpointLogService{
		runners:        runners,
		checkpoints:    checkpoints,
		checkpointLogs: checkpointLogs,
		session:        session,
		loc:            loc,
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
		return portsvc.LogBibResult{
			Runner:      runner,
			IsDuplicate: true,
			Log:         entity.CheckpointLog{Source: input.Source},
		}, nil
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

// QueryRunner returns a compact, ready-to-send text summary of a runner's
// status, last known checkpoint, and pace, for replying to a mesh "query"
// command.
func (s *CheckpointLogService) QueryRunner(ctx context.Context, bibNumber int) (string, error) {
	sess, err := s.session.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("getting session: %w", err)
	}
	if sess.EventID == nil {
		return "", domain.ErrNoSession
	}

	runner, err := s.runners.GetByBibInEvent(ctx, *sess.EventID, bibNumber)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return fmt.Sprintf("%d not found", bibNumber), nil
		}
		return "", fmt.Errorf("bib %d: %w", bibNumber, err)
	}

	checkpoints, err := s.checkpoints.List(ctx, runner.RaceID)
	if err != nil {
		return "", fmt.Errorf("listing checkpoints: %w", err)
	}
	logs, err := s.checkpointLogs.ListByRace(ctx, runner.RaceID)
	if err != nil {
		return "", fmt.Errorf("listing checkpoint logs: %w", err)
	}

	name := fmt.Sprintf("%s %s", runner.FirstName, runner.LastName)
	reply := fmt.Sprintf("%d %s: %s", runner.BibNumber, name, runner.Status)

	cp, log, ok := pace.LastLoggedCheckpoint(checkpoints, logs, runner.ID)
	if !ok {
		return reply + " not yet seen", nil
	}
	reply += fmt.Sprintf(" last %s %s", cp.DisplayName, log.RecordedAt.Format("15:04"))

	p := pace.ComputeRunnerPace(runner, checkpoints, logs)
	if p.PaceMinPerMile != nil {
		reply += " pace " + pace.FormatPace(*p.PaceMinPerMile)
	}

	return reply, nil
}

// CorrectLog creates or overwrites a runner's checkpoint log at an explicit
// time, for correcting a mis-logged bib after the fact — possibly at a
// checkpoint other than the one currently active at this station.
func (s *CheckpointLogService) CorrectLog(ctx context.Context, raceID, checkpointID, bibNumber int, timeStr string) (entity.CheckpointLog, error) {
	recordedAt, err := parseWallClockTime(s.loc, timeStr)
	if err != nil {
		return entity.CheckpointLog{}, err
	}

	runner, err := s.findRunnerByBibInRace(ctx, raceID, bibNumber)
	if err != nil {
		return entity.CheckpointLog{}, fmt.Errorf("bib %d: %w", bibNumber, err)
	}

	log, _, err := s.checkpointLogs.Upsert(ctx, entity.CheckpointLog{
		RunnerID:     runner.ID,
		CheckpointID: checkpointID,
		RecordedAt:   recordedAt,
		Source:       entity.SourceCorrection,
	})
	if err != nil {
		return entity.CheckpointLog{}, fmt.Errorf("upserting log: %w", err)
	}

	if runner.Status == entity.StatusUnknown {
		if err := s.runners.UpdateStatus(ctx, runner.ID, entity.StatusActive); err != nil {
			return entity.CheckpointLog{}, fmt.Errorf("updating runner status: %w", err)
		}
	}

	return log, nil
}

// DeleteLog removes a runner's checkpoint log for the given race+checkpoint+bib.
func (s *CheckpointLogService) DeleteLog(ctx context.Context, raceID, checkpointID, bibNumber int) error {
	runner, err := s.findRunnerByBibInRace(ctx, raceID, bibNumber)
	if err != nil {
		return fmt.Errorf("bib %d: %w", bibNumber, err)
	}
	return s.checkpointLogs.Delete(ctx, runner.ID, checkpointID)
}

func (s *CheckpointLogService) findRunnerByBibInRace(ctx context.Context, raceID, bibNumber int) (entity.Runner, error) {
	runners, err := s.runners.List(ctx, raceID)
	if err != nil {
		return entity.Runner{}, fmt.Errorf("listing runners: %w", err)
	}
	for _, r := range runners {
		if r.BibNumber == bibNumber {
			return r, nil
		}
	}
	return entity.Runner{}, domain.ErrNotFound
}

func activeCheckpointForRace(sess entity.ActiveSession, raceID int) (int, bool) {
	for _, sc := range sess.Checkpoints {
		if sc.RaceID == raceID {
			return sc.CheckpointID, true
		}
	}
	return 0, false
}

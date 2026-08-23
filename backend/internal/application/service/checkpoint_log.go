package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/pace"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

// maxSearchResults caps the number of runners a mesh "search" reply lists,
// keeping the reply short enough for a single mesh packet.
const maxSearchResults = 5

type CheckpointLogService struct {
	runners        portrepo.RunnerRepository
	checkpoints    portrepo.CheckpointRepository
	checkpointLogs portrepo.CheckpointLogRepository
	session        portrepo.ActiveSessionRepository
	races          portrepo.RaceRepository
	loc            *time.Location
}

func NewCheckpointLogService(
	runners portrepo.RunnerRepository,
	checkpoints portrepo.CheckpointRepository,
	checkpointLogs portrepo.CheckpointLogRepository,
	session portrepo.ActiveSessionRepository,
	races portrepo.RaceRepository,
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
		races:          races,
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
// checkpoint other than the one currently active at this station. dateStr is
// an optional YYYY-MM-DD date; when empty, today's date (in the service's
// configured timezone) is used, matching prior behavior.
func (s *CheckpointLogService) CorrectLog(ctx context.Context, raceID, checkpointID, bibNumber int, dateStr, timeStr string) (entity.CheckpointLog, error) {
	recordedAt, err := parseWallClockTime(s.loc, dateStr, timeStr)
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

// StationCheckpoints returns a compact summary of this station's active
// checkpoint for each race in the active event, for replying to a mesh
// "checkpoint" command.
func (s *CheckpointLogService) StationCheckpoints(ctx context.Context) (string, error) {
	sess, err := s.session.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("getting session: %w", err)
	}
	if sess.EventID == nil {
		return "", domain.ErrNoSession
	}
	if len(sess.Checkpoints) == 0 {
		return "no active checkpoint set for any race", nil
	}

	parts := make([]string, 0, len(sess.Checkpoints))
	for _, sc := range sess.Checkpoints {
		race, err := s.races.Get(ctx, sc.RaceID)
		if err != nil {
			return "", fmt.Errorf("getting race %d: %w", sc.RaceID, err)
		}
		cp, err := s.checkpoints.Get(ctx, sc.CheckpointID)
		if err != nil {
			return "", fmt.Errorf("getting checkpoint %d: %w", sc.CheckpointID, err)
		}
		parts = append(parts, fmt.Sprintf("%s %s", race.Name, cp.Code))
	}

	return strings.Join(parts, " | "), nil
}

// StationCount returns the number of checkpoint logs recorded at this
// station's active checkpoint(s), for replying to a mesh "count" command.
func (s *CheckpointLogService) StationCount(ctx context.Context) (string, error) {
	sess, err := s.session.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("getting session: %w", err)
	}
	if sess.EventID == nil {
		return "", domain.ErrNoSession
	}
	if len(sess.Checkpoints) == 0 {
		return "no active checkpoint set for any race", nil
	}

	total := 0
	parts := make([]string, 0, len(sess.Checkpoints))
	for _, sc := range sess.Checkpoints {
		logs, err := s.checkpointLogs.ListByRaceAndCheckpoint(ctx, sc.RaceID, sc.CheckpointID)
		if err != nil {
			return "", fmt.Errorf("listing logs for race %d: %w", sc.RaceID, err)
		}
		total += len(logs)

		cp, err := s.checkpoints.Get(ctx, sc.CheckpointID)
		if err != nil {
			return "", fmt.Errorf("getting checkpoint %d: %w", sc.CheckpointID, err)
		}
		if len(sess.Checkpoints) == 1 {
			return fmt.Sprintf("logged %d at %s", len(logs), cp.Code), nil
		}

		race, err := s.races.Get(ctx, sc.RaceID)
		if err != nil {
			return "", fmt.Errorf("getting race %d: %w", sc.RaceID, err)
		}
		parts = append(parts, fmt.Sprintf("%s %s=%d", race.Name, cp.Code, len(logs)))
	}

	return fmt.Sprintf("logged %d total (%s)", total, strings.Join(parts, ", ")), nil
}

// SearchRunners returns bib/name/race for runners in the active event whose
// last name contains the given text (case-insensitive), for replying to a
// mesh "search <name>" command.
func (s *CheckpointLogService) SearchRunners(ctx context.Context, lastName string) (string, error) {
	sess, err := s.session.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("getting session: %w", err)
	}
	if sess.EventID == nil {
		return "", domain.ErrNoSession
	}

	races, err := s.races.List(ctx, *sess.EventID)
	if err != nil {
		return "", fmt.Errorf("listing races: %w", err)
	}
	raceNames := make(map[int]string, len(races))
	for _, r := range races {
		raceNames[r.ID] = r.Name
	}

	needle := strings.ToLower(lastName)
	type match struct {
		runner   entity.Runner
		raceName string
	}
	var matches []match
	for _, r := range races {
		runners, err := s.runners.List(ctx, r.ID)
		if err != nil {
			return "", fmt.Errorf("listing runners for race %d: %w", r.ID, err)
		}
		for _, runner := range runners {
			if strings.Contains(strings.ToLower(runner.LastName), needle) {
				matches = append(matches, match{runner: runner, raceName: raceNames[r.ID]})
			}
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("no runners matching %q", lastName), nil
	}

	shown := matches
	truncated := 0
	if len(shown) > maxSearchResults {
		truncated = len(shown) - maxSearchResults
		shown = shown[:maxSearchResults]
	}

	parts := make([]string, 0, len(shown))
	for _, m := range shown {
		parts = append(parts, fmt.Sprintf("%d %s %s (%s)",
			m.runner.BibNumber, m.runner.FirstName, m.runner.LastName, m.raceName))
	}

	reply := fmt.Sprintf("%d match(es): %s", len(matches), strings.Join(parts, ", "))
	if truncated > 0 {
		reply += fmt.Sprintf(" +%d more", truncated)
	}
	return reply, nil
}

// CheckDuplicate reports whether a bib has already been logged at this
// station's active checkpoint for that runner's race, for replying to a mesh
// "dup <bib>" command.
func (s *CheckpointLogService) CheckDuplicate(ctx context.Context, bibNumber int) (string, error) {
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
	name := fmt.Sprintf("%s %s", runner.FirstName, runner.LastName)

	checkpointID, ok := activeCheckpointForRace(sess, runner.RaceID)
	if !ok {
		race, err := s.races.Get(ctx, runner.RaceID)
		if err != nil {
			return "", fmt.Errorf("getting race %d: %w", runner.RaceID, err)
		}
		return fmt.Sprintf("%d %s: no active checkpoint for %s", runner.BibNumber, name, race.Name), nil
	}

	cp, err := s.checkpoints.Get(ctx, checkpointID)
	if err != nil {
		return "", fmt.Errorf("getting checkpoint %d: %w", checkpointID, err)
	}

	logs, err := s.checkpointLogs.ListByRaceAndCheckpoint(ctx, runner.RaceID, checkpointID)
	if err != nil {
		return "", fmt.Errorf("listing logs: %w", err)
	}
	for _, log := range logs {
		if log.RunnerID == runner.ID {
			return fmt.Sprintf("%d %s: already logged at %s %s",
				runner.BibNumber, name, cp.Code, log.RecordedAt.Format("15:04")), nil
		}
	}

	return fmt.Sprintf("%d %s: not yet logged at %s", runner.BibNumber, name, cp.Code), nil
}

package service

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
	portrepo "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/repository"
	portsvc "github.com/kevinball/ares-bib-logger/backend/internal/domain/port/service"
)

type WinlinkService struct {
	runners        portrepo.RunnerRepository
	checkpoints    portrepo.CheckpointRepository
	checkpointLogs portrepo.CheckpointLogRepository
	session        portrepo.ActiveSessionRepository
}

func NewWinlinkService(
	runners portrepo.RunnerRepository,
	checkpoints portrepo.CheckpointRepository,
	checkpointLogs portrepo.CheckpointLogRepository,
	session portrepo.ActiveSessionRepository,
) *WinlinkService {
	return &WinlinkService{
		runners:        runners,
		checkpoints:    checkpoints,
		checkpointLogs: checkpointLogs,
		session:        session,
	}
}

var _ portsvc.WinlinkService = (*WinlinkService)(nil)

func (s *WinlinkService) Export(ctx context.Context, raceID int) (string, error) {
	sess, err := s.session.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("getting session: %w", err)
	}

	checkpointID, ok := activeCheckpointForRace(sess, raceID)
	if !ok {
		return "", fmt.Errorf("no active checkpoint for race %d", raceID)
	}

	cp, err := s.checkpoints.Get(ctx, checkpointID)
	if err != nil {
		return "", fmt.Errorf("getting checkpoint: %w", err)
	}

	runners, err := s.runners.List(ctx, raceID)
	if err != nil {
		return "", fmt.Errorf("listing runners: %w", err)
	}

	logs, err := s.checkpointLogs.ListByRaceAndCheckpoint(ctx, raceID, checkpointID)
	if err != nil {
		return "", fmt.Errorf("listing checkpoint logs: %w", err)
	}

	logByRunner := make(map[int]entity.CheckpointLog, len(logs))
	for _, l := range logs {
		logByRunner[l.RunnerID] = l
	}

	var sb strings.Builder
	sb.WriteString(cp.Code)
	sb.WriteByte('\n')

	for _, r := range runners {
		if log, seen := logByRunner[r.ID]; seen {
			sb.WriteString(log.RecordedAt.Format("15:04:05"))
		} else {
			switch r.Status {
			case entity.StatusDNS:
				sb.WriteString("DNS")
			case entity.StatusDNF:
				sb.WriteString("DNF")
			default:
				// blank — runner not yet seen at this checkpoint
			}
		}
		sb.WriteByte('\n')
	}

	return sb.String(), nil
}

func (s *WinlinkService) Import(ctx context.Context, raceID, checkpointID int, text string) (portsvc.WinlinkImportResult, error) {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) == 0 {
		return portsvc.WinlinkImportResult{}, nil
	}

	// Skip a non-numeric header line if present.
	start := 0
	if len(lines) > 0 && !looksLikeTimeOrStatus(lines[0]) {
		start = 1
	}
	lines = lines[start:]

	runners, err := s.runners.List(ctx, raceID)
	if err != nil {
		return portsvc.WinlinkImportResult{}, fmt.Errorf("listing runners: %w", err)
	}

	byOrder := make(map[int]entity.Runner, len(runners))
	for _, r := range runners {
		byOrder[r.SortOrder] = r
	}

	var result portsvc.WinlinkImportResult

	for i, line := range lines {
		line = strings.TrimSpace(line)
		sortOrder := i + 1

		if line == "" {
			result.Skipped++
			continue
		}

		runner, ok := byOrder[sortOrder]
		if !ok {
			result.Skipped++
			continue
		}

		upper := strings.ToUpper(line)
		switch upper {
		case "DNS":
			if err := s.runners.UpdateStatus(ctx, runner.ID, entity.StatusDNS); err != nil {
				return result, fmt.Errorf("updating DNS status for bib %d: %w", runner.BibNumber, err)
			}
			result.Updated++
		case "DNF":
			if err := s.runners.UpdateStatus(ctx, runner.ID, entity.StatusDNF); err != nil {
				return result, fmt.Errorf("updating DNF status for bib %d: %w", runner.BibNumber, err)
			}
			result.Updated++
		default:
			t, err := parseTimeOfDay(line)
			if err != nil {
				result.Skipped++
				continue
			}
			exists, err := s.checkpointLogs.ExistsByRunnerAndCheckpoint(ctx, runner.ID, checkpointID)
			if err != nil {
				return result, fmt.Errorf("checking duplicate for bib %d: %w", runner.BibNumber, err)
			}
			if exists {
				result.Skipped++
				continue
			}
			if _, err := s.checkpointLogs.Create(ctx, entity.CheckpointLog{
				RunnerID:     runner.ID,
				CheckpointID: checkpointID,
				RecordedAt:   t,
				Source:       entity.SourceWinlinkImport,
				RawMessage:   line,
			}); err != nil {
				return result, fmt.Errorf("creating log for bib %d: %w", runner.BibNumber, err)
			}
			result.Created++
		}
	}

	return result, nil
}

// looksLikeTimeOrStatus returns true if the line appears to be a time (HH:MM or HH:MM:SS) or DNS/DNF.
func looksLikeTimeOrStatus(s string) bool {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "DNS" || s == "DNF" || s == "" {
		return true
	}
	return len(s) >= 5 && s[2] == ':' && unicode.IsDigit(rune(s[0]))
}

// parseTimeOfDay parses HH:MM:SS or HH:MM, combining with today's date.
func parseTimeOfDay(s string) (time.Time, error) {
	now := time.Now()
	base := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	for _, layout := range []string{"15:04:05", "15:04"} {
		t, err := time.Parse(layout, s)
		if err == nil {
			return base.Add(time.Duration(t.Hour())*time.Hour +
				time.Duration(t.Minute())*time.Minute +
				time.Duration(t.Second())*time.Second), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", s)
}

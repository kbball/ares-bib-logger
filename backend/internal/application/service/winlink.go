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
	races          portrepo.RaceRepository
	loc            *time.Location
}

func NewWinlinkService(
	runners portrepo.RunnerRepository,
	checkpoints portrepo.CheckpointRepository,
	checkpointLogs portrepo.CheckpointLogRepository,
	session portrepo.ActiveSessionRepository,
	races portrepo.RaceRepository,
	loc *time.Location,
) *WinlinkService {
	if loc == nil {
		loc = time.Local
	}
	return &WinlinkService{
		runners:        runners,
		checkpoints:    checkpoints,
		checkpointLogs: checkpointLogs,
		session:        session,
		races:          races,
		loc:            loc,
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

	// For MOVED runners, find the race they transferred to.
	movedToRace := make(map[int]string) // bib → target race name
	if sess.EventID != nil {
		var movedBibs []int
		for _, r := range runners {
			if r.Status == entity.StatusMoved {
				movedBibs = append(movedBibs, r.BibNumber)
			}
		}
		if len(movedBibs) > 0 {
			movedBibSet := make(map[int]bool, len(movedBibs))
			for _, b := range movedBibs {
				movedBibSet[b] = true
			}
			allRaces, err := s.races.List(ctx, *sess.EventID)
			if err != nil {
				return "", fmt.Errorf("listing races for moved runners: %w", err)
			}
			for _, race := range allRaces {
				if race.ID == raceID {
					continue
				}
				raceRunners, err := s.runners.List(ctx, race.ID)
				if err != nil {
					return "", fmt.Errorf("listing runners for race %d: %w", race.ID, err)
				}
				for _, r := range raceRunners {
					if movedBibSet[r.BibNumber] && r.Status != entity.StatusMoved {
						movedToRace[r.BibNumber] = race.Name
					}
				}
			}
		}
	}

	var sb strings.Builder
	sb.WriteString(cp.Code)
	sb.WriteByte('\n')

	for _, r := range runners {
		if log, seen := logByRunner[r.ID]; seen {
			sb.WriteString(log.RecordedAt.In(s.loc).Format("15:04"))
		} else {
			switch r.Status {
			case entity.StatusDNS:
				sb.WriteString("DNS")
			case entity.StatusDNF:
				sb.WriteString("DNF")
			case entity.StatusMoved:
				if raceName, ok := movedToRace[r.BibNumber]; ok {
					sb.WriteString("MOVED " + raceName)
				} else {
					sb.WriteString("MOVED")
				}
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

	skip := func(pos, bib int, reason string) {
		result.Skipped++
		result.SkippedDetails = append(result.SkippedDetails, portsvc.WinlinkSkipDetail{
			Position:  pos,
			BibNumber: bib,
			Reason:    reason,
		})
	}

	for i, line := range lines {
		line = strings.TrimSpace(line)
		sortOrder := i + 1
		pos := i + 1

		if line == "" {
			skip(pos, 0, "blank")
			continue
		}

		runner, ok := byOrder[sortOrder]
		if !ok {
			skip(pos, 0, "no_runner")
			continue
		}

		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "MOVED") {
			// Runner was transferred out of this race; no action needed.
			skip(pos, runner.BibNumber, "moved")
			continue
		}
		switch upper {
		case "DNS", "DNF":
			status := entity.StatusDNS
			if upper == "DNF" {
				status = entity.StatusDNF
			}
			if err := s.runners.UpdateStatus(ctx, runner.ID, status); err != nil {
				return result, fmt.Errorf("updating %s status for bib %d: %w", upper, runner.BibNumber, err)
			}
			if _, _, err := s.checkpointLogs.Upsert(ctx, entity.CheckpointLog{
				RunnerID:     runner.ID,
				CheckpointID: checkpointID,
				RecordedAt:   time.Now().UTC(),
				Source:       entity.SourceWinlinkImport,
				RawMessage:   upper,
			}); err != nil {
				return result, fmt.Errorf("upserting %s log for bib %d: %w", upper, runner.BibNumber, err)
			}
			result.Updated++
		default:
			t, err := s.parseTimeOfDay(line)
			if err != nil {
				skip(pos, runner.BibNumber, "parse_error")
				continue
			}
			_, wasCreated, err := s.checkpointLogs.Upsert(ctx, entity.CheckpointLog{
				RunnerID:     runner.ID,
				CheckpointID: checkpointID,
				RecordedAt:   t,
				Source:       entity.SourceWinlinkImport,
				RawMessage:   line,
			})
			if err != nil {
				return result, fmt.Errorf("upserting log for bib %d: %w", runner.BibNumber, err)
			}
			if runner.Status == entity.StatusUnknown {
				if err := s.runners.UpdateStatus(ctx, runner.ID, entity.StatusActive); err != nil {
					return result, fmt.Errorf("updating status for bib %d: %w", runner.BibNumber, err)
				}
			}
			if wasCreated {
				result.Created++
			} else {
				result.Updated++
			}
		}
	}

	return result, nil
}

// looksLikeTimeOrStatus returns true if the line appears to be a data row:
// a time (HH:MM or HH:MM:SS), DNS, DNF, blank, or MOVED (with optional race name).
// Returns false for a station-name header such as "AS #6".
func looksLikeTimeOrStatus(s string) bool {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" || s == "DNS" || s == "DNF" {
		return true
	}
	if strings.HasPrefix(s, "MOVED") {
		return true
	}
	// HH:MM or HH:MM:SS
	if len(s) >= 5 && s[2] == ':' && unicode.IsDigit(rune(s[0])) {
		return true
	}
	// H:MM or H:MM:SS (single-digit hour)
	return len(s) >= 4 && s[1] == ':' && unicode.IsDigit(rune(s[0]))
}

// parseTimeOfDay parses HH:MM:SS or HH:MM as a wall-clock time on today's date
// in the service's configured timezone.
func (s *WinlinkService) parseTimeOfDay(str string) (time.Time, error) {
	now := time.Now()
	base := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, s.loc)

	for _, layout := range []string{"15:04:05", "15:04"} {
		t, err := time.Parse(layout, str)
		if err == nil {
			return base.Add(time.Duration(t.Hour())*time.Hour +
				time.Duration(t.Minute())*time.Minute +
				time.Duration(t.Second())*time.Second), nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse time: %q", str)
}

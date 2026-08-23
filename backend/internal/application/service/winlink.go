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
	sb.WriteString(cp.DisplayName)
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

// rowKind classifies a single pasted line, independent of whether it will be
// written to the DB (Import) or merely reported (Preview).
type rowKind string

const (
	rowBlank      rowKind = "blank"
	rowNoRunner   rowKind = "no_runner"
	rowMoved      rowKind = "moved"
	rowParseError rowKind = "parse_error"
	rowDNS        rowKind = "dns"
	rowDNF        rowKind = "dnf"
	rowTime       rowKind = "time"
)

// parsedRow is the pure, side-effect-free classification of one pasted line.
type parsedRow struct {
	position   int
	sortOrder  int
	runner     entity.Runner
	hasRunner  bool
	kind       rowKind
	raw        string // trimmed original line
	parsedTime time.Time
}

// parseImportRows classifies each data row (after header stripping) by
// looking up the runner at that sort_order and inspecting the line's
// content. It performs no I/O, so Import and Preview can share it and can
// never disagree about what a row *is* — only about what to do with it.
func (s *WinlinkService) parseImportRows(text string, byOrder map[int]entity.Runner) []parsedRow {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if len(lines) == 0 {
		return nil
	}

	// Skip a non-numeric header line if present.
	start := 0
	if len(lines) > 0 && !looksLikeTimeOrStatus(lines[0]) {
		start = 1
	}
	lines = lines[start:]

	rows := make([]parsedRow, 0, len(lines))
	for i, line := range lines {
		line = strings.TrimSpace(line)
		sortOrder := i + 1
		pos := i + 1

		if line == "" {
			rows = append(rows, parsedRow{position: pos, sortOrder: sortOrder, kind: rowBlank})
			continue
		}

		runner, ok := byOrder[sortOrder]
		if !ok {
			rows = append(rows, parsedRow{position: pos, sortOrder: sortOrder, kind: rowNoRunner, raw: line})
			continue
		}

		upper := strings.ToUpper(line)
		if strings.HasPrefix(upper, "MOVED") {
			rows = append(rows, parsedRow{
				position: pos, sortOrder: sortOrder, runner: runner, hasRunner: true, kind: rowMoved, raw: line,
			})
			continue
		}
		switch upper {
		case "DNS":
			rows = append(rows, parsedRow{
				position: pos, sortOrder: sortOrder, runner: runner, hasRunner: true, kind: rowDNS, raw: upper,
			})
		case "DNF":
			rows = append(rows, parsedRow{
				position: pos, sortOrder: sortOrder, runner: runner, hasRunner: true, kind: rowDNF, raw: upper,
			})
		default:
			t, err := s.parseTimeOfDay(line)
			if err != nil {
				rows = append(rows, parsedRow{
					position: pos, sortOrder: sortOrder, runner: runner, hasRunner: true, kind: rowParseError, raw: line,
				})
				continue
			}
			rows = append(rows, parsedRow{
				position: pos, sortOrder: sortOrder, runner: runner, hasRunner: true, kind: rowTime, raw: line, parsedTime: t,
			})
		}
	}
	return rows
}

func (s *WinlinkService) Import(ctx context.Context, raceID, checkpointID int, text string) (portsvc.WinlinkImportResult, error) {
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

	for _, row := range s.parseImportRows(text, byOrder) {
		switch row.kind {
		case rowBlank:
			skip(row.position, 0, "blank")
		case rowNoRunner:
			skip(row.position, 0, "no_runner")
		case rowMoved:
			// Runner was transferred out of this race; no action needed.
			skip(row.position, row.runner.BibNumber, "moved")
		case rowParseError:
			skip(row.position, row.runner.BibNumber, "parse_error")
		case rowDNS, rowDNF:
			status := entity.StatusDNS
			if row.kind == rowDNF {
				status = entity.StatusDNF
			}
			if err := s.runners.UpdateStatus(ctx, row.runner.ID, status); err != nil {
				return result, fmt.Errorf("updating %s status for bib %d: %w", row.raw, row.runner.BibNumber, err)
			}
			if _, _, err := s.checkpointLogs.Upsert(ctx, entity.CheckpointLog{
				RunnerID:     row.runner.ID,
				CheckpointID: checkpointID,
				RecordedAt:   time.Now().UTC(),
				Source:       entity.SourceWinlinkImport,
				RawMessage:   row.raw,
			}); err != nil {
				return result, fmt.Errorf("upserting %s log for bib %d: %w", row.raw, row.runner.BibNumber, err)
			}
			result.Updated++
		case rowTime:
			_, wasCreated, err := s.checkpointLogs.Upsert(ctx, entity.CheckpointLog{
				RunnerID:     row.runner.ID,
				CheckpointID: checkpointID,
				RecordedAt:   row.parsedTime,
				Source:       entity.SourceWinlinkImport,
				RawMessage:   row.raw,
			})
			if err != nil {
				return result, fmt.Errorf("upserting log for bib %d: %w", row.runner.BibNumber, err)
			}
			if row.runner.Status == entity.StatusUnknown {
				if err := s.runners.UpdateStatus(ctx, row.runner.ID, entity.StatusActive); err != nil {
					return result, fmt.Errorf("updating status for bib %d: %w", row.runner.BibNumber, err)
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

// Preview classifies every row the same way Import would, without writing
// anything to the database, so an operator can review what would happen
// before committing.
func (s *WinlinkService) Preview(ctx context.Context, raceID, checkpointID int, text string) (portsvc.WinlinkPreviewResult, error) {
	runners, err := s.runners.List(ctx, raceID)
	if err != nil {
		return portsvc.WinlinkPreviewResult{}, fmt.Errorf("listing runners: %w", err)
	}

	byOrder := make(map[int]entity.Runner, len(runners))
	for _, r := range runners {
		byOrder[r.SortOrder] = r
	}

	existingLogs, err := s.checkpointLogs.ListByRaceAndCheckpoint(ctx, raceID, checkpointID)
	if err != nil {
		return portsvc.WinlinkPreviewResult{}, fmt.Errorf("listing checkpoint logs: %w", err)
	}
	hasLog := make(map[int]bool, len(existingLogs))
	for _, l := range existingLogs {
		hasLog[l.RunnerID] = true
	}

	var result portsvc.WinlinkPreviewResult

	addRow := func(row portsvc.WinlinkRowOutcome) {
		switch row.Kind {
		case "create":
			result.Created++
		case "update":
			result.Updated++
		case "skip":
			result.Skipped++
		}
		result.Rows = append(result.Rows, row)
	}

	for _, row := range s.parseImportRows(text, byOrder) {
		switch row.kind {
		case rowBlank:
			addRow(portsvc.WinlinkRowOutcome{Position: row.position, Kind: "skip", Reason: "blank"})
		case rowNoRunner:
			addRow(portsvc.WinlinkRowOutcome{Position: row.position, Kind: "skip", Reason: "no_runner"})
		case rowMoved:
			addRow(portsvc.WinlinkRowOutcome{
				Position: row.position, BibNumber: row.runner.BibNumber, Kind: "skip", Value: row.raw, Reason: "moved",
			})
		case rowParseError:
			addRow(portsvc.WinlinkRowOutcome{
				Position: row.position, BibNumber: row.runner.BibNumber, Kind: "skip", Value: row.raw, Reason: "parse_error",
			})
		case rowDNS, rowDNF:
			addRow(portsvc.WinlinkRowOutcome{
				Position: row.position, BibNumber: row.runner.BibNumber, Kind: "update", Value: row.raw,
			})
		case rowTime:
			kind := "create"
			if hasLog[row.runner.ID] {
				kind = "update"
			}
			addRow(portsvc.WinlinkRowOutcome{
				Position: row.position, BibNumber: row.runner.BibNumber, Kind: kind, Value: row.raw,
			})
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

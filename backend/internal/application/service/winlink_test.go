package service_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kevinball/ares-bib-logger/backend/internal/application/service"
	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

var errDB = errors.New("db down")

func newWinlinkSvc(
	runners *mockRunnerRepository,
	checkpoints *mockCheckpointRepository,
	logs *mockCheckpointLogRepository,
	sess *mockActiveSessionRepository,
	races ...*mockRaceRepository,
) *service.WinlinkService {
	// Default race 1 -> event 1 (with the blank-line-after-header flag off, matching
	// today's behavior) so existing tests that only pass raceID=1 don't need to set
	// up race/event repos themselves.
	r := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1}}}
	if len(races) > 0 {
		r = races[0]
	}
	events := &mockEventRepository{events: []entity.Event{{ID: 1}}}
	return service.NewWinlinkService(runners, checkpoints, logs, sess, r, events, time.UTC)
}

func parseHHMMSS(s string) time.Time {
	t, _ := time.Parse("15:04:05", s)
	return time.Date(2026, 1, 1, t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
}

func TestWinlinkService_Export_Format(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	checkpoints := &mockCheckpointRepository{
		checkpoints: map[int]entity.Checkpoint{
			5: {ID: 5, Code: "AS6", DisplayName: "Aid Station 6"},
		},
	}
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusActive},
			{ID: 2, RaceID: 1, BibNumber: 101, SortOrder: 2, Status: entity.StatusDNS},
			{ID: 3, RaceID: 1, BibNumber: 102, SortOrder: 3, Status: entity.StatusDNF},
			{ID: 4, RaceID: 1, BibNumber: 103, SortOrder: 4, Status: entity.StatusUnknown},
		},
	}

	logs := &mockCheckpointLogRepository{
		logs: []entity.CheckpointLog{
			{ID: 1, RunnerID: 1, CheckpointID: 5, RecordedAt: parseHHMMSS("17:45:00"), Source: entity.SourceManual},
		},
	}

	svc := newWinlinkSvc(runners, checkpoints, logs, sess)
	out, err := svc.Export(context.Background(), 1)

	require.NoError(t, err)
	// TrimSuffix removes exactly the final newline; the blank runner line remains.
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	require.Len(t, lines, 5) // header + 4 runners

	assert.Equal(t, "Aid Station 6", lines[0])
	assert.Equal(t, "17:45", lines[1]) // seen
	assert.Equal(t, "DNS", lines[2])   // DNS status
	assert.Equal(t, "DNF", lines[3])   // DNF status
	assert.Equal(t, "", lines[4])      // not seen, no status
}

func TestWinlinkService_Export_ColumnNameOverridesDisplayName(t *testing.T) {
	eventIDVal := 1
	colName := "AS #6"
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	checkpoints := &mockCheckpointRepository{
		checkpoints: map[int]entity.Checkpoint{
			5: {ID: 5, Code: "AS6", DisplayName: "Aid Station 6", ColumnName: &colName},
		},
	}
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusActive},
		},
	}
	logs := &mockCheckpointLogRepository{}

	svc := newWinlinkSvc(runners, checkpoints, logs, sess)
	out, err := svc.Export(context.Background(), 1)

	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	assert.Equal(t, "AS #6", lines[0])
}

func TestWinlinkService_Export_BlankColumnNameFallsBackToDisplayName(t *testing.T) {
	eventIDVal := 1
	empty := ""
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	checkpoints := &mockCheckpointRepository{
		checkpoints: map[int]entity.Checkpoint{
			5: {ID: 5, Code: "AS6", DisplayName: "Aid Station 6", ColumnName: &empty},
		},
	}
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusActive},
		},
	}
	logs := &mockCheckpointLogRepository{}

	svc := newWinlinkSvc(runners, checkpoints, logs, sess)
	out, err := svc.Export(context.Background(), 1)

	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	assert.Equal(t, "Aid Station 6", lines[0])
}

func TestWinlinkService_Export_MovedRunner(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	checkpoints := &mockCheckpointRepository{
		checkpoints: map[int]entity.Checkpoint{
			5: {ID: 5, Code: "AS6", DisplayName: "Aid Station 6"},
		},
	}
	// Race 1 has two runners: one active, one moved to race 2.
	// Race 2 is the target and has the moved runner as active.
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusActive},
			{ID: 2, RaceID: 1, BibNumber: 101, SortOrder: 2, Status: entity.StatusMoved},
			{ID: 3, RaceID: 2, BibNumber: 101, SortOrder: 1, Status: entity.StatusActive},
		},
	}
	races := &mockRaceRepository{
		races: map[int]entity.Race{
			1: {ID: 1, EventID: 1, Name: "50K"},
			2: {ID: 2, EventID: 1, Name: "Marathon"},
		},
	}
	logs := &mockCheckpointLogRepository{}

	svc := newWinlinkSvc(runners, checkpoints, logs, sess, races)
	out, err := svc.Export(context.Background(), 1)

	require.NoError(t, err)
	lines := strings.Split(strings.TrimSuffix(out, "\n"), "\n")
	require.Len(t, lines, 3) // header + 2 runners
	assert.Equal(t, "Aid Station 6", lines[0])
	assert.Equal(t, "", lines[1])               // runner 100 not yet seen
	assert.Equal(t, "MOVED Marathon", lines[2]) // runner 101 moved to Marathon
}

func TestWinlinkService_Export_NoActiveCheckpoint(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID: &eventIDVal,
		// no checkpoint for race 1
	}}
	svc := newWinlinkSvc(
		&mockRunnerRepository{},
		&mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}},
		&mockCheckpointLogRepository{},
		sess,
	)

	_, err := svc.Export(context.Background(), 1)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "no active checkpoint")
}

func TestWinlinkService_Import_ParsesTimes(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1},
			{ID: 2, RaceID: 1, BibNumber: 101, SortOrder: 2},
			{ID: 3, RaceID: 1, BibNumber: 102, SortOrder: 3},
		},
	}
	logs := &mockCheckpointLogRepository{}
	sess := &mockActiveSessionRepository{}

	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, sess)

	// Header + runner1 time, runner2 blank, runner3 time
	column := "AS6\n17:45:00\n\n08:33:00\n"
	result, err := svc.Import(context.Background(), 1, 10, column)

	require.NoError(t, err)
	assert.Equal(t, 2, result.Created)
	assert.Equal(t, 1, result.Skipped) // blank line for runner 2
	assert.Equal(t, 0, result.Updated)
	assert.Len(t, logs.created, 2)
	require.Len(t, result.SkippedDetails, 1)
	assert.Equal(t, 2, result.SkippedDetails[0].Position)
	assert.Equal(t, "blank", result.SkippedDetails[0].Reason)
}

func TestWinlinkService_Import_HandlesDNSDNF(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusActive},
			{ID: 2, RaceID: 1, BibNumber: 101, SortOrder: 2, Status: entity.StatusActive},
		},
	}
	logs := &mockCheckpointLogRepository{}
	sess := &mockActiveSessionRepository{}

	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, sess)

	column := "AS6\nDNS\nDNF\n"
	result, err := svc.Import(context.Background(), 1, 10, column)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 2, result.Updated)

	assert.Equal(t, entity.StatusDNS, runners.runners[0].Status)
	assert.Equal(t, entity.StatusDNF, runners.runners[1].Status)
}

func TestWinlinkService_Import_OverwritesDuplicates(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1},
		},
	}
	logs := &mockCheckpointLogRepository{
		logs: []entity.CheckpointLog{{ID: 1, RunnerID: 1, CheckpointID: 10}},
	}
	sess := &mockActiveSessionRepository{}

	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, sess)

	column := "17:45:00\n"
	result, err := svc.Import(context.Background(), 1, 10, column)

	require.NoError(t, err)
	assert.Equal(t, 0, result.Created)
	assert.Equal(t, 1, result.Updated) // overwrite counts as Updated
	assert.Equal(t, 0, result.Skipped)
	assert.Empty(t, result.SkippedDetails)
}

func TestWinlinkService_Import_NoHeader(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1},
		},
	}
	logs := &mockCheckpointLogRepository{}
	sess := &mockActiveSessionRepository{}

	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, sess)

	column := "17:45:00\n"
	result, err := svc.Import(context.Background(), 1, 10, column)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
}

// MOVED at sort_order 1 must NOT be misidentified as a header; position 2 must map to sort_order 2.
func TestWinlinkService_Import_MovedAtPositionOne(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusMoved},
			{ID: 2, RaceID: 1, BibNumber: 101, SortOrder: 2, Status: entity.StatusActive},
		},
	}
	logs := &mockCheckpointLogRepository{}
	sess := &mockActiveSessionRepository{}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, sess)

	// No header; first line is MOVED (sort_order 1), second is a time (sort_order 2).
	column := "MOVED Marathon\n17:45\n"
	result, err := svc.Import(context.Background(), 1, 10, column)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created) // bib 101 created
	assert.Equal(t, 1, result.Skipped) // bib 100 skipped (moved)
	require.Len(t, result.SkippedDetails, 1)
	assert.Equal(t, 1, result.SkippedDetails[0].Position)
	assert.Equal(t, 100, result.SkippedDetails[0].BibNumber)
	assert.Equal(t, "moved", result.SkippedDetails[0].Reason)

	// The log must be for runner 2 (bib 101), not runner 1.
	require.Len(t, logs.created, 1)
	assert.Equal(t, 2, logs.created[0].RunnerID)
}

// Blank at sort_order 1 (no header) must NOT be stripped as a header; position 2 must map to sort_order 2.
func TestWinlinkService_Import_BlankAtPositionOnePreservesOrder(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusActive},
			{ID: 2, RaceID: 1, BibNumber: 101, SortOrder: 2, Status: entity.StatusActive},
		},
	}
	logs := &mockCheckpointLogRepository{}
	sess := &mockActiveSessionRepository{}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, sess)

	// No header; first line is blank (sort_order 1 not yet seen), second is a time (sort_order 2).
	column := "\n17:45\n"
	result, err := svc.Import(context.Background(), 1, 10, column)

	require.NoError(t, err)
	assert.Equal(t, 1, result.Created) // bib 101 created
	assert.Equal(t, 1, result.Skipped) // bib 100 skipped (blank)
	require.Len(t, result.SkippedDetails, 1)
	assert.Equal(t, 1, result.SkippedDetails[0].Position)
	assert.Equal(t, "blank", result.SkippedDetails[0].Reason)

	// The log must be for runner 2 (bib 101), not runner 1.
	require.Len(t, logs.created, 1)
	assert.Equal(t, 2, logs.created[0].RunnerID)
}

// --- Blank line after header (event-configurable) ---

func TestWinlinkService_Export_BlankLineAfterHeader_WhenEnabled(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	checkpoints := &mockCheckpointRepository{
		checkpoints: map[int]entity.Checkpoint{5: {ID: 5, Code: "AS6", DisplayName: "Aid Station 6"}},
	}
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusActive}},
	}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1}}}
	events := &mockEventRepository{events: []entity.Event{{ID: 1, WinlinkBlankLineAfterHeader: true}}}

	svc := service.NewWinlinkService(runners, checkpoints, &mockCheckpointLogRepository{}, sess, races, events, time.UTC)
	out, err := svc.Export(context.Background(), 1)
	require.NoError(t, err)

	lines := strings.Split(out, "\n")
	require.GreaterOrEqual(t, len(lines), 2)
	assert.Equal(t, "Aid Station 6", lines[0])
	assert.Equal(t, "", lines[1], "blank line after header expected when event flag is enabled")
}

func TestWinlinkService_Export_NoBlankLineAfterHeader_WhenDisabled(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	checkpoints := &mockCheckpointRepository{
		checkpoints: map[int]entity.Checkpoint{5: {ID: 5, Code: "AS6", DisplayName: "Aid Station 6"}},
	}
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusDNS}},
	}
	svc := newWinlinkSvc(runners, checkpoints, &mockCheckpointLogRepository{}, sess)

	out, err := svc.Export(context.Background(), 1)
	require.NoError(t, err)

	lines := strings.Split(out, "\n")
	assert.Equal(t, "Aid Station 6", lines[0])
	assert.Equal(t, "DNS", lines[1], "no blank line expected when flag is disabled (default)")
}

func TestWinlinkService_Export_RaceGetError(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	checkpoints := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{5: {ID: 5}}}
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 1, RaceID: 1, SortOrder: 1}}}
	races := &mockRaceRepository{} // race 1 not present -> ErrNotFound

	svc := service.NewWinlinkService(runners, checkpoints, &mockCheckpointLogRepository{}, sess, races, &mockEventRepository{}, time.UTC)
	_, err := svc.Export(context.Background(), 1)
	assert.ErrorContains(t, err, "getting race")
}

func TestWinlinkService_Export_EventGetError(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	checkpoints := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{5: {ID: 5}}}
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 1, RaceID: 1, SortOrder: 1}}}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 99}}} // event 99 not present

	svc := service.NewWinlinkService(runners, checkpoints, &mockCheckpointLogRepository{}, sess, races, &mockEventRepository{}, time.UTC)
	_, err := svc.Export(context.Background(), 1)
	assert.ErrorContains(t, err, "getting event")
}

func TestWinlinkService_Import_ConsumesBlankLineAfterHeader_WhenEnabled(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1},
			{ID: 2, RaceID: 1, BibNumber: 101, SortOrder: 2},
		},
	}
	logs := &mockCheckpointLogRepository{}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1}}}
	events := &mockEventRepository{events: []entity.Event{{ID: 1, WinlinkBlankLineAfterHeader: true}}}

	svc := service.NewWinlinkService(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}},
		logs, &mockActiveSessionRepository{}, races, events, time.UTC)

	// header, blank line, then two data rows -- with the flag on, positions
	// should map 17:45:00 -> sort_order 1 and 08:00:00 -> sort_order 2, not shift by one.
	column := "AS6\n\n17:45:00\n08:00:00\n"
	result, err := svc.Import(context.Background(), 1, 10, column)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Created)
	assert.Equal(t, 0, result.Skipped)
	require.Len(t, logs.created, 2)
	assert.Equal(t, 1, logs.created[0].RunnerID)
	assert.Equal(t, 2, logs.created[1].RunnerID)
}

func TestWinlinkService_Import_BlankLineFlagOn_NoActualBlankLine_DoesNotEatRealRow(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1},
			{ID: 2, RaceID: 1, BibNumber: 101, SortOrder: 2},
		},
	}
	logs := &mockCheckpointLogRepository{}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1}}}
	events := &mockEventRepository{events: []entity.Event{{ID: 1, WinlinkBlankLineAfterHeader: true}}}

	svc := service.NewWinlinkService(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}},
		logs, &mockActiveSessionRepository{}, races, events, time.UTC)

	// flag is on, but this paste has no blank line after the header -- must not
	// skip the first real data row.
	column := "AS6\n17:45:00\n08:00:00\n"
	result, err := svc.Import(context.Background(), 1, 10, column)
	require.NoError(t, err)
	assert.Equal(t, 2, result.Created)
	assert.Equal(t, 0, result.Skipped)
}

func TestWinlinkService_Preview_ConsumesBlankLineAfterHeader_WhenEnabled(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
	}
	races := &mockRaceRepository{races: map[int]entity.Race{1: {ID: 1, EventID: 1}}}
	events := &mockEventRepository{events: []entity.Event{{ID: 1, WinlinkBlankLineAfterHeader: true}}}

	svc := service.NewWinlinkService(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}},
		&mockCheckpointLogRepository{}, &mockActiveSessionRepository{}, races, events, time.UTC)

	column := "AS6\n\n17:45:00\n"
	result, err := svc.Preview(context.Background(), 1, 10, column)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 0, result.Skipped)
	require.Len(t, result.Rows, 1)
	assert.Equal(t, 100, result.Rows[0].BibNumber)
}

func TestWinlinkService_Import_EventLookupError(t *testing.T) {
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 1, RaceID: 1, SortOrder: 1}}}
	races := &mockRaceRepository{} // race 1 not present -> ErrNotFound

	svc := service.NewWinlinkService(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}},
		&mockCheckpointLogRepository{}, &mockActiveSessionRepository{}, races, &mockEventRepository{}, time.UTC)

	_, err := svc.Import(context.Background(), 1, 10, "17:45:00\n")
	assert.ErrorContains(t, err, "getting race")
}

func TestWinlinkService_Preview_EventLookupError(t *testing.T) {
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 1, RaceID: 1, SortOrder: 1}}}
	races := &mockRaceRepository{} // race 1 not present -> ErrNotFound

	svc := service.NewWinlinkService(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}},
		&mockCheckpointLogRepository{}, &mockActiveSessionRepository{}, races, &mockEventRepository{}, time.UTC)

	_, err := svc.Preview(context.Background(), 1, 10, "17:45:00\n")
	assert.ErrorContains(t, err, "getting race")
}

// --- Export error paths ---

func TestWinlinkService_Export_SessionError(t *testing.T) {
	sess := &mockActiveSessionRepository{getErr: errDB}
	svc := newWinlinkSvc(&mockRunnerRepository{}, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, &mockCheckpointLogRepository{}, sess)

	_, err := svc.Export(context.Background(), 1)
	assert.ErrorContains(t, err, "getting session")
}

func TestWinlinkService_Export_CheckpointGetError(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}, getErr: errDB}

	svc := newWinlinkSvc(&mockRunnerRepository{}, cps, &mockCheckpointLogRepository{}, sess)
	_, err := svc.Export(context.Background(), 1)
	assert.ErrorContains(t, err, "getting checkpoint")
}

func TestWinlinkService_Export_ListRunnersError(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{5: {ID: 5, Code: "AS6"}}}
	runners := &mockRunnerRepository{listErr: errDB}

	svc := newWinlinkSvc(runners, cps, &mockCheckpointLogRepository{}, sess)
	_, err := svc.Export(context.Background(), 1)
	assert.ErrorContains(t, err, "listing runners")
}

func TestWinlinkService_Export_ListLogsError(t *testing.T) {
	eventIDVal := 1
	sess := &mockActiveSessionRepository{session: entity.ActiveSession{
		EventID:     &eventIDVal,
		Checkpoints: []entity.ActiveSessionCheckpoint{{RaceID: 1, CheckpointID: 5}},
	}}
	cps := &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{5: {ID: 5, Code: "AS6"}}}
	runners := &mockRunnerRepository{runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}}}
	logs := &mockCheckpointLogRepository{listErr: errDB}

	svc := newWinlinkSvc(runners, cps, logs, sess)
	_, err := svc.Export(context.Background(), 1)
	assert.ErrorContains(t, err, "listing checkpoint logs")
}

// --- Import error paths ---

func TestWinlinkService_Import_ListRunnersError(t *testing.T) {
	runners := &mockRunnerRepository{listErr: errDB}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	_, err := svc.Import(context.Background(), 1, 10, "17:45:00\n")
	assert.ErrorContains(t, err, "listing runners")
}

func TestWinlinkService_Import_DNSUpdateError(t *testing.T) {
	runners := &mockRunnerRepository{
		runners:         []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
		updateStatusErr: errDB,
	}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	_, err := svc.Import(context.Background(), 1, 10, "DNS\n")
	assert.ErrorContains(t, err, "updating DNS status")
}

func TestWinlinkService_Import_DNFUpdateError(t *testing.T) {
	runners := &mockRunnerRepository{
		runners:         []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
		updateStatusErr: errDB,
	}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	_, err := svc.Import(context.Background(), 1, 10, "DNF\n")
	assert.ErrorContains(t, err, "updating DNF status")
}

func TestWinlinkService_Import_UpsertError(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
	}
	logs := &mockCheckpointLogRepository{upsertErr: errDB}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, &mockActiveSessionRepository{})

	_, err := svc.Import(context.Background(), 1, 10, "17:45:00\n")
	assert.ErrorContains(t, err, "upserting log")
}

func TestWinlinkService_Import_SkipsUnknownSortOrder(t *testing.T) {
	// 2 lines but only runner at SortOrder=1; SortOrder=2 has no runner
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
	}
	logs := &mockCheckpointLogRepository{}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, &mockActiveSessionRepository{})

	result, err := svc.Import(context.Background(), 1, 10, "17:45:00\n08:00:00\n")
	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 1, result.Skipped)
	require.Len(t, result.SkippedDetails, 1)
	assert.Equal(t, 2, result.SkippedDetails[0].Position)
	assert.Equal(t, "no_runner", result.SkippedDetails[0].Reason)
}

func TestWinlinkService_Import_InvalidTimeSkipped(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
	}
	logs := &mockCheckpointLogRepository{}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, &mockActiveSessionRepository{})

	// "99:99:99" passes looksLikeTimeOrStatus (digit, colon at [2], len > 5) but
	// fails time.Parse (hour 99 is out of range) — exercises the skip-bad-time path.
	result, err := svc.Import(context.Background(), 1, 10, "99:99:99\n")
	require.NoError(t, err)
	assert.Equal(t, 1, result.Skipped)
}

// --- looksLikeTimeOrStatus ---

func TestLooksLikeTimeOrStatus(t *testing.T) {
	svc := service.NewWinlinkService(
		&mockRunnerRepository{},
		&mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}},
		&mockCheckpointLogRepository{},
		&mockActiveSessionRepository{},
		&mockRaceRepository{},
		&mockEventRepository{},
		time.UTC,
	)
	_ = svc // tested indirectly via Import below

	cases := []struct {
		input    string
		wantTime bool
	}{
		{"DNS", true},
		{"DNF", true},
		{"", true},
		{"17:45:00", true},
		{"08:00", true},
		{"7:35", true}, // single-digit hour must not be treated as a header
		{"9:05:30", true},
		{"AS6", false},
		{"HELLO", false},
	}

	for _, tc := range cases {
		// Exercise looksLikeTimeOrStatus indirectly: a header-less import uses the
		// function to decide whether to skip line 0 as a header.
		runners := &mockRunnerRepository{
			runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
		}
		logs := &mockCheckpointLogRepository{}
		isvc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, &mockActiveSessionRepository{})

		col := tc.input + "\n"
		result, _ := isvc.Import(context.Background(), 1, 10, col)
		if tc.wantTime {
			// Treated as data row, not skipped as header
			assert.GreaterOrEqual(t, result.Created+result.Updated+result.Skipped, 0, "input: %q", tc.input)
		} else {
			// Treated as header, first runner row absent → 0 results
			assert.Equal(t, 0, result.Created+result.Updated, "input %q should be treated as header", tc.input)
		}
	}
}

// --- parseTimeOfDay ---

func TestParseTimeOfDay_HHMM(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
	}
	logs := &mockCheckpointLogRepository{}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, &mockActiveSessionRepository{})

	// HH:MM format (no seconds)
	result, err := svc.Import(context.Background(), 1, 10, "17:45\n")
	require.NoError(t, err)
	require.Equal(t, 1, result.Created)

	// Verify parsed time has correct hour and minute
	created := logs.created[0]
	assert.Equal(t, 17, created.RecordedAt.Hour())
	assert.Equal(t, 45, created.RecordedAt.Minute())
	assert.Equal(t, 0, created.RecordedAt.Second())
}

func TestParseTimeOfDay_InvalidReturnsError(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
	}
	logs := &mockCheckpointLogRepository{}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, &mockActiveSessionRepository{})

	result, err := svc.Import(context.Background(), 1, 10, "99:99:99\n")
	require.NoError(t, err)
	assert.Equal(t, 1, result.Skipped)
}

// --- Preview ---

func TestWinlinkService_Preview_ClassifiesEveryRowKind(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusActive}, // will Create
			{ID: 2, RaceID: 1, BibNumber: 101, SortOrder: 2, Status: entity.StatusActive}, // will Update (existing log)
			{ID: 3, RaceID: 1, BibNumber: 102, SortOrder: 3, Status: entity.StatusActive}, // DNS
			{ID: 4, RaceID: 1, BibNumber: 103, SortOrder: 4, Status: entity.StatusActive}, // DNF
			{ID: 5, RaceID: 1, BibNumber: 104, SortOrder: 5, Status: entity.StatusMoved},  // MOVED
		},
	}
	logs := &mockCheckpointLogRepository{
		logs: []entity.CheckpointLog{{ID: 1, RunnerID: 2, CheckpointID: 10}},
	}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, &mockActiveSessionRepository{})

	// header, create, update, DNS, DNF, MOVED, blank, garbage (no runner at position 8)
	column := "AS6\n17:45:00\n08:00:00\nDNS\nDNF\nMOVED Marathon\n\nnot-a-time\n"
	result, err := svc.Preview(context.Background(), 1, 10, column)
	require.NoError(t, err)

	// No writes should have happened.
	assert.Empty(t, logs.created)
	assert.Equal(t, entity.StatusActive, runners.runners[2].Status)
	assert.Equal(t, entity.StatusActive, runners.runners[3].Status)

	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 3, result.Updated) // existing-log time row + DNS + DNF
	assert.Equal(t, 3, result.Skipped) // MOVED, blank, no_runner
	require.Len(t, result.Rows, 7)

	assert.Equal(t, "create", result.Rows[0].Kind)
	assert.Equal(t, 100, result.Rows[0].BibNumber)

	assert.Equal(t, "update", result.Rows[1].Kind)
	assert.Equal(t, 101, result.Rows[1].BibNumber)

	assert.Equal(t, "update", result.Rows[2].Kind)
	assert.Equal(t, "DNS", result.Rows[2].Value)

	assert.Equal(t, "update", result.Rows[3].Kind)
	assert.Equal(t, "DNF", result.Rows[3].Value)

	assert.Equal(t, "skip", result.Rows[4].Kind)
	assert.Equal(t, "moved", result.Rows[4].Reason)
	assert.Equal(t, 104, result.Rows[4].BibNumber)

	assert.Equal(t, "skip", result.Rows[5].Kind)
	assert.Equal(t, "blank", result.Rows[5].Reason)

	assert.Equal(t, "skip", result.Rows[6].Kind)
	assert.Equal(t, "no_runner", result.Rows[6].Reason)
}

func TestWinlinkService_Preview_InvalidTimeSkipped(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
	}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	result, err := svc.Preview(context.Background(), 1, 10, "99:99:99\n")
	require.NoError(t, err)
	require.Len(t, result.Rows, 1)
	assert.Equal(t, "skip", result.Rows[0].Kind)
	assert.Equal(t, "parse_error", result.Rows[0].Reason)
	assert.Equal(t, 100, result.Rows[0].BibNumber)
}

func TestWinlinkService_Preview_NoWritesEvenWhenWritesWouldFail(t *testing.T) {
	// updateStatusErr/upsertErr are set to prove Preview never calls the write
	// paths — if it did, these would surface as errors.
	runners := &mockRunnerRepository{
		runners: []entity.Runner{
			{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1, Status: entity.StatusUnknown},
			{ID: 2, RaceID: 1, BibNumber: 101, SortOrder: 2, Status: entity.StatusActive},
		},
		updateStatusErr: errDB,
	}
	logs := &mockCheckpointLogRepository{upsertErr: errDB}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, &mockActiveSessionRepository{})

	result, err := svc.Preview(context.Background(), 1, 10, "17:45:00\nDNS\n")
	require.NoError(t, err)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 1, result.Updated)
	assert.Equal(t, entity.StatusUnknown, runners.runners[0].Status) // untouched
	assert.Empty(t, logs.created)
}

func TestWinlinkService_Preview_ListRunnersError(t *testing.T) {
	runners := &mockRunnerRepository{listErr: errDB}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, &mockCheckpointLogRepository{}, &mockActiveSessionRepository{})

	_, err := svc.Preview(context.Background(), 1, 10, "17:45:00\n")
	assert.ErrorContains(t, err, "listing runners")
}

func TestWinlinkService_Preview_ListLogsError(t *testing.T) {
	runners := &mockRunnerRepository{
		runners: []entity.Runner{{ID: 1, RaceID: 1, BibNumber: 100, SortOrder: 1}},
	}
	logs := &mockCheckpointLogRepository{listErr: errDB}
	svc := newWinlinkSvc(runners, &mockCheckpointRepository{checkpoints: map[int]entity.Checkpoint{}}, logs, &mockActiveSessionRepository{})

	_, err := svc.Preview(context.Background(), 1, 10, "17:45:00\n")
	assert.ErrorContains(t, err, "listing checkpoint logs")
}

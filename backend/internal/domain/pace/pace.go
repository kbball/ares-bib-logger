// Package pace computes a runner's current pace and projected arrival at a
// future checkpoint from their logged checkpoint history. It mirrors
// frontend/src/domain/pace.ts so the two stay behaviorally identical.
package pace

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/kevinball/ares-bib-logger/backend/internal/domain/entity"
)

// RunnerPace holds a computed pace and the checkpoint it was derived from.
// All fields are nil when pace cannot be computed.
type RunnerPace struct {
	PaceMinPerMile *float64
	LastLoggedDist *float64
	LastLoggedAt   *time.Time
}

type loggedCheckpoint struct {
	log entity.CheckpointLog
	cp  entity.Checkpoint
}

// ComputeRunnerPace computes pace from the last two logged checkpoints (in
// display order) that both have a known distance. DNS/DNF/MOVED/FINISHED
// runners always return an empty RunnerPace — once a runner has stopped,
// velocity isn't meaningful. Use LastLoggedCheckpoint if you need their last
// known location regardless of status.
func ComputeRunnerPace(runner entity.Runner, checkpoints []entity.Checkpoint, logs []entity.CheckpointLog) RunnerPace {
	switch runner.Status {
	case entity.StatusDNS, entity.StatusDNF, entity.StatusMoved, entity.StatusFinished:
		return RunnerPace{}
	}

	loggedWithCP := loggedCheckpointsForRunner(checkpoints, logs, runner.ID)

	var withDist []loggedCheckpoint
	for _, x := range loggedWithCP {
		if x.cp.DistanceFromStart != nil {
			withDist = append(withDist, x)
		}
	}

	if len(withDist) < 2 {
		if len(withDist) == 0 {
			return RunnerPace{}
		}
		last := withDist[0]
		dist := *last.cp.DistanceFromStart
		at := last.log.RecordedAt
		return RunnerPace{LastLoggedDist: &dist, LastLoggedAt: &at}
	}

	prev := withDist[len(withDist)-2]
	last := withDist[len(withDist)-1]

	distDelta := *last.cp.DistanceFromStart - *prev.cp.DistanceFromStart
	timeDelta := last.log.RecordedAt.Sub(prev.log.RecordedAt)

	dist := *last.cp.DistanceFromStart
	at := last.log.RecordedAt

	if distDelta <= 0 || timeDelta <= 0 {
		return RunnerPace{LastLoggedDist: &dist, LastLoggedAt: &at}
	}

	paceMinPerMile := timeDelta.Minutes() / distDelta
	return RunnerPace{PaceMinPerMile: &paceMinPerMile, LastLoggedDist: &dist, LastLoggedAt: &at}
}

// ProjectArrival projects arrival at a target checkpoint using a computed
// pace. Returns nil if pace or target distance is unavailable.
func ProjectArrival(p RunnerPace, targetDist float64) *time.Time {
	if p.PaceMinPerMile == nil || p.LastLoggedDist == nil || p.LastLoggedAt == nil {
		return nil
	}
	distToGo := targetDist - *p.LastLoggedDist
	if distToGo <= 0 {
		return nil
	}
	t := p.LastLoggedAt.Add(time.Duration(distToGo * *p.PaceMinPerMile * float64(time.Minute)))
	return &t
}

// FormatPace formats pace as "MM:SS /mi".
func FormatPace(paceMinPerMile float64) string {
	mins := int(paceMinPerMile)
	secs := int(math.Round((paceMinPerMile - float64(mins)) * 60))
	return fmt.Sprintf("%d:%02d /mi", mins, secs)
}

// LastLoggedCheckpoint returns the runner's most-recently-passed checkpoint
// (max DisplayOrder among their logs), independent of status. Unlike
// ComputeRunnerPace, this never comes back empty just because the runner's
// status is terminal — a dropped runner's last known station is exactly the
// information a search-and-rescue query needs.
func LastLoggedCheckpoint(checkpoints []entity.Checkpoint, logs []entity.CheckpointLog, runnerID int) (entity.Checkpoint, entity.CheckpointLog, bool) {
	loggedWithCP := loggedCheckpointsForRunner(checkpoints, logs, runnerID)
	if len(loggedWithCP) == 0 {
		return entity.Checkpoint{}, entity.CheckpointLog{}, false
	}
	best := loggedWithCP[len(loggedWithCP)-1]
	return best.cp, best.log, true
}

// loggedCheckpointsForRunner returns this runner's logged checkpoints,
// sorted ascending by display order.
func loggedCheckpointsForRunner(checkpoints []entity.Checkpoint, logs []entity.CheckpointLog, runnerID int) []loggedCheckpoint {
	cpByID := make(map[int]entity.Checkpoint, len(checkpoints))
	for _, cp := range checkpoints {
		cpByID[cp.ID] = cp
	}

	var out []loggedCheckpoint
	for _, l := range logs {
		if l.RunnerID != runnerID {
			continue
		}
		cp, ok := cpByID[l.CheckpointID]
		if !ok {
			continue
		}
		out = append(out, loggedCheckpoint{log: l, cp: cp})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].cp.DisplayOrder < out[j].cp.DisplayOrder
	})
	return out
}

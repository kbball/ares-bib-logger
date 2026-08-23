import type { Checkpoint, CheckpointLog, Runner } from './types'

export interface RunnerPace {
  paceMinPerMile: number | null
  lastLoggedDist: number | null
  lastLoggedAt: Date | null
}

interface LoggedCheckpoint {
  log: CheckpointLog
  cp: Checkpoint
}

// A DNS/DNF/MOVED/FINISHED runner has no meaningful pace or splits.
function hasNoMeaningfulPace(runner: Runner): boolean {
  return (
    runner.Status === 'DNS' ||
    runner.Status === 'DNF' ||
    runner.Status === 'MOVED' ||
    runner.Status === 'FINISHED'
  )
}

// This runner's logged checkpoints that have a known distance, sorted by display order.
function loggedWithDistance(
  runnerID: number,
  checkpoints: Checkpoint[],
  logs: CheckpointLog[],
): LoggedCheckpoint[] {
  const cpMap = new Map(checkpoints.map((cp) => [cp.ID, cp]))
  return logs
    .filter((l) => l.RunnerID === runnerID)
    .map((l) => ({ log: l, cp: cpMap.get(l.CheckpointID) }))
    .filter((x): x is LoggedCheckpoint => x.cp !== undefined && x.cp.DistanceFromStart !== null)
    .sort((a, b) => a.cp.DisplayOrder - b.cp.DisplayOrder)
}

// Pace between two logged-with-distance checkpoints, or null if the delta is zero/negative.
function pairPace(prev: LoggedCheckpoint, cur: LoggedCheckpoint): number | null {
  const distDelta = cur.cp.DistanceFromStart! - prev.cp.DistanceFromStart!
  const timeDeltaMs =
    new Date(cur.log.RecordedAt).getTime() - new Date(prev.log.RecordedAt).getTime()
  return distDelta > 0 && timeDeltaMs > 0 ? timeDeltaMs / 60000 / distDelta : null
}

// Computes pace from the last two logged checkpoints that both have a known distance.
// DNS/DNF/MOVED/FINISHED runners return nulls.
export function computeRunnerPace(
  runner: Runner,
  checkpoints: Checkpoint[],
  logs: CheckpointLog[],
): RunnerPace {
  if (hasNoMeaningfulPace(runner)) {
    return { paceMinPerMile: null, lastLoggedDist: null, lastLoggedAt: null }
  }

  const withDist = loggedWithDistance(runner.ID, checkpoints, logs)

  if (withDist.length < 2) {
    const last = withDist[0] ?? null
    return {
      paceMinPerMile: null,
      lastLoggedDist: last?.cp.DistanceFromStart ?? null,
      lastLoggedAt: last ? new Date(last.log.RecordedAt) : null,
    }
  }

  const prev = withDist[withDist.length - 2]
  const last = withDist[withDist.length - 1]

  return {
    paceMinPerMile: pairPace(prev, last),
    lastLoggedDist: last.cp.DistanceFromStart,
    lastLoggedAt: new Date(last.log.RecordedAt),
  }
}

// Pace between each pair of consecutive logged checkpoints (both with a known
// distance) for a runner — a full splits view, unlike computeRunnerPace, which
// only reports the most recent split. Keyed by checkpoint ID; the first
// logged-with-distance checkpoint has no entry (no prior split to compare
// against), and checkpoints that are unlogged or lack a distance never appear
// as keys — render as blank/— in the UI.
export function computeSplitPaces(
  runner: Runner,
  checkpoints: Checkpoint[],
  logs: CheckpointLog[],
): Map<number, number | null> {
  const splits = new Map<number, number | null>()
  if (hasNoMeaningfulPace(runner)) return splits

  const withDist = loggedWithDistance(runner.ID, checkpoints, logs)
  for (let i = 1; i < withDist.length; i++) {
    splits.set(withDist[i].cp.ID, pairPace(withDist[i - 1], withDist[i]))
  }
  return splits
}

// Projects arrival at a target checkpoint using a computed pace.
// Returns null if pace or target distance is unavailable.
export function projectArrival(pace: RunnerPace, targetDist: number): Date | null {
  if (pace.paceMinPerMile === null || pace.lastLoggedDist === null || pace.lastLoggedAt === null)
    return null

  const distToGo = targetDist - pace.lastLoggedDist
  if (distToGo <= 0) return null

  return new Date(pace.lastLoggedAt.getTime() + distToGo * pace.paceMinPerMile * 60000)
}

// Formats pace as "MM:SS /mi" (e.g. "12:30 /mi").
export function formatPace(paceMinPerMile: number): string {
  const mins = Math.floor(paceMinPerMile)
  const secs = Math.round((paceMinPerMile - mins) * 60)
  return `${mins}:${String(secs).padStart(2, '0')} /mi`
}

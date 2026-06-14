import type { Checkpoint, CheckpointLog, Runner } from './types'

export interface RunnerPace {
  paceMinPerMile: number | null
  lastLoggedDist: number | null
  lastLoggedAt: Date | null
}

// Computes pace from the last two logged checkpoints that both have a known distance.
// DNS/DNF/MOVED/FINISHED runners return nulls.
export function computeRunnerPace(
  runner: Runner,
  checkpoints: Checkpoint[],
  logs: CheckpointLog[],
): RunnerPace {
  if (runner.Status === 'DNS' || runner.Status === 'DNF' || runner.Status === 'MOVED' || runner.Status === 'FINISHED') {
    return { paceMinPerMile: null, lastLoggedDist: null, lastLoggedAt: null }
  }

  const cpMap = new Map(checkpoints.map((cp) => [cp.ID, cp]))

  // Logged checkpoints for this runner, sorted by display order
  const loggedWithCP = logs
    .filter((l) => l.RunnerID === runner.ID)
    .map((l) => ({ log: l, cp: cpMap.get(l.CheckpointID) }))
    .filter((x): x is { log: CheckpointLog; cp: Checkpoint } => x.cp !== undefined)
    .sort((a, b) => a.cp.DisplayOrder - b.cp.DisplayOrder)

  // Only checkpoints with a known distance
  const withDist = loggedWithCP.filter((x) => x.cp.DistanceFromStart !== null)

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

  const distDelta = last.cp.DistanceFromStart! - prev.cp.DistanceFromStart!
  const timeDeltaMs = new Date(last.log.RecordedAt).getTime() - new Date(prev.log.RecordedAt).getTime()

  if (distDelta <= 0 || timeDeltaMs <= 0) {
    return {
      paceMinPerMile: null,
      lastLoggedDist: last.cp.DistanceFromStart,
      lastLoggedAt: new Date(last.log.RecordedAt),
    }
  }

  return {
    paceMinPerMile: timeDeltaMs / 60000 / distDelta,
    lastLoggedDist: last.cp.DistanceFromStart,
    lastLoggedAt: new Date(last.log.RecordedAt),
  }
}

// Projects arrival at a target checkpoint using a computed pace.
// Returns null if pace or target distance is unavailable.
export function projectArrival(pace: RunnerPace, targetDist: number): Date | null {
  if (
    pace.paceMinPerMile === null ||
    pace.lastLoggedDist === null ||
    pace.lastLoggedAt === null
  ) return null

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

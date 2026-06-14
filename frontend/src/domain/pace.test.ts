import { describe, it, expect } from 'vitest'
import { computeRunnerPace, projectArrival, formatPace } from './pace'
import type { Checkpoint, CheckpointLog, Runner } from './types'

const makeRunner = (overrides: Partial<Runner> = {}): Runner => ({
  ID: 1,
  RaceID: 1,
  BibNumber: 100,
  FirstName: 'Alice',
  LastName: 'Smith',
  SortOrder: 1,
  Status: 'ACTIVE',
  CreatedAt: '',
  UpdatedAt: '',
  ...overrides,
})

const makeCheckpoint = (id: number, order: number, dist: number | null): Checkpoint => ({
  ID: id,
  RaceID: 1,
  Code: `AS${id}`,
  DisplayName: `Aid Station ${id}`,
  DisplayOrder: order,
  DistanceFromStart: dist,
  CreatedAt: '',
})

const makeLog = (runnerID: number, checkpointID: number, recordedAt: string): CheckpointLog => ({
  ID: checkpointID,
  RunnerID: runnerID,
  CheckpointID: checkpointID,
  RecordedAt: recordedAt,
  Source: 'MANUAL',
  RawMessage: '',
  CreatedAt: recordedAt,
})

describe('computeRunnerPace', () => {
  it('returns nulls for DNS runner', () => {
    const result = computeRunnerPace(makeRunner({ Status: 'DNS' }), [], [])
    expect(result).toEqual({ paceMinPerMile: null, lastLoggedDist: null, lastLoggedAt: null })
  })

  it('returns nulls for DNF runner', () => {
    const result = computeRunnerPace(makeRunner({ Status: 'DNF' }), [], [])
    expect(result).toEqual({ paceMinPerMile: null, lastLoggedDist: null, lastLoggedAt: null })
  })

  it('returns nulls for MOVED runner', () => {
    const result = computeRunnerPace(makeRunner({ Status: 'MOVED' }), [], [])
    expect(result).toEqual({ paceMinPerMile: null, lastLoggedDist: null, lastLoggedAt: null })
  })

  it('returns nulls for FINISHED runner', () => {
    const result = computeRunnerPace(makeRunner({ Status: 'FINISHED' }), [], [])
    expect(result).toEqual({ paceMinPerMile: null, lastLoggedDist: null, lastLoggedAt: null })
  })

  it('returns nulls when runner has no logs', () => {
    const cps = [makeCheckpoint(1, 1, 10.0)]
    const result = computeRunnerPace(makeRunner(), cps, [])
    expect(result).toEqual({ paceMinPerMile: null, lastLoggedDist: null, lastLoggedAt: null })
  })

  it('returns null pace but last position when only one distance checkpoint logged', () => {
    const cps = [makeCheckpoint(1, 1, 10.0), makeCheckpoint(2, 2, 20.0)]
    const logs = [makeLog(1, 1, '2026-06-14T10:00:00Z')]
    const result = computeRunnerPace(makeRunner(), cps, logs)
    expect(result.paceMinPerMile).toBeNull()
    expect(result.lastLoggedDist).toBe(10.0)
    expect(result.lastLoggedAt).toEqual(new Date('2026-06-14T10:00:00Z'))
  })

  it('computes pace from two distance-tagged checkpoints', () => {
    const cps = [makeCheckpoint(1, 1, 10.0), makeCheckpoint(2, 2, 20.0)]
    // 60 minutes between checkpoints, 10 miles apart → 6 min/mi
    const logs = [makeLog(1, 1, '2026-06-14T10:00:00Z'), makeLog(1, 2, '2026-06-14T11:00:00Z')]
    const result = computeRunnerPace(makeRunner(), cps, logs)
    expect(result.paceMinPerMile).toBeCloseTo(6.0)
    expect(result.lastLoggedDist).toBe(20.0)
    expect(result.lastLoggedAt).toEqual(new Date('2026-06-14T11:00:00Z'))
  })

  it('uses the last two distance checkpoints when more than two are logged', () => {
    const cps = [makeCheckpoint(1, 1, 5.0), makeCheckpoint(2, 2, 15.0), makeCheckpoint(3, 3, 25.0)]
    // Between cp2 and cp3: 10 miles in 90 min → 9 min/mi
    const logs = [
      makeLog(1, 1, '2026-06-14T09:00:00Z'),
      makeLog(1, 2, '2026-06-14T10:00:00Z'),
      makeLog(1, 3, '2026-06-14T11:30:00Z'),
    ]
    const result = computeRunnerPace(makeRunner(), cps, logs)
    expect(result.paceMinPerMile).toBeCloseTo(9.0)
  })

  it('skips checkpoints with null distance', () => {
    const cps = [
      makeCheckpoint(1, 1, 10.0),
      makeCheckpoint(2, 2, null), // no distance
      makeCheckpoint(3, 3, 20.0),
    ]
    const logs = [
      makeLog(1, 1, '2026-06-14T10:00:00Z'),
      makeLog(1, 2, '2026-06-14T10:30:00Z'),
      makeLog(1, 3, '2026-06-14T11:00:00Z'),
    ]
    // Between cp1 and cp3: 10 miles in 60 min → 6 min/mi
    const result = computeRunnerPace(makeRunner(), cps, logs)
    expect(result.paceMinPerMile).toBeCloseTo(6.0)
  })

  it('returns null pace when distance delta is zero', () => {
    const cps = [makeCheckpoint(1, 1, 10.0), makeCheckpoint(2, 2, 10.0)]
    const logs = [makeLog(1, 1, '2026-06-14T10:00:00Z'), makeLog(1, 2, '2026-06-14T11:00:00Z')]
    const result = computeRunnerPace(makeRunner(), cps, logs)
    expect(result.paceMinPerMile).toBeNull()
  })

  it('ignores logs for other runners', () => {
    const cps = [makeCheckpoint(1, 1, 10.0), makeCheckpoint(2, 2, 20.0)]
    const logs = [
      makeLog(99, 1, '2026-06-14T10:00:00Z'), // different runner
      makeLog(99, 2, '2026-06-14T11:00:00Z'),
    ]
    const result = computeRunnerPace(makeRunner({ ID: 1 }), cps, logs)
    expect(result.paceMinPerMile).toBeNull()
  })
})

describe('projectArrival', () => {
  const basePace = {
    paceMinPerMile: 6.0,
    lastLoggedDist: 10.0,
    lastLoggedAt: new Date('2026-06-14T10:00:00Z'),
  }

  it('returns null when pace is null', () => {
    expect(projectArrival({ ...basePace, paceMinPerMile: null }, 20)).toBeNull()
  })

  it('returns null when lastLoggedDist is null', () => {
    expect(projectArrival({ ...basePace, lastLoggedDist: null }, 20)).toBeNull()
  })

  it('returns null when lastLoggedAt is null', () => {
    expect(projectArrival({ ...basePace, lastLoggedAt: null }, 20)).toBeNull()
  })

  it('returns null when target is not ahead of last logged position', () => {
    expect(projectArrival(basePace, 10)).toBeNull()
    expect(projectArrival(basePace, 5)).toBeNull()
  })

  it('projects arrival correctly: 10mi at 6min/mi = +60min', () => {
    const result = projectArrival(basePace, 20)
    expect(result).toEqual(new Date('2026-06-14T11:00:00Z'))
  })

  it('projects fractional distances correctly', () => {
    const result = projectArrival(basePace, 15) // 5 miles at 6 min/mi = 30 min
    expect(result).toEqual(new Date('2026-06-14T10:30:00Z'))
  })
})

describe('formatPace', () => {
  it('formats whole minutes', () => {
    expect(formatPace(8)).toBe('8:00 /mi')
  })

  it('formats minutes with seconds', () => {
    expect(formatPace(8.5)).toBe('8:30 /mi')
  })

  it('pads seconds with leading zero', () => {
    expect(formatPace(6.1)).toBe('6:06 /mi')
  })

  it('formats sub-minute pace (edge case)', () => {
    expect(formatPace(0.5)).toBe('0:30 /mi')
  })
})

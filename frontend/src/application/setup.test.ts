import { describe, it, expect } from 'vitest'
import { setupProgress, setupSteps } from './setup'
import type { Checkpoint, Event, Race } from '../domain/types'

const event = { ID: 1, Name: 'Ridgeline' } as Event
const race = (over: Partial<Race> = {}): Race =>
  ({ ID: 1, EventID: 1, Name: '50K', RosterCount: 0, OrderLocked: false, ...over }) as Race
const cp = (over: Partial<Checkpoint> = {}): Checkpoint =>
  ({ ID: 1, RaceID: 1, Code: 'A1', DistanceFromStart: 5, ...over }) as Checkpoint

const state = (steps: ReturnType<typeof setupSteps>) =>
  Object.fromEntries(steps.map((s) => [s.id, s.state]))

describe('setupSteps', () => {
  it('waits on everything after the event when none is selected', () => {
    const s = state(
      setupSteps({ event: undefined, races: [], checkpointsByRace: {}, session: null }),
    )
    expect(s).toEqual({
      event: 'todo',
      races: 'waiting',
      checkpoints: 'waiting',
      roster: 'waiting',
      lock: 'waiting',
      active: 'waiting',
      share: 'optional',
    })
  })

  it('asks for races once there is an event', () => {
    const s = state(setupSteps({ event, races: [], checkpointsByRace: {}, session: null }))
    expect(s.event).toBe('done')
    expect(s.races).toBe('todo')
  })

  it('lists the races that still need checkpoints and a roster', () => {
    const steps = setupSteps({
      event,
      races: [race(), race({ ID: 2, Name: '25K', RosterCount: 10 })],
      checkpointsByRace: { 1: [], 2: [cp({ RaceID: 2 })] },
      session: null,
    })
    const by = Object.fromEntries(steps.map((s) => [s.id, s]))
    expect(by.checkpoints.state).toBe('todo')
    expect(by.checkpoints.detail).toContain('50K')
    expect(by.checkpoints.detail).not.toContain('25K')
    expect(by.roster.state).toBe('todo')
    expect(by.roster.detail).toContain('50K')
    expect(by.active.state).toBe('todo')
  })

  it('marks a fully set up event done and notes missing distances', () => {
    const steps = setupSteps({
      event,
      races: [race({ RosterCount: 40, OrderLocked: true })],
      checkpointsByRace: { 1: [cp(), cp({ ID: 2, DistanceFromStart: null })] },
      session: { EventID: 1, Checkpoints: [{ RaceID: 1, CheckpointID: 1 }] },
    })
    const by = Object.fromEntries(steps.map((s) => [s.id, s]))
    expect(by.checkpoints.state).toBe('done')
    expect(by.checkpoints.detail).toContain('distances')
    expect(by.roster.detail).toContain('40 runners')
    expect(by.lock.state).toBe('done')
    expect(by.active.state).toBe('done')
    expect(setupProgress(steps)).toEqual({ done: 5, total: 5 })
  })

  it('counts optional steps out of the progress total', () => {
    const steps = setupSteps({ event, races: [race()], checkpointsByRace: {}, session: null })
    expect(setupProgress(steps).total).toBe(steps.filter((s) => s.required).length)
  })

  it('says all checkpoints have distances when they do', () => {
    const steps = setupSteps({
      event,
      races: [race()],
      checkpointsByRace: { 1: [cp()] },
      session: null,
    })
    expect(steps.find((s) => s.id === 'checkpoints')!.detail).toContain('all with distances')
  })
})

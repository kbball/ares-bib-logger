import type { ActiveSession, Checkpoint, Event, Race } from '../domain/types'

export type StepState = 'done' | 'todo' | 'waiting' | 'optional'

export interface SetupStep {
  id: string
  title: string
  state: StepState
  /** Whether this step counts toward "n of m done". Optional extras do not. */
  required: boolean
  detail: string
  /** The id of the element on the Admin page to jump to. */
  target: string
}

export interface SetupInput {
  /** The active event, if one is selected. */
  event: Event | undefined
  races: Race[]
  checkpointsByRace: Record<number, Checkpoint[]>
  session: ActiveSession | null
}

const names = (races: Race[]) => races.map((r) => r.Name).join(', ')

/**
 * The order an event is best set up in, and where each step stands. Later steps
 * depend on earlier ones: races need an event, checkpoints and rosters need
 * races, and the active checkpoint needs the checkpoints to exist.
 */
export function setupSteps({ event, races, checkpointsByRace, session }: SetupInput): SetupStep[] {
  const cps = (r: Race) => checkpointsByRace[r.ID] ?? []
  const noCheckpoints = races.filter((r) => cps(r).length === 0)
  const noRoster = races.filter((r) => r.RosterCount === 0)
  const activeFor = (r: Race) => session?.Checkpoints?.some((c) => c.RaceID === r.ID) ?? false
  const noActive = races.filter((r) => !activeFor(r))
  const hasRaces = races.length > 0
  const anyCheckpoints = races.some((r) => cps(r).length > 0)

  const steps: SetupStep[] = []

  steps.push(
    event
      ? {
          id: 'event',
          title: 'Event',
          state: 'done',
          required: true,
          target: 'setup-event',
          detail: `${event.Name} is selected.`,
        }
      : {
          id: 'event',
          title: 'Event',
          state: 'todo',
          required: true,
          target: 'setup-event',
          detail: 'Create an event, or select an existing one to work on.',
        },
  )

  if (!event) {
    steps.push({
      id: 'races',
      title: 'Races',
      state: 'waiting',
      required: true,
      target: 'setup-races',
      detail: 'Needs an active event first.',
    })
  } else if (!hasRaces) {
    steps.push({
      id: 'races',
      title: 'Races',
      state: 'todo',
      required: true,
      target: 'setup-races',
      detail: 'Add each race in the event, for example 50K and 25K.',
    })
  } else {
    steps.push({
      id: 'races',
      title: 'Races',
      state: 'done',
      required: true,
      target: 'setup-races',
      detail: `${races.length} race${races.length === 1 ? '' : 's'}: ${names(races)}.`,
    })
  }

  if (!hasRaces) {
    steps.push({
      id: 'checkpoints',
      title: 'Checkpoints',
      state: 'waiting',
      required: true,
      target: 'setup-checkpoints',
      detail: 'Needs a race first.',
    })
  } else if (noCheckpoints.length > 0) {
    steps.push({
      id: 'checkpoints',
      title: 'Checkpoints',
      state: 'todo',
      required: true,
      target: 'setup-checkpoints',
      detail: `No checkpoints yet for ${names(noCheckpoints)}. Add them in order, or paste a list with Bulk Checkpoint Import.`,
    })
  } else {
    const all = races.flatMap(cps)
    const withDistance = all.filter((c) => c.DistanceFromStart !== null).length
    steps.push({
      id: 'checkpoints',
      title: 'Checkpoints',
      state: 'done',
      required: true,
      target: 'setup-checkpoints',
      detail:
        withDistance === all.length
          ? `${all.length} checkpoints, all with distances.`
          : `${all.length} checkpoints. Add distances (miles from start) to get arrival projections.`,
    })
  }

  if (!hasRaces) {
    steps.push({
      id: 'roster',
      title: 'Roster',
      state: 'waiting',
      required: true,
      target: 'setup-roster',
      detail: 'Needs a race first.',
    })
  } else if (noRoster.length > 0) {
    steps.push({
      id: 'roster',
      title: 'Roster',
      state: 'todo',
      required: true,
      target: 'setup-roster',
      detail: `No runners yet for ${names(noRoster)}. Paste the roster as bib, first name, last name.`,
    })
  } else {
    const total = races.reduce((n, r) => n + r.RosterCount, 0)
    steps.push({
      id: 'roster',
      title: 'Roster',
      state: 'done',
      required: true,
      target: 'setup-roster',
      detail: `${total} runners across ${races.length} race${races.length === 1 ? '' : 's'}.`,
    })
  }

  if (!hasRaces) {
    steps.push({
      id: 'lock',
      title: 'Lock the order',
      state: 'waiting',
      required: false,
      target: 'setup-races',
      detail: 'Needs a race first.',
    })
  } else if (races.every((r) => r.OrderLocked)) {
    steps.push({
      id: 'lock',
      title: 'Lock the order',
      state: 'done',
      required: false,
      target: 'setup-races',
      detail: 'Checkpoint order is locked for every race.',
    })
  } else {
    steps.push({
      id: 'lock',
      title: 'Lock the order',
      state: 'optional',
      required: false,
      target: 'setup-races',
      detail:
        'Once the order is right, lock it so it cannot change mid-race. Locking is permanent for that race.',
    })
  }

  if (!anyCheckpoints) {
    steps.push({
      id: 'active',
      title: 'Active checkpoint',
      state: 'waiting',
      required: true,
      target: 'setup-active',
      detail: 'Needs checkpoints first.',
    })
  } else if (noActive.length > 0) {
    steps.push({
      id: 'active',
      title: 'Active checkpoint',
      state: 'todo',
      required: true,
      target: 'setup-active',
      detail: `Pick the checkpoint your station covers for ${names(noActive)}. Bibs are logged there.`,
    })
  } else {
    steps.push({
      id: 'active',
      title: 'Active checkpoint',
      state: 'done',
      required: true,
      target: 'setup-active',
      detail: 'Every race has an active checkpoint, so Data Entry is ready.',
    })
  }

  steps.push({
    id: 'share',
    title: 'Share',
    state: 'optional',
    required: false,
    target: 'setup-event',
    detail: 'Export the event config to set up another station with the same races and roster.',
  })

  return steps
}

/** How many of the steps that matter are done. */
export function setupProgress(steps: SetupStep[]): { done: number; total: number } {
  const needed = steps.filter((s) => s.required)
  return {
    done: needed.filter((s) => s.state === 'done').length,
    total: needed.length,
  }
}

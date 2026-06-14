import { describe, it, expect } from 'vitest'
import { http, HttpResponse } from 'msw'
import { server } from '../../test/server'
import * as api from './index'

describe('Events', () => {
  it('listEvents returns array', async () => {
    const result = await api.listEvents()
    expect(Array.isArray(result)).toBe(true)
    expect(result[0].ID).toBe(1)
  })

  it('createEvent posts name and returns event', async () => {
    const result = await api.createEvent('New Event')
    expect(result.Name).toBe('GDR 2026')
  })

  it('archiveEvent sends PUT and returns void', async () => {
    const result = await api.archiveEvent(1)
    expect(result).toBeUndefined()
  })
})

describe('Races', () => {
  it('listRaces returns array for event', async () => {
    const result = await api.listRaces(1)
    expect(result[0].EventID).toBe(1)
  })

  it('createRace returns race', async () => {
    const result = await api.createRace(1, 'GDR')
    expect(result.Name).toBe('GDR')
  })

  it('deleteRace sends DELETE', async () => {
    const result = await api.deleteRace(1)
    expect(result).toBeUndefined()
  })

  it('lockRaceOrder sends PUT', async () => {
    const result = await api.lockRaceOrder(1)
    expect(result).toBeUndefined()
  })
})

describe('Checkpoints', () => {
  it('listCheckpoints returns array', async () => {
    const result = await api.listCheckpoints(1)
    expect(result.length).toBe(2)
    expect(result[0].RaceID).toBe(1)
  })

  it('createCheckpoint returns checkpoint', async () => {
    const result = await api.createCheckpoint(1, 'AS1', 'Aid Station 1', 10.5)
    expect(result.Code).toBe('AS1')
  })

  it('createCheckpoint without distance uses null', async () => {
    const result = await api.createCheckpoint(1, 'AS1', 'Aid Station 1')
    expect(result.Code).toBe('AS1')
  })

  it('updateCheckpoint returns updated checkpoint', async () => {
    const result = await api.updateCheckpoint(1, 'AS1', 'Aid Station 1', null)
    expect(result.Code).toBe('AS1')
  })

  it('updateCheckpoint with distance uses provided value', async () => {
    const result = await api.updateCheckpoint(1, 'AS1', 'Aid Station 1', 10.5)
    expect(result.Code).toBe('AS1')
  })

  it('deleteCheckpoint sends DELETE', async () => {
    const result = await api.deleteCheckpoint(1)
    expect(result).toBeUndefined()
  })

  it('reorderCheckpoints sends PUT', async () => {
    const result = await api.reorderCheckpoints(1, [2, 1])
    expect(result).toBeUndefined()
  })
})

describe('Runners / Roster', () => {
  it('listRunners returns array', async () => {
    const result = await api.listRunners(1)
    expect(result.length).toBe(2)
    expect(result[0].BibNumber).toBe(100)
  })

  it('importRoster returns imported count', async () => {
    const result = await api.importRoster(1, '100\tAlice\tSmith')
    expect(result.imported).toBe(2)
  })

  it('transferRunner sends POST', async () => {
    const result = await api.transferRunner(100, 1, 2)
    expect(result).toBeUndefined()
  })
})

describe('Checkpoint logs', () => {
  it('listCheckpointLogs returns array', async () => {
    const result = await api.listCheckpointLogs(1)
    expect(result[0].RunnerID).toBe(1)
  })
})

describe('Bib logging', () => {
  it('logBib returns result with runner and log', async () => {
    const result = await api.logBib(100)
    expect(result.runner.BibNumber).toBe(100)
    expect(result.is_duplicate).toBe(false)
  })

  it('logStatus sends POST and returns void', async () => {
    const result = await api.logStatus(100, 'DNS')
    expect(result).toBeUndefined()
  })
})

describe('Session', () => {
  it('getSession returns session', async () => {
    const result = await api.getSession()
    expect(result.EventID).toBe(1)
    expect(result.Checkpoints).toHaveLength(1)
  })

  it('setSessionEvent sends PUT', async () => {
    const result = await api.setSessionEvent(1)
    expect(result).toBeUndefined()
  })

  it('setSessionCheckpoint sends PUT', async () => {
    const result = await api.setSessionCheckpoint(1, 1)
    expect(result).toBeUndefined()
  })

  it('clearSessionCheckpoint sends DELETE', async () => {
    const result = await api.clearSessionCheckpoint(1)
    expect(result).toBeUndefined()
  })
})

describe('Winlink', () => {
  it('exportWinlink returns text', async () => {
    const result = await api.exportWinlink(1)
    expect(typeof result).toBe('string')
    expect(result).toContain('AS1')
  })

  it('importWinlink returns import result', async () => {
    const result = await api.importWinlink(1, 1, 'AS1\n10:00\n')
    expect(result.Created).toBe(1)
    expect(result.Skipped).toBe(0)
  })

  it('importWinlink propagates API errors', async () => {
    server.use(
      http.post('/api/winlink/import', () =>
        HttpResponse.json({ error: 'no active checkpoint' }, { status: 400 }),
      ),
    )
    await expect(api.importWinlink(1, 1, 'bad')).rejects.toMatchObject({
      message: 'no active checkpoint',
    })
  })
})

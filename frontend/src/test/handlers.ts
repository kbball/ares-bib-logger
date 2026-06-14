import { http, HttpResponse } from 'msw'
import type {
  ActiveSession,
  Checkpoint,
  CheckpointLog,
  Event,
  Race,
  Runner,
} from '../domain/types'

// ── Shared mock data ─────────────────────────────────────────────────────────

export const mockEvent: Event = {
  ID: 1,
  Name: 'GDR 2026',
  Archived: false,
  CreatedAt: '2026-06-14T00:00:00Z',
}

export const mockRace: Race = {
  ID: 1,
  EventID: 1,
  Name: 'GDR',
  RosterLocked: false,
  OrderLocked: false,
  CreatedAt: '2026-06-14T00:00:00Z',
}

export const mockCheckpoint: Checkpoint = {
  ID: 1,
  RaceID: 1,
  Code: 'AS1',
  DisplayName: 'Aid Station 1',
  DisplayOrder: 1,
  DistanceFromStart: 10.5,
  CreatedAt: '2026-06-14T00:00:00Z',
}

export const mockCheckpoint2: Checkpoint = {
  ID: 2,
  RaceID: 1,
  Code: 'AS2',
  DisplayName: 'Aid Station 2',
  DisplayOrder: 2,
  DistanceFromStart: 21.0,
  CreatedAt: '2026-06-14T00:00:00Z',
}

export const mockRunner: Runner = {
  ID: 1,
  RaceID: 1,
  BibNumber: 100,
  FirstName: 'Alice',
  LastName: 'Smith',
  SortOrder: 1,
  Status: 'ACTIVE',
  CreatedAt: '2026-06-14T00:00:00Z',
  UpdatedAt: '2026-06-14T00:00:00Z',
}

export const mockRunner2: Runner = {
  ID: 2,
  RaceID: 1,
  BibNumber: 101,
  FirstName: 'Bob',
  LastName: 'Jones',
  SortOrder: 2,
  Status: 'UNKNOWN',
  CreatedAt: '2026-06-14T00:00:00Z',
  UpdatedAt: '2026-06-14T00:00:00Z',
}

export const mockLog: CheckpointLog = {
  ID: 1,
  RunnerID: 1,
  CheckpointID: 1,
  RecordedAt: '2026-06-14T10:00:00Z',
  Source: 'MANUAL',
  RawMessage: '10:00',
  CreatedAt: '2026-06-14T10:00:00Z',
}

export const mockSession: ActiveSession = {
  EventID: 1,
  Checkpoints: [{ RaceID: 1, CheckpointID: 1 }],
}

export const noSession: ActiveSession = {
  EventID: null,
  Checkpoints: [],
}

// ── Default handlers ──────────────────────────────────────────────────────────

export const handlers = [
  // Events
  http.get('/api/events', () => HttpResponse.json([mockEvent])),
  http.post('/api/events', () => HttpResponse.json(mockEvent, { status: 201 })),
  http.put('/api/events/:id/archive', () => new HttpResponse(null, { status: 204 })),
  http.get('/api/events/:id', () => HttpResponse.json(mockEvent)),

  // Races
  http.get('/api/events/:eventID/races', () => HttpResponse.json([mockRace])),
  http.post('/api/events/:eventID/races', () => HttpResponse.json(mockRace, { status: 201 })),
  http.delete('/api/races/:id', () => new HttpResponse(null, { status: 204 })),
  http.put('/api/races/:id/lock-order', () => new HttpResponse(null, { status: 204 })),

  // Checkpoints
  http.get('/api/races/:raceID/checkpoints', () =>
    HttpResponse.json([mockCheckpoint, mockCheckpoint2]),
  ),
  http.post('/api/races/:raceID/checkpoints', () =>
    HttpResponse.json(mockCheckpoint, { status: 201 }),
  ),
  http.put('/api/checkpoints/:id', () => HttpResponse.json(mockCheckpoint)),
  http.delete('/api/checkpoints/:id', () => new HttpResponse(null, { status: 204 })),
  http.put('/api/races/:raceID/checkpoints/order', () => new HttpResponse(null, { status: 204 })),

  // Runners / logs
  http.get('/api/races/:raceID/runners', () => HttpResponse.json([mockRunner, mockRunner2])),
  http.post('/api/races/:raceID/roster', () => HttpResponse.json({ imported: 2 })),
  http.post('/api/runners/transfer', () => new HttpResponse(null, { status: 204 })),
  http.get('/api/races/:raceID/logs', () => HttpResponse.json([mockLog])),

  // Bib logging
  http.post('/api/log/bib', () =>
    HttpResponse.json({ runner: mockRunner, log: mockLog, is_duplicate: false }),
  ),
  http.post('/api/log/status', () => new HttpResponse(null, { status: 204 })),

  // Session
  http.get('/api/session', () => HttpResponse.json(mockSession)),
  http.put('/api/session/event', () => new HttpResponse(null, { status: 204 })),
  http.put('/api/session/checkpoint', () => new HttpResponse(null, { status: 204 })),
  http.delete('/api/session/checkpoint/:raceID', () => new HttpResponse(null, { status: 204 })),

  // Winlink
  http.get('/api/winlink/export/:raceID', () => new HttpResponse('AS1\n10:00\n', { status: 200 })),
  http.post('/api/winlink/import', () =>
    HttpResponse.json({ Created: 1, Updated: 0, Skipped: 0, SkippedDetails: [], Errors: [] }),
  ),
]

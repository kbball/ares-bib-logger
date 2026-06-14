import { get, post, put, del } from './client'
import type {
  Event,
  Race,
  Checkpoint,
  CheckpointLog,
  Runner,
  ActiveSession,
  LogBibResult,
  RunnerStatus,
  WinlinkImportResult,
} from '../../domain/types'

// Events
export const listEvents = () => get<Event[]>('/api/events')
export const createEvent = (name: string) => post<Event>('/api/events', { name })
export const archiveEvent = (id: number) => put<void>(`/api/events/${id}/archive`)

// Races
export const listRaces = (eventID: number) => get<Race[]>(`/api/events/${eventID}/races`)
export const createRace = (eventID: number, name: string) =>
  post<Race>(`/api/events/${eventID}/races`, { name })
export const deleteRace = (id: number) => del<void>(`/api/races/${id}`)
export const lockRaceOrder = (id: number) => put<void>(`/api/races/${id}/lock-order`)

// Checkpoints
export const listCheckpoints = (raceID: number) =>
  get<Checkpoint[]>(`/api/races/${raceID}/checkpoints`)
export const createCheckpoint = (raceID: number, code: string, displayName: string) =>
  post<Checkpoint>(`/api/races/${raceID}/checkpoints`, { code, display_name: displayName })
export const updateCheckpoint = (id: number, code: string, displayName: string) =>
  put<Checkpoint>(`/api/checkpoints/${id}`, { code, display_name: displayName })
export const deleteCheckpoint = (id: number) => del<void>(`/api/checkpoints/${id}`)
export const reorderCheckpoints = (raceID: number, ids: number[]) =>
  put<void>(`/api/races/${raceID}/checkpoints/order`, { ids })

// Checkpoint logs
export const listCheckpointLogs = (raceID: number) => get<CheckpointLog[]>(`/api/races/${raceID}/logs`)

// Runners / Roster
export const listRunners = (raceID: number) => get<Runner[]>(`/api/races/${raceID}/runners`)
export const importRoster = (raceID: number, tsv: string) =>
  post<{ imported: number }>(`/api/races/${raceID}/roster`, { tsv })
export const transferRunner = (bibNumber: number, fromRaceID: number, toRaceID: number) =>
  post<void>('/api/runners/transfer', { bib_number: bibNumber, from_race_id: fromRaceID, to_race_id: toRaceID })

// Bib logging
export const logBib = (bibNumber: number) =>
  post<LogBibResult>('/api/log/bib', { bib_number: bibNumber })
export const logStatus = (bibNumber: number, status: RunnerStatus) =>
  post<void>('/api/log/status', { bib_number: bibNumber, status })

// Session
export const getSession = () => get<ActiveSession>('/api/session')
export const setSessionEvent = (eventID: number) =>
  put<void>('/api/session/event', { event_id: eventID })
export const setSessionCheckpoint = (raceID: number, checkpointID: number) =>
  put<void>('/api/session/checkpoint', { race_id: raceID, checkpoint_id: checkpointID })
export const clearSessionCheckpoint = (raceID: number) =>
  del<void>(`/api/session/checkpoint/${raceID}`)

// Winlink
export const exportWinlink = (raceID: number) =>
  fetch(`/api/winlink/export/${raceID}`).then((r) => r.text())
export const importWinlink = (raceID: number, checkpointID: number, text: string) =>
  post<WinlinkImportResult>('/api/winlink/import', { race_id: raceID, checkpoint_id: checkpointID, text })

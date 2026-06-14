export interface Event {
  ID: number
  Name: string
  Archived: boolean
  CreatedAt: string
}

export interface Race {
  ID: number
  EventID: number
  Name: string
  RosterLocked: boolean
  OrderLocked: boolean
  CreatedAt: string
}

export interface Checkpoint {
  ID: number
  RaceID: number
  Code: string
  DisplayName: string
  DisplayOrder: number
  DistanceFromStart: number | null
  CreatedAt: string
}

export type RunnerStatus = 'UNKNOWN' | 'ACTIVE' | 'DNS' | 'DNF' | 'FINISHED' | 'MOVED'
export type LogSource = 'MESHTASTIC' | 'MANUAL' | 'WINLINK_IMPORT'

export interface Runner {
  ID: number
  RaceID: number
  BibNumber: number
  FirstName: string
  LastName: string
  SortOrder: number
  Status: RunnerStatus
  CreatedAt: string
  UpdatedAt: string
}

export interface CheckpointLog {
  ID: number
  RunnerID: number
  CheckpointID: number
  RecordedAt: string
  Source: LogSource
  RawMessage: string
  CreatedAt: string
}

export interface ActiveSessionCheckpoint {
  RaceID: number
  CheckpointID: number
}

export interface ActiveSession {
  EventID: number | null
  Checkpoints: ActiveSessionCheckpoint[]
}

export interface LogBibResult {
  runner: Runner
  log: CheckpointLog
  is_duplicate: boolean
}

export interface WinlinkSkipDetail {
  Position: number
  BibNumber: number
  Reason: string // "blank" | "no_runner" | "duplicate" | "parse_error"
}

export interface WinlinkImportResult {
  Created: number
  Updated: number
  Skipped: number
  SkippedDetails: WinlinkSkipDetail[]
  Errors: string[]
}

// SSE event envelope
export interface SSEEvent<T = unknown> {
  type: 'connected' | 'bib_logged' | 'session_changed'
  payload: T
}

import { useEffect, useState } from 'react'
import {
  Box,
  Typography,
  Divider,
  TextField,
  Button,
  Select,
  MenuItem,
  FormControl,
  InputLabel,
  Stack,
  Chip,
  Alert,
  Paper,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogContentText,
  DialogActions,
  Tooltip,
  Switch,
  FormControlLabel,
  Accordion,
  AccordionSummary,
  AccordionDetails,
} from '@mui/material'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import DeleteIcon from '@mui/icons-material/Delete'
import ArchiveIcon from '@mui/icons-material/Archive'
import FileDownloadIcon from '@mui/icons-material/FileDownload'
import LockIcon from '@mui/icons-material/Lock'
import EditIcon from '@mui/icons-material/Edit'
import CheckIcon from '@mui/icons-material/Check'
import CloseIcon from '@mui/icons-material/Close'
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward'
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward'
import type {
  Event,
  Race,
  Checkpoint,
  ActiveSession,
  Runner,
  RunnerStatus,
} from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'

const ACCORDION_STORAGE_KEY = 'ares-bib-logger:admin-accordions-open'

// localStorage can be unavailable (private browsing, disabled storage, some
// test environments) — fail open rather than crash the app.
function loadOpenAccordions(): Set<string> {
  try {
    const stored = window.localStorage?.getItem(ACCORDION_STORAGE_KEY)
    const names = stored ? (JSON.parse(stored) as unknown) : []
    return new Set(Array.isArray(names) ? names.filter((n) => typeof n === 'string') : [])
  } catch {
    return new Set()
  }
}

function storeOpenAccordions(names: Set<string>) {
  try {
    window.localStorage?.setItem(ACCORDION_STORAGE_KEY, JSON.stringify([...names]))
  } catch {
    // ignore — persistence is best-effort
  }
}

export default function AdminTab() {
  const [openAccordions, setOpenAccordions] = useState<Set<string>>(loadOpenAccordions)
  const toggleAccordion = (name: string) => (_: unknown, expanded: boolean) => {
    setOpenAccordions((prev) => {
      const next = new Set(prev)
      if (expanded) next.add(name)
      else next.delete(name)
      storeOpenAccordions(next)
      return next
    })
  }

  const [events, setEvents] = useState<Event[]>([])
  const [session, setSession] = useState<ActiveSession | null>(null)
  const [races, setRaces] = useState<Race[]>([])
  const [checkpointsByRace, setCheckpointsByRace] = useState<Record<number, Checkpoint[]>>({})
  const [error, setError] = useState('')

  // Create-event form
  const [newEventName, setNewEventName] = useState('')
  // Create-race form
  const [newRaceName, setNewRaceName] = useState('')
  // Checkpoint create form
  const [cpRaceID, setCpRaceID] = useState<number | ''>('')
  const [cpCode, setCpCode] = useState('')
  const [cpName, setCpName] = useState('')
  const [cpDist, setCpDist] = useState('')
  const [cpColumnName, setCpColumnName] = useState('')
  // Checkpoint inline edit
  const [editingCpID, setEditingCpID] = useState<number | null>(null)
  const [editCode, setEditCode] = useState('')
  const [editName, setEditName] = useState('')
  const [editDist, setEditDist] = useState('')
  const [editColumnName, setEditColumnName] = useState('')
  // Roster import
  const [rosterRaceID, setRosterRaceID] = useState<number | ''>('')
  const [rosterTsv, setRosterTsv] = useState('')
  const [rosterMsg, setRosterMsg] = useState('')
  // Bulk checkpoint import
  const [bulkCpRaceID, setBulkCpRaceID] = useState<number | ''>('')
  const [bulkCpTsv, setBulkCpTsv] = useState('')
  const [bulkCpMsg, setBulkCpMsg] = useState('')
  // Event config import
  const [importConfigJson, setImportConfigJson] = useState('')
  const [importConfigMsg, setImportConfigMsg] = useState('')

  // Runner status
  const [statusRaceID, setStatusRaceID] = useState<number | ''>('')
  const [statusBib, setStatusBib] = useState('')
  const [statusRunner, setStatusRunner] = useState<Runner | null>(null)
  const [statusNew, setStatusNew] = useState<RunnerStatus>('ACTIVE')
  const [statusMsg, setStatusMsg] = useState('')
  const [statusSearchErr, setStatusSearchErr] = useState('')

  // Correction: manually log a bib with an explicit time
  const [correctionRaceID, setCorrectionRaceID] = useState<number | ''>('')
  const [correctionCheckpointID, setCorrectionCheckpointID] = useState<number | ''>('')
  const [correctionBib, setCorrectionBib] = useState('')
  const [correctionTime, setCorrectionTime] = useState('')
  const [correctionMsg, setCorrectionMsg] = useState('')
  const [correctionErr, setCorrectionErr] = useState('')

  // Correction: remove a checkpoint log
  const [removeLogRaceID, setRemoveLogRaceID] = useState<number | ''>('')
  const [removeLogCheckpointID, setRemoveLogCheckpointID] = useState<number | ''>('')
  const [removeLogBib, setRemoveLogBib] = useState('')
  const [removeLogMsg, setRemoveLogMsg] = useState('')
  const [removeLogErr, setRemoveLogErr] = useState('')
  const [removeLogTarget, setRemoveLogTarget] = useState<{
    raceID: number
    checkpointID: number
    bibNumber: number
  } | null>(null)

  // Confirmation dialogs
  const [deleteTarget, setDeleteTarget] = useState<{
    type: 'race' | 'checkpoint'
    id: number
    label: string
  } | null>(null)
  const [archiveTarget, setArchiveTarget] = useState<{ id: number; label: string } | null>(null)
  const [lockTarget, setLockTarget] = useState<{ id: number; label: string } | null>(null)
  const [rosterConfirm, setRosterConfirm] = useState(false)

  const loadSession = () =>
    api
      .getSession()
      .then(setSession)
      .catch(() => {})

  const loadEvents = () =>
    api
      .listEvents()
      .then(setEvents)
      .catch(() => {})

  const loadRaces = (eventID: number) =>
    api
      .listRaces(eventID)
      .then(setRaces)
      .catch(() => {})

  const loadCheckpoints = async (raceIDs: number[]) => {
    const entries = await Promise.all(
      raceIDs.map(async (id) => [id, await api.listCheckpoints(id)] as [number, Checkpoint[]]),
    )
    setCheckpointsByRace(Object.fromEntries(entries))
  }

  useEffect(() => {
    loadEvents()
    loadSession()
  }, [])

  useEffect(() => {
    if (session?.EventID) loadRaces(session.EventID)
    else setRaces([])
  }, [session?.EventID])

  useEffect(() => {
    if (races.length) loadCheckpoints(races.map((r) => r.ID))
  }, [races])

  useStream({
    onSessionChanged: (payload) => setSession(payload as ActiveSession),
  })

  const activeCheckpointFor = (raceID: number) =>
    session?.Checkpoints?.find((c) => c.RaceID === raceID)?.CheckpointID ?? null

  // Guard against out-of-range MUI warning: only use the CP ID as value once options have loaded.
  const activeCpSelectValue = (raceID: number) => {
    const cpID = activeCheckpointFor(raceID)
    return cpID !== null && (checkpointsByRace[raceID] ?? []).some((cp) => cp.ID === cpID)
      ? cpID
      : ''
  }

  const wrap = (fn: () => Promise<unknown>, onDone?: () => void) =>
    fn()
      .then(() => {
        setError('')
        onDone?.()
      })
      .catch((e: Error) => setError(e.message))

  const confirmDelete = () => {
    if (!deleteTarget) return
    const { type, id } = deleteTarget
    if (type === 'race') {
      wrap(
        () => api.deleteRace(id).then(() => loadRaces(session!.EventID!)),
        () => setDeleteTarget(null),
      )
    } else {
      wrap(
        () => api.deleteCheckpoint(id).then(() => loadCheckpoints(races.map((r) => r.ID))),
        () => setDeleteTarget(null),
      )
    }
  }

  const confirmArchive = () => {
    if (!archiveTarget) return
    wrap(() =>
      api.archiveEvent(archiveTarget.id).then(() => {
        setArchiveTarget(null)
        return Promise.all([loadEvents(), loadSession()])
      }),
    )
  }

  const confirmLockOrder = () => {
    if (!lockTarget) return
    wrap(() =>
      api.lockRaceOrder(lockTarget.id).then(() => {
        setLockTarget(null)
        return loadRaces(session!.EventID!)
      }),
    )
  }

  const confirmRosterImport = () => {
    setRosterConfirm(false)
    wrap(() =>
      api.importRoster(Number(rosterRaceID), rosterTsv).then((r) => {
        setRosterMsg(`Imported ${r.imported} runners.`)
        setRosterTsv('')
        return loadRaces(session!.EventID!)
      }),
    )
  }

  const searchRunner = async () => {
    if (!statusRaceID || !statusBib.trim()) return
    setStatusRunner(null)
    setStatusMsg('')
    setStatusSearchErr('')
    try {
      const runners = await api.listRunners(Number(statusRaceID))
      const found = runners.find((r) => r.BibNumber === Number(statusBib))
      if (!found) {
        setStatusSearchErr(`Bib ${statusBib} not found in this race.`)
      } else {
        setStatusRunner(found)
        setStatusNew(
          found.Status === 'MOVED' || found.Status === 'UNKNOWN' ? 'ACTIVE' : found.Status,
        )
      }
    } catch (e: unknown) {
      setStatusSearchErr((e as Error).message)
    }
  }

  const applyRunnerStatus = async () => {
    if (!statusRunner) return
    try {
      await api.logStatus(statusRunner.BibNumber, statusNew)
      setStatusRunner({ ...statusRunner, Status: statusNew })
      setStatusMsg(`Status updated to ${statusNew}.`)
      setStatusSearchErr('')
    } catch (e: unknown) {
      setStatusSearchErr((e as Error).message)
    }
  }

  const submitCorrection = async () => {
    if (
      !correctionRaceID ||
      !correctionCheckpointID ||
      !correctionBib.trim() ||
      !correctionTime.trim()
    )
      return
    setCorrectionMsg('')
    setCorrectionErr('')
    try {
      await api.correctLog(
        Number(correctionRaceID),
        Number(correctionCheckpointID),
        Number(correctionBib),
        correctionTime.trim(),
      )
      setCorrectionMsg(`Logged bib ${correctionBib} at ${correctionTime}.`)
      setCorrectionBib('')
      setCorrectionTime('')
    } catch (e: unknown) {
      setCorrectionErr((e as Error).message)
    }
  }

  const confirmRemoveLog = async () => {
    if (!removeLogTarget) return
    setRemoveLogMsg('')
    setRemoveLogErr('')
    try {
      await api.deleteLog(
        removeLogTarget.raceID,
        removeLogTarget.checkpointID,
        removeLogTarget.bibNumber,
      )
      setRemoveLogMsg(`Removed log for bib ${removeLogTarget.bibNumber}.`)
      setRemoveLogBib('')
    } catch (e: unknown) {
      setRemoveLogErr((e as Error).message)
    } finally {
      setRemoveLogTarget(null)
    }
  }

  const moveCheckpoint = (raceID: number, cp: Checkpoint, direction: 'up' | 'down') => {
    const cps = [...(checkpointsByRace[raceID] ?? [])].sort(
      (a, b) => a.DisplayOrder - b.DisplayOrder,
    )
    const idx = cps.findIndex((c) => c.ID === cp.ID)
    const swapIdx = direction === 'up' ? idx - 1 : idx + 1
    if (swapIdx < 0 || swapIdx >= cps.length) return
    const reordered = [...cps]
    ;[reordered[idx], reordered[swapIdx]] = [reordered[swapIdx], reordered[idx]]
    wrap(() =>
      api
        .reorderCheckpoints(
          raceID,
          reordered.map((c) => c.ID),
        )
        .then(() => loadCheckpoints(races.map((r) => r.ID))),
    )
  }

  const startEditCp = (cp: Checkpoint) => {
    setEditingCpID(cp.ID)
    setEditCode(cp.Code)
    setEditName(cp.DisplayName)
    setEditDist(cp.DistanceFromStart != null ? String(cp.DistanceFromStart) : '')
    setEditColumnName(cp.ColumnName ?? '')
  }

  const saveEditCp = () => {
    if (!editingCpID || !editCode.trim() || !editName.trim()) return
    const dist = editDist.trim() ? parseFloat(editDist) : null
    const columnName = editColumnName.trim() ? editColumnName.trim() : null
    wrap(() =>
      api
        .updateCheckpoint(editingCpID, editCode.trim(), editName.trim(), dist, columnName)
        .then(() => {
          setEditingCpID(null)
          return loadCheckpoints(races.map((r) => r.ID))
        }),
    )
  }

  const cancelEditCp = () => setEditingCpID(null)

  return (
    <Box sx={{ maxWidth: 800 }}>
      <Typography variant="h5" gutterBottom>
        Admin
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}

      <Accordion expanded={openAccordions.has('setup')} onChange={toggleAccordion('setup')}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="h6">Setup</Typography>
        </AccordionSummary>
        <AccordionDetails>
          {/* ── Active event ── */}
          <Typography variant="h6" gutterBottom>
            Active Event
          </Typography>
          <Stack direction="row" spacing={1} sx={{ mb: 1, alignItems: 'center' }}>
            <FormControl size="small" sx={{ minWidth: 220 }}>
              <InputLabel id="event-label">Event</InputLabel>
              <Select
                value={
                  events.some((e) => e.ID === session?.EventID) ? (session?.EventID ?? '') : ''
                }
                label="Event"
                labelId="event-label"
                onChange={(e) =>
                  wrap(() => api.setSessionEvent(Number(e.target.value)), loadSession)
                }
              >
                {events.map((ev) => (
                  <MenuItem key={ev.ID} value={ev.ID}>
                    {ev.Name}
                  </MenuItem>
                ))}
              </Select>
            </FormControl>
            {session?.EventID && (
              <Chip label={`Event #${session.EventID} active`} color="success" size="small" />
            )}
            {session?.EventID && (
              <Tooltip
                title="Export event config (races, checkpoints, roster) as JSON for sharing with other stations"
                describeChild
              >
                <IconButton
                  size="small"
                  aria-label="Export event config"
                  onClick={async () => {
                    const eventID = session?.EventID
                    if (!eventID) return
                    const ev = events.find((e) => e.ID === eventID)
                    try {
                      const json = await api.exportEventConfig(eventID)
                      const blob = new Blob([json], { type: 'application/json' })
                      const url = URL.createObjectURL(blob)
                      const a = document.createElement('a')
                      a.href = url
                      a.download = `event-${(ev?.Name ?? eventID).toString().replace(/\s+/g, '-').toLowerCase()}.json`
                      a.click()
                      URL.revokeObjectURL(url)
                    } catch {
                      setError('Failed to export event config')
                    }
                  }}
                >
                  <FileDownloadIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            )}
            {session?.EventID && (
              <Tooltip title="Archive event — hides it from the dropdown; data is preserved">
                <IconButton
                  size="small"
                  aria-label="Archive this event"
                  onClick={() => {
                    const ev = events.find((e) => e.ID === session.EventID)
                    if (ev) setArchiveTarget({ id: ev.ID, label: ev.Name })
                  }}
                >
                  <ArchiveIcon fontSize="small" />
                </IconButton>
              </Tooltip>
            )}
          </Stack>

          {session?.EventID &&
            (() => {
              const activeEvent = events.find((e) => e.ID === session.EventID)
              if (!activeEvent) return null
              return (
                <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
                  <Tooltip title="Some stations' Winlink exports have a blank row between the header and the first runner's data row. Enable this to match that convention on both export and import for this event.">
                    <FormControlLabel
                      control={
                        <Switch
                          size="small"
                          checked={activeEvent.WinlinkBlankLineAfterHeader}
                          onChange={(e) =>
                            wrap(
                              () => api.setEventWinlinkFormat(activeEvent.ID, e.target.checked),
                              loadEvents,
                            )
                          }
                        />
                      }
                      label="Blank line between header and first row (Winlink)"
                    />
                  </Tooltip>
                </Stack>
              )
            })()}

          <Stack direction="row" spacing={1} sx={{ mb: 3 }}>
            <TextField
              size="small"
              label="New event name"
              value={newEventName}
              onChange={(e) => setNewEventName(e.target.value)}
            />
            <Tooltip title="Create a new event">
              <span>
                <Button
                  variant="outlined"
                  disabled={!newEventName.trim()}
                  onClick={() =>
                    wrap(() =>
                      api.createEvent(newEventName.trim()).then(() => {
                        setNewEventName('')
                        return loadEvents()
                      }),
                    )
                  }
                >
                  Create Event
                </Button>
              </span>
            </Tooltip>
          </Stack>

          <Stack direction="row" spacing={1} sx={{ mb: 2, alignItems: 'flex-start' }}>
            <TextField
              size="small"
              multiline
              rows={3}
              label="Import event config (paste JSON)"
              placeholder='{"version":1,"event":{"name":"..."},...}'
              value={importConfigJson}
              onChange={(e) => {
                setImportConfigJson(e.target.value)
                setImportConfigMsg('')
              }}
              sx={{ flex: 1 }}
            />
            <Tooltip
              title="Import event config — creates a new event with races, checkpoints, and roster from the JSON"
              describeChild
            >
              <span>
                <Button
                  variant="outlined"
                  disabled={!importConfigJson.trim()}
                  onClick={async () => {
                    try {
                      const result = await api.importEventConfig(importConfigJson.trim())
                      setImportConfigMsg(`Imported event #${result.event_id}`)
                      setImportConfigJson('')
                      await loadEvents()
                    } catch (e: unknown) {
                      setImportConfigMsg(`Error: ${(e as Error).message}`)
                    }
                  }}
                >
                  Import Config
                </Button>
              </span>
            </Tooltip>
          </Stack>
          {importConfigMsg && (
            <Alert
              severity={importConfigMsg.startsWith('Error') ? 'error' : 'success'}
              sx={{ mb: 2 }}
            >
              {importConfigMsg}
            </Alert>
          )}

          <Divider sx={{ my: 2 }} />

          {/* ── Races ── */}
          <Typography variant="h6" gutterBottom>
            Races
          </Typography>
          {!session?.EventID && (
            <Alert severity="info" sx={{ mb: 2 }}>
              Select an active event first.
            </Alert>
          )}
          {session?.EventID && (
            <>
              <Stack direction="row" spacing={1} sx={{ mb: 2 }}>
                <TextField
                  size="small"
                  label="New race name"
                  value={newRaceName}
                  onChange={(e) => setNewRaceName(e.target.value)}
                />
                <Tooltip title="Add a new race to this event">
                  <span>
                    <Button
                      variant="outlined"
                      disabled={!newRaceName.trim()}
                      onClick={() =>
                        wrap(() =>
                          api.createRace(session.EventID!, newRaceName.trim()).then(() => {
                            setNewRaceName('')
                            return loadRaces(session.EventID!)
                          }),
                        )
                      }
                    >
                      Create Race
                    </Button>
                  </span>
                </Tooltip>
              </Stack>

              {races.map((race) => (
                <Paper key={race.ID} sx={{ p: 2, mb: 2 }}>
                  <Stack
                    direction="row"
                    sx={{ justifyContent: 'space-between', alignItems: 'center' }}
                  >
                    <Typography variant="subtitle1" sx={{ fontWeight: 'bold' }}>
                      {race.Name}
                    </Typography>
                    <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                      {race.RosterLocked && (
                        <Chip label="Roster locked" size="small" color="warning" />
                      )}
                      {race.OrderLocked ? (
                        <Chip label="Order locked" size="small" color="warning" />
                      ) : (
                        <Tooltip title="Lock checkpoint order — prevents mid-race column shifts that break Winlink import">
                          <IconButton
                            size="small"
                            aria-label="Lock checkpoint order"
                            onClick={() => setLockTarget({ id: race.ID, label: race.Name })}
                          >
                            <LockIcon fontSize="small" />
                          </IconButton>
                        </Tooltip>
                      )}
                      <Tooltip title="Delete race and all its data">
                        <IconButton
                          size="small"
                          color="error"
                          aria-label="Delete race and all its data"
                          onClick={() =>
                            setDeleteTarget({ type: 'race', id: race.ID, label: race.Name })
                          }
                        >
                          <DeleteIcon fontSize="small" />
                        </IconButton>
                      </Tooltip>
                    </Stack>
                  </Stack>

                  {/* Active checkpoint selector */}
                  <Stack direction="row" spacing={1} sx={{ mt: 1, alignItems: 'center' }}>
                    <FormControl size="small" sx={{ minWidth: 180 }}>
                      <InputLabel id={`active-cp-label-${race.ID}`}>Active Checkpoint</InputLabel>
                      <Select
                        value={activeCpSelectValue(race.ID)}
                        label="Active Checkpoint"
                        labelId={`active-cp-label-${race.ID}`}
                        onChange={(e) => {
                          if (!e.target.value) {
                            wrap(() => api.clearSessionCheckpoint(race.ID), loadSession)
                          } else {
                            wrap(
                              () => api.setSessionCheckpoint(race.ID, Number(e.target.value)),
                              loadSession,
                            )
                          }
                        }}
                      >
                        <MenuItem value="">— none —</MenuItem>
                        {(checkpointsByRace[race.ID] ?? []).map((cp) => (
                          <MenuItem key={cp.ID} value={cp.ID}>
                            {cp.Code} – {cp.DisplayName}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                  </Stack>

                  {/* Checkpoint list + reorder + edit + delete */}
                  <Table size="small" sx={{ mt: 1 }}>
                    <TableHead>
                      <TableRow>
                        <TableCell>Order</TableCell>
                        <TableCell>Code</TableCell>
                        <TableCell>Name</TableCell>
                        <TableCell>Column Name</TableCell>
                        <TableCell>Dist (mi)</TableCell>
                        <TableCell align="right">Actions</TableCell>
                      </TableRow>
                    </TableHead>
                    <TableBody>
                      {(checkpointsByRace[race.ID] ?? [])
                        .sort((a, b) => a.DisplayOrder - b.DisplayOrder)
                        .map((cp, idx, arr) => (
                          <TableRow key={cp.ID}>
                            <TableCell>{cp.DisplayOrder}</TableCell>
                            {editingCpID === cp.ID ? (
                              <>
                                <TableCell>
                                  <TextField
                                    size="small"
                                    value={editCode}
                                    onChange={(e) => setEditCode(e.target.value)}
                                    sx={{ width: 90 }}
                                    autoFocus
                                  />
                                </TableCell>
                                <TableCell>
                                  <TextField
                                    size="small"
                                    value={editName}
                                    onChange={(e) => setEditName(e.target.value)}
                                    sx={{ width: 180 }}
                                  />
                                </TableCell>
                                <TableCell>
                                  <TextField
                                    size="small"
                                    value={editColumnName}
                                    onChange={(e) => setEditColumnName(e.target.value)}
                                    sx={{ width: 140 }}
                                    placeholder={editName}
                                  />
                                </TableCell>
                                <TableCell>
                                  <TextField
                                    size="small"
                                    type="number"
                                    value={editDist}
                                    onChange={(e) => setEditDist(e.target.value)}
                                    sx={{ width: 80 }}
                                    slotProps={{ htmlInput: { step: '0.1', min: '0' } }}
                                  />
                                </TableCell>
                              </>
                            ) : (
                              <>
                                <TableCell>{cp.Code}</TableCell>
                                <TableCell>{cp.DisplayName}</TableCell>
                                <TableCell>{cp.ColumnName || '—'}</TableCell>
                                <TableCell>
                                  {cp.DistanceFromStart != null ? cp.DistanceFromStart : '—'}
                                </TableCell>
                              </>
                            )}
                            <TableCell align="right">
                              <Stack
                                direction="row"
                                spacing={0}
                                sx={{ justifyContent: 'flex-end' }}
                              >
                                {editingCpID === cp.ID ? (
                                  <>
                                    <Tooltip title="Save">
                                      <IconButton
                                        size="small"
                                        color="success"
                                        aria-label="Save"
                                        onClick={saveEditCp}
                                      >
                                        <CheckIcon fontSize="small" />
                                      </IconButton>
                                    </Tooltip>
                                    <Tooltip title="Cancel">
                                      <IconButton
                                        size="small"
                                        aria-label="Cancel"
                                        onClick={cancelEditCp}
                                      >
                                        <CloseIcon fontSize="small" />
                                      </IconButton>
                                    </Tooltip>
                                  </>
                                ) : (
                                  <>
                                    {!race.OrderLocked && (
                                      <>
                                        <Tooltip title="Move checkpoint up">
                                          <span>
                                            <IconButton
                                              size="small"
                                              aria-label="Move checkpoint up"
                                              disabled={idx === 0}
                                              onClick={() => moveCheckpoint(race.ID, cp, 'up')}
                                            >
                                              <ArrowUpwardIcon fontSize="small" />
                                            </IconButton>
                                          </span>
                                        </Tooltip>
                                        <Tooltip title="Move checkpoint down">
                                          <span>
                                            <IconButton
                                              size="small"
                                              aria-label="Move checkpoint down"
                                              disabled={idx === arr.length - 1}
                                              onClick={() => moveCheckpoint(race.ID, cp, 'down')}
                                            >
                                              <ArrowDownwardIcon fontSize="small" />
                                            </IconButton>
                                          </span>
                                        </Tooltip>
                                        <Tooltip title="Edit checkpoint code and name">
                                          <IconButton
                                            size="small"
                                            aria-label="Edit checkpoint code and name"
                                            onClick={() => startEditCp(cp)}
                                          >
                                            <EditIcon fontSize="small" />
                                          </IconButton>
                                        </Tooltip>
                                        <Tooltip title="Delete checkpoint">
                                          <IconButton
                                            size="small"
                                            color="error"
                                            aria-label="Delete checkpoint"
                                            onClick={() =>
                                              setDeleteTarget({
                                                type: 'checkpoint',
                                                id: cp.ID,
                                                label: `${cp.Code} – ${cp.DisplayName}`,
                                              })
                                            }
                                          >
                                            <DeleteIcon fontSize="small" />
                                          </IconButton>
                                        </Tooltip>
                                      </>
                                    )}
                                  </>
                                )}
                              </Stack>
                            </TableCell>
                          </TableRow>
                        ))}
                    </TableBody>
                  </Table>

                  {!race.OrderLocked && (
                    <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                      <TextField
                        size="small"
                        label="Code"
                        value={cpRaceID === race.ID ? cpCode : ''}
                        onChange={(e) => {
                          setCpRaceID(race.ID)
                          setCpCode(e.target.value)
                        }}
                        sx={{ width: 100 }}
                      />
                      <TextField
                        size="small"
                        label="Display name"
                        value={cpRaceID === race.ID ? cpName : ''}
                        onChange={(e) => {
                          setCpRaceID(race.ID)
                          setCpName(e.target.value)
                        }}
                        sx={{ width: 180 }}
                      />
                      <TextField
                        size="small"
                        label="Column name"
                        value={cpRaceID === race.ID ? cpColumnName : ''}
                        onChange={(e) => {
                          setCpRaceID(race.ID)
                          setCpColumnName(e.target.value)
                        }}
                        placeholder={cpRaceID === race.ID ? cpName : ''}
                        sx={{ width: 140 }}
                      />
                      <TextField
                        size="small"
                        label="Dist (mi)"
                        type="number"
                        value={cpRaceID === race.ID ? cpDist : ''}
                        onChange={(e) => {
                          setCpRaceID(race.ID)
                          setCpDist(e.target.value)
                        }}
                        sx={{ width: 90 }}
                        slotProps={{ htmlInput: { step: '0.1', min: '0' } }}
                      />
                      <Tooltip title="Add checkpoint to this race">
                        <span>
                          <Button
                            variant="outlined"
                            size="small"
                            disabled={cpRaceID !== race.ID || !cpCode.trim() || !cpName.trim()}
                            onClick={() => {
                              const dist = cpDist.trim() ? parseFloat(cpDist) : null
                              const columnName = cpColumnName.trim() ? cpColumnName.trim() : null
                              wrap(() =>
                                api
                                  .createCheckpoint(
                                    race.ID,
                                    cpCode.trim(),
                                    cpName.trim(),
                                    dist,
                                    columnName,
                                  )
                                  .then(() => {
                                    setCpCode('')
                                    setCpName('')
                                    setCpDist('')
                                    setCpColumnName('')
                                    setCpRaceID('')
                                    return loadCheckpoints(races.map((r) => r.ID))
                                  }),
                              )
                            }}
                          >
                            Add Checkpoint
                          </Button>
                        </span>
                      </Tooltip>
                    </Stack>
                  )}
                </Paper>
              ))}
            </>
          )}

          <Divider sx={{ my: 2 }} />

          {/* ── Roster Import ── */}
          <Typography variant="h6" gutterBottom>
            Roster Import
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Paste TSV with columns: BibNumber, FirstName, LastName (no header row). Importing locks
            the roster permanently.
          </Typography>
          <Stack spacing={1} data-testid="roster-section">
            <FormControl size="small" sx={{ maxWidth: 220 }}>
              <InputLabel id="roster-race-label">Race</InputLabel>
              <Select
                value={rosterRaceID}
                label="Race"
                labelId="roster-race-label"
                onChange={(e) => setRosterRaceID(Number(e.target.value))}
              >
                {races
                  .filter((r) => !r.RosterLocked)
                  .map((r) => (
                    <MenuItem key={r.ID} value={r.ID}>
                      {r.Name}
                    </MenuItem>
                  ))}
              </Select>
            </FormControl>
            <TextField
              multiline
              rows={6}
              size="small"
              placeholder="101&#9;Alice&#9;Smith&#10;102&#9;Bob&#9;Jones"
              value={rosterTsv}
              onChange={(e) => setRosterTsv(e.target.value)}
              sx={{ fontFamily: 'monospace' }}
            />
            <Box>
              <Tooltip title="Import and permanently lock roster for this race">
                <span>
                  <Button
                    variant="contained"
                    disabled={!rosterRaceID || !rosterTsv.trim()}
                    onClick={() => setRosterConfirm(true)}
                  >
                    Import Roster
                  </Button>
                </span>
              </Tooltip>
              {rosterMsg && (
                <Typography variant="body2" sx={{ ml: 2, display: 'inline' }}>
                  {rosterMsg}
                </Typography>
              )}
            </Box>
          </Stack>

          <Divider sx={{ my: 2 }} />

          {/* ── Bulk Checkpoint Import ── */}
          <Typography variant="h6" gutterBottom>
            Bulk Checkpoint Import
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Paste TSV with columns: Code, DisplayName, DistFromStart, ColumnName (distance and
            column name optional, no header row).
          </Typography>
          <Stack spacing={1} data-testid="bulk-cp-section">
            <FormControl size="small" sx={{ maxWidth: 220 }}>
              <InputLabel id="bulk-cp-race-label">Race</InputLabel>
              <Select
                value={bulkCpRaceID}
                label="Race"
                labelId="bulk-cp-race-label"
                onChange={(e) => {
                  setBulkCpRaceID(Number(e.target.value))
                  setBulkCpMsg('')
                }}
              >
                {races
                  .filter((r) => !r.OrderLocked)
                  .map((r) => (
                    <MenuItem key={r.ID} value={r.ID}>
                      {r.Name}
                    </MenuItem>
                  ))}
              </Select>
            </FormControl>
            <TextField
              multiline
              rows={6}
              size="small"
              placeholder={'AS1\tAid Station 1\t10.5\tAS #1\nAS2\tAid Station 2\t21.0\tAS #2'}
              value={bulkCpTsv}
              onChange={(e) => setBulkCpTsv(e.target.value)}
              sx={{ fontFamily: 'monospace', maxWidth: 500 }}
            />
            <Box>
              <Tooltip title="Create all checkpoints from the pasted TSV">
                <span>
                  <Button
                    variant="contained"
                    disabled={!bulkCpRaceID || !bulkCpTsv.trim()}
                    onClick={async () => {
                      const rows = bulkCpTsv
                        .trim()
                        .split('\n')
                        .map((l) => l.split('\t'))
                      let created = 0
                      const errs: string[] = []
                      for (const [code, name, dist, columnName] of rows) {
                        if (!code?.trim() || !name?.trim()) {
                          errs.push(`Skipped: "${code ?? ''}" — code and name required`)
                          continue
                        }
                        const distVal = dist?.trim() ? parseFloat(dist.trim()) : null
                        const columnNameVal = columnName?.trim() ? columnName.trim() : null
                        try {
                          await api.createCheckpoint(
                            Number(bulkCpRaceID),
                            code.trim(),
                            name.trim(),
                            distVal,
                            columnNameVal,
                          )
                          created++
                        } catch (e: unknown) {
                          errs.push(`${code.trim()}: ${(e as Error).message}`)
                        }
                      }
                      await loadCheckpoints(races.map((r) => r.ID))
                      setBulkCpTsv('')
                      setBulkCpMsg(
                        `Created ${created}${errs.length ? ` — errors: ${errs.join('; ')}` : ''}`,
                      )
                    }}
                  >
                    Import Checkpoints
                  </Button>
                </span>
              </Tooltip>
              {bulkCpMsg && (
                <Typography variant="body2" sx={{ ml: 2, display: 'inline' }}>
                  {bulkCpMsg}
                </Typography>
              )}
            </Box>
          </Stack>
        </AccordionDetails>
      </Accordion>

      <Accordion expanded={openAccordions.has('runners')} onChange={toggleAccordion('runners')}>
        <AccordionSummary expandIcon={<ExpandMoreIcon />}>
          <Typography variant="h6">Edit Runners</Typography>
        </AccordionSummary>
        <AccordionDetails>
          {/* ── Runner Status ── */}
          <Typography variant="h6" gutterBottom>
            Change Runner Status
          </Typography>
          {!session?.EventID && (
            <Alert severity="info" sx={{ mb: 2 }}>
              Select an active event first.
            </Alert>
          )}
          {session?.EventID && (
            <Stack spacing={2} data-testid="runner-status-form">
              <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-end' }}>
                <FormControl size="small" sx={{ minWidth: 160 }}>
                  <InputLabel id="status-race-label">Race</InputLabel>
                  <Select
                    value={statusRaceID}
                    label="Race"
                    labelId="status-race-label"
                    onChange={(e) => {
                      setStatusRaceID(Number(e.target.value))
                      setStatusRunner(null)
                      setStatusMsg('')
                    }}
                  >
                    {races.map((r) => (
                      <MenuItem key={r.ID} value={r.ID}>
                        {r.Name}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
                <TextField
                  size="small"
                  label="Bib number"
                  type="number"
                  value={statusBib}
                  onChange={(e) => {
                    setStatusBib(e.target.value)
                    setStatusRunner(null)
                    setStatusMsg('')
                  }}
                  onKeyDown={(e) => e.key === 'Enter' && searchRunner()}
                  sx={{ width: 120 }}
                />
                <Tooltip title="Find runner in this race">
                  <span>
                    <Button
                      variant="outlined"
                      disabled={!statusRaceID || !statusBib.trim()}
                      onClick={searchRunner}
                    >
                      Search
                    </Button>
                  </span>
                </Tooltip>
              </Stack>

              {statusSearchErr && <Alert severity="error">{statusSearchErr}</Alert>}

              {statusRunner && (
                <Paper sx={{ p: 2 }}>
                  <Typography variant="body2" sx={{ mb: 1 }}>
                    <strong>
                      {statusRunner.FirstName} {statusRunner.LastName}
                    </strong>{' '}
                    — Bib {statusRunner.BibNumber}
                  </Typography>
                  <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                    <Chip label={statusRunner.Status} size="small" />
                    <Typography variant="body2">→</Typography>
                    <FormControl size="small" sx={{ minWidth: 130 }}>
                      <InputLabel id="new-status-label">New status</InputLabel>
                      <Select
                        value={statusNew}
                        label="New status"
                        labelId="new-status-label"
                        onChange={(e) => setStatusNew(e.target.value as RunnerStatus)}
                      >
                        {(['ACTIVE', 'DNS', 'DNF', 'FINISHED'] as RunnerStatus[]).map((s) => (
                          <MenuItem key={s} value={s}>
                            {s}
                          </MenuItem>
                        ))}
                      </Select>
                    </FormControl>
                    <Tooltip title="Apply selected status to this runner" describeChild>
                      <Button variant="contained" size="small" onClick={applyRunnerStatus}>
                        Set
                      </Button>
                    </Tooltip>
                  </Stack>
                  {statusMsg && (
                    <Alert severity="success" sx={{ mt: 1 }}>
                      {statusMsg}
                    </Alert>
                  )}
                </Paper>
              )}
            </Stack>
          )}

          <Divider sx={{ my: 2 }} />

          {/* ── Manually log a bib (correction) ── */}
          <Typography variant="h6" gutterBottom>
            Manually Log a Bib
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Create or overwrite a runner's checkpoint log at an explicit time — for correcting a
            mis-logged bib after the fact, possibly at a checkpoint other than the one currently
            active at this station.
          </Typography>
          {!session?.EventID && (
            <Alert severity="info" sx={{ mb: 2 }}>
              Select an active event first.
            </Alert>
          )}
          {session?.EventID && (
            <Stack spacing={1} data-testid="correction-form">
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
                <FormControl size="small" sx={{ minWidth: 160 }}>
                  <InputLabel id="correction-race-label">Race</InputLabel>
                  <Select
                    value={correctionRaceID}
                    label="Race"
                    labelId="correction-race-label"
                    onChange={(e) => {
                      setCorrectionRaceID(Number(e.target.value))
                      setCorrectionCheckpointID('')
                    }}
                  >
                    {races.map((r) => (
                      <MenuItem key={r.ID} value={r.ID}>
                        {r.Name}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
                <FormControl size="small" sx={{ minWidth: 200 }}>
                  <InputLabel id="correction-cp-label">Checkpoint</InputLabel>
                  <Select
                    value={correctionCheckpointID}
                    label="Checkpoint"
                    labelId="correction-cp-label"
                    disabled={!correctionRaceID}
                    onChange={(e) => setCorrectionCheckpointID(Number(e.target.value))}
                  >
                    {(checkpointsByRace[correctionRaceID as number] ?? [])
                      .sort((a, b) => a.DisplayOrder - b.DisplayOrder)
                      .map((cp) => (
                        <MenuItem key={cp.ID} value={cp.ID}>
                          {cp.Code} – {cp.DisplayName}
                        </MenuItem>
                      ))}
                  </Select>
                </FormControl>
                <TextField
                  size="small"
                  label="Bib number"
                  type="number"
                  value={correctionBib}
                  onChange={(e) => setCorrectionBib(e.target.value)}
                  sx={{ width: 120 }}
                />
                <TextField
                  size="small"
                  label="Time"
                  placeholder="HH:MM"
                  value={correctionTime}
                  onChange={(e) => setCorrectionTime(e.target.value)}
                  sx={{ width: 100 }}
                />
                <Tooltip title="Create or overwrite this runner's log at the checkpoint with the given time">
                  <span>
                    <Button
                      variant="contained"
                      size="small"
                      disabled={
                        !correctionRaceID ||
                        !correctionCheckpointID ||
                        !correctionBib.trim() ||
                        !correctionTime.trim()
                      }
                      onClick={submitCorrection}
                    >
                      Log
                    </Button>
                  </span>
                </Tooltip>
              </Stack>
              {correctionErr && <Alert severity="error">{correctionErr}</Alert>}
              {correctionMsg && <Alert severity="success">{correctionMsg}</Alert>}
            </Stack>
          )}

          <Divider sx={{ my: 2 }} />

          {/* ── Remove a checkpoint log ── */}
          <Typography variant="h6" gutterBottom>
            Remove a Checkpoint Log
          </Typography>
          <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
            Delete a runner's log at a checkpoint — for undoing a fat-fingered bib logged at the
            wrong station.
          </Typography>
          {!session?.EventID && (
            <Alert severity="info" sx={{ mb: 2 }}>
              Select an active event first.
            </Alert>
          )}
          {session?.EventID && (
            <Stack spacing={1} data-testid="remove-log-form">
              <Stack direction="row" spacing={1} sx={{ alignItems: 'center', flexWrap: 'wrap' }}>
                <FormControl size="small" sx={{ minWidth: 160 }}>
                  <InputLabel id="remove-log-race-label">Race</InputLabel>
                  <Select
                    value={removeLogRaceID}
                    label="Race"
                    labelId="remove-log-race-label"
                    onChange={(e) => {
                      setRemoveLogRaceID(Number(e.target.value))
                      setRemoveLogCheckpointID('')
                    }}
                  >
                    {races.map((r) => (
                      <MenuItem key={r.ID} value={r.ID}>
                        {r.Name}
                      </MenuItem>
                    ))}
                  </Select>
                </FormControl>
                <FormControl size="small" sx={{ minWidth: 200 }}>
                  <InputLabel id="remove-log-cp-label">Checkpoint</InputLabel>
                  <Select
                    value={removeLogCheckpointID}
                    label="Checkpoint"
                    labelId="remove-log-cp-label"
                    disabled={!removeLogRaceID}
                    onChange={(e) => setRemoveLogCheckpointID(Number(e.target.value))}
                  >
                    {(checkpointsByRace[removeLogRaceID as number] ?? [])
                      .sort((a, b) => a.DisplayOrder - b.DisplayOrder)
                      .map((cp) => (
                        <MenuItem key={cp.ID} value={cp.ID}>
                          {cp.Code} – {cp.DisplayName}
                        </MenuItem>
                      ))}
                  </Select>
                </FormControl>
                <TextField
                  size="small"
                  label="Bib number"
                  type="number"
                  value={removeLogBib}
                  onChange={(e) => setRemoveLogBib(e.target.value)}
                  sx={{ width: 120 }}
                />
                <Tooltip title="Delete this runner's log at the selected checkpoint">
                  <span>
                    <Button
                      variant="outlined"
                      color="error"
                      size="small"
                      disabled={!removeLogRaceID || !removeLogCheckpointID || !removeLogBib.trim()}
                      onClick={() =>
                        setRemoveLogTarget({
                          raceID: Number(removeLogRaceID),
                          checkpointID: Number(removeLogCheckpointID),
                          bibNumber: Number(removeLogBib),
                        })
                      }
                    >
                      Remove
                    </Button>
                  </span>
                </Tooltip>
              </Stack>
              {removeLogErr && <Alert severity="error">{removeLogErr}</Alert>}
              {removeLogMsg && <Alert severity="success">{removeLogMsg}</Alert>}
            </Stack>
          )}
        </AccordionDetails>
      </Accordion>

      {/* ── Remove checkpoint log confirmation ── */}
      <Dialog open={!!removeLogTarget} onClose={() => setRemoveLogTarget(null)}>
        <DialogTitle>Remove Checkpoint Log</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Remove bib {removeLogTarget?.bibNumber}'s log at this checkpoint? This cannot be undone.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRemoveLogTarget(null)}>Cancel</Button>
          <Button color="error" variant="contained" onClick={confirmRemoveLog}>
            Remove
          </Button>
        </DialogActions>
      </Dialog>

      {/* ── Archive confirmation ── */}
      <Dialog open={!!archiveTarget} onClose={() => setArchiveTarget(null)}>
        <DialogTitle>Archive Event</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Archive "{archiveTarget?.label}"? It will be hidden from the event dropdown. All race
            data is preserved and can be recovered by a developer if needed.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setArchiveTarget(null)}>Cancel</Button>
          <Button color="warning" variant="contained" onClick={confirmArchive}>
            Archive
          </Button>
        </DialogActions>
      </Dialog>

      {/* ── Lock race order confirmation ── */}
      <Dialog open={!!lockTarget} onClose={() => setLockTarget(null)}>
        <DialogTitle>Lock Checkpoint Order</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Lock the checkpoint order for "{lockTarget?.label}"? Once locked, checkpoints cannot be
            reordered, edited, or deleted. This is required before Winlink import to ensure bib
            positions don't shift.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setLockTarget(null)}>Cancel</Button>
          <Button color="warning" variant="contained" onClick={confirmLockOrder}>
            Lock Order
          </Button>
        </DialogActions>
      </Dialog>

      {/* ── Roster import confirmation ── */}
      <Dialog open={rosterConfirm} onClose={() => setRosterConfirm(false)}>
        <DialogTitle>Import Roster</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Import this roster? This will permanently lock the roster for the selected race —
            runners cannot be added or removed via import again.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setRosterConfirm(false)}>Cancel</Button>
          <Button color="warning" variant="contained" onClick={confirmRosterImport}>
            Import &amp; Lock
          </Button>
        </DialogActions>
      </Dialog>

      {/* ── Delete confirmation ── */}
      <Dialog open={!!deleteTarget} onClose={() => setDeleteTarget(null)}>
        <DialogTitle>Confirm Delete</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {deleteTarget?.type === 'race'
              ? `Delete race "${deleteTarget.label}" and ALL its runners, checkpoints, and logs? This cannot be undone.`
              : `Delete checkpoint "${deleteTarget?.label}"? Any existing logs for this checkpoint will also be deleted.`}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleteTarget(null)}>Cancel</Button>
          <Button color="error" variant="contained" onClick={confirmDelete}>
            Delete
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

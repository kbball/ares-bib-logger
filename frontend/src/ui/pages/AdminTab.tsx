import { useEffect, useState } from 'react'
import {
  Box, Typography, Divider, TextField, Button, Select, MenuItem,
  FormControl, InputLabel, Stack, Chip, Alert, Paper,
  Table, TableHead, TableRow, TableCell, TableBody,
  IconButton, Dialog, DialogTitle, DialogContent, DialogContentText, DialogActions,
} from '@mui/material'
import DeleteIcon from '@mui/icons-material/Delete'
import ArchiveIcon from '@mui/icons-material/Archive'
import LockIcon from '@mui/icons-material/Lock'
import ArrowUpwardIcon from '@mui/icons-material/ArrowUpward'
import ArrowDownwardIcon from '@mui/icons-material/ArrowDownward'
import type { Event, Race, Checkpoint, ActiveSession } from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'

export default function AdminTab() {
  const [events, setEvents] = useState<Event[]>([])
  const [session, setSession] = useState<ActiveSession | null>(null)
  const [races, setRaces] = useState<Race[]>([])
  const [checkpointsByRace, setCheckpointsByRace] = useState<Record<number, Checkpoint[]>>({})
  const [error, setError] = useState('')

  // Create-event form
  const [newEventName, setNewEventName] = useState('')
  // Create-race form
  const [newRaceName, setNewRaceName] = useState('')
  // Checkpoint form
  const [cpRaceID, setCpRaceID] = useState<number | ''>('')
  const [cpCode, setCpCode] = useState('')
  const [cpName, setCpName] = useState('')
  // Roster import
  const [rosterRaceID, setRosterRaceID] = useState<number | ''>('')
  const [rosterTsv, setRosterTsv] = useState('')
  const [rosterMsg, setRosterMsg] = useState('')
  // Delete / archive confirmation dialogs
  const [deleteTarget, setDeleteTarget] = useState<{ type: 'race' | 'checkpoint'; id: number; label: string } | null>(null)
  const [archiveTarget, setArchiveTarget] = useState<{ id: number; label: string } | null>(null)

  const loadSession = () =>
    api.getSession().then(setSession).catch(() => {})

  const loadEvents = () =>
    api.listEvents().then(setEvents).catch(() => {})

  const loadRaces = (eventID: number) =>
    api.listRaces(eventID).then(setRaces).catch(() => {})

  const loadCheckpoints = async (raceIDs: number[]) => {
    const entries = await Promise.all(
      raceIDs.map(async (id) => [id, await api.listCheckpoints(id)] as [number, Checkpoint[]])
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

  const wrap = (fn: () => Promise<unknown>, onDone?: () => void) =>
    fn()
      .then(() => { setError(''); onDone?.() })
      .catch((e: Error) => setError(e.message))

  const confirmDelete = () => {
    if (!deleteTarget) return
    const { type, id } = deleteTarget
    if (type === 'race') {
      wrap(() => api.deleteRace(id).then(() => loadRaces(session!.EventID!)), () => setDeleteTarget(null))
    } else {
      wrap(
        () => api.deleteCheckpoint(id).then(() => loadCheckpoints(races.map((r) => r.ID))),
        () => setDeleteTarget(null),
      )
    }
  }

  const confirmArchive = () => {
    if (!archiveTarget) return
    wrap(
      () => api.archiveEvent(archiveTarget.id).then(() => {
        setArchiveTarget(null)
        return Promise.all([loadEvents(), loadSession()])
      }),
    )
  }

  const moveCheckpoint = (raceID: number, cp: Checkpoint, direction: 'up' | 'down') => {
    const cps = [...(checkpointsByRace[raceID] ?? [])].sort((a, b) => a.DisplayOrder - b.DisplayOrder)
    const idx = cps.findIndex((c) => c.ID === cp.ID)
    const swapIdx = direction === 'up' ? idx - 1 : idx + 1
    if (swapIdx < 0 || swapIdx >= cps.length) return
    const reordered = [...cps]
    ;[reordered[idx], reordered[swapIdx]] = [reordered[swapIdx], reordered[idx]]
    wrap(
      () => api.reorderCheckpoints(raceID, reordered.map((c) => c.ID))
        .then(() => loadCheckpoints(races.map((r) => r.ID))),
    )
  }

  return (
    <Box sx={{ maxWidth: 800 }}>
      <Typography variant="h5" gutterBottom>Admin</Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}

      {/* ── Active event ── */}
      <Typography variant="h6" gutterBottom>Active Event</Typography>
      <Stack direction="row" spacing={1} sx={{ mb: 1, alignItems: 'center' }}>
        <FormControl size="small" sx={{ minWidth: 220 }}>
          <InputLabel>Event</InputLabel>
          <Select
            value={session?.EventID ?? ''}
            label="Event"
            onChange={(e) =>
              wrap(() => api.setSessionEvent(Number(e.target.value)), loadSession)
            }
          >
            {events.map((ev) => (
              <MenuItem key={ev.ID} value={ev.ID}>{ev.Name}</MenuItem>
            ))}
          </Select>
        </FormControl>
        {session?.EventID && (
          <Chip label={`Event #${session.EventID} active`} color="success" size="small" />
        )}
        {session?.EventID && (
          <IconButton
            size="small"
            title="Archive this event — removes it from the dropdown"
            onClick={() => {
              const ev = events.find((e) => e.ID === session.EventID)
              if (ev) setArchiveTarget({ id: ev.ID, label: ev.Name })
            }}
          >
            <ArchiveIcon fontSize="small" />
          </IconButton>
        )}
      </Stack>

      <Stack direction="row" spacing={1} sx={{ mb: 3 }}>
        <TextField
          size="small"
          label="New event name"
          value={newEventName}
          onChange={(e) => setNewEventName(e.target.value)}
        />
        <Button
          variant="outlined"
          disabled={!newEventName.trim()}
          onClick={() =>
            wrap(() => api.createEvent(newEventName.trim()).then(() => {
              setNewEventName('')
              return loadEvents()
            }))
          }
        >
          Create Event
        </Button>
      </Stack>

      <Divider sx={{ my: 2 }} />

      {/* ── Races ── */}
      <Typography variant="h6" gutterBottom>Races</Typography>
      {!session?.EventID && (
        <Alert severity="info" sx={{ mb: 2 }}>Select an active event first.</Alert>
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
            <Button
              variant="outlined"
              disabled={!newRaceName.trim()}
              onClick={() =>
                wrap(() =>
                  api.createRace(session.EventID!, newRaceName.trim()).then(() => {
                    setNewRaceName('')
                    return loadRaces(session.EventID!)
                  })
                )
              }
            >
              Create Race
            </Button>
          </Stack>

          {races.map((race) => (
            <Paper key={race.ID} sx={{ p: 2, mb: 2 }}>
              <Stack direction="row" sx={{ justifyContent: 'space-between', alignItems: 'center' }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 'bold' }}>{race.Name}</Typography>
                <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
                  {race.RosterLocked && <Chip label="Roster locked" size="small" color="warning" />}
                  {race.OrderLocked
                    ? <Chip label="Order locked" size="small" color="warning" />
                    : (
                      <IconButton
                        size="small"
                        title="Finalize checkpoint order — locks it so Winlink mappings can't shift"
                        onClick={() => wrap(
                          () => api.lockRaceOrder(race.ID).then(() => loadRaces(session!.EventID!))
                        )}
                      >
                        <LockIcon fontSize="small" />
                      </IconButton>
                    )
                  }
                  <IconButton
                    size="small"
                    color="error"
                    title="Delete race and all its data"
                    onClick={() => setDeleteTarget({ type: 'race', id: race.ID, label: race.Name })}
                  >
                    <DeleteIcon fontSize="small" />
                  </IconButton>
                </Stack>
              </Stack>

              {/* Active checkpoint selector */}
              <Stack direction="row" spacing={1} sx={{ mt: 1, alignItems: 'center' }}>
                <FormControl size="small" sx={{ minWidth: 180 }}>
                  <InputLabel>Active Checkpoint</InputLabel>
                  <Select
                    value={activeCheckpointFor(race.ID) ?? ''}
                    label="Active Checkpoint"
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

              {/* Checkpoint list + reorder + delete */}
              <Table size="small" sx={{ mt: 1 }}>
                <TableHead>
                  <TableRow>
                    <TableCell>Order</TableCell>
                    <TableCell>Code</TableCell>
                    <TableCell>Name</TableCell>
                    <TableCell align="right">Actions</TableCell>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {(checkpointsByRace[race.ID] ?? [])
                    .sort((a, b) => a.DisplayOrder - b.DisplayOrder)
                    .map((cp, idx, arr) => (
                      <TableRow key={cp.ID}>
                        <TableCell>{cp.DisplayOrder}</TableCell>
                        <TableCell>{cp.Code}</TableCell>
                        <TableCell>{cp.DisplayName}</TableCell>
                        <TableCell align="right">
                          <Stack direction="row" spacing={0} sx={{ justifyContent: 'flex-end' }}>
                            {!race.OrderLocked && (
                              <>
                                <IconButton
                                  size="small"
                                  disabled={idx === 0}
                                  onClick={() => moveCheckpoint(race.ID, cp, 'up')}
                                >
                                  <ArrowUpwardIcon fontSize="small" />
                                </IconButton>
                                <IconButton
                                  size="small"
                                  disabled={idx === arr.length - 1}
                                  onClick={() => moveCheckpoint(race.ID, cp, 'down')}
                                >
                                  <ArrowDownwardIcon fontSize="small" />
                                </IconButton>
                              </>
                            )}
                            <IconButton
                              size="small"
                              color="error"
                              onClick={() => setDeleteTarget({ type: 'checkpoint', id: cp.ID, label: `${cp.Code} – ${cp.DisplayName}` })}
                            >
                              <DeleteIcon fontSize="small" />
                            </IconButton>
                          </Stack>
                        </TableCell>
                      </TableRow>
                    ))}
                </TableBody>
              </Table>

              {!race.OrderLocked && (
                <Stack direction="row" spacing={1} sx={{ mt: 1 }}>
                  <TextField size="small" label="Code" value={cpRaceID === race.ID ? cpCode : ''}
                    onChange={(e) => { setCpRaceID(race.ID); setCpCode(e.target.value) }} sx={{ width: 100 }} />
                  <TextField size="small" label="Display name" value={cpRaceID === race.ID ? cpName : ''}
                    onChange={(e) => { setCpRaceID(race.ID); setCpName(e.target.value) }} sx={{ width: 180 }} />
                  <Button variant="outlined" size="small"
                    disabled={cpRaceID !== race.ID || !cpCode.trim() || !cpName.trim()}
                    onClick={() =>
                      wrap(
                        () => api.createCheckpoint(race.ID, cpCode.trim(), cpName.trim()).then(() => {
                          setCpCode(''); setCpName(''); setCpRaceID('')
                          return loadCheckpoints(races.map((r) => r.ID))
                        })
                      )
                    }
                  >
                    Add Checkpoint
                  </Button>
                </Stack>
              )}
            </Paper>
          ))}
        </>
      )}

      <Divider sx={{ my: 2 }} />

      {/* ── Roster Import ── */}
      <Typography variant="h6" gutterBottom>Roster Import</Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        Paste TSV with columns: BibNumber, FirstName, LastName (no header row).
      </Typography>
      <Stack spacing={1}>
        <FormControl size="small" sx={{ maxWidth: 220 }}>
          <InputLabel>Race</InputLabel>
          <Select value={rosterRaceID} label="Race" onChange={(e) => setRosterRaceID(Number(e.target.value))}>
            {races
              .filter((r) => !r.RosterLocked)
              .map((r) => (
                <MenuItem key={r.ID} value={r.ID}>{r.Name}</MenuItem>
              ))}
          </Select>
        </FormControl>
        <TextField
          multiline rows={6} size="small" placeholder="101&#9;Alice&#9;Smith&#10;102&#9;Bob&#9;Jones"
          value={rosterTsv}
          onChange={(e) => setRosterTsv(e.target.value)}
          sx={{ fontFamily: 'monospace' }}
        />
        <Box>
          <Button
            variant="contained"
            disabled={!rosterRaceID || !rosterTsv.trim()}
            onClick={() => {
              wrap(
                () => api.importRoster(Number(rosterRaceID), rosterTsv).then((r) => {
                  setRosterMsg(`Imported ${r.imported} runners.`)
                  setRosterTsv('')
                  return loadRaces(session!.EventID!)
                })
              )
            }}
          >
            Import Roster
          </Button>
          {rosterMsg && <Typography variant="body2" sx={{ ml: 2, display: 'inline' }}>{rosterMsg}</Typography>}
        </Box>
      </Stack>

      {/* ── Archive confirmation dialog ── */}
      <Dialog open={!!archiveTarget} onClose={() => setArchiveTarget(null)}>
        <DialogTitle>Archive Event</DialogTitle>
        <DialogContent>
          <DialogContentText>
            Archive "{archiveTarget?.label}"? It will be hidden from the event dropdown. All race data is preserved and can be recovered by a developer if needed.
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setArchiveTarget(null)}>Cancel</Button>
          <Button color="warning" variant="contained" onClick={confirmArchive}>Archive</Button>
        </DialogActions>
      </Dialog>

      {/* ── Delete confirmation dialog ── */}
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
          <Button color="error" variant="contained" onClick={confirmDelete}>Delete</Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

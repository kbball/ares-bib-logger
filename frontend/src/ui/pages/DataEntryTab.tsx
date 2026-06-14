import { useCallback, useEffect, useRef, useState } from 'react'
import {
  Alert, Box, Button, Chip, Divider, FormControl, InputLabel,
  MenuItem, Paper, Select, Stack, Table, TableBody, TableCell,
  TableHead, TableRow, TextField, Tooltip, Typography,
} from '@mui/material'
import type { ActiveSession, Checkpoint, LogBibResult, Race, Runner } from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'

const SOURCE_LABEL: Record<string, string> = {
  MANUAL: 'Manual',
  MESHTASTIC: 'Mesh',
  WINLINK_IMPORT: 'Winlink',
}

export default function DataEntryTab() {
  const [session, setSession] = useState<ActiveSession | null>(null)
  const [races, setRaces] = useState<Race[]>([])
  const [runners, setRunners] = useState<Runner[]>([])
  const [checkpointsByRace, setCheckpointsByRace] = useState<Record<number, Checkpoint[]>>({})
  const [recentLogs, setRecentLogs] = useState<LogBibResult[]>([])
  const [dupAlert, setDupAlert] = useState<string | null>(null)

  // Manual bib entry
  const [bib, setBib] = useState('')
  const bibRef = useRef<HTMLInputElement>(null)
  const [bibError, setBibError] = useState('')

  // Status update
  const [statusBib, setStatusBib] = useState('')
  const [status, setStatus] = useState<'DNS' | 'DNF'>('DNS')
  const [statusMsg, setStatusMsg] = useState('')

  // Race transfer
  const [transferBib, setTransferBib] = useState('')
  const [transferRace, setTransferRace] = useState<number | ''>('')
  const [transferMsg, setTransferMsg] = useState('')

  const [error, setError] = useState('')

  const loadSession = useCallback(() => api.getSession().then(setSession).catch(() => {}), [])
  const loadRaces = useCallback(
    (eventID: number) => api.listRaces(eventID).then(setRaces).catch(() => {}),
    [],
  )
  const loadRunners = useCallback(async (raceIDs: number[]) => {
    const all = await Promise.all(raceIDs.map((id) => api.listRunners(id)))
    setRunners(all.flat())
  }, [])
  const loadCheckpoints = useCallback(async (raceIDs: number[]) => {
    const entries = await Promise.all(
      raceIDs.map(async (id) => [id, await api.listCheckpoints(id)] as [number, Checkpoint[]])
    )
    setCheckpointsByRace(Object.fromEntries(entries))
  }, [])

  useEffect(() => { loadSession() }, [loadSession])

  useEffect(() => {
    if (session?.EventID) loadRaces(session.EventID)
  }, [session?.EventID, loadRaces])

  useEffect(() => {
    if (races.length) {
      const ids = races.map((r) => r.ID)
      loadRunners(ids)
      loadCheckpoints(ids)
    }
  }, [races, loadRunners, loadCheckpoints])

  const pushLog = useCallback((result: LogBibResult) => {
    if (result.is_duplicate) {
      setDupAlert(`DUPLICATE: Bib ${result.runner.BibNumber} (${result.runner.FirstName} ${result.runner.LastName})`)
      setTimeout(() => setDupAlert(null), 8000)
    }
    setRecentLogs((prev) => [result, ...prev].slice(0, 50))
  }, [])

  useStream({
    onBibLogged: (payload) => {
      pushLog(payload as LogBibResult)
      loadRunners(races.map((r) => r.ID))
    },
    onSessionChanged: (payload) => setSession(payload as ActiveSession),
  })

  const hasActiveCheckpoint = (session?.Checkpoints?.length ?? 0) > 0

  const submitBib = async () => {
    const n = parseInt(bib, 10)
    if (isNaN(n) || n <= 0) { setBibError('Enter a valid bib number'); return }
    try {
      const result = await api.logBib(n)
      pushLog(result)
      setBib('')
      setBibError('')
      setError('')
      bibRef.current?.focus()
      loadRunners(races.map((r) => r.ID))
    } catch (e: unknown) {
      setBibError((e as Error).message)
    }
  }

  const submitStatus = async () => {
    const n = parseInt(statusBib, 10)
    if (isNaN(n)) { setStatusMsg('Invalid bib'); return }
    try {
      await api.logStatus(n, status)
      setStatusMsg(`Bib ${n} marked ${status}`)
      setStatusBib('')
      setError('')
      loadRunners(races.map((r) => r.ID))
    } catch (e: unknown) {
      setError((e as Error).message)
    }
  }

  const submitTransfer = async () => {
    const n = parseInt(transferBib, 10)
    if (isNaN(n) || !transferRace) return
    const runner = runners.find((r) => r.BibNumber === n)
    if (!runner) { setTransferMsg(`Bib ${n} not found`); return }
    try {
      await api.transferRunner(runner.BibNumber, runner.RaceID, Number(transferRace))
      setTransferMsg(`Bib ${n} transferred`)
      setTransferBib('')
      setTransferRace('')
      setError('')
      loadRunners(races.map((r) => r.ID))
    } catch (e: unknown) {
      setError((e as Error).message)
    }
  }

  const activeCheckpointFor = (raceID: number) => {
    const cpID = session?.Checkpoints?.find((c) => c.RaceID === raceID)?.CheckpointID
    if (!cpID) return null
    const cp = (checkpointsByRace[raceID] ?? []).find((c) => c.ID === cpID)
    return cp ?? null
  }

  return (
    <Box sx={{ maxWidth: 900 }}>
      <Typography variant="h5" gutterBottom>Data Entry</Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {dupAlert && (
        <Alert severity="warning" sx={{ mb: 2 }} onClose={() => setDupAlert(null)}>
          {dupAlert}
        </Alert>
      )}
      {!hasActiveCheckpoint && session?.EventID && (
        <Alert severity="warning" sx={{ mb: 2 }}>
          No active checkpoint set. Go to Admin and set an active checkpoint before logging bibs.
        </Alert>
      )}

      {!session?.EventID && (
        <Alert severity="info" sx={{ mb: 2 }}>No active event. Set one in Admin.</Alert>
      )}

      {/* ── Race stats ── */}
      {races.length > 0 && (
        <Stack direction="row" spacing={2} sx={{ mb: 2, flexWrap: 'wrap' }}>
          {races.map((race) => {
            const raceRunners = runners.filter((r) => r.RaceID === race.ID)
            const cp = activeCheckpointFor(race.ID)
            return (
              <Paper key={race.ID} sx={{ p: 1.5, minWidth: 160 }}>
                <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>{race.Name}</Typography>
                <Tooltip title="Total runners in this race (includes all statuses)">
                  <Typography variant="body2">Runners: {raceRunners.length}</Typography>
                </Tooltip>
                <Tooltip title="Runners still on course — status is ACTIVE or UNKNOWN">
                  <Typography variant="body2">
                    On course: {raceRunners.filter((r) => r.Status === 'ACTIVE' || r.Status === 'UNKNOWN').length}
                  </Typography>
                </Tooltip>
                <Tooltip title="Runners who did not start (DNS) or did not finish (DNF)">
                  <Typography variant="body2">
                    DNS/DNF: {raceRunners.filter((r) => r.Status === 'DNS' || r.Status === 'DNF').length}
                  </Typography>
                </Tooltip>
                <Tooltip title={cp ? 'Active checkpoint for this race' : 'No checkpoint assigned — set one in Admin'}>
                  <Typography variant="body2" color={cp ? 'success.main' : 'error.main'}>
                    {cp ? `${cp.Code} – ${cp.DisplayName}` : 'No active CP'}
                  </Typography>
                </Tooltip>
              </Paper>
            )
          })}
        </Stack>
      )}

      <Stack direction={{ xs: 'column', md: 'row' }} spacing={3}>
        {/* ── Manual bib entry ── */}
        <Paper sx={{ p: 2, flex: 1 }}>
          <Typography variant="h6" gutterBottom>Log Bib</Typography>
          <Stack direction="row" spacing={1}>
            <TextField
              inputRef={bibRef}
              label="Bib #"
              value={bib}
              size="small"
              sx={{ width: 120 }}
              onChange={(e) => setBib(e.target.value)}
              onKeyDown={(e) => e.key === 'Enter' && submitBib()}
              error={!!bibError}
              helperText={bibError}
              autoFocus
              disabled={!hasActiveCheckpoint}
            />
            <Button
              variant="contained"
              onClick={submitBib}
              disabled={!bib || !hasActiveCheckpoint}
            >
              Log
            </Button>
          </Stack>
        </Paper>

        {/* ── DNS / DNF ── */}
        <Paper sx={{ p: 2, flex: 1 }}>
          <Typography variant="h6" gutterBottom>DNS / DNF</Typography>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start' }}>
            <TextField
              label="Bib #"
              value={statusBib}
              size="small"
              sx={{ width: 100 }}
              onChange={(e) => setStatusBib(e.target.value)}
              disabled={!hasActiveCheckpoint}
            />
            <FormControl size="small" sx={{ width: 90 }}>
              <InputLabel>Status</InputLabel>
              <Select
                value={status}
                label="Status"
                onChange={(e) => setStatus(e.target.value as 'DNS' | 'DNF')}
                disabled={!hasActiveCheckpoint}
              >
                <MenuItem value="DNS">DNS</MenuItem>
                <MenuItem value="DNF">DNF</MenuItem>
              </Select>
            </FormControl>
            <Button
              variant="outlined"
              onClick={submitStatus}
              disabled={!statusBib || !hasActiveCheckpoint}
            >
              Submit
            </Button>
          </Stack>
          {statusMsg && <Typography variant="body2" sx={{ mt: 1 }}>{statusMsg}</Typography>}
        </Paper>

        {/* ── Race transfer ── */}
        <Paper sx={{ p: 2, flex: 1 }}>
          <Typography variant="h6" gutterBottom>Transfer Runner</Typography>
          <Stack direction="row" spacing={1} sx={{ alignItems: 'flex-start' }}>
            <TextField
              label="Bib #"
              value={transferBib}
              size="small"
              sx={{ width: 100 }}
              onChange={(e) => setTransferBib(e.target.value)}
            />
            <FormControl size="small" sx={{ minWidth: 120 }}>
              <InputLabel>To Race</InputLabel>
              <Select
                value={transferRace}
                label="To Race"
                onChange={(e) => setTransferRace(Number(e.target.value))}
              >
                {races.map((r) => (
                  <MenuItem key={r.ID} value={r.ID}>{r.Name}</MenuItem>
                ))}
              </Select>
            </FormControl>
            <Button
              variant="outlined"
              onClick={submitTransfer}
              disabled={!transferBib || !transferRace}
            >
              Transfer
            </Button>
          </Stack>
          {transferMsg && <Typography variant="body2" sx={{ mt: 1 }}>{transferMsg}</Typography>}
        </Paper>
      </Stack>

      <Divider sx={{ my: 3 }} />

      {/* ── Recent log ── */}
      <Typography variant="h6" gutterBottom>Recent Log</Typography>
      {recentLogs.length === 0 ? (
        <Typography variant="body2" color="text.secondary">No entries yet.</Typography>
      ) : (
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>Bib</TableCell>
              <TableCell>Name</TableCell>
              <TableCell>Race</TableCell>
              <TableCell>Source</TableCell>
              <TableCell>Status</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {recentLogs.map((entry, i) => {
              const race = races.find((r) => r.ID === entry.runner.RaceID)
              return (
                <TableRow key={i} sx={{ bgcolor: entry.is_duplicate ? 'warning.dark' : undefined }}>
                  <TableCell>{entry.runner.BibNumber}</TableCell>
                  <TableCell>{entry.runner.FirstName} {entry.runner.LastName}</TableCell>
                  <TableCell>{race?.Name ?? `Race ${entry.runner.RaceID}`}</TableCell>
                  <TableCell>
                    <Chip
                      label={SOURCE_LABEL[entry.log.Source] ?? entry.log.Source}
                      size="small"
                      color={entry.log.Source === 'MESHTASTIC' ? 'primary' : 'default'}
                    />
                  </TableCell>
                  <TableCell>
                    {entry.is_duplicate && <Chip label="DUPLICATE" size="small" color="warning" />}
                  </TableCell>
                </TableRow>
              )
            })}
          </TableBody>
        </Table>
      )}
    </Box>
  )
}

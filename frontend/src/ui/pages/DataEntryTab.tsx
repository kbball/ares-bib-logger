import { useCallback, useEffect, useRef, useState } from 'react'
import {
  Alert, Box, Button, Chip, Divider, FormControl, InputLabel,
  MenuItem, Paper, Select, Stack, Table, TableBody, TableCell,
  TableHead, TableRow, TextField, Typography,
} from '@mui/material'
import type { ActiveSession, LogBibResult, Race, Runner } from '../../domain/types'
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

  useEffect(() => {
    loadSession()
  }, [loadSession])

  useEffect(() => {
    if (session?.EventID) {
      loadRaces(session.EventID)
    }
  }, [session?.EventID, loadRaces])

  useEffect(() => {
    if (races.length) loadRunners(races.map((r) => r.ID))
  }, [races, loadRunners])

  const pushLog = useCallback((result: LogBibResult) => {
    if (result.is_duplicate) {
      setDupAlert(`DUPLICATE: Bib ${result.runner.BibNumber} (${result.runner.FirstName} ${result.runner.LastName})`)
      setTimeout(() => setDupAlert(null), 8000)
    }
    setRecentLogs((prev) => [result, ...prev].slice(0, 50))
  }, [])

  useStream({
    onBibLogged: (payload) => pushLog(payload as LogBibResult),
    onSessionChanged: (payload) => setSession(payload as ActiveSession),
  })

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
      await api.transferRunner(runner.ID, Number(transferRace))
      setTransferMsg(`Bib ${n} transferred`)
      setTransferBib('')
      setTransferRace('')
      setError('')
    } catch (e: unknown) {
      setError((e as Error).message)
    }
  }

  const activeCheckpoint = (raceID: number) =>
    session?.Checkpoints?.find((c) => c.RaceID === raceID)?.CheckpointID

  return (
    <Box sx={{ maxWidth: 900 }}>
      <Typography variant="h5" gutterBottom>Data Entry</Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {dupAlert && (
        <Alert severity="warning" sx={{ mb: 2 }} onClose={() => setDupAlert(null)}>
          {dupAlert}
        </Alert>
      )}

      {/* ── Race stats ── */}
      {races.length > 0 && (
        <Stack direction="row" spacing={2} sx={{ mb: 2, flexWrap: 'wrap' }}>
          {races.map((race) => {
            const raceRunners = runners.filter((r) => r.RaceID === race.ID)
            const cpID = activeCheckpoint(race.ID)
            return (
              <Paper key={race.ID} sx={{ p: 1.5, minWidth: 160 }}>
                <Typography variant="subtitle2" sx={{ fontWeight: 'bold' }}>{race.Name}</Typography>
                <Typography variant="body2">Runners: {raceRunners.length}</Typography>
                <Typography variant="body2">
                  Active: {raceRunners.filter((r) => r.Status === 'ACTIVE').length}
                </Typography>
                <Typography variant="body2" color={cpID ? 'success.main' : 'error.main'}>
                  {cpID ? `CP #${cpID}` : 'No active CP'}
                </Typography>
              </Paper>
            )
          })}
        </Stack>
      )}

      {!session?.EventID && (
        <Alert severity="info" sx={{ mb: 2 }}>No active event. Set one in Admin.</Alert>
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
            />
            <Button variant="contained" onClick={submitBib} disabled={!bib}>
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
            />
            <FormControl size="small" sx={{ width: 90 }}>
              <InputLabel>Status</InputLabel>
              <Select
                value={status}
                label="Status"
                onChange={(e) => setStatus(e.target.value as 'DNS' | 'DNF')}
              >
                <MenuItem value="DNS">DNS</MenuItem>
                <MenuItem value="DNF">DNF</MenuItem>
              </Select>
            </FormControl>
            <Button variant="outlined" onClick={submitStatus} disabled={!statusBib}>
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
                <TableRow key={i} sx={{ bgcolor: entry.is_duplicate ? 'warning.light' : undefined }}>
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

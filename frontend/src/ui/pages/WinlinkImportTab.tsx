import { useEffect, useState } from 'react'
import {
  Alert, Box, Button, FormControl, InputLabel, MenuItem,
  Paper, Select, Stack, TextField, Tooltip, Typography,
  Table, TableHead, TableRow, TableCell, TableBody,
} from '@mui/material'
import type { ActiveSession, Checkpoint, Race, WinlinkImportResult } from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'

const SKIP_REASON: Record<string, string> = {
  blank: 'Blank line',
  no_runner: 'No runner at this position',
  parse_error: 'Could not parse time',
  moved: 'Runner transferred out of this race',
}

function skipLabel(d: { Reason: string }): string {
  return SKIP_REASON[d.Reason] ?? d.Reason
}

export default function WinlinkImportTab() {
  const [session, setSession] = useState<ActiveSession | null>(null)
  const [races, setRaces] = useState<Race[]>([])
  const [checkpointsByRace, setCheckpointsByRace] = useState<Record<number, Checkpoint[]>>({})

  const [raceID, setRaceID] = useState<number | ''>('')
  const [checkpointID, setCheckpointID] = useState<number | ''>('')
  const [text, setText] = useState('')
  const [result, setResult] = useState<WinlinkImportResult | null>(null)
  const [error, setError] = useState('')

  useEffect(() => {
    api.getSession().then(setSession).catch(() => {})
  }, [])

  useEffect(() => {
    if (session?.EventID) {
      api.listRaces(session.EventID).then(setRaces).catch(() => {})
    }
  }, [session?.EventID])

  useEffect(() => {
    if (!races.length) return
    Promise.all(races.map((r) => api.listCheckpoints(r.ID).then((cps) => [r.ID, cps] as [number, Checkpoint[]]))).then(
      (entries) => setCheckpointsByRace(Object.fromEntries(entries)),
    )
  }, [races])

  useStream({ onSessionChanged: (p) => setSession(p as ActiveSession) })

  const checkpoints = raceID ? (checkpointsByRace[raceID] ?? []) : []

  const submit = async () => {
    if (!raceID || !checkpointID || !text.trim()) return
    try {
      const r = await api.importWinlink(Number(raceID), Number(checkpointID), text)
      setResult(r)
      setError('')
    } catch (e: unknown) {
      setError((e as Error).message)
    }
  }

  return (
    <Box sx={{ maxWidth: 700 }}>
      <Typography variant="h5" gutterBottom>Winlink Import</Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {!session?.EventID && (
        <Alert severity="info" sx={{ mb: 2 }}>No active event. Set one in Admin.</Alert>
      )}

      <Stack spacing={2}>
        <Stack direction="row" spacing={2}>
          <FormControl size="small" sx={{ minWidth: 160 }}>
            <InputLabel id="import-race-label">Race</InputLabel>
            <Select value={raceID} label="Race" labelId="import-race-label" onChange={(e) => { setRaceID(Number(e.target.value)); setCheckpointID('') }}>
              {races.map((r) => (
                <MenuItem key={r.ID} value={r.ID}>{r.Name}</MenuItem>
              ))}
            </Select>
          </FormControl>

          <FormControl size="small" sx={{ minWidth: 200 }}>
            <InputLabel id="import-cp-label">Checkpoint</InputLabel>
            <Select
              value={checkpointID}
              label="Checkpoint"
              labelId="import-cp-label"
              disabled={!raceID}
              onChange={(e) => setCheckpointID(Number(e.target.value))}
            >
              {checkpoints
                .sort((a, b) => a.DisplayOrder - b.DisplayOrder)
                .map((cp) => (
                  <MenuItem key={cp.ID} value={cp.ID}>
                    {cp.Code} – {cp.DisplayName}
                  </MenuItem>
                ))}
            </Select>
          </FormControl>
        </Stack>

        <TextField
          multiline
          rows={10}
          size="small"
          label="Paste Winlink column"
          placeholder="Paste the exported Winlink column here…"
          value={text}
          onChange={(e) => setText(e.target.value)}
          sx={{ fontFamily: 'monospace' }}
        />

        <Box>
          <Tooltip title="Parse column by row position and import checkpoint times">
            <span>
              <Button
                variant="contained"
                onClick={submit}
                disabled={!raceID || !checkpointID || !text.trim()}
              >
                Import
              </Button>
            </span>
          </Tooltip>
        </Box>

        {result && (
          <Paper sx={{ p: 2 }}>
            <Typography variant="subtitle1" gutterBottom>Import Summary</Typography>
            <Typography>Created: {result.Created}</Typography>
            <Typography>Updated: {result.Updated}</Typography>
            <Typography>Skipped: {result.Skipped}</Typography>
            {result.SkippedDetails?.length > 0 && (
              <>
                <Typography variant="body2" sx={{ mt: 1, mb: 0.5 }}>Skipped details:</Typography>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Position</TableCell>
                      <TableCell>Bib</TableCell>
                      <TableCell>Reason</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {result.SkippedDetails.map((d, i) => (
                      <TableRow key={i}>
                        <TableCell>{d.Position}</TableCell>
                        <TableCell>{d.BibNumber || '—'}</TableCell>
                        <TableCell>{skipLabel(d)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </>
            )}
            {result.Errors?.length > 0 && (
              <>
                <Typography color="error" sx={{ mt: 1 }}>Errors:</Typography>
                <Table size="small">
                  <TableHead>
                    <TableRow><TableCell>Message</TableCell></TableRow>
                  </TableHead>
                  <TableBody>
                    {result.Errors.map((e, i) => (
                      <TableRow key={i}><TableCell>{e}</TableCell></TableRow>
                    ))}
                  </TableBody>
                </Table>
              </>
            )}
          </Paper>
        )}
      </Stack>
    </Box>
  )
}

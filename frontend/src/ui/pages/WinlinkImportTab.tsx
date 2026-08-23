import { useEffect, useState } from 'react'
import {
  Alert,
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  FormControl,
  InputLabel,
  MenuItem,
  Paper,
  Select,
  Stack,
  TextField,
  Tooltip,
  Typography,
  Table,
  TableHead,
  TableRow,
  TableCell,
  TableBody,
} from '@mui/material'
import type {
  ActiveSession,
  Checkpoint,
  Race,
  WinlinkImportResult,
  WinlinkPreviewResult,
} from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'

const SKIP_REASON: Record<string, string> = {
  blank: 'Blank line',
  no_runner: 'No runner at this position',
  parse_error: 'Could not parse time',
  moved: 'Runner transferred out of this race',
}

function skipLabel(reason: string): string {
  return SKIP_REASON[reason] ?? reason
}

const KIND_LABEL: Record<string, string> = {
  create: 'Create',
  update: 'Update',
  skip: 'Skip',
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
  const [pendingPreview, setPendingPreview] = useState<WinlinkPreviewResult | null>(null)

  useEffect(() => {
    api
      .getSession()
      .then(setSession)
      .catch(() => {})
  }, [])

  useEffect(() => {
    if (session?.EventID) {
      api
        .listRaces(session.EventID)
        .then(setRaces)
        .catch(() => {})
    }
  }, [session?.EventID])

  useEffect(() => {
    if (!races.length) return
    Promise.all(
      races.map((r) =>
        api.listCheckpoints(r.ID).then((cps) => [r.ID, cps] as [number, Checkpoint[]]),
      ),
    ).then((entries) => setCheckpointsByRace(Object.fromEntries(entries)))
  }, [races])

  useStream({ onSessionChanged: (p) => setSession(p as ActiveSession) })

  const activeCheckpointID = session?.Checkpoints?.find(
    (c) => c.RaceID === Number(raceID),
  )?.CheckpointID
  const checkpoints = raceID
    ? (checkpointsByRace[raceID] ?? []).filter((cp) => cp.ID !== activeCheckpointID)
    : []

  const doImport = async () => {
    const r = await api.importWinlink(Number(raceID), Number(checkpointID), text)
    setResult(r)
    setError('')
    setPendingPreview(null)
  }

  const submit = async () => {
    if (!raceID || !checkpointID || !text.trim()) return
    try {
      const preview = await api.previewWinlink(Number(raceID), Number(checkpointID), text)
      if (preview.Skipped === 0 && !preview.HeaderMismatch) {
        await doImport()
      } else {
        setPendingPreview(preview)
      }
    } catch (e: unknown) {
      setError((e as Error).message)
    }
  }

  const confirmImport = async () => {
    try {
      await doImport()
    } catch (e: unknown) {
      setError((e as Error).message)
      setPendingPreview(null)
    }
  }

  return (
    <Box sx={{ maxWidth: 700 }}>
      <Typography variant="h5" gutterBottom>
        Winlink Import
      </Typography>

      {error && (
        <Alert severity="error" sx={{ mb: 2 }}>
          {error}
        </Alert>
      )}
      {!session?.EventID && (
        <Alert severity="info" sx={{ mb: 2 }}>
          No active event. Set one in Admin.
        </Alert>
      )}

      <Stack spacing={2}>
        <Stack direction="row" spacing={2}>
          <FormControl size="small" sx={{ minWidth: 160 }}>
            <InputLabel id="import-race-label">Race</InputLabel>
            <Select
              value={raceID}
              label="Race"
              labelId="import-race-label"
              onChange={(e) => {
                setRaceID(Number(e.target.value))
                setCheckpointID('')
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
            <Typography variant="subtitle1" gutterBottom>
              Import Summary
            </Typography>
            <Typography>Created: {result.Created}</Typography>
            <Typography>Updated: {result.Updated}</Typography>
            <Typography>Skipped: {result.Skipped}</Typography>
            {result.SkippedDetails?.length > 0 && (
              <>
                <Typography variant="body2" sx={{ mt: 1, mb: 0.5 }}>
                  Skipped details:
                </Typography>
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
                        <TableCell>{skipLabel(d.Reason)}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </>
            )}
            {result.Errors?.length > 0 && (
              <>
                <Typography color="error" sx={{ mt: 1 }}>
                  Errors:
                </Typography>
                <Table size="small">
                  <TableHead>
                    <TableRow>
                      <TableCell>Message</TableCell>
                    </TableRow>
                  </TableHead>
                  <TableBody>
                    {result.Errors.map((e, i) => (
                      <TableRow key={i}>
                        <TableCell>{e}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </>
            )}
          </Paper>
        )}
      </Stack>

      <Dialog
        open={!!pendingPreview}
        onClose={() => setPendingPreview(null)}
        maxWidth="sm"
        fullWidth
      >
        <DialogTitle>Confirm Winlink Import</DialogTitle>
        <DialogContent>
          {pendingPreview?.HeaderMismatch && (
            <Alert severity="warning" sx={{ mb: 2 }}>
              Pasted header &quot;{pendingPreview.PastedHeader}&quot; doesn&apos;t match the
              expected header &quot;{pendingPreview.ExpectedHeader}&quot; for this checkpoint.
              Double-check the race/checkpoint selection before importing.
            </Alert>
          )}
          <DialogContentText sx={{ mb: 2 }}>
            {pendingPreview?.Skipped} of {pendingPreview?.Rows.length} rows will be skipped. Review
            the full breakdown below before importing.
          </DialogContentText>
          <Table size="small">
            <TableHead>
              <TableRow>
                <TableCell>Position</TableCell>
                <TableCell>Bib</TableCell>
                <TableCell>Action</TableCell>
                <TableCell>Value / Reason</TableCell>
              </TableRow>
            </TableHead>
            <TableBody>
              {pendingPreview?.Rows.map((row, i) => (
                <TableRow key={i}>
                  <TableCell>{row.Position}</TableCell>
                  <TableCell>{row.BibNumber || '—'}</TableCell>
                  <TableCell>{KIND_LABEL[row.Kind] ?? row.Kind}</TableCell>
                  <TableCell>{row.Kind === 'skip' ? skipLabel(row.Reason) : row.Value}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPendingPreview(null)}>Cancel</Button>
          <Button color="warning" variant="contained" onClick={confirmImport}>
            Confirm &amp; Import
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

import { useEffect, useState } from 'react'
import {
  Alert, Box, Button, FormControl, InputLabel, MenuItem,
  Select, Stack, TextField, Tooltip, Typography,
} from '@mui/material'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import type { ActiveSession, Checkpoint, Race } from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'

function currentHHMM(): string {
  const now = new Date()
  return `${String(now.getHours()).padStart(2, '0')}:${String(now.getMinutes()).padStart(2, '0')}`
}

export default function WinlinkExportTab() {
  const [session, setSession] = useState<ActiveSession | null>(null)
  const [races, setRaces] = useState<Race[]>([])
  const [raceID, setRaceID] = useState<number | ''>('')
  const [checkpoints, setCheckpoints] = useState<Checkpoint[]>([])
  const [column, setColumn] = useState('')
  const [subject, setSubject] = useState('')
  const [copied, setCopied] = useState(false)
  const [subjectCopied, setSubjectCopied] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.getSession().then(setSession).catch(() => {})
  }, [])

  useEffect(() => {
    if (!session?.EventID) return
    const eventID = session.EventID
    const sessionCheckpoints = session.Checkpoints
    api.listRaces(eventID).then((r) => {
      setRaces(r)
      const activeRaceID = sessionCheckpoints?.[0]?.RaceID
      if (activeRaceID && r.find((race) => race.ID === activeRaceID)) {
        setRaceID(activeRaceID)
      }
    }).catch(() => {})
  }, [session?.EventID, session?.Checkpoints])

  useEffect(() => {
    if (!raceID) { setCheckpoints([]); return }
    api.listCheckpoints(Number(raceID)).then(setCheckpoints).catch(() => {})
  }, [raceID])

  useStream({
    onSessionChanged: (p) => {
      const s = p as ActiveSession
      setSession(s)
    },
  })

  const buildSubject = () => {
    const race = races.find((r) => r.ID === Number(raceID))
    const sessionCp = session?.Checkpoints.find((c) => c.RaceID === Number(raceID))
    const cp = checkpoints.find((c) => c.ID === sessionCp?.CheckpointID)
    return [cp?.DisplayName, race?.Name, currentHHMM(), 'update']
      .filter(Boolean)
      .join(' ')
  }

  const generate = async () => {
    if (!raceID) return
    try {
      const text = await api.exportWinlink(Number(raceID))
      setColumn(text)
      setSubject(buildSubject())
      setError('')
    } catch (e: unknown) {
      setError((e as Error).message)
    }
  }

  const copySubject = () => {
    const fresh = buildSubject()
    navigator.clipboard.writeText(fresh).then(() => {
      setSubject(fresh)
      setSubjectCopied(true)
      setTimeout(() => setSubjectCopied(false), 2000)
    })
  }

  const copy = () => {
    navigator.clipboard.writeText(column).then(() => {
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    })
  }

  return (
    <Box sx={{ maxWidth: 700 }}>
      <Typography variant="h5" gutterBottom>Winlink Export</Typography>

      {error && <Alert severity="error" sx={{ mb: 2 }}>{error}</Alert>}
      {!session?.EventID && (
        <Alert severity="info" sx={{ mb: 2 }}>No active event. Set one in Admin.</Alert>
      )}

      <Stack spacing={2}>
        <Stack direction="row" spacing={2} sx={{ alignItems: 'center' }}>
          <FormControl size="small" sx={{ minWidth: 180 }}>
            <InputLabel id="export-race-label">Race</InputLabel>
            <Select value={raceID} label="Race" labelId="export-race-label" onChange={(e) => { setRaceID(Number(e.target.value)); setColumn(''); setSubject('') }}>
              {races.map((r) => (
                <MenuItem key={r.ID} value={r.ID}>{r.Name}</MenuItem>
              ))}
            </Select>
          </FormControl>
          <Tooltip title="Generate Winlink export column for the selected race">
            <Button variant="contained" onClick={generate} disabled={!raceID}>
              Generate
            </Button>
          </Tooltip>
        </Stack>

        {column && (
          <>
            <Typography variant="subtitle2" color="text.secondary">Email Subject</Typography>
            <Stack direction="row" spacing={1} sx={{ alignItems: 'center' }}>
              <TextField
                value={subject}
                slotProps={{ input: { readOnly: true }, htmlInput: { 'aria-label': 'Email subject' } }}
                size="small"
                fullWidth
              />
              <Tooltip title="Copy email subject to clipboard" describeChild>
                <Button
                  variant="outlined"
                  startIcon={<ContentCopyIcon />}
                  onClick={copySubject}
                  sx={{ whiteSpace: 'nowrap' }}
                >
                  {subjectCopied ? 'Copied!' : 'Copy Subject'}
                </Button>
              </Tooltip>
            </Stack>

            <Typography variant="subtitle2" color="text.secondary">Column</Typography>
            <TextField
              multiline
              rows={20}
              value={column}
              slotProps={{ input: { readOnly: true } }}
              sx={{ fontFamily: 'monospace', fontSize: 13 }}
              size="small"
            />
            <Box>
              <Tooltip title="Copy column to clipboard" describeChild>
                <Button
                  variant="outlined"
                  startIcon={<ContentCopyIcon />}
                  onClick={copy}
                >
                  {copied ? 'Copied!' : 'Copy to Clipboard'}
                </Button>
              </Tooltip>
            </Box>
          </>
        )}
      </Stack>
    </Box>
  )
}

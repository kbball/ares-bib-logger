import { useEffect, useState } from 'react'
import {
  Alert, Box, Button, FormControl, InputLabel, MenuItem,
  Select, Stack, TextField, Typography,
} from '@mui/material'
import ContentCopyIcon from '@mui/icons-material/ContentCopy'
import type { ActiveSession, Race } from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'

export default function WinlinkExportTab() {
  const [session, setSession] = useState<ActiveSession | null>(null)
  const [races, setRaces] = useState<Race[]>([])
  const [raceID, setRaceID] = useState<number | ''>('')
  const [column, setColumn] = useState('')
  const [copied, setCopied] = useState(false)
  const [error, setError] = useState('')

  useEffect(() => {
    api.getSession().then(setSession).catch(() => {})
  }, [])

  useEffect(() => {
    if (!session?.EventID) return
    const eventID = session.EventID
    const checkpoints = session.Checkpoints
    api.listRaces(eventID).then((r) => {
      setRaces(r)
      const activeRaceID = checkpoints?.[0]?.RaceID
      if (activeRaceID && r.find((race) => race.ID === activeRaceID)) {
        setRaceID(activeRaceID)
      }
    }).catch(() => {})
  }, [session?.EventID, session?.Checkpoints])

  useStream({
    onSessionChanged: (p) => {
      const s = p as ActiveSession
      setSession(s)
    },
  })

  const generate = async () => {
    if (!raceID) return
    try {
      const text = await api.exportWinlink(Number(raceID))
      setColumn(text)
      setError('')
    } catch (e: unknown) {
      setError((e as Error).message)
    }
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
            <InputLabel>Race</InputLabel>
            <Select value={raceID} label="Race" onChange={(e) => { setRaceID(Number(e.target.value)); setColumn('') }}>
              {races.map((r) => (
                <MenuItem key={r.ID} value={r.ID}>{r.Name}</MenuItem>
              ))}
            </Select>
          </FormControl>
          <Button variant="contained" onClick={generate} disabled={!raceID}>
            Generate
          </Button>
        </Stack>

        {column && (
          <>
            <TextField
              multiline
              rows={20}
              value={column}
              slotProps={{ input: { readOnly: true } }}
              sx={{ fontFamily: 'monospace', fontSize: 13 }}
              size="small"
            />
            <Box>
              <Button
                variant="outlined"
                startIcon={<ContentCopyIcon />}
                onClick={copy}
              >
                {copied ? 'Copied!' : 'Copy to Clipboard'}
              </Button>
            </Box>
          </>
        )}
      </Stack>
    </Box>
  )
}

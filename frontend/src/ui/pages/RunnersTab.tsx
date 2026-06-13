import { useEffect, useMemo, useState } from 'react'
import {
  Alert, Box, Chip, FormControl, InputLabel, MenuItem,
  Select, Stack, Table, TableBody, TableCell, TableHead,
  TableRow, TextField, Typography,
} from '@mui/material'
import type { ActiveSession, Checkpoint, Race, Runner } from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'

const STATUS_COLOR: Record<string, 'default' | 'success' | 'error' | 'warning' | 'info'> = {
  ACTIVE: 'success',
  DNS: 'error',
  DNF: 'warning',
  FINISHED: 'info',
  MOVED: 'default',
  UNKNOWN: 'default',
}

export default function RunnersTab() {
  const [session, setSession] = useState<ActiveSession | null>(null)
  const [races, setRaces] = useState<Race[]>([])
  const [allRunners, setAllRunners] = useState<Runner[]>([])
  const [checkpointsByRace, setCheckpointsByRace] = useState<Record<number, Checkpoint[]>>({})
  const [filterRaceID, setFilterRaceID] = useState<number | ''>('')
  const [search, setSearch] = useState('')

  useEffect(() => {
    api.getSession().then(setSession).catch(() => {})
  }, [])

  useEffect(() => {
    if (!session?.EventID) return
    api.listRaces(session.EventID).then(setRaces).catch(() => {})
  }, [session?.EventID])

  useEffect(() => {
    if (!races.length) return
    Promise.all(races.map((r) => api.listRunners(r.ID))).then((arr) =>
      setAllRunners(arr.flat()),
    )
    Promise.all(
      races.map((r) => api.listCheckpoints(r.ID).then((cps) => [r.ID, cps] as [number, Checkpoint[]])),
    ).then((entries) => setCheckpointsByRace(Object.fromEntries(entries)))
  }, [races])

  useStream({
    onSessionChanged: (p) => setSession(p as ActiveSession),
    onBibLogged: () => {
      // Refresh runner list on any bib event to pick up status changes
      if (races.length) {
        Promise.all(races.map((r) => api.listRunners(r.ID))).then((arr) =>
          setAllRunners(arr.flat()),
        )
      }
    },
  })

  const visibleRace = filterRaceID ? races.find((r) => r.ID === filterRaceID) : null
  const checkpoints = visibleRace
    ? (checkpointsByRace[visibleRace.ID] ?? []).sort((a, b) => a.DisplayOrder - b.DisplayOrder)
    : []

  const filtered = useMemo(() => {
    let runners = filterRaceID
      ? allRunners.filter((r) => r.RaceID === filterRaceID)
      : allRunners

    const q = search.toLowerCase().trim()
    if (q) {
      runners = runners.filter(
        (r) =>
          String(r.BibNumber).includes(q) ||
          r.FirstName.toLowerCase().includes(q) ||
          r.LastName.toLowerCase().includes(q),
      )
    }

    return [...runners].sort((a, b) => a.SortOrder - b.SortOrder)
  }, [allRunners, filterRaceID, search])

  const raceForRunner = (r: Runner) => races.find((rc) => rc.ID === r.RaceID)

  return (
    <Box>
      <Typography variant="h5" gutterBottom>Runners</Typography>

      {!session?.EventID && (
        <Alert severity="info" sx={{ mb: 2 }}>No active event. Set one in Admin.</Alert>
      )}

      <Stack direction="row" spacing={2} sx={{ mb: 2 }}>
        <TextField
          size="small"
          label="Search bib / name"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          sx={{ width: 220 }}
        />
        <FormControl size="small" sx={{ minWidth: 160 }}>
          <InputLabel>Race</InputLabel>
          <Select
            value={filterRaceID}
            label="Race"
            onChange={(e) => setFilterRaceID(e.target.value as number | '')}
          >
            <MenuItem value="">All races</MenuItem>
            {races.map((r) => (
              <MenuItem key={r.ID} value={r.ID}>{r.Name}</MenuItem>
            ))}
          </Select>
        </FormControl>
      </Stack>

      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {filtered.length} runner{filtered.length !== 1 ? 's' : ''}
      </Typography>

      <Box sx={{ overflowX: 'auto' }}>
        <Table size="small" stickyHeader>
          <TableHead>
            <TableRow>
              <TableCell>Bib</TableCell>
              <TableCell>Name</TableCell>
              {!filterRaceID && <TableCell>Race</TableCell>}
              <TableCell>Status</TableCell>
              {checkpoints.map((cp) => (
                <TableCell key={cp.ID}>{cp.Code}</TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {filtered.map((runner) => (
              <TableRow key={runner.ID} hover>
                <TableCell>{runner.BibNumber}</TableCell>
                <TableCell>{runner.FirstName} {runner.LastName}</TableCell>
                {!filterRaceID && (
                  <TableCell>{raceForRunner(runner)?.Name ?? `Race ${runner.RaceID}`}</TableCell>
                )}
                <TableCell>
                  <Chip
                    label={runner.Status}
                    size="small"
                    color={STATUS_COLOR[runner.Status] ?? 'default'}
                  />
                </TableCell>
                {checkpoints.map((cp) => (
                  <TableCell key={cp.ID}>—</TableCell>
                ))}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Box>
    </Box>
  )
}

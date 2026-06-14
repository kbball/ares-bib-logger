import { useEffect, useMemo, useState } from 'react'
import {
  Alert, Box, Chip, Stack, Tab, Table, TableBody, TableCell, TableHead,
  TableRow, TableSortLabel, Tabs, TextField, Typography,
} from '@mui/material'
import type { ActiveSession, Checkpoint, CheckpointLog, Race, Runner } from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'

const STATUS_COLOR: Record<string, 'default' | 'success' | 'error' | 'warning' | 'info'> = {
  ACTIVE: 'success',
  DNS: 'error',
  DNF: 'warning',
  FINISHED: 'info',
  MOVED: 'warning',
  UNKNOWN: 'default',
}

type SortKey = 'BibNumber' | 'Name' | 'Status' | 'SortOrder'
type SortDir = 'asc' | 'desc'

export default function RunnersTab() {
  const [session, setSession] = useState<ActiveSession | null>(null)
  const [races, setRaces] = useState<Race[]>([])
  const [allRunners, setAllRunners] = useState<Runner[]>([])
  const [checkpointsByRace, setCheckpointsByRace] = useState<Record<number, Checkpoint[]>>({})
  const [logsByRace, setLogsByRace] = useState<Record<number, CheckpointLog[]>>({})
  const [filterRaceID, setFilterRaceID] = useState<number | ''>('')
  const [search, setSearch] = useState('')
  const [sortKey, setSortKey] = useState<SortKey>('BibNumber')
  const [sortDir, setSortDir] = useState<SortDir>('asc')

  const loadRunners = (raceIDs: number[]) =>
    Promise.all(raceIDs.map((id) => api.listRunners(id))).then((arr) =>
      setAllRunners(arr.flat()),
    )

  const loadLogs = (raceIDs: number[]) =>
    Promise.all(raceIDs.map((id) => api.listCheckpointLogs(id).then((logs) => [id, logs] as [number, CheckpointLog[]]))).then(
      (entries) => setLogsByRace(Object.fromEntries(entries)),
    )

  useEffect(() => {
    api.getSession().then(setSession).catch(() => {})
  }, [])

  useEffect(() => {
    if (!session?.EventID) return
    api.listRaces(session.EventID).then(setRaces).catch(() => {})
  }, [session?.EventID])

  useEffect(() => {
    if (!races.length) return
    const ids = races.map((r) => r.ID)
    loadRunners(ids)
    Promise.all(
      races.map((r) => api.listCheckpoints(r.ID).then((cps) => [r.ID, cps] as [number, Checkpoint[]])),
    ).then((entries) => setCheckpointsByRace(Object.fromEntries(entries)))
    loadLogs(ids)
  }, [races])

  useStream({
    onSessionChanged: (p) => setSession(p as ActiveSession),
    onBibLogged: () => {
      if (races.length) {
        const ids = races.map((r) => r.ID)
        loadRunners(ids)
        loadLogs(ids)
      }
    },
  })

  const handleSort = (key: SortKey) => {
    if (sortKey === key) {
      setSortDir((d) => (d === 'asc' ? 'desc' : 'asc'))
    } else {
      setSortKey(key)
      setSortDir('asc')
    }
  }

  // Which race IDs have at least one runner matching the current search query
  const racesWithMatches = useMemo(() => {
    const q = search.toLowerCase().trim()
    if (!q) return new Set(races.map((r) => r.ID))
    return new Set(
      allRunners
        .filter(
          (r) =>
            String(r.BibNumber).includes(q) ||
            r.FirstName.toLowerCase().includes(q) ||
            r.LastName.toLowerCase().includes(q),
        )
        .map((r) => r.RaceID),
    )
  }, [races, allRunners, search])

  // If the selected tab's race has no matches, fall back to All
  useEffect(() => {
    if (filterRaceID !== '' && !racesWithMatches.has(filterRaceID as number)) {
      setFilterRaceID('')
    }
  }, [racesWithMatches, filterRaceID])

  const visibleRace = filterRaceID ? races.find((r) => r.ID === filterRaceID) : null
  const checkpoints = visibleRace
    ? (checkpointsByRace[visibleRace.ID] ?? []).sort((a, b) => a.DisplayOrder - b.DisplayOrder)
    : []

  const logMap = useMemo(() => {
    const m = new Map<string, CheckpointLog>()
    Object.values(logsByRace).flat().forEach((log) => {
      m.set(`${log.RunnerID}-${log.CheckpointID}`, log)
    })
    return m
  }, [logsByRace])

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

    return [...runners].sort((a, b) => {
      let cmp = 0
      if (sortKey === 'BibNumber' || sortKey === 'SortOrder') {
        cmp = a[sortKey] - b[sortKey]
      } else if (sortKey === 'Name') {
        cmp = `${a.LastName} ${a.FirstName}`.localeCompare(`${b.LastName} ${b.FirstName}`)
      } else if (sortKey === 'Status') {
        cmp = a.Status.localeCompare(b.Status)
      }
      return sortDir === 'asc' ? cmp : -cmp
    })
  }, [allRunners, filterRaceID, search, sortKey, sortDir])

  const raceForRunner = (r: Runner) => races.find((rc) => rc.ID === r.RaceID)

  const formatLogCell = (log: CheckpointLog | undefined) => {
    if (!log) return '—'
    const raw = log.RawMessage?.toUpperCase()
    if (raw === 'DNS' || raw === 'DNF') return raw
    return new Date(log.RecordedAt).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', hour12: false })
  }

  const col = (label: string, key: SortKey) => (
    <TableCell sx={{ fontWeight: 'bold', border: 1, borderColor: 'divider' }}>
      <TableSortLabel
        active={sortKey === key}
        direction={sortKey === key ? sortDir : 'asc'}
        onClick={() => handleSort(key)}
      >
        {label}
      </TableSortLabel>
    </TableCell>
  )

  // Visible race tabs — only show races with search matches (all visible when not searching)
  const visibleRaces = races.filter((r) => racesWithMatches.has(r.ID))

  // Count matching runners per race (only used when a search is active)
  const matchCountFor = (raceID: number | '') => {
    const q = search.toLowerCase().trim()
    if (!q) return null
    const pool = raceID === '' ? allRunners : allRunners.filter((r) => r.RaceID === raceID)
    return pool.filter(
      (r) =>
        String(r.BibNumber).includes(q) ||
        r.FirstName.toLowerCase().includes(q) ||
        r.LastName.toLowerCase().includes(q),
    ).length
  }

  return (
    <Box>
      <Typography variant="h5" gutterBottom>Runners</Typography>

      {!session?.EventID && (
        <Alert severity="info" sx={{ mb: 2 }}>No active event. Set one in Admin.</Alert>
      )}

      <Stack direction="row" spacing={2} sx={{ mb: 1 }}>
        <TextField
          size="small"
          label="Search bib / name"
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          sx={{ width: 220 }}
        />
      </Stack>

      <Tabs
        value={filterRaceID}
        onChange={(_, v) => setFilterRaceID(v as number | '')}
        variant="scrollable"
        scrollButtons="auto"
        sx={{ mb: 1, borderBottom: 1, borderColor: 'divider' }}
      >
        <Tab
          label={matchCountFor('') !== null ? `All (${matchCountFor('')})` : 'All'}
          value=""
        />
        {visibleRaces.map((race) => {
          const count = matchCountFor(race.ID)
          return (
            <Tab
              key={race.ID}
              label={count !== null ? `${race.Name} (${count})` : race.Name}
              value={race.ID}
            />
          )
        })}
      </Tabs>

      <Typography variant="body2" color="text.secondary" sx={{ mb: 1 }}>
        {filtered.length} runner{filtered.length !== 1 ? 's' : ''}
      </Typography>

      <Box sx={{ overflowX: 'auto' }}>
        <Table size="small" stickyHeader sx={{ borderCollapse: 'collapse' }}>
          <TableHead>
            <TableRow>
              {col('Bib', 'BibNumber')}
              {col('Name', 'Name')}
              {!filterRaceID && (
                <TableCell sx={{ fontWeight: 'bold', border: 1, borderColor: 'divider' }}>Race</TableCell>
              )}
              {col('Status', 'Status')}
              {checkpoints.map((cp) => (
                <TableCell key={cp.ID} sx={{ fontWeight: 'bold', border: 1, borderColor: 'divider' }}>
                  {cp.DisplayName}
                </TableCell>
              ))}
            </TableRow>
          </TableHead>
          <TableBody>
            {filtered.map((runner) => (
              <TableRow key={runner.ID} hover>
                <TableCell sx={{ border: 1, borderColor: 'divider' }}>{runner.BibNumber}</TableCell>
                <TableCell sx={{ border: 1, borderColor: 'divider' }}>{runner.FirstName} {runner.LastName}</TableCell>
                {!filterRaceID && (
                  <TableCell sx={{ border: 1, borderColor: 'divider' }}>
                    {raceForRunner(runner)?.Name ?? `Race ${runner.RaceID}`}
                  </TableCell>
                )}
                <TableCell sx={{ border: 1, borderColor: 'divider' }}>
                  <Chip
                    label={runner.Status}
                    size="small"
                    color={STATUS_COLOR[runner.Status] ?? 'default'}
                  />
                </TableCell>
                {checkpoints.map((cp) => {
                  const log = logMap.get(`${runner.ID}-${cp.ID}`)
                  return (
                    <TableCell key={cp.ID} sx={{ border: 1, borderColor: 'divider', fontFamily: 'monospace' }}>
                      {formatLogCell(log)}
                    </TableCell>
                  )
                })}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Box>
    </Box>
  )
}

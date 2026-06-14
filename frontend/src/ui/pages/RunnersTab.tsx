import { useEffect, useMemo, useState } from 'react'
import {
  Alert,
  Box,
  Chip,
  Dialog,
  DialogContent,
  DialogTitle,
  Divider,
  Stack,
  Tab,
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableRow,
  TableSortLabel,
  Tabs,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
import type { ActiveSession, Checkpoint, CheckpointLog, Race, Runner } from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'
import { computeRunnerPace, projectArrival, formatPace } from '../../domain/pace'

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
  const [selectedRunner, setSelectedRunner] = useState<Runner | null>(null)

  const loadRunners = (raceIDs: number[]) =>
    Promise.all(raceIDs.map((id) => api.listRunners(id))).then((arr) => setAllRunners(arr.flat()))

  const loadLogs = (raceIDs: number[]) =>
    Promise.all(
      raceIDs.map((id) =>
        api.listCheckpointLogs(id).then((logs) => [id, logs] as [number, CheckpointLog[]]),
      ),
    ).then((entries) => setLogsByRace(Object.fromEntries(entries)))

  useEffect(() => {
    api
      .getSession()
      .then(setSession)
      .catch(() => {})
  }, [])

  useEffect(() => {
    if (!session?.EventID) return
    api
      .listRaces(session.EventID)
      .then(setRaces)
      .catch(() => {})
  }, [session?.EventID])

  useEffect(() => {
    if (!races.length) return
    const ids = races.map((r) => r.ID)
    loadRunners(ids)
    Promise.all(
      races.map((r) =>
        api.listCheckpoints(r.ID).then((cps) => [r.ID, cps] as [number, Checkpoint[]]),
      ),
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
    Object.values(logsByRace)
      .flat()
      .forEach((log) => {
        m.set(`${log.RunnerID}-${log.CheckpointID}`, log)
      })
    return m
  }, [logsByRace])

  const filtered = useMemo(() => {
    let runners = filterRaceID ? allRunners.filter((r) => r.RaceID === filterRaceID) : allRunners

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

  // Show pace columns only when a single race is selected and ≥2 checkpoints have distances
  const showPace = useMemo(() => {
    if (!filterRaceID) return false
    const cps = checkpointsByRace[filterRaceID as number] ?? []
    return cps.filter((cp) => cp.DistanceFromStart !== null).length >= 2
  }, [filterRaceID, checkpointsByRace])

  // Per-runner pace map (only computed when showPace)
  const paceMap = useMemo(() => {
    if (!showPace || !filterRaceID) return new Map<number, ReturnType<typeof computeRunnerPace>>()
    const cps = checkpointsByRace[filterRaceID as number] ?? []
    const allLogs = Object.values(logsByRace).flat()
    const m = new Map<number, ReturnType<typeof computeRunnerPace>>()
    filtered.forEach((runner) => {
      m.set(runner.ID, computeRunnerPace(runner, cps, allLogs))
    })
    return m
  }, [showPace, filterRaceID, checkpointsByRace, logsByRace, filtered])

  // Next unlogged checkpoint per runner (for projection target)
  const nextCPMap = useMemo(() => {
    if (!showPace || !filterRaceID) return new Map<number, Checkpoint | null>()
    const cps = [...(checkpointsByRace[filterRaceID as number] ?? [])].sort(
      (a, b) => a.DisplayOrder - b.DisplayOrder,
    )
    const allLogs = Object.values(logsByRace).flat()
    const m = new Map<number, Checkpoint | null>()
    filtered.forEach((runner) => {
      const loggedIDs = new Set(
        allLogs.filter((l) => l.RunnerID === runner.ID).map((l) => l.CheckpointID),
      )
      m.set(runner.ID, cps.find((cp) => !loggedIDs.has(cp.ID)) ?? null)
    })
    return m
  }, [showPace, filterRaceID, checkpointsByRace, logsByRace, filtered])

  const raceForRunner = (r: Runner) => races.find((rc) => rc.ID === r.RaceID)

  const formatLogCell = (log: CheckpointLog | undefined) => {
    if (!log) return '—'
    const raw = log.RawMessage?.toUpperCase()
    if (raw === 'DNS' || raw === 'DNF') return raw
    return new Date(log.RecordedAt).toLocaleTimeString([], {
      hour: '2-digit',
      minute: '2-digit',
      hour12: false,
    })
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
      <Typography variant="h5" gutterBottom>
        Runners
      </Typography>

      {!session?.EventID && (
        <Alert severity="info" sx={{ mb: 2 }}>
          No active event. Set one in Admin.
        </Alert>
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
        <Tab label={matchCountFor('') !== null ? `All (${matchCountFor('')})` : 'All'} value="" />
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
                <TableCell sx={{ fontWeight: 'bold', border: 1, borderColor: 'divider' }}>
                  Race
                </TableCell>
              )}
              {col('Status', 'Status')}
              {checkpoints.map((cp) => (
                <TableCell
                  key={cp.ID}
                  sx={{ fontWeight: 'bold', border: 1, borderColor: 'divider' }}
                >
                  {cp.DisplayName}
                  {cp.DistanceFromStart != null && (
                    <Typography variant="caption" color="text.secondary" sx={{ ml: 0.5 }}>
                      {cp.DistanceFromStart}mi
                    </Typography>
                  )}
                </TableCell>
              ))}
              {showPace && (
                <>
                  <TableCell sx={{ fontWeight: 'bold', border: 1, borderColor: 'divider' }}>
                    Pace
                  </TableCell>
                  <TableCell sx={{ fontWeight: 'bold', border: 1, borderColor: 'divider' }}>
                    <Tooltip title="Projected arrival at next checkpoint based on current pace">
                      <span>Proj. Next</span>
                    </Tooltip>
                  </TableCell>
                </>
              )}
            </TableRow>
          </TableHead>
          <TableBody>
            {filtered.map((runner) => (
              <TableRow
                key={runner.ID}
                hover
                onClick={() => setSelectedRunner(runner)}
                sx={{ cursor: 'pointer' }}
              >
                <TableCell sx={{ border: 1, borderColor: 'divider' }}>{runner.BibNumber}</TableCell>
                <TableCell sx={{ border: 1, borderColor: 'divider' }}>
                  {runner.FirstName} {runner.LastName}
                </TableCell>
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
                    <TableCell
                      key={cp.ID}
                      sx={{ border: 1, borderColor: 'divider', fontFamily: 'monospace' }}
                    >
                      {formatLogCell(log)}
                    </TableCell>
                  )
                })}
                {showPace &&
                  (() => {
                    const pace = paceMap.get(runner.ID)
                    const nextCP = nextCPMap.get(runner.ID) ?? null
                    const proj =
                      pace && nextCP?.DistanceFromStart != null
                        ? projectArrival(pace, nextCP.DistanceFromStart)
                        : null
                    return (
                      <>
                        <TableCell
                          sx={{ border: 1, borderColor: 'divider', fontFamily: 'monospace' }}
                        >
                          {pace?.paceMinPerMile != null ? formatPace(pace.paceMinPerMile) : '—'}
                        </TableCell>
                        <TableCell
                          sx={{ border: 1, borderColor: 'divider', fontFamily: 'monospace' }}
                        >
                          {proj
                            ? proj.toLocaleTimeString([], {
                                hour: '2-digit',
                                minute: '2-digit',
                                hour12: false,
                              })
                            : '—'}
                        </TableCell>
                      </>
                    )
                  })()}
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </Box>
      {/* ── Runner detail modal ── */}
      {selectedRunner &&
        (() => {
          const race = raceForRunner(selectedRunner)
          const cps = [...(checkpointsByRace[selectedRunner.RaceID] ?? [])].sort(
            (a, b) => a.DisplayOrder - b.DisplayOrder,
          )
          const runnerLogs = (logsByRace[selectedRunner.RaceID] ?? []).filter(
            (l) => l.RunnerID === selectedRunner.ID,
          )
          const logByCp = new Map(runnerLogs.map((l) => [l.CheckpointID, l]))

          const activeCheckpointID =
            session?.Checkpoints?.find((c) => c.RaceID === selectedRunner.RaceID)?.CheckpointID ??
            null
          const activeCP =
            activeCheckpointID != null
              ? (cps.find((cp) => cp.ID === activeCheckpointID) ?? null)
              : null
          const pace = computeRunnerPace(selectedRunner, cps, runnerLogs)
          const projectedArrival =
            activeCP?.DistanceFromStart != null && !logByCp.has(activeCP.ID)
              ? projectArrival(pace, activeCP.DistanceFromStart)
              : null
          return (
            <Dialog open onClose={() => setSelectedRunner(null)} maxWidth="xs" fullWidth>
              <DialogTitle>
                {selectedRunner.FirstName} {selectedRunner.LastName}
              </DialogTitle>
              <DialogContent dividers>
                <Stack spacing={0.5} sx={{ mb: 2 }}>
                  <Typography variant="body2">
                    <strong>Bib:</strong> {selectedRunner.BibNumber}
                  </Typography>
                  <Typography variant="body2">
                    <strong>Race:</strong> {race?.Name ?? `Race ${selectedRunner.RaceID}`}
                  </Typography>
                  <Typography
                    component="div"
                    variant="body2"
                    sx={{ display: 'flex', alignItems: 'center', gap: 1 }}
                  >
                    <strong>Status:</strong>
                    <Chip
                      label={selectedRunner.Status}
                      size="small"
                      color={STATUS_COLOR[selectedRunner.Status] ?? 'default'}
                    />
                  </Typography>
                  {pace.paceMinPerMile != null && (
                    <Typography variant="body2">
                      <strong>Current pace:</strong> {formatPace(pace.paceMinPerMile)}
                    </Typography>
                  )}
                  {activeCP && (
                    <Typography variant="body2">
                      <strong>Proj. arrival at {activeCP.DisplayName}:</strong>{' '}
                      {projectedArrival
                        ? projectedArrival.toLocaleTimeString([], {
                            hour: '2-digit',
                            minute: '2-digit',
                            hour12: false,
                          })
                        : logByCp.has(activeCP.ID)
                          ? 'Already logged'
                          : 'Insufficient data'}
                    </Typography>
                  )}
                </Stack>
                {cps.length > 0 && (
                  <>
                    <Divider sx={{ mb: 1 }} />
                    <Typography variant="subtitle2" sx={{ mb: 1 }}>
                      Checkpoint Log
                    </Typography>
                    <Table size="small">
                      <TableHead>
                        <TableRow>
                          <TableCell>
                            <strong>Checkpoint</strong>
                          </TableCell>
                          <TableCell>
                            <strong>Time</strong>
                          </TableCell>
                        </TableRow>
                      </TableHead>
                      <TableBody>
                        {cps.map((cp) => {
                          const log = logByCp.get(cp.ID)
                          return (
                            <TableRow key={cp.ID}>
                              <TableCell>
                                {cp.Code} – {cp.DisplayName}
                              </TableCell>
                              <TableCell sx={{ fontFamily: 'monospace' }}>
                                {formatLogCell(log)}
                              </TableCell>
                            </TableRow>
                          )
                        })}
                      </TableBody>
                    </Table>
                  </>
                )}
              </DialogContent>
            </Dialog>
          )
        })()}
    </Box>
  )
}

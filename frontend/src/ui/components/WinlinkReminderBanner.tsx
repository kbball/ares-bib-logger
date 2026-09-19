import { useEffect, useState } from 'react'
import { Alert, AlertTitle, Box, Button, Stack } from '@mui/material'
import type { WinlinkReminderStatus } from '../../domain/types'
import * as api from '../../adapters/api'
import { useStream } from '../../adapters/sse/useStream'

// How often to re-check the clock against each reminder's DueAt, so a
// reminder that becomes due while the operator is mid-page still appears
// without requiring a refetch.
const TICK_MS = 15_000

export default function WinlinkReminderBanner() {
  const [reminders, setReminders] = useState<WinlinkReminderStatus[]>([])
  const [now, setNow] = useState(() => Date.now())

  const refresh = () => {
    api
      .listWinlinkReminders()
      .then(setReminders)
      .catch(() => {})
  }

  useEffect(refresh, [])

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), TICK_MS)
    return () => clearInterval(id)
  }, [])

  useStream({ onWinlinkReminderChanged: refresh })

  const overdue = reminders.filter((r) => !r.Dismissed && new Date(r.DueAt).getTime() <= now)
  if (!overdue.length) return null

  const dismiss = async (raceID: number) => {
    setReminders((rs) => rs.map((r) => (r.RaceID === raceID ? { ...r, Dismissed: true } : r)))
    try {
      await api.dismissWinlinkReminder(raceID)
    } catch {
      // Refetch to recover the true state if the dismiss didn't take.
      refresh()
    }
  }

  return (
    <Stack spacing={0.5} sx={{ px: 2, pt: 1 }}>
      {overdue.map((r) => (
        <Alert
          key={r.RaceID}
          severity="warning"
          action={
            <Button color="inherit" size="small" onClick={() => dismiss(r.RaceID)}>
              Dismiss
            </Button>
          }
        >
          <AlertTitle>Send another Winlink update</AlertTitle>
          <Box component="span">
            {r.CheckpointName || r.RaceName} ({r.RaceName}) hasn&apos;t exported since{' '}
            {new Date(r.LastExportAt).toLocaleTimeString([], {
              hour: '2-digit',
              minute: '2-digit',
            })}
            . Time to send an update over the mesh.
          </Box>
        </Alert>
      ))}
    </Stack>
  )
}

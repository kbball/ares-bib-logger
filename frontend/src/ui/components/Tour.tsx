import { useCallback, useEffect, useRef, useState } from 'react'
import { useLocation, useNavigate } from 'react-router-dom'
import { Box, Button, Paper, Typography } from '@mui/material'
import { OPEN_SETUP_EVENT, TOUR_EVENT, TOUR_STEPS, markTourSeen, tourUnseen } from './tourState'

interface Box4 {
  top: number
  left: number
  width: number
  height: number
}

const PAD = 6

const findTarget = (targets: string[]) => {
  for (const t of targets) {
    const el = document.querySelector(t)
    if (el) return el
  }
  return null
}

/**
 * First-run walkthrough. Shown once (the flag is set as soon as it appears); the welcome screen
 * can opt out, and the header can reopen it. Each step goes to the page it is about and
 * spotlights the part of it being described.
 */
export default function Tour() {
  const [step, setStep] = useState<number>() // undefined = closed, -1 = welcome
  const [box, setBox] = useState<Box4>()
  const primary = useRef<HTMLButtonElement>(null)
  const origin = useRef('/')
  const navigate = useNavigate()
  const { pathname } = useLocation()
  const here = useRef(pathname)
  here.current = pathname

  const open = useCallback(() => {
    origin.current = here.current
    setStep(-1)
  }, [])

  useEffect(() => {
    if (tourUnseen()) {
      markTourSeen()
      open()
    }
    window.addEventListener(TOUR_EVENT, open)
    return () => window.removeEventListener(TOUR_EVENT, open)
  }, [open])

  const s = step !== undefined && step >= 0 ? TOUR_STEPS[step] : undefined

  // Go to the step's page, then wait for its target to appear (pages load their data first)
  // and scroll it into view.
  useEffect(() => {
    setBox(undefined)
    if (!s) return
    if (pathname !== s.route) {
      navigate(s.route)
      return
    }
    if (s.opensSetup) window.dispatchEvent(new Event(OPEN_SETUP_EVENT))
    let tries = 0
    let timer: ReturnType<typeof setTimeout>
    const measure = () => {
      const el = findTarget(s.targets)
      if (!el) return setBox(undefined)
      const r = el.getBoundingClientRect()
      setBox({
        top: r.top - PAD,
        left: r.left - PAD,
        width: r.width + 2 * PAD,
        height: r.height + 2 * PAD,
      })
    }
    const find = () => {
      const el = findTarget(s.targets)
      if (el) {
        el.scrollIntoView?.({ block: 'center' })
        // Let an expanding section finish moving before measuring.
        timer = setTimeout(measure, s.opensSetup ? 350 : 0)
      } else if (++tries < 20) {
        timer = setTimeout(find, 100)
      }
    }
    find()
    window.addEventListener('resize', measure)
    window.addEventListener('scroll', measure, true)
    return () => {
      clearTimeout(timer)
      window.removeEventListener('resize', measure)
      window.removeEventListener('scroll', measure, true)
    }
  }, [s, pathname, navigate])

  useEffect(() => {
    if (step !== undefined) primary.current?.focus()
  }, [step])

  if (step === undefined) return null

  const close = () => {
    setStep(undefined)
    if (here.current !== origin.current) navigate(origin.current) // back to where the user was
  }
  const last = step === TOUR_STEPS.length - 1
  // Put the card on the far side of the spotlight so it never covers it.
  const below = box ? box.top + box.height / 2 < window.innerHeight / 2 : false
  const placed = box
    ? {
        position: 'fixed',
        left: '50%',
        transform: 'translateX(-50%)',
        [below ? 'bottom' : 'top']: 16,
      }
    : {}

  return (
    <Box
      onKeyDown={(e) => {
        if (e.key === 'Escape') close()
      }}
      sx={{
        position: 'fixed',
        inset: 0,
        zIndex: 'modal',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        bgcolor: box ? 'transparent' : 'rgba(0, 0, 0, .45)',
      }}
    >
      {box && (
        <Box
          data-testid="tour-spot"
          sx={{
            position: 'fixed',
            ...box,
            borderRadius: '14px',
            boxShadow: (t) =>
              `0 0 0 9999px rgba(0, 0, 0, .55), 0 0 0 3px ${t.palette.primary.main}`,
            pointerEvents: 'none',
          }}
        />
      )}
      <Paper
        role="dialog"
        aria-modal="true"
        aria-labelledby="tour-title"
        sx={{ p: 2.5, m: 2, maxWidth: 420, width: 'calc(100% - 32px)', ...placed }}
      >
        {s ? (
          <>
            <Typography variant="caption" color="text.secondary">
              Step {step + 1} of {TOUR_STEPS.length}
            </Typography>
            <Typography id="tour-title" variant="h6" gutterBottom>
              {s.title}
            </Typography>
            <Typography variant="body2">{s.body}</Typography>
          </>
        ) : (
          <>
            <Typography id="tour-title" variant="h6" gutterBottom>
              Welcome to ARES Bib Logger
            </Typography>
            <Typography variant="body2" sx={{ mb: 1 }}>
              Log runners as they pass your checkpoint, share times with other stations, and keep
              the whole event in one place. Want a one-minute tour of how to set up an event?
            </Typography>
            <Typography variant="body2" color="text.secondary">
              You won&apos;t be asked again. You can replay it any time from the header.
            </Typography>
          </>
        )}
        <Box sx={{ display: 'flex', justifyContent: 'flex-end', gap: 1, mt: 2 }}>
          {s ? (
            <>
              <Button onClick={close}>Close</Button>
              {step > 0 && <Button onClick={() => setStep(step - 1)}>Back</Button>}
              <Button
                ref={primary}
                variant="contained"
                onClick={() => (last ? close() : setStep(step + 1))}
              >
                {last ? 'Done' : 'Next'}
              </Button>
            </>
          ) : (
            <>
              <Button onClick={close}>No thanks</Button>
              <Button ref={primary} variant="contained" onClick={() => setStep(0)}>
                Take the tour
              </Button>
            </>
          )}
        </Box>
      </Paper>
    </Box>
  )
}

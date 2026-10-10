import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  Chip,
  Typography,
} from '@mui/material'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'
import { setupProgress, type SetupStep, type StepState } from '../../application/setup'

const LABEL: Record<StepState, string> = {
  done: 'Done',
  todo: 'To do',
  waiting: 'Waiting',
  optional: 'Optional',
}
const COLOR: Record<StepState, 'success' | 'warning' | 'default'> = {
  done: 'success',
  todo: 'warning',
  waiting: 'default',
  optional: 'default',
}

interface Props {
  steps: SetupStep[]
  onJump: (target: string) => void
}

/** A checklist of what setting up an event involves, in order, with a jump button for each step. */
export default function SetupGuide({ steps, onJump }: Props) {
  const { done, total } = setupProgress(steps)
  return (
    <Accordion defaultExpanded className="setup-guide" aria-label="Setup guide">
      <AccordionSummary expandIcon={<ExpandMoreIcon />}>
        <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5 }}>
          <Typography variant="h6">Setup guide</Typography>
          <Chip
            size="small"
            color={done === total ? 'success' : 'default'}
            label={`${done} of ${total} done`}
          />
        </Box>
      </AccordionSummary>
      <AccordionDetails sx={{ pt: 0 }}>
        <Box component="ol" sx={{ listStyle: 'none', m: 0, p: 0 }}>
          {steps.map((s, i) => (
            <Box
              component="li"
              key={s.id}
              sx={{
                display: 'grid',
                gridTemplateColumns: '28px 1fr auto',
                gap: 1.5,
                alignItems: 'start',
                py: 1,
                borderTop: i === 0 ? 0 : 1,
                borderColor: 'divider',
                opacity: s.state === 'waiting' || s.state === 'optional' ? 0.8 : 1,
              }}
            >
              <Box
                aria-hidden="true"
                sx={{
                  width: 28,
                  height: 28,
                  borderRadius: '50%',
                  display: 'grid',
                  placeItems: 'center',
                  fontSize: '0.8125rem',
                  fontWeight: 700,
                  bgcolor: s.state === 'done' ? 'success.main' : 'action.selected',
                  color: s.state === 'done' ? 'background.paper' : 'text.secondary',
                }}
              >
                {i + 1}
              </Box>
              <Box sx={{ minWidth: 0 }}>
                <Typography component="span" variant="body2" sx={{ fontWeight: 600, mr: 1 }}>
                  {s.title}
                </Typography>
                <Chip size="small" color={COLOR[s.state]} label={LABEL[s.state]} />
                <Typography variant="body2" color="text.secondary" sx={{ mt: 0.25 }}>
                  {s.detail}
                </Typography>
              </Box>
              <Button size="small" aria-label={`Go to ${s.title}`} onClick={() => onJump(s.target)}>
                Go
              </Button>
            </Box>
          ))}
        </Box>
      </AccordionDetails>
    </Accordion>
  )
}

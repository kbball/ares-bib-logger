import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  List,
  ListItem,
  ListItemText,
  Typography,
} from '@mui/material'
import ExpandMoreIcon from '@mui/icons-material/ExpandMore'

type Step = { primary: string; secondary?: string }

function StepList({ steps }: { steps: Step[] }) {
  return (
    <List dense disablePadding>
      {steps.map((s, i) => (
        <ListItem key={i} disableGutters sx={{ alignItems: 'flex-start', pb: 0.5 }}>
          <ListItemText
            primary={`${i + 1}. ${s.primary}`}
            secondary={s.secondary}
            slotProps={{
              primary: { variant: 'body2', sx: { fontWeight: 600 } },
              secondary: { variant: 'body2' },
            }}
          />
        </ListItem>
      ))}
    </List>
  )
}

const SECTIONS = [
  {
    title: 'Before Race Day',
    content: (
      <StepList
        steps={[
          {
            primary: 'Create the event',
            secondary: 'Admin tab → type a name (e.g. "GA Death Race 2026") → Create Event.',
          },
          {
            primary: 'Add races',
            secondary:
              'For GDR: one race named "GDR". For GA Jewel: four races — 100M, 50M, 35M, 18M.',
          },
          {
            primary: 'Define checkpoints',
            secondary:
              'For each race, add checkpoints in order (Start → AS 1 → AS 2 → Finish). Set distance from start (miles) if you want pace and projected arrival times.',
          },
          {
            primary: 'Lock checkpoint order',
            secondary:
              'Once checkpoints are set, lock the order. This prevents column shifts that would break Winlink import mappings mid-race.',
          },
          {
            primary: 'Import the runner roster',
            secondary:
              'Copy three columns from the spreadsheet (Bib, First Name, Last Name) and paste into Roster Import. Lock the roster when confirmed.',
          },
          {
            primary: 'Share config with other stations',
            secondary:
              'Export the event config (download icon next to the active event) and send the JSON file to other stations so they can import it instead of re-entering everything.',
          },
          {
            primary: 'Set your Active Checkpoint',
            secondary:
              'In Admin, under Active Checkpoints, pick the checkpoint your station is physically covering for each race.',
          },
        ]}
      />
    ),
  },
  {
    title: 'On Race Day',
    content: (
      <StepList
        steps={[
          {
            primary: 'Verify your Active Checkpoint',
            secondary:
              'Confirm the correct checkpoint is set for each race — this is what gets logged when you enter a bib number.',
          },
          {
            primary: 'Log bibs as runners pass through',
            secondary:
              'Data Entry tab → type the bib number → press Enter or click Log. The time is recorded automatically.',
          },
          {
            primary: 'Mark DNS / DNF',
            secondary:
              'Data Entry tab → DNS/DNF panel → enter bib → choose status → Submit. This also records a checkpoint log entry.',
          },
          {
            primary: 'Send your Winlink update',
            secondary:
              'Every 20–30 minutes: Winlink Export tab → select race → Generate → copy subject → copy column → paste both into a Winlink message to all stations.',
          },
          {
            primary: 'Receive Winlink updates',
            secondary:
              'Winlink Import tab → select the race and the source checkpoint → paste the column → Import. Repeat for each station that sent you data.',
          },
          {
            primary: 'Monitor the Runners tab',
            secondary:
              'Shows the full picture across all checkpoints as Winlink data comes in. Search by bib or name; click a row for pace and projected arrival.',
          },
        ]}
      />
    ),
  },
  {
    title: 'Winlink Workflow',
    content: (
      <Box>
        <Typography variant="subtitle2" sx={{ fontWeight: 600, mb: 0.5 }}>
          Sending your data
        </Typography>
        <StepList
          steps={[
            { primary: 'Winlink Export tab → select race → Generate' },
            { primary: 'Copy Subject → paste as your email subject line' },
            {
              primary: 'Copy Column Data → paste as the email body',
              secondary:
                'Format: header line (checkpoint name), then one row per runner in roster order — HH:MM, DNS, DNF, MOVED <race>, or blank.',
            },
            { primary: 'Send the Winlink message to all other stations' },
          ]}
        />
        <Typography variant="subtitle2" sx={{ fontWeight: 600, mt: 1.5, mb: 0.5 }}>
          Receiving data from another station
        </Typography>
        <StepList
          steps={[
            {
              primary:
                'Open the Winlink message and copy the column text (everything below the subject)',
            },
            { primary: 'Winlink Import tab → select the race the column belongs to' },
            {
              primary: 'Select the checkpoint the column came from',
              secondary: "Your own active checkpoint is excluded — you can't import your own data.",
            },
            { primary: 'Paste the column → Import' },
            { primary: 'Review the import summary for any skipped rows' },
          ]}
        />
      </Box>
    ),
  },
  {
    title: 'Transferring a Runner Between Races',
    content: (
      <StepList
        steps={[
          {
            primary: 'Data Entry tab → Transfer Runner panel',
            secondary: 'Used when a runner switches from one GA Jewel race to another.',
          },
          {
            primary: 'Enter the bib and select the destination race → Transfer',
            secondary:
              'The runner is marked MOVED in the original race and appended to the bottom of the new race roster.',
          },
          {
            primary: 'Winlink export automatically handles MOVED runners',
            secondary:
              'The original race column shows "MOVED <race name>" for that row so other stations see the transfer.',
          },
        ]}
      />
    ),
  },
  {
    title: 'Tips & Troubleshooting',
    content: (
      <StepList
        steps={[
          {
            primary: 'MQTT / auto-capture not working',
            secondary:
              'Check that MQTT_ENABLED=true in your .env and that the Mosquitto broker is running. The Data Entry tab still works for manual entry with MQTT disabled.',
          },
          {
            primary: 'Wrong DNS/DNF entered',
            secondary:
              'Admin tab → Change Runner Status → search by bib → pick the correct status → Set.',
          },
          {
            primary: 'Winlink import times are off by several hours',
            secondary:
              "Set TIMEZONE in your .env to the event venue's IANA timezone (e.g. America/New_York). Restart the app after changing.",
          },
          {
            primary: 'Checkpoint column shifted after import',
            secondary:
              'This means checkpoint order changed mid-race. Lock checkpoint order in Admin before the race starts to prevent this.',
          },
          {
            primary: 'App data after a container restart',
            secondary:
              'All data is stored in Postgres and survives restarts. The active checkpoint and event are restored automatically on boot.',
          },
        ]}
      />
    ),
  },
]

export default function GuideTab() {
  return (
    <Box sx={{ maxWidth: 720 }}>
      <Typography variant="h5" gutterBottom>
        Operator Guide
      </Typography>
      <Typography variant="body2" color="text.secondary" sx={{ mb: 2 }}>
        A complete walkthrough for setting up and running the app at a race event.
      </Typography>
      {SECTIONS.map((s) => (
        <Accordion key={s.title} disableGutters>
          <AccordionSummary expandIcon={<ExpandMoreIcon />}>
            <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
              {s.title}
            </Typography>
          </AccordionSummary>
          <AccordionDetails>{s.content}</AccordionDetails>
        </Accordion>
      ))}
    </Box>
  )
}

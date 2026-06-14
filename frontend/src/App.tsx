import { useMemo, useState } from 'react'
import {
  ThemeProvider,
  CssBaseline,
  Box,
  Tabs,
  Tab,
  AppBar,
  Toolbar,
  Typography,
  IconButton,
  Tooltip,
  Drawer,
  List,
  ListItem,
  ListItemText,
  Divider,
} from '@mui/material'
import DarkModeIcon from '@mui/icons-material/DarkMode'
import LightModeIcon from '@mui/icons-material/LightMode'
import HelpOutlinedIcon from '@mui/icons-material/HelpOutlined'
import CloseIcon from '@mui/icons-material/Close'
import { createAppTheme, type ColorMode } from './theme'
import DataEntryTab from './ui/pages/DataEntryTab'
import WinlinkImportTab from './ui/pages/WinlinkImportTab'
import WinlinkExportTab from './ui/pages/WinlinkExportTab'
import RunnersTab from './ui/pages/RunnersTab'
import AdminTab from './ui/pages/AdminTab'
import GuideTab from './ui/pages/GuideTab'

const TABS = ['Data Entry', 'Runners', 'Winlink Import', 'Winlink Export', 'Admin', 'Guide']

type HelpItem = { heading: string; text: string }

const HELP: { title: string; items: HelpItem[] }[] = [
  {
    title: 'Data Entry',
    items: [
      {
        heading: 'Log a bib',
        text: 'Type the bib number and press Enter or click Log. The runner is recorded at the active checkpoint with the current time.',
      },
      {
        heading: 'DNS / DNF',
        text: 'Enter the bib, choose DNS (did not start) or DNF (did not finish), and click Submit.',
      },
      {
        heading: 'Transfer runner',
        text: "Enter the bib, pick the destination race, and click Transfer. The runner's row moves to the new race and shows MOVED in the original.",
      },
      {
        heading: 'Race cards',
        text: 'Show live counts of still-to-come, through, and DNS/DNF at the active checkpoint. "Next expected" projects the earliest arrival based on runner paces (requires checkpoint distances in Admin).',
      },
    ],
  },
  {
    title: 'Runners',
    items: [
      { heading: 'Overview', text: 'Shows every runner in the active event across all races.' },
      {
        heading: 'Filter and search',
        text: 'Use the race tabs to focus on one race. Type in the search box to filter by bib number or name.',
      },
      {
        heading: 'Runner detail',
        text: 'Click any row to open the detail panel: full checkpoint log with times and sources, current pace, and projected arrival at the active checkpoint.',
      },
    ],
  },
  {
    title: 'Winlink Import',
    items: [
      {
        heading: 'What it does',
        text: "Imports checkpoint times you received from another station via Winlink radio email. Each row position maps to a runner's sort order in the roster.",
      },
      {
        heading: 'How to use',
        text: 'Select the race the column belongs to, then the checkpoint it came from. Paste the column text and click Import.',
      },
      {
        heading: 'Your checkpoint is excluded',
        text: 'The active checkpoint for this station is not offered — you would never import your own data.',
      },
      {
        heading: 'Import summary',
        text: 'Shows how many times were created, updated, or skipped, with per-row details for skips.',
      },
    ],
  },
  {
    title: 'Winlink Export',
    items: [
      {
        heading: 'What it does',
        text: 'Generates a time column for your active checkpoint ready to paste into a Winlink radio email.',
      },
      {
        heading: 'How to use',
        text: 'Select the race and click Generate. Copy the email subject and the column, then paste both into a Winlink message addressed to the other stations.',
      },
      {
        heading: 'Email subject',
        text: 'Pre-built as "CP Name Race HH:MM update". The time in Copy Subject is refreshed to the moment you click, so it reflects when you actually send.',
      },
      {
        heading: 'Column format',
        text: 'HH:MM for logged times, DNS/DNF for status, MOVED <race> for transferred runners, blank for runners not yet seen.',
      },
    ],
  },
  {
    title: 'Admin',
    items: [
      {
        heading: 'Before race day',
        text: 'Create the event, add races, define checkpoints in order, set distances (miles from start) if you want pace/arrival projections, and import the roster (TSV: BibNumber, FirstName, LastName).',
      },
      {
        heading: 'On race day',
        text: 'Set your Active Checkpoint — the one your station is physically covering. This controls which checkpoint gets logged when bibs are entered.',
      },
      {
        heading: 'Bulk Checkpoint Import',
        text: 'Paste a TSV (Code, DisplayName, DistFromStart) to create multiple checkpoints at once. Faster than adding them one by one.',
      },
      {
        heading: 'Change Runner Status',
        text: 'Correct a DNS or DNF entry: search by bib, pick the new status, and click Set.',
      },
      {
        heading: 'Locks',
        text: 'Lock Roster (via Import Roster) and Lock Order prevent accidental changes mid-race. Both are permanent for that race.',
      },
    ],
  },
  {
    title: 'Guide',
    items: [
      {
        heading: 'Operator Guide',
        text: 'Step-by-step instructions for setting up and running the app at a race event. Expand each section for details.',
      },
    ],
  },
]

export default function App() {
  const [tab, setTab] = useState(0)
  const [colorMode, setColorMode] = useState<ColorMode>('light')
  const [helpOpen, setHelpOpen] = useState(false)
  const theme = useMemo(() => createAppTheme(colorMode), [colorMode])
  const toggleMode = () => setColorMode((m) => (m === 'dark' ? 'light' : 'dark'))

  const help = HELP[tab]

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <AppBar position="static">
        <Toolbar variant="dense">
          <Box
            component="img"
            src="/logo.png"
            alt="ARES Bib Logger"
            sx={{ height: 40, width: 'auto', mr: 1.5 }}
          />
          <Typography variant="h6" sx={{ mr: 2 }}>
            ARES Bib Logger
          </Typography>
          <Box sx={{ flexGrow: 1 }} />
          <Tooltip title="Help for this tab">
            <IconButton
              onClick={() => setHelpOpen(true)}
              size="small"
              color="inherit"
              aria-label="Open help"
            >
              <HelpOutlinedIcon />
            </IconButton>
          </Tooltip>
          <Tooltip title={colorMode === 'dark' ? 'Switch to light mode' : 'Switch to dark mode'}>
            <IconButton onClick={toggleMode} size="small" color="inherit">
              {colorMode === 'dark' ? <LightModeIcon /> : <DarkModeIcon />}
            </IconButton>
          </Tooltip>
        </Toolbar>
        <Tabs
          value={tab}
          onChange={(_, v) => setTab(v)}
          textColor="inherit"
          indicatorColor="secondary"
          variant="scrollable"
        >
          {TABS.map((label) => (
            <Tab key={label} label={label} />
          ))}
        </Tabs>
      </AppBar>

      <Box sx={{ p: 2 }}>
        {tab === 0 && <DataEntryTab />}
        {tab === 1 && <RunnersTab />}
        {tab === 2 && <WinlinkImportTab />}
        {tab === 3 && <WinlinkExportTab />}
        {tab === 4 && <AdminTab />}
        {tab === 5 && <GuideTab />}
      </Box>

      <Drawer anchor="right" open={helpOpen} onClose={() => setHelpOpen(false)}>
        <Box sx={{ width: 320, p: 2 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', mb: 1 }}>
            <Typography variant="h6" sx={{ flexGrow: 1 }}>
              {help.title}
            </Typography>
            <IconButton size="small" onClick={() => setHelpOpen(false)} aria-label="Close help">
              <CloseIcon />
            </IconButton>
          </Box>
          <Divider sx={{ mb: 1 }} />
          <List disablePadding>
            {help.items.map((item, i) => (
              <ListItem
                key={i}
                disableGutters
                alignItems="flex-start"
                sx={{ flexDirection: 'column', pb: 1.5 }}
              >
                <ListItemText
                  primary={item.heading}
                  secondary={item.text}
                  slotProps={{
                    primary: { variant: 'body2', sx: { fontWeight: 600 } },
                    secondary: { variant: 'body2' },
                  }}
                />
              </ListItem>
            ))}
          </List>
        </Box>
      </Drawer>
    </ThemeProvider>
  )
}

import { useMemo, useState } from 'react'
import {
  ThemeProvider, CssBaseline, Box, Tabs, Tab,
  AppBar, Toolbar, Typography, IconButton, Tooltip,
} from '@mui/material'
import DarkModeIcon from '@mui/icons-material/DarkMode'
import LightModeIcon from '@mui/icons-material/LightMode'
import { createAppTheme, type ColorMode } from './theme'
import DataEntryTab from './ui/pages/DataEntryTab'
import WinlinkImportTab from './ui/pages/WinlinkImportTab'
import WinlinkExportTab from './ui/pages/WinlinkExportTab'
import RunnersTab from './ui/pages/RunnersTab'
import AdminTab from './ui/pages/AdminTab'

const TABS = ['Data Entry', 'Winlink Import', 'Winlink Export', 'Runners', 'Admin']

export default function App() {
  const [tab, setTab] = useState(0)
  const [colorMode, setColorMode] = useState<ColorMode>('light')
  const theme = useMemo(() => createAppTheme(colorMode), [colorMode])
  const toggleMode = () => setColorMode((m) => (m === 'dark' ? 'light' : 'dark'))

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
        {tab === 1 && <WinlinkImportTab />}
        {tab === 2 && <WinlinkExportTab />}
        {tab === 3 && <RunnersTab />}
        {tab === 4 && <AdminTab />}
      </Box>
    </ThemeProvider>
  )
}

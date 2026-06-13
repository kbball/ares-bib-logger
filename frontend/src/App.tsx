import { useState } from 'react'
import { ThemeProvider, createTheme, CssBaseline, Box, Tabs, Tab, AppBar, Toolbar, Typography } from '@mui/material'
import DataEntryTab from './ui/pages/DataEntryTab'
import WinlinkImportTab from './ui/pages/WinlinkImportTab'
import WinlinkExportTab from './ui/pages/WinlinkExportTab'
import RunnersTab from './ui/pages/RunnersTab'
import AdminTab from './ui/pages/AdminTab'

const theme = createTheme({ palette: { mode: 'light' } })

const TABS = ['Data Entry', 'Winlink Import', 'Winlink Export', 'Runners', 'Admin']

export default function App() {
  const [tab, setTab] = useState(0)

  return (
    <ThemeProvider theme={theme}>
      <CssBaseline />
      <AppBar position="static">
        <Toolbar variant="dense">
          <Typography variant="h6" sx={{ mr: 2 }}>
            Ares Bib Logger
          </Typography>
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

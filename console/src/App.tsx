import { Box, CssBaseline, Paper } from '@mui/material'
import { useState } from 'react'

import { Collectors } from './components/Collectors'
import { Dashboard } from './components/Dashboard'
import { Intelligence } from './components/Intelligence'
import { Signals } from './components/Signals'
import { Translation } from './components/Translation'
import { Settings } from './components/Settings'
import { AppMenu } from './components/AppMenu'
import { useAdminTheme } from './theme/AdminThemeContext'

export function App() {
  const { mode } = useAdminTheme()
  const [currentView, setCurrentView] = useState('dashboard')

  return (
    <Box sx={{ display: 'flex', height: '100vh' }}>
      <CssBaseline />
      <AppMenu currentView={currentView} onViewChange={setCurrentView} />
      <Box
        component="main"
        sx={{
          flexGrow: 1,
          bgcolor: 'background.default',
          p: 3,
          overflow: 'auto',
        }}
      >
        <Paper
          sx={{
            p: 3,
            minHeight: 'calc(100vh - 48px)',
            borderRadius: 0,
            border: '1px solid',
            borderColor: 'divider',
          }}
        >
          {currentView === 'dashboard' && <Dashboard />}
          {currentView === 'collectors' && <Collectors />}
          {currentView === 'intelligence' && <Intelligence />}
          {currentView === 'signals' && <Signals />}
          {currentView === 'translation' && <Translation />}
          {currentView === 'settings' && <Settings />}
        </Paper>
      </Box>
    </Box>
  )
}

export default App

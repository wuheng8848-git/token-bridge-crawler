import { ThemeProvider, createTheme } from '@mui/material/styles'
import { createContext, useCallback, useContext, useMemo, useState, type ReactNode } from 'react'

type ThemeMode = 'light' | 'dark'

type Ctx = {
  mode: ThemeMode
  setMode: (mode: ThemeMode) => void
}

const AdminThemeContext = createContext<Ctx | null>(null)

// 浅色主题
const lightTheme = createTheme({
  palette: {
    mode: 'light',
    primary: { main: '#2563EB' },
    secondary: { main: '#7C3AED' },
    background: { default: '#F8FAFC', paper: '#FFFFFF' },
  },
  typography: {
    fontFamily: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
  },
  shape: { borderRadius: 4 },
})

// 深色主题
const darkTheme = createTheme({
  palette: {
    mode: 'dark',
    primary: { main: '#60A5FA' },
    secondary: { main: '#A78BFA' },
    background: { default: '#0F172A', paper: '#1E293B' },
  },
  typography: {
    fontFamily: '"Inter", -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif',
  },
  shape: { borderRadius: 4 },
})

export function AdminThemeProvider({ children }: { children: ReactNode }) {
  const [mode, setModeState] = useState<ThemeMode>('light')

  const setMode = useCallback((newMode: ThemeMode) => {
    setModeState(newMode)
    localStorage.setItem('intelligence-console-theme', newMode)
  }, [])

  const theme = useMemo(() => (mode === 'dark' ? darkTheme : lightTheme), [mode])

  const value = useMemo(() => ({ mode, setMode }), [mode, setMode])

  return (
    <AdminThemeContext.Provider value={value}>
      <ThemeProvider theme={theme}>{children}</ThemeProvider>
    </AdminThemeContext.Provider>
  )
}

export function useAdminTheme() {
  const v = useContext(AdminThemeContext)
  if (!v) {
    throw new Error('useAdminTheme must be used within AdminThemeProvider')
  }
  return v
}

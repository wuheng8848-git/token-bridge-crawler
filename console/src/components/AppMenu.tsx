import {
  Dashboard as DashboardIcon,
  Memory as MemoryIcon,
  Article as ArticleIcon,
  BugReport as BugReportIcon,
  Translate as TranslateIcon,
  Settings as SettingsIcon,
  DarkMode as DarkModeIcon,
  LightMode as LightModeIcon,
} from '@mui/icons-material'
import {
  Box,
  Drawer,
  IconButton,
  List,
  ListItem,
  ListItemButton,
  ListItemIcon,
  ListItemText,
  Toolbar,
  Typography,
} from '@mui/material'

import { useAdminTheme } from '../theme/AdminThemeContext'

const DRAWER_WIDTH = 240

const MENU_ITEMS = [
  { id: 'dashboard', label: '监控大盘', icon: DashboardIcon },
  { id: 'collectors', label: '采集器管理', icon: MemoryIcon },
  { id: 'intelligence', label: '情报浏览', icon: ArticleIcon },
  { id: 'signals', label: '营销信号调试（未启用）', icon: BugReportIcon },
  { id: 'translation', label: '翻译服务', icon: TranslateIcon },
  { id: 'settings', label: '系统配置', icon: SettingsIcon },
]

interface AppMenuProps {
  currentView: string
  onViewChange: (view: string) => void
}

export function AppMenu({ currentView, onViewChange }: AppMenuProps) {
  const { mode, setMode } = useAdminTheme()

  const toggleTheme = () => {
    setMode(mode === 'light' ? 'dark' : 'light')
  }

  return (
    <Drawer
      variant="permanent"
      sx={{
        width: DRAWER_WIDTH,
        flexShrink: 0,
        '& .MuiDrawer-paper': {
          width: DRAWER_WIDTH,
          boxSizing: 'border-box',
        },
      }}
    >
      <Toolbar>
        <Typography variant="h6" noWrap component="div" sx={{ fontWeight: 700 }}>
          TB Intelligence
        </Typography>
      </Toolbar>
      <Box sx={{ overflow: 'auto' }}>
        <List>
          {MENU_ITEMS.map((item) => {
            const Icon = item.icon
            const isSelected = currentView === item.id
            return (
              <ListItem key={item.id} disablePadding>
                <ListItemButton
                  selected={isSelected}
                  onClick={() => onViewChange(item.id)}
                >
                  <ListItemIcon>
                    <Icon color={isSelected ? 'primary' : 'inherit'} />
                  </ListItemIcon>
                  <ListItemText primary={item.label} />
                </ListItemButton>
              </ListItem>
            )
          })}
        </List>
      </Box>
      <Box sx={{ mt: 'auto', p: 2 }}>
        <IconButton onClick={toggleTheme} color="inherit">
          {mode === 'light' ? <DarkModeIcon /> : <LightModeIcon />}
        </IconButton>
      </Box>
    </Drawer>
  )
}

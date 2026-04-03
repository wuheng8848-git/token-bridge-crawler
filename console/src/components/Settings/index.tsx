import {
  Box,
  Card,
  CardContent,
  Typography,
  TextField,
  Switch,
  FormControlLabel,
  Button,
  Divider,
  Alert,
  Snackbar,
  Grid,
  Chip,
} from '@mui/material'
import { Save, Refresh } from '@mui/icons-material'
import { useEffect, useState } from 'react'

interface CollectorConfig {
  name: string
  enabled: boolean
  schedule: string
  rateLimit: string
}

interface SystemSettings {
  databaseUrl: string
  logLevel: string
  timezone: string
  collectors: CollectorConfig[]
  translationEnabled: boolean
  aiReportEnabled: boolean
  emailEnabled: boolean
}

export function Settings() {
  const [settings, setSettings] = useState<SystemSettings | null>(null)
  const [loading, setLoading] = useState(false)
  const [saving, setSaving] = useState(false)
  const [snackbar, setSnackbar] = useState<{ open: boolean; message: string; severity: 'success' | 'error' }>({
    open: false,
    message: '',
    severity: 'success',
  })

  useEffect(() => {
    fetchSettings()
  }, [])

  const fetchSettings = async () => {
    setLoading(true)
    try {
      const res = await fetch('/api/v1/settings')
      const data = await res.json()
      setSettings(data)
    } catch (error) {
      console.error('Failed to fetch settings:', error)
      setSnackbar({ open: true, message: '加载配置失败', severity: 'error' })
    } finally {
      setLoading(false)
    }
  }

  const handleSave = async () => {
    if (!settings) return
    
    setSaving(true)
    try {
      await fetch('/api/v1/settings', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings),
      })
      setSnackbar({ open: true, message: '配置已保存', severity: 'success' })
    } catch (error) {
      console.error('Failed to save settings:', error)
      setSnackbar({ open: true, message: '保存失败', severity: 'error' })
    } finally {
      setSaving(false)
    }
  }

  const toggleCollector = (name: string) => {
    if (!settings) return
    setSettings({
      ...settings,
      collectors: settings.collectors.map(c =>
        c.name === name ? { ...c, enabled: !c.enabled } : c
      ),
    })
  }

  if (!settings) {
    return (
      <Box>
        <Typography variant="h4" gutterBottom sx={{ fontWeight: 700 }}>
          系统配置
        </Typography>
        <Alert severity="info">加载中...</Alert>
      </Box>
    )
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom sx={{ fontWeight: 700 }}>
        系统配置
      </Typography>

      {/* 基础配置 */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
            基础配置
          </Typography>
          <Grid container spacing={3}>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="日志级别"
                value={settings.logLevel}
                onChange={(e) => setSettings({ ...settings, logLevel: e.target.value })}
                select
                SelectProps={{ native: true }}
              >
                <option value="debug">Debug</option>
                <option value="info">Info</option>
                <option value="warn">Warn</option>
                <option value="error">Error</option>
              </TextField>
            </Grid>
            <Grid item xs={12} md={6}>
              <TextField
                fullWidth
                label="时区"
                value={settings.timezone}
                onChange={(e) => setSettings({ ...settings, timezone: e.target.value })}
              />
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* 功能开关 */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
            功能开关
          </Typography>
          <Grid container spacing={2}>
            <Grid item xs={12} sm={4}>
              <FormControlLabel
                control={
                  <Switch
                    checked={settings.translationEnabled}
                    onChange={(e) => setSettings({ ...settings, translationEnabled: e.target.checked })}
                  />
                }
                label="翻译服务"
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <FormControlLabel
                control={
                  <Switch
                    checked={settings.aiReportEnabled}
                    onChange={(e) => setSettings({ ...settings, aiReportEnabled: e.target.checked })}
                  />
                }
                label="AI日报"
              />
            </Grid>
            <Grid item xs={12} sm={4}>
              <FormControlLabel
                control={
                  <Switch
                    checked={settings.emailEnabled}
                    onChange={(e) => setSettings({ ...settings, emailEnabled: e.target.checked })}
                  />
                }
                label="邮件通知"
              />
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* 采集器配置 */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
            采集器配置
          </Typography>
          {settings.collectors.map((collector) => (
            <Box key={collector.name} sx={{ mb: 2, p: 2, bgcolor: 'grey.50', borderRadius: 1 }}>
              <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', mb: 1 }}>
                <Typography variant="subtitle1" sx={{ fontWeight: 600 }}>
                  {collector.name}
                </Typography>
                <FormControlLabel
                  control={
                    <Switch
                      checked={collector.enabled}
                      onChange={() => toggleCollector(collector.name)}
                    />
                  }
                  label={collector.enabled ? '启用' : '禁用'}
                />
              </Box>
              <Box sx={{ display: 'flex', gap: 2, flexWrap: 'wrap' }}>
                <Chip label={`调度: ${collector.schedule}`} size="small" />
                <Chip label={`限流: ${collector.rateLimit}`} size="small" />
              </Box>
            </Box>
          ))}
        </CardContent>
      </Card>

      {/* 环境变量提示 */}
      <Alert severity="info" sx={{ mb: 3 }}>
        注意：数据库连接等敏感配置请通过环境变量设置，修改后需要重启服务生效。
      </Alert>

      {/* 操作按钮 */}
      <Box sx={{ display: 'flex', gap: 2 }}>
        <Button
          variant="contained"
          startIcon={<Save />}
          onClick={handleSave}
          disabled={saving}
        >
          {saving ? '保存中...' : '保存配置'}
        </Button>
        <Button
          variant="outlined"
          startIcon={<Refresh />}
          onClick={fetchSettings}
          disabled={loading}
        >
          刷新
        </Button>
      </Box>

      <Snackbar
        open={snackbar.open}
        autoHideDuration={3000}
        onClose={() => setSnackbar({ ...snackbar, open: false })}
      >
        <Alert severity={snackbar.severity} onClose={() => setSnackbar({ ...snackbar, open: false })}>
          {snackbar.message}
        </Alert>
      </Snackbar>
    </Box>
  )
}

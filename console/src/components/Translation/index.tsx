import {
  Box,
  Card,
  CardContent,
  Chip,
  LinearProgress,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Paper,
  Grid,
  Alert,
} from '@mui/material'
import { CheckCircle, Error, Schedule } from '@mui/icons-material'
import { useEffect, useState } from 'react'

interface TranslationStats {
  total: number
  pending: number
  completed: number
  failed: number
  successRate: number
  avgLatency: number
}

interface TranslationProvider {
  name: string
  enabled: boolean
  priority: number
  usageCount: number
  errorCount: number
  avgLatency: number
  status: 'healthy' | 'degraded' | 'down'
}

interface TranslationTask {
  id: string
  provider: string
  sourceLang: string
  targetLang: string
  status: string
  createdAt: string
  completedAt?: string
  error?: string
}

export function Translation() {
  const [stats, setStats] = useState<TranslationStats | null>(null)
  const [providers, setProviders] = useState<TranslationProvider[]>([])
  const [tasks, setTasks] = useState<TranslationTask[]>([])
  const [loading, setLoading] = useState(false)

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 10000)
    return () => clearInterval(interval)
  }, [])

  const fetchData = async () => {
    setLoading(true)
    try {
      const [statsRes, providersRes, tasksRes] = await Promise.all([
        fetch('/api/v1/translation/stats'),
        fetch('/api/v1/translation/providers'),
        fetch('/api/v1/translation/tasks'),
      ])

      const statsData = await statsRes.json()
      const providersData = await providersRes.json()
      const tasksData = await tasksRes.json()

      setStats(statsData)
      setProviders(providersData.items || [])
      setTasks(tasksData.items || [])
    } catch (error) {
      console.error('Failed to fetch translation data:', error)
    } finally {
      setLoading(false)
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case 'completed':
        return <CheckCircle color="success" fontSize="small" />
      case 'failed':
        return <Error color="error" fontSize="small" />
      case 'pending':
        return <Schedule color="warning" fontSize="small" />
      default:
        return null
    }
  }

  const getProviderStatusColor = (status: string): any => {
    switch (status) {
      case 'healthy':
        return 'success'
      case 'degraded':
        return 'warning'
      case 'down':
        return 'error'
      default:
        return 'default'
    }
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom sx={{ fontWeight: 700 }}>
        翻译服务
      </Typography>

      {/* 统计卡片 */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                总任务数
              </Typography>
              <Typography variant="h4" sx={{ fontWeight: 700 }}>
                {stats?.total || 0}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                待处理
              </Typography>
              <Typography variant="h4" sx={{ fontWeight: 700, color: 'warning.main' }}>
                {stats?.pending || 0}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                成功率
              </Typography>
              <Typography variant="h4" sx={{ fontWeight: 700, color: 'success.main' }}>
                {stats ? `${(stats.successRate * 100).toFixed(1)}%` : 'N/A'}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                平均延迟
              </Typography>
              <Typography variant="h4" sx={{ fontWeight: 700 }}>
                {stats ? `${stats.avgLatency}ms` : 'N/A'}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* 服务商状态 */}
      <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
        翻译服务商
      </Typography>
      <TableContainer component={Paper} sx={{ mb: 3 }}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>服务商</TableCell>
              <TableCell>状态</TableCell>
              <TableCell>优先级</TableCell>
              <TableCell>使用量</TableCell>
              <TableCell>错误数</TableCell>
              <TableCell>平均延迟</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {providers.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} align="center">
                  暂无服务商数据
                </TableCell>
              </TableRow>
            ) : (
              providers.map((provider) => (
                <TableRow key={provider.name}>
                  <TableCell>{provider.name}</TableCell>
                  <TableCell>
                    <Chip
                      label={provider.status}
                      color={getProviderStatusColor(provider.status)}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>{provider.priority}</TableCell>
                  <TableCell>{provider.usageCount}</TableCell>
                  <TableCell>
                    {provider.errorCount > 0 ? (
                      <Typography color="error">{provider.errorCount}</Typography>
                    ) : (
                      provider.errorCount
                    )}
                  </TableCell>
                  <TableCell>{provider.avgLatency}ms</TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {/* 最近任务 */}
      <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
        最近任务
      </Typography>
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>ID</TableCell>
              <TableCell>服务商</TableCell>
              <TableCell>语言</TableCell>
              <TableCell>状态</TableCell>
              <TableCell>创建时间</TableCell>
              <TableCell>耗时</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {tasks.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} align="center">
                  暂无任务数据
                </TableCell>
              </TableRow>
            ) : (
              tasks.map((task) => (
                <TableRow key={task.id}>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                    {task.id.slice(0, 8)}...
                  </TableCell>
                  <TableCell>{task.provider}</TableCell>
                  <TableCell>
                    {task.sourceLang} → {task.targetLang}
                  </TableCell>
                  <TableCell>
                    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
                      {getStatusIcon(task.status)}
                      <Chip
                        label={task.status}
                        color={
                          task.status === 'completed'
                            ? 'success'
                            : task.status === 'failed'
                            ? 'error'
                            : 'warning'
                        }
                        size="small"
                      />
                    </Box>
                  </TableCell>
                  <TableCell>
                    {new Date(task.createdAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    {task.completedAt
                      ? `${new Date(task.completedAt).getTime() - new Date(task.createdAt).getTime()}ms`
                      : '-'}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>

      {loading && <LinearProgress sx={{ mt: 2 }} />}
    </Box>
  )
}

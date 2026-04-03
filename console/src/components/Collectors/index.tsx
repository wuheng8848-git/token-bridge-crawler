import { PlayArrow, Pause, Refresh, Schedule } from '@mui/icons-material'
import {
  Box,
  Button,
  Card,
  CardContent,
  Chip,
  Grid,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Typography,
  Paper,
  IconButton,
  Tooltip,
} from '@mui/material'
import { useEffect, useState } from 'react'

interface Collector {
  name: string
  type: string
  source: string
  rateLimit: string
  enabled?: boolean
  lastRun?: string
  status?: 'running' | 'stopped' | 'error'
}

interface CollectorRun {
  id: string
  collectorName: string
  intelType: string
  source: string
  status: string
  itemsCount: number
  startedAt: string
  durationMs: number
}

export function Collectors() {
  const [collectors, setCollectors] = useState<Collector[]>([])
  const [runs, setRuns] = useState<CollectorRun[]>([])
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    fetchData()
    const interval = setInterval(fetchData, 30000)
    return () => clearInterval(interval)
  }, [])

  const fetchData = async () => {
    try {
      const [collectorsRes, runsRes] = await Promise.all([
        fetch('/api/v1/collectors'),
        fetch('/api/v1/collector-runs'),
      ])

      const collectorsData = await collectorsRes.json()
      const runsData = await runsRes.json()

      setCollectors(collectorsData.items || [])
      setRuns(runsData.items || [])
    } catch (error) {
      console.error('Failed to fetch collectors:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleTrigger = async (collectorName: string) => {
    try {
      await fetch(`/api/v1/collectors/${collectorName}/trigger`, {
        method: 'POST',
      })
      fetchData()
    } catch (error) {
      console.error('Failed to trigger collector:', error)
    }
  }

  const getStatusColor = (status?: string) => {
    switch (status) {
      case 'running':
        return 'success'
      case 'error':
        return 'error'
      case 'stopped':
      default:
        return 'default'
    }
  }

  const getStatusLabel = (status?: string) => {
    switch (status) {
      case 'running':
        return '运行中'
      case 'error':
        return '异常'
      case 'stopped':
        return '已停止'
      default:
        return '未知'
    }
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom sx={{ fontWeight: 700 }}>
        采集器管理
      </Typography>

      {/* 采集器卡片 */}
      <Grid container spacing={2} sx={{ mb: 4 }}>
        {collectors.map((collector) => (
          <Grid item xs={12} sm={6} md={4} key={collector.name}>
            <Card>
              <CardContent>
                <Box
                  sx={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'flex-start',
                    mb: 2,
                  }}
                >
                  <Box>
                    <Typography variant="h6" sx={{ fontWeight: 600 }}>
                      {collector.name}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      {collector.source}
                    </Typography>
                  </Box>
                  <Chip
                    label={getStatusLabel(collector.status)}
                    color={getStatusColor(collector.status) as any}
                    size="small"
                  />
                </Box>

                <Typography variant="body2" color="text.secondary" gutterBottom>
                  类型: {collector.type}
                </Typography>
                <Typography variant="body2" color="text.secondary" gutterBottom>
                  限流: {collector.rateLimit}
                </Typography>

                <Box sx={{ mt: 2, display: 'flex', gap: 1 }}>
                  <Tooltip title="立即执行">
                    <IconButton
                      size="small"
                      color="primary"
                      onClick={() => handleTrigger(collector.name)}
                    >
                      <PlayArrow />
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="启用/暂停">
                    <IconButton size="small">
                      {collector.enabled !== false ? <Pause /> : <PlayArrow />}
                    </IconButton>
                  </Tooltip>
                  <Tooltip title="刷新">
                    <IconButton size="small" onClick={fetchData}>
                      <Refresh />
                    </IconButton>
                  </Tooltip>
                </Box>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      {/* 最近运行记录 */}
      <Typography variant="h6" gutterBottom sx={{ fontWeight: 600 }}>
        最近运行记录
      </Typography>
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>采集器</TableCell>
              <TableCell>类型</TableCell>
              <TableCell>状态</TableCell>
              <TableCell>采集数量</TableCell>
              <TableCell>耗时(ms)</TableCell>
              <TableCell>开始时间</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {runs.length === 0 ? (
              <TableRow>
                <TableCell colSpan={6} align="center">
                  暂无运行记录
                </TableCell>
              </TableRow>
            ) : (
              runs.map((run) => (
                <TableRow key={run.id}>
                  <TableCell>{run.collectorName}</TableCell>
                  <TableCell>{run.intelType}</TableCell>
                  <TableCell>
                    <Chip
                      label={run.status}
                      color={
                        run.status === 'success'
                          ? 'success'
                          : run.status === 'failed'
                          ? 'error'
                          : 'default'
                      }
                      size="small"
                    />
                  </TableCell>
                  <TableCell>{run.itemsCount}</TableCell>
                  <TableCell>{run.durationMs}</TableCell>
                  <TableCell>
                    {new Date(run.startedAt).toLocaleString()}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
      </TableContainer>
    </Box>
  )
}

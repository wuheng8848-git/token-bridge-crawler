import {
  Box,
  Card,
  CardContent,
  Chip,
  FormControl,
  InputLabel,
  MenuItem,
  Select,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TablePagination,
  TableRow,
  Typography,
  Paper,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Grid,
  Slider,
} from '@mui/material'
import { Visibility, CheckCircle, Cancel } from '@mui/icons-material'
import { useEffect, useState } from 'react'

interface Signal {
  id: string
  signalType: string
  strength: number
  content: string
  platform: string
  author: string
  url: string
  status: string
  detectedAt: string
  metadata: Record<string, any>
}

const SIGNAL_TYPES = [
  { value: '', label: '全部类型' },
  { value: 'cost_pressure', label: '成本压力' },
  { value: 'config_friction', label: '配置摩擦' },
  { value: 'tool_fragmentation', label: '工具碎片化' },
  { value: 'governance_start', label: '治理起点' },
  { value: 'migration_intent', label: '迁移意愿' },
  { value: 'general_interest', label: '泛兴趣' },
]

const STRENGTH_LABELS: Record<number, string> = {
  1: '弱',
  2: '中',
  3: '强',
}

export function Signals() {
  const [signals, setSignals] = useState<Signal[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(0)
  const [rowsPerPage, setRowsPerPage] = useState(10)
  const [total, setTotal] = useState(0)
  
  // 筛选条件
  const [filterType, setFilterType] = useState('')
  const [filterStrength, setFilterStrength] = useState<number | ''>('')
  const [filterStatus, setFilterStatus] = useState('')
  
  // 详情弹窗
  const [selectedSignal, setSelectedSignal] = useState<Signal | null>(null)
  // 阈值调整弹窗
  const [thresholdDialogOpen, setThresholdDialogOpen] = useState(false)
  const [threshold, setThreshold] = useState(1)

  useEffect(() => {
    fetchData()
  }, [page, rowsPerPage, filterType, filterStrength, filterStatus])

  const fetchData = async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({
        page: String(page + 1),
        perPage: String(rowsPerPage),
        ...(filterType && { type: filterType }),
        ...(filterStrength && { strength: String(filterStrength) }),
        ...(filterStatus && { status: filterStatus }),
      })
      
      const res = await fetch(`/api/v1/signals?${params}`)
      const data = await res.json()
      
      setSignals(data.items || [])
      setTotal(data.total || 0)
    } catch (error) {
      console.error('Failed to fetch signals:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleValidate = async (id: string, valid: boolean) => {
    try {
      await fetch(`/api/v1/signals/${id}/validate`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ valid }),
      })
      fetchData()
    } catch (error) {
      console.error('Failed to validate signal:', error)
    }
  }

  const getTypeColor = (type: string): any => {
    const colors: Record<string, any> = {
      cost_pressure: 'error',
      config_friction: 'warning',
      tool_fragmentation: 'info',
      governance_start: 'success',
      migration_intent: 'secondary',
      general_interest: 'default',
    }
    return colors[type] || 'default'
  }

  const getTypeLabel = (type: string) => {
    const found = SIGNAL_TYPES.find(t => t.value === type)
    return found?.label || type
  }

  const getStrengthColor = (strength: number): any => {
    if (strength >= 3) return 'error'
    if (strength >= 2) return 'warning'
    return 'default'
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom sx={{ fontWeight: 700 }}>
        信号调试
      </Typography>

      {/* 统计卡片 */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                总信号数
              </Typography>
              <Typography variant="h4" sx={{ fontWeight: 700 }}>
                {total}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                高优先级
              </Typography>
              <Typography variant="h4" sx={{ fontWeight: 700, color: 'error.main' }}>
                {signals.filter(s => s.strength >= 3).length}
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
              <Typography variant="h4" sx={{ fontWeight: 700 }}>
                {signals.filter(s => s.status === 'new').length}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <Card>
            <CardContent>
              <Typography color="text.secondary" gutterBottom>
                已验证
              </Typography>
              <Typography variant="h4" sx={{ fontWeight: 700, color: 'success.main' }}>
                {signals.filter(s => s.status === 'validated').length}
              </Typography>
            </CardContent>
          </Card>
        </Grid>
      </Grid>

      {/* 筛选栏 */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} sm={6} md={3}>
              <FormControl fullWidth size="small">
                <InputLabel>信号类型</InputLabel>
                <Select
                  value={filterType}
                  label="信号类型"
                  onChange={(e) => setFilterType(e.target.value as string)}
                >
                  {SIGNAL_TYPES.map(t => (
                    <MenuItem key={t.value} value={t.value}>{t.label}</MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <FormControl fullWidth size="small">
                <InputLabel>信号强度</InputLabel>
                <Select
                  value={filterStrength}
                  label="信号强度"
                  onChange={(e) => setFilterStrength(e.target.value as number | '')}
                >
                  <MenuItem value="">全部</MenuItem>
                  <MenuItem value={3}>强</MenuItem>
                  <MenuItem value={2}>中</MenuItem>
                  <MenuItem value={1}>弱</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <FormControl fullWidth size="small">
                <InputLabel>状态</InputLabel>
                <Select
                  value={filterStatus}
                  label="状态"
                  onChange={(e) => setFilterStatus(e.target.value as string)}
                >
                  <MenuItem value="">全部</MenuItem>
                  <MenuItem value="new">新增</MenuItem>
                  <MenuItem value="validated">已验证</MenuItem>
                  <MenuItem value="invalid">无效</MenuItem>
                  <MenuItem value="processed">已处理</MenuItem>
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <Button
                variant="outlined"
                onClick={() => setThresholdDialogOpen(true)}
              >
                调整阈值
              </Button>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* 数据表格 */}
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>类型</TableCell>
              <TableCell>强度</TableCell>
              <TableCell>内容摘要</TableCell>
              <TableCell>平台</TableCell>
              <TableCell>状态</TableCell>
              <TableCell>检测时间</TableCell>
              <TableCell>操作</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {signals.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center">
                  暂无信号数据
                </TableCell>
              </TableRow>
            ) : (
              signals.map((signal) => (
                <TableRow key={signal.id} hover>
                  <TableCell>
                    <Chip
                      label={getTypeLabel(signal.signalType)}
                      color={getTypeColor(signal.signalType)}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={STRENGTH_LABELS[signal.strength]}
                      color={getStrengthColor(signal.strength)}
                      size="small"
                    />
                  </TableCell>
                  <TableCell sx={{ maxWidth: 300 }}>
                    <Typography noWrap variant="body2">
                      {signal.content}
                    </Typography>
                  </TableCell>
                  <TableCell>{signal.platform}</TableCell>
                  <TableCell>
                    <Chip
                      label={signal.status}
                      color={
                        signal.status === 'new'
                          ? 'success'
                          : signal.status === 'validated'
                          ? 'primary'
                          : signal.status === 'invalid'
                          ? 'error'
                          : 'default'
                      }
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    {new Date(signal.detectedAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <IconButton
                      size="small"
                      onClick={() => setSelectedSignal(signal)}
                    >
                      <Visibility />
                    </IconButton>
                    {signal.status === 'new' && (
                      <>
                        <IconButton
                          size="small"
                          color="success"
                          onClick={() => handleValidate(signal.id, true)}
                        >
                          <CheckCircle />
                        </IconButton>
                        <IconButton
                          size="small"
                          color="error"
                          onClick={() => handleValidate(signal.id, false)}
                        >
                          <Cancel />
                        </IconButton>
                      </>
                    )}
                  </TableCell>
                </TableRow>
              ))
            )}
          </TableBody>
        </Table>
        <TablePagination
          component="div"
          count={total}
          page={page}
          onPageChange={(_, newPage) => setPage(newPage)}
          rowsPerPage={rowsPerPage}
          onRowsPerPageChange={(e) => {
            setRowsPerPage(parseInt(e.target.value, 10))
            setPage(0)
          }}
          labelRowsPerPage="每页行数"
          labelDisplayedRows={({ from, to, count }) => `${from}-${to} / ${count}`}
        />
      </TableContainer>

      {/* 详情弹窗 */}
      <Dialog
        open={!!selectedSignal}
        onClose={() => setSelectedSignal(null)}
        maxWidth="md"
        fullWidth
      >
        {selectedSignal && (
          <>
            <DialogTitle>信号详情</DialogTitle>
            <DialogContent>
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  信号类型
                </Typography>
                <Chip
                  label={getTypeLabel(selectedSignal.signalType)}
                  color={getTypeColor(selectedSignal.signalType)}
                />
              </Box>
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  信号强度
                </Typography>
                <Chip
                  label={`${STRENGTH_LABELS[selectedSignal.strength]} (${selectedSignal.strength})`}
                  color={getStrengthColor(selectedSignal.strength)}
                />
              </Box>
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  内容
                </Typography>
                <Typography variant="body1" sx={{ whiteSpace: 'pre-wrap' }}>
                  {selectedSignal.content}
                </Typography>
              </Box>
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  平台
                </Typography>
                <Typography variant="body2">
                  {selectedSignal.platform}
                </Typography>
              </Box>
              {selectedSignal.author && (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    作者
                  </Typography>
                  <Typography variant="body2">
                    {selectedSignal.author}
                  </Typography>
                </Box>
              )}
              {selectedSignal.url && (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    链接
                  </Typography>
                  <Typography
                    variant="body2"
                    component="a"
                    href={selectedSignal.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    sx={{ color: 'primary.main' }}
                  >
                    {selectedSignal.url}
                  </Typography>
                </Box>
              )}
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  检测依据
                </Typography>
                <Paper sx={{ p: 1, bgcolor: 'grey.50' }}>
                  <pre style={{ margin: 0, fontSize: '0.75rem' }}>
                    {JSON.stringify(selectedSignal.metadata, null, 2)}
                  </pre>
                </Paper>
              </Box>
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setSelectedSignal(null)}>关闭</Button>
              {selectedSignal.status === 'new' && (
                <>
                  <Button
                    color="error"
                    onClick={() => {
                      handleValidate(selectedSignal.id, false)
                      setSelectedSignal(null)
                    }}
                  >
                    标记无效
                  </Button>
                  <Button
                    variant="contained"
                    onClick={() => {
                      handleValidate(selectedSignal.id, true)
                      setSelectedSignal(null)
                    }}
                  >
                    确认有效
                  </Button>
                </>
              )}
            </DialogActions>
          </>
        )}
      </Dialog>

      {/* 阈值调整弹窗 */}
      <Dialog
        open={thresholdDialogOpen}
        onClose={() => setThresholdDialogOpen(false)}
      >
        <DialogTitle>调整信号检测阈值</DialogTitle>
        <DialogContent>
          <Box sx={{ width: 300, mt: 2 }}>
            <Typography gutterBottom>
              最小信号强度: {threshold}
            </Typography>
            <Slider
              value={threshold}
              onChange={(_, value) => setThreshold(value as number)}
              step={1}
              marks
              min={1}
              max={3}
              valueLabelDisplay="auto"
            />
            <Typography variant="caption" color="text.secondary">
              只检测强度大于等于此值的信号
            </Typography>
          </Box>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setThresholdDialogOpen(false)}>取消</Button>
          <Button
            variant="contained"
            onClick={() => {
              // TODO: 保存阈值配置
              setThresholdDialogOpen(false)
            }}
          >
            保存
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

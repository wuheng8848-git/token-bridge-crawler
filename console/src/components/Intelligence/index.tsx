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
  TextField,
  Typography,
  Paper,
  IconButton,
  Dialog,
  DialogTitle,
  DialogContent,
  DialogActions,
  Button,
  Grid,
} from '@mui/material'
import { Refresh, Visibility, Translate } from '@mui/icons-material'
import { useEffect, useState } from 'react'

interface IntelItem {
  id: string
  intelType: string
  source: string
  title: string
  content: string
  url: string
  metadata: Record<string, any>
  capturedAt: string
  status: string
}

const INTEL_TYPES = [
  { value: '', label: '全部类型' },
  { value: 'price', label: '价格' },
  { value: 'api_doc', label: 'API文档' },
  { value: 'user_pain', label: '用户痛点' },
  { value: 'tool_ecosystem', label: '工具生态' },
  { value: 'community', label: '社区' },
]

const STATUS_OPTIONS = [
  { value: '', label: '全部状态' },
  { value: 'new', label: '新增' },
  { value: 'processed', label: '已处理' },
  { value: 'ignored', label: '已忽略' },
]

export function Intelligence() {
  const [items, setItems] = useState<IntelItem[]>([])
  const [loading, setLoading] = useState(false)
  const [page, setPage] = useState(0)
  const [rowsPerPage, setRowsPerPage] = useState(10)
  const [total, setTotal] = useState(0)
  
  // 筛选条件
  const [filterType, setFilterType] = useState('')
  const [filterStatus, setFilterStatus] = useState('')
  const [filterSource, setFilterSource] = useState('')
  const [filterKeyword, setFilterKeyword] = useState('')
  
  // 详情弹窗
  const [selectedItem, setSelectedItem] = useState<IntelItem | null>(null)

  useEffect(() => {
    fetchData()
  }, [page, rowsPerPage, filterType, filterStatus, filterSource])

  const fetchData = async () => {
    setLoading(true)
    try {
      const params = new URLSearchParams({
        page: String(page + 1),
        perPage: String(rowsPerPage),
        ...(filterType && { type: filterType }),
        ...(filterStatus && { status: filterStatus }),
        ...(filterSource && { source: filterSource }),
        ...(filterKeyword && { keyword: filterKeyword }),
      })
      
      const res = await fetch(`/api/v1/intelligence?${params}`)
      const data = await res.json()
      
      setItems(data.items || [])
      setTotal(data.total || 0)
    } catch (error) {
      console.error('Failed to fetch intelligence:', error)
    } finally {
      setLoading(false)
    }
  }

  const handleRetranslate = async (id: string) => {
    try {
      await fetch(`/api/v1/intelligence/${id}/retranslate`, { method: 'POST' })
      fetchData()
    } catch (error) {
      console.error('Failed to retranslate:', error)
    }
  }

  const getTypeColor = (type: string) => {
    const colors: Record<string, any> = {
      price: 'primary',
      api_doc: 'secondary',
      user_pain: 'error',
      tool_ecosystem: 'success',
      community: 'warning',
    }
    return colors[type] || 'default'
  }

  const getTypeLabel = (type: string) => {
    const found = INTEL_TYPES.find(t => t.value === type)
    return found?.label || type
  }

  return (
    <Box>
      <Typography variant="h4" gutterBottom sx={{ fontWeight: 700 }}>
        情报浏览
      </Typography>

      {/* 筛选栏 */}
      <Card sx={{ mb: 3 }}>
        <CardContent>
          <Grid container spacing={2} alignItems="center">
            <Grid item xs={12} sm={6} md={3}>
              <TextField
                fullWidth
                size="small"
                label="搜索关键词"
                value={filterKeyword}
                onChange={(e) => setFilterKeyword(e.target.value)}
                onKeyPress={(e) => e.key === 'Enter' && fetchData()}
              />
            </Grid>
            <Grid item xs={12} sm={6} md={2}>
              <FormControl fullWidth size="small">
                <InputLabel>类型</InputLabel>
                <Select
                  value={filterType}
                  label="类型"
                  onChange={(e) => setFilterType(e.target.value)}
                >
                  {INTEL_TYPES.map(t => (
                    <MenuItem key={t.value} value={t.value}>{t.label}</MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} sm={6} md={2}>
              <FormControl fullWidth size="small">
                <InputLabel>状态</InputLabel>
                <Select
                  value={filterStatus}
                  label="状态"
                  onChange={(e) => setFilterStatus(e.target.value)}
                >
                  {STATUS_OPTIONS.map(s => (
                    <MenuItem key={s.value} value={s.value}>{s.label}</MenuItem>
                  ))}
                </Select>
              </FormControl>
            </Grid>
            <Grid item xs={12} sm={6} md={3}>
              <TextField
                fullWidth
                size="small"
                label="来源"
                value={filterSource}
                onChange={(e) => setFilterSource(e.target.value)}
              />
            </Grid>
            <Grid item xs={12} sm={6} md={2}>
              <IconButton onClick={fetchData} color="primary">
                <Refresh />
              </IconButton>
            </Grid>
          </Grid>
        </CardContent>
      </Card>

      {/* 数据表格 */}
      <TableContainer component={Paper}>
        <Table>
          <TableHead>
            <TableRow>
              <TableCell>ID</TableCell>
              <TableCell>类型</TableCell>
              <TableCell>来源</TableCell>
              <TableCell>标题</TableCell>
              <TableCell>状态</TableCell>
              <TableCell>采集时间</TableCell>
              <TableCell>操作</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {items.length === 0 ? (
              <TableRow>
                <TableCell colSpan={7} align="center">
                  暂无数据
                </TableCell>
              </TableRow>
            ) : (
              items.map((item) => (
                <TableRow key={item.id} hover>
                  <TableCell sx={{ fontFamily: 'monospace', fontSize: '0.75rem' }}>
                    {item.id.slice(0, 8)}...
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={getTypeLabel(item.intelType)}
                      color={getTypeColor(item.intelType)}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>{item.source}</TableCell>
                  <TableCell sx={{ maxWidth: 300 }}>
                    <Typography noWrap variant="body2">
                      {item.metadata?.title_zh || item.title}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={item.status}
                      color={item.status === 'new' ? 'success' : 'default'}
                      size="small"
                    />
                  </TableCell>
                  <TableCell>
                    {new Date(item.capturedAt).toLocaleString()}
                  </TableCell>
                  <TableCell>
                    <IconButton
                      size="small"
                      onClick={() => setSelectedItem(item)}
                    >
                      <Visibility />
                    </IconButton>
                    <IconButton
                      size="small"
                      onClick={() => handleRetranslate(item.id)}
                      title="重新翻译"
                    >
                      <Translate />
                    </IconButton>
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
        open={!!selectedItem}
        onClose={() => setSelectedItem(null)}
        maxWidth="md"
        fullWidth
      >
        {selectedItem && (
          <>
            <DialogTitle>情报详情</DialogTitle>
            <DialogContent>
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  ID
                </Typography>
                <Typography variant="body2" sx={{ fontFamily: 'monospace' }}>
                  {selectedItem.id}
                </Typography>
              </Box>
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  类型
                </Typography>
                <Chip
                  label={getTypeLabel(selectedItem.intelType)}
                  color={getTypeColor(selectedItem.intelType)}
                  size="small"
                />
              </Box>
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  标题
                </Typography>
                <Typography variant="body1">
                  {selectedItem.metadata?.title_zh || selectedItem.title}
                </Typography>
              </Box>
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  内容
                </Typography>
                <Typography variant="body2" sx={{ whiteSpace: 'pre-wrap' }}>
                  {selectedItem.metadata?.content_zh || selectedItem.content}
                </Typography>
              </Box>
              {selectedItem.url && (
                <Box sx={{ mb: 2 }}>
                  <Typography variant="subtitle2" color="text.secondary">
                    链接
                  </Typography>
                  <Typography
                    variant="body2"
                    component="a"
                    href={selectedItem.url}
                    target="_blank"
                    rel="noopener noreferrer"
                    sx={{ color: 'primary.main' }}
                  >
                    {selectedItem.url}
                  </Typography>
                </Box>
              )}
              <Box sx={{ mb: 2 }}>
                <Typography variant="subtitle2" color="text.secondary">
                  元数据
                </Typography>
                <Paper sx={{ p: 1, bgcolor: 'grey.50' }}>
                  <pre style={{ margin: 0, fontSize: '0.75rem' }}>
                    {JSON.stringify(selectedItem.metadata, null, 2)}
                  </pre>
                </Paper>
              </Box>
            </DialogContent>
            <DialogActions>
              <Button onClick={() => setSelectedItem(null)}>关闭</Button>
              <Button
                onClick={() => handleRetranslate(selectedItem.id)}
                variant="contained"
              >
                重新翻译
              </Button>
            </DialogActions>
          </>
        )}
      </Dialog>
    </Box>
  )
}

import {
  Box,
  Card,
  CardContent,
  Chip,
  Grid,
  Typography,
  Table,
  TableBody,
  TableCell,
  TableContainer,
  TableHead,
  TableRow,
  Paper,
  LinearProgress,
} from '@mui/material'
import { useEffect, useState } from 'react'

import { KPICard } from './KPICard'
import { LineChartCard, BarChartCard, PieChartCard } from './StatsChart'

interface CollectorStats {
  name: string
  type: string
  source: string
  totalItems: number
  recent24h: number
  status: string
}

interface QualityData {
  summary: {
    totalItems: number
    overallRelevance: number
    topKeywords: { keyword: string; count: number }[]
  }
  themeDistribution: Record<string, number>
  sourceAnalysis: Record<string, {
    total: number
    matchedItems: number
    relevanceScore: number
    keywordHits: number
    themes: Record<string, number>
  }>
}

interface DashboardData {
  collectors: {
    name: string
    type: string
    source: string
    rateLimit: string
  }[]
  collectorStats: CollectorStats[]
  stats: {
    total: number
    byType: Record<string, number>
    bySource: Record<string, number>
    collectorRuns: number
  } | null
  translationStats: {
    total: number
    translated: number
  } | null
  qualityData: QualityData | null
  signals: {
    total: number
    byType: Record<string, number>
    highValue: number  // migration + competitor
  } | null
}

export function Dashboard() {
  const [data, setData] = useState<DashboardData>({
    collectors: [],
    collectorStats: [],
    stats: null,
    translationStats: null,
    qualityData: null,
    signals: null,
  })
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    const fetchData = async () => {
      try {
        // 获取采集器列表
        const collectorsRes = await fetch('/api/v1/collectors')
        const collectorsData = await collectorsRes.json()

        // 获取统计信息
        const statsRes = await fetch('/api/v1/stats/intelligence')
        const statsData = await statsRes.json()

        // 获取采集器产出统计
        const collectorStatsRes = await fetch('/api/v1/stats/collectors')
        const collectorStatsData = await collectorStatsRes.json()

        // 获取质量分析
        const qualityRes = await fetch('/api/v1/stats/quality?limit=100')
        const qualityData = await qualityRes.json()

        // 获取信号统计（从情报数据中计算）
        // 获取最近5000条情报用于信号统计
        const signalsRes = await fetch('/api/v1/intelligence?limit=5000&offset=0')
        const signalsData = await signalsRes.json()
        const signalItems = signalsData.items || []

        // 计算信号分布
        const signalByType: Record<string, number> = {}
        let highValueCount = 0

        signalItems.forEach((item: any) => {
          const signalType = item.metadata?.signal_type || item.signal_type
          if (signalType) {
            signalByType[signalType] = (signalByType[signalType] || 0) + 1
            // 高价值信号：migration + competitor
            if (signalType === 'migration' || signalType === 'competitor') {
              highValueCount++
            }
          }
        })

        setData({
          collectors: collectorsData.items || [],
          collectorStats: collectorStatsData.collectors || [],
          stats: statsData,
          translationStats: collectorStatsData.translationStats || null,
          qualityData: qualityData || null,
          signals: {
            total: Object.values(signalByType).reduce((a, b) => a + b, 0),
            byType: signalByType,
            highValue: highValueCount,
          },
        })
      } catch (error) {
        console.error('Failed to fetch dashboard data:', error)
      } finally {
        setLoading(false)
      }
    }

    fetchData()
    // 每30秒刷新一次
    const interval = setInterval(fetchData, 30000)
    return () => clearInterval(interval)
  }, [])

  return (
    <Box>
      <Typography variant="h4" gutterBottom sx={{ fontWeight: 700 }}>
        监控大盘
      </Typography>

      {/* KPI 卡片 */}
      <Grid container spacing={2} sx={{ mb: 3 }}>
        <Grid item xs={12} sm={6} md={3}>
          <KPICard
            title="采集器数量"
            value={data.collectors.length.toString()}
            loading={loading}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <KPICard
            title="情报总数"
            value={data.stats?.total?.toLocaleString() || '—'}
            loading={loading}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <KPICard
            title="高价值信号"
            value={data.signals?.highValue?.toString() || '—'}
            color="warning"
            loading={loading}
            subtitle="迁移意愿+竞品动态"
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <KPICard
            title="信号总数"
            value={data.signals?.total?.toString() || '—'}
            color="primary"
            loading={loading}
          />
        </Grid>
      </Grid>

      {/* 采集器列表 */}
      <Typography variant="h6" gutterBottom sx={{ mt: 4, fontWeight: 600 }}>
        采集器状态
      </Typography>
      <Grid container spacing={2}>
        {data.collectors.map((collector) => (
          <Grid item xs={12} sm={6} md={4} key={collector.name}>
            <Card>
              <CardContent>
                <Box
                  sx={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'flex-start',
                  }}
                >
                  <Box>
                    <Typography variant="h6" sx={{ fontWeight: 600 }}>
                      {collector.name}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      类型: {collector.type}
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      来源: {collector.source}
                    </Typography>
                  </Box>
                  <Chip label="正常" color="success" size="small" />
                </Box>
              </CardContent>
            </Card>
          </Grid>
        ))}
      </Grid>

      {/* 采集器产出统计表格 */}
      <Typography variant="h6" gutterBottom sx={{ mt: 4, fontWeight: 600 }}>
        采集器产出统计
      </Typography>
      <TableContainer component={Paper} sx={{ mb: 3 }}>
        <Table size="small">
          <TableHead>
            <TableRow>
              <TableCell>采集器</TableCell>
              <TableCell>类型</TableCell>
              <TableCell>来源</TableCell>
              <TableCell align="right">总产出</TableCell>
              <TableCell align="right">24小时</TableCell>
              <TableCell>状态</TableCell>
            </TableRow>
          </TableHead>
          <TableBody>
            {data.collectorStats
              .sort((a, b) => b.totalItems - a.totalItems)
              .map((collector) => (
                <TableRow key={collector.name} hover>
                  <TableCell sx={{ fontWeight: 500 }}>{collector.name}</TableCell>
                  <TableCell>
                    <Chip label={collector.type} size="small" variant="outlined" />
                  </TableCell>
                  <TableCell>{collector.source}</TableCell>
                  <TableCell align="right">
                    <Typography fontWeight={600} color={collector.totalItems > 0 ? 'primary' : 'text.disabled'}>
                      {collector.totalItems.toLocaleString()}
                    </Typography>
                  </TableCell>
                  <TableCell align="right">
                    <Typography color={collector.recent24h > 0 ? 'success.main' : 'text.disabled'}>
                      +{collector.recent24h}
                    </Typography>
                  </TableCell>
                  <TableCell>
                    <Chip
                      label={collector.totalItems > 0 ? '有产出' : '待采集'}
                      color={collector.totalItems > 0 ? 'success' : 'default'}
                      size="small"
                    />
                  </TableCell>
                </TableRow>
              ))}
          </TableBody>
        </Table>
      </TableContainer>

      {/* 翻译覆盖率 */}
      {data.translationStats && (
        <Card sx={{ mb: 3 }}>
          <CardContent>
            <Typography variant="subtitle1" gutterBottom sx={{ fontWeight: 600 }}>
              翻译覆盖率
            </Typography>
            <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
              <Box sx={{ flexGrow: 1 }}>
                <LinearProgress
                  variant="determinate"
                  value={(data.translationStats.translated / data.translationStats.total) * 100}
                  sx={{ height: 10, borderRadius: 5 }}
                />
              </Box>
              <Typography variant="body2" color="text.secondary">
                {data.translationStats.translated.toLocaleString()} / {data.translationStats.total.toLocaleString()}
                {' '}({((data.translationStats.translated / data.translationStats.total) * 100).toFixed(1)}%)
              </Typography>
            </Box>
          </CardContent>
        </Card>
      )}

      {/* 情报质量分析 */}
      {data.qualityData && (
        <>
          <Typography variant="h6" gutterBottom sx={{ mt: 4, fontWeight: 600 }}>
            情报质量分析
          </Typography>
          <Grid container spacing={2}>
            {/* 相关性得分 */}
            <Grid item xs={12} md={4}>
              <Card sx={{ height: '100%' }}>
                <CardContent>
                  <Typography variant="body2" color="text.secondary" gutterBottom>
                    整体相关性
                  </Typography>
                  <Box sx={{ display: 'flex', alignItems: 'center', gap: 2 }}>
                    <Typography variant="h3" sx={{ fontWeight: 700, color: 'success.main' }}>
                      {data.qualityData.summary.overallRelevance.toFixed(0)}%
                    </Typography>
                    <Typography variant="body2" color="text.secondary">
                      与项目焦点匹配
                    </Typography>
                  </Box>
                  <LinearProgress
                    variant="determinate"
                    value={data.qualityData.summary.overallRelevance}
                    color="success"
                    sx={{ mt: 2, height: 8, borderRadius: 4 }}
                  />
                </CardContent>
              </Card>
            </Grid>

            {/* 热门关键词 */}
            <Grid item xs={12} md={4}>
              <Card sx={{ height: '100%' }}>
                <CardContent>
                  <Typography variant="body2" color="text.secondary" gutterBottom>
                    热门关键词
                  </Typography>
                  <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, mt: 1 }}>
                    {data.qualityData.summary.topKeywords.slice(0, 8).map((kw) => (
                      <Chip
                        key={kw.keyword}
                        label={`${kw.keyword} (${kw.count})`}
                        size="small"
                        variant="outlined"
                        color="primary"
                      />
                    ))}
                  </Box>
                </CardContent>
              </Card>
            </Grid>

            {/* 主题分布 */}
            <Grid item xs={12} md={4}>
              <Card sx={{ height: '100%' }}>
                <CardContent>
                  <Typography variant="body2" color="text.secondary" gutterBottom>
                    主题分布
                  </Typography>
                  <Box sx={{ mt: 1 }}>
                    {Object.entries(data.qualityData.themeDistribution)
                      .sort(([, a], [, b]) => b - a)
                      .slice(0, 5)
                      .map(([theme, count]) => (
                        <Box key={theme} sx={{ display: 'flex', justifyContent: 'space-between', mb: 0.5 }}>
                          <Typography variant="body2">{theme}</Typography>
                          <Typography variant="body2" fontWeight={600}>{count}</Typography>
                        </Box>
                      ))}
                  </Box>
                </CardContent>
              </Card>
            </Grid>
          </Grid>

          {/* 来源质量对比 */}
          <Typography variant="subtitle1" gutterBottom sx={{ mt: 3, fontWeight: 600 }}>
            来源质量对比
          </Typography>
          <TableContainer component={Paper} sx={{ mb: 3 }}>
            <Table size="small">
              <TableHead>
                <TableRow>
                  <TableCell>来源</TableCell>
                  <TableCell align="right">样本数</TableCell>
                  <TableCell align="right">关键词命中</TableCell>
                  <TableCell align="right">相关性</TableCell>
                  <TableCell>主要主题</TableCell>
                </TableRow>
              </TableHead>
              <TableBody>
                {Object.entries(data.qualityData.sourceAnalysis)
                  .sort(([, a], [, b]) => b.relevanceScore - a.relevanceScore)
                  .map(([source, analysis]) => (
                    <TableRow key={source} hover>
                      <TableCell sx={{ fontWeight: 500 }}>{source}</TableCell>
                      <TableCell align="right">{analysis.total}</TableCell>
                      <TableCell align="right">{analysis.keywordHits}</TableCell>
                      <TableCell align="right">
                        <Chip
                          label={`${analysis.relevanceScore.toFixed(0)}%`}
                          size="small"
                          color={analysis.relevanceScore >= 80 ? 'success' : analysis.relevanceScore >= 50 ? 'warning' : 'error'}
                        />
                      </TableCell>
                      <TableCell>
                        <Box sx={{ display: 'flex', gap: 0.5, flexWrap: 'wrap' }}>
                          {Object.entries(analysis.themes)
                            .sort(([, a], [, b]) => b - a)
                            .slice(0, 3)
                            .map(([theme, count]) => (
                              <Chip key={theme} label={`${theme}:${count}`} size="small" variant="outlined" />
                            ))}
                        </Box>
                      </TableCell>
                    </TableRow>
                  ))}
              </TableBody>
            </Table>
          </TableContainer>
        </>
      )}

      {/* 情报类型统计 */}
      {data.stats?.byType && (
        <>
          <Typography variant="h6" gutterBottom sx={{ mt: 4, fontWeight: 600 }}>
            情报类型分布
          </Typography>
          <Grid container spacing={2}>
            {Object.entries(data.stats.byType).map(([type, count]) => (
              <Grid item xs={12} sm={6} md={3} key={type}>
                <Card>
                  <CardContent>
                    <Typography variant="body2" color="text.secondary">
                      {type}
                    </Typography>
                    <Typography variant="h4" sx={{ fontWeight: 700 }}>
                      {count.toLocaleString()}
                    </Typography>
                  </CardContent>
                </Card>
              </Grid>
            ))}
          </Grid>
        </>
      )}

      {/* 信号类型分布 */}
      {data.signals && data.signals.total > 0 && (
        <>
          <Typography variant="h6" gutterBottom sx={{ mt: 4, fontWeight: 600 }}>
            信号类型分布
          </Typography>
          <Grid container spacing={2}>
            {Object.entries(data.signals.byType)
              .sort(([, a], [, b]) => b - a)
              .map(([type, count]) => {
                const isHighValue = type === 'migration' || type === 'competitor'
                return (
                  <Grid item xs={12} sm={6} md={3} key={type}>
                    <Card sx={{ bgcolor: isHighValue ? 'warning.light' : 'background.paper' }}>
                      <CardContent>
                        <Box sx={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                          <Box>
                            <Typography variant="body2" color="text.secondary">
                              {type}
                              {isHighValue && (
                                <Chip label="高价值" size="small" color="warning" sx={{ ml: 1 }} />
                              )}
                            </Typography>
                            <Typography variant="h4" sx={{ fontWeight: 700, color: isHighValue ? 'warning.dark' : 'primary.main' }}>
                              {count}
                            </Typography>
                          </Box>
                        </Box>
                      </CardContent>
                    </Card>
                  </Grid>
                )
              })}
          </Grid>
        </>
      )}

      {/* 图表区域 */}
      <Typography variant="h6" gutterBottom sx={{ mt: 4, fontWeight: 600 }}>
        数据可视化
      </Typography>
      <Grid container spacing={2}>
        <Grid item xs={12} md={4}>
          <PieChartCard
            title="情报类型占比"
            data={Object.entries(data.stats?.byType || {}).map(([name, value]) => ({
              name,
              value: Number(value),
            }))}
          />
        </Grid>
        <Grid item xs={12} md={4}>
          <BarChartCard
            title="来源分布"
            data={Object.entries(data.stats?.bySource || {}).map(([name, value]) => ({
              name,
              value: Number(value),
            }))}
          />
        </Grid>
        <Grid item xs={12} md={4}>
          <LineChartCard
            title="采集趋势（24小时）"
            data={[
              { time: '00:00', value: 12 },
              { time: '04:00', value: 8 },
              { time: '08:00', value: 25 },
              { time: '12:00', value: 18 },
              { time: '16:00', value: 32 },
              { time: '20:00', value: 15 },
            ]}
          />
        </Grid>
      </Grid>
    </Box>
  )
}

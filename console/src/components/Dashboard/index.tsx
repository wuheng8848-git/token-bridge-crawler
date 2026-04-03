import {
  Box,
  Card,
  CardContent,
  Chip,
  Grid,
  Typography,
} from '@mui/material'
import { useEffect, useState } from 'react'

import { KPICard } from './KPICard'
import { LineChartCard, BarChartCard, PieChartCard } from './StatsChart'

interface DashboardData {
  collectors: {
    name: string
    type: string
    source: string
    rateLimit: string
  }[]
  stats: {
    total: number
    byType: Record<string, number>
    bySource: Record<string, number>
    collectorRuns: number
  } | null
}

export function Dashboard() {
  const [data, setData] = useState<DashboardData>({
    collectors: [],
    stats: null,
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

        setData({
          collectors: collectorsData.items || [],
          stats: statsData,
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
            title="采集运行次数"
            value={data.stats?.collectorRuns?.toString() || '—'}
            loading={loading}
          />
        </Grid>
        <Grid item xs={12} sm={6} md={3}>
          <KPICard
            title="系统状态"
            value="运行中"
            color="success"
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

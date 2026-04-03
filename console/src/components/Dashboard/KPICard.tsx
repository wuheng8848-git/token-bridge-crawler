import { Card, CardContent, Skeleton, Typography } from '@mui/material'

interface KPICardProps {
  title: string
  value: string
  subtitle?: string
  color?: 'primary' | 'success' | 'warning' | 'error'
  loading?: boolean
}

export function KPICard({
  title,
  value,
  subtitle,
  color = 'primary',
  loading = false,
}: KPICardProps) {
  return (
    <Card sx={{ height: '100%' }}>
      <CardContent>
        <Typography variant="body2" color="text.secondary" gutterBottom>
          {title}
        </Typography>
        {loading ? (
          <Skeleton variant="text" width="60%" height={40} />
        ) : (
          <Typography variant="h4" sx={{ fontWeight: 700 }} color={color}>
            {value}
          </Typography>
        )}
        {subtitle && (
          <Typography variant="caption" color="text.secondary">
            {subtitle}
          </Typography>
        )}
      </CardContent>
    </Card>
  )
}

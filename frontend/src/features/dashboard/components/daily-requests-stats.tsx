'use client'

import { useTranslation } from 'react-i18next'
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, XAxis, YAxis, Tooltip } from 'recharts'
import { formatNumber } from '@/utils/format-number'
import { Skeleton } from '@/components/ui/skeleton'
import { useDailyRequestStats } from '../data/dashboard'

export function DailyRequestStats() {
  const { t } = useTranslation()
  const { data: dailyStats, isLoading, error } = useDailyRequestStats(30)

  if (isLoading) {
    return (
      <div className='flex h-[350px] items-center justify-center'>
        <Skeleton className='h-full w-full' />
      </div>
    )
  }

  if (error) {
    return (
      <div className='flex h-[350px] items-center justify-center text-red-500'>
        {t('dashboard.charts.errorLoadingChart')} {error.message}
      </div>
    )
  }

  // Transform data for the chart
  const chartData =
    dailyStats?.map((stat) => ({
      name: new Date(stat.date).toLocaleDateString('en-US', {
        month: 'short',
        day: 'numeric',
      }),
      display: formatNumber(stat.count),
      value: stat.count,
    })) || []

  // Calculate dynamic YAxis max value (max daily count, rounded up to nearest integer)
  const maxCount = Math.max(...chartData.map((d) => d.value), 1)
  const yAxisMax = Math.ceil(maxCount)

  return (
    <ResponsiveContainer width='100%' height={350}>
      <BarChart data={chartData}>
        <CartesianGrid strokeDasharray='3 3' />
        <XAxis dataKey='name' stroke='#888888' fontSize={12} tickLine={false} axisLine={false} />
        <YAxis stroke='#888888' fontSize={12} tickLine={false} axisLine={false} tickFormatter={(value) => `${value}`} domain={[0, yAxisMax]} />
        <Tooltip formatter={(value: number) => [formatNumber(value), '']} labelStyle={{ color: 'var(--foreground)' }} />
        <Bar dataKey='value' fill='var(--chart-1)' radius={[4, 4, 0, 0]} />
      </BarChart>
    </ResponsiveContainer>
  )
}

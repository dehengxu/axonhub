'use client'

import { useTranslation } from 'react-i18next'
import { Bar, BarChart, CartesianGrid, ResponsiveContainer, XAxis, YAxis, Tooltip } from 'recharts'
import { formatNumber } from '@/utils/format-number'
import { Skeleton } from '@/components/ui/skeleton'
import { ModelTokenTrend } from '../data/dashboard'

interface DailyTokensChartProps {
  trends: ModelTokenTrend[]
  isLoading?: boolean
}

export function DailyTokensChart({ trends, isLoading }: DailyTokensChartProps) {
  const { t } = useTranslation()

  if (isLoading) {
    return (
      <div className='flex h-[200px] items-center justify-center'>
        <Skeleton className='h-full w-full' />
      </div>
    )
  }

  if (!trends || trends.length === 0) {
    return (
      <div className='flex h-[200px] items-center justify-center'>
        <div className='text-muted-foreground text-sm'>{t('dashboard.charts.noTokenData')}</div>
      </div>
    )
  }

  // 按日期聚合数据，计算每天的总token数
  const dailyTotals = trends.reduce((acc, trend) => {
    const existing = acc.find(item => item.date === trend.date)
    if (existing) {
      existing.totalTokens += trend.totalTokens
    } else {
      acc.push({
        date: trend.date,
        totalTokens: trend.totalTokens,
        formattedDate: new Date(trend.date).toLocaleDateString('en-US', {
          month: 'short',
          day: 'numeric',
        })
      })
    }
    return acc
  }, [] as Array<{ date: string; totalTokens: number; formattedDate: string }>)

  // 按日期排序
  const chartData = dailyTotals.sort((a, b) => new Date(a.date).getTime() - new Date(b.date).getTime())

  return (
    <ResponsiveContainer width='100%' height={200}>
      <BarChart data={chartData} margin={{ top: 5, right: 5, left: 5, bottom: 5 }}>
        <CartesianGrid strokeDasharray='3 3' stroke='var(--border)' />
        <XAxis 
          dataKey='formattedDate' 
          tick={{ fontSize: 12, fill: 'var(--muted-foreground)' }}
          tickLine={false} 
          axisLine={false} 
        />
        <YAxis 
          tick={{ fontSize: 12, fill: 'var(--muted-foreground)' }}
          tickLine={false} 
          axisLine={false}
          tickFormatter={(value) => formatNumber(value)}
        />
        <Tooltip 
          formatter={(value: number) => [formatNumber(value), t('dashboard.stats.totalTokens')]}
          labelStyle={{ color: 'var(--foreground)' }}
          contentStyle={{
            backgroundColor: 'var(--background)',
            border: '1px solid var(--border)',
            borderRadius: '6px'
          }}
        />
        <Bar 
          dataKey='totalTokens' 
          fill='var(--chart-1)' 
          radius={[4, 4, 0, 0]} 
        />
      </BarChart>
    </ResponsiveContainer>
  )
}

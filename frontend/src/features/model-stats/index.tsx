import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { RefreshCw, Download, BarChart3 } from 'lucide-react'

import { formatNumber } from '@/utils/format-number'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Button } from '@/components/ui/button'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { useModelTokenStats, useRequestsByModel } from '../dashboard/data/dashboard'
import { ModelTokenChart } from '../dashboard/components/model-token-chart'
import { ModelTokenTable } from '../dashboard/components/model-token-table'
import { DailyTokensChart } from '../dashboard/components/daily-tokens-chart'

export default function ModelStatsPage() {
  const { t } = useTranslation()
  const [period, setPeriod] = useState<'day' | 'week' | 'month'>('day')
  const [isRefreshing, setIsRefreshing] = useState(false)

  // Get available models from request stats
  const { data: modelStats } = useRequestsByModel()
  const availableModels = modelStats?.map(stat => stat.modelId) || []

  const { data: stats, isLoading, error, refetch } = useModelTokenStats(
    availableModels.length > 0 ? availableModels : undefined,
    period,
    undefined
  )

  // Auto refresh functionality
  useEffect(() => {
    const interval = setInterval(() => {
      refetch()
    }, 30000) // Refresh every 30 seconds

    return () => clearInterval(interval)
  }, [refetch])

  const handleRefresh = async () => {
    setIsRefreshing(true)
    await refetch()
    setIsRefreshing(false)
  }

  const handleExportCSV = () => {
    if (!stats?.currentPeriod || stats.currentPeriod.length === 0) return

    const headers = ['Model', 'Period', 'Date', 'Input Tokens', 'Output Tokens', 'Cached Tokens', 'Total Tokens']
    const rows = stats.currentPeriod.map(stat => [
      stat.modelId,
      stat.period,
      stat.date,
      stat.totalInputTokens.toString(),
      stat.totalOutputTokens.toString(),
      stat.totalCachedTokens.toString(),
      stat.totalTokens.toString()
    ])

    const csvContent = [headers, ...rows].map(row => row.join(',')).join('\n')
    const blob = new Blob([csvContent], { type: 'text/csv' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `model-token-stats-${period}-${new Date().toISOString().split('T')[0]}.csv`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
  }

  if (isLoading) {
    return (
      <div className='flex-1 space-y-6 p-8 pt-6'>
        <Header />
        <div className='space-y-4'>
          <Skeleton className='h-8 w-[200px]' />
          <Skeleton className='h-10 w-[300px]' />
          <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-4'>
            <Skeleton className='h-[100px]' />
            <Skeleton className='h-[100px]' />
            <Skeleton className='h-[100px]' />
            <Skeleton className='h-[100px]' />
          </div>
          <Skeleton className='h-[300px]' />
          <Skeleton className='h-[400px]' />
        </div>
      </div>
    )
  }

  if (error) {
    return (
      <div className='flex-1 space-y-6 p-8 pt-6'>
        <Header />
        <div className='text-red-500'>
          {t('common.loadError')} {error.message}
        </div>
      </div>
    )
  }

  return (
    <div className='flex-1 space-y-6 p-8 pt-6'>
      <Header />

      {/* Page Header */}
      <div className='flex flex-wrap items-center justify-between gap-4'>
        <div>
          <h2 className='text-2xl font-bold tracking-tight'>{t('modelStats.title')}</h2>
          <p className='text-muted-foreground'>{t('modelStats.description')}</p>
        </div>
        <div className='flex items-center gap-2'>
          <Select value={period} onValueChange={(value) => setPeriod(value as 'day' | 'week' | 'month')}>
            <SelectTrigger className='w-[120px]'>
              <SelectValue placeholder={t('dashboard.stats.selectPeriod')} />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value='day'>{t('dashboard.stats.day')}</SelectItem>
              <SelectItem value='week'>{t('dashboard.stats.week')}</SelectItem>
              <SelectItem value='month'>{t('dashboard.stats.month')}</SelectItem>
            </SelectContent>
          </Select>
          <Button
            variant='outline'
            onClick={handleRefresh}
            disabled={isRefreshing}
          >
            <RefreshCw className={`h-4 w-4 mr-2 ${isRefreshing ? 'animate-spin' : ''}`} />
            {t('common.refresh')}
          </Button>
          <Button
            variant='outline'
            onClick={handleExportCSV}
            disabled={!stats?.currentPeriod || stats.currentPeriod.length === 0}
          >
            <Download className='h-4 w-4 mr-2' />
            {t('common.export')}
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <div className='grid gap-4 md:grid-cols-2 lg:grid-cols-4'>
        {stats?.currentPeriod?.slice(0, 4).map((stat) => (
          <Card key={stat.modelId}>
            <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
              <CardTitle className='text-sm font-medium'>{stat.modelId}</CardTitle>
              <BarChart3 className='h-4 w-4 text-muted-foreground' />
            </CardHeader>
            <CardContent>
              <div className='text-2xl font-bold'>{formatNumber(stat.totalTokens)}</div>
              <p className='text-xs text-muted-foreground'>
                {formatNumber(stat.totalInputTokens)} in • {formatNumber(stat.totalOutputTokens)} out
              </p>
            </CardContent>
          </Card>
        ))}
        {(!stats?.currentPeriod || stats.currentPeriod.length === 0) && (
          <Card className='col-span-4'>
            <CardContent className='flex h-[100px] items-center justify-center'>
              <p className='text-muted-foreground'>{t('dashboard.stats.noDataAvailable')}</p>
            </CardContent>
          </Card>
        )}
      </div>

      {/* Daily Token Consumption Chart */}
      <Card>
        <CardHeader>
          <CardTitle>{t('dashboard.stats.dailyTokenConsumption')}</CardTitle>
        </CardHeader>
        <CardContent>
          <DailyTokensChart trends={stats?.trends?.trends || []} isLoading={isLoading} />
        </CardContent>
      </Card>

      {/* Model Token Trends */}
      <Card>
        <CardHeader>
          <CardTitle>{t('modelStats.tokenTrends')}</CardTitle>
        </CardHeader>
        <CardContent>
          {stats && stats.currentPeriod.length > 0 ? (
            <ModelTokenChart
              trends={stats.trends.trends}
              models={stats.trends.models}
              dates={stats.trends.dates}
            />
          ) : (
            <div className='flex h-[300px] items-center justify-center'>
              <p className='text-muted-foreground'>{t('dashboard.charts.noTrendData')}</p>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

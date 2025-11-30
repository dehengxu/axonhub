import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { BarChart4, TrendingUp, ArrowUpDown } from 'lucide-react'

import { formatNumber } from '@/utils/format-number'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { Alert, AlertTitle } from '@/components/ui/alert'
import { useTokenStats, useModelTokenStats, useRequestsByModel } from '../data/dashboard'
import { ModelTokenChart } from './model-token-chart'
import { ModelTokenTable } from './model-token-table'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'

export function TokenStatsCard() {
  const { t } = useTranslation()
  const { data: stats, isLoading, error } = useTokenStats()
  const { data: modelStats } = useRequestsByModel()
  const [selectedModels, setSelectedModels] = useState<string[]>([])
  const [showModelDetails, setShowModelDetails] = useState(false)
  const [modelSortOrder, setModelSortOrder] = useState<'asc' | 'desc'>('desc')

  // Get available models from model stats
  const availableModels = modelStats?.map(stat => stat.modelId) || []

  // Get detailed model stats for all available models
  const { data: detailedModelStats, isLoading: isLoadingModelStats } = useModelTokenStats(
    selectedModels.length > 0 ? selectedModels : availableModels,
    'day'
  )

  // Calculate total consumption across all models
  const totals = useMemo(() => {
    if (!detailedModelStats?.modelData || detailedModelStats.modelData.length === 0) {
      return null;
    }
    return detailedModelStats.modelData.reduce(
      (acc, curr) => ({
        totalRequests: acc.totalRequests + curr.count,
        totalPromptTokens: acc.totalPromptTokens + curr.promptTokens,
        totalCompletionTokens: acc.totalCompletionTokens + curr.completionTokens,
        totalTokens: acc.totalTokens + curr.totalTokens
      }),
      { totalRequests: 0, totalPromptTokens: 0, totalCompletionTokens: 0, totalTokens: 0 }
    );
  }, [detailedModelStats])

  if (isLoading) {
    return (
      <Card>
        <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
          <Skeleton className='h-4 w-[120px]' />
          <Skeleton className='h-4 w-4' />
        </CardHeader>
        <CardContent>
          <Skeleton className='mb-2 h-8 w-[80px]' />
          <Skeleton className='h-4 w-[140px]' />
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
          <CardTitle className='text-sm font-medium'>{t('dashboard.cards.tokenStats')}</CardTitle>
        </CardHeader>
        <CardContent>
          <div className='text-sm text-red-500'>{t('common.loadError')}</div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
        <div className='flex items-center gap-2'>
          <div className='bg-primary/10 text-primary flex h-9 w-9 items-center justify-center rounded-full dark:bg-primary/20'>
            <BarChart4 className='h-4 w-4' />
          </div>
          <CardTitle className='text-sm font-medium'>{t('dashboard.cards.tokensByTime')}</CardTitle>
        </div>

        {availableModels.length > 0 && (
          <Dialog open={showModelDetails} onOpenChange={setShowModelDetails}>
            <DialogTrigger asChild>
              <Button variant='ghost' size='sm' className='h-8 gap-1'>
                <TrendingUp className='h-3 w-3' />
                {t('dashboard.stats.modelDetails')}
              </Button>
            </DialogTrigger>
            <DialogContent className='sm:max-w-[90vw] max-w-[95vw] max-h-[90vh] overflow-y-auto'>
              <DialogHeader>
                <DialogTitle>{t('dashboard.stats.modelTokenStats')}</DialogTitle>
                <DialogDescription>
                  {t('dashboard.stats.detailedModelTokenConsumption')}
                </DialogDescription>
              </DialogHeader>

              {isLoadingModelStats ? null : totals ? (
                <Alert className='mt-4'>
                  <AlertTitle>{t('dashboard.stats.totalConsumption')}</AlertTitle>
                  <div className='grid grid-cols-3 gap-4 mt-2'>
                    <div>
                      <div className='text-sm text-muted-foreground'>{t('dashboard.stats.totalRequests')}</div>
                      <div className='text-xl font-bold'>{formatNumber(totals.totalRequests)}</div>
                    </div>
                    <div>
                      <div className='text-sm text-muted-foreground'>{t('dashboard.stats.totalPromptTokens')}</div>
                      <div className='text-xl font-bold'>{formatNumber(totals.totalPromptTokens)}</div>
                    </div>
                    <div>
                      <div className='text-sm text-muted-foreground'>{t('dashboard.stats.totalTokens')}</div>
                      <div className='text-xl font-bold'>{formatNumber(totals.totalTokens)}</div>
                    </div>
                  </div>
                </Alert>
              ) : null}

              {isLoadingModelStats ? (
                <div className='flex items-center justify-center h-64'>
                  <Skeleton className='h-8 w-32' />
                </div>
              ) : detailedModelStats ? (
                <>
                  {/* Sort controls */}
                  <div className='flex items-center justify-end gap-2'>
                    <span className='text-sm text-muted-foreground'>{t('common.sort.label')}:</span>
                    <Button
                      variant='outline'
                      size='sm'
                      onClick={() => setModelSortOrder(modelSortOrder === 'asc' ? 'desc' : 'asc')}
                      className='gap-2'
                    >
                      <ArrowUpDown className={`h-3 w-3 ${modelSortOrder === 'asc' ? 'rotate-180' : ''}`} />
                      {modelSortOrder === 'desc' ? t('common.sort.desc') : t('common.sort.asc')}
                    </Button>
                  </div>

                  <div className='space-y-6'>
                    <div className='grid grid-cols-2 md:grid-cols-4 gap-4'>
                      {detailedModelStats.currentPeriod
                        .slice()
                        .sort((a, b) => {
                          const comparison = a.totalTokens - b.totalTokens;
                          return modelSortOrder === 'asc' ? comparison : -comparison;
                        })
                        .map((stat) => (
                          <div key={stat.modelId} className='rounded-lg border p-3'>
                            <div className='text-sm font-medium text-muted-foreground mb-1'>{stat.modelId}</div>
                            <div className='text-2xl font-bold'>{formatNumber(stat.totalTokens)}</div>
                            <div className='text-xs text-muted-foreground mt-1'>
                              {formatNumber(stat.totalInputTokens)} in • {formatNumber(stat.totalOutputTokens)} out
                            </div>
                          </div>
                        ))}
                    </div>

                    <div className='space-y-4'>
                      <h4 className='text-sm font-medium'>{t('dashboard.stats.tokenTrends')}</h4>
                      <ModelTokenChart
                        trends={detailedModelStats.trends.trends}
                        models={detailedModelStats.trends.models.slice().sort((a, b) => {
                          const modelA = detailedModelStats.currentPeriod.find(m => m.modelId === a);
                          const modelB = detailedModelStats.currentPeriod.find(m => m.modelId === b);
                          const tokensA = modelA?.totalTokens || 0;
                          const tokensB = modelB?.totalTokens || 0;
                          const comparison = tokensA - tokensB;
                          return modelSortOrder === 'asc' ? comparison : -comparison;
                        })}
                        dates={detailedModelStats.trends.dates}
                      />
                    </div>
                  </div>
                </>
              ) : (
                <div className='text-center py-8 text-muted-foreground'>
                  {t('dashboard.stats.noModelData')}
                </div>
              )}
            </DialogContent>
          </Dialog>
        )}
      </CardHeader>
      <CardContent>
        <div className='space-y-3'>
          {/* This month row */}
          <div className='flex items-center justify-between'>
            <span className='text-sm'>{t('dashboard.stats.thisMonth')}:</span>
            <div className='grid grid-cols-3 gap-3 text-xs'>
              <div className='flex flex-col items-center min-w-[3rem]'>
                <span className='text-muted-foreground text-center'>{t('dashboard.stats.input')}</span>
                <span className='font-semibold text-center'>{formatNumber(stats?.totalInputTokensThisMonth || 0)}</span>
              </div>
              <div className='flex flex-col items-center min-w-[3rem]'>
                <span className='text-muted-foreground text-center'>{t('dashboard.stats.output')}</span>
                <span className='font-semibold text-center'>{formatNumber(stats?.totalOutputTokensThisMonth || 0)}</span>
              </div>
              <div className='flex flex-col items-center min-w-[3rem]'>
                <span className='text-muted-foreground text-center'>{t('dashboard.stats.cached')}</span>
                <span className='font-semibold text-center'>{formatNumber(stats?.totalCachedTokensThisMonth || 0)}</span>
              </div>
            </div>
          </div>

          {/* This week row */}
          <div className='flex items-center justify-between'>
            <span className='text-sm'>{t('dashboard.stats.thisWeek')}:</span>
            <div className='grid grid-cols-3 gap-3 text-xs'>
              <div className='flex flex-col items-center min-w-[3rem]'>
                <span className='text-muted-foreground text-center'>{t('dashboard.stats.input')}</span>
                <span className='font-semibold text-center'>{formatNumber(stats?.totalInputTokensThisWeek || 0)}</span>
              </div>
              <div className='flex flex-col items-center min-w-[3rem]'>
                <span className='text-muted-foreground text-center'>{t('dashboard.stats.output')}</span>
                <span className='font-semibold text-center'>{formatNumber(stats?.totalOutputTokensThisWeek || 0)}</span>
              </div>
              <div className='flex flex-col items-center min-w-[3rem]'>
                <span className='text-muted-foreground text-center'>{t('dashboard.stats.cached')}</span>
                <span className='font-semibold text-center'>{formatNumber(stats?.totalCachedTokensThisWeek || 0)}</span>
              </div>
            </div>
          </div>

          {/* Today row */}
          <div className='flex items-center justify-between'>
            <span className='text-sm'>{t('dashboard.stats.today')}:</span>
            <div className='grid grid-cols-3 gap-3 text-xs'>
              <div className='flex flex-col items-center min-w-[3rem]'>
                <span className='text-muted-foreground text-center'>{t('dashboard.stats.input')}</span>
                <span className='font-semibold text-center'>{formatNumber(stats?.totalInputTokensToday || 0)}</span>
              </div>
              <div className='flex flex-col items-center min-w-[3rem]'>
                <span className='text-muted-foreground text-center'>{t('dashboard.stats.output')}</span>
                <span className='font-semibold text-center'>{formatNumber(stats?.totalOutputTokensToday || 0)}</span>
              </div>
              <div className='flex flex-col items-center min-w-[3rem]'>
                <span className='text-muted-foreground text-center'>{t('dashboard.stats.cached')}</span>
                <span className='font-semibold text-center'>{formatNumber(stats?.totalCachedTokensToday || 0)}</span>
              </div>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

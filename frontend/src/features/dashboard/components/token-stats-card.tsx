import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { BarChart4, TrendingUp } from 'lucide-react'

import { formatNumber } from '@/utils/format-number'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Button } from '@/components/ui/button'
import { Alert, AlertTitle } from '@/components/ui/alert'
import { useTokenStats, useModelTokenStats, useRequestsByModel } from '../data/dashboard'
import { ModelTokenChart } from './model-token-chart'
import { ModelTokenTable } from './model-token-table'
import { ModelTokenStatsCard } from './model-token-stats-card'
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
          <div className='space-y-2'>
            <Skeleton className='h-8 w-[80px]' />
            <Skeleton className='h-4 w-[140px] mt-1' />
          </div>
        </CardContent>
      </Card>
    )
  }

  if (error) {
    return (
      <Card>
        <CardHeader className='flex flex-row items-center justify-between space-y-0 pb-2'>
          <div className='flex items-center gap-2'>
            <div className='p-1.5 bg-primary/10 text-primary rounded-lg dark:bg-primary/20'>
              <BarChart4 className='h-4 w-4' />
            </div>
            <CardTitle className='text-sm font-medium'>{t('dashboard.cards.tokenStats')}</CardTitle>
          </div>
          <div className='flex items-center gap-1'>
            {/* <span className='text-xs text-muted-foreground'>{t('dashboard.stats.this')}</span> */}
            <span className='text-xs bg-primary/10 text-primary px-2 py-1 rounded-md dark:bg-primary/20'>{t('dashboard.stats.month')}</span>
          </div>
        </CardHeader>
        <CardContent>
          <div className='text-sm text-red-500'>{t('common.loadError')}</div>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card className='hover-card'>
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
                <ModelTokenStatsCard defaultModels={availableModels} />
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
        <div className='flex justify-between items-end'>
          <div className='text-center'>
            <div className='text-xs text-muted-foreground mb-1'>{t('dashboard.stats.input')}</div>
            <div className='text-lg font-bold font-mono'>{formatNumber(stats?.totalInputTokensThisMonth || 0)}</div>
          </div>
          <div className='w-px h-8 bg-border'></div>
          <div className='text-center'>
            <div className='text-xs text-muted-foreground mb-1'>{t('dashboard.stats.output')}</div>
            <div className='text-lg font-bold font-mono'>{formatNumber(stats?.totalOutputTokensThisMonth || 0)}</div>
          </div>
          <div className='w-px h-8 bg-border'></div>
          <div className='text-center'>
            <div className='text-xs text-muted-foreground mb-1'>{t('dashboard.stats.cached')}</div>
            <div className='text-lg font-bold font-mono text-muted-foreground'>{formatNumber(stats?.totalCachedTokensThisMonth || 0)}</div>
          </div>
        </div>
      </CardContent>
    </Card>
  )
}

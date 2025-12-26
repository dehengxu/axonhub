import { useState, useCallback, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import { DateRange } from 'react-day-picker'
import { usePaginationSearch } from '@/hooks/use-pagination-search'
import { Header } from '@/components/layout/header'
import { Main } from '@/components/layout/main'
import { RequestsTable } from './components'
import { RequestsProvider } from './context'
import { useRequests } from './data'
import { Badge } from '@/components/ui/badge'
import { buildDateRangeWhereClause } from '@/utils/date-range'

function RequestsContent() {
  const { t } = useTranslation()
  const { pageSize, setCursors, setPageSize, resetCursor, paginationArgs, cursorHistory } = usePaginationSearch({
    defaultPageSize: 20,
  })
  const [statusFilter, setStatusFilter] = useState<string[]>([])
  const [sourceFilter, setSourceFilter] = useState<string[]>([])
  const [channelFilter, setChannelFilter] = useState<string[]>([])
  const [dateRange, setDateRange] = useState<DateRange | undefined>()

  // Build where clause with filters
  const whereClause = (() => {
    const where: { [key: string]: any } = {
      ...buildDateRangeWhereClause(dateRange),
    }
    if (statusFilter.length > 0) {
      where.statusIn = statusFilter
    }
    if (sourceFilter.length > 0) {
      where.sourceIn = sourceFilter
    }
    if (channelFilter.length > 0) {
      where.channelIDIn = channelFilter
    }
    return Object.keys(where).length > 0 ? where : undefined
  })()

  const { data, isLoading, refetch } = useRequests({
    ...paginationArgs,
    where: whereClause,
    orderBy: {
      field: 'CREATED_AT',
      direction: 'DESC',
    },
  })

  const requests = data?.edges?.map((edge) => edge.node) || []
  const pageInfo = data?.pageInfo

  // Calculate token totals for all displayed requests
  const pageTokenTotals = useMemo(() => {
    let totalTokens = 0
    let promptTokens = 0
    let completionTokens = 0

    requests.forEach((request) => {
      const usageLogs = (request as any).usageLogs
      if (usageLogs && usageLogs.edges && usageLogs.edges.length > 0) {
        const usage = usageLogs.edges[0].node
        totalTokens += usage.totalTokens || 0
        promptTokens += usage.promptTokens || 0
        completionTokens += usage.completionTokens || 0
      }
    })

    return { totalTokens, promptTokens, completionTokens }
  }, [requests])

  const isFirstPage = !paginationArgs.after && cursorHistory.length === 0

  const handleNextPage = () => {
    if (data?.pageInfo?.hasNextPage && data?.pageInfo?.endCursor) {
      setCursors(data.pageInfo.startCursor ?? undefined, data.pageInfo.endCursor ?? undefined, 'after')
    }
  }

  const handlePreviousPage = () => {
    if (data?.pageInfo?.hasPreviousPage) {
      setCursors(data.pageInfo.startCursor ?? undefined, data.pageInfo.endCursor ?? undefined, 'before')
    }
  }

  const handleStatusFilterChange = useCallback(
    (filters: string[]) => {
      setStatusFilter(filters)
      resetCursor()
    },
    [resetCursor]
  )

  const handleSourceFilterChange = useCallback(
    (filters: string[]) => {
      setSourceFilter(filters)
      resetCursor()
    },
    [resetCursor]
  )

  const handleChannelFilterChange = useCallback(
    (filters: string[]) => {
      setChannelFilter(filters)
      resetCursor()
    },
    [resetCursor]
  )

  const handleDateRangeChange = useCallback(
    (range: DateRange | undefined) => {
      setDateRange(range)
      resetCursor()
    },
    [resetCursor]
  )

  return (
    <div className='flex flex-1 flex-col overflow-hidden'>
      <div className='mb-4 flex flex-wrap items-center justify-between'>
        <div>
          <h2 className='text-2xl font-bold tracking-tight'>{t('requests.title')}</h2>
          <p className='text-muted-foreground'>{t('requests.description')}</p>
        </div>
        <div className='flex items-center gap-4'>
          {requests.length > 0 && (
            <>
              <div className='flex items-center gap-2'>
                <span className='text-sm text-muted-foreground'>{t('common.total')}:</span>
                <Badge variant='secondary' className='font-mono text-xs'>
                  {pageTokenTotals.totalTokens.toLocaleString()}
                </Badge>
              </div>
              <div className='flex items-center gap-2'>
                <span className='text-sm text-muted-foreground'>{t('requests.columns.promptTokens')}:</span>
                <Badge variant='secondary' className='font-mono text-xs'>
                  {pageTokenTotals.promptTokens.toLocaleString()}
                </Badge>
              </div>
              <div className='flex items-center gap-2'>
                <span className='text-sm text-muted-foreground'>{t('requests.columns.completionTokens')}:</span>
                <Badge variant='secondary' className='font-mono text-xs'>
                  {pageTokenTotals.completionTokens.toLocaleString()}
                </Badge>
              </div>
            </>
          )}
        </div>
      </div>
      <RequestsTable
        data={requests}
        loading={isLoading}
        pageInfo={pageInfo}
        pageSize={pageSize}
        totalCount={data?.totalCount}
        statusFilter={statusFilter}
        sourceFilter={sourceFilter}
        channelFilter={channelFilter}
        dateRange={dateRange}
        onNextPage={handleNextPage}
        onPreviousPage={handlePreviousPage}
        onPageSizeChange={setPageSize}
        onStatusFilterChange={handleStatusFilterChange}
        onSourceFilterChange={handleSourceFilterChange}
        onChannelFilterChange={handleChannelFilterChange}
        onDateRangeChange={handleDateRangeChange}
        onRefresh={refetch}
        showRefresh={isFirstPage}
      />
    </div>
  )
}

export default function RequestsManagement() {
  const { t } = useTranslation()

  return (
    <RequestsProvider>
      {/* <Header fixed></Header> */}

      <Main fixed>
        <RequestsContent />
      </Main>
    </RequestsProvider>
  )
}

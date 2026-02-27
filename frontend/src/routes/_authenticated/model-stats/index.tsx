import { createFileRoute } from '@tanstack/react-router'
import ModelStatsPage from '@/features/model-stats'

export const Route = createFileRoute('/_authenticated/model-stats/')({
  component: ModelStatsPage,
})

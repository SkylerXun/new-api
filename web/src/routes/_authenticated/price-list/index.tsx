import { createFileRoute } from '@tanstack/react-router'

import { PriceList } from '@/features/price-list'

export const Route = createFileRoute('/_authenticated/price-list/')({
  component: PriceList,
})

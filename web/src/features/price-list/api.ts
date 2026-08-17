import { api } from '@/lib/api'

import type { PriceListData } from './types'

export async function getPriceList(): Promise<PriceListData> {
  const res = await api.get('/api/price-list')
  return res.data
}

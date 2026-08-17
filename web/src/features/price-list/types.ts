import type { PricingModel, PricingVendor } from '@/features/pricing/types'

export type PriceListGroupInfo = {
  desc: string
  ratio: number
  ratio_label: string
}

export type PriceListData = {
  success: boolean
  message?: string
  data: PricingModel[]
  vendors: PricingVendor[]
  group_ratio: Record<string, number>
  group_info: Record<string, PriceListGroupInfo>
}

export type PriceField =
  | 'input'
  | 'output'
  | 'cache_read'
  | 'cache_write'

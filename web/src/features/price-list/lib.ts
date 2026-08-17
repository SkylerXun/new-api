import type { PricingModel } from '@/features/pricing/types'

import type { PriceField } from './types'

function isFiniteNumber(value: number | null | undefined): value is number {
  return typeof value === 'number' && Number.isFinite(value)
}

export function getActualPrice(
  model: PricingModel,
  field: PriceField,
  groupRatio: number
): number | null {
  const base = model.model_ratio * 2 * groupRatio
  if (!Number.isFinite(base)) return null

  switch (field) {
    case 'input':
      return base
    case 'output':
      return base * model.completion_ratio
    case 'cache_read':
      return isFiniteNumber(model.cache_ratio) ? base * model.cache_ratio : null
    case 'cache_write':
      return isFiniteNumber(model.create_cache_ratio)
        ? base * model.create_cache_ratio
        : null
  }
}

export function getOfficialPrice(
  model: PricingModel,
  field: PriceField
): number | null {
  switch (field) {
    case 'input':
      return model.official_input_price ?? null
    case 'output':
      return model.official_output_price ?? null
    case 'cache_read':
      return model.official_cache_read_price ?? null
    case 'cache_write':
      return model.official_cache_write_price ?? null
  }
}

export function formatUsdPrice(value: number | null): string {
  if (value === null || !Number.isFinite(value)) return '—'
  return new Intl.NumberFormat(undefined, {
    style: 'currency',
    currency: 'USD',
    currencyDisplay: 'narrowSymbol',
    minimumFractionDigits: 0,
    maximumFractionDigits: Math.abs(value) < 1 ? 6 : 4,
  }).format(value)
}

export function formatDiscountRatio(
  actualInput: number | null,
  officialInput: number | null
): string {
  if (
    actualInput === null ||
    officialInput === null ||
    officialInput <= 0
  ) {
    return '—'
  }

  const ratio = actualInput / officialInput
  return `${Number(ratio.toFixed(4))}x`
}

export function filterPriceListModels(
  models: PricingModel[],
  group: string,
  vendorID: string,
  query: string
): PricingModel[] {
  const normalizedQuery = query.trim().toLowerCase()
  return models.filter((model) => {
    const modelGroups = model.enable_groups || []
    const matchesGroup =
      group === 'all' ||
      modelGroups.includes('all') ||
      modelGroups.includes(group)
    const matchesVendor =
      vendorID === 'all' || String(model.vendor_id || '') === vendorID
    const matchesQuery =
      normalizedQuery === '' ||
      model.model_name.toLowerCase().includes(normalizedQuery)
    return matchesGroup && matchesVendor && matchesQuery
  })
}

import { useTranslation } from 'react-i18next'

import type { PricingModel } from '@/features/pricing/types'

import {
  formatDiscountRatio,
  formatUsdPrice,
  getActualPrice,
  getOfficialPrice,
} from '../lib'
import type { PriceField } from '../types'

const PRICE_FIELDS: Array<{ key: PriceField; label: string }> = [
  { key: 'input', label: 'Input' },
  { key: 'output', label: 'Output' },
  { key: 'cache_read', label: 'Cache read' },
  { key: 'cache_write', label: 'Cache write' },
]

type PriceListTableProps = {
  models: PricingModel[]
  groupRatio: number
}

export function PriceListTable(props: PriceListTableProps) {
  const { t } = useTranslation()

  return (
    <div className='overflow-x-auto border'>
      <table className='w-full min-w-[1050px] border-collapse text-left text-sm'>
        <thead>
          <tr className='border-b bg-muted/35'>
            <th className='w-[240px] px-4 py-3 font-medium' rowSpan={2}>
              {t('Model')}
            </th>
            <th
              className='border-l border-emerald-200 bg-emerald-50/70 px-3 py-3 text-center font-medium text-emerald-700 dark:border-emerald-900 dark:bg-emerald-950/30 dark:text-emerald-300'
              colSpan={4}
            >
              {t('Actual price')} <span className='font-normal'>USD / 1M tokens</span>
            </th>
            <th className='border-l px-3 py-3 text-center font-medium' colSpan={4}>
              {t('Official price')} <span className='font-normal'>USD / 1M tokens</span>
            </th>
            <th className='border-l px-4 py-3 text-right font-medium' rowSpan={2}>
              {t('Discount')}
            </th>
          </tr>
          <tr className='border-b bg-muted/20 text-xs text-muted-foreground'>
            {PRICE_FIELDS.map((field) => (
              <th
                className='border-l border-emerald-200 bg-emerald-50/50 px-3 py-2 font-medium dark:border-emerald-900 dark:bg-emerald-950/20'
                key={`actual-${field.key}`}
              >
                {t(field.label)}
              </th>
            ))}
            {PRICE_FIELDS.map((field) => (
              <th className='border-l px-3 py-2 font-medium' key={`official-${field.key}`}>
                {t(field.label)}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {props.models.map((model) => {
            const actualInput = getActualPrice(model, 'input', props.groupRatio)
            const officialInput = getOfficialPrice(model, 'input')
            return (
              <tr className='border-b last:border-0 hover:bg-muted/25' key={model.model_name}>
                <td className='px-4 py-3 font-mono text-xs font-medium'>
                  {model.model_name}
                </td>
                {PRICE_FIELDS.map((field) => (
                  <td
                    className='border-l border-emerald-100 bg-emerald-50/35 px-3 py-3 font-mono text-xs tabular-nums dark:border-emerald-950 dark:bg-emerald-950/15'
                    key={`actual-${field.key}`}
                  >
                    {formatUsdPrice(getActualPrice(model, field.key, props.groupRatio))}
                  </td>
                ))}
                {PRICE_FIELDS.map((field) => (
                  <td className='border-l px-3 py-3 font-mono text-xs tabular-nums' key={`official-${field.key}`}>
                    {formatUsdPrice(getOfficialPrice(model, field.key))}
                  </td>
                ))}
                <td className='border-l px-4 py-3 text-right font-mono text-xs tabular-nums'>
                  {formatDiscountRatio(actualInput, officialInput)}
                </td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

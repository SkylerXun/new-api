/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useTranslation } from 'react-i18next'

import {
  getMonthlyDiscountPosition,
  getMonthlyDiscountScale,
} from '../lib/monthly-discount-progress'
import type { MonthlyBillingProgress } from '../types'

export function MonthlyDiscountProgress(props: {
  progress: MonthlyBillingProgress | null
}) {
  const { t } = useTranslation()
  if (!props.progress?.enabled || props.progress.tiers.length === 0) return null

  const maximumTier = props.progress.tiers.at(-1)?.threshold_usd ?? 1
  const scaleMaximum = getMonthlyDiscountScale(maximumTier)
  const progressPercent = getMonthlyDiscountPosition(
    props.progress.spent_usd,
    scaleMaximum
  )

  return (
    <section
      className='w-full overflow-x-auto py-2'
      aria-label={t('Monthly discount progress')}
    >
      <div className='min-w-[640px] px-1 pb-16'>
        <div className='mb-3 flex items-center justify-between text-sm'>
          <span className='font-medium'>{t('Monthly usage')}</span>
          <span className='font-mono tabular-nums'>
            ${props.progress.spent_usd.toFixed(2)}
          </span>
        </div>
        <div className='relative h-3 rounded-sm bg-gradient-to-r from-emerald-500 via-amber-400 to-red-500'>
          <div
            className='border-background bg-foreground absolute top-1/2 size-4 -translate-x-1/2 -translate-y-1/2 rounded-full border-2 shadow'
            style={{ left: `${progressPercent}%` }}
          />
          {props.progress.tiers.map((tier, index) => {
            const left = getMonthlyDiscountPosition(
              tier.threshold_usd,
              scaleMaximum
            )
            return (
              <div
                className='bg-background/90 absolute top-0 h-3 w-px'
                key={tier.threshold_usd}
                style={{ left: `${left}%` }}
              >
                <div
                  className={`absolute top-5 -translate-x-1/2 text-center text-xs whitespace-nowrap ${index % 2 ? 'translate-y-4' : ''}`}
                >
                  <div>${tier.threshold_usd.toLocaleString()}</div>
                  <div className='font-medium'>{tier.discount_percent}%</div>
                </div>
              </div>
            )
          })}
        </div>
      </div>
    </section>
  )
}

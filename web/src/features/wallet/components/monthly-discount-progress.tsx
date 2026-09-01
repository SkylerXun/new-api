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
import { CircleHelp } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import {
  getMonthlyDiscountPosition,
  getMonthlyDiscountScale,
} from '../lib/monthly-discount-progress'
import type { MonthlyBillingProgress } from '../types'

const TIER_COLORS = ['#10b981', '#f59e0b', '#f97316', '#b91c1c', '#7f1d1d']

function formatUSD(value: number, maximumFractionDigits = 2) {
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
    minimumFractionDigits: maximumFractionDigits,
    maximumFractionDigits,
  }).format(value)
}

function formatTail(value: number) {
  return `$${new Intl.NumberFormat('en-US', {
    notation: 'compact',
    maximumFractionDigits: 0,
  }).format(value)}+`
}

function getTooltipTransform(progressPercent: number) {
  if (progressPercent < 9) return 'translateX(0)'
  if (progressPercent > 91) return 'translateX(-100%)'
  return 'translateX(-50%)'
}

export function MonthlyDiscountProgress(props: {
  progress: MonthlyBillingProgress | null
}) {
  const { t } = useTranslation()
  if (!props.progress?.enabled || props.progress.tiers.length === 0) return null

  const { spent_usd: spentUSD, tiers } = props.progress
  const maximumTier = tiers.at(-1)?.threshold_usd ?? 1
  const scaleMaximum = getMonthlyDiscountScale(maximumTier)
  const progressPercent = getMonthlyDiscountPosition(spentUSD, scaleMaximum)
  const nextTier = tiers.find((tier) => tier.threshold_usd > spentUSD)
  const tooltipTransform = getTooltipTransform(progressPercent)

  return (
    <section
      className='w-full overflow-hidden rounded-[20px] border border-slate-200 bg-white shadow-[0_8px_24px_rgba(15,23,42,0.06)] dark:border-slate-800 dark:bg-slate-950'
      aria-label={t('Monthly discount progress')}
    >
      <div className='overflow-x-auto'>
        <div className='min-w-[680px] px-8 pt-7 pb-6'>
          <header className='flex items-start justify-between gap-8'>
            <div>
              <div className='text-base font-semibold text-slate-600 dark:text-slate-300'>
                {t('Monthly usage')}
              </div>
              <div className='mt-1 text-4xl leading-none font-bold tracking-tight text-slate-900 tabular-nums dark:text-white'>
                {formatUSD(spentUSD)}
              </div>
            </div>

            <div className='min-w-44 rounded-lg bg-emerald-50 px-4 py-3 text-center text-sm font-semibold text-emerald-700 dark:bg-emerald-950/60 dark:text-emerald-300'>
              {nextTier ? (
                <>
                  <div>{t('Until next discount')}</div>
                  <div className='mt-0.5 text-lg leading-none tabular-nums'>
                    {formatUSD(nextTier.threshold_usd - spentUSD)}
                  </div>
                </>
              ) : (
                <>
                  <div>{t('Highest discount reached')}</div>
                  <div className='mt-0.5 text-lg leading-none tabular-nums'>
                    {props.progress.current_discount_percent}%
                  </div>
                </>
              )}
            </div>
          </header>

          <div className='mt-16 pb-24'>
            <div
              className='relative h-4 rounded-full'
              style={{
                background:
                  'linear-gradient(90deg, #fde68a 0%, #f59e0b 38%, #f97316 62%, #b91c1c 82%, #7f1d1d 100%)',
              }}
            >
              <div
                className='absolute inset-y-0 right-0 rounded-r-full bg-slate-900 transition-[left] duration-700 ease-out dark:bg-slate-800'
                style={{ left: `${progressPercent}%` }}
              />

              <div
                className='absolute top-1/2 z-20 size-5 rounded-full bg-amber-500/20 shadow-[0_0_18px_rgba(249,115,22,0.45)] transition-[left] duration-700 ease-out'
                style={{
                  left: `${progressPercent}%`,
                  transform: 'translate(-50%, -50%)',
                }}
              >
                <span className='absolute inset-[5px] rounded-full border-2 border-amber-200 bg-white shadow-sm' />
              </div>

              <div
                className='absolute -top-[56px] z-30 rounded-md bg-[#2b1f1f] px-3 py-2 text-center text-white shadow-lg transition-[left] duration-700 ease-out'
                style={{
                  left: `${progressPercent}%`,
                  transform: tooltipTransform,
                }}
              >
                <div className='text-sm leading-5 font-bold tabular-nums'>
                  {formatUSD(spentUSD)}
                </div>
                <div className='text-[11px] font-semibold text-amber-200'>
                  {t('Current usage')}
                </div>
                <span className='absolute bottom-[-6px] left-1/2 size-3 -translate-x-1/2 rotate-45 bg-[#2b1f1f]' />
              </div>

              {tiers.map((tier, index) => {
                const left = getMonthlyDiscountPosition(
                  tier.threshold_usd,
                  scaleMaximum
                )
                const color =
                  TIER_COLORS[Math.min(index, TIER_COLORS.length - 1)]

                return (
                  <div
                    className='absolute top-1/2 z-10 h-[90px] w-px -translate-y-1/2 border-l border-dashed border-slate-400/80'
                    key={`${tier.threshold_usd}-${tier.discount_percent}`}
                    style={{ left: `${left}%` }}
                  >
                    <span
                      className='absolute top-1/2 left-1/2 size-3 -translate-x-1/2 -translate-y-1/2 rounded-full border-[3px] bg-slate-900 dark:bg-slate-800'
                      style={{ borderColor: color }}
                    />
                    <div className='absolute top-[58px] left-1/2 -translate-x-1/2 text-center whitespace-nowrap'>
                      <div className='text-sm font-semibold text-slate-700 tabular-nums dark:text-slate-200'>
                        {formatUSD(tier.threshold_usd, 0)}
                      </div>
                      <div className='mt-0.5 rounded bg-slate-800 px-2 py-0.5 text-xs font-semibold text-amber-200 dark:bg-slate-700'>
                        {t('Discount rate')}: {tier.discount_percent}%
                      </div>
                    </div>
                  </div>
                )
              })}

              <div className='absolute top-8 right-0 text-xs font-medium text-slate-500 tabular-nums dark:text-slate-400'>
                {formatTail(scaleMaximum)}
              </div>
            </div>
          </div>

          <footer className='flex items-center gap-8 border-t border-slate-200 pt-5 text-sm text-slate-600 dark:border-slate-800 dark:text-slate-300'>
            <div className='flex items-center gap-2 whitespace-nowrap'>
              <span className='size-2.5 rounded-full bg-teal-600' />
              {t('Base rate')}
            </div>
            {tiers.map((tier, index) => (
              <div
                className='flex items-center gap-2 whitespace-nowrap'
                key={`legend-${tier.threshold_usd}-${tier.discount_percent}`}
              >
                <span
                  className='size-2.5 rounded-full'
                  style={{
                    backgroundColor:
                      TIER_COLORS[Math.min(index, TIER_COLORS.length - 1)],
                  }}
                />
                {t('Discount tier {{tier}} ({{discount}}% off)', {
                  tier: index + 1,
                  discount: tier.discount_percent,
                  rate: (10 - tier.discount_percent / 10)
                    .toFixed(1)
                    .replace('.0', ''),
                })}
              </div>
            ))}
            <CircleHelp
              className='ml-auto size-4 shrink-0 text-slate-400'
              aria-label={t('Monthly discount progress')}
            />
          </footer>
        </div>
      </div>
    </section>
  )
}

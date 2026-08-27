/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { z } from 'zod'

export const BILLING_CURVE_OPTION_KEY = 'billing_curve_setting.config'

const MAX_MULTIPLIER = 1_000_000
const MAX_USAGE_USD = 1_000_000_000_000

export type BillingCurveConfig = {
  enabled: boolean
  k1: number
  k2: number
  threshold_usd: number
  window_usd: number
  target_average_k: number
  monthly_enabled: boolean
  monthly_tiers: Array<{ threshold_usd: number; discount_percent: number }>
  monthly_backfill_cutoff: number
}

export const DEFAULT_BILLING_CURVE_CONFIG: BillingCurveConfig = {
  enabled: false,
  k1: 5,
  k2: 15,
  threshold_usd: 75,
  window_usd: 150,
  target_average_k: 10,
  monthly_enabled: false,
  monthly_tiers: [],
  monthly_backfill_cutoff: 0,
}

type Translate = (key: string) => string

function finiteNumberSchema(t: Translate) {
  return z.number().finite(t('Enter a finite number'))
}

export function createBillingCurveConfigSchema(t: Translate) {
  return z
    .object({
      enabled: z.boolean(),
      k1: finiteNumberSchema(t)
        .gt(0, t('Enter a value greater than zero'))
        .max(
          MAX_MULTIPLIER,
          t('Enter a value no greater than the allowed maximum')
        ),
      k2: finiteNumberSchema(t)
        .gt(0, t('Enter a value greater than zero'))
        .max(
          MAX_MULTIPLIER,
          t('Enter a value no greater than the allowed maximum')
        ),
      threshold_usd: finiteNumberSchema(t)
        .min(0, t('Enter a non-negative value'))
        .max(
          MAX_USAGE_USD,
          t('Enter a value no greater than the allowed maximum')
        ),
      window_usd: finiteNumberSchema(t)
        .gt(0, t('Enter a value greater than zero'))
        .max(
          MAX_USAGE_USD,
          t('Enter a value no greater than the allowed maximum')
        ),
      target_average_k: z.number().optional().default(10),
      monthly_enabled: z.boolean().default(false),
      monthly_tiers: z
        .array(
          z.object({
            threshold_usd: finiteNumberSchema(t).gt(0).max(MAX_USAGE_USD),
            discount_percent: finiteNumberSchema(t).min(0).lt(100),
          })
        )
        .max(100)
        .default([]),
      monthly_backfill_cutoff: z.number().int().nonnegative().default(0),
    })
    .superRefine((config, context) => {
      if (config.monthly_enabled && config.monthly_tiers.length === 0) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['monthly_tiers'],
          message: t('Add at least one monthly discount tier'),
        })
      }
      if (config.k2 < config.k1) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['k2'],
          message: t(
            'The final multiplier must be at least the initial multiplier.'
          ),
        })
      }
      for (let i = 1; i < config.monthly_tiers.length; i += 1) {
        const previous = config.monthly_tiers.at(i - 1)
        const current = config.monthly_tiers.at(i)
        if (!previous || !current) {
          continue
        }
        if (current.threshold_usd <= previous.threshold_usd) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['monthly_tiers', i, 'threshold_usd'],
            message: t('Thresholds must be strictly increasing'),
          })
        }
        if (current.discount_percent < previous.discount_percent) {
          context.addIssue({
            code: z.ZodIssueCode.custom,
            path: ['monthly_tiers', i, 'discount_percent'],
            message: t('Discounts must be non-decreasing'),
          })
        }
      }
    })
}

const billingCurveConfigSchema = createBillingCurveConfigSchema((key) => key)

export function parseBillingCurveConfig(
  rawConfig: string | undefined
): BillingCurveConfig {
  if (!rawConfig) return { ...DEFAULT_BILLING_CURVE_CONFIG }

  try {
    const result = billingCurveConfigSchema.safeParse(JSON.parse(rawConfig))
    if (result.success) return result.data
  } catch {
    // A malformed persisted value must not block access to the settings page.
  }

  return { ...DEFAULT_BILLING_CURVE_CONFIG }
}

export function serializeBillingCurveConfig(
  config: BillingCurveConfig
): string {
  return JSON.stringify({
    enabled: config.enabled,
    k1: config.k1,
    k2: config.k2,
    threshold_usd: config.threshold_usd,
    window_usd: config.window_usd,
    target_average_k: config.target_average_k,
    monthly_enabled: config.monthly_enabled,
    monthly_tiers: config.monthly_tiers,
    monthly_backfill_cutoff: config.monthly_backfill_cutoff,
  })
}

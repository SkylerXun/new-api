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
}

export const DEFAULT_BILLING_CURVE_CONFIG: BillingCurveConfig = {
  enabled: false,
  k1: 5,
  k2: 15,
  threshold_usd: 75,
  window_usd: 150,
  target_average_k: 10,
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
      target_average_k: finiteNumberSchema(t),
    })
    .superRefine((config, context) => {
      if (config.k2 < config.k1) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['k2'],
          message: t(
            'The final multiplier must be at least the initial multiplier.'
          ),
        })
      }

      if (
        config.target_average_k < config.k1 ||
        config.target_average_k > config.k2
      ) {
        context.addIssue({
          code: z.ZodIssueCode.custom,
          path: ['target_average_k'],
          message: t(
            'The target average multiplier must be between the initial and final multipliers.'
          ),
        })
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
  })
}

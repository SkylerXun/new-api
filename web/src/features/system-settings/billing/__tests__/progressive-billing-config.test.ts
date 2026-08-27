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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  DEFAULT_BILLING_CURVE_CONFIG,
  createBillingCurveConfigSchema,
  parseBillingCurveConfig,
  serializeBillingCurveConfig,
} from '../progressive-billing-config.ts'

const schema = createBillingCurveConfigSchema((key) => key)

describe('progressive billing configuration', () => {
  test('uses the documented defaults when no valid configuration is stored', () => {
    assert.deepEqual(parseBillingCurveConfig(undefined), {
      enabled: false,
      k1: 5,
      k2: 15,
      threshold_usd: 75,
      window_usd: 150,
      target_average_k: 10,
      monthly_enabled: false,
      monthly_tiers: [],
      monthly_backfill_cutoff: 0,
    })
    assert.deepEqual(parseBillingCurveConfig('{'), DEFAULT_BILLING_CURVE_CONFIG)
  })

  test('parses a stored configuration for form defaults', () => {
    const config = {
      enabled: true,
      k1: 6,
      k2: 12,
      threshold_usd: 20,
      window_usd: 80,
      target_average_k: 9,
      monthly_enabled: true,
      monthly_tiers: [{ threshold_usd: 1000, discount_percent: 10 }],
      monthly_backfill_cutoff: 1_787_673_600,
    }

    assert.deepEqual(parseBillingCurveConfig(JSON.stringify(config)), config)
  })

  test('preserves legacy curve values while defaulting new monthly fields', () => {
    const legacy = {
      enabled: true,
      k1: 2,
      k2: 4,
      threshold_usd: 30,
      window_usd: 60,
      target_average_k: 3,
    }
    assert.deepEqual(parseBillingCurveConfig(JSON.stringify(legacy)), {
      ...legacy,
      monthly_enabled: false,
      monthly_tiers: [],
      monthly_backfill_cutoff: 0,
    })
  })

  test('accepts a complete configuration within the backend bounds', () => {
    const result = schema.safeParse({
      enabled: true,
      k1: 5,
      k2: 15,
      threshold_usd: 75,
      window_usd: 150,
      target_average_k: 10,
      monthly_enabled: true,
      monthly_tiers: [{ threshold_usd: 1000, discount_percent: 10 }],
      monthly_backfill_cutoff: 1_787_673_600,
    })

    assert.equal(result.success, true)
  })

  test('rejects invalid multiplier ordering and usage windows', () => {
    for (const config of [
      { ...DEFAULT_BILLING_CURVE_CONFIG, k2: 4 },
      { ...DEFAULT_BILLING_CURVE_CONFIG, window_usd: 0 },
    ]) {
      assert.equal(schema.safeParse(config).success, false)
    }
  })

  test('rejects unordered thresholds and decreasing discounts', () => {
    assert.equal(
      schema.safeParse({
        ...DEFAULT_BILLING_CURVE_CONFIG,
        monthly_tiers: [
          { threshold_usd: 1000, discount_percent: 20 },
          { threshold_usd: 900, discount_percent: 30 },
        ],
      }).success,
      false
    )
    assert.equal(
      schema.safeParse({
        ...DEFAULT_BILLING_CURVE_CONFIG,
        monthly_tiers: [
          { threshold_usd: 1000, discount_percent: 20 },
          { threshold_usd: 2000, discount_percent: 10 },
        ],
      }).success,
      false
    )
  })

  test('serializes all fields as one atomic option value', () => {
    assert.equal(
      serializeBillingCurveConfig({
        enabled: true,
        k1: 6,
        k2: 12,
        threshold_usd: 20,
        window_usd: 80,
        target_average_k: 9,
        monthly_enabled: true,
        monthly_tiers: [{ threshold_usd: 1000, discount_percent: 10 }],
        monthly_backfill_cutoff: 1_787_673_600,
      }),
      '{"enabled":true,"k1":6,"k2":12,"threshold_usd":20,"window_usd":80,"target_average_k":9,"monthly_enabled":true,"monthly_tiers":[{"threshold_usd":1000,"discount_percent":10}],"monthly_backfill_cutoff":1787673600}'
    )
  })
})

import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  getMonthlyDiscountPosition,
  getMonthlyDiscountScale,
} from '../monthly-discount-progress.ts'

describe('monthly discount progress layout', () => {
  test('keeps the maximum tier at five sixths of the track', () => {
    const scale = getMonthlyDiscountScale(1000)
    assert.equal(scale, 1200)
    assert.ok(
      Math.abs(getMonthlyDiscountPosition(1000, scale) - 1000 / 12) < 1e-10
    )
  })

  test('caps usage beyond the visible tail without changing its source value', () => {
    assert.equal(getMonthlyDiscountPosition(1500, 1200), 100)
  })
})

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

import { getRemainingSeconds, splitCountdown } from '../countdown'

describe('activity countdown', () => {
  test('subtracts whole elapsed seconds from the server-synchronized deadline', () => {
    const remaining = getRemainingSeconds(1_100, 1_000, 9_999)

    assert.equal(remaining, 91)
  })

  test('does not add time when the client clock moves backwards', () => {
    const remaining = getRemainingSeconds(1_100, 1_000, -5_000)

    assert.equal(remaining, 100)
  })

  test('clamps expired and non-finite countdown inputs to zero', () => {
    assert.equal(getRemainingSeconds(1_000, 1_001, 0), 0)
    assert.equal(getRemainingSeconds(Number.NaN, 1_000, 0), 0)
    assert.equal(getRemainingSeconds(1_100, 1_000, Number.POSITIVE_INFINITY), 0)
    assert.deepEqual(splitCountdown(Number.NaN), {
      days: 0,
      hours: 0,
      minutes: 0,
      seconds: 0,
    })
  })

  test('splits a multi-day countdown into bounded display units', () => {
    const parts = splitCountdown(2 * 86_400 + 3 * 3_600 + 4 * 60 + 5.9)

    assert.deepEqual(parts, {
      days: 2,
      hours: 3,
      minutes: 4,
      seconds: 5,
    })
  })
})

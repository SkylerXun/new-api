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

import type { AllUsersActivityGrantTask } from '../../../types'
import {
  activityCampaignSchema,
  activitySettingsSchema,
  getAllUsersGrantProgress,
  grantAllUsersSchema,
  isActiveAllUsersGrantTask,
  parseActivityCampaignEndAt,
} from '../lib'

describe('activity settings validation', () => {
  test('accepts the configured new-user bonus boundaries', () => {
    const result = activitySettingsSchema.safeParse({
      activity_setting: {
        new_user_redeem_bonus_enabled: true,
        new_user_redeem_bonus_percent: 30,
        new_user_redeem_bonus_window_days: 1,
      },
    })

    assert.equal(result.success, true)
  })

  test('rejects bonus percentages and windows outside backend limits', () => {
    for (const values of [
      {
        new_user_redeem_bonus_enabled: true,
        new_user_redeem_bonus_percent: 1001,
        new_user_redeem_bonus_window_days: 1,
      },
      {
        new_user_redeem_bonus_enabled: true,
        new_user_redeem_bonus_percent: 30,
        new_user_redeem_bonus_window_days: 0,
      },
    ]) {
      const result = activitySettingsSchema.safeParse({
        activity_setting: values,
      })
      assert.equal(result.success, false)
    }
  })
})

describe('all-user grant validation and progress', () => {
  test('accepts a positive decimal USD amount and trims its reason', () => {
    const result = grantAllUsersSchema.safeParse({
      amountUSD: ' 1.25 ',
      reason: ' launch credit ',
    })

    assert.equal(result.success, true)
    if (!result.success) return
    assert.deepEqual(result.data, {
      amountUSD: '1.25',
      reason: 'launch credit',
    })
  })

  test('rejects empty, non-finite, and non-positive USD amounts', () => {
    for (const amountUSD of ['', '0', '-1', 'NaN', 'Infinity']) {
      assert.equal(
        grantAllUsersSchema.safeParse({ amountUSD, reason: '' }).success,
        false
      )
    }
  })

  test('clamps reported task progress and identifies active states', () => {
    const task = {
      status: 'running',
      state: { progress: 140 },
    } as AllUsersActivityGrantTask

    assert.equal(isActiveAllUsersGrantTask(task), true)
    assert.equal(getAllUsersGrantProgress(task), 100)
    assert.equal(
      isActiveAllUsersGrantTask({ ...task, status: 'succeeded' }),
      false
    )
  })
})

describe('activity campaign validation', () => {
  test('requires a valid deadline for claimable campaigns', () => {
    const result = activityCampaignSchema.safeParse({
      type: 'claimable',
      title: 'Launch credit',
      description: '',
      amountUSD: '5',
      endsAt: '',
      audienceType: 'all',
    })

    assert.equal(result.success, false)
  })

  test('allows immediate campaigns without a deadline', () => {
    const result = activityCampaignSchema.safeParse({
      type: 'immediate',
      title: 'Launch credit',
      description: 'Thanks for joining.',
      amountUSD: '5',
      endsAt: '',
      audienceType: 'all',
    })

    assert.equal(result.success, true)
  })

  test('requires selected-user campaigns to be claimable', () => {
    const result = activityCampaignSchema.safeParse({
      type: 'immediate',
      title: 'Targeted credit',
      description: '',
      amountUSD: '5',
      endsAt: '',
      audienceType: 'selected',
    })

    assert.equal(result.success, false)
  })

  test('converts a local datetime input to a Unix timestamp', () => {
    const timestamp = parseActivityCampaignEndAt('2030-01-02T03:04')

    assert.equal(
      timestamp,
      Math.floor(new Date('2030-01-02T03:04').getTime() / 1000)
    )
  })
})

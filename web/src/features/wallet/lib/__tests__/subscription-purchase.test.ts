import assert from 'node:assert/strict'
import test from 'node:test'

import { isSubscriptionPurchaseBlocked } from '../subscription-purchase.ts'

test('blocks a new purchase while the user has an active subscription', () => {
  assert.equal(isSubscriptionPurchaseBlocked(1), true)
})

test('allows a new purchase after all subscriptions have ended', () => {
  assert.equal(isSubscriptionPurchaseBlocked(0), false)
})

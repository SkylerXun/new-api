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
import { after, describe, test } from 'node:test'

import { Window } from 'happy-dom'

import type { ActivityCampaign } from '../../../types'

const domWindow = new Window()
for (const key of [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'CustomEvent',
  'MutationObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

const { act } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { ActivityCampaignsList } = await import('../activity-campaigns-list')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const campaign: ActivityCampaign = {
  id: 1,
  activity_key: 'claim-details-test',
  type: 'claimable',
  audience_type: 'all',
  status: 'active',
  title: 'Claim details test',
  description: '',
  reason: '',
  amount_usd: '5',
  quota: 2_500_000,
  starts_at: 1,
  ends_at: 2_000_000_000,
  recipient_max_user_id: 12,
  recipient_count: 12,
  granted_count: 3,
  created_by: 1,
  created_at: 1,
  updated_at: 1,
}

describe('activity campaign history', () => {
  after(() => domWindow.close())

  test('shows audience and claim counts and opens the matching details view', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)
    let selectedCampaign: ActivityCampaign | null = null

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <ActivityCampaignsList
            campaigns={[campaign]}
            isLoading={false}
            isError={false}
            isTaskProgressError={false}
            taskById={{}}
            onClose={() => {}}
            onViewGrants={(selected) => {
              selectedCampaign = selected
            }}
            onRetry={() => {}}
            onRetryTaskProgress={() => {}}
          />
        </I18nextProvider>
      )
    })

    assert.match(container.textContent, /Activity audience\s*12 users/)
    assert.match(container.textContent, /Claimed users\s*3 users/)
    assert.match(container.textContent, /Quota per user\s*2,500,000/)
    assert.equal(container.querySelector('[data-slot="card"]'), null)

    const detailsButton = [...container.querySelectorAll('button')].find(
      (button) => button.textContent?.includes('View claim details')
    )
    assert.ok(detailsButton)
    await act(async () => detailsButton.click())
    assert.equal(
      (selectedCampaign as ActivityCampaign | null)?.activity_key,
      campaign.activity_key
    )

    await act(async () => root.unmount())
    container.remove()
  })
})

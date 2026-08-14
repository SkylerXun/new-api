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

import type { UsageLog } from '../../data/schema'
import type { LogOtherData } from '../../types'

const domWindow = new Window()
const domGlobals = [
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
  'customElements',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const

for (const key of domGlobals) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}
Object.defineProperty(globalThis, 'matchMedia', {
  configurable: true,
  value: domWindow.matchMedia.bind(domWindow),
})

const React = await import('react')
const { act } = React
Object.defineProperty(globalThis, 'React', {
  configurable: true,
  value: React,
})
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { buildDetailSegments } = await import('../columns/common-logs-columns')
const { BillingBreakdown } = await import('../dialogs/details-dialog')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: { en: { translation: {} } },
})

const reactTestGlobals = globalThis as typeof globalThis & {
  IS_REACT_ACT_ENVIRONMENT?: boolean
}
reactTestGlobals.IS_REACT_ACT_ENVIRONMENT = true

const log: UsageLog = {
  id: 1,
  user_id: 1,
  created_at: 1,
  type: 2,
  content: '',
  username: 'user',
  token_name: 'token',
  model_name: 'model',
  quota: 10000,
  prompt_tokens: 100,
  completion_tokens: 20,
  use_time: 1,
  is_stream: false,
  channel: 1,
  channel_name: '',
  token_id: 1,
  group: 'default',
  ip: '',
  other: '',
  request_id: 'request',
  upstream_request_id: '',
}

const sensitiveBilling: LogOtherData = {
  model_ratio: 37.5,
  completion_ratio: 5,
  group_ratio: 0.2,
  cache_tokens: 10,
  cache_ratio: 0.1,
}

describe('usage log billing visibility', () => {
  after(() => domWindow.close())

  test('hides price segments from users while retaining them for admins', () => {
    const translate = (key: string) => key
    const userSegments = buildDetailSegments(
      log,
      sensitiveBilling,
      translate,
      false
    )
    const adminSegments = buildDetailSegments(
      log,
      sensitiveBilling,
      translate,
      true
    )

    assert.deepEqual(userSegments, [])
    assert.equal(adminSegments[0]?.text.includes('Standard'), true)
    assert.equal(adminSegments[0]?.text.includes('$75'), true)
  })

  test('shows users only total cost in the billing breakdown', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <BillingBreakdown
            log={log}
            other={sensitiveBilling}
            isAdmin={false}
          />
        </I18nextProvider>
      )
    })

    assert.equal(container.textContent?.includes('Total Cost'), true)
    assert.equal(container.textContent?.includes('Input'), false)
    assert.equal(container.textContent?.includes('Group Ratio'), false)
    assert.equal(container.textContent?.includes('$75'), false)

    await act(async () => root.unmount())
    container.remove()
  })

  test('keeps the full billing breakdown for admins', async () => {
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => {
      root.render(
        <I18nextProvider i18n={i18n}>
          <BillingBreakdown log={log} other={sensitiveBilling} isAdmin />
        </I18nextProvider>
      )
    })

    assert.equal(container.textContent?.includes('Input'), true)
    assert.equal(container.textContent?.includes('Group Ratio'), true)
    assert.equal(container.textContent?.includes('$75'), true)

    await act(async () => root.unmount())
    container.remove()
  })
})

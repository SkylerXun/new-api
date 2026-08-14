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

const domWindow = new Window()
for (const key of [
  'window',
  'document',
  'navigator',
  'HTMLElement',
  'HTMLButtonElement',
  'HTMLInputElement',
  'SVGElement',
  'Node',
  'Element',
  'Event',
  'KeyboardEvent',
  'PointerEvent',
  'CustomEvent',
  'MutationObserver',
  'ResizeObserver',
  'requestAnimationFrame',
  'cancelAnimationFrame',
  'getComputedStyle',
] as const) {
  Object.defineProperty(globalThis, key, {
    configurable: true,
    value: domWindow[key],
  })
}

let shouldReduceMotion = false
const mediaQuery = domWindow.matchMedia('(prefers-reduced-motion)')
Object.defineProperty(mediaQuery, 'matches', {
  configurable: true,
  get: () => shouldReduceMotion,
})
Object.defineProperty(domWindow, 'matchMedia', {
  configurable: true,
  value: () => mediaQuery,
})

function setReducedMotion(value: boolean) {
  shouldReduceMotion = value
  mediaQuery.dispatchEvent(new domWindow.Event('change'))
}

const { act, useState } = await import('react')
const { createRoot } = await import('react-dom/client')
const { createInstance } = await import('i18next')
const { I18nextProvider, initReactI18next } = await import('react-i18next')
const { ApiKeyGroupCombobox } = await import('../api-key-group-combobox')

const i18n = createInstance()
await i18n.use(initReactI18next).init({
  lng: 'en',
  resources: {
    en: {
      translation: {
        'Search...': 'Search...',
        'No group found.': 'No group found.',
        'Select a group': 'Select a group',
      },
    },
  },
})

Object.assign(globalThis, { IS_REACT_ACT_ENVIRONMENT: true })

const options = [
  {
    value: 'auto',
    label: 'auto',
    desc: 'Global automatic routing',
    ratioLabel: 'Automatic tier',
  },
  {
    value: 'default',
    label: 'default',
    desc: 'User group',
    ratioLabel: 'Standard tier',
  },
  {
    value: 'vip',
    label: 'vip',
    desc: 'Priority group',
    ratioLabel: 'Premium tier',
  },
]

function Harness(props: { initialValue: string }) {
  const [value, setValue] = useState(props.initialValue)
  return (
    <I18nextProvider i18n={i18n}>
      <ApiKeyGroupCombobox
        options={options}
        value={value}
        onValueChange={setValue}
      />
      <output data-testid='selected-group'>{value}</output>
    </I18nextProvider>
  )
}

function getTrigger(container: ParentNode): HTMLButtonElement {
  const trigger = container.querySelector<HTMLButtonElement>(
    'button[role="combobox"]'
  )
  assert.ok(trigger)
  return trigger
}

function getCommandItem(label: string): HTMLElement {
  const item = [
    ...document.querySelectorAll<HTMLElement>('[data-slot="command-item"]'),
  ].find((candidate) => candidate.textContent?.includes(label))
  assert.ok(item)
  return item
}

describe('API key group combobox', () => {
  after(() => domWindow.close())

  test('shows configured labels and applies the Auto motion treatment', async () => {
    setReducedMotion(false)
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => root.render(<Harness initialValue='auto' />))

    const trigger = getTrigger(container)
    assert.equal(trigger.dataset.autoGroupEffect, 'trigger')
    assert.ok(trigger.querySelector('[data-auto-group-flow-border]'))
    assert.equal(trigger.textContent?.includes('Automatic tier'), true)
    assert.equal(container.textContent?.includes('Ratio'), false)

    await act(async () => trigger.click())
    const autoOption = getCommandItem('Global automatic routing')
    assert.equal(autoOption.dataset.autoGroupEffect, 'option')
    assert.ok(autoOption.querySelector('[data-auto-group-flow-border]'))
    assert.equal(autoOption.textContent?.includes('Automatic tier'), true)

    const defaultOption = getCommandItem('User group')
    assert.equal(defaultOption.textContent?.includes('Standard tier'), true)
    assert.equal(defaultOption.textContent?.includes('1x'), false)
    assert.equal(defaultOption.textContent?.includes('Ratio'), false)

    await act(async () => root.unmount())
    container.remove()
  })

  test('keeps search and selection behavior for display labels', async () => {
    setReducedMotion(false)
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => root.render(<Harness initialValue='auto' />))

    const trigger = getTrigger(container)
    await act(async () => trigger.click())

    const searchInput = document.querySelector<HTMLInputElement>(
      'input[placeholder="Search..."]'
    )
    assert.ok(searchInput)
    await act(async () => {
      const valueSetter = Object.getOwnPropertyDescriptor(
        domWindow.HTMLInputElement.prototype,
        'value'
      )?.set
      assert.ok(valueSetter)
      valueSetter.call(searchInput, 'Premium tier')
      searchInput.dispatchEvent(
        new domWindow.Event('input', { bubbles: true }) as unknown as Event
      )
    })

    assert.equal(
      [
        ...document.querySelectorAll<HTMLElement>('[data-slot="command-item"]'),
      ].some((option) => option.textContent?.includes('Automatic tier')),
      false
    )
    const vipOption = getCommandItem('Premium tier')
    await act(async () => vipOption.click())

    assert.equal(
      container.querySelector('[data-testid="selected-group"]')?.textContent,
      'vip'
    )
    assert.equal(trigger.getAttribute('aria-expanded'), 'false')
    assert.equal(trigger.textContent?.includes('Premium tier'), true)
    assert.equal(trigger.hasAttribute('data-auto-group-effect'), false)
    assert.equal(trigger.querySelector('[data-auto-group-flow-border]'), null)

    await act(async () => root.unmount())
    container.remove()
  })

  test('preserves static Auto styling but omits motion when reduced', async () => {
    setReducedMotion(true)
    const container = document.createElement('div')
    document.body.append(container)
    const root = createRoot(container)

    await act(async () => root.render(<Harness initialValue='auto' />))

    const trigger = getTrigger(container)
    assert.equal(trigger.dataset.autoGroupEffect, 'trigger')
    assert.equal(trigger.querySelector('[data-auto-group-flow-border]'), null)
    assert.equal(trigger.textContent?.includes('Automatic tier'), true)

    await act(async () => trigger.click())
    const autoOption = getCommandItem('Global automatic routing')
    assert.equal(autoOption.dataset.autoGroupEffect, 'option')
    assert.equal(
      autoOption.querySelector('[data-auto-group-flow-border]'),
      null
    )
    assert.equal(autoOption.textContent?.includes('Automatic tier'), true)

    await act(async () => root.unmount())
    container.remove()
    setReducedMotion(false)
  })
})

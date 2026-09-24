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

import { Window } from 'happy-dom'
import { createInstance } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

import { ReferralOverviewCard } from '../referral-overview-card'

const testI18n = createInstance()
void testI18n.use(initReactI18next).init({
  lng: 'en',
  initAsync: false,
  resources: { en: { translation: {} } },
})

function renderReferralPage(affQuota = 100000) {
  const window = new Window()
  window.document.body.innerHTML = renderToStaticMarkup(
    <I18nextProvider i18n={testI18n}>
      <ReferralOverviewCard
        user={{ aff_quota: affQuota, aff_history_quota: 200000, aff_count: 3 }}
        referralLink='https://example.com/sign-up?aff=demo'
        onTransfer={() => undefined}
        complianceConfirmed
        rebateEnabled
        rebatePercent={10}
        loading={false}
      />
    </I18nextProvider>
  )
  return window
}

describe('referral program layout', () => {
  test('stacks the hero by default and switches to two columns when content space allows', () => {
    const window = renderReferralPage()
    const hero = window.document.querySelector(
      '[data-slot="referral-hero-grid"]'
    )

    assert.ok(hero)
    assert.match(hero.className, /grid/)
    assert.match(hero.className, /@3xl\/content:grid-cols-/)
    assert.doesNotMatch(hero.className, /(?:^|\s)grid-cols-2(?:\s|$)/)
    window.close()
  })

  test('keeps the three referral steps stacked until the medium breakpoint', () => {
    const window = renderReferralPage()
    const steps = window.document.querySelector('[data-slot="referral-steps"]')

    assert.ok(steps)
    assert.match(steps.className, /md:grid-cols-3/)
    assert.equal(steps.children.length, 3)
    window.close()
  })

  test('wraps the referral input and copy action on narrow screens', () => {
    const window = renderReferralPage()
    const input = window.document.querySelector('#referral-link')
    const controls = input?.parentElement

    assert.ok(controls)
    assert.match(controls.className, /flex-col/)
    assert.match(controls.className, /sm:flex-row/)
    window.close()
  })

  test('keeps the transfer action visible and disables it when no rebate is available', () => {
    const window = renderReferralPage(0)
    const transferButton = [...window.document.querySelectorAll('button')].find(
      (button) => button.textContent?.includes('Transfer to Balance')
    )

    assert.ok(transferButton)
    assert.equal(transferButton.disabled, true)
    window.close()
  })

  test('enables the transfer action when rebate is available', () => {
    const window = renderReferralPage(100000)
    const transferButton = [...window.document.querySelectorAll('button')].find(
      (button) => button.textContent?.includes('Transfer to Balance')
    )

    assert.ok(transferButton)
    assert.equal(transferButton.disabled, false)
    window.close()
  })

  test('explains that both balance-code redemptions and purchases earn rebates', () => {
    const window = renderReferralPage()

    assert.match(
      window.document.body.textContent,
      /redeems a balance code or purchases quota/
    )
    window.close()
  })
})

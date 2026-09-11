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
import { test } from 'node:test'

import { Window } from 'happy-dom'
import { createInstance } from 'i18next'
import { renderToStaticMarkup } from 'react-dom/server'
import { I18nextProvider, initReactI18next } from 'react-i18next'

import zh from '@/i18n/locales/zh.json'

import { StatementRecipientFields } from '../statement-recipient-fields'

test('statement recipient fields place the address after the title and keep profile values read-only', async () => {
  const i18n = createInstance()
  await i18n.use(initReactI18next).init({
    lng: 'zh',
    resources: { zh },
  })
  const window = new Window()
  window.document.body.innerHTML = renderToStaticMarkup(
    <I18nextProvider i18n={i18n}>
      <StatementRecipientFields
        billingTitle='Title'
        billingAddress='Address'
        billingUsername='Name'
        billingContact='Contact'
        onBillingTitleChange={() => undefined}
        onBillingAddressChange={() => undefined}
      />
    </I18nextProvider>
  )

  const inputs = [...window.document.querySelectorAll('input')]
  assert.deepEqual(
    inputs.map((input) => input.id),
    [
      'statement-title',
      'statement-address',
      'statement-username',
      'statement-contact',
    ]
  )
  assert.equal(inputs[0]?.readOnly, false)
  assert.equal(inputs[1]?.readOnly, false)
  assert.equal(inputs[2]?.readOnly, true)
  assert.equal(inputs[3]?.readOnly, true)
  assert.equal(
    window.document.querySelector('label[for="statement-username"]')
      ?.textContent,
    '本人姓名（可在个人资料修改）'
  )
  assert.equal(
    window.document.querySelector('label[for="statement-contact"]')
      ?.textContent,
    '联系方式（可在个人资料修改）'
  )

  window.close()
})

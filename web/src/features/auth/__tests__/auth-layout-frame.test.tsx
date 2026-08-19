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
import { renderToStaticMarkup } from 'react-dom/server'

import { AuthLayoutFrame } from '../auth-layout-frame'

describe('authentication layout frame', () => {
  test('keeps the brand and authentication content in one centered card', () => {
    const window = new Window()
    window.document.body.innerHTML = renderToStaticMarkup(
      <AuthLayoutFrame brand={<div data-testid='brand'>Brand</div>}>
        <div data-testid='auth-content'>Authentication content</div>
      </AuthLayoutFrame>
    )

    const card = window.document.querySelector('[data-slot="auth-card"]')
    assert.ok(card)
    assert.ok(card.querySelector('[data-testid="brand"]'))
    assert.ok(card.querySelector('[data-testid="auth-content"]'))
    assert.match(card.className, /max-w-\[520px\]/)
    assert.match(card.className, /auth-light-surface/)

    window.close()
  })

  test('renders a non-interactive backdrop and allows long pages to scroll', () => {
    const window = new Window()
    window.document.body.innerHTML = renderToStaticMarkup(
      <AuthLayoutFrame brand={<div>Brand</div>}>
        <div>Authentication content</div>
      </AuthLayoutFrame>
    )

    const page = window.document.querySelector('[data-slot="auth-page"]')
    const backdrop = window.document.querySelector(
      '[data-slot="auth-background"]'
    )
    assert.ok(page)
    assert.ok(backdrop)
    assert.match(page.className, /min-h-svh/)
    assert.doesNotMatch(page.className, /h-svh(?:\s|$)/)
    assert.equal(backdrop.getAttribute('aria-hidden'), 'true')
    assert.match(backdrop.className, /pointer-events-none/)

    window.close()
  })
})

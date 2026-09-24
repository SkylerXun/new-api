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

import type { TFunction } from 'i18next'
import { Flame } from 'lucide-react'

import { getSidebarData } from '../use-sidebar-data'

const translate = ((key: string) => key) as TFunction

test('referral navigation uses a flame accent to draw attention', () => {
  const data = getSidebarData(translate, false)
  const referral = data.navGroups
    .find((group) => group.id === 'personal')
    ?.items.find((item) => 'url' in item && item.url === '/referrals')

  assert.ok(referral)
  assert.equal(referral.accentIcon, Flame)
})

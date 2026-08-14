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

import type { TFunction } from 'i18next'

import type { SidebarData } from '@/components/layout/types'

import { filterSidebarNavGroups } from '../use-sidebar-config'
import { getSidebarData } from '../use-sidebar-data'

const bunTestModule = 'bun:test'
const { test } = (await import(bunTestModule)) as {
  test: typeof import('node:test').test
}

const translate = ((key: string) => key) as TFunction

function personalUrls(data: SidebarData): string[] {
  const personal = data.navGroups.find((group) => group.id === 'personal')
  assert.ok(personal)
  return personal.items.flatMap((item) =>
    'url' in item && item.url ? [String(item.url)] : []
  )
}

test('enabled check-in appears immediately after profile', () => {
  const urls = personalUrls(getSidebarData(translate, true))

  assert.equal(urls.indexOf('/checkin'), urls.indexOf('/profile') + 1)
})

test('disabled check-in feature omits the sidebar entry', () => {
  const urls = personalUrls(getSidebarData(translate, false))

  assert.equal(urls.includes('/checkin'), false)
})

test('admin check-in setting hides the enabled sidebar entry', () => {
  const data = getSidebarData(translate, true)
  const filtered = filterSidebarNavGroups(
    data.navGroups,
    JSON.stringify({
      personal: { enabled: true, checkin: false },
    }),
    null
  )

  assert.equal(
    personalUrls({ navGroups: filtered }).includes('/checkin'),
    false
  )
})

test('user check-in setting hides the enabled sidebar entry', () => {
  const data = getSidebarData(translate, true)
  const filtered = filterSidebarNavGroups(
    data.navGroups,
    null,
    JSON.stringify({
      personal: { enabled: true, checkin: false },
    })
  )

  assert.equal(
    personalUrls({ navGroups: filtered }).includes('/checkin'),
    false
  )
})

test('legacy sidebar config without check-in keeps the entry visible', () => {
  const data = getSidebarData(translate, true)
  const filtered = filterSidebarNavGroups(
    data.navGroups,
    JSON.stringify({
      personal: { enabled: true, personal: true },
    }),
    JSON.stringify({
      personal: { enabled: true, personal: true },
    })
  )

  assert.equal(personalUrls({ navGroups: filtered }).includes('/checkin'), true)
})

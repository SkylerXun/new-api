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
import { afterEach, describe, test } from 'node:test'

import { api, getUserModels } from './api'

const apiClient = api as unknown as {
  get: (url: string) => Promise<{ data: unknown }>
}
const originalGet = apiClient.get

afterEach(() => {
  apiClient.get = originalGet
})

describe('getUserModels', () => {
  test('keeps the unfiltered endpoint when no group is supplied', async () => {
    const urls: string[] = []
    apiClient.get = async (url) => {
      urls.push(url)
      return { data: { success: true, data: [] } }
    }

    await getUserModels()

    assert.deepEqual(urls, ['/api/user/models'])
  })

  test('requests models for a selected group', async () => {
    const urls: string[] = []
    apiClient.get = async (url) => {
      urls.push(url)
      return { data: { success: true, data: ['gpt-test'] } }
    }

    const result = await getUserModels({ group: 'vip group' })

    assert.deepEqual(urls, ['/api/user/models?group=vip%20group'])
    assert.deepEqual(result.data, ['gpt-test'])
  })

  test('passes the auto group through unchanged', async () => {
    const urls: string[] = []
    apiClient.get = async (url) => {
      urls.push(url)
      return { data: { success: true, data: [] } }
    }

    await getUserModels({ group: 'auto' })

    assert.deepEqual(urls, ['/api/user/models?group=auto'])
  })
})

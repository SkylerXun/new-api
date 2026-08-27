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

import type { Channel } from '../../types'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  channelFormSchema,
  transformChannelToFormDefaults,
  transformFormDataToCreatePayload,
  transformFormDataToUpdatePayload,
} from '../channel-form'

const validMapping = JSON.stringify({
  '503': '服务繁忙，请稍后重试',
  default: '服务暂时不可用',
})

function formWithMapping(errorMessageMapping: string) {
  return {
    ...CHANNEL_FORM_DEFAULT_VALUES,
    name: 'Mapped errors',
    key: 'test-key',
    models: 'gpt-5',
    error_message_mapping: errorMessageMapping,
  }
}

describe('channel error message mapping', () => {
  test('accepts HTTP status keys and default with non-empty messages', () => {
    assert.equal(
      channelFormSchema.safeParse(formWithMapping(validMapping)).success,
      true
    )
  })

  test('rejects invalid keys and empty or non-string messages', () => {
    const invalidMappings = [
      { '99': 'too low' },
      { '600': 'too high' },
      { fallback: 'unsupported key' },
      { '503': '   ' },
      { '503': 503 },
    ]

    for (const mapping of invalidMappings) {
      const result = channelFormSchema.safeParse(
        formWithMapping(JSON.stringify(mapping))
      )
      assert.equal(result.success, false)
    }
  })

  test('loads and sends the mapping, including an explicit clear on update', () => {
    const channel = {
      id: 7,
      type: 1,
      name: 'Mapped errors',
      status: 1,
      error_message_mapping: validMapping,
      channel_info: { multi_key_mode: 'random' },
    } as Channel

    assert.equal(
      transformChannelToFormDefaults(channel).error_message_mapping,
      validMapping
    )
    assert.equal(
      transformFormDataToCreatePayload(formWithMapping(validMapping)).channel
        .error_message_mapping,
      validMapping
    )
    assert.equal(
      transformFormDataToUpdatePayload(formWithMapping(''), channel.id)
        .error_message_mapping,
      ''
    )
  })
})

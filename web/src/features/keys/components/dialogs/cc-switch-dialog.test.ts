import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import { buildCCSwitchURL } from './cc-switch-url.ts'

describe('CC Switch import link', () => {
  test('normalizes a trailing slash before adding the Codex API path', () => {
    const url = buildCCSwitchURL(
      'codex',
      'test',
      { model: 'gpt-5.5' },
      'sk-test',
      'https://api.example.com/'
    )
    const params = new URL(url).searchParams

    assert.equal(params.get('endpoint'), 'https://api.example.com/v1')
    assert.equal(params.get('homepage'), 'https://api.example.com')
    assert.equal(params.get('enabled'), 'true')
    assert.equal(params.get('apiKey'), 'sk-test')
    assert.equal(params.get('usageEnabled'), 'true')
    assert.equal(params.get('usageAutoInterval'), '30')
    assert.match(atob(params.get('usageScript') || ''), /\{\{baseUrl\}\}\/v1\/usage/)
    assert.match(atob(params.get('usageScript') || ''), /Bearer \{\{apiKey\}\}/)
  })
})

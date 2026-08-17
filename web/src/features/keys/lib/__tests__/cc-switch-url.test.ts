import { describe, expect, it } from 'vitest'

import { resolveCcSwitchServerAddress } from '../cc-switch-url'

describe('resolveCcSwitchServerAddress', () => {
  it('uses the current public origin when a stale localhost address is configured', () => {
    expect(
      resolveCcSwitchServerAddress(
        'http://localhost:3000',
        'https://api.example.com'
      )
    ).toBe('https://api.example.com')
  })

  it('keeps localhost for local development', () => {
    expect(
      resolveCcSwitchServerAddress('http://localhost:3000', 'http://localhost:3000')
    ).toBe('http://localhost:3000')
  })

  it('falls back to the current origin when no address is configured', () => {
    expect(resolveCcSwitchServerAddress('', 'https://api.example.com/')).toBe(
      'https://api.example.com'
    )
  })
})

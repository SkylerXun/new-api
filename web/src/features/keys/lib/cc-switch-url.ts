/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.
*/

function isLoopbackHost(hostname: string): boolean {
  const host = hostname.toLowerCase()
  return host === 'localhost' || host === '127.0.0.1' || host === '::1' || host === '[::1]'
}

function isLoopbackUrl(value: string): boolean {
  try {
    return isLoopbackHost(new URL(value).hostname)
  } catch {
    return false
  }
}

/** Prefer the browser origin when a stale local development address is published. */
export function resolveCcSwitchServerAddress(
  configuredAddress: string | undefined,
  currentOrigin: string
): string {
  const configured = configuredAddress?.trim().replace(/\/+$/, '')
  const origin = currentOrigin.trim().replace(/\/+$/, '')
  if (!configured) return origin
  if (origin && !isLoopbackUrl(origin) && isLoopbackUrl(configured)) return origin
  return configured
}

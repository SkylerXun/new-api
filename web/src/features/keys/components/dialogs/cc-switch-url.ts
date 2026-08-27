export function getServerAddress(): string {
  try {
    const raw = localStorage.getItem('status')
    if (raw) {
      const status = JSON.parse(raw)
      if (status.server_address) return status.server_address
    }
  } catch {
    /* empty */
  }
  return window.location.origin
}

export function buildCCSwitchURL(
  app: string,
  name: string,
  models: Record<string, string>,
  apiKey: string,
  serverAddress = getServerAddress()
): string {
  const normalizedServerAddress = serverAddress.replace(/\/+$/, '')
  const endpoint =
    app === 'codex'
      ? `${normalizedServerAddress}/v1`
      : normalizedServerAddress
  const params = new URLSearchParams()
  params.set('resource', 'provider')
  params.set('app', app)
  params.set('name', name)
  params.set('endpoint', endpoint)
  params.set('apiKey', apiKey)
  for (const [key, value] of Object.entries(models)) {
    if (value) params.set(key, value)
  }
  params.set('homepage', normalizedServerAddress)
  params.set('enabled', 'true')
  const usageScript = `({
    request: {
      url: "{{baseUrl}}/v1/usage",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      const remaining = response?.remaining ?? response?.quota?.remaining ?? response?.balance;
      const unit = response?.unit ?? response?.quota?.unit ?? "USD";
      return {
        isValid: response?.is_active ?? response?.isValid ?? true,
        remaining,
        unit
      };
    }
  })`
  params.set('configFormat', 'json')
  params.set('usageEnabled', 'true')
  params.set('usageScript', btoa(usageScript))
  params.set('usageAutoInterval', '30')
  return `ccswitch://v1/import?${params.toString()}`
}

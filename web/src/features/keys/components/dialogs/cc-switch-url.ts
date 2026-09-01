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
  // The Codex provider endpoint already contains the OpenAI `/v1` prefix.
  // CCSwitch substitutes `{{baseUrl}}` literally, so appending `/v1/usage`
  // here would probe `/v1/v1/usage` and silently leave the balance blank.
  const usagePath = app === 'codex' ? '/usage' : '/v1/usage'
  const usageScript = `({
    request: {
      url: "{{baseUrl}}${usagePath}",
      method: "GET",
      headers: { "Authorization": "Bearer {{apiKey}}" }
    },
    extractor: function(response) {
      // Keep this ES5-compatible: older CCSwitch builds do not parse optional
      // chaining or nullish coalescing, causing the extractor to fail before
      // it can read the response.
      response = response || {};
      const quota = response.quota || {};
      const data = response.data || {};
      const remaining = response.remaining != null ? response.remaining
        : (quota.remaining != null ? quota.remaining
          : (data.remaining != null ? data.remaining : response.balance));
      const unit = response.unit || quota.unit || data.unit || "USD";
      return {
        isValid: response.is_active != null ? response.is_active
          : (response.isValid != null ? response.isValid : true),
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

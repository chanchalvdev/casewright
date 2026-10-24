import { afterEach, describe, expect, it, vi } from 'vitest'

// The key is read when the module loads, so each case re-imports it.
async function loadApi(key: string, name = 'VITE_CASEWRIGHT_API_KEY') {
  vi.stubEnv('VITE_CASEWRIGHT_API_KEY', '')
  vi.stubEnv('VITE_SIEM_API_KEY', '')
  vi.stubEnv(name, key)
  vi.resetModules()
  return import('../lib/api')
}

afterEach(() => {
  vi.unstubAllEnvs()
})

describe('API key helpers', () => {
  it('sends no auth when no key is configured', async () => {
    const api = await loadApi('')
    expect(api.authHeaders()).toEqual({})
    expect(api.withApiKey('ws://host/ws/alerts')).toBe('ws://host/ws/alerts')
  })

  it('adds the key as a header and as a WebSocket query parameter', async () => {
    const api = await loadApi('s3cret&x=1')
    expect(api.authHeaders()).toEqual({ 'X-API-Key': 's3cret&x=1' })
    expect(api.withApiKey('ws://host/ws/alerts')).toBe('ws://host/ws/alerts?api_key=s3cret%26x%3D1')
    expect(api.withApiKey('ws://host/ws?a=b')).toBe('ws://host/ws?a=b&api_key=s3cret%26x%3D1')
  })

  it('still reads the key from its pre-rename name', async () => {
    const api = await loadApi('old-key', 'VITE_SIEM_API_KEY')
    expect(api.authHeaders()).toEqual({ 'X-API-Key': 'old-key' })
  })
})

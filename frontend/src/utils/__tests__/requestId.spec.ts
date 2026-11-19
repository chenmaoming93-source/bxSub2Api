import { afterEach, describe, expect, it, vi } from 'vitest'
import { createRequestId } from '../requestId'

describe('createRequestId', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('returns UUID-shaped unique request IDs', () => {
    const first = createRequestId()
    const second = createRequestId()
    expect(first).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i)
    expect(second).not.toBe(first)
  })

  it('falls back when randomUUID is unavailable on an insecure origin', () => {
    vi.stubGlobal('crypto', {
      getRandomValues(bytes: Uint8Array) {
        bytes.fill(0)
        return bytes
      },
    })

    expect(createRequestId()).toBe('00000000-0000-4000-8000-000000000000')
  })
})

import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get }
}))

import { adminAPI } from '@/api/admin'

describe('admin accounts api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  it('gets and unwraps the available models for an account', async () => {
    const models = [
      { id: 'claude-sonnet-4', display_name: 'Claude Sonnet 4' },
      { id: 'claude-opus-4' }
    ]
    const controller = new AbortController()
    get.mockResolvedValue({ data: models })

    await expect(
      adminAPI.accounts.getAvailableModels(42, { signal: controller.signal })
    ).resolves.toEqual(models)
    expect(get).toHaveBeenCalledWith('/admin/accounts/42/models', {
      signal: controller.signal
    })
  })
})

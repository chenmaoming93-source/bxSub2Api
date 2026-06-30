import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, put } = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get, put }
}))

import {
  getGlobalModelTokenQuotas,
  getUserModelTokenQuotas,
  updateGlobalModelTokenQuota,
  updateUserModelTokenQuotas,
  type ModelTokenQuotaUpdateItem
} from '@/api/admin/modelTokenQuotas'

describe('admin model token quota api', () => {
  beforeEach(() => {
    get.mockReset()
    put.mockReset()
  })

  it('uses the backend global GET and PUT route and preserves limit values', async () => {
    const quotas = [
      { model: 'unlimited-null', usage_date: '2026-06-30', used_tokens: 1, daily_limit_tokens: null },
      { model: 'unlimited-zero', usage_date: '2026-06-30', used_tokens: 2, daily_limit_tokens: 0 },
      { model: 'limited', usage_date: '2026-06-30', used_tokens: 3, daily_limit_tokens: 100 }
    ]
    get.mockResolvedValue({ data: { quotas } })
    put.mockImplementation((_url: string, input: ModelTokenQuotaUpdateItem) =>
      Promise.resolve({ data: { quota: { ...quotas[0], ...input } } })
    )

    expect(await getGlobalModelTokenQuotas()).toEqual({ quotas })
    expect(get).toHaveBeenCalledWith('/admin/model-token-quotas')

    for (const daily_limit_tokens of [null, 0, 100]) {
      const input = { model: 'gpt-5', daily_limit_tokens }
      expect((await updateGlobalModelTokenQuota(input)).daily_limit_tokens).toBe(daily_limit_tokens)
      expect(put).toHaveBeenLastCalledWith('/admin/model-token-quotas', input)
    }
  })

  it('uses the backend user GET and PUT route with the exact quota payload', async () => {
    const response = {
      quotas: [
        { user_id: 42, model: 'gpt-5', usage_date: '2026-06-30', used_tokens: 7, daily_limit_tokens: 500 }
      ]
    }
    get.mockResolvedValue({ data: response })
    put.mockResolvedValue({ data: response })
    const payload: ModelTokenQuotaUpdateItem[] = [
      { model: 'gpt-5', daily_limit_tokens: null },
      { model: 'claude-sonnet', daily_limit_tokens: 0 },
      { model: 'gemini-pro', daily_limit_tokens: 900 }
    ]

    expect(await getUserModelTokenQuotas(42)).toEqual(response)
    expect(get).toHaveBeenCalledWith('/admin/users/42/model-token-quotas')
    expect(await updateUserModelTokenQuotas(42, payload)).toEqual(response)
    expect(put).toHaveBeenCalledWith('/admin/users/42/model-token-quotas', { quotas: payload })
  })
})

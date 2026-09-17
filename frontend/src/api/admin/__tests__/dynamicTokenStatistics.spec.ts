import { beforeEach, describe, expect, it, vi } from 'vitest'

const client = vi.hoisted(() => ({
  get: vi.fn(),
  post: vi.fn(),
  put: vi.fn()
}))

vi.mock('@/api/client', () => ({ apiClient: client }))

import { dynamicTokenStatisticsAPI } from '@/api/admin/dynamicTokenStatistics'

describe('dynamic token statistics admin api', () => {
  beforeEach(() => {
    client.get.mockReset()
    client.post.mockReset()
    client.put.mockReset()
  })

  it('uses only the new token-statistics namespace', async () => {
    client.get.mockResolvedValue({ data: { projections: [] } })
    await expect(dynamicTokenStatisticsAPI.projections()).resolves.toEqual([])
    expect(client.get).toHaveBeenCalledWith('/admin/token-statistics/projections')

    client.post.mockResolvedValue({ data: { quota: { id: 1 } } })
    await dynamicTokenStatisticsAPI.createQuota({
      name: 'daily user', dimension_codes: ['user_id'],
      dimension_values: { user_id: { type: 'int64', int64: 1 } },
      metric_code: 'total_tokens', period_type: 'D', limit_value: 100, mode: 'OBSERVE'
    })
    expect(client.post).toHaveBeenCalledWith('/admin/token-statistics/quotas', expect.any(Object))
  })

  it('resets quota usage through the admin contract', async () => {
    const input = {
      dimension_values: {
        user_id: { type: 'int64' as const, int64: 42 },
        group_id: { type: 'int64' as const, int64: 7 }
      },
      metric_code: 'total_tokens' as const,
      period_type: 'D' as const
    }
    const result = { status: 'RESET', matched_quota_count: 2, matched_usage_count: 3, reset_count: 3, no_usage_count: 0, failed_count: 0 }
    client.post.mockResolvedValueOnce({ data: result })
    await expect(dynamicTokenStatisticsAPI.resetQuotaUsage(input)).resolves.toEqual(result)
    expect(client.post).toHaveBeenCalledWith('/admin/token-statistics/quota-usage/reset', input)
  })

  it('queries and exports through the generic endpoint', async () => {
    const input = {
      projection_id: 1, metric_code: 'total_tokens' as const, period_type: 'D' as const,
      start: '2026-07-01T00:00:00Z', end: '2026-07-02T00:00:00Z'
    }
    client.post.mockResolvedValueOnce({ data: { rows: [], total: 0, summary: 0 } })
    await dynamicTokenStatisticsAPI.query(input)
    expect(client.post).toHaveBeenLastCalledWith('/admin/token-statistics/query', input)

    const blob = new Blob()
    client.post.mockResolvedValueOnce({ data: blob })
    await expect(dynamicTokenStatisticsAPI.exportCSV(input)).resolves.toBe(blob)
    expect(client.post).toHaveBeenLastCalledWith('/admin/token-statistics/query', expect.objectContaining({ format: 'csv' }), { responseType: 'blob' })
  })
})

import { describe, expect, it, vi } from 'vitest'

const { getMock } = vi.hoisted(() => ({ getMock: vi.fn() }))
vi.mock('../../client', () => ({
  apiClient: { get: getMock },
}))

import {
  buildObservabilityParams,
  getObservabilityPaths,
  getObservabilitySnapshot,
} from '../observability'

describe('observability API parameters', () => {
  it('uses canonical repeated parameters and never display names', () => {
    const params = buildObservabilityParams({
      window: '1h',
      bucket: '5m',
      search: '  alice ',
      filters: {
        userIds: [1, 2],
        apiKeyIds: [],
        groupIds: [7],
        routeAliases: ['gpt-*', 'coding'],
        accountIds: [101],
        department: '研发部',
      },
    })

    expect(params.getAll('user_id')).toEqual(['1', '2'])
    expect(params.getAll('group_id')).toEqual(['7'])
    expect(params.getAll('route_alias')).toEqual(['gpt-*', 'coding'])
    expect(params.getAll('account_id')).toEqual(['101'])
    expect(params.get('department')).toBe('研发部')
    expect(params.get('search')).toBe('alice')
    expect(params.toString()).not.toContain('Alice Example')
  })

  it('accepts the response payload after the global interceptor unwraps it', async () => {
    const payload = {
      meta: { warnings: [], complete: true },
      metrics: [],
      top_users: [],
      top_groups: [],
      top_aliases: [],
      top_accounts: [],
      top_models: [],
      pressure: { accounts: [], candidates: [] },
      filter_options: { users: [], groups: [], aliases: [], accounts: [] },
    }
    getMock.mockResolvedValueOnce({
      data: payload,
      status: 200,
      headers: { etag: '"etag"' },
    })
    const result = await getObservabilitySnapshot({
      window: '1h', bucket: '5m',
      filters: { userIds: [], apiKeyIds: [], groupIds: [], routeAliases: [], accountIds: [], department: '' },
    })
    expect(result.data).toEqual(payload)
    expect(result.notModified).toBe(false)
  })

  it('sends normalized absolute range parameters', () => {
    const params = buildObservabilityParams({
      window: 'custom',
      bucket: '15m',
      startAt: '2026-11-12T09:00:00+08:00',
      endAt: '2026-11-12T11:30:00+08:00',
      filters: { userIds: [], apiKeyIds: [], groupIds: [], routeAliases: [], accountIds: [], department: '' },
    })

    expect(params.get('window')).toBe('custom')
    expect(params.get('bucket')).toBe('15m')
    expect(params.get('start_at')).toBe('2026-11-12T09:00:00+08:00')
    expect(params.get('end_at')).toBe('2026-11-12T11:30:00+08:00')
  })

  it('keeps pagination and filters in the path request', async () => {
    getMock.mockResolvedValueOnce({
      data: { meta: { warnings: [], complete: true }, total: 0, rows: [] },
      status: 200, headers: { etag: '"paths"' },
    })
    await getObservabilityPaths({
      window: '6h', bucket: '15m', search: 'alice', page: 3, pageSize: 20,
      sort: 'error_rate', status: 'degraded', includeIdle: true,
      filters: { userIds: [9], apiKeyIds: [], groupIds: [7], routeAliases: [], accountIds: [101], department: '' },
    })
    const url = String(getMock.mock.calls.at(-1)?.[0])
    expect(url).toContain('page=3')
    expect(url).toContain('page_size=20')
    expect(url).toContain('sort=error_rate')
    expect(url).toContain('status=degraded')
    expect(url).toContain('include_idle=true')
    expect(url).toContain('user_id=9')
    expect(url).toContain('search=alice')
  })
})

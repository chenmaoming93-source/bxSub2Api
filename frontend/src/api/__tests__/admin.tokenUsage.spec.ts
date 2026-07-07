import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn()
}))

vi.mock('@/api/client', () => ({
  apiClient: { get }
}))

import {
  getModelTokenUsageReport,
  getRouteTokenUsageReport,
  getUserModelTokenUsageReport,
  searchModelOptions,
  searchGroupOptions,
  searchRouteOptions,
  searchRouteModelOptions,
  searchUserOptions,
  searchUserModelOptions,
  getDefaultTarget,
  type ModelReportParams,
  type RouteReportParams,
  type UserReportParams
} from '@/api/admin/tokenUsage'

describe('admin token usage api', () => {
  beforeEach(() => {
    get.mockReset()
  })

  describe('report endpoints', () => {
    it('calls the model report endpoint with correct params', async () => {
      const expected = { items: [], summary: { used_tokens: 0 }, pagination: { page: 1, page_size: 20, total: 0 } }
      get.mockResolvedValue({ data: expected })
      const params: ModelReportParams = { model: 'gpt-4', start_date: '2026-07-01', end_date: '2026-07-06', page: 1, page_size: 20, sort_by: 'usage_date', sort_order: 'desc' }
      const result = await getModelTokenUsageReport(params)
      expect(result).toEqual(expected)
      expect(get).toHaveBeenCalledWith('/admin/token-usage/models', { params })
    })

    it('calls the route report endpoint with required params', async () => {
      const expected = { items: [], summary: { used_tokens: 0 }, pagination: { page: 1, page_size: 20, total: 0 } }
      get.mockResolvedValue({ data: expected })
      const params: RouteReportParams = { group_id: 1, route_alias: 'fast', start_date: '2026-07-06', end_date: '2026-07-06', page: 1, page_size: 20, sort_by: 'usage_date', sort_order: 'desc' }
      const result = await getRouteTokenUsageReport(params)
      expect(result).toEqual(expected)
      expect(get).toHaveBeenCalledWith('/admin/token-usage/routes', { params })
    })

    it('calls the route report endpoint with optional upstream_model', async () => {
      get.mockResolvedValue({ data: { items: [], summary: { used_tokens: 0 }, pagination: { page: 1, page_size: 20, total: 0 } } })
      const params: RouteReportParams = { group_id: 1, route_alias: 'fast', upstream_model: 'gpt-4', start_date: '2026-07-06', end_date: '2026-07-06', page: 1, page_size: 20, sort_by: 'used_tokens', sort_order: 'asc' }
      await getRouteTokenUsageReport(params)
      expect(get).toHaveBeenCalledWith('/admin/token-usage/routes', { params })
    })

    it('calls the user report endpoint with user_id required', async () => {
      get.mockResolvedValue({ data: { items: [], summary: { used_tokens: 0 }, pagination: { page: 1, page_size: 20, total: 0 } } })
      const params: UserReportParams = { user_id: 42, start_date: '2026-07-06', end_date: '2026-07-06', page: 1, page_size: 20, sort_by: 'usage_date', sort_order: 'desc' }
      await getUserModelTokenUsageReport(params)
      expect(get).toHaveBeenCalledWith('/admin/token-usage/users', { params })
    })

    it('calls the user report endpoint with optional model filter', async () => {
      get.mockResolvedValue({ data: { items: [], summary: { used_tokens: 0 }, pagination: { page: 1, page_size: 20, total: 0 } } })
      const params: UserReportParams = { user_id: 42, model: 'gpt-4', start_date: '2026-07-06', end_date: '2026-07-06', page: 1, page_size: 20, sort_by: 'usage_date', sort_order: 'desc' }
      await getUserModelTokenUsageReport(params)
      expect(get).toHaveBeenCalledWith('/admin/token-usage/users', { params })
    })

    it('passes the deleted-user search option', async () => {
      get.mockResolvedValue({ data: { items: [], summary: { used_tokens: 0 }, pagination: { page: 1, page_size: 20, total: 0 } } })
      const params: UserReportParams = { include_deleted: true, start_date: '2026-07-06', end_date: '2026-07-06' }
      await getUserModelTokenUsageReport(params)
      expect(get).toHaveBeenCalledWith('/admin/token-usage/users', { params })
    })
  })

  describe('options endpoints', () => {
    it('searches models with bounded limit', async () => {
      get.mockResolvedValue({ data: { items: [{ id: 0, label: 'gpt-4', model: 'gpt-4' }] } })
      const result = await searchModelOptions('gpt', 20)
      expect(result).toEqual({ items: [{ id: 0, label: 'gpt-4', model: 'gpt-4' }] })
      expect(get).toHaveBeenCalledWith('/admin/token-usage/options/models', { params: { q: 'gpt', limit: 20 } })
    })

    it('searches groups', async () => {
      get.mockResolvedValue({ data: { items: [{ id: 1, label: 'Test Group', group_id: 1 }] } })
      await searchGroupOptions('test')
      expect(get).toHaveBeenCalledWith('/admin/token-usage/options/groups', { params: { q: 'test', limit: 20 } })
    })

    it('searches routes for a given group', async () => {
      get.mockResolvedValue({ data: { items: [] } })
      await searchRouteOptions(1, 'fast')
      expect(get).toHaveBeenCalledWith('/admin/token-usage/options/groups/1/routes', { params: { q: 'fast', limit: 20 } })
    })

    it('searches upstream models for a selected route', async () => {
      get.mockResolvedValue({ data: { items: [] } })
      await searchRouteModelOptions(1, 'fast route', 'gpt')
      expect(get).toHaveBeenCalledWith('/admin/token-usage/options/groups/1/routes/fast%20route/models', { params: { q: 'gpt', limit: 20 } })
    })

    it('searches users', async () => {
      get.mockResolvedValue({ data: { items: [] } })
      await searchUserOptions('alice')
      expect(get).toHaveBeenCalledWith('/admin/token-usage/options/users', { params: { q: 'alice', limit: 20 } })
    })

    it('searches user models', async () => {
      get.mockResolvedValue({ data: { items: [] } })
      await searchUserModelOptions(42, 'gpt')
      expect(get).toHaveBeenCalledWith('/admin/token-usage/options/users/42/models', { params: { q: 'gpt', limit: 20 } })
    })
  })

  describe('default target', () => {
    it('requests default target for a dimension', async () => {
      const target = { id: 0, label: 'gpt-4', model: 'gpt-4' }
      get.mockResolvedValue({ data: { target } })
      const result = await getDefaultTarget('model')
      expect(result).toEqual({ target })
      expect(get).toHaveBeenCalledWith('/admin/token-usage/default-target', { params: { dimension: 'model', date: undefined } })
    })

    it('includes date when provided', async () => {
      get.mockResolvedValue({ data: { target: null } })
      await getDefaultTarget('user', '2026-07-06')
      expect(get).toHaveBeenCalledWith('/admin/token-usage/default-target', { params: { dimension: 'user', date: '2026-07-06' } })
    })

    it('returns null target when backend has no data', async () => {
      get.mockResolvedValue({ data: { target: null } })
      const result = await getDefaultTarget('route', '2026-07-06')
      expect(result.target).toBeNull()
    })
  })
})

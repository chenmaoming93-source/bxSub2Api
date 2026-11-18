import { describe, expect, it } from 'vitest'
import {
  appendUniqueByEntryId,
  canRevokeTaskQuota,
  isOverdueFailed,
  orderMatches,
  productPublishError,
  reasonIsValid,
  taskActions,
  taskRuleDifferences,
} from '../growthRules'
import type { ExchangeOrder, GrowthProduct, GrowthTask } from '@/api/growth'

const baseTask = {
  status: 'DRAFT',
  pending_rule_version: undefined,
} as Pick<GrowthTask, 'status' | 'pending_rule_version'>

describe('growth page rules', () => {
  it('derives task actions for each lifecycle state without delete', () => {
    expect(taskActions(baseTask, true)).toEqual(['edit', 'history', 'publish', 'quota'])
    expect(taskActions({ status: 'ACTIVE' }, true)).toEqual(['edit', 'history', 'offline', 'quota'])
    expect(taskActions({ status: 'ACTIVE', pending_rule_version: 3 }, true)).toEqual(['edit', 'history', 'publish', 'offline', 'quota'])
    expect(taskActions({ status: 'OFFLINE' }, true)).toEqual(['edit', 'history', 'online'])
    expect(taskActions({ status: 'ACTIVE' }, false)).toEqual(['history'])
    expect(taskActions({ status: 'ACTIVE' }, true)).not.toContain('delete')
  })

  it('validates dangerous operation inputs and quota reclaim state', () => {
    expect(reasonIsValid('调整奖励')).toBe(false)
    expect(reasonIsValid('调整任务奖励规则')).toBe(true)
    expect(canRevokeTaskQuota({ status: 'ACTIVE' })).toBe(false)
    expect(canRevokeTaskQuota({ status: 'OFFLINE' })).toBe(true)
  })

  it('blocks invalid automatic token products', () => {
    const product = { product_type: 'TOKEN_QUOTA', fulfillment_mode: 'AUTOMATIC', token_amount: 0 } as Pick<GrowthProduct, 'product_type' | 'fulfillment_mode' | 'token_amount'>
    expect(productPublishError(product)).toBe('INVALID_TOKEN_PRODUCT')
    expect(productPublishError({ ...product, token_amount: 100 })).toBeNull()
  })

  it('filters orders and deduplicates cursor pages', () => {
    const order = {
      order_no: 'EX202611180001',
      status: 'FAILED',
      points_cost: 50,
      product_snapshot: { title: '100万 Token' },
      created_at: '2026-11-18T08:00:00+08:00',
    } as ExchangeOrder
    expect(orderMatches(order, { search: '0001', status: 'FAILED', product: '100万 Token', from: '2026-11-18', to: '2026-11-18' })).toBe(true)
    expect(orderMatches(order, { from: '2026-11-19' })).toBe(false)
    expect(appendUniqueByEntryId([{ entry_id: 2 }, { entry_id: 1 }], [{ entry_id: 1 }, { entry_id: 0 }])).toEqual([
      { entry_id: 2 }, { entry_id: 1 }, { entry_id: 0 },
    ])
  })

  it('only marks failed orders older than 24 hours as overdue', () => {
    const now = Date.parse('2026-11-18T12:00:00+08:00')
    expect(isOverdueFailed({ status: 'FAILED', created_at: '2026-11-17T11:59:59+08:00' }, now)).toBe(true)
    expect(isOverdueFailed({ status: 'FAILED', created_at: '2026-11-17T12:00:01+08:00' }, now)).toBe(false)
    expect(isOverdueFailed({ status: 'REFUNDED', created_at: '2026-11-16T12:00:00+08:00' }, now)).toBe(false)
  })

  it('shows only changed fields between rule versions', () => {
    expect(taskRuleDifferences(
      { reward_points: 20, target: 1, period_type: 'DAILY', daily_limit: 1, per_user_limit: 1 },
      { reward_points: 10, target: 1, period_type: 'DAILY', daily_limit: 1, per_user_limit: 1 }
    )).toEqual(['reward_points'])
  })
})

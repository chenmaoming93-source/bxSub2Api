import { describe, expect, it } from 'vitest'
import { aggregateSceneDailyStats, aggregateSceneStats, buildDashboardQuery } from '../adminDashboardData'

describe('admin dashboard data coordinator', () => {
  it('aggregates scene rows across dates by group id and sorts by tokens', () => {
    const rows = aggregateSceneStats([
      { group_id: 1, group_name: '研发', requests: 2, total_tokens: 100, cost: 1, actual_cost: 0.8, account_cost: 0.4 },
      { group_id: 2, group_name: '运营', requests: 4, total_tokens: 300, cost: 2, actual_cost: 1.5, account_cost: 1 },
      { group_id: 1, group_name: '研发', requests: 3, total_tokens: 250, cost: 3, actual_cost: 2.2, account_cost: 1.4 }
    ])
    expect(rows).toHaveLength(2)
    expect(rows[0]).toMatchObject({ id: 1, label: '研发', requests: 5, totalTokens: 350 })
    expect(rows[0].percentage).toBeCloseTo(350 / 650 * 100)
    expect(rows[1]).toMatchObject({ id: 2, totalTokens: 300 })
  })

  it('uses daily scene totals as the main ranking source', () => {
    const rows = aggregateSceneDailyStats({
      timezone: 'UTC', start_date: '2026-01-01', end_date: '2026-01-02', complete: true, consistency: 'ok', projection_id: 1,
      days: [
        { date: '2026-01-01', scenes: [{ group_id: 7, group_name: '研发', scene_name: '研发', total_tokens: 120, accounts: [] }] },
        { date: '2026-01-02', scenes: [{ group_id: 7, group_name: '研发', scene_name: '研发', total_tokens: 80, accounts: [] }] }
      ]
    })
    expect(rows[0]).toMatchObject({ id: 7, label: '研发', totalTokens: 200 })
  })

  it('creates the shared range query contract', () => {
    expect(buildDashboardQuery({ startDate: '2026-01-01', endDate: '2026-01-31', granularity: 'day' })).toEqual({
      start_date: '2026-01-01', end_date: '2026-01-31', granularity: 'day'
    })
  })
})

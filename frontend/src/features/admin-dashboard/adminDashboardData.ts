import { computed, ref } from 'vue'
import { adminAPI } from '@/api/admin'
import type { DashboardSnapshotV2Response } from '@/api/admin/dashboard'
import type { AdminUsageStatsResponse, DepartmentUsageRow, SceneAccountDailyUsageResponse } from '@/api/admin/usage'
import type { GroupStat, UserUsageTrendPoint } from '@/types'

export interface AdminDashboardDateRange {
  startDate: string
  endDate: string
  granularity: 'day' | 'hour'
}

export interface SceneRankingRow {
  id: number
  label: string
  requests: number
  totalTokens: number
  cost: number
  actualCost: number
  accountCost: number
  percentage: number
}

export interface AdminDashboardData {
  snapshot: DashboardSnapshotV2Response | null
  usageStats: AdminUsageStatsResponse | null
  usersTrend: UserUsageTrendPoint[]
  departments: DepartmentUsageRow[]
  scenes: SceneRankingRow[]
}

function finiteNumber(value: number | undefined | null): number {
  return Number.isFinite(value) ? Number(value) : 0
}

/** Aggregates rows by group_id so daily scene rows become one range ranking. */
export function aggregateSceneStats(rows: GroupStat[] = []): SceneRankingRow[] {
  const grouped = new Map<number, SceneRankingRow>()
  for (const row of rows) {
    const current = grouped.get(row.group_id) ?? {
      id: row.group_id,
      label: row.group_name,
      requests: 0,
      totalTokens: 0,
      cost: 0,
      actualCost: 0,
      accountCost: 0,
      percentage: 0
    }
    current.label = row.group_name || current.label
    current.requests += finiteNumber(row.requests)
    current.totalTokens += finiteNumber(row.total_tokens)
    current.cost += finiteNumber(row.cost)
    current.actualCost += finiteNumber(row.actual_cost)
    current.accountCost += finiteNumber(row.account_cost)
    grouped.set(row.group_id, current)
  }

  const totalTokens = [...grouped.values()].reduce((sum, row) => sum + row.totalTokens, 0)
  return [...grouped.values()]
    .map((row) => ({ ...row, percentage: totalTokens > 0 ? (row.totalTokens / totalTokens) * 100 : 0 }))
    .sort((a, b) => b.totalTokens - a.totalTokens || a.label.localeCompare(b.label))
}

/** Uses the same daily projection as the detail dialog for the main scene ranking. */
export function aggregateSceneDailyStats(response: SceneAccountDailyUsageResponse | null): SceneRankingRow[] {
  const rows: GroupStat[] = []
  for (const day of response?.days ?? []) {
    for (const scene of day.scenes) {
      rows.push({
        group_id: scene.group_id,
        group_name: scene.group_name,
        requests: 0,
        total_tokens: scene.total_tokens,
        cost: 0,
        actual_cost: 0,
        account_cost: 0
      })
    }
  }
  return aggregateSceneStats(rows)
}

export function buildDashboardQuery(range: AdminDashboardDateRange) {
  return {
    start_date: range.startDate,
    end_date: range.endDate,
    granularity: range.granularity
  }
}

export function useAdminDashboardData(initialRange: AdminDashboardDateRange) {
  const range = ref<AdminDashboardDateRange>({ ...initialRange })
  const data = ref<AdminDashboardData>({ snapshot: null, usageStats: null, usersTrend: [], departments: [], scenes: [] })
  const loading = ref(false)
  const error = ref<unknown>(null)
  let requestSequence = 0

  const sceneRows = computed(() => data.value.scenes)

  async function load(nextRange: AdminDashboardDateRange = range.value) {
    const sequence = ++requestSequence
    range.value = { ...nextRange }
    loading.value = true
    error.value = null
    try {
      const params = buildDashboardQuery(nextRange)
      const results = await Promise.allSettled([
        adminAPI.dashboard.getSnapshotV2({
          ...params,
          include_stats: true,
          include_trend: true,
          include_model_stats: true,
          include_group_stats: true,
          include_users_trend: false
        }),
        adminAPI.usage.getStats({ start_date: nextRange.startDate, end_date: nextRange.endDate }),
        // Load a large candidate set once; the chart derives Top 12 by range tokens.
        adminAPI.dashboard.getUserUsageTrend({ ...params, limit: 10000 }),
        adminAPI.usage.queryDepartmentStats({ start_date: nextRange.startDate, end_date: nextRange.endDate }),
        // Use the detail projection for the main scene totals so both views share one source of truth.
        adminAPI.usage.querySceneAccountDaily({ start_date: nextRange.startDate, end_date: nextRange.endDate })
      ])
      if (sequence !== requestSequence) return
      const snapshot = results[0].status === 'fulfilled' ? results[0].value : null
      const usageStats = results[1].status === 'fulfilled' ? results[1].value : null
      const usersTrend = results[2].status === 'fulfilled' ? results[2].value.trend ?? [] : []
      const departments = results[3].status === 'fulfilled' ? results[3].value.rows ?? [] : []
      const sceneDaily = results[4].status === 'fulfilled' ? results[4].value : null
      const failures = results.filter((result) => result.status === 'rejected')
      const scenes = sceneDaily ? aggregateSceneDailyStats(sceneDaily) : aggregateSceneStats(snapshot?.groups ?? [])
      data.value = { snapshot, usageStats, usersTrend, departments, scenes }
      if (failures.length) error.value = failures.map((result) => result.status === 'rejected' ? result.reason : null)

    } catch (caught) {
      if (sequence !== requestSequence) return
      error.value = caught
      data.value = { snapshot: null, usageStats: null, usersTrend: [], departments: [], scenes: [] }
    } finally {
      if (sequence === requestSequence) loading.value = false
    }
  }

  return { range, data, sceneRows, loading, error, load }
}

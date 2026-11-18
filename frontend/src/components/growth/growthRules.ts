import type { ExchangeOrder, GrowthProduct, GrowthTask } from '@/api/growth'

export type TaskAction = 'edit' | 'publish' | 'offline' | 'online' | 'quota' | 'history'

export function taskActions(task: Pick<GrowthTask, 'status' | 'pending_rule_version'>, canManage: boolean): TaskAction[] {
  const actions: TaskAction[] = ['history']
  if (canManage) {
    actions.unshift('edit')
    if (task.status === 'DRAFT' || (task.status === 'ACTIVE' && Boolean(task.pending_rule_version))) {
      actions.push('publish')
    }
    if (task.status === 'ACTIVE') actions.push('offline')
    if (task.status === 'OFFLINE') actions.push('online')
    if (task.status === 'DRAFT' || task.status === 'ACTIVE') actions.push('quota')
  }
  return actions
}

export function canRevokeTaskQuota(task: Pick<GrowthTask | { status: string }, 'status'>): boolean {
  return task.status === 'OFFLINE'
}

export function quotaTone(remaining: number, usageRate: number): 'healthy' | 'warning' | 'tight' | 'exhausted' {
  if (remaining <= 0) return 'exhausted'
  if (usageRate > 0.85) return 'tight'
  if (usageRate >= 0.6) return 'warning'
  return 'healthy'
}

export function productPublishError(product: Pick<GrowthProduct, 'product_type' | 'fulfillment_mode' | 'token_amount'>): string | null {
  if (product.product_type === 'TOKEN_QUOTA' && (product.token_amount <= 0 || product.fulfillment_mode !== 'AUTOMATIC')) {
    return 'INVALID_TOKEN_PRODUCT'
  }
  return null
}

export function reasonIsValid(reason: string): boolean {
  return reason.trim().length >= 5
}

export function positiveInteger(value: string | number): number | null {
  const parsed = typeof value === 'number' ? value : Number(value)
  return Number.isSafeInteger(parsed) && parsed > 0 ? parsed : null
}

export function productTitle(order: Pick<ExchangeOrder, 'product_snapshot'>): string {
  const title = order.product_snapshot?.title
  return typeof title === 'string' ? title : '-'
}

export function isOverdueFailed(order: Pick<ExchangeOrder, 'status' | 'created_at'>, now = Date.now()): boolean {
  if (order.status !== 'FAILED') return false
  const created = new Date(order.created_at).getTime()
  return Number.isFinite(created) && now - created > 24 * 60 * 60 * 1000
}

export function orderMatches(
  order: ExchangeOrder,
  filters: { search?: string; status?: string; product?: string; from?: string; to?: string }
): boolean {
  const search = filters.search?.trim().toLowerCase() ?? ''
  if (search && !order.order_no.toLowerCase().includes(search)) return false
  if (filters.status && order.status !== filters.status) return false
  if (filters.product && productTitle(order) !== filters.product) return false
  const created = new Date(order.created_at).getTime()
  if (filters.from) {
    const from = new Date(`${filters.from}T00:00:00`).getTime()
    if (Number.isFinite(from) && created < from) return false
  }
  if (filters.to) {
    const to = new Date(`${filters.to}T23:59:59.999`).getTime()
    if (Number.isFinite(to) && created > to) return false
  }
  return true
}

export function appendUniqueByEntryId<T extends { entry_id: number }>(current: T[], next: T[]): T[] {
  const seen = new Set(current.map(item => item.entry_id))
  return [...current, ...next.filter(item => !seen.has(item.entry_id))]
}

export function productEligibilityText(
  product: { stock: number; points_cost?: number },
  reason?: string,
  balance = 0
): { key: string; params?: Record<string, number> } {
  if (reason === 'PRODUCT_SOLD_OUT' || product.stock === 0) return { key: 'soldOut' }
  if (reason === 'PURCHASE_LIMIT_REACHED') return { key: 'limitReached' }
  if (reason === 'INSUFFICIENT_POINTS') {
    return { key: 'insufficient', params: { points: Math.max(0, Number(product.points_cost ?? 0) - balance) } }
  }
  return { key: reason ? 'unavailable' : 'exchangeable' }
}

export function taskRuleDifferences(
  current: Record<string, unknown>,
  previous?: Record<string, unknown>
): string[] {
  if (!previous) return []
  const fields = ['reward_points', 'target', 'period_type', 'daily_limit', 'per_user_limit']
  return fields.filter(field => current[field] !== previous[field])
}

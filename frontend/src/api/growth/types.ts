export interface BudgetAccount {
  total_amount: number
  allocated_amount: number
  distributed_amount: number
  unallocated_amount: number
  distributed_rate: number
  version: number
  updated_at?: string
}

export interface BudgetEntry {
  id: number
  entry_type: string
  points: number
  budget_after: number
  allocated_after: number
  distributed_after: number
  task_id?: number
  task_title?: string
  operator_id: number
  reason: string
  created_at: string
}

export interface TaskAllocation {
  task_id: number
  task_code: string
  title: string
  status: string
  allocated_points: number
  distributed_points: number
  remaining_points: number
  usage_rate: number
  updated_at: string
}

export interface GrowthTask {
  task_id: number
  task_code: string
  title: string
  description: string
  category: string
  task_type: string
  icon: string
  sort_order: number
  status: string
  allocated_points: number
  distributed_points: number
  remaining_points: number
  usage_rate: number
  rule_version?: number
  pending_rule_version?: number
  reward_points: number
  period_type: string
  daily_limit: number
  per_user_limit: number
  updated_at: string
}

export interface TaskRuleVersion {
  id: number
  version: number
  trigger_event: string
  conditions?: Record<string, unknown>
  target: number
  reward_points: number
  period_type: string
  daily_limit: number
  per_user_limit: number
  created_by: number
  effective_from: string
  effective_to?: string
  published_at?: string
  created_at: string
}

export interface GrowthProduct {
  product_id: number
  product_code: string
  title: string
  product_type: string
  icon: string
  fulfillment_mode: string
  sort_order: number
  status: string
  current_version: number
  pending_version?: number
  points_cost: number
  benefit_summary: string
  benefit_detail?: Record<string, unknown>
  token_amount: number
  per_user_limit: number
  stock_mode: string
  stock: number
  reserved: number
  sold: number
  updated_at: string
}

export interface ProductVersion {
  id: number
  version: number
  points_cost: number
  benefit_summary: string
  benefit_detail?: Record<string, unknown>
  token_amount: number
  per_user_limit: number
  created_by: number
  published_at?: string
  effective_from?: string
  effective_to?: string
  created_at: string
}

export interface ExchangeOrder {
  order_id: number
  order_no: string
  status: string
  product_snapshot: Record<string, unknown>
  points_cost: number
  transfer_id?: number
  fulfillment_status: string
  failure_reason: string
  refund_transfer_id?: number
  created_at: string
  fulfilled_at?: string
}

export interface UserPoints {
  user_id: number
  email: string
  username: string
  available_points: number
  total_earned: number
  total_spent: number
  latest_change_at?: string
}

export interface LedgerItem {
  entry_id: number
  transfer_id: number
  type: string
  direction: 'IN' | 'OUT'
  points: number
  balance_after?: number
  biz_type: string
  title: string
  created_at: string
}

export interface LedgerPage {
  items: LedgerItem[]
  next_cursor: string
  has_more: boolean
}

export interface UserOverview {
  available_points: number
  total_earned: number
  total_spent: number
  month_earned: number
  checkin: {
    claimed_today: boolean
    streak_days: number
    next_eligible_at?: string
  }
  server_time: string
}

export interface UserTask {
  task_id: number
  title: string
  description: string
  points: number
  period: string
  daily_limit: number
  per_user_limit: number
  completed_today: number
  status: string
  action: string
}

export interface UserProduct {
  product_id: number
  name: string
  description: string
  points_cost: number
  stock: number
  per_user_limit: number
  can_exchange: boolean
  reason?: string
  fulfillment_note?: string
  benefit_detail?: Record<string, unknown>
}

export interface UserExchange {
  order_id: number
  order_no: string
  product_title: string
  product_version: number
  points_cost: number
  status: string
  failure_reason?: string
  refunded: boolean
  created_at: string
}

export interface ExchangeCreateResult {
  order_id: number
  order_no: string
  status: string
  product_version: number
  points_cost: number
  transfer_id?: number
  current_balance: number
  fulfillment_status: string
  server_time: string
}

export interface UserOrderStatus {
  order_id: number
  order_no: string
  status: string
  fulfillment_status: string
  failure_reason?: string
  refunded: boolean
  server_time?: string
}

export interface Paginated<T> {
  items: T[]
  total: number
  page: number
  page_size: number
  pages: number
}

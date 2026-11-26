import { apiClient } from '@/api/client'
import type {
  BudgetAccount, BudgetEntry, ExchangeOrder, GrowthProduct, GrowthTask, Paginated,
  TaskAllocation, UserPoints, LedgerPage, UserExchange, TaskRuleVersion, ProductVersion
} from './types'

const clean = (params?: Record<string, unknown>) =>
  Object.fromEntries(Object.entries(params ?? {}).filter(([, value]) => value !== '' && value != null))

export const growthAdminAPI = {
  budget: () => apiClient.get<BudgetAccount>('/admin/growth/budget'),
  overview: () => apiClient.get<BudgetAccount>('/admin/growth/stats/overview'),
  budgetEntries: (params?: Record<string, unknown>) => apiClient.get<Paginated<BudgetEntry>>('/admin/growth/budget/entries', { params: clean(params) }),
  createBudgetEntry: (data: Record<string, unknown>) => apiClient.post<BudgetAccount>('/admin/growth/budget/entries', data),
  allocations: (params?: Record<string, unknown>) => apiClient.get<Paginated<TaskAllocation>>('/admin/growth/budget/allocations', { params: clean(params) }),
  allocate: (data: Record<string, unknown>) => apiClient.post<BudgetAccount>('/admin/growth/budget/allocations', data),
  revoke: (taskId: number, data: Record<string, unknown>) => apiClient.post<BudgetAccount>(`/admin/growth/tasks/${taskId}/budget/revoke`, data),
  tasks: (params?: Record<string, unknown>) => apiClient.get<Paginated<GrowthTask>>('/admin/growth/tasks', { params: clean(params) }),
  task: (id: number) => apiClient.get<GrowthTask>(`/admin/growth/tasks/${id}`),
  createTask: (data: Record<string, unknown>) => apiClient.post<GrowthTask>('/admin/growth/tasks', data),
  updateTask: (id: number, data: Record<string, unknown>) => apiClient.put<GrowthTask>(`/admin/growth/tasks/${id}`, data),
  publishTask: (id: number, data: Record<string, unknown> = {}) => apiClient.post<GrowthTask>(`/admin/growth/tasks/${id}/publish`, data),
  offlineTask: (id: number, data: Record<string, unknown>) => apiClient.post<GrowthTask>(`/admin/growth/tasks/${id}/offline`, data),
  onlineTask: (id: number, data: Record<string, unknown>) => apiClient.post<GrowthTask>(`/admin/growth/tasks/${id}/online`, data),
  ruleVersions: (id: number) => apiClient.get<{ items: TaskRuleVersion[] }>(`/admin/growth/tasks/${id}/rule-versions`),
  products: (params?: Record<string, unknown>) => apiClient.get<Paginated<GrowthProduct>>('/admin/growth/products', { params: clean(params) }),
  createProduct: (data: Record<string, unknown>) => apiClient.post<GrowthProduct>('/admin/growth/products', data),
  updateProduct: (id: number, data: Record<string, unknown>) => apiClient.put<GrowthProduct>(`/admin/growth/products/${id}`, data),
  publishProduct: (id: number) => apiClient.post<GrowthProduct>(`/admin/growth/products/${id}/publish`, {}),
  productVersions: (id: number) => apiClient.get<{ items: ProductVersion[] }>(`/admin/growth/products/${id}/versions`),
  adjustStock: (id: number, data: Record<string, unknown>) => apiClient.post<GrowthProduct>(`/admin/growth/products/${id}/stock`, data),
  offlineProduct: (id: number, data: Record<string, unknown>) => apiClient.post<GrowthProduct>(`/admin/growth/products/${id}/offline`, data),
  exchanges: (params?: Record<string, unknown>) => apiClient.get<Paginated<ExchangeOrder>>('/admin/growth/exchanges', { params: clean(params) }),
  order: (id: number) => apiClient.get<ExchangeOrder>(`/admin/growth/exchanges/${id}`),
  retryOrder: (id: number) => apiClient.post<ExchangeOrder>(`/admin/growth/exchanges/${id}/retry`, {}),
  completeOrder: (id: number, data: Record<string, unknown>) => apiClient.post<ExchangeOrder>(`/admin/growth/exchanges/${id}/manual-complete`, data),
  refundOrder: (id: number, data: Record<string, unknown>) => apiClient.post<ExchangeOrder>(`/admin/growth/exchanges/${id}/refund`, data),
  users: (params?: Record<string, unknown>) => apiClient.get<Paginated<UserPoints>>('/admin/growth/users', { params: clean(params) }),
  userLedger: (id: number, params?: Record<string, unknown>) => apiClient.get<LedgerPage>(`/admin/growth/users/${id}/ledger`, { params: clean(params) }),
  userExchanges: (id: number, params?: Record<string, unknown>) => apiClient.get<Paginated<UserExchange>>(`/admin/growth/users/${id}/exchanges`, { params: clean(params) }),
  adjust: (id: number, data: Record<string, unknown>) => apiClient.post<Record<string, number>>(`/admin/growth/users/${id}/adjustments`, data),
  reconciliation: () => apiClient.get<Record<string, unknown>>('/admin/growth/reconciliation'),
  auditLogs: (params?: Record<string, unknown>) => apiClient.get<Paginated<Record<string, unknown>>>('/admin/growth/audit-logs', { params: clean(params) }),
}

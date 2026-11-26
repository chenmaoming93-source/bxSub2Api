import { apiClient } from '@/api/client'
import type { ExchangeCreateResult, LedgerPage, Paginated, UserExchange, UserOrderStatus, UserOverview, UserProduct, UserTask } from './types'

export const growthUserAPI = {
  overview: () => apiClient.get<UserOverview>('/user/growth/overview'),
  tasks: () => apiClient.get<{ items: UserTask[] }>('/user/growth/tasks'),
  products: () => apiClient.get<{ items: UserProduct[] }>('/user/growth/products'),
  product: (id: number) => apiClient.get<UserProduct>(`/user/growth/products/${id}`),
  exchanges: (params?: Record<string, unknown>) => apiClient.get<Paginated<UserExchange>>('/user/growth/exchanges', { params }),
  exchange: (productId: number, key: string) => apiClient.post<ExchangeCreateResult>('/user/growth/exchanges', { product_id: productId }, { headers: { 'Idempotency-Key': key } }),
  order: (id: number) => apiClient.get<UserOrderStatus>(`/user/growth/exchanges/${id}`),
  ledger: (params?: Record<string, unknown>) => apiClient.get<LedgerPage>('/user/growth/ledger', { params }),
}

export default growthUserAPI

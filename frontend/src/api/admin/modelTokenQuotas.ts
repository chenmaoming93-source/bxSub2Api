import { apiClient } from '../client'

export interface ModelTokenQuotaItem {
  model: string
  usage_date: string
  used_tokens: number
  daily_limit_tokens: number | null
}

export interface UserModelTokenQuotaItem extends ModelTokenQuotaItem {
  user_id: number
}

export interface ModelTokenQuotaUpdateItem {
  model: string
  daily_limit_tokens: number | null
}

export interface ModelTokenQuotasResponse {
  quotas: ModelTokenQuotaItem[]
}

export interface UserModelTokenQuotasResponse {
  quotas: UserModelTokenQuotaItem[]
}

export async function getGlobalModelTokenQuotas(): Promise<ModelTokenQuotasResponse> {
  const { data } = await apiClient.get<ModelTokenQuotasResponse>('/admin/model-token-quotas')
  return data
}

export async function updateGlobalModelTokenQuota(
  input: ModelTokenQuotaUpdateItem
): Promise<ModelTokenQuotaItem> {
  const { data } = await apiClient.put<{ quota: ModelTokenQuotaItem }>(
    '/admin/model-token-quotas',
    input
  )
  return data.quota
}

export async function getUserModelTokenQuotas(
  userId: number
): Promise<UserModelTokenQuotasResponse> {
  const { data } = await apiClient.get<UserModelTokenQuotasResponse>(
    `/admin/users/${userId}/model-token-quotas`
  )
  return data
}

export async function updateUserModelTokenQuotas(
  userId: number,
  quotas: ModelTokenQuotaUpdateItem[]
): Promise<UserModelTokenQuotasResponse> {
  const { data } = await apiClient.put<UserModelTokenQuotasResponse>(
    `/admin/users/${userId}/model-token-quotas`,
    { quotas }
  )
  return data
}

export const modelTokenQuotasAPI = {
  getGlobal: getGlobalModelTokenQuotas,
  updateGlobal: updateGlobalModelTokenQuota,
  getUser: getUserModelTokenQuotas,
  updateUser: updateUserModelTokenQuotas
}

export default modelTokenQuotasAPI

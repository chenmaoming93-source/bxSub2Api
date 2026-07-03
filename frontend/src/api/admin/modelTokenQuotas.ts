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

// ─── Default model token quotas (new-user defaults) ───

export interface DefaultModelTokenQuotaItem {
  model: string
  daily_limit_tokens: number | null
}

export interface DefaultModelTokenQuotasResponse {
  quotas: DefaultModelTokenQuotaItem[]
}

export async function getDefaultModelTokenQuotas(): Promise<DefaultModelTokenQuotasResponse> {
  const { data } = await apiClient.get<DefaultModelTokenQuotasResponse>(
    '/admin/settings/default-model-token-quotas'
  )
  return data
}

export async function updateDefaultModelTokenQuotas(
  quotas: DefaultModelTokenQuotaItem[]
): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>(
    '/admin/settings/default-model-token-quotas',
    { quotas }
  )
  return data
}

// ─── Batch model token quota operations ───

export interface BatchModelTokenQuotaOperation {
  action: 'create' | 'update' | 'delete'
  model: string
  daily_limit_tokens?: number | null
}

export interface BatchModelTokenQuotaResult {
  affected_users: number
  operations: number
  errors?: string[]
}

export async function batchApplyModelTokenQuotas(
  operations: BatchModelTokenQuotaOperation[]
): Promise<BatchModelTokenQuotaResult> {
  const { data } = await apiClient.post<BatchModelTokenQuotaResult>(
    '/admin/users/model-token-quotas/batch',
    { operations }
  )
  return data
}

export const modelTokenQuotasAPI = {
  getGlobal: getGlobalModelTokenQuotas,
  updateGlobal: updateGlobalModelTokenQuota,
  getUser: getUserModelTokenQuotas,
  updateUser: updateUserModelTokenQuotas,
  getDefault: getDefaultModelTokenQuotas,
  updateDefault: updateDefaultModelTokenQuotas,
  batchApply: batchApplyModelTokenQuotas
}

export default modelTokenQuotasAPI

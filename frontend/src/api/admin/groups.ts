/**
 * Admin Groups API endpoints
 * Handles API key group management for administrators
 */

import { apiClient } from '../client'
import type {
  AdminGroup,
  GroupPlatform,
  CreateGroupRequest,
  UpdateGroupRequest,
  PaginatedResponse
} from '@/types'

export type { ModelRoutingCandidate, ModelRoutingConfig, ModelRoutingRuleRow } from '@/types'

export async function updateModelRouteConcurrency(
  groupId: number,
  input: { updates: Array<{ route_alias: string; account_id: number; max_concurrency: number | null }> }
): Promise<void> {
  await apiClient.put(`/admin/groups/${groupId}/model-route-references/concurrency`, input)
}

export async function listModelRouteReferences(groupId: number) {
  const { data } = await apiClient.get<Array<{ route_alias: string; account_id: number; max_concurrency: number | null; account_concurrency: number; allocated_concurrency: number }>>(`/admin/groups/${groupId}/model-route-references`)
  return data
}

export async function listModelRouteConcurrency(groupId: number) {
  const { data } = await apiClient.get<Array<{ route_alias: string; account_id: number; max_concurrency: number | null; account_concurrency: number; allocated_concurrency: number; current_concurrency: number; effective_max_concurrency: number | null }>>(`/admin/groups/${groupId}/model-route-concurrency`)
  return data
}

export interface ModelRouteConcurrencySchedule {
  id?: number
  start: string
  end: string
  max_concurrency: number | null
}

export async function listModelRouteConcurrencySchedules(
  groupId: number,
  routeAlias: string,
  accountId: number
): Promise<ModelRouteConcurrencySchedule[]> {
  const { data } = await apiClient.get<ModelRouteConcurrencySchedule[]>(
    `/admin/groups/${groupId}/model-route-references/concurrency-schedules`,
    { params: { route_alias: routeAlias, account_id: accountId } }
  )
  return data || []
}

export async function replaceModelRouteConcurrencySchedules(
  groupId: number,
  input: { route_alias: string; account_id: number; schedules: ModelRouteConcurrencySchedule[] }
): Promise<void> {
  await apiClient.put(`/admin/groups/${groupId}/model-route-references/concurrency-schedules`, input)
}

export interface ModelRouteConcurrencyScheduleRefreshResponse {
  task_id: string
  message: string
}

export async function refreshModelRouteConcurrencySchedules(
  groupId?: number
): Promise<ModelRouteConcurrencyScheduleRefreshResponse> {
  const { data } = await apiClient.post<ModelRouteConcurrencyScheduleRefreshResponse>(
    groupId
      ? `/admin/groups/${groupId}/model-route-references/concurrency-schedules/refresh`
      : '/admin/groups/model-route-references/concurrency-schedules/refresh'
  )
  return data
}

export async function rebuildModelRouteReferences(): Promise<unknown> {
  const { data } = await apiClient.post('/admin/groups/model-route-references/rebuild')
  return data
}

/**
 * List all groups with pagination
 * @param page - Page number (default: 1)
 * @param pageSize - Items per page (default: 20)
 * @param filters - Optional filters (platform, status, is_exclusive, search)
 * @returns Paginated list of groups
 */
export async function list(
  page: number = 1,
  pageSize: number = 20,
  filters?: {
    platform?: GroupPlatform
    status?: 'active' | 'inactive'
    is_exclusive?: boolean
    search?: string
    sort_by?: string
    sort_order?: 'asc' | 'desc'
  },
  options?: {
    signal?: AbortSignal
  }
): Promise<PaginatedResponse<AdminGroup>> {
  const { data } = await apiClient.get<PaginatedResponse<AdminGroup>>('/admin/groups', {
    params: {
      page,
      page_size: pageSize,
      ...filters
    },
    signal: options?.signal
  })
  return data
}

/**
 * Get all active groups (without pagination)
 * @param platform - Optional platform filter
 * @returns List of all active groups
 */
export async function getAll(platform?: GroupPlatform): Promise<AdminGroup[]> {
  const { data } = await apiClient.get<AdminGroup[]>('/admin/groups/all', {
    params: platform ? { platform } : undefined
  })
  return data
}

/**
 * Get ALL groups including disabled ones — used by the API Key group filter so
 * that admins can filter users whose keys are still bound to a now-disabled group.
 */
export async function getAllIncludingInactive(): Promise<AdminGroup[]> {
  const { data } = await apiClient.get<AdminGroup[]>('/admin/groups/all', {
    params: { include_inactive: true }
  })
  return data
}

/**
 * Get active groups by platform
 * @param platform - Platform to filter by
 * @returns List of groups for the specified platform
 */
export async function getByPlatform(platform: GroupPlatform): Promise<AdminGroup[]> {
  return getAll(platform)
}

/**
 * Get group by ID
 * @param id - Group ID
 * @returns Group details
 */
export async function getById(id: number): Promise<AdminGroup> {
  const { data } = await apiClient.get<AdminGroup>(`/admin/groups/${id}`)
  return data
}

export async function updateSecurityCheck(
  id: number,
  config: import('@/types').SecurityCheckConfig
): Promise<AdminGroup> {
  const { data } = await apiClient.put<AdminGroup>(`/admin/groups/${id}/security-check`, config)
  return data
}

export interface SecurityCheckLogSummary {
  id: number
  event_id: string
  request_id?: string
  group_id?: number
  group_name?: string
  model?: string
  protocol?: string
  check_status: string
  decision: string
  is_unsafe: boolean
  triggered_rules: Array<{ dimension: string; threshold: number; action: string; risk_prob: number }>
  latency_ms?: number
  created_at: string
  request_body_original_bytes: number
  request_body_stored_bytes: number
  request_body_truncated: boolean
}

export interface SecurityCheckLogPage {
  items: SecurityCheckLogSummary[]
  total: number
  page: number
  page_size: number
}

export interface SecurityCheckLogRetentionConfig {
  retention_days: number
  cleanup_time: string
  timezone: string
  next_cleanup_at: string
}

export async function listSecurityCheckLogs(params?: Record<string, string | number>): Promise<SecurityCheckLogPage> {
  const { data } = await apiClient.get<SecurityCheckLogPage>('/admin/groups/security-check/logs', { params })
  return data
}

export async function getSecurityCheckLog(id: number) {
  const { data } = await apiClient.get(`/admin/groups/security-check/logs/${id}`)
  return data as SecurityCheckLogSummary & { config_version: number; rules_snapshot: unknown[]; request_body?: string; singguard_response?: string; exception_type?: string; exception_message?: string }
}

export async function getSecurityCheckLogRetention(): Promise<SecurityCheckLogRetentionConfig> {
  const { data } = await apiClient.get<SecurityCheckLogRetentionConfig>('/admin/groups/security-check/retention')
  return data
}

export async function updateSecurityCheckLogRetention(config: Pick<SecurityCheckLogRetentionConfig, 'retention_days' | 'cleanup_time'>): Promise<SecurityCheckLogRetentionConfig> {
  const { data } = await apiClient.put<SecurityCheckLogRetentionConfig>('/admin/groups/security-check/retention', config)
  return data
}

export async function getSecurityCheckCollectionStatus() {
  const { data } = await apiClient.get<{ circuit_open: boolean; failure_count: number }>('/admin/groups/security-check/collection-status')
  return data
}

export async function reopenSecurityCheckCollection() {
  const { data } = await apiClient.post<{ reopened: boolean }>('/admin/groups/security-check/collection-reopen')
  return data
}

/**
 * Get candidate models for custom /v1/models list.
 * id=0 returns platform default models for create flow.
 */
export async function getModelsListCandidates(
  id: number,
  platform?: GroupPlatform
): Promise<string[]> {
  const { data } = await apiClient.get<{ models: string[] }>(
    `/admin/groups/${id}/models-list-candidates`,
    {
      params: platform ? { platform } : undefined
    }
  )
  return data.models || []
}

/**
 * Create new group
 * @param groupData - Group data
 * @returns Created group
 */
export async function create(groupData: CreateGroupRequest): Promise<AdminGroup> {
  const { data } = await apiClient.post<AdminGroup>('/admin/groups', groupData)
  return data
}

/**
 * Update group
 * @param id - Group ID
 * @param updates - Fields to update
 * @returns Updated group
 */
export async function update(id: number, updates: UpdateGroupRequest): Promise<AdminGroup> {
  const { data } = await apiClient.put<AdminGroup>(`/admin/groups/${id}`, updates)
  return data
}

/**
 * Delete group
 * @param id - Group ID
 * @returns Success confirmation
 */
export async function deleteGroup(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/groups/${id}`)
  return data
}

/**
 * Toggle group status
 * @param id - Group ID
 * @param status - New status
 * @returns Updated group
 */
export async function toggleStatus(id: number, status: 'active' | 'inactive'): Promise<AdminGroup> {
  return update(id, { status })
}

/**
 * Get group statistics
 * @param id - Group ID
 * @returns Group usage statistics
 */
export async function getStats(id: number): Promise<{
  total_api_keys: number
  active_api_keys: number
  total_requests: number
  total_cost: number
}> {
  const { data } = await apiClient.get<{
    total_api_keys: number
    active_api_keys: number
    total_requests: number
    total_cost: number
  }>(`/admin/groups/${id}/stats`)
  return data
}

/**
 * Get API keys in a group
 * @param id - Group ID
 * @param page - Page number
 * @param pageSize - Items per page
 * @returns Paginated list of API keys in the group
 */
export async function getGroupApiKeys(
  id: number,
  page: number = 1,
  pageSize: number = 20
): Promise<PaginatedResponse<any>> {
  const { data } = await apiClient.get<PaginatedResponse<any>>(`/admin/groups/${id}/api-keys`, {
    params: { page, page_size: pageSize }
  })
  return data
}

/**
 * Rate multiplier entry for a user in a group
 */
export interface GroupRateMultiplierEntry {
  user_id: number
  user_name: string
  user_email: string
  user_notes: string
  user_status: string
  rate_multiplier?: number | null
  rpm_override?: number | null
}

/**
 * Get rate multipliers for users in a group
 * @param id - Group ID
 * @returns List of user rate multiplier entries
 */
export async function getGroupRateMultipliers(id: number): Promise<GroupRateMultiplierEntry[]> {
  const { data } = await apiClient.get<GroupRateMultiplierEntry[]>(
    `/admin/groups/${id}/rate-multipliers`
  )
  return data
}

/**
 * Update group sort orders
 * @param updates - Array of { id, sort_order } objects
 * @returns Success confirmation
 */
export async function updateSortOrder(
  updates: Array<{ id: number; sort_order: number }>
): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>('/admin/groups/sort-order', {
    updates
  })
  return data
}

/**
 * Clear all rate multipliers for a group
 * @param id - Group ID
 * @returns Success confirmation
 */
export async function clearGroupRateMultipliers(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/groups/${id}/rate-multipliers`)
  return data
}

/**
 * Batch set rate multipliers for users in a group
 * Only touches rate_multiplier column; preserves rpm_override on existing rows.
 */
export async function batchSetGroupRateMultipliers(
  id: number,
  entries: Array<{ user_id: number; rate_multiplier: number }>
): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>(
    `/admin/groups/${id}/rate-multipliers`,
    { entries }
  )
  return data
}

/**
 * RPM override entry for a user in a group
 */
export interface GroupRPMOverrideEntry {
  user_id: number
  user_name: string
  user_email: string
  user_notes: string
  user_status: string
  rpm_override: number
}

/**
 * Get RPM overrides for users in a group (subset of rate-multipliers endpoint).
 */
export async function getGroupRPMOverrides(id: number): Promise<GroupRPMOverrideEntry[]> {
  const { data } = await apiClient.get<GroupRateMultiplierEntry[]>(
    `/admin/groups/${id}/rate-multipliers`
  )
  return data
    .filter(e => e.rpm_override != null)
    .map(e => ({
      user_id: e.user_id,
      user_name: e.user_name,
      user_email: e.user_email,
      user_notes: e.user_notes,
      user_status: e.user_status,
      rpm_override: e.rpm_override as number
    }))
}

/**
 * Batch set RPM overrides for users in a group.
 * Only touches rpm_override column; preserves rate_multiplier on existing rows.
 */
export async function batchSetGroupRPMOverrides(
  id: number,
  entries: Array<{ user_id: number; rpm_override: number }>
): Promise<{ message: string }> {
  const { data } = await apiClient.put<{ message: string }>(
    `/admin/groups/${id}/rpm-overrides`,
    { entries }
  )
  return data
}

/**
 * Clear all RPM overrides for a group (preserves rate_multiplier).
 */
export async function clearGroupRPMOverrides(id: number): Promise<{ message: string }> {
  const { data } = await apiClient.delete<{ message: string }>(`/admin/groups/${id}/rpm-overrides`)
  return data
}

/**
 * Get usage summary (today + cumulative cost) for all groups
 * @param timezone - IANA timezone string (e.g. "Asia/Shanghai")
 * @returns Array of group usage summaries
 */
export async function getUsageSummary(
  timezone?: string
): Promise<{ group_id: number; today_cost: number; total_cost: number }[]> {
  const { data } = await apiClient.get<
    { group_id: number; today_cost: number; total_cost: number }[]
  >('/admin/groups/usage-summary', {
    params: timezone ? { timezone } : undefined
  })
  return data
}

/**
 * Get capacity summary (concurrency/sessions/RPM) for all active groups
 */
export async function getCapacitySummary(): Promise<
  { group_id: number; concurrency_used: number; concurrency_max: number; sessions_used: number; sessions_max: number; rpm_used: number; rpm_max: number }[]
> {
  const { data } = await apiClient.get<
    { group_id: number; concurrency_used: number; concurrency_max: number; sessions_used: number; sessions_max: number; rpm_used: number; rpm_max: number }[]
  >('/admin/groups/capacity-summary')
  return data
}

export const groupsAPI = {
  list,
  getAll,
  getByPlatform,
  getAllIncludingInactive,
  getById,
  updateSecurityCheck,
  listSecurityCheckLogs,
  getSecurityCheckLog,
  getSecurityCheckLogRetention,
  updateSecurityCheckLogRetention,
  getSecurityCheckCollectionStatus,
  reopenSecurityCheckCollection,
  getModelsListCandidates,
  create,
  update,
  updateModelRouteConcurrency,
  listModelRouteReferences,
  listModelRouteConcurrency,
  listModelRouteConcurrencySchedules,
  replaceModelRouteConcurrencySchedules,
  refreshModelRouteConcurrencySchedules,
  rebuildModelRouteReferences,
  delete: deleteGroup,
  toggleStatus,
  getStats,
  getGroupApiKeys,
  getGroupRateMultipliers,
  clearGroupRateMultipliers,
  batchSetGroupRateMultipliers,
  getGroupRPMOverrides,
  clearGroupRPMOverrides,
  batchSetGroupRPMOverrides,
  updateSortOrder,
  getUsageSummary,
  getCapacitySummary
}

export default groupsAPI

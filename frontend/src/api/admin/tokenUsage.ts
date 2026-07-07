import { apiClient } from '../client'

// ── Report item types ──

export interface ModelTokenUsageItem {
  usage_date: string
  model: string
  used_tokens: number
  daily_limit_tokens: number | null
  usage_rate: number | null
  status: 'normal' | 'warning' | 'exceeded' | 'unlimited'
}

export interface RouteTokenUsageItem {
  usage_date: string
  group_id: number
  group_name: string
  route_alias: string
  upstream_model: string
  priority: number | null
  used_tokens: number
  daily_limit_tokens: number | null
  usage_rate: number | null
  status: 'normal' | 'warning' | 'exceeded' | 'unlimited'
}

export interface UserModelTokenUsageItem {
  usage_date: string
  user_id: number
  email: string
  username: string
  user_deleted: boolean
  model: string
  used_tokens: number
  daily_limit_tokens: number | null
  usage_rate: number | null
  status: 'normal' | 'warning' | 'exceeded' | 'unlimited'
}

// ── Pagination & summary ──

export interface TokenUsageSummary {
  used_tokens: number
}

export interface TokenUsagePagination {
  page: number
  page_size: number
  total: number
}

export interface TokenUsageReport<T> {
  items: T[]
  summary: TokenUsageSummary
  pagination: TokenUsagePagination
}

// ── Query parameters ──

export interface TokenUsageQueryParams {
  start_date?: string
  end_date?: string
  page?: number
  page_size?: number
  sort_by?: TokenUsageSortField
  sort_order?: 'asc' | 'desc'
}

export type TokenUsageSortField =
  | 'usage_date' | 'model' | 'user' | 'user_deleted' | 'group' | 'route_alias'
  | 'upstream_model' | 'priority' | 'used_tokens' | 'daily_limit_tokens'
  | 'usage_rate' | 'status'

export interface ModelReportParams extends TokenUsageQueryParams {
  model?: string
}

export interface RouteReportParams extends TokenUsageQueryParams {
  group_id?: number
  route_alias?: string
  upstream_model?: string
}

export interface UserReportParams extends TokenUsageQueryParams {
  user_id?: number
  model?: string
  include_deleted?: boolean
}

// ── Options types ──

export interface TokenUsageOption {
  id: number
  label: string
  model?: string
  group_id?: number
  route_alias?: string
  user_id?: number
}

export interface TokenUsageOptionsResponse {
  items: TokenUsageOption[]
}

export interface TokenUsageDefaultTargetResponse {
  target: TokenUsageOption | null
}

// ── Report API ──

export async function getModelTokenUsageReport(
  params: ModelReportParams
): Promise<TokenUsageReport<ModelTokenUsageItem>> {
  const { data } = await apiClient.get<TokenUsageReport<ModelTokenUsageItem>>(
    '/admin/token-usage/models',
    { params }
  )
  return data
}

export async function getRouteTokenUsageReport(
  params: RouteReportParams
): Promise<TokenUsageReport<RouteTokenUsageItem>> {
  const { data } = await apiClient.get<TokenUsageReport<RouteTokenUsageItem>>(
    '/admin/token-usage/routes',
    { params }
  )
  return data
}

export async function getUserModelTokenUsageReport(
  params: UserReportParams
): Promise<TokenUsageReport<UserModelTokenUsageItem>> {
  const { data } = await apiClient.get<TokenUsageReport<UserModelTokenUsageItem>>(
    '/admin/token-usage/users',
    { params }
  )
  return data
}

// ── Options API ──

export async function searchModelOptions(
  q: string,
  limit = 20
): Promise<TokenUsageOptionsResponse> {
  const { data } = await apiClient.get<TokenUsageOptionsResponse>(
    '/admin/token-usage/options/models',
    { params: { q, limit } }
  )
  return data
}

export async function searchGroupOptions(
  q: string,
  limit = 20
): Promise<TokenUsageOptionsResponse> {
  const { data } = await apiClient.get<TokenUsageOptionsResponse>(
    '/admin/token-usage/options/groups',
    { params: { q, limit } }
  )
  return data
}

export async function searchRouteOptions(
  groupId: number,
  q: string,
  limit = 20
): Promise<TokenUsageOptionsResponse> {
  const { data } = await apiClient.get<TokenUsageOptionsResponse>(
    `/admin/token-usage/options/groups/${groupId}/routes`,
    { params: { q, limit } }
  )
  return data
}

export async function searchRouteModelOptions(
  groupId: number,
  routeAlias: string,
  q: string,
  limit = 20
): Promise<TokenUsageOptionsResponse> {
  const { data } = await apiClient.get<TokenUsageOptionsResponse>(
    `/admin/token-usage/options/groups/${groupId}/routes/${encodeURIComponent(routeAlias)}/models`,
    { params: { q, limit } }
  )
  return data
}

export async function searchUserOptions(
  q: string,
  limit = 20
): Promise<TokenUsageOptionsResponse> {
  const { data } = await apiClient.get<TokenUsageOptionsResponse>(
    '/admin/token-usage/options/users',
    { params: { q, limit } }
  )
  return data
}

export async function searchUserModelOptions(
  userId: number,
  q: string,
  limit = 20
): Promise<TokenUsageOptionsResponse> {
  const { data } = await apiClient.get<TokenUsageOptionsResponse>(
    `/admin/token-usage/options/users/${userId}/models`,
    { params: { q, limit } }
  )
  return data
}

// ── Default target API ──

export async function getDefaultTarget(
  dimension: 'model' | 'route' | 'user',
  date?: string
): Promise<TokenUsageDefaultTargetResponse> {
  const { data } = await apiClient.get<TokenUsageDefaultTargetResponse>(
    '/admin/token-usage/default-target',
    { params: { dimension, date } }
  )
  return data
}

// ── Unified export ──

export const tokenUsageAPI = {
  getModelReport: getModelTokenUsageReport,
  getRouteReport: getRouteTokenUsageReport,
  getUserReport: getUserModelTokenUsageReport,
  searchModels: searchModelOptions,
  searchGroups: searchGroupOptions,
  searchRoutes: searchRouteOptions,
  searchRouteModels: searchRouteModelOptions,
  searchUsers: searchUserOptions,
  searchUserModels: searchUserModelOptions,
  getDefaultTarget
}

export default tokenUsageAPI

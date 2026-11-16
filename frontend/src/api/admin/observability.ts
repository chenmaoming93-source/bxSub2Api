import { apiClient } from '../client'
import type { ApiResponse } from '@/types'

export type ObservabilityWindow = '1h' | '6h' | '24h' | '7d' | 'today' | 'yesterday' | 'custom'
export type ObservabilityBucket = '5m' | '15m' | '1h' | '6h'
export type ObservabilityGroupBy = 'user_id' | 'group_id' | 'route_alias' | 'account_id'

export interface ObservabilityFilters {
  userIds: number[]
  apiKeyIds: number[]
  groupIds: number[]
  routeAliases: string[]
  accountIds: number[]
  department: string
}

export interface ObservabilityMeta {
  as_of: string
  start_at: string
  end_at: string
  window: ObservabilityWindow
  bucket: ObservabilityBucket
  timezone: string
  sources: string[]
  consistency: string
  lag_seconds: number
  complete: boolean
  warnings: string[]
  coverage?: string
  etag?: string
}

export interface ObservabilityMetric {
  code: string
  value: number
  unit?: string
}

export interface DimensionRow {
  key: string
  name: string
  value: number
}

export interface UserFilterOption {
  user_id: number
  email: string
  department: string
}

export interface NamedFilterOption {
  id: number
  name: string
}

export interface AccountFilterOption {
  account_id: number
  name: string
  bound_model: string
}

export interface SnapshotFilterOptions {
  users: UserFilterOption[]
  groups: NamedFilterOption[]
  aliases: NamedFilterOption[]
  accounts: AccountFilterOption[]
}

export interface PressureRow {
  account_id: number
  account_name?: string
  group_id?: number
  group_name?: string
  route_alias?: string
  current: number
  limit: number
  utilization: number
  waiting: number
}

export interface ObservabilitySnapshot {
  meta: ObservabilityMeta
  metrics: ObservabilityMetric[]
  top_users: DimensionRow[]
  top_groups: DimensionRow[]
  top_aliases: DimensionRow[]
  top_accounts: DimensionRow[]
  top_models: DimensionRow[]
  pressure: {
    accounts: PressureRow[]
    candidates: PressureRow[]
  }
  capacity: CapacitySummary
  filter_options: SnapshotFilterOptions
}

export interface CapacityLevel {
  name?: string
  current: number
  limit: number
  utilization: number
  available: boolean
}

export interface CapacitySummary {
  global: CapacityLevel
  user: CapacityLevel
  account: CapacityLevel
  route: CapacityLevel
  queue_current: number
  average_wait_ms?: number
  today_rate_limited_count: number
  rate_limit_available: boolean
  wait_sample_count: number
}

export interface Series {
  key: string
  name: string
  points: number[]
}

export interface TimeSeriesResponse {
  meta: ObservabilityMeta
  labels: string[]
  series: Series[]
  summary?: Series[]
}

export interface HeatmapCell {
  row: string
  column: string
  value: number
  available: boolean
  actual_concurrency?: number
  concurrency_limit?: number
  sample_at?: string
}

export interface HeatmapResponse {
  meta: ObservabilityMeta
  value_kind: string
  available: boolean
  rows: string[]
  columns: string[]
  cells: HeatmapCell[]
}

export interface SnapshotRequestOptions {
  window: ObservabilityWindow
  bucket: ObservabilityBucket
  startAt?: string
  endAt?: string
  filters: ObservabilityFilters
  search?: string
  signal?: AbortSignal
  etag?: string
}

export interface PathRow {
  user_id: number
  user_email: string
  department: string
  api_key_id: number
  api_key_name: string
  group_id: number
  group_name: string
  route_alias: string
  account_id: number
  account_name: string
  bound_model: string
  tokens: number
  request_count: number
  success_rate: number
  average_duration_ms: number
  average_first_token_ms: number
  duration_sample_count: number
  ttft_sample_count: number
  p95_first_token_ms: number
  data_at?: string
  error_stage?: string
  previous_average_duration_ms?: number
  previous_average_first_token_ms?: number
  current: number
  limit: number
  legacy_inferred: boolean
  status: string
  model_attribution: string
}

export interface PathsResponse {
  meta: ObservabilityMeta
  total: number
  rows: PathRow[]
}

export interface RelationNode {
  id: string
  kind: string
  label: string
  user_id?: number
  user_email?: string
  department?: string
  api_key_id?: number
  group_id?: number
  route_alias?: string
  route_pattern?: string
  account_id?: number
  bound_model?: string
  model_attribution?: string
}

export interface RelationEdge {
  from: string
  to: string
  kind: string
  idle: boolean
}

export interface RelationsResponse {
  meta: ObservabilityMeta
  nodes: RelationNode[]
  edges: RelationEdge[]
  data_quality_errors: string[]
}

export interface FetchResult<T> {
  data: T | null
  notModified: boolean
  etag?: string
}

function appendValues(params: URLSearchParams, key: string, values: readonly (string | number)[]) {
  for (const value of values) params.append(key, String(value))
}

export function buildObservabilityParams(
  options: Omit<SnapshotRequestOptions, 'signal' | 'etag'>,
  includeOptions = true,
): URLSearchParams {
  const params = new URLSearchParams({
    window: options.window,
    bucket: options.bucket,
  })
  if (options.startAt) params.set('start_at', options.startAt)
  if (options.endAt) params.set('end_at', options.endAt)
  appendValues(params, 'user_id', options.filters.userIds)
  appendValues(params, 'api_key_id', options.filters.apiKeyIds)
  appendValues(params, 'group_id', options.filters.groupIds)
  appendValues(params, 'route_alias', options.filters.routeAliases)
  appendValues(params, 'account_id', options.filters.accountIds)
  if (options.filters.department) params.set('department', options.filters.department)
  if (includeOptions && options.search?.trim()) params.set('search', options.search.trim())
  return params
}

async function getWithETag<T>(path: string, params: URLSearchParams, options: Pick<SnapshotRequestOptions, 'signal' | 'etag'>): Promise<FetchResult<T>> {
  const response = await apiClient.get<ApiResponse<T>>(`${path}?${params.toString()}`, {
    signal: options.signal,
    headers: options.etag ? { 'If-None-Match': options.etag } : undefined,
    validateStatus: (status) => status === 200 || status === 304,
  })
  const etagHeader = response.headers.etag as string | undefined
  if (response.status === 304) {
    return { data: null, notModified: true, etag: etagHeader || options.etag }
  }
  return { data: response.data as T, notModified: false, etag: etagHeader }
}

export function getObservabilitySnapshot(options: SnapshotRequestOptions) {
  return getWithETag<ObservabilitySnapshot>(
    '/admin/observability/snapshot',
    buildObservabilityParams(options),
    options,
  )
}

export function getObservabilityTimeseries(
  options: SnapshotRequestOptions & { groupBy?: ObservabilityGroupBy },
) {
  const params = buildObservabilityParams(options, false)
  if (options.groupBy) params.set('group_by', options.groupBy)
  return getWithETag<TimeSeriesResponse>('/admin/observability/timeseries', params, options)
}

export function getObservabilityLatency(
  options: SnapshotRequestOptions,
) {
  return getWithETag<TimeSeriesResponse>(
    '/admin/observability/latency-timeseries',
    buildObservabilityParams(options),
    options,
  )
}

export function getObservabilityHeatmap(
  options: SnapshotRequestOptions & {
    groupBy?: ObservabilityGroupBy
    valueKind?: 'sampled_concurrency' | 'token_rate' | 'error_rate'
  },
) {
  const params = buildObservabilityParams(options, false)
  if (options.groupBy) params.set('group_by', options.groupBy)
  if (options.valueKind) params.set('value_kind', options.valueKind)
  return getWithETag<HeatmapResponse>(
    '/admin/observability/heatmap',
    params,
    options,
  )
}

export interface PathRequestOptions extends SnapshotRequestOptions {
  page?: number
  pageSize?: number
  sort?: string
  status?: string
  includeIdle?: boolean
}

export function getObservabilityPaths(options: PathRequestOptions) {
  const params = buildObservabilityParams(options)
  params.set('page', String(options.page || 1))
  params.set('page_size', String(options.pageSize || 20))
  if (options.sort) params.set('sort', options.sort)
  if (options.status) params.set('status', options.status)
  params.set('include_idle', String(Boolean(options.includeIdle)))
  return getWithETag<PathsResponse>('/admin/observability/paths', params, options)
}

export function getObservabilityRelations(options: PathRequestOptions) {
  const params = buildObservabilityParams(options)
  params.set('include_idle', String(Boolean(options.includeIdle)))
  return getWithETag<RelationsResponse>('/admin/observability/relations', params, options)
}

import { apiClient } from '../client'

export type DimensionCode = 'user_id' | 'api_key_id' | 'group_id' | 'route_alias' | 'account_id' | 'upstream_model'
export type MetricCode = 'total_tokens'
export type PeriodType = 'D' | 'W' | 'M'
export type ProjectionStatus = 'DRAFT' | 'PUBLISHED' | 'ACTIVE' | 'DISABLED'
export type QuotaStatus = 'PENDING' | 'ENABLED' | 'DISABLED'

export interface DimensionDefinition {
  code: DimensionCode
  display_name: string
  value_type: 'int64' | 'string'
  order: number
  version: number
}

export interface MetricDefinition {
  code: MetricCode
  display_name: string
  unit: string
  allow_quota: boolean
  version: number
}

export interface Projection {
  id: number
  name: string
  dimension_codes: DimensionCode[]
  status: ProjectionStatus
  published_at?: string
  enabled_at?: string
  disabled_at?: string
}

export interface DimensionValue {
  type: 'int64' | 'string'
  int64?: number
  string?: string
}

export interface Quota {
  id: number
  name: string
  projection_id: number
  dimension_values: Record<string, number | string>
  metric_code: MetricCode
  period_type: PeriodType
  limit_value: number
  enforcement_mode: 'OBSERVE' | 'ENFORCE'
  status: QuotaStatus
  effective_from: string
}

export interface ProjectionInput {
  name: string
  dimension_codes: DimensionCode[]
  metric_codes: MetricCode[]
}

export interface QuotaInput {
  name: string
  dimension_codes: DimensionCode[]
  dimension_values: Partial<Record<DimensionCode, DimensionValue>>
  metric_code: MetricCode
  period_type: PeriodType
  limit_value: number
  mode: 'OBSERVE' | 'ENFORCE'
}

export interface SyncStatus {
  periods: Array<{ id: number; period_type: PeriodType; period_start: string; period_end: string; state: string; last_error?: string }>
  last_synced_at?: string
  metrics: Record<string, number>
}

export interface RuntimeState {
  available: boolean
  enabled: boolean
}

export interface UsageQueryInput {
  projection_id: number
  metric_code: MetricCode
  period_type: PeriodType
  start: string
  end: string
  filters?: Partial<Record<DimensionCode, DimensionValue>>
  group_by?: DimensionCode[]
  sort?: 'time_asc' | 'time_desc' | 'value_asc' | 'value_desc'
  page?: number
  page_size?: number
  format?: 'json' | 'csv'
}

export interface UsageQueryResult {
  rows: Array<{ period_start: string; period_end: string; dimensions: Partial<Record<DimensionCode, DimensionValue>>; value: number }>
  total: number
  summary: number
  projection_enabled_at?: string
  last_synced_at?: string
  complete: boolean
  consistency: 'mysql_eventual'
}

const base = '/admin/token-statistics'

export const dynamicTokenStatisticsAPI = {
  async dimensions() {
    const { data } = await apiClient.get<{ dimensions: DimensionDefinition[] }>(`${base}/dimensions`)
    return data.dimensions
  },
  async metrics() {
    const { data } = await apiClient.get<{ metrics: MetricDefinition[] }>(`${base}/metrics`)
    return data.metrics
  },
  async projections() {
    const { data } = await apiClient.get<{ projections: Projection[] }>(`${base}/projections`)
    return data.projections
  },
  async createProjection(input: ProjectionInput) {
    const { data } = await apiClient.post<{ projection: Projection }>(`${base}/projections`, input)
    return data.projection
  },
  async updateProjection(id: number, input: ProjectionInput) {
    const { data } = await apiClient.put<{ projection: Projection }>(`${base}/projections/${id}`, input)
    return data.projection
  },
  async projectionAction(id: number, action: 'publish' | 'activate' | 'disable') {
    const { data } = await apiClient.post<{ projection: Projection }>(`${base}/projections/${id}/${action}`)
    return data.projection
  },
  async quotas() {
    const { data } = await apiClient.get<{ quotas: Quota[] }>(`${base}/quotas`)
    return data.quotas
  },
  async createQuota(input: QuotaInput) {
    const { data } = await apiClient.post<{ quota: Quota }>(`${base}/quotas`, input)
    return data.quota
  },
  async updateQuota(id: number, input: Pick<QuotaInput, 'name' | 'limit_value' | 'mode'>) {
    const { data } = await apiClient.put<{ quota: Quota }>(`${base}/quotas/${id}`, input)
    return data.quota
  },
  async deleteQuota(id: number) {
    await apiClient.delete(`${base}/quotas/${id}`)
  },
  async quotaAction(id: number, action: 'enable' | 'disable') {
    const { data } = await apiClient.post<{ quota: Quota }>(`${base}/quotas/${id}/${action}`)
    return data.quota
  },
  async status() {
    const { data } = await apiClient.get<SyncStatus>(`${base}/status`)
    return data
  },
  async runtime() {
    const { data } = await apiClient.get<RuntimeState>(`${base}/runtime`)
    return data
  },
  async updateRuntime(enabled: boolean) {
    const { data } = await apiClient.put<RuntimeState>(`${base}/runtime`, { enabled })
    return data
  },
  async query(input: UsageQueryInput) {
    const { data } = await apiClient.post<UsageQueryResult>(`${base}/query`, input)
    return data
  },
  async exportCSV(input: UsageQueryInput) {
    const { data } = await apiClient.post<Blob>(`${base}/query`, { ...input, format: 'csv', page: 1, page_size: 1000 }, { responseType: 'blob' })
    return data
  }
}

export default dynamicTokenStatisticsAPI

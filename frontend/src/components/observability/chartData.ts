import type {
  HeatmapResponse,
  Series,
  TimeSeriesResponse,
} from '@/api/admin/observability'

export interface ChartDataset {
  label: string
  data: number[]
  borderColor: string
  backgroundColor: string
  fill?: boolean
  tension?: number
  yAxisID?: string
  borderDash?: number[]
}

type LabelResolver = (key: string, fallback: string) => string

export function alignedSeries(labels: string[], series: Series[]): Series[] {
  return series.filter((item) => item.points.length === labels.length)
}

export function tokenChartDatasets(
  labels: string[],
  response: TimeSeriesResponse | null,
  concurrency: number[] = [],
  metric: 'total' | 'input' | 'output' | 'cache' = 'total',
  labelFor: LabelResolver = (_key, fallback) => fallback,
): ChartDataset[] {
  if (!response) return []
  const datasets: ChartDataset[] = []
  const tokenKey = metric === 'total' ? 'total_tokens' : `${metric}_tokens`
  const tokens = response.series.find((item) => item.key === tokenKey)
  const requests = response.series.find((item) => item.key === 'total_requests')
  if (tokens?.points.length === labels.length) {
    datasets.push({
      label: labelFor(tokenKey, tokens.name || 'Tokens'),
      data: [...tokens.points],
      borderColor: '#14b8a6',
      backgroundColor: '#14b8a620',
      fill: true,
      tension: 0.3,
      yAxisID: 'yTokens',
    })
  }
  if (requests?.points.length === labels.length) {
    datasets.push({
      label: labelFor('total_requests', requests.name || 'Requests'),
      data: [...requests.points],
      borderColor: '#10b981',
      backgroundColor: '#10b98120',
      fill: false,
      tension: 0.3,
      yAxisID: 'yRequests',
    })
  }
  if (concurrency.length === labels.length) {
    datasets.push({
      label: labelFor('sampled_concurrency', 'Sampled concurrency'),
      data: [...concurrency],
      borderColor: '#f59e0b',
      backgroundColor: '#f59e0b20',
      fill: false,
      tension: 0.2,
      borderDash: [5, 4],
      yAxisID: 'yConcurrency',
    })
  }
  return datasets
}

export function qualityChartDatasets(
  labels: string[],
  response: TimeSeriesResponse | null,
  mode: 'all' | 'duration' | 'ttft' | 'failure' = 'all',
  labelFor: LabelResolver = (_key, fallback) => fallback,
): ChartDataset[] {
  if (!response) return []
  const hasTTFTSamples = !response.meta.warnings.includes('partial_latency_histogram')
  const definitions: Array<{ key: string; color: string }> = [
    { key: 'avg_duration_ms', color: '#ef4444' },
    { key: 'avg_first_token_ms', color: '#14b8a6' },
    { key: 'client_failure_rate', color: '#f59e0b' },
    { key: 'business_limited_rate', color: '#0ea5e9' },
    { key: 'sla_error_rate', color: '#dc2626' },
  ]
  const datasets: ChartDataset[] = alignedSeries(labels, response.series)
    .filter((item) => {
      if (!definitions.some((definition) => definition.key === item.key)) return false
      if (mode === 'duration') return item.key === 'avg_duration_ms'
      if (mode === 'ttft') return item.key === 'avg_first_token_ms'
      if (mode === 'failure') return item.key.endsWith('_rate')
      return true
    })
    .filter((item) => item.key !== 'avg_first_token_ms' || hasTTFTSamples)
    .map((item) => {
      const color = definitions.find((definition) => definition.key === item.key)?.color || '#64748b'
      const rate = item.key.endsWith('_rate')
      return {
        label: labelFor(item.key, item.name || item.key),
        data: rate ? item.points.map((point) => point * 100) : [...item.points],
        borderColor: color,
        backgroundColor: `${color}20`,
        fill: false,
        tension: 0.25,
        yAxisID: rate ? 'yRate' : 'yLatency',
      }
    })
  if (labels.length && mode !== 'failure') {
    datasets.push({
      label: labelFor('slo_2000ms', 'SLO 2000ms'),
      data: labels.map(() => 2000),
      borderColor: '#94a3b8',
      backgroundColor: '#94a3b820',
      fill: false,
      tension: 0,
      borderDash: [6, 4],
      yAxisID: 'yLatency',
    })
  }
  return datasets
}

export interface HeatmapMatrix {
  rows: string[]
  columns: string[]
  cells: Array<Array<{
    value: number
    available: boolean
    actual_concurrency?: number
    concurrency_limit?: number
    sample_at?: string
  }>>
}

export function heatmapMatrix(response: HeatmapResponse | null): HeatmapMatrix {
  if (!response) return { rows: [], columns: [], cells: [] }
  const columns = [...response.columns]
  const rows = [...response.rows]
  const lookup = new Map(response.cells.map((cell) => [`${cell.row}\u0000${cell.column}`, cell]))
  const cells = rows.map((row) =>
    columns.map((column) => {
      const cell = lookup.get(`${row}\u0000${column}`)
      return cell
        ? {
            value: cell.value,
            available: cell.available,
            actual_concurrency: cell.actual_concurrency,
            concurrency_limit: cell.concurrency_limit,
            sample_at: cell.sample_at,
          }
        : { value: 0, available: false }
    }),
  )
  return { rows, columns, cells }
}

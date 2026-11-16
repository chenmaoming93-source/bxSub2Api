import { describe, expect, it } from 'vitest'
import type { HeatmapResponse, TimeSeriesResponse } from '@/api/admin/observability'
import {
  heatmapMatrix,
  qualityChartDatasets,
  tokenChartDatasets,
} from '../chartData'

function response(overrides: Partial<TimeSeriesResponse> = {}): TimeSeriesResponse {
  return {
    meta: {
      as_of: '2026-11-11T00:00:00Z', window: '1h', bucket: '5m', timezone: 'Asia/Shanghai',
      sources: [], consistency: 'eventual', lag_seconds: 0, complete: true, warnings: [],
    },
    labels: ['10:00', '10:05'],
    series: [
      { key: 'total_tokens', name: 'Tokens', points: [10, 20] },
      { key: 'total_requests', name: 'Requests', points: [1, 2] },
    ],
    ...overrides,
  }
}

describe('observability chart adapters', () => {
  it('keeps server buckets and never interpolates or averages points', () => {
    const datasets = tokenChartDatasets(['10:00', '10:05'], response())
    expect(datasets.map((item) => item.data)).toEqual([[10, 20], [1, 2]])
  })

  it('drops misaligned series instead of inventing points', () => {
    const payload = response({ series: [{ key: 'total_tokens', name: 'Tokens', points: [10] }] })
    expect(tokenChartDatasets(['10:00', '10:05'], payload)).toEqual([])
  })

  it('hides zero-looking TTFT when histogram samples are missing', () => {
    const payload = response({
      meta: {
        ...response().meta,
        complete: false,
        warnings: ['partial_latency_histogram'],
      },
      series: [
        { key: 'avg_duration_ms', name: 'Average duration', points: [300, 400] },
        { key: 'avg_first_token_ms', name: 'Average TTFT', points: [0, 0] },
      ],
    })
    expect(qualityChartDatasets(payload.labels, payload).map((item) => item.label)).toEqual(['Average duration', 'SLO 2000ms'])
  })

  it('uses average latency metrics rather than P95 for the primary trend', () => {
    const payload = response({
      series: [
        { key: 'avg_duration_ms', name: 'Average duration', points: [1200, 1300] },
        { key: 'avg_first_token_ms', name: 'Average TTFT', points: [300, 320] },
        { key: 'p95_duration_ms', name: 'P95 duration', points: [9000, 9500] },
      ],
    })
    const labels = qualityChartDatasets(payload.labels, payload).map((item) => item.label)
    expect(labels).toContain('Average duration')
    expect(labels).toContain('Average TTFT')
    expect(labels).not.toContain('P95 duration')
  })

  it('does not fabricate missing heatmap cells', () => {
    const payload: HeatmapResponse = {
      meta: response().meta,
      value_kind: 'sampled_concurrency', available: true,
      rows: ['7'], columns: ['10:00', '10:05'],
      cells: [{ row: '7', column: '10:00', value: 5, available: true }],
    }
    expect(heatmapMatrix(payload).cells).toEqual([
      [{ value: 5, available: true }, { value: 0, available: false }],
    ])
  })
})

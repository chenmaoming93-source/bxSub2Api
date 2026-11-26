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

  it('derives cache hit rate from cache divided by input plus cache', () => {
    const labels = ['10:00', '10:05', '10:10']
    const payload = response({
      series: [
        { key: 'total_tokens', name: 'Tokens', points: [100, 10, 0] },
        { key: 'input_tokens', name: 'Input', points: [30, 0, 0] },
        { key: 'output_tokens', name: 'Output', points: [50, 10, 0] },
        { key: 'cache_tokens', name: 'Cache', points: [10, 5, 0] },
        { key: 'total_requests', name: 'Requests', points: [1, 1, 0] },
      ],
    })

    const hitRate = tokenChartDatasets(labels, payload).find((item) => item.key === 'cache_hit_rate')
    expect(hitRate).toMatchObject({
      data: [0.25, 1, 0],
      yAxisID: 'yRate',
    })
  })

  it('renders all supported Token metrics as named dimension series', () => {
    const labels = ['10:00', '10:05']
    const metrics = [
      ['total', 'tokens'],
      ['input', 'input_tokens'],
      ['output', 'output_tokens'],
      ['cache', 'cache_tokens'],
    ] as const
    for (const [metric, prefix] of metrics) {
      const payload = response({
        series: [
          { key: `${metric === 'total' ? 'total' : metric}_tokens`, name: 'Overall', points: [30, 40] },
          { key: `${prefix}:7`, name: '7', points: [10, 15] },
          { key: `${prefix}:8`, name: '8', points: [20, 25] },
        ],
      })
      const datasets = tokenChartDatasets(
        labels,
        payload,
        [],
        metric,
        (key, fallback) => key === 'overall_token' ? 'Overall Token' : fallback,
        'account_id',
        { '7': 'claude-pro-01', '8': 'codex-team-a' },
      )
      expect(datasets.find((item) => item.label === 'claude-pro-01')).toMatchObject({ data: [10, 15] })
      expect(datasets.find((item) => item.label === 'codex-team-a')).toMatchObject({ data: [20, 25] })
      expect(datasets.some((item) => item.label === 'Overall Token')).toBe(true)
    }
  })

  it('keeps the top eight dimensions and reconciles the remainder into Other', () => {
    const labels = ['10:00']
    const grouped = Array.from({ length: 9 }, (_value, index) => ({
      key: `tokens:${index}`,
      name: String(index),
      points: [10],
    }))
    const payload = response({
      labels,
      series: [
        { key: 'total_tokens', name: 'Overall Token', points: [100] },
        ...grouped,
      ],
    })
    const labelsByKey = Object.fromEntries(grouped.map((item, index) => [String(index), `账号 ${index}`]))
    const datasets = tokenChartDatasets(
      labels,
      payload,
      [],
      'total',
      (key, fallback) => key === 'dimension_other'
        ? '其他'
        : key === 'overall_token' ? 'Overall Token' : fallback,
      'account_id',
      labelsByKey,
    )
    const dimensionDatasets = datasets.filter((item) => item.yAxisID === 'yTokens')
    expect(dimensionDatasets.map((item) => item.label)).toContain('其他')
    expect(dimensionDatasets.some((item) => item.label === '账号 8')).toBe(false)
    const splitTotal = dimensionDatasets
      .filter((item) => item.label !== 'Overall Token')
      .reduce((total, item) => total + item.data[0], 0)
    expect(splitTotal).toBe(100)
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

import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import type { TimeSeriesResponse } from '@/api/admin/observability'
import QualityTrendChart from '../QualityTrendChart.vue'

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'Line',
    props: ['data', 'options'],
    template: '<div data-testid="quality-chart" />',
  },
}))

const meta = {
  as_of: '2026-11-16T10:00:00Z',
  start_at: '2026-11-16T09:00:00Z',
  end_at: '2026-11-16T10:00:00Z',
  window: '1h' as const,
  bucket: '5m' as const,
  timezone: 'Asia/Shanghai',
  sources: [],
  consistency: 'eventual',
  lag_seconds: 0,
  complete: true,
  warnings: [],
}

const response: TimeSeriesResponse = {
  meta,
  labels: ['09:55', '10:00'],
  series: [
    { key: 'client_failure_rate', name: 'Final failure rate', points: [0.01, 0.02] },
    { key: 'business_limited_rate', name: 'Business-limited rate', points: [0.005, 0.006] },
    { key: 'sla_error_rate', name: 'SLA error rate', points: [0.001, 0.002] },
  ],
}

describe('QualityTrendChart rate legend', () => {
  it('explains rate semantics on hover and keeps legend toggles accessible', async () => {
    const wrapper = mount(QualityTrendChart, {
      props: { labels: response.labels, response, mode: 'failure' },
    })
    const client = wrapper.findAll('button').find((button) => button.text().includes('admin.observability.clientFailureRate'))!
    const business = wrapper.findAll('button').find((button) => button.text().includes('admin.observability.businessLimitedRate'))!
    const sla = wrapper.findAll('button').find((button) => button.text().includes('admin.observability.slaErrorRate'))!

    expect(client.attributes('title')).toContain('admin.observability.clientFailureRateHelp')
    expect(business.attributes('title')).toContain('admin.observability.businessLimitedRateHelp')
    expect(sla.attributes('title')).toContain('admin.observability.slaErrorRateHelp')
    expect(client.attributes('aria-pressed')).toBe('true')

    await client.trigger('click')
    expect(client.attributes('aria-pressed')).toBe('false')
  })
})

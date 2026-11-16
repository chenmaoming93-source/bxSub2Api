import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import QualityTopPeriods from '../QualityTopPeriods.vue'
import type { TimeSeriesResponse } from '@/api/admin/observability'

const response: TimeSeriesResponse = {
  meta: {
    as_of: '2026-11-13T00:00:00Z', start_at: '2026-11-12T23:00:00Z', end_at: '2026-11-13T00:00:00Z',
    window: '1h', bucket: '5m', timezone: 'Asia/Shanghai', sources: [], consistency: 'eventual',
    lag_seconds: 0, complete: true, warnings: [],
  },
  labels: ['10:00', '10:05'],
  series: [
    { key: 'avg_duration_ms', name: 'Average duration', points: [1200, 1800] },
    { key: 'avg_first_token_ms', name: 'Average TTFT', points: [0, 400] },
  ],
}

describe('QualityTopPeriods', () => {
  it('sorts server-provided averages and shows missing TTFT as unavailable', () => {
    const wrapper = mount(QualityTopPeriods, {
      props: { response },
    })
    expect(wrapper.text()).toContain('10:05')
    expect(wrapper.text()).toContain('1800ms')
    expect(wrapper.text()).toContain('400ms')
    expect(wrapper.text()).toContain('admin.observability.noSampling')
  })

  it('keeps loading, error and empty states independent', async () => {
    const wrapper = mount(QualityTopPeriods, {
      props: { response: null, loading: true },
    })
    expect(wrapper.find('.animate-pulse').exists()).toBe(true)
    await wrapper.setProps({ loading: false, error: 'query failed' })
    expect(wrapper.text()).toContain('query failed')
  })
})

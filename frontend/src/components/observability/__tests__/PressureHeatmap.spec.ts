import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { HeatmapResponse } from '@/api/admin/observability'
import PressureHeatmap from '../PressureHeatmap.vue'

const meta = {
  as_of: '2026-11-11T00:00:00Z', window: '1h' as const, bucket: '5m' as const,
  timezone: 'Asia/Shanghai', sources: [], consistency: 'eventual', lag_seconds: 0,
  complete: true, warnings: [],
}

describe('PressureHeatmap', () => {
  it('shows unavailable instead of a fabricated zero heatmap', () => {
    const response: HeatmapResponse = {
      meta, value_kind: 'sampled_concurrency', available: false,
      rows: [], columns: [], cells: [],
    }
    const wrapper = mount(PressureHeatmap, {
      props: { response, valueKind: 'sampled_concurrency', singleAccountFiltered: true },
    })
    expect(wrapper.text()).toContain('unavailable')
    expect(wrapper.find('table').exists()).toBe(false)
  })

  it('hides rows with no cell that has a valid concurrency limit', () => {
    const response: HeatmapResponse = {
      meta, value_kind: 'sampled_concurrency', available: true,
      rows: ['7'], columns: ['10:00'],
      cells: [{ row: '7', column: '10:00', value: 0, available: false }],
    }
    const wrapper = mount(PressureHeatmap, {
      props: { response, valueKind: 'sampled_concurrency', singleAccountFiltered: true },
    })
    expect(wrapper.find('table').exists()).toBe(false)
    expect(wrapper.text()).toContain('admin.observability.noHeatmapRows')
  })

  it('emits row selection without mutating URL data itself', async () => {
    const response: HeatmapResponse = {
      meta, value_kind: 'sampled_concurrency', available: true,
      rows: ['7'], columns: ['10:00'],
      cells: [{
        row: '7', column: '10:00', value: 0.8, available: true,
        actual_concurrency: 8, concurrency_limit: 10, sample_at: '2026-11-11T00:02:20Z',
      }],
    }
    const wrapper = mount(PressureHeatmap, {
      props: { response, valueKind: 'sampled_concurrency' },
    })
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('select')?.[0]?.[0]).toMatchObject({ row: '7', column: '10:00' })
  })

  it('renders row display names while preserving the raw key for selection', async () => {
    const response: HeatmapResponse = {
      meta, value_kind: 'error_rate', available: true,
      rows: ['7'], columns: ['10:00'],
      cells: [{ row: '7', column: '10:00', value: 0.12, available: true }],
    }
    const wrapper = mount(PressureHeatmap, {
      props: {
        response,
        valueKind: 'error_rate',
        rowLabels: { '7': 'DS-ZLX测试' },
      },
    })
    expect(wrapper.get('tbody th').text()).toBe('DS-ZLX测试')
    await wrapper.get('button').trigger('mouseenter')
    expect(wrapper.text()).toContain('DS-ZLX测试')
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('select')?.[0]?.[0]).toMatchObject({ row: '7' })
  })

  it('explains the error-rate metric on hover', () => {
    const wrapper = mount(PressureHeatmap, {
      props: {
        response: {
          meta,
          value_kind: 'error_rate',
          available: true,
          rows: ['7'],
          columns: ['10:00'],
          cells: [{ row: '7', column: '10:00', value: 0.1, available: true }],
        },
        valueKind: 'error_rate',
        valueKindHelp: 'admin.observability.heatmapErrorRateHelp',
      },
    })
    expect(wrapper.get('[title="admin.observability.heatmapErrorRateHelp"]').text()).toContain('admin.observability.errorRate')
  })

  it('shows paired actual/limit values and sample time in the hover tooltip', async () => {
    const response: HeatmapResponse = {
      meta, value_kind: 'sampled_concurrency', available: true,
      rows: ['claude-pro-01'], columns: ['10:00'],
      cells: [{
        row: 'claude-pro-01', column: '10:00', value: 0.8, available: true,
        actual_concurrency: 8, concurrency_limit: 10, sample_at: '2026-11-11T00:02:20Z',
      }],
    }
    const wrapper = mount(PressureHeatmap, {
      props: { response, valueKind: 'sampled_concurrency', singleAccountFiltered: true },
    })
    await wrapper.get('button').trigger('mouseenter')
    expect(wrapper.text()).toContain('8 / 10')
    expect(wrapper.text()).toContain('admin.observability.accountConcurrencyDenominator')
    expect(wrapper.text()).toContain('admin.observability.samplePeakTime')
  })

  it('colors sampled concurrency by absolute saturation instead of screen maximum', () => {
    const response: HeatmapResponse = {
      meta, value_kind: 'sampled_concurrency', available: true,
      rows: ['a', 'b'], columns: ['10:00'],
      cells: [
        { row: 'a', column: '10:00', value: 4, available: true, actual_concurrency: 4, concurrency_limit: 10 },
        { row: 'b', column: '10:00', value: 8, available: true, actual_concurrency: 8, concurrency_limit: 10 },
      ],
    }
    const wrapper = mount(PressureHeatmap, {
      props: { response, valueKind: 'sampled_concurrency' },
    })
    const buttons = wrapper.findAll('button')
    expect(buttons[0].attributes('style')).toContain('rgb(187, 247, 208)')
    expect(buttons[1].attributes('style')).toContain('rgb(253, 186, 116)')
  })

  it('uses the account denominator only when a single account is filtered', async () => {
    const response: HeatmapResponse = {
      meta, value_kind: 'sampled_concurrency', available: true,
      rows: ['one-row'], columns: ['10:00'],
      cells: [{
        row: 'one-row', column: '10:00', value: 0.8, available: true,
        actual_concurrency: 8, concurrency_limit: 10,
      }],
    }
    const rangeWrapper = mount(PressureHeatmap, {
      props: { response, valueKind: 'sampled_concurrency', singleAccountFiltered: false },
    })
    await rangeWrapper.get('button').trigger('mouseenter')
    expect(rangeWrapper.text()).toContain('admin.observability.rangeConcurrencyDenominator')
    const accountWrapper = mount(PressureHeatmap, {
      props: { response, valueKind: 'sampled_concurrency', singleAccountFiltered: true },
    })
    await accountWrapper.get('button').trigger('mouseenter')
    expect(accountWrapper.text()).toContain('admin.observability.accountConcurrencyDenominator')
  })
})

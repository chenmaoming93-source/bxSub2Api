import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ConcurrencyCapacityCard from '../ConcurrencyCapacityCard.vue'
import type { CapacitySummary } from '@/api/admin/observability'

const level = { name: 'claude-pro-01', current: 38, limit: 40, utilization: 0.95, available: true }
const capacity: CapacitySummary = {
  global: { ...level, name: '全局并发槽' },
  user: { ...level, name: '编码助手-公共' },
  account: level,
  route: { ...level, name: 'coding-fast', current: 34 },
  queue_current: 37,
  average_wait_ms: 420,
  today_rate_limited_count: 126,
  rate_limit_available: true,
  wait_sample_count: 100,
}

describe('ConcurrencyCapacityCard', () => {
  it('shows four levels and three bottom metrics', () => {
    const wrapper = mount(ConcurrencyCapacityCard, {
      props: { capacity },
    })
    expect(wrapper.text()).toContain('全局并发槽')
    expect(wrapper.text()).toContain('编码助手-公共')
    expect(wrapper.text()).toContain('claude-pro-01')
    expect(wrapper.text()).toContain('coding-fast')
    expect(wrapper.text()).toContain('37')
    expect(wrapper.text()).toContain('420 ms')
    expect(wrapper.text()).toContain('126')
    expect(wrapper.text()).toContain('admin.observability.calendarDayTotal')
  })

  it('shows unavailable wait state instead of 0ms', () => {
    const wrapper = mount(ConcurrencyCapacityCard, {
      props: { capacity: { ...capacity, average_wait_ms: undefined } },
    })
    expect(wrapper.text()).toContain('admin.observability.noSampling')
    expect(wrapper.text()).not.toContain('0 ms')
  })
})

import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import UsageStatusBadge from '@/components/admin/token-usage/UsageStatusBadge.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => ({ 'tokenUsageReport.normal': 'Normal', 'tokenUsageReport.warning': 'Warning', 'tokenUsageReport.exceeded': 'Exceeded', 'tokenUsageReport.unlimited': 'Unlimited' }[key] ?? key) }) }))

describe('UsageStatusBadge', () => {
  it('renders normal status', () => {
    const wrapper = mount(UsageStatusBadge, { props: { status: 'normal' } })
    expect(wrapper.text()).toContain('Normal')
    expect(wrapper.find('.bg-green-100').exists()).toBe(true)
  })

  it('renders warning status', () => {
    const wrapper = mount(UsageStatusBadge, { props: { status: 'warning' } })
    expect(wrapper.text()).toContain('Warning')
    expect(wrapper.find('.bg-yellow-100').exists()).toBe(true)
  })

  it('renders exceeded status', () => {
    const wrapper = mount(UsageStatusBadge, { props: { status: 'exceeded' } })
    expect(wrapper.text()).toContain('Exceeded')
    expect(wrapper.find('.bg-red-100').exists()).toBe(true)
  })

  it('renders unlimited status', () => {
    const wrapper = mount(UsageStatusBadge, { props: { status: 'unlimited' } })
    expect(wrapper.text()).toContain('Unlimited')
    expect(wrapper.find('.bg-gray-100').exists()).toBe(true)
  })

  it('renders unknown status as plain text', () => {
    const wrapper = mount(UsageStatusBadge, { props: { status: 'custom' } })
    expect(wrapper.text()).toContain('custom')
  })
})

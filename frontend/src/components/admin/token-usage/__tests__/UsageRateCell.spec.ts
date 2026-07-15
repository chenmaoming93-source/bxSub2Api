import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import UsageRateCell from '@/components/admin/token-usage/UsageRateCell.vue'

describe('UsageRateCell', () => {
  it('renders a capped progress bar and percentage', () => {
    const wrapper = mount(UsageRateCell, { props: { rate: 1.25 } })
    expect(wrapper.text()).toContain('125.0%')
    expect(wrapper.find('.bg-red-500').attributes('style')).toContain('width: 100%')
  })

  it('renders unlimited usage without a progress bar', () => {
    const wrapper = mount(UsageRateCell, { props: { rate: null } })
    expect(wrapper.text()).toBe('—')
    expect(wrapper.find('.h-2').exists()).toBe(false)
  })
})

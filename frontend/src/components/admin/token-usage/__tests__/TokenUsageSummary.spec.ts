import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import TokenUsageSummary from '@/components/admin/token-usage/TokenUsageSummary.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('TokenUsageSummary', () => {
  it('formats tokens under 1000', () => {
    const wrapper = mount(TokenUsageSummary, {
      props: { usedTokens: 42 }
    })
    expect(wrapper.text()).toContain('42')
  })

  it('formats thousands as K', () => {
    const wrapper = mount(TokenUsageSummary, {
      props: { usedTokens: 1500 }
    })
    expect(wrapper.text()).toContain('1.5K')
  })

  it('formats millions as M', () => {
    const wrapper = mount(TokenUsageSummary, {
      props: { usedTokens: 2_500_000 }
    })
    expect(wrapper.text()).toContain('2.5M')
  })

  it('formats billions as B', () => {
    const wrapper = mount(TokenUsageSummary, {
      props: { usedTokens: 1_200_000_000 }
    })
    expect(wrapper.text()).toContain('1.2B')
  })

  it('shows usage rate bar when limit is set', () => {
    const wrapper = mount(TokenUsageSummary, {
      props: { usedTokens: 80, dailyLimitTokens: 100 }
    })
    expect(wrapper.text()).toContain('80.0%')
    expect(wrapper.find('.bg-yellow-500').exists()).toBe(true)
  })

  it('shows red bar when exceeded', () => {
    const wrapper = mount(TokenUsageSummary, {
      props: { usedTokens: 150, dailyLimitTokens: 100 }
    })
    expect(wrapper.find('.bg-red-500').exists()).toBe(true)
  })

  it('hides rate bar when no limit', () => {
    const wrapper = mount(TokenUsageSummary, {
      props: { usedTokens: 100, dailyLimitTokens: null }
    })
    expect(wrapper.find('.h-2.w-24').exists()).toBe(false)
  })

  it('hides rate bar when limit is zero', () => {
    const wrapper = mount(TokenUsageSummary, {
      props: { usedTokens: 100, dailyLimitTokens: 0 }
    })
    expect(wrapper.find('.h-2.w-24').exists()).toBe(false)
  })
})

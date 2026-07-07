import { describe, it, expect, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import TokenUsageReportLayout from '@/components/admin/token-usage/TokenUsageReportLayout.vue'
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => ({ 'tokenUsageReport.retry': 'Retry', 'tokenUsageReport.empty': 'Select a target' }[key] ?? key) }) }))

describe('TokenUsageReportLayout', () => {
  it('renders title', () => {
    const wrapper = mount(TokenUsageReportLayout, {
      props: { title: 'Model Token Usage' }
    })
    expect(wrapper.text()).toContain('Model Token Usage')
  })

  it('shows loading skeleton when loading and no data', () => {
    const wrapper = mount(TokenUsageReportLayout, {
      props: { title: 'Test', loading: true, hasData: false }
    })
    expect(wrapper.find('.animate-pulse').exists()).toBe(true)
  })

  it('shows error state with retry button', () => {
    const wrapper = mount(TokenUsageReportLayout, {
      props: { title: 'Test', error: 'Something went wrong', hasData: false }
    })
    expect(wrapper.text()).toContain('Something went wrong')
    const buttons = wrapper.findAll('button')
    const retryBtn = buttons.find(b => b.text() === 'Retry')
    expect(retryBtn).toBeTruthy()
  })

  it('shows empty state when not loading and no data', () => {
    const wrapper = mount(TokenUsageReportLayout, {
      props: { title: 'Test', loading: false, hasData: false }
    })
    expect(wrapper.text()).toContain('Select a target')
  })

  it('shows custom empty message', () => {
    const wrapper = mount(TokenUsageReportLayout, {
      props: { title: 'Test', loading: false, hasData: false, emptyMessage: 'No results found' }
    })
    expect(wrapper.text()).toContain('No results found')
  })

  it('renders data slots when hasData is true', () => {
    const wrapper = mount(TokenUsageReportLayout, {
      props: { title: 'Test', hasData: true },
      slots: {
        summary: '<div class="summary-slot">Summary</div>',
        table: '<table><tr><td>data</td></tr></table>',
        pagination: '<div class="pagination-slot">Page 1</div>'
      }
    })
    expect(wrapper.find('.summary-slot').exists()).toBe(true)
    expect(wrapper.find('table').exists()).toBe(true)
    expect(wrapper.find('.pagination-slot').exists()).toBe(true)
  })

  it('hides skeleton when hasData is true even if loading', () => {
    const wrapper = mount(TokenUsageReportLayout, {
      props: { title: 'Test', loading: true, hasData: true }
    })
    expect(wrapper.find('.animate-pulse').exists()).toBe(false)
  })

  it('emits refresh event', async () => {
    const wrapper = mount(TokenUsageReportLayout, {
      props: { title: 'Test', hasData: true }
    })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('refresh')).toBeTruthy()
  })

  it('emits retry event from error state', async () => {
    const wrapper = mount(TokenUsageReportLayout, {
      props: { title: 'Test', error: 'Failed', hasData: false }
    })
    const buttons = wrapper.findAll('button')
    // The "Retry" button
    const retryBtn = buttons.find(b => b.text() === 'Retry')
    expect(retryBtn).toBeTruthy()
    await retryBtn!.trigger('click')
    expect(wrapper.emitted('retry')).toBeTruthy()
  })
})

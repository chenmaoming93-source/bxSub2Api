import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import DepartmentDistributionChart from '../DepartmentDistributionChart.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('vue-chartjs', () => ({
  Doughnut: { props: ['data'], template: '<div data-test="doughnut" />' }
}))

describe('DepartmentDistributionChart', () => {
  it('uses the same sorted token values for chart and list', () => {
    const wrapper = mount(DepartmentDistributionChart, {
      props: {
        rows: [
          { department: '研发部', total_tokens: 100, user_count: 2, average_tokens: 50, percentage: 25 },
          { department: '未设置', total_tokens: 300, user_count: 3, average_tokens: 100, percentage: 75 },
        ]
      }
    })

    const rows = wrapper.findAll('tbody tr')
    expect(rows[0].text()).toContain('未设置')
    expect(rows[0].text()).toContain('300')
    expect(rows[0].text()).toContain('75.0%')
    expect(rows[1].text()).toContain('研发部')
    expect(wrapper.find('[data-test="doughnut"]').exists()).toBe(true)
  })

  it('emits the selected department when a row is clicked', async () => {
    const wrapper = mount(DepartmentDistributionChart, {
      props: { rows: [{ department: '研发部', total_tokens: 100, user_count: 1, average_tokens: 100, percentage: 100 }] }
    })
    await wrapper.find('tbody tr').trigger('click')
    expect(wrapper.emitted('select')).toEqual([['研发部']])
  })
})

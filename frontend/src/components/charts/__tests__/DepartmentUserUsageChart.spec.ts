import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import DepartmentUserUsageChart from '../DepartmentUserUsageChart.vue'

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

vi.mock('vue-chartjs', () => ({
  Bar: { props: ['data'], template: '<div data-test="bar-chart"><span data-test="labels">{{ data.labels.join(\',\') }}</span><span data-test="colors">{{ data.datasets[0].backgroundColor.join(\',\') }}</span></div>' }
}))

describe('DepartmentUserUsageChart', () => {
  it('shows the highest-consuming users first and limits chart labels', () => {
    const rows = Array.from({ length: 21 }, (_, index) => ({
      user_id: index + 1,
      email: `user-${index + 1}@example.com`,
      username: `User ${index + 1}`,
      total_tokens: 21 - index,
      percentage: 0
    }))
    const wrapper = mount(DepartmentUserUsageChart, { props: { rows } })

    const labels = wrapper.find('[data-test="labels"]').text().split(',')
    const colors = wrapper.find('[data-test="colors"]').text().split(',')
    expect(labels).toHaveLength(20)
    expect(labels[0]).toBe('User 1')
    expect(labels[19]).toBe('User 20')
    expect(colors[0]).toBe('#4f46e5')
    expect(colors[1]).toBe('#818cf8')
    expect(colors[0]).not.toBe(colors[1])
  })
})

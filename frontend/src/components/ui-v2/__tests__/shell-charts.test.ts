import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { UiAppShell, UiChartCard, UiHorizontalRanking, UiResponsiveGrid } from '..'

describe('UI V2 shell and chart primitives', () => {
  it('renders the dark-sidebar shell and navigation items', () => {
    const wrapper = mount(UiAppShell, { props: { items: [{ label: '仪表盘', path: '/admin/dashboard', active: true }] } })
    expect(wrapper.find('.ui-v2-sidebar').exists()).toBe(true)
    expect(wrapper.find('.ui-v2-sidebar__item--active').text()).toContain('仪表盘')
    expect(wrapper.find('.ui-v2-shell__main').exists()).toBe(true)
  })

  it('renders chart loading state and responsive grid classes', () => {
    const chart = mount(UiChartCard, { props: { title: 'Token 趋势', loading: true } })
    expect(chart.find('.ui-v2-chart-card__state').text()).toContain('加载中')
    const grid = mount(UiResponsiveGrid, { props: { columns: 3 } })
    expect(grid.classes()).toContain('ui-v2-responsive-grid--3')
  })

  it('provides a dedicated search and Top 12/all switch for rankings', async () => {
    const ranking = mount(UiHorizontalRanking, { props: { items: Array.from({ length: 13 }, (_, index) => ({ id: index, label: `场景 ${index}`, value: 100 - index })) } })
    expect(ranking.find('.ui-v2-ranking__toolbar .ui-v2-search-input').exists()).toBe(true)
    expect(ranking.findAll('.ui-v2-ranking__row')).toHaveLength(12)
    await ranking.findAll('.ui-v2-ranking__scope button')[1].trigger('click')
    expect(ranking.findAll('.ui-v2-ranking__row')).toHaveLength(13)
  })
})

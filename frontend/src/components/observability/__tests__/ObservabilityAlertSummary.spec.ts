import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import ObservabilityAlertSummary from '../ObservabilityAlertSummary.vue'

const { listAlertEventsMock } = vi.hoisted(() => ({ listAlertEventsMock: vi.fn() }))
vi.mock('@/api/admin/ops', () => ({
  opsAPI: { listAlertEvents: listAlertEventsMock },
}))

describe('ObservabilityAlertSummary', () => {
  it('keeps only model-scoped alerts and emits canonical deep-link filters', async () => {
    listAlertEventsMock.mockResolvedValueOnce([
      {
        id: 1, rule_id: 9, severity: 'P1', status: 'firing',
        title: 'Account saturation', fired_at: '2026-11-11T10:00:00Z', email_sent: true,
        created_at: '2026-11-11T10:00:00Z',
        dimensions: { group_id: 7, route_alias: 'gpt-5', account_id: 101, sample_count: 6, data_at: '2026-11-11T09:59:00Z' },
      },
      {
        id: 2, rule_id: 10, severity: 'P2', status: 'firing',
        title: 'System CPU', fired_at: '2026-11-11T10:01:00Z', email_sent: false,
        created_at: '2026-11-11T10:01:00Z', dimensions: {},
      },
    ])
    const wrapper = mount(ObservabilityAlertSummary)
    await flushPromises()
    expect(wrapper.findAll('button')).toHaveLength(1)
    expect(wrapper.text()).toContain('admin.observability.groupRef')
    expect(wrapper.text()).toContain('admin.observability.samples')
    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('open')?.[0]).toEqual([{ groupId: 7, routeAlias: 'gpt-5', accountId: 101 }])
  })
})

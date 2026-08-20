import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { listSchedules, replaceSchedules } = vi.hoisted(() => ({
  listSchedules: vi.fn(),
  replaceSchedules: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      listModelRouteConcurrencySchedules: listSchedules,
      replaceModelRouteConcurrencySchedules: replaceSchedules
    }
  }
}))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (_key: string, fallback?: string) => fallback || _key }) }))

import ModelRouteConcurrencyScheduleEditor from '../ModelRouteConcurrencyScheduleEditor.vue'

describe('ModelRouteConcurrencyScheduleEditor', () => {
  beforeEach(() => {
    listSchedules.mockReset()
    replaceSchedules.mockReset()
    listSchedules.mockResolvedValue([{ id: 1, start: '09:30', end: '20:30', max_concurrency: 50 }])
    replaceSchedules.mockResolvedValue(undefined)
  })

  it('loads, adds, and saves numeric/unlimited windows', async () => {
    const wrapper = mount(ModelRouteConcurrencyScheduleEditor, {
      props: { groupId: 1, routeAlias: 'test-zlx', accountId: 2 }
    })
    await flushPromises()

    expect(wrapper.findAll('[data-test="schedule-row"]')).toHaveLength(1)
    await wrapper.find('[data-test="add-schedule"]').trigger('click')
    const starts = wrapper.findAll('[data-test="schedule-start"]')
    const ends = wrapper.findAll('[data-test="schedule-end"]')
    await starts[1].findAll('select')[0].setValue('20')
    await starts[1].findAll('select')[1].setValue('30')
    await ends[1].findAll('select')[0].setValue('24')
    await ends[1].findAll('select')[1].setValue('0')
    await wrapper.find('[data-test="save-schedules"]').trigger('click')
    await flushPromises()

    expect(replaceSchedules).toHaveBeenCalledWith(1, {
      route_alias: 'test-zlx',
      account_id: 2,
      schedules: [
        { id: 1, start: '09:30', end: '20:30', max_concurrency: 50 },
        { id: undefined, start: '20:30', end: '24:00', max_concurrency: null }
      ]
    })
    expect(wrapper.find('[data-test="schedule-saved"]').exists()).toBe(true)
    wrapper.unmount()
  })

  it('rejects overlap before calling the API', async () => {
    const wrapper = mount(ModelRouteConcurrencyScheduleEditor, {
      props: { groupId: 1, routeAlias: 'test-zlx', accountId: 2 }
    })
    await flushPromises()
    await wrapper.find('[data-test="add-schedule"]').trigger('click')
    await wrapper.find('[data-test="save-schedules"]').trigger('click')

    expect(replaceSchedules).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="schedule-error"]').text()).toContain('不能重叠')
    wrapper.unmount()
  })

  it('supports draft schedules before a group exists', async () => {
    const wrapper = mount(ModelRouteConcurrencyScheduleEditor, {
      props: { routeAlias: 'test-zlx', accountId: 2, modelValue: [] }
    })
    await wrapper.find('[data-test="add-schedule"]').trigger('click')
    const row = wrapper.find('[data-test="schedule-row"]')
    await row.find('[data-test="schedule-limit"]').setValue('10')
    await wrapper.find('[data-test="save-schedules"]').trigger('click')
    await flushPromises()

    expect(replaceSchedules).not.toHaveBeenCalled()
    expect(wrapper.find('[data-test="schedule-saved"]').exists()).toBe(true)
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toEqual([
      { start: '00:00', end: '24:00', maxConcurrency: 10 },
    ])
    wrapper.unmount()
  })
})

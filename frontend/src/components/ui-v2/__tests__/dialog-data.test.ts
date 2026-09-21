import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { UiDataTable, UiDialog, UiHorizontalRanking } from '..'

describe('UI V2 dialog and data components', () => {
  it('renders a centered dialog only when open and emits close', async () => {
    const wrapper = mount(UiDialog, { props: { modelValue: true, title: '详情' } })
    expect(document.body.querySelector('.ui-v2-dialog-backdrop')).not.toBeNull()
    const closeButton = document.body.querySelector<HTMLButtonElement>('.ui-v2-dialog__close')
    expect(closeButton).not.toBeNull()
    closeButton?.click()
    expect(wrapper.emitted('close')).toHaveLength(1)
    wrapper.unmount()
  })

  it('renders table slots and emits row click', async () => {
    const wrapper = mount(UiDataTable, {
      props: {
        columns: [{ key: 'name', label: '名称' }],
        rows: [{ id: 1, name: '模型 A' }]
      },
      slots: { 'cell-name': '<span class="custom-cell">{{ value }}</span>' }
    })
    expect(wrapper.find('.custom-cell').text()).toBe('模型 A')
    await wrapper.find('tbody tr').trigger('click')
    expect(wrapper.emitted('rowClick')?.[0]).toEqual([{ id: 1, name: '模型 A' }])
  })

  it('emits ranking selection without expanding the row', async () => {
    const wrapper = mount(UiHorizontalRanking, { props: { items: [{ id: 1, label: '场景 A', value: 100 }] } })
    await wrapper.find('.ui-v2-ranking__row').trigger('click')
    expect(wrapper.emitted('select')?.[0]?.[0]).toMatchObject({ label: '场景 A', value: 100 })
    expect(wrapper.find('.ui-v2-ranking__row').find('.ui-v2-table').exists()).toBe(false)
  })

  it('can hide the detail action for non-drilldown rankings', () => {
    const wrapper = mount(UiHorizontalRanking, { props: { showAction: false, items: [{ id: 1, label: '模型 A', value: 100 }] } })
    expect(wrapper.find('.ui-v2-ranking__action').exists()).toBe(false)
  })
})

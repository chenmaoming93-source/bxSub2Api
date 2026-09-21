import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import { UiButton, UiInput, UiSwitch } from '..'

describe('UI V2 primitives', () => {
  it('renders a primary button and exposes loading state', () => {
    const wrapper = mount(UiButton, { props: { loading: true } })
    expect(wrapper.classes()).toContain('ui-v2-button--primary')
    expect(wrapper.find('.ui-v2-spinner').exists()).toBe(true)
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
  })

  it('emits an updated input value', async () => {
    const wrapper = mount(UiInput, { props: { modelValue: 'old' } })
    await wrapper.find('input').setValue('new')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual(['new'])
  })

  it('toggles switch with a boolean payload', async () => {
    const wrapper = mount(UiSwitch, { props: { modelValue: false } })
    await wrapper.find('button').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]).toEqual([true])
  })
})

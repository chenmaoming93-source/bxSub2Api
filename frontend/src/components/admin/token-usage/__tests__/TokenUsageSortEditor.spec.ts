import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import TokenUsageSortEditor from '@/components/admin/token-usage/TokenUsageSortEditor.vue'

vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

const options = [
  { value: 'usage_date' as const, label: 'Date' },
  { value: 'model' as const, label: 'Model' },
  { value: 'used_tokens' as const, label: 'Used tokens' }
]

describe('TokenUsageSortEditor', () => {
  it('adds rules in priority order without reusing a field', async () => {
    const wrapper = mount(TokenUsageSortEditor, {
      props: { modelValue: [{ field: 'usage_date', order: 'desc' }], options }
    })
    await wrapper.get('button.text-primary-600').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toEqual([
      { field: 'usage_date', order: 'desc' },
      { field: 'model', order: 'asc' }
    ])
  })

  it('emits the independently selected direction for a rule', async () => {
    const wrapper = mount(TokenUsageSortEditor, {
      props: { modelValue: [{ field: 'usage_date', order: 'desc' }], options }
    })
    await wrapper.findAll('select')[1].setValue('asc')
    expect(wrapper.emitted('update:modelValue')?.[0]?.[0]).toEqual([{ field: 'usage_date', order: 'asc' }])
  })
})

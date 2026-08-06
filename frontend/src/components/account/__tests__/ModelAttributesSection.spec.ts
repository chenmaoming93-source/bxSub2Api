import { describe, expect, it, vi, beforeEach } from 'vitest'
import { defineComponent, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import ModelAttributesSection from '../ModelAttributesSection.vue'
import type { ModelAttributes } from '@/types'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, unknown>) =>
        params ? `${key}:${String(params.key ?? '')}` : key
    })
  }
})

async function mountSection(modelValue?: ModelAttributes | null) {
  const wrapper = mount(ModelAttributesSection, {
    props: { modelValue },
    global: {}
  })
  await flushPromises()
  return wrapper
}

async function setInputValue(wrapper: ReturnType<typeof mountSection>, selector: string, value: string) {
  const input = wrapper.get(selector)
  await input.setValue(value)
  await flushPromises()
}

describe('ModelAttributesSection', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows empty state when no attributes', async () => {
    const wrapper = await mountSection()
    expect(wrapper.text()).toContain('admin.accounts.modelAttributes.empty')
    expect(wrapper.findAll('[data-testid="attribute-row"]')).toHaveLength(0)
  })

  it('adds a blank custom row when clicking the add button', async () => {
    const wrapper = await mountSection()
    await wrapper.get('[data-testid="add-attribute-button"]').trigger('click')
    await flushPromises()

    const rows = wrapper.findAll('[data-testid="attribute-row"]')
    expect(rows).toHaveLength(1)
    expect((wrapper.get('[data-testid="row-key"]').element as HTMLInputElement).value).toBe('')
    expect(wrapper.get('[data-testid="row-description"]').element as HTMLInputElement).toHaveProperty('value', '')
  })

  it('adds a preset row with key and description pre-filled', async () => {
    const wrapper = await mountSection()
    const preset = wrapper.get('[data-testid="preset-chip"][data-preset-key="context_window"]')
    await preset.trigger('click')
    await flushPromises()

    const rows = wrapper.findAll('[data-testid="attribute-row"]')
    expect(rows).toHaveLength(1)
    expect(wrapper.get('[data-testid="row-key"]').element as HTMLInputElement).toHaveProperty('value', 'context_window')
    expect(wrapper.get('[data-testid="row-description"]').element as HTMLInputElement).toHaveProperty(
      'value',
      'admin.accounts.modelAttributes.presets.contextWindow'
    )
    // value 留空 → 提交 map 为空对象
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([{}])
  })

  it('marks already-added presets as disabled', async () => {
    const wrapper = await mountSection()
    await wrapper.get('[data-testid="preset-chip"][data-preset-key="context_window"]').trigger('click')
    await flushPromises()

    const preset = wrapper.get('[data-testid="preset-chip"][data-preset-key="context_window"]')
    expect(preset.attributes('disabled')).toBeDefined()
    // 再次点击不新增行
    await preset.trigger('click')
    await flushPromises()
    expect(wrapper.findAll('[data-testid="attribute-row"]')).toHaveLength(1)
  })

  it('emits parsed value map when value text is entered', async () => {
    const wrapper = await mountSection()
    await wrapper.get('[data-testid="preset-chip"][data-preset-key="context_window"]').trigger('click')
    await flushPromises()

    await setInputValue(wrapper, '[data-testid="row-value"]', '200000')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([
      { context_window: { description: 'admin.accounts.modelAttributes.presets.contextWindow', value: 200000 } }
    ])
  })

  it('parses boolean and array values', async () => {
    const wrapper = await mountSection()
    await wrapper.get('[data-testid="preset-chip"][data-preset-key="supports_vision"]').trigger('click')
    await flushPromises()
    await setInputValue(wrapper, '[data-testid="row-value"]', 'true')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({ supports_vision: { value: true } })

    await wrapper.get('[data-testid="row-remove"]').trigger('click')
    await flushPromises()
    await wrapper.get('[data-testid="preset-chip"][data-preset-key="modalities"]').trigger('click')
    await flushPromises()
    await setInputValue(wrapper, '[data-testid="row-value"]', '["text","image"]')
    expect(wrapper.emitted('update:modelValue')?.at(-1)?.[0]).toMatchObject({ modalities: { value: ['text', 'image'] } })
  })

  it('shows duplicate hint and keeps first occurrence in emitted map', async () => {
    const wrapper = await mountSection()
    // 行 A：key=dup, value=1（自定义行）
    await wrapper.get('[data-testid="add-attribute-button"]').trigger('click')
    await flushPromises()
    await setInputValue(wrapper, '[data-testid="row-key"]', 'dup')
    await setInputValue(wrapper, '[data-testid="row-value"]', '1')
    // 行 B：key=dup, value=2（自定义行）
    await wrapper.get('[data-testid="add-attribute-button"]').trigger('click')
    await flushPromises()
    const keys = wrapper.findAll('[data-testid="row-key"]')
    const values = wrapper.findAll('[data-testid="row-value"]')
    await keys[1].setValue('dup')
    await values[1].setValue('2')
    await flushPromises()

    expect(wrapper.findAll('[data-testid="duplicate-hint"]').length).toBeGreaterThan(0)
    // 第一份保留：value=1
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([{ dup: { value: 1 } }])
  })

  it('removes a row and emits updated map', async () => {
    const wrapper = await mountSection()
    await wrapper.get('[data-testid="preset-chip"][data-preset-key="context_window"]').trigger('click')
    await flushPromises()
    await setInputValue(wrapper, '[data-testid="row-value"]', '200000')
    await wrapper.get('[data-testid="row-remove"]').trigger('click')
    await flushPromises()

    expect(wrapper.findAll('[data-testid="attribute-row"]')).toHaveLength(0)
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([{}])
  })

  it('renders existing modelValue as rows (edit echo)', async () => {
    const wrapper = await mountSection({
      context_window: { description: '上下文窗口总大小（token）', value: 200000 },
      supports_vision: { description: '支持图片输入', value: true }
    })

    const rows = wrapper.findAll('[data-testid="attribute-row"]')
    expect(rows).toHaveLength(2)
    expect((wrapper.findAll('[data-testid="row-key"]')[0].element as HTMLInputElement).value).toBe('context_window')
    expect((wrapper.findAll('[data-testid="row-value"]')[0].element as HTMLInputElement).value).toBe('200000')
    expect((wrapper.findAll('[data-testid="row-value"]')[1].element as HTMLInputElement).value).toBe('true')
  })

  it('keeps the new row when a real parent v-model rewrites the prop (regression)', async () => {
    // 模拟真实弹窗中的 v-model 父组件：emit 会被回写到 modelValue，引用相同
    const Host = defineComponent({
      components: { ModelAttributesSection },
      template: '<ModelAttributesSection v-model="value" />',
      setup() {
        const value = ref<ModelAttributes>({})
        return { value }
      }
    })
    const wrapper = mount(Host)
    await flushPromises()

    await wrapper.get('[data-testid="add-attribute-button"]').trigger('click')
    await flushPromises()
    // 父组件 v-model 回写后，新行必须保留（不会被 watch 重建清掉）
    expect(wrapper.findAll('[data-testid="attribute-row"]')).toHaveLength(1)

    // 输入 key 与 value 后再验证：行仍保留
    await wrapper.get('[data-testid="row-key"]').setValue('custom_flag')
    await wrapper.get('[data-testid="row-value"]').setValue('abc')
    await flushPromises()
    expect(wrapper.findAll('[data-testid="attribute-row"]')).toHaveLength(1)
    expect((wrapper.get('[data-testid="row-key"]').element as HTMLInputElement).value).toBe('custom_flag')
  })
})

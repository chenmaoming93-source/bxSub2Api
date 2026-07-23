import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { getAvailableModels, list } = vi.hoisted(() => ({
  getAvailableModels: vi.fn(),
  list: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: { accounts: { getAvailableModels, list } }
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string, fallback?: string) => fallback || key })
}))

import GroupModelRoutingEditor from '../GroupModelRoutingEditor.vue'
import Select from '@/components/common/Select.vue'
import {
  addRoutingCandidate,
  createEmptyRoutingCandidate,
  intersectAccountModels,
  type RoutingEditorRule
} from '../groupModelRoutingEditor'

describe('GroupModelRoutingEditor', () => {
  beforeEach(() => {
    getAvailableModels.mockReset()
    list.mockReset()
  })

  it('creates candidates with the same defaults as GroupsView', () => {
    expect(createEmptyRoutingCandidate()).toEqual({
      model: '', accounts: [], priority: 0, daily_token_limit: null
    })
  })

  it('appends a candidate with the next priority', () => {
    const rule: RoutingEditorRule = {
      alias: 'coding',
      candidates: [
        { model: 'first', accounts: [{ id: 1, name: 'one' }], priority: 4, daily_token_limit: null },
        { model: 'second', accounts: [{ id: 2, name: 'two' }], priority: 9, daily_token_limit: 100 }
      ]
    }
    addRoutingCandidate(rule)
    expect(rule.candidates.at(-1)).toEqual({
      model: '', accounts: [], priority: 10, daily_token_limit: null
    })
  })

  it('deduplicates and sorts a single account model list without mutating it', () => {
    const input = [
      { id: 'z-model' },
      { id: 'a-model' },
      { id: 'z-model', display_name: 'Z Model' }
    ]
    const snapshot = structuredClone(input)

    expect(intersectAccountModels([input])).toEqual([
      { id: 'a-model' },
      { id: 'z-model', display_name: 'Z Model' }
    ])
    expect(input).toEqual(snapshot)
  })

  it('returns only model ids shared by every account', () => {
    expect(intersectAccountModels([
      [{ id: 'shared' }, { id: 'first-only' }],
      [{ id: 'second-only' }, { id: 'shared', display_name: 'Shared Model' }],
      [{ id: 'shared' }]
    ])).toEqual([{ id: 'shared', display_name: 'Shared Model' }])
  })

  it('returns an empty result for no accounts, an empty list, or no intersection', () => {
    expect(intersectAccountModels([])).toEqual([])
    expect(intersectAccountModels([[{ id: 'model' }], []])).toEqual([])
    expect(intersectAccountModels([[{ id: 'one' }], [{ id: 'two' }]])).toEqual([])
  })

  it('loads a selected account models into a disabled-until-account selector', async () => {
    getAvailableModels.mockResolvedValue([
      { id: 'model-a', display_name: 'Model A' },
      { id: 'model-b' }
    ])
    const rules: RoutingEditorRule[] = [{
      alias: 'coding',
      candidates: [{
        model: '',
        accounts: [{ id: 7, name: 'account-seven' }],
        priority: 0,
        daily_token_limit: null
      }]
    }]
    const wrapper = mount(GroupModelRoutingEditor, {
      props: { enabled: true, rules, platform: 'anthropic' },
      global: { stubs: { Icon: true } }
    })
    await flushPromises()

    const selector = wrapper.findComponent(Select)
    expect(getAvailableModels).toHaveBeenCalledWith(7, { signal: expect.any(AbortSignal) })
    expect(selector.props('disabled')).toBe(false)
    expect(selector.props('options')).toEqual([
      { value: 'model-a', label: 'Model A' },
      { value: 'model-b', label: 'model-b' }
    ])

    selector.vm.$emit('update:modelValue', 'model-a')
    await wrapper.vm.$nextTick()
    expect(rules[0].candidates[0]).toMatchObject({
      model: 'model-a',
      accounts: [{ id: 7, name: 'account-seven' }]
    })
    wrapper.unmount()

    const emptyWrapper = mount(GroupModelRoutingEditor, {
      props: {
        enabled: true,
        rules: [{ alias: 'empty', candidates: [createEmptyRoutingCandidate()] }],
        platform: 'anthropic'
      },
      global: { stubs: { Icon: true } }
    })
    expect(emptyWrapper.findComponent(Select).props('disabled')).toBe(true)
    emptyWrapper.unmount()
  })

  it('loads accounts in parallel, intersects models, caches successes, and recomputes after removal', async () => {
    getAvailableModels.mockImplementation(async (id: number) => {
      const models: Record<number, Array<{ id: string }>> = {
        1: [{ id: 'shared' }, { id: 'one-only' }],
        2: [{ id: 'shared' }, { id: 'two-only' }]
      }
      return models[id]
    })
    const rules: RoutingEditorRule[] = [{
      alias: 'multi',
      candidates: [
        { model: 'shared', accounts: [{ id: 1, name: 'one' }, { id: 2, name: 'two' }], priority: 0, daily_token_limit: null },
        { model: '', accounts: [{ id: 1, name: 'one' }], priority: 1, daily_token_limit: null }
      ]
    }]
    const wrapper = mount(GroupModelRoutingEditor, {
      props: { enabled: true, rules, platform: 'anthropic' },
      global: { stubs: { Icon: true } }
    })
    await flushPromises()

    expect(getAvailableModels).toHaveBeenCalledTimes(2)
    expect(wrapper.findAllComponents(Select)[0].props('options')).toEqual([{ value: 'shared', label: 'shared' }])

    await wrapper.get('[data-test="remove-account"][data-account-id="2"]').trigger('click')
    await flushPromises()
    expect(getAvailableModels).toHaveBeenCalledTimes(2)
    expect(wrapper.findAllComponents(Select)[0].props('options')).toEqual([
      { value: 'one-only', label: 'one-only' },
      { value: 'shared', label: 'shared' }
    ])
    expect(rules[0].candidates[0].model).toBe('shared')
    wrapper.unmount()
  })

  it('disables the selector after a load failure and retries without caching the failure', async () => {
    getAvailableModels
      .mockRejectedValueOnce(new Error('upstream unavailable'))
      .mockResolvedValueOnce([{ id: 'recovered' }])
    const rules: RoutingEditorRule[] = [{
      alias: 'retry',
      candidates: [{ model: '', accounts: [{ id: 9, name: 'nine' }], priority: 0, daily_token_limit: null }]
    }]
    const wrapper = mount(GroupModelRoutingEditor, {
      props: { enabled: true, rules, platform: 'anthropic' },
      global: { stubs: { Icon: true } }
    })
    await flushPromises()

    expect(wrapper.get('[data-test="model-error"]').exists()).toBe(true)
    expect(wrapper.findComponent(Select).props('disabled')).toBe(true)
    await wrapper.get('[data-test="model-error"] button').trigger('click')
    await flushPromises()
    expect(getAvailableModels).toHaveBeenCalledTimes(2)
    expect(wrapper.findComponent(Select).props('disabled')).toBe(false)
    expect(wrapper.find('[data-test="model-error"]').exists()).toBe(false)
    wrapper.unmount()
  })

  it('ignores an older multi-account result after the account selection changes', async () => {
    let resolveFirst!: (models: Array<{ id: string }>) => void
    const firstRequest = new Promise<Array<{ id: string }>>(resolve => { resolveFirst = resolve })
    getAvailableModels.mockImplementation((id: number) => id === 1
      ? firstRequest
      : Promise.resolve([{ id: 'shared' }]))
    const rules: RoutingEditorRule[] = [{
      alias: 'race',
      candidates: [{
        model: '',
        accounts: [{ id: 1, name: 'one' }, { id: 2, name: 'two' }],
        priority: 0,
        daily_token_limit: null
      }]
    }]
    const wrapper = mount(GroupModelRoutingEditor, {
      props: { enabled: true, rules, platform: 'anthropic' },
      global: { stubs: { Icon: true } }
    })

    await wrapper.get('[data-test="remove-account"][data-account-id="2"]').trigger('click')
    resolveFirst([{ id: 'one-only' }])
    await flushPromises()

    expect(wrapper.findComponent(Select).props('options')).toEqual([{ value: 'one-only', label: 'one-only' }])
    wrapper.unmount()
  })

  it('preserves a valid historical model and requires reselection for an invalid one', async () => {
    getAvailableModels.mockResolvedValue([{ id: 'still-valid' }])
    const rules: RoutingEditorRule[] = [{
      alias: 'history',
      candidates: [
        { model: 'still-valid', accounts: [{ id: 1, name: 'one' }], priority: 0, daily_token_limit: null },
        { model: 'removed-model', accounts: [{ id: 1, name: 'one' }], priority: 1, daily_token_limit: null }
      ]
    }]
    const wrapper = mount(GroupModelRoutingEditor, {
      props: { enabled: true, rules, platform: 'anthropic' },
      global: { stubs: { Icon: true } }
    })
    await flushPromises()

    expect(rules[0].candidates[0].model).toBe('still-valid')
    expect(rules[0].candidates[1].model).toBe('')
    expect(wrapper.findAll('[data-test="model-invalid"]')).toHaveLength(1)
    expect((wrapper.vm as unknown as { isValid: () => boolean }).isValid()).toBe(false)

    wrapper.findAllComponents(Select)[1].vm.$emit('update:modelValue', 'still-valid')
    await wrapper.vm.$nextTick()
    expect((wrapper.vm as unknown as { isValid: () => boolean }).isValid()).toBe(true)
    wrapper.unmount()
  })
})

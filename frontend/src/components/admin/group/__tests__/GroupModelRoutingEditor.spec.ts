import { mount, flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'

const { list } = vi.hoisted(() => ({ list: vi.fn() }))

vi.mock('@/api/admin', () => ({ adminAPI: { accounts: { list } } }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (_key: string, fallback?: string) => fallback || _key }) }))

import GroupModelRoutingEditor from '../GroupModelRoutingEditor.vue'
import { addRoutingCandidate, createEmptyRoutingCandidate, type RoutingEditorRule } from '../groupModelRoutingEditor'

describe('GroupModelRoutingEditor', () => {
  beforeEach(() => list.mockReset())

  it('creates account-only candidates and increments priority', () => {
    expect(createEmptyRoutingCandidate()).toEqual({ accounts: [], priority: 0, maxConcurrency: null })
    const rule: RoutingEditorRule = {
      alias: 'coding',
      candidates: [{ accounts: [{ id: 1, name: 'one' }], priority: 4 }]
    }
    addRoutingCandidate(rule)
    expect(rule.candidates.at(-1)).toEqual({ accounts: [], priority: 5, maxConcurrency: null })
  })

  it('renders only alias, accounts and priority and validates without loading models', () => {
    const rules: RoutingEditorRule[] = [{
      alias: 'coding',
      candidates: [{ accounts: [{ id: 7, name: 'seven' }], priority: 0 }]
    }]
    const wrapper = mount(GroupModelRoutingEditor, {
      props: { enabled: true, rules, platform: 'anthropic' },
      global: { stubs: { Icon: true } }
    })

    expect(wrapper.find('[data-test="candidate-model"]').exists()).toBe(false)
    expect(wrapper.text()).not.toContain('Upstream model')
    expect((wrapper.vm as unknown as { isValid: () => boolean }).isValid()).toBe(true)
    expect(list).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('shows the account upstream model extracted from credentials.model_mapping', () => {
    const rules: RoutingEditorRule[] = [{
      alias: 'coding',
      candidates: [{
        accounts: [{ id: 7, name: 'seven', upstreamModel: 'gpt-5.3-codex' }],
        priority: 0
      }]
    }]
    const wrapper = mount(GroupModelRoutingEditor, {
      props: { enabled: true, rules, platform: 'anthropic' },
      global: { stubs: { Icon: true } }
    })

    expect(wrapper.text()).toContain('gpt-5.3-codex')
    wrapper.unmount()
  })

  it('maps account search results with upstreamModel from credentials', async () => {
    list.mockResolvedValue({
      items: [
        { id: 1, name: 'one', credentials: { model_mapping: { 'zeta-model': 'upstream' } } },
        { id: 2, name: 'two', credentials: { model_mapping: { 'alpha-model': 'upstream', 'beta-model': 'upstream' } } },
        { id: 3, name: 'three', credentials: {} },
      ],
      total: 3,
    })
    const wrapper = mount(GroupModelRoutingEditor, {
      props: { enabled: true, rules: [{ alias: 'coding', candidates: [createEmptyRoutingCandidate()] }], platform: 'anthropic' },
      global: { stubs: { Icon: true } }
    })
    const input = wrapper.find('.account-search-container input')
    await input.setValue('one')
    await input.trigger('input')
    await vi.waitFor(() => {
      const results = (wrapper.vm as unknown as { results: Record<string, any[]> }).results
      const key = Object.keys(results)[0]
      if (!results[key]) throw new Error('search result not populated yet')
      expect(results[key]).toHaveLength(3)
      expect(results[key][0].upstreamModel).toBe('zeta-model')
      expect(results[key][1].upstreamModel).toBe('alpha-model')
      expect(results[key][2].upstreamModel).toBeUndefined()
    })
    wrapper.unmount()
  })

  it('still requires an account and non-negative integer priority', () => {
    const wrapper = mount(GroupModelRoutingEditor, {
      props: { enabled: true, rules: [{ alias: 'coding', candidates: [createEmptyRoutingCandidate()] }], platform: 'anthropic' },
      global: { stubs: { Icon: true } }
    })
    expect((wrapper.vm as unknown as { isValid: () => boolean }).isValid()).toBe(false)
    wrapper.unmount()
  })
})

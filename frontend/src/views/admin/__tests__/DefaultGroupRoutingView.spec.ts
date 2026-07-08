import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DefaultGroupRoutingView from '../DefaultGroupRoutingView.vue'

const { getDefaultGroup, getGroup, createGroup, updateGroup, getAccount, showSuccess, showError } = vi.hoisted(() => ({
  getDefaultGroup: vi.fn(), getGroup: vi.fn(), createGroup: vi.fn(), updateGroup: vi.fn(), getAccount: vi.fn(), showSuccess: vi.fn(), showError: vi.fn()
}))
vi.mock('@/api/admin/settings', () => ({ settingsAPI: { getDefaultGroup } }))
vi.mock('@/api/admin', () => ({ adminAPI: { groups: { getById: getGroup, create: createGroup, update: updateGroup }, accounts: { getById: getAccount } } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))
vi.mock('@/components/layout/AppLayout.vue', () => ({ default: { template: '<div><slot /></div>' } }))
vi.mock('@/components/admin/group/GroupModelRoutingEditor.vue', () => ({ default: { template: '<div data-test="routing-editor" />' } }))

const mountPage = () => mount(DefaultGroupRoutingView, { global: { stubs: { RouterLink: { template: '<a><slot /></a>' } } } })

describe('DefaultGroupRoutingView', () => {
  beforeEach(() => { getDefaultGroup.mockReset(); getGroup.mockReset(); createGroup.mockReset(); updateGroup.mockReset(); getAccount.mockReset(); showError.mockReset() })

  it('renders the settings guide when unconfigured', async () => {
    getDefaultGroup.mockResolvedValue({ configured: false, name: '', exists: false, group: null })
    const wrapper = mountPage(); await flushPromises()
    expect(wrapper.text()).toContain('admin.defaultGroupRouting.unconfigured')
  })

  it('renders the missing state', async () => {
    getDefaultGroup.mockResolvedValue({ configured: true, name: 'future', exists: false, group: null })
    const wrapper = mountPage(); await flushPromises()
    expect(wrapper.text()).toContain('admin.defaultGroupRouting.missing')
    expect((wrapper.get('[data-test="locked-default-group-name"]').element as HTMLInputElement).disabled).toBe(true)
  })

  it('creates the locked name and enters edit state', async () => {
    getDefaultGroup.mockResolvedValueOnce({ configured: true, name: 'default', exists: false, group: null }).mockResolvedValueOnce({ configured: true, name: 'default', exists: true, group: { id: 9 } })
    createGroup.mockResolvedValue({ id: 9 })
    getGroup.mockResolvedValue({ id: 9, name: 'default', platform: 'openai', model_routing_enabled: false, model_routing: null })
    const wrapper = mountPage(); await flushPromises()
    await wrapper.get('button').trigger('click'); await flushPromises()
    expect(createGroup).toHaveBeenCalledWith(expect.objectContaining({ name: 'default' }))
    expect(wrapper.find('[data-test="routing-editor"]').exists()).toBe(true)
  })

  it('refreshes an existing group after a concurrent name conflict', async () => {
    getDefaultGroup.mockResolvedValueOnce({ configured: true, name: 'default', exists: false, group: null }).mockResolvedValueOnce({ configured: true, name: 'default', exists: true, group: { id: 10 } })
    createGroup.mockRejectedValue(new Error('duplicate'))
    getGroup.mockResolvedValue({ id: 10, name: 'default', platform: 'openai', model_routing_enabled: false, model_routing: null })
    const wrapper = mountPage(); await flushPromises()
    await wrapper.get('button').trigger('click'); await flushPromises()
    expect(createGroup).toHaveBeenCalledTimes(1)
    expect(wrapper.find('[data-test="routing-editor"]').exists()).toBe(true)
    expect(showError).not.toHaveBeenCalled()
  })

  it('loads and saves only routing fields for an existing group', async () => {
    getDefaultGroup.mockResolvedValue({ configured: true, name: 'default', exists: true, group: { id: 7 } })
    getGroup.mockResolvedValue({ id: 7, name: 'default', platform: 'openai', model_routing_enabled: true, model_routing: null })
    updateGroup.mockResolvedValue({ id: 7, name: 'default', platform: 'openai', model_routing_enabled: true, model_routing: null })
    const wrapper = mountPage(); await flushPromises()
    expect(wrapper.find('[data-test="routing-editor"]').exists()).toBe(true)
    await wrapper.get('button').trigger('click'); await flushPromises()
    expect(updateGroup).toHaveBeenCalledWith(7, { model_routing_enabled: true, model_routing: null })
  })
})

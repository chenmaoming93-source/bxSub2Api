import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DefaultGroupSettingsCard from '../DefaultGroupSettingsCard.vue'

const { getDefaultGroup, updateDefaultGroup, showSuccess, showError } = vi.hoisted(() => ({
  getDefaultGroup: vi.fn(), updateDefaultGroup: vi.fn(), showSuccess: vi.fn(), showError: vi.fn()
}))
vi.mock('@/api/admin/settings', () => ({ settingsAPI: { getDefaultGroup, updateDefaultGroup } }))
vi.mock('@/stores', () => ({ useAppStore: () => ({ showSuccess, showError }) }))
vi.mock('vue-i18n', () => ({ useI18n: () => ({ t: (key: string) => key }) }))

describe('DefaultGroupSettingsCard', () => {
  beforeEach(() => {
    getDefaultGroup.mockReset().mockResolvedValue({ configured: true, name: 'missing-group', exists: false, group: null })
    updateDefaultGroup.mockReset()
  })

  it('shows missing state and saves a trimmed name', async () => {
    updateDefaultGroup.mockResolvedValue({ configured: true, name: 'future-group', exists: false, group: null })
    const wrapper = mount(DefaultGroupSettingsCard)
    await flushPromises()
    expect(wrapper.text()).toContain('admin.settings.defaultGroup.missing')
    await wrapper.get('input').setValue('  future-group  ')
    await wrapper.get('button').trigger('click')
    await flushPromises()
    expect(updateDefaultGroup).toHaveBeenCalledWith('future-group')
    expect(wrapper.text()).toContain('admin.settings.defaultGroup.missing')
  })
})

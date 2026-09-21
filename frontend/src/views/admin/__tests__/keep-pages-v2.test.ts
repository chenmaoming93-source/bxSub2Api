import { describe, expect, it } from 'vitest'
import { shallowMount } from '@vue/test-utils'
import UiLegacyPageFrame from '@/components/ui-v2/layout/UiLegacyPageFrame.vue'
import AccountsV2View from '../AccountsV2View.vue'
import GroupsV2View from '../GroupsV2View.vue'
import RolesV2View from '../RolesV2View.vue'
import SettingsV2View from '../SettingsV2View.vue'
import TokenStatisticsV2View from '../TokenStatisticsV2View.vue'

const wrapperCases = [
  [AccountsV2View, 'accounts-view-stub'],
  [GroupsV2View, 'groups-view-stub'],
  [RolesV2View, 'roles-view-stub'],
  [SettingsV2View, 'settings-view-stub'],
  [TokenStatisticsV2View, 'token-statistics-view-stub']
] as const

describe('MVP-017 retained admin page wrappers', () => {
  it.each(wrapperCases)('embeds the retained business view without a nested layout', (component, legacyTag) => {
    const wrapper = shallowMount(component, {
      global: { stubs: { UiLegacyPageFrame: { template: '<section><slot /></section>' } } }
    })
    const legacy = wrapper.find(legacyTag)
    expect(legacy.exists()).toBe(true)
    expect(legacy.attributes('embedded')).toBe('true')
    expect(wrapper.find('app-layout').exists()).toBe(false)
  })

  it('owns the single AppLayout and applies the shared UI V2 page header', () => {
    const wrapper = shallowMount(UiLegacyPageFrame, {
      props: { title: '账号管理', description: '说明' },
      slots: { default: '<div class="business-content" />' },
      global: { stubs: {
        AppLayout: { name: 'AppLayout', template: '<main><slot /></main>' },
        UiPageContainer: { template: '<div><slot /></div>' },
        UiPageHeader: { props: ['title', 'description'], template: '<header class="ui-v2-page-header"><h1>{{ title }}</h1><p>{{ description }}</p><slot /></header>' }
      } }
    })
    expect(wrapper.findAllComponents({ name: 'AppLayout' })).toHaveLength(1)
    expect(wrapper.find('.ui-v2-page-header').text()).toContain('账号管理')
    expect(wrapper.find('.business-content').exists()).toBe(true)
  })
})

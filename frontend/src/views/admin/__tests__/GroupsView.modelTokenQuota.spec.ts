import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import GlobalModelTokenQuotaModal from '@/components/admin/group/GlobalModelTokenQuotaModal.vue'

const { getGlobal, updateGlobal } = vi.hoisted(() => ({
  getGlobal: vi.fn(),
  updateGlobal: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    modelTokenQuotas: { getGlobal, updateGlobal }
  }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) =>
        key.startsWith('admin.groups.modelTokenQuota.') ? `translated-${key.split('.').at(-1)}` : key
    })
  }
})

describe('global model token quota modal', () => {
  beforeEach(() => {
    getGlobal.mockReset()
    updateGlobal.mockReset()
    getGlobal.mockResolvedValue({
      quotas: [{ model: 'gpt-5', usage_date: '2026-06-30', used_tokens: 11, daily_limit_tokens: 100 }]
    })
  })

  it('loads global quotas, rejects invalid rows, and refreshes saved usage', async () => {
    updateGlobal.mockResolvedValue({
      model: 'gpt-5',
      usage_date: '2026-06-30',
      used_tokens: 11,
      daily_limit_tokens: 250
    })
    const wrapper = mount(GlobalModelTokenQuotaModal, {
      props: { show: true },
      global: {
        stubs: {
          BaseDialog: { props: ['show'], template: '<div v-if="show"><slot /><slot name="footer" /></div>' },
          Icon: true
        }
      }
    })
    await flushPromises()

    expect(getGlobal).toHaveBeenCalledOnce()
    expect(wrapper.text()).toContain('11')
    await wrapper.get('[data-test="global-model-quota-add"]').trigger('click')
    await wrapper.get('[data-test="global-model-quota-save"]').trigger('click')
    expect(updateGlobal).not.toHaveBeenCalled()
    expect(wrapper.get('[data-test="global-model-quota-error"]').exists()).toBe(true)

    const models = wrapper.findAll('[data-test="global-model-quota-model"]')
    const limits = wrapper.findAll('[data-test="global-model-quota-limit"]')
    await models[1].setValue('claude-sonnet')
    await limits[0].setValue('250')
    await limits[1].setValue('0')
    updateGlobal.mockImplementation((item: { model: string; daily_limit_tokens: number | null }) =>
      Promise.resolve({ model: item.model, usage_date: '2026-06-30', used_tokens: item.model === 'gpt-5' ? 11 : 0, daily_limit_tokens: item.daily_limit_tokens })
    )
    await wrapper.get('[data-test="global-model-quota-save"]').trigger('click')
    await flushPromises()

    expect(updateGlobal).toHaveBeenCalledWith({ model: 'gpt-5', daily_limit_tokens: 250 })
    expect(updateGlobal).toHaveBeenCalledWith({ model: 'claude-sonnet', daily_limit_tokens: 0 })
    expect(wrapper.text()).toContain('11')
    expect(wrapper.text()).not.toContain('admin.groups.modelTokenQuota.')
  })
})

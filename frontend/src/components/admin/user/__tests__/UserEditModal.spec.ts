import { describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import UserEditModal from '../UserEditModal.vue'
import type { AdminUser } from '@/types'

const { update } = vi.hoisted(() => ({ update: vi.fn() }))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: { update },
    userAttributes: { updateUserAttributeValues: vi.fn() }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({ showError: vi.fn(), showSuccess: vi.fn() })
}))

vi.mock('@/composables/useClipboard', () => ({
  useClipboard: () => ({ copyToClipboard: vi.fn() })
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

const user: AdminUser = {
  id: 7,
  email: 'user@example.com',
  username: 'user',
  department: '研发部',
  notes: '',
  role: 'user',
  balance: 0,
  concurrency: 1,
  status: 'active',
  allowed_groups: [],
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-01-01T00:00:00Z',
  updated_at: '2026-01-01T00:00:00Z'
}

describe('UserEditModal department', () => {
  it('loads and submits the editable department', async () => {
    update.mockReset()
    update.mockResolvedValue(user)
    const wrapper = mount(UserEditModal, {
      props: { show: true, user },
      global: {
        stubs: {
          BaseDialog: { template: '<div><slot /><slot name="footer" /></div>' },
          UserAttributeForm: true,
          Icon: true
        }
      }
    })

    const inputs = wrapper.findAll('form input')
    expect((inputs[3].element as HTMLInputElement).value).toBe('研发部')
    expect((inputs[0].element as HTMLInputElement).type).toBe('text')
    await inputs[0].setValue('ldap-user-001')
    await inputs[3].setValue('华北研发部')
    await wrapper.find('form').trigger('submit')
    await flushPromises()

    expect(update).toHaveBeenCalledWith(7, expect.objectContaining({ email: 'ldap-user-001', department: '华北研发部' }))
  })
})

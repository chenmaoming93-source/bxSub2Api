import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import type { AdminUser } from '@/types'
import UsersView from '../UsersView.vue'
import UserModelTokenQuotaModal from '@/components/admin/user/UserModelTokenQuotaModal.vue'

const {
  listUsers,
  getAllGroups,
  getBatchUsersUsage,
  listEnabledDefinitions,
  getBatchUserAttributes,
  getUserModelTokenQuotas,
  updateUserModelTokenQuotas
} = vi.hoisted(() => ({
  listUsers: vi.fn(),
  getAllGroups: vi.fn(),
  getBatchUsersUsage: vi.fn(),
  listEnabledDefinitions: vi.fn(),
  getBatchUserAttributes: vi.fn(),
  getUserModelTokenQuotas: vi.fn(),
  updateUserModelTokenQuotas: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    users: {
      list: listUsers,
      toggleStatus: vi.fn(),
      delete: vi.fn()
    },
    groups: {
      getAll: getAllGroups
    },
    dashboard: {
      getBatchUsersUsage
    },
    userAttributes: {
      listEnabledDefinitions,
      getBatchUserAttributes
    },
    modelTokenQuotas: {
      getUser: getUserModelTokenQuotas,
      updateUser: updateUserModelTokenQuotas
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) =>
        key.startsWith('admin.users.modelTokenQuota.') ? `translated-${key.split('.').at(-1)}` : key
    })
  }
})

const createAdminUser = (): AdminUser => ({
  id: 42,
  username: 'scoped-user',
  email: 'scoped@example.com',
  role: 'user',
  balance: 0,
  concurrency: 1,
  status: 'active',
  allowed_groups: [],
  balance_notify_enabled: false,
  balance_notify_threshold: null,
  balance_notify_extra_emails: [],
  created_at: '2026-04-17T00:00:00Z',
  updated_at: '2026-04-17T00:00:00Z',
  notes: '',
  last_active_at: '2026-04-16T02:00:00Z',
  last_used_at: '2026-04-17T02:00:00Z',
  current_concurrency: 0
})

const DataTableStub = {
  props: ['columns', 'data'],
  emits: ['sort'],
  template: `
    <div>
      <div data-test="columns">{{ columns.map(col => col.key).join(',') }}</div>
      <button data-test="sort-last-used" @click="$emit('sort', 'last_used_at', 'desc')">sort</button>
      <div v-for="row in data" :key="row.id">
        <slot name="cell-last_used_at" :value="row.last_used_at" :row="row" />
      </div>
    </div>
  `
}

describe('admin UsersView', () => {
  beforeEach(() => {
    localStorage.clear()

    listUsers.mockReset()
    getAllGroups.mockReset()
    getBatchUsersUsage.mockReset()
    listEnabledDefinitions.mockReset()
    getBatchUserAttributes.mockReset()
    getUserModelTokenQuotas.mockReset()
    updateUserModelTokenQuotas.mockReset()

    listUsers.mockResolvedValue({
      items: [createAdminUser()],
      total: 1,
      page: 1,
      page_size: 20,
      pages: 1
    })
    getAllGroups.mockResolvedValue([])
    getBatchUsersUsage.mockResolvedValue({ stats: {} })
    listEnabledDefinitions.mockResolvedValue([])
    getBatchUserAttributes.mockResolvedValue({ values: {} })
    getUserModelTokenQuotas.mockResolvedValue({ quotas: [] })
  })

  it('shows active, used, and created activity columns in order and requests last_used_at sort', async () => {
    const wrapper = mount(UsersView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          TablePageLayout: {
            template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
          },
          DataTable: DataTableStub,
          Pagination: true,
          ConfirmDialog: true,
          EmptyState: true,
          GroupBadge: true,
          Select: true,
          UserAttributesConfigModal: true,
          UserConcurrencyCell: true,
          UserCreateModal: true,
          UserEditModal: true,
          UserModelTokenQuotaModal: true,
          UserApiKeysModal: true,
          UserAllowedGroupsModal: true,
          UserBalanceModal: true,
          UserBalanceHistoryModal: true,
          GroupReplaceModal: true,
          Icon: true,
          Teleport: true
        }
      }
    })

    await flushPromises()

    const columns = wrapper.get('[data-test="columns"]').text()
    const visibleColumns = columns.split(',')
    expect(visibleColumns.slice(-4, -1)).toEqual(['last_active_at', 'last_used_at', 'created_at'])
    expect(visibleColumns).not.toContain('last_login_at')

    await wrapper.get('[data-test="sort-last-used"]').trigger('click')
    await flushPromises()

    expect(listUsers).toHaveBeenLastCalledWith(
      1,
      20,
      expect.objectContaining({
        sort_by: 'last_used_at',
        sort_order: 'desc'
      }),
      expect.any(Object)
    )
  })

  it('loads, validates, and saves model token quotas for only the selected user', async () => {
    getUserModelTokenQuotas.mockResolvedValue({
      quotas: [
        {
          user_id: 42,
          model: 'gpt-5',
          usage_date: '2026-06-30',
          used_tokens: 7,
          daily_limit_tokens: 100
        }
      ]
    })
    updateUserModelTokenQuotas.mockResolvedValue({
      quotas: [
        {
          user_id: 42,
          model: 'gpt-5',
          usage_date: '2026-06-30',
          used_tokens: 9,
          daily_limit_tokens: 200
        }
      ]
    })
    const wrapper = mount(UserModelTokenQuotaModal, {
      props: { show: true, user: createAdminUser() },
      global: {
        stubs: {
          BaseDialog: {
            props: ['show'],
            template: '<div v-if="show"><slot /><slot name="footer" /></div>'
          },
          Icon: true
        }
      }
    })
    await flushPromises()

    expect(getUserModelTokenQuotas).toHaveBeenCalledWith(42)
    await wrapper.get('[data-test="model-quota-model"]').setValue('')
    await wrapper.get('[data-test="model-quota-save"]').trigger('click')
    expect(updateUserModelTokenQuotas).not.toHaveBeenCalled()
    expect(wrapper.get('[data-test="model-quota-error"]').exists()).toBe(true)

    await wrapper.get('[data-test="model-quota-model"]').setValue('gpt-5')
    await wrapper.get('[data-test="model-quota-limit"]').setValue('200')
    await wrapper.get('[data-test="model-quota-save"]').trigger('click')
    await flushPromises()

    expect(updateUserModelTokenQuotas).toHaveBeenCalledWith(42, [
      { model: 'gpt-5', daily_limit_tokens: 200 }
    ])
    expect(wrapper.text()).toContain('9')
    expect(wrapper.text()).not.toContain('admin.users.modelTokenQuota.')
    expect(listUsers).not.toHaveBeenCalled()
  })
})

import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import DepartmentUserTokenDialog from '../DepartmentUserTokenDialog.vue'
import { adminAPI } from '@/api/admin'

afterEach(() => { vi.restoreAllMocks(); document.body.innerHTML = '' })

describe('DepartmentUserTokenDialog', () => {
  it('loads department users into a centered modal table', async () => {
    vi.spyOn(adminAPI.usage, 'queryDepartmentUsers').mockResolvedValue({
      department: '研发', department_total_tokens: 300, total: 1, page: 1, page_size: 100, complete: true, consistency: 'ok',
      rows: [{ user_id: 3, email: 'dev@example.com', username: '研发用户', total_tokens: 300, percentage: 100 }]
    })
    mount(DepartmentUserTokenDialog, { props: { modelValue: true, department: '研发', startDate: '2026-01-01', endDate: '2026-01-02' } })
    await flushPromises()
    expect(document.body.querySelector('.ui-v2-dialog-backdrop')).not.toBeNull()
    expect(document.body.textContent).toContain('研发用户')
    expect(adminAPI.usage.queryDepartmentUsers).toHaveBeenCalledWith({ department: '研发', start_date: '2026-01-01', end_date: '2026-01-02', page: 1, page_size: 100 })
  })
})

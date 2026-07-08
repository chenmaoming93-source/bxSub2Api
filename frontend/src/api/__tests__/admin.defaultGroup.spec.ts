import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get, put } = vi.hoisted(() => ({ get: vi.fn(), put: vi.fn() }))
vi.mock('@/api/client', () => ({ apiClient: { get, put } }))

import { getDefaultGroup, updateDefaultGroup } from '@/api/admin/settings'

describe('admin default group api', () => {
  beforeEach(() => { get.mockReset(); put.mockReset() })

  it('loads status from the dedicated endpoint', async () => {
    const status = { configured: true, name: 'default', exists: false, group: null }
    get.mockResolvedValue({ data: status })
    await expect(getDefaultGroup()).resolves.toEqual(status)
    expect(get).toHaveBeenCalledWith('/admin/default-group')
  })

  it('updates the name and returns existence status', async () => {
    const status = { configured: true, name: 'future', exists: false, group: null }
    put.mockResolvedValue({ data: status })
    await expect(updateDefaultGroup('future')).resolves.toEqual(status)
    expect(put).toHaveBeenCalledWith('/admin/settings/default-group', { name: 'future' })
  })
})

import { afterEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import SceneAccountTokenDialog from '../SceneAccountTokenDialog.vue'
import { adminAPI } from '@/api/admin'

afterEach(() => { vi.restoreAllMocks(); document.body.innerHTML = '' })

describe('SceneAccountTokenDialog', () => {
  it('loads existing scene daily data into a centered detail dialog', async () => {
    vi.spyOn(adminAPI.usage, 'querySceneAccountDaily').mockResolvedValue({
      timezone: 'UTC', start_date: '2026-01-01', end_date: '2026-01-02', complete: true, consistency: 'ok', projection_id: 1,
      days: [{ date: '2026-01-01', scenes: [{ group_id: 7, group_name: '研发', scene_name: '研发', total_tokens: 120, accounts: [{ account_id: 9, account_name: '账号 A', upstream_model: 'model-a', total_tokens: 120 }] }] }]
    })
    mount(SceneAccountTokenDialog, {
      props: { modelValue: true, scene: { id: 7, label: '研发', requests: 1, totalTokens: 120, cost: 0, actualCost: 0, accountCost: 0, percentage: 100 }, startDate: '2026-01-01', endDate: '2026-01-02' }
    })
    await flushPromises()
    expect(document.body.querySelector('.ui-v2-dialog-backdrop')).not.toBeNull()
    expect(document.body.textContent).toContain('账号 A')
    expect(adminAPI.usage.querySceneAccountDaily).toHaveBeenCalledWith({ start_date: '2026-01-01', end_date: '2026-01-02', group_name: '研发' })
  })
})

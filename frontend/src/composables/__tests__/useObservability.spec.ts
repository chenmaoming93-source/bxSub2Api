import { defineComponent } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useObservability } from '../useObservability'
import { useAdminSettingsStore } from '@/stores/adminSettings'

const { snapshotMock, trendMock, latencyMock, heatmapMock } = vi.hoisted(() => ({
  snapshotMock: vi.fn(),
  trendMock: vi.fn(),
  latencyMock: vi.fn(),
  heatmapMock: vi.fn(),
}))

vi.mock('@/composables/usePermission', () => ({
  usePermission: () => ({ can: () => true }),
}))

vi.mock('@/api/admin/observability', () => ({
  getObservabilitySnapshot: (...args: unknown[]) => snapshotMock(...args),
  getObservabilityTimeseries: (...args: unknown[]) => trendMock(...args),
  getObservabilityLatency: (...args: unknown[]) => latencyMock(...args),
  getObservabilityHeatmap: (...args: unknown[]) => heatmapMock(...args),
}))

describe('useObservability polling', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    setActivePinia(createPinia())
    useAdminSettingsStore().setObservabilityEnabledLocal(true)
    snapshotMock.mockResolvedValue({
      data: {
        meta: { as_of: new Date().toISOString(), warnings: [], complete: true },
        metrics: [],
        top_users: [],
        top_groups: [],
        top_aliases: [],
        top_accounts: [],
        top_models: [],
        pressure: { accounts: [], candidates: [] },
        filter_options: { users: [], groups: [], aliases: [], accounts: [] },
      },
      notModified: false,
      etag: '"snapshot"',
    })
    trendMock.mockResolvedValue({
      data: { meta: { warnings: [], complete: true }, labels: [], series: [], summary: [] },
      notModified: false,
      etag: '"trend"',
    })
    latencyMock.mockResolvedValue({
      data: { meta: { warnings: [], complete: true }, labels: [], series: [], summary: [] },
      notModified: false,
      etag: '"latency"',
    })
    heatmapMock.mockResolvedValue({
      data: { meta: { warnings: [], complete: true }, value_kind: 'sampled_concurrency', available: true, rows: [], columns: [], cells: [] },
      notModified: false,
      etag: '"heatmap"',
    })
  })

  afterEach(() => {
    vi.clearAllMocks()
    vi.useRealTimers()
    vi.restoreAllMocks()
  })

  it('loads immediately and pauses polling while the page is hidden', async () => {
    let hidden = false
    vi.spyOn(document, 'visibilityState', 'get').mockImplementation(() => hidden ? 'hidden' : 'visible')
    const wrapper = mount(defineComponent({
      setup() {
        useObservability()
        return () => null
      },
    }))
    await flushPromises()
    expect(snapshotMock).toHaveBeenCalledTimes(1)
    expect(trendMock).toHaveBeenCalledTimes(1)
    expect(latencyMock).toHaveBeenCalledTimes(1)
    expect(heatmapMock).toHaveBeenCalledTimes(1)

    hidden = true
    document.dispatchEvent(new Event('visibilitychange'))
    await vi.advanceTimersByTimeAsync(60_000)
    expect(snapshotMock).toHaveBeenCalledTimes(1)
    expect(trendMock).toHaveBeenCalledTimes(1)
    expect(latencyMock).toHaveBeenCalledTimes(1)
    expect(heatmapMock).toHaveBeenCalledTimes(1)

    hidden = false
    document.dispatchEvent(new Event('visibilitychange'))
    await flushPromises()
    expect(snapshotMock).toHaveBeenCalledTimes(2)
    expect(trendMock).toHaveBeenCalledTimes(2)
    expect(latencyMock).toHaveBeenCalledTimes(2)
    expect(heatmapMock).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('issues no observability requests when the release flag is disabled', async () => {
    useAdminSettingsStore().setObservabilityEnabledLocal(false)
    const wrapper = mount(defineComponent({
      setup() {
        useObservability()
        return () => null
      },
    }))
    await flushPromises()
    await vi.advanceTimersByTimeAsync(120_000)
    expect(snapshotMock).not.toHaveBeenCalled()
    expect(trendMock).not.toHaveBeenCalled()
    expect(latencyMock).not.toHaveBeenCalled()
    expect(heatmapMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })

  it('sends absolute bounds for custom ranges', async () => {
    let api!: ReturnType<typeof useObservability>
    const wrapper = mount(defineComponent({
      setup() {
        api = useObservability()
        return () => null
      },
    }))
    await flushPromises()
    api.rangeStart.value = '2026-11-13T09:00'
    api.rangeEnd.value = '2026-11-13T10:00'
    api.window.value = 'custom'
    await flushPromises()
    const lastCall = snapshotMock.mock.calls.at(-1)?.[0]
    expect(lastCall.startAt).toBe('2026-11-13T09:00')
    expect(lastCall.endAt).toBe('2026-11-13T10:00')
    wrapper.unmount()
  })

  it('refreshes every data domain when global filters change', async () => {
    let api!: ReturnType<typeof useObservability>
    const wrapper = mount(defineComponent({
      setup() {
        api = useObservability()
        return () => null
      },
    }))
    await flushPromises()
    snapshotMock.mockClear()
    trendMock.mockClear()
    latencyMock.mockClear()
    heatmapMock.mockClear()

    api.filters.groupIds = [7]
    await vi.advanceTimersByTimeAsync(0)
    await flushPromises()

    expect(snapshotMock).toHaveBeenCalledTimes(1)
    expect(trendMock).toHaveBeenCalledTimes(1)
    expect(latencyMock).toHaveBeenCalledTimes(1)
    expect(heatmapMock).toHaveBeenCalledTimes(1)
    expect(snapshotMock.mock.calls[0][0].filters.groupIds).toEqual([7])
    expect(trendMock.mock.calls[0][0].filters.groupIds).toEqual([7])
    expect(latencyMock.mock.calls[0][0].filters.groupIds).toEqual([7])
    expect(heatmapMock.mock.calls[0][0].filters.groupIds).toEqual([7])
    wrapper.unmount()
  })

  it('refreshes only heatmaps when heatmap controls change', async () => {
    let api!: ReturnType<typeof useObservability>
    const wrapper = mount(defineComponent({
      setup() {
        api = useObservability()
        return () => null
      },
    }))
    await flushPromises()
    snapshotMock.mockClear()
    trendMock.mockClear()
    latencyMock.mockClear()
    heatmapMock.mockClear()

    api.heatmapValueKind.value = 'token_rate'
    await flushPromises()

    expect(snapshotMock).not.toHaveBeenCalled()
    expect(trendMock).not.toHaveBeenCalled()
    expect(latencyMock).not.toHaveBeenCalled()
    expect(heatmapMock).toHaveBeenCalledTimes(2)
    expect(heatmapMock.mock.calls.map((call) => call[0].valueKind)).toEqual([
      'token_rate',
      'sampled_concurrency',
    ])
    wrapper.unmount()
  })

  it('deduplicates the sampled-concurrency request when it is the selected heatmap', async () => {
    let api!: ReturnType<typeof useObservability>
    const wrapper = mount(defineComponent({
      setup() {
        api = useObservability()
        return () => null
      },
    }))
    await flushPromises()
    heatmapMock.mockClear()

    api.heatmapGroupBy.value = 'account_id'
    await flushPromises()

    expect(heatmapMock).toHaveBeenCalledTimes(1)
    expect(heatmapMock.mock.calls[0][0]).toMatchObject({
      groupBy: 'account_id',
      valueKind: 'sampled_concurrency',
    })
    wrapper.unmount()
  })

  it('refreshes only the time series when the trend dimension changes', async () => {
    let api!: ReturnType<typeof useObservability>
    const wrapper = mount(defineComponent({
      setup() {
        api = useObservability()
        return () => null
      },
    }))
    await flushPromises()
    snapshotMock.mockClear()
    trendMock.mockClear()
    latencyMock.mockClear()
    heatmapMock.mockClear()

    api.groupBy.value = 'account_id'
    await flushPromises()

    expect(snapshotMock).not.toHaveBeenCalled()
    expect(trendMock).toHaveBeenCalledTimes(1)
    expect(trendMock.mock.calls[0][0].groupBy).toBe('account_id')
    expect(latencyMock).not.toHaveBeenCalled()
    expect(heatmapMock).not.toHaveBeenCalled()
    wrapper.unmount()
  })
})

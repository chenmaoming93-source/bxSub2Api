import { defineComponent, h, reactive, ref } from 'vue'
import { flushPromises, mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { useObservabilityDrilldown } from '../useObservabilityDrilldown'
import type { ObservabilityFilters } from '@/api/admin/observability'

const { pathsMock, relationsMock } = vi.hoisted(() => ({
  pathsMock: vi.fn(),
  relationsMock: vi.fn(),
}))

vi.mock('@/api/admin/observability', () => ({
  getObservabilityPaths: (...args: unknown[]) => pathsMock(...args),
  getObservabilityRelations: (...args: unknown[]) => relationsMock(...args),
}))

function makeOptions() {
  const filters = reactive<ObservabilityFilters>({
    userIds: [],
    apiKeyIds: [],
    groupIds: [],
    routeAliases: [],
    accountIds: [],
    department: '',
  })
  return {
    filters,
    window: ref<'1h'>('1h'),
    bucket: ref<'5m'>('5m'),
    rangeStart: ref(''),
    rangeEnd: ref(''),
    search: ref(''),
  }
}

describe('useObservabilityDrilldown synchronization', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    pathsMock.mockResolvedValue({
      data: { meta: {}, total: 0, rows: [] },
      notModified: false,
      etag: '"paths"',
    })
    relationsMock.mockResolvedValue({
      data: { meta: {}, nodes: [], edges: [], data_quality_errors: [] },
      notModified: false,
      etag: '"relations"',
    })
  })

  afterEach(() => {
    vi.clearAllMocks()
    vi.useRealTimers()
  })

  it('loads paths and relations on first mount', async () => {
    mount(defineComponent({
      setup() {
        useObservabilityDrilldown(makeOptions())
        return () => h('div')
      },
    }))
    await flushPromises()
    expect(pathsMock).toHaveBeenCalledTimes(1)
    expect(relationsMock).toHaveBeenCalledTimes(1)
  })

  it('refreshes both paths and relations when global filters change', async () => {
    const options = makeOptions()
    mount(defineComponent({
      setup() {
        useObservabilityDrilldown(options)
        return () => h('div')
      },
    }))
    await flushPromises()
    pathsMock.mockClear()
    relationsMock.mockClear()

    options.filters.groupIds = [7]
    await vi.advanceTimersByTimeAsync(180)
    await flushPromises()

    expect(pathsMock).toHaveBeenCalledTimes(1)
    expect(relationsMock).toHaveBeenCalledTimes(1)
    expect(pathsMock.mock.calls[0][0].filters.groupIds).toEqual([7])
    expect(relationsMock.mock.calls[0][0].filters.groupIds).toEqual([7])
  })
})

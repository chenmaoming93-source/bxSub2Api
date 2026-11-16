import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { defineComponent } from 'vue'

const mocks = vi.hoisted(() => ({
  route: { query: {} as Record<string, string> },
  replace: vi.fn(),
  observability: null as Awaited<ReturnType<typeof buildObservabilityMock>> | null,
  drilldown: null as Awaited<ReturnType<typeof buildDrilldownMock>> | null,
}))

function buildObservabilityMock() {
  return import('vue').then(({ computed, reactive, ref }) => {
    const window = ref<'1h' | '6h' | '24h' | '7d' | 'today' | 'yesterday' | 'custom'>('1h')
    const rangeStart = ref('2026-11-14T13:00')
    const rangeEnd = ref('2026-11-14T14:00')
    const filters = reactive({
      userIds: [] as number[], apiKeyIds: [] as number[], groupIds: [] as number[],
      routeAliases: [] as string[], accountIds: [] as number[], department: '',
    })
    const snapshot = ref({
      meta: {
        as_of: '2026-11-14T06:00:00Z', start_at: '2026-11-14T05:00:00Z', end_at: '2026-11-14T06:00:00Z',
        window: '1h', bucket: '5m', timezone: 'Asia/Shanghai', sources: ['rollup'],
        consistency: 'eventual', lag_seconds: 3, complete: true, warnings: [], coverage: 'full',
      },
      metrics: [
        { code: 'token_rate', value: 12.5 }, { code: 'total_tokens', value: 12000 },
        { code: 'total_requests', value: 100 }, { code: 'request_count', value: 100 },
        { code: 'avg_first_token_ms', value: 850 }, { code: 'client_failure_rate', value: 0.01 },
      ],
      top_users: [], top_groups: [], top_aliases: [], top_accounts: [], top_models: [],
      pressure: {
        accounts: [{ account_id: 4, account_name: 'claude-pro-01', current: 8, limit: 10, utilization: 0.8, waiting: 2 }],
        candidates: [],
      },
      capacity: {
        global: { current: 8, limit: 10, utilization: 0.8, available: true },
        user: { current: 2, limit: 4, utilization: 0.5, available: true },
        account: { current: 8, limit: 10, utilization: 0.8, available: true },
        route: { current: 5, limit: 8, utilization: 0.625, available: true },
        queue_current: 2, average_wait_ms: 120, today_rate_limited_count: 3, rate_limit_available: true, wait_sample_count: 5,
      },
      filter_options: {
        users: [{ user_id: 1, email: 'alice@example.com', department: '研发部' }],
        groups: [{ id: 7, name: 'coding' }],
        aliases: [{ id: 8, name: 'gpt-5' }],
        accounts: [{ account_id: 4, name: 'claude-pro-01', bound_model: 'claude-sonnet-4-5' }],
      },
    })
    return {
      filters, window, rangeStart, rangeEnd,
      bucket: computed(() => '5m' as const),
      groupBy: ref('group_id'), search: ref(''), highPressureOnly: ref(false), autoRefresh: ref(true),
      snapshot, trend: ref(null), latency: ref(null), heatmap: ref(null), concurrencyHeatmap: ref(null),
      heatmapValueKind: ref('sampled_concurrency'), heatmapGroupBy: ref('group_id'),
      featureEnabled: ref(true), snapshotLoading: ref(false), trendLoading: ref(false),
      snapshotError: ref(''), trendError: ref(''), latencyError: ref(''), heatmapError: ref(''),
      lastFreshAt: ref(new Date('2026-11-14T06:00:00Z')),
      visiblePressureAccounts: computed(() => snapshot.value.pressure.accounts),
      visiblePressureCandidates: computed(() => []),
      activeFilterChips: computed(() => []),
      concurrencyOverlay: computed(() => []),
      refreshAll: vi.fn(), clearFilters: vi.fn(), removeFilter: vi.fn(),
    }
  })
}

function buildDrilldownMock() {
  return import('vue').then(({ ref }) => ({
    paths: ref(null), relations: ref(null), pathsLoading: ref(false), relationsLoading: ref(false),
    pathsError: ref(''), relationsError: ref(''), page: ref(1), pageSize: ref(20),
    sort: ref('tokens'), status: ref(''), includeIdle: ref(false), selectedPath: ref(null),
    openDetail: vi.fn(), closeDetail: vi.fn(),
  }))
}

vi.mock('vue-router', () => ({
  useRoute: () => mocks.route,
  useRouter: () => ({ replace: mocks.replace }),
}))

vi.mock('@/composables/useObservability', async () => {
  mocks.observability = await buildObservabilityMock()
  return { useObservability: () => mocks.observability }
})

vi.mock('@/composables/useObservabilityDrilldown', async () => {
  mocks.drilldown = await buildDrilldownMock()
  return { useObservabilityDrilldown: () => mocks.drilldown }
})

import ModelObservabilityView from '../ModelObservabilityView.vue'

const AppLayoutStub = defineComponent({ name: 'AppLayout', template: '<div><slot /></div>' })
const DrilldownPanelStub = defineComponent({
  name: 'DrilldownPanel',
  props: ['focus'],
  template: '<div data-testid="drilldown" />',
})
const PressureHeatmapStub = defineComponent({
  name: 'PressureHeatmap',
  props: ['rowLabels'],
  emits: ['select'],
  template: '<button data-testid="heatmap" @click="$emit(\'select\', { row: \'7\', column: \'10:00\', cell: { actual_concurrency: 8, concurrency_limit: 10 } })">heatmap</button>',
})
const TokenTrendChartStub = defineComponent({
  name: 'TokenTrendChart',
  props: ['groupBy', 'dimensionLabels'],
  template: '<div data-testid="token-trend" />',
})

function mountView() {
  return mount(ModelObservabilityView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TokenTrendChart: TokenTrendChartStub,
        QualityTrendChart: true,
        PressureRanking: true,
        PressureHeatmap: PressureHeatmapStub,
        DrilldownPanel: DrilldownPanelStub,
        ObservabilityAlertSummary: true,
        QualityTopPeriods: true,
        ConcurrencyCapacityCard: true,
        Sparkline: true,
      },
    },
  })
}

describe('ModelObservabilityView interactions', () => {
  beforeEach(() => {
    mocks.route.query = {}
    mocks.replace.mockReset()
    Object.defineProperty(Element.prototype, 'scrollIntoView', {
      configurable: true,
      value: vi.fn(),
    })
  })

  it('renders all seven KPI cards from the snapshot contract', async () => {
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.findAll('article')).toHaveLength(7)
    expect(wrapper.text()).toContain('12.5/s')
    expect(wrapper.text()).toContain('850ms')
    expect(wrapper.text()).toContain('claude-pro-01')
  })

  it('fills and applies a quick timestamp range', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[aria-controls="observability-time-menu"]').trigger('click')
    const yesterday = wrapper.findAll('button').find((button) => button.text().includes('admin.observability.quickYesterday'))
    expect(yesterday).toBeDefined()
    await yesterday!.trigger('click')
    expect(mocks.observability?.window.value).toBe('yesterday')
    expect(mocks.observability?.rangeStart.value).not.toBe('')
    expect(mocks.observability?.rangeEnd.value).not.toBe('')
  })

  it('restores linked object filters from route query', async () => {
    mocks.route.query = { group_id: '7', account_id: '4', route_alias: 'gpt-5' }
    mountView()
    await flushPromises()
    expect(mocks.observability?.filters.groupIds).toEqual([7])
    expect(mocks.observability?.filters.accountIds).toEqual([4])
    expect(mocks.observability?.filters.routeAliases).toEqual(['gpt-5'])
  })

  it('links a heatmap click to the inline drilldown without changing filters', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="heatmap"]').trigger('click')
    await flushPromises()
    expect(mocks.observability?.filters.groupIds).toEqual([])
    expect(wrapper.getComponent(DrilldownPanelStub).props('focus')).toMatchObject({ kind: 'group', key: '7' })
  })

  it('passes human-readable names for heatmap row keys', async () => {
    const wrapper = mountView()
    await flushPromises()
    mocks.observability!.heatmapGroupBy.value = 'account_id'
    await flushPromises()
    expect(wrapper.getComponent(PressureHeatmapStub).props('rowLabels')).toMatchObject({
      '4': 'claude-pro-01',
    })
  })

  it('passes the selected trend dimension and display labels to Token trend', async () => {
    const wrapper = mountView()
    await flushPromises()
    mocks.observability!.groupBy.value = 'account_id'
    await flushPromises()
    const chart = wrapper.getComponent(TokenTrendChartStub)
    expect(chart.props('groupBy')).toBe('account_id')
    expect(chart.props('dimensionLabels')).toMatchObject({ '4': 'claude-pro-01' })
  })
})

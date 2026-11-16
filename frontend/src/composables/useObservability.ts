import { computed, onBeforeUnmount, onMounted, reactive, ref, shallowRef, watch } from 'vue'
import {
  getObservabilitySnapshot,
  getObservabilityTimeseries,
  getObservabilityLatency,
  getObservabilityHeatmap,
  type ObservabilityBucket,
  type ObservabilityFilters,
  type ObservabilityGroupBy,
  type ObservabilitySnapshot,
  type ObservabilityWindow,
  type TimeSeriesResponse,
  type HeatmapResponse,
} from '@/api/admin/observability'
import { usePermission } from '@/composables/usePermission'
import { useAdminSettingsStore } from '@/stores/adminSettings'
import { isHighPressure } from '@/components/observability/status'

const SNAPSHOT_INTERVAL_MS = 5_000
const TREND_INTERVAL_MS = 45_000
const MAX_BACKOFF_MS = 60_000

export function useObservability() {
  const { can } = usePermission()
  const adminSettings = useAdminSettingsStore()
  const filters = reactive<ObservabilityFilters>({
    userIds: [],
    apiKeyIds: [],
    groupIds: [],
    routeAliases: [],
    accountIds: [],
    department: '',
  })
  const window = ref<ObservabilityWindow>('1h')
  const rangeStart = ref('')
  const rangeEnd = ref('')
  const bucket = computed<ObservabilityBucket>(() => {
    switch (window.value) {
      case '6h': return '15m'
      case '24h': return '1h'
      case '7d': return '6h'
      case 'custom': {
        const duration = new Date(rangeEnd.value).getTime() - new Date(rangeStart.value).getTime()
        if (!Number.isFinite(duration) || duration <= 0) return '5m'
        if (duration <= 2 * 60 * 60 * 1000) return '5m'
        if (duration <= 12 * 60 * 60 * 1000) return '15m'
        if (duration <= 48 * 60 * 60 * 1000) return '1h'
        return '6h'
      }
      case 'today':
      case 'yesterday':
        return '1h'
      default: return '5m'
    }
  })
  const groupBy = ref<ObservabilityGroupBy>('group_id')
  const search = ref('')
  const highPressureOnly = ref(false)
  const autoRefresh = ref(true)
  const snapshot = shallowRef<ObservabilitySnapshot | null>(null)
  const trend = shallowRef<TimeSeriesResponse | null>(null)
  const latency = shallowRef<TimeSeriesResponse | null>(null)
  const heatmap = shallowRef<HeatmapResponse | null>(null)
  const concurrencyHeatmap = shallowRef<HeatmapResponse | null>(null)
  const heatmapValueKind = ref<'sampled_concurrency' | 'token_rate' | 'error_rate'>('sampled_concurrency')
  const snapshotLoading = ref(false)
  const trendLoading = ref(false)
  const snapshotError = ref('')
  const trendError = ref('')
  const latencyError = ref('')
  const heatmapError = ref('')
  const lastFreshAt = ref<Date | null>(null)

  let snapshotTimer: ReturnType<typeof setTimeout> | undefined
  let trendTimer: ReturnType<typeof setTimeout> | undefined
  let searchTimer: ReturnType<typeof setTimeout> | undefined
  let filterTimer: ReturnType<typeof setTimeout> | undefined
  let snapshotAbort: AbortController | undefined
  let trendAbort: AbortController | undefined
  let snapshotSeq = 0
  let trendSeq = 0
  let snapshotEtag = ''
  let trendEtag = ''
  let latencyEtag = ''
  let heatmapEtag = ''
  let concurrencyHeatmapEtag = ''
  let snapshotFailures = 0
  let trendFailures = 0
  let disposed = false

  const featureEnabled = computed(() => adminSettings.observabilityEnabled)
  const canRead = computed(() => featureEnabled.value && can('observability.read'))
  const heatmapGroupBy = ref<ObservabilityGroupBy>('group_id')
  const visiblePressureAccounts = computed(() => {
    const rows = snapshot.value?.pressure.accounts || []
    return highPressureOnly.value ? rows.filter((row) => isHighPressure(row.utilization)) : rows
  })
  const visiblePressureCandidates = computed(() => {
    const rows = snapshot.value?.pressure.candidates || []
    return highPressureOnly.value ? rows.filter((row) => isHighPressure(row.utilization)) : rows
  })
  const activeFilterChips = computed(() => {
    const options = snapshot.value?.filter_options
    const chips: Array<{ key: string; type: keyof ObservabilityFilters; value: number | string; label: string }> = []
    for (const id of filters.userIds) {
      const user = options?.users.find((item) => item.user_id === id)
      chips.push({ key: `user:${id}`, type: 'userIds', value: id, label: user ? `${user.email} · ${user.department}` : `#${id}` })
    }
    for (const id of filters.groupIds) {
      const group = options?.groups.find((item) => item.id === id)
      chips.push({ key: `group:${id}`, type: 'groupIds', value: id, label: group?.name || `#${id}` })
    }
    for (const alias of filters.routeAliases) {
      chips.push({ key: `alias:${alias}`, type: 'routeAliases', value: alias, label: alias })
    }
    for (const id of filters.accountIds) {
      const account = options?.accounts.find((item) => item.account_id === id)
      chips.push({
        key: `account:${id}`,
        type: 'accountIds',
        value: id,
        label: account ? `${account.name} · ${account.bound_model || 'unknown'}` : `#${id}`,
      })
    }
    if (filters.department) {
      chips.push({ key: `department:${filters.department}`, type: 'department', value: filters.department, label: filters.department })
    }
    return chips
  })
  const concurrencyOverlay = computed(() => {
    const response = concurrencyHeatmap.value
    if (!response?.available) return []
    return response.columns.map((column) => {
      let total = 0
      for (const cell of response.cells) {
        if (cell.column === column && cell.available) total += cell.value
      }
      return total
    })
  })

  function pageHidden() {
    return typeof document !== 'undefined' && document.visibilityState === 'hidden'
  }

  function clearTimer(timer: ReturnType<typeof setTimeout> | undefined) {
    if (timer) clearTimeout(timer)
  }

  async function refreshSnapshot() {
    if (!canRead.value || disposed || pageHidden()) return
    clearTimer(snapshotTimer)
    snapshotAbort?.abort()
    snapshotAbort = new AbortController()
    const seq = ++snapshotSeq
    snapshotLoading.value = true
    snapshotError.value = ''
    try {
      const result = await getObservabilitySnapshot({
        window: window.value,
        bucket: bucket.value,
        filters,
        startAt: window.value === 'custom' ? rangeStart.value : undefined,
        endAt: window.value === 'custom' ? rangeEnd.value : undefined,
        search: search.value,
        etag: snapshotEtag,
        signal: snapshotAbort.signal,
      })
      if (seq !== snapshotSeq || disposed) return
      if (result.notModified) {
        lastFreshAt.value = new Date()
      } else if (result.data) {
        snapshot.value = result.data
        lastFreshAt.value = new Date(result.data.meta.as_of || Date.now())
      }
      snapshotEtag = result.etag || snapshotEtag
      snapshotFailures = 0
    } catch (error) {
      if ((error as { name?: string })?.name === 'CanceledError') return
      if (seq !== snapshotSeq || disposed) return
      snapshotFailures += 1
      snapshotError.value = error instanceof Error ? error.message : 'Unable to load observability snapshot'
    } finally {
      snapshotLoading.value = false
      scheduleSnapshot()
    }
  }

  async function refreshTrend() {
    if (!canRead.value || disposed || pageHidden()) return
    clearTimer(trendTimer)
    trendAbort?.abort()
    trendAbort = new AbortController()
    const seq = ++trendSeq
    trendLoading.value = true
    trendError.value = ''
    latencyError.value = ''
    heatmapError.value = ''
    try {
      const common = {
        window: window.value, bucket: bucket.value, filters,
        startAt: window.value === 'custom' ? rangeStart.value : undefined,
        endAt: window.value === 'custom' ? rangeEnd.value : undefined,
        signal: trendAbort.signal,
      }
      const capture = async <T>(promise: Promise<T>, setError: (message: string) => void): Promise<T | null> => {
        try {
          return await promise
        } catch (error) {
          if ((error as { name?: string })?.name === 'CanceledError') throw error
          setError(error instanceof Error ? error.message : 'Unable to load observability data')
          return null
        }
      }
      const selectedHeatmapPromise = capture(getObservabilityHeatmap({
        ...common, groupBy: heatmapGroupBy.value,
        valueKind: heatmapValueKind.value, etag: heatmapEtag,
      }), (message) => { heatmapError.value = message })
      const concurrencyPromise = heatmapValueKind.value === 'sampled_concurrency'
        ? Promise.resolve(null)
        : capture(getObservabilityHeatmap({
          ...common, groupBy: heatmapGroupBy.value,
          valueKind: 'sampled_concurrency', etag: concurrencyHeatmapEtag,
        }), (message) => { heatmapError.value = message })
      const [trendResult, latencyResult, heatmapResult, concurrencyResult] = await Promise.all([
        capture(getObservabilityTimeseries({ ...common, groupBy: groupBy.value, etag: trendEtag }), (message) => { trendError.value = message }),
        capture(getObservabilityLatency({ ...common, etag: latencyEtag }), (message) => { latencyError.value = message }),
        selectedHeatmapPromise,
        concurrencyPromise,
      ])
      if (seq !== trendSeq || disposed) return
      if (trendResult?.data) trend.value = trendResult.data
      if (latencyResult?.data) latency.value = latencyResult.data
      if (heatmapResult?.data) heatmap.value = heatmapResult.data
      if (heatmapValueKind.value === 'sampled_concurrency' && heatmapResult?.data) {
        concurrencyHeatmap.value = heatmapResult.data
      } else if (concurrencyResult?.data) {
        concurrencyHeatmap.value = concurrencyResult.data
      }
      trendEtag = trendResult?.etag || trendEtag
      latencyEtag = latencyResult?.etag || latencyEtag
      heatmapEtag = heatmapResult?.etag || heatmapEtag
      concurrencyHeatmapEtag = heatmapValueKind.value === 'sampled_concurrency'
        ? heatmapEtag
        : concurrencyResult?.etag || concurrencyHeatmapEtag
      trendFailures = trendError.value ? trendFailures + 1 : 0
    } catch (error) {
      if ((error as { name?: string })?.name === 'CanceledError') return
      if (seq !== trendSeq || disposed) return
      trendFailures += 1
      trendError.value = error instanceof Error ? error.message : 'Unable to load observability trend'
    } finally {
      trendLoading.value = false
      scheduleTrend()
    }
  }

  async function refreshTimeseries() {
    if (!canRead.value || disposed || pageHidden()) return
    const seq = ++trendSeq
    trendAbort?.abort()
    trendAbort = new AbortController()
    trendError.value = ''
    try {
      const result = await getObservabilityTimeseries({
        window: window.value,
        bucket: bucket.value,
        startAt: window.value === 'custom' ? rangeStart.value : undefined,
        endAt: window.value === 'custom' ? rangeEnd.value : undefined,
        filters,
        groupBy: groupBy.value,
        etag: trendEtag,
        signal: trendAbort.signal,
      })
      if (seq !== trendSeq || disposed) return
      if (result.data) trend.value = result.data
      trendEtag = result.etag || trendEtag
    } catch (error) {
      if ((error as { name?: string })?.name === 'CanceledError' || seq !== trendSeq) return
      trendError.value = error instanceof Error ? error.message : 'Unable to load observability trend'
    }
  }

  async function refreshHeatmaps() {
    if (!canRead.value || disposed || pageHidden()) return
    clearTimer(trendTimer)
    trendAbort?.abort()
    trendAbort = new AbortController()
    const seq = ++trendSeq
    heatmapError.value = ''
    const common = {
      window: window.value,
      bucket: bucket.value,
      startAt: window.value === 'custom' ? rangeStart.value : undefined,
      endAt: window.value === 'custom' ? rangeEnd.value : undefined,
      filters,
      groupBy: heatmapGroupBy.value,
      signal: trendAbort.signal,
    }
    try {
      if (heatmapValueKind.value === 'sampled_concurrency') {
        const result = await getObservabilityHeatmap({
          ...common, valueKind: 'sampled_concurrency', etag: heatmapEtag,
        })
        if (seq !== trendSeq || disposed) return
        if (result.data) {
          heatmap.value = result.data
          concurrencyHeatmap.value = result.data
        }
        heatmapEtag = result.etag || heatmapEtag
        concurrencyHeatmapEtag = heatmapEtag
        return
      }
      const [selected, concurrencyResult] = await Promise.all([
        getObservabilityHeatmap({ ...common, valueKind: heatmapValueKind.value, etag: heatmapEtag }),
        getObservabilityHeatmap({ ...common, valueKind: 'sampled_concurrency', etag: concurrencyHeatmapEtag }),
      ])
      if (seq !== trendSeq || disposed) return
      if (selected.data) heatmap.value = selected.data
      if (concurrencyResult.data) concurrencyHeatmap.value = concurrencyResult.data
      heatmapEtag = selected.etag || heatmapEtag
      concurrencyHeatmapEtag = concurrencyResult.etag || concurrencyHeatmapEtag
    } catch (error) {
      if ((error as { name?: string })?.name === 'CanceledError' || seq !== trendSeq) return
      heatmapError.value = error instanceof Error ? error.message : 'Unable to load observability heatmap'
    }
  }

  function scheduleSnapshot() {
    clearTimer(snapshotTimer)
    if (disposed || !canRead.value || pageHidden() || !autoRefresh.value) return
    const delay = Math.min(MAX_BACKOFF_MS, SNAPSHOT_INTERVAL_MS * 2 ** Math.min(snapshotFailures, 4))
    snapshotTimer = setTimeout(refreshSnapshot, delay)
  }

  function scheduleTrend() {
    clearTimer(trendTimer)
    if (disposed || !canRead.value || pageHidden() || !autoRefresh.value) return
    const delay = Math.min(MAX_BACKOFF_MS, TREND_INTERVAL_MS * 2 ** Math.min(trendFailures, 4))
    trendTimer = setTimeout(refreshTrend, delay)
  }

  function refreshAll() {
    snapshotEtag = ''
    trendEtag = ''
    latencyEtag = ''
    heatmapEtag = ''
    concurrencyHeatmapEtag = ''
    void refreshSnapshot()
    void refreshTrend()
  }

  function clearFilters() {
    filters.userIds = []
    filters.apiKeyIds = []
    filters.groupIds = []
    filters.routeAliases = []
    filters.accountIds = []
    filters.department = ''
    search.value = ''
  }

  function removeFilter(type: keyof ObservabilityFilters, value: number | string) {
    if (type === 'department') {
      filters.department = ''
      return
    }
    const list = filters[type] as unknown[]
    filters[type] = list.filter((item) => item !== value) as never
  }

  function onVisibilityChange() {
    if (pageHidden()) {
      clearTimer(snapshotTimer)
      clearTimer(trendTimer)
      snapshotAbort?.abort()
      trendAbort?.abort()
      return
    }
    refreshAll()
  }

  watch([window, bucket, rangeStart, rangeEnd], refreshAll)
  watch(groupBy, () => void refreshTimeseries())
  watch([heatmapGroupBy, heatmapValueKind], () => void refreshHeatmaps())
  watch([
    () => filters.userIds.slice(),
    () => filters.groupIds.slice(),
    () => filters.routeAliases.slice(),
    () => filters.accountIds.slice(),
    () => filters.department,
  ], () => {
    clearTimer(filterTimer)
    filterTimer = setTimeout(refreshAll, 0)
  })
  watch(search, () => {
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(refreshSnapshot, 300)
  })
  watch(autoRefresh, (enabled) => {
    if (enabled) {
      refreshAll()
      return
    }
    clearTimer(snapshotTimer)
    clearTimer(trendTimer)
  })

  onMounted(() => {
    if (!canRead.value) return
    document.addEventListener('visibilitychange', onVisibilityChange)
    refreshAll()
  })

  onBeforeUnmount(() => {
    disposed = true
    document.removeEventListener('visibilitychange', onVisibilityChange)
    clearTimer(snapshotTimer)
    clearTimer(trendTimer)
    clearTimer(searchTimer)
    clearTimer(filterTimer)
    snapshotAbort?.abort()
    trendAbort?.abort()
  })

  return {
    filters,
    window,
    rangeStart,
    rangeEnd,
    bucket,
    groupBy,
    search,
    highPressureOnly,
    autoRefresh,
    snapshot,
    trend,
    latency,
    heatmap,
    concurrencyHeatmap,
    heatmapValueKind,
    heatmapGroupBy,
    snapshotLoading,
    trendLoading,
    snapshotError,
    trendError,
    latencyError,
    heatmapError,
    lastFreshAt,
    canRead,
    featureEnabled,
    visiblePressureAccounts,
    visiblePressureCandidates,
    activeFilterChips,
    concurrencyOverlay,
    refreshAll,
    clearFilters,
    removeFilter,
  }
}

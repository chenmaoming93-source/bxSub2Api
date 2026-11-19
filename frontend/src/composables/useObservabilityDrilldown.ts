import { nextTick, onBeforeUnmount, onMounted, ref, shallowRef, watch, type Ref } from 'vue'
import {
  getObservabilityPaths,
  getObservabilityRelations,
  type ObservabilityBucket,
  type ObservabilityFilters,
  type ObservabilityWindow,
  type PathRow,
  type PathsResponse,
  type RelationsResponse,
} from '@/api/admin/observability'

interface DrilldownOptions {
  filters: ObservabilityFilters
  window: Ref<ObservabilityWindow>
  bucket: Ref<ObservabilityBucket>
  rangeStart: Ref<string>
  rangeEnd: Ref<string>
  search: Ref<string>
}

export function useObservabilityDrilldown(options: DrilldownOptions) {
  const paths = shallowRef<PathsResponse | null>(null)
  const relations = shallowRef<RelationsResponse | null>(null)
  const pathsLoading = ref(false)
  const relationsLoading = ref(false)
  const pathsError = ref('')
  const relationsError = ref('')
  const page = ref(1)
  const pageSize = ref(20)
  const sort = ref('tokens')
  const status = ref('')
  const includeIdle = ref(false)
  const selectedPath = shallowRef<PathRow | null>(null)

  let pathAbort: AbortController | undefined
  let relationAbort: AbortController | undefined
  let pathSeq = 0
  let relationSeq = 0
  let pathTimer: ReturnType<typeof setTimeout> | undefined
  let relationTimer: ReturnType<typeof setTimeout> | undefined
  let lastFocused: HTMLElement | null = null

  async function refreshPaths() {
    pathAbort?.abort()
    pathAbort = new AbortController()
    const seq = ++pathSeq
    pathsLoading.value = true
    pathsError.value = ''
    try {
      const result = await getObservabilityPaths({
        window: options.window.value,
        bucket: options.bucket.value,
        startAt: options.window.value === 'custom' ? options.rangeStart.value : undefined,
        endAt: options.window.value === 'custom' ? options.rangeEnd.value : undefined,
        filters: options.filters,
        search: options.search.value,
        page: page.value,
        pageSize: pageSize.value,
        sort: sort.value,
        status: status.value,
        includeIdle: includeIdle.value,
        signal: pathAbort.signal,
      })
      if (seq !== pathSeq) return
      if (result.data) paths.value = result.data
    } catch (error) {
      if ((error as { name?: string })?.name === 'CanceledError' || seq !== pathSeq) return
      pathsError.value = error instanceof Error ? error.message : 'Unable to load paths'
    } finally {
      if (seq === pathSeq) pathsLoading.value = false
    }
  }

  async function refreshRelations() {
    relationAbort?.abort()
    relationAbort = new AbortController()
    const seq = ++relationSeq
    relationsLoading.value = true
    relationsError.value = ''
    try {
      const result = await getObservabilityRelations({
        window: options.window.value,
        bucket: options.bucket.value,
        startAt: options.window.value === 'custom' ? options.rangeStart.value : undefined,
        endAt: options.window.value === 'custom' ? options.rangeEnd.value : undefined,
        filters: options.filters,
        search: options.search.value,
        includeIdle: includeIdle.value,
        signal: relationAbort.signal,
      })
      if (seq !== relationSeq) return
      if (result.data) relations.value = result.data
    } catch (error) {
      if ((error as { name?: string })?.name === 'CanceledError' || seq !== relationSeq) return
      relationsError.value = error instanceof Error ? error.message : 'Unable to load relations'
    } finally {
      if (seq === relationSeq) relationsLoading.value = false
    }
  }

  function schedulePaths() {
    if (pathTimer) clearTimeout(pathTimer)
    pathTimer = setTimeout(refreshPaths, 180)
  }

  function scheduleRelations() {
    if (relationTimer) clearTimeout(relationTimer)
    relationTimer = setTimeout(refreshRelations, 180)
  }

  function openDetail(row: PathRow) {
    lastFocused = typeof document !== 'undefined' ? document.activeElement as HTMLElement : null
    selectedPath.value = row
    void nextTick(() => document.querySelector<HTMLElement>('[role="dialog"] button')?.focus())
  }

  function closeDetail() {
    selectedPath.value = null
    lastFocused?.focus()
    lastFocused = null
  }

  function handleEscape(event: KeyboardEvent) {
    if (event.key === 'Escape' && selectedPath.value) closeDetail()
  }

  function nextPage() {
    if (paths.value && page.value * pageSize.value < paths.value.total) page.value += 1
  }

  function previousPage() {
    if (page.value > 1) page.value -= 1
  }

  watch([
    options.window,
    options.bucket,
    options.rangeStart,
    options.rangeEnd,
    page,
    pageSize,
    sort,
    status,
    includeIdle,
    options.search,
  ], () => {
    if (page.value !== 1 && (sort.value || status.value || options.search.value)) page.value = 1
    schedulePaths()
    scheduleRelations()
  })
  watch([
    () => options.filters.userIds.slice(),
    () => options.filters.groupIds.slice(),
    () => options.filters.routeAliases.slice(),
    () => options.filters.accountIds.slice(),
    () => options.filters.department,
  ], () => {
    if (page.value !== 1) page.value = 1
    schedulePaths()
    scheduleRelations()
  })

  onMounted(() => {
    void refreshPaths()
    void refreshRelations()
  })
  onMounted(() => document.addEventListener('keydown', handleEscape))
  onBeforeUnmount(() => {
    pathAbort?.abort()
    relationAbort?.abort()
    if (pathTimer) clearTimeout(pathTimer)
    if (relationTimer) clearTimeout(relationTimer)
    document.removeEventListener('keydown', handleEscape)
  })

  return {
    paths,
    relations,
    pathsLoading,
    relationsLoading,
    pathsError,
    relationsError,
    page,
    pageSize,
    sort,
    status,
    includeIdle,
    selectedPath,
    refreshPaths,
    refreshRelations,
    openDetail,
    closeDetail,
    nextPage,
    previousPage,
  }
}

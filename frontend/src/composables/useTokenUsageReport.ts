import { ref, readonly, onUnmounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'

export interface UseTokenUsageReportOptions {
  /** Prefix for URL query parameter names (e.g. "model", "route", "user") */
  paramPrefix: string
}

/**
 * Composable for token usage report pages.
 *
 * Provides:
 * - URL-driven state (target, dates, pagination) synced with query parameters
 * - AbortController lifecycle tied to component unmount
 * - Request sequence number to reject stale responses
 * - Default dates set to today via global timezone
 */
export function useTokenUsageReport(options: UseTokenUsageReportOptions) {
  const route = useRoute()
  const router = useRouter()
  const { paramPrefix } = options

  let seq = 0
  let currentController: AbortController | null = null

  // ── Helpers ──

  function todayStr(): string {
    const d = new Date()
    const y = d.getFullYear()
    const m = String(d.getMonth() + 1).padStart(2, '0')
    const day = String(d.getDate()).padStart(2, '0')
    return `${y}-${m}-${day}`
  }

  function getQuery(key: string): string {
    return (route.query[key] as string) || ''
  }

  async function updateQuery(obj: Record<string, string | undefined>) {
    const next = { ...route.query }
    for (const [k, v] of Object.entries(obj)) {
      if (v === undefined || v === '') {
        delete next[k]
      } else {
        next[k] = v
      }
    }
    await router.replace({ query: next })
  }

  // ── State ──

  const startDate = ref(getQuery(`${paramPrefix}_start`) || todayStr())
  const endDate = ref(getQuery(`${paramPrefix}_end`) || todayStr())
  const page = ref(Number(getQuery(`${paramPrefix}_page`)) || 1)
  const pageSize = ref(Math.min(Number(getQuery(`${paramPrefix}_page_size`)) || 20, 100))
  const targetId = ref(getQuery(`${paramPrefix}_target`) || '')
  const targetLabel = ref(getQuery(`${paramPrefix}_label`) || '')

  const loading = ref(false)
  const error = ref<string | null>(null)

  // ── Abort controller ──

  function nextSignal(): { signal: AbortSignal; seq: number } {
    currentController?.abort()
    const ctrl = new AbortController()
    currentController = ctrl
    seq++
    return { signal: ctrl.signal, seq }
  }

  function isCurrent(s: number): boolean {
    return s === seq
  }

  function isAbortError(e: unknown): boolean {
    if (e instanceof DOMException && e.name === 'AbortError') return true
    if (e && typeof e === 'object' && 'code' in e && (e as any).code === 'ERR_CANCELED') return true
    return false
  }

  // ── URL sync ──

  interface SyncParams {
    startDate?: string
    endDate?: string
    page?: number
    pageSize?: number
    targetId?: string
    targetLabel?: string
  }

  async function syncToUrl(p: SyncParams = {}) {
    const prefix = paramPrefix
    await updateQuery({
      [`${prefix}_start`]: p.startDate ?? startDate.value,
      [`${prefix}_end`]: p.endDate ?? endDate.value,
      [`${prefix}_page`]: String(p.page ?? page.value),
      [`${prefix}_page_size`]: String(p.pageSize ?? pageSize.value),
      [`${prefix}_target`]: (p.targetId ?? targetId.value) || undefined,
      [`${prefix}_label`]: (p.targetLabel ?? targetLabel.value) || undefined
    })
  }

  async function setDates(start: string, end: string) {
    startDate.value = start
    endDate.value = end
    page.value = 1
    await syncToUrl({ startDate: start, endDate: end, page: 1 })
  }

  async function setPage(p: number) {
    page.value = p
    await syncToUrl({ page: p })
  }

  async function setPageSize(ps: number) {
    const capped = Math.min(ps, 100)
    pageSize.value = capped
    page.value = 1
    await syncToUrl({ pageSize: capped, page: 1 })
  }

  async function setTarget(id: string, label: string) {
    targetId.value = id
    targetLabel.value = label
    page.value = 1
    await syncToUrl({ targetId: id, targetLabel: label, page: 1 })
  }

  function resetPage() {
    page.value = 1
  }

  // ── Cleanup ──

  onUnmounted(() => {
    currentController?.abort()
  })

  return {
    // state
    startDate: readonly(startDate),
    endDate: readonly(endDate),
    page: readonly(page),
    pageSize: readonly(pageSize),
    targetId: readonly(targetId),
    targetLabel: readonly(targetLabel),
    loading,
    error,
    // actions
    setDates,
    setPage,
    setPageSize,
    setTarget,
    resetPage,
    syncToUrl,
    // abort + sequence
    nextSignal,
    isCurrent,
    isAbortError,
    // helpers
    todayStr
  }
}

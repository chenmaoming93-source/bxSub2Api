import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'

// Mock vue-router
const mockQuery: Record<string, string> = {}
const mockReplace = vi.fn().mockResolvedValue(undefined)

vi.mock('vue-router', () => ({
  useRoute: () => ({
    query: mockQuery
  }),
  useRouter: () => ({
    replace: mockReplace
  })
}))

import { useTokenUsageReport } from '@/composables/useTokenUsageReport'

describe('useTokenUsageReport', () => {
  beforeEach(() => {
    for (const k of Object.keys(mockQuery)) {
      delete mockQuery[k]
    }
    mockReplace.mockReset()
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  describe('default date = today', () => {
    it('defaults start/end to today when no URL params exist', () => {
      const { startDate, endDate } = useTokenUsageReport({ paramPrefix: 'model' })
      const today = new Date()
      const y = today.getFullYear()
      const m = String(today.getMonth() + 1).padStart(2, '0')
      const d = String(today.getDate()).padStart(2, '0')
      const expected = `${y}-${m}-${d}`
      expect(startDate.value).toBe(expected)
      expect(endDate.value).toBe(expected)
    })

    it('uses URL params when present', () => {
      mockQuery.model_start = '2026-06-01'
      mockQuery.model_end = '2026-06-30'
      const { startDate, endDate } = useTokenUsageReport({ paramPrefix: 'model' })
      expect(startDate.value).toBe('2026-06-01')
      expect(endDate.value).toBe('2026-06-30')
    })
  })

  describe('page size is bounded to 100', () => {
    it('defaults to 20', () => {
      const { pageSize } = useTokenUsageReport({ paramPrefix: 'test' })
      expect(pageSize.value).toBe(20)
    })

    it('reads from URL', () => {
      mockQuery.test_page_size = '50'
      const { pageSize } = useTokenUsageReport({ paramPrefix: 'test' })
      expect(pageSize.value).toBe(50)
    })

    it('caps at 100', async () => {
      const { pageSize, setPageSize } = useTokenUsageReport({ paramPrefix: 'test' })
      await setPageSize(200)
      expect(pageSize.value).toBe(100)
    })
  })

  describe('URL state sync', () => {
    it('setDates updates URL and resets page', async () => {
      const { setDates, page } = useTokenUsageReport({ paramPrefix: 'route' })
      await setDates('2026-07-01', '2026-07-05')
      expect(page.value).toBe(1)
      expect(mockReplace).toHaveBeenCalled()
    })

    it('setTarget updates URL with id and label', async () => {
      const { setTarget, targetId, targetLabel } = useTokenUsageReport({ paramPrefix: 'user' })
      await setTarget('42', 'alice@example.com')
      expect(targetId.value).toBe('42')
      expect(targetLabel.value).toBe('alice@example.com')
    })

    it('setPage updates URL', async () => {
      const { setPage, page } = useTokenUsageReport({ paramPrefix: 'model' })
      await setPage(3)
      expect(page.value).toBe(3)
    })
  })

  describe('abort controller lifecycle', () => {
    it('nextSignal creates a new AbortController each call', () => {
      const { nextSignal } = useTokenUsageReport({ paramPrefix: 'test' })
      const s1 = nextSignal()
      const s2 = nextSignal()
      expect(s2.seq).toBeGreaterThan(s1.seq)
      expect(s1.signal.aborted).toBe(true) // previous aborted
    })

    it('isCurrent returns false for stale seq', () => {
      const { nextSignal, isCurrent } = useTokenUsageReport({ paramPrefix: 'test' })
      const { seq } = nextSignal()
      nextSignal() // invalidates previous
      expect(isCurrent(seq)).toBe(false)
    })

    it('isCurrent returns true for current seq', () => {
      const { nextSignal, isCurrent } = useTokenUsageReport({ paramPrefix: 'test' })
      const { seq } = nextSignal()
      expect(isCurrent(seq)).toBe(true)
    })

    it('isAbortError detects AbortError', () => {
      const { isAbortError } = useTokenUsageReport({ paramPrefix: 'test' })
      expect(isAbortError(new DOMException('aborted', 'AbortError'))).toBe(true)
      expect(isAbortError(new Error('other'))).toBe(false)
      expect(isAbortError({ code: 'ERR_CANCELED' })).toBe(true)
    })
  })
})

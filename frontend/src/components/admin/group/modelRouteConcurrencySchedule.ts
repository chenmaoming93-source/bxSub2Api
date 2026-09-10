export interface ScheduleDraft {
  id?: number
  start: string
  end: string
  maxConcurrency: number | null | string
}

export function parseScheduleMinute(value: string, allowEndOfDay: boolean): number | null {
  const trimmed = value.trim()
  if (allowEndOfDay && trimmed === '24:00') return 1440
  if (!/^\d{2}:\d{2}$/.test(trimmed)) return null
  const hour = Number(trimmed.slice(0, 2))
  const minute = Number(trimmed.slice(3))
  if (!Number.isInteger(hour) || !Number.isInteger(minute) || hour < 0 || hour > 23 || minute < 0 || minute > 59) return null
  return hour * 60 + minute
}

export function validateScheduleDrafts(rows: ScheduleDraft[]): string | null {
  const parsed: Array<{ start: number; end: number; index: number }> = []
  for (let index = 0; index < rows.length; index += 1) {
    const row = rows[index]
    const start = parseScheduleMinute(row.start, false)
    const end = parseScheduleMinute(row.end, true)
    if (start == null || end == null || start >= end) return `第 ${index + 1} 行时间段无效`
    const maxConcurrency = row.maxConcurrency === '' ? null : row.maxConcurrency
    const numericMaxConcurrency = maxConcurrency === null ? null : Number(maxConcurrency)
    if (numericMaxConcurrency !== null && (!Number.isInteger(numericMaxConcurrency) || numericMaxConcurrency <= 0)) {
      return `第 ${index + 1} 行并发数必须是正整数或不限制`
    }
    parsed.push({ start, end, index })
  }
  parsed.sort((left, right) => left.start - right.start || left.end - right.end)
  for (let index = 1; index < parsed.length; index += 1) {
    if (parsed[index].start < parsed[index - 1].end) return '分时段时间不能重叠'
  }
  return null
}

export function toSchedulePayload(rows: ScheduleDraft[]) {
  return rows.map(row => ({
    id: row.id,
    start: row.start.trim(),
    end: row.end.trim(),
    max_concurrency: row.maxConcurrency === '' || row.maxConcurrency == null ? null : Number(row.maxConcurrency)
  }))
}

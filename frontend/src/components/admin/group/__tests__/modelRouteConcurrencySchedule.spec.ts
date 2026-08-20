import { describe, expect, it } from 'vitest'
import { parseScheduleMinute, validateScheduleDrafts, type ScheduleDraft } from '../modelRouteConcurrencySchedule'

const row = (start: string, end: string, maxConcurrency: number | null = 10): ScheduleDraft => ({
  start,
  end,
  maxConcurrency
})

describe('model route concurrency schedule validation', () => {
  it('converts minute boundaries and accepts 24:00 only as end', () => {
    expect(parseScheduleMinute('00:00', false)).toBe(0)
    expect(parseScheduleMinute('09:30', false)).toBe(570)
    expect(parseScheduleMinute('24:00', true)).toBe(1440)
    expect(parseScheduleMinute('24:00', false)).toBeNull()
  })

  it('allows adjacent and incomplete windows but rejects overlap', () => {
    expect(validateScheduleDrafts([row('00:00', '09:30'), row('09:30', '20:30')])).toBeNull()
    expect(validateScheduleDrafts([row('09:30', '20:30')])).toBeNull()
    expect(validateScheduleDrafts([row('09:00', '10:00'), row('09:30', '11:00')])).toBe('分时段时间不能重叠')
  })

  it('allows unlimited and rejects invalid numeric limits', () => {
    expect(validateScheduleDrafts([row('00:00', '01:00', null)])).toBeNull()
    expect(validateScheduleDrafts([{ ...row('00:00', '01:00'), maxConcurrency: 0 }])).toContain('正整数')
  })
})

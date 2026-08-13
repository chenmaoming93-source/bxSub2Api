import { describe, expect, it } from 'vitest'
import {
  buildModelAttributes,
  displayAttributeValue,
  parseAttributeValue,
  rowsFromModelAttributes,
  type ModelAttributeRow
} from '../modelAttributeUtils'

describe('parseAttributeValue', () => {
  it('parses numbers, booleans and JSON arrays', () => {
    expect(parseAttributeValue('200000')).toBe(200000)
    expect(parseAttributeValue('true')).toBe(true)
    expect(parseAttributeValue('false')).toBe(false)
    expect(parseAttributeValue('["text","image"]')).toEqual(['text', 'image'])
  })

  it('keeps plain text and dates as strings', () => {
    expect(parseAttributeValue('2025-06-01')).toBe('2025-06-01')
    expect(parseAttributeValue('任意文本')).toBe('任意文本')
  })

  it('returns empty string for blank input (explicit empty value is legal)', () => {
    expect(parseAttributeValue('')).toBe('')
    expect(parseAttributeValue('   ')).toBe('')
  })
})

describe('displayAttributeValue', () => {
  it('keeps strings as-is and stringifies other types', () => {
    expect(displayAttributeValue('raw')).toBe('raw')
    expect(displayAttributeValue(200000)).toBe('200000')
    expect(displayAttributeValue(true)).toBe('true')
    expect(displayAttributeValue(['text', 'image'])).toBe('["text","image"]')
  })

  it('returns empty text for null/undefined', () => {
    expect(displayAttributeValue(null)).toBe('')
    expect(displayAttributeValue(undefined)).toBe('')
  })
})

describe('buildModelAttributes', () => {
  const row = (partial: Partial<ModelAttributeRow>): ModelAttributeRow => ({
    uid: 'u',
    key: '',
    description: '',
    valueText: '',
    ...partial
  })

  it('drops rows with blank key or blank value', () => {
    const out = buildModelAttributes([
      row({ key: 'context_window', valueText: '200000' }),
      row({ key: '   ', valueText: 'x' }),
      row({ key: 'supports_vision', valueText: '' }),
      row({ key: '', valueText: 'y' })
    ])
    expect(Object.keys(out)).toEqual(['context_window'])
  })

  it('trims key and parses value; omits empty description', () => {
    const out = buildModelAttributes([
      row({ key: '  context_window  ', description: '上下文窗口', valueText: '200000' }),
      row({ key: 'supports_vision', valueText: 'true' })
    ])
    expect(Object.keys(out)).toEqual(['context_window', 'supports_vision'])
    expect(out.context_window).toEqual({ description: '上下文窗口', value: 200000 })
    expect(out.supports_vision).toEqual({ value: true })
  })

  it('keeps first occurrence when keys duplicate', () => {
    const out = buildModelAttributes([
      row({ key: 'dup', description: 'first', valueText: '1' }),
      row({ key: ' dup ', description: 'second', valueText: '2' })
    ])
    expect(Object.keys(out)).toEqual(['dup'])
    expect(out.dup).toEqual({ description: 'first', value: 1 })
  })

  it('returns empty object for no rows', () => {
    expect(buildModelAttributes([])).toEqual({})
  })
})

describe('rowsFromModelAttributes', () => {
  const uid = () => 'uid'

  it('returns empty rows for null/undefined', () => {
    expect(rowsFromModelAttributes(null, uid)).toEqual([])
    expect(rowsFromModelAttributes(undefined, uid)).toEqual([])
  })

  it('maps entries to rows preserving order and value display', () => {
    const rows = rowsFromModelAttributes(
      {
        context_window: { description: '上下文窗口', value: 200000 },
        supports_vision: { description: '', value: true },
        raw: { value: '2025-06-01' }
      },
      uid
    )
    expect(rows).toEqual([
      { uid: 'uid', key: 'context_window', description: '上下文窗口', valueText: '200000' },
      { uid: 'uid', key: 'supports_vision', description: '', valueText: 'true' },
      { uid: 'uid', key: 'raw', description: '', valueText: '2025-06-01' }
    ])
  })
})

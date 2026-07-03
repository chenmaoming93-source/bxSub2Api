import { describe, expect, it } from 'vitest'
import en from '../locales/en'
import zh from '../locales/zh'

function keyShape(value: unknown): unknown {
  if (value === null || typeof value !== 'object') return typeof value
  return Object.fromEntries(
    Object.entries(value as Record<string, unknown>)
      .sort(([left], [right]) => left.localeCompare(right))
      .map(([key, child]) => [key, keyShape(child)])
  )
}

describe('model routing and token quota locale keys', () => {
  it('keeps the new zh/en locale key trees symmetric', () => {
    expect(keyShape(zh.admin.groups.modelRouting)).toEqual(keyShape(en.admin.groups.modelRouting))
    expect(keyShape(zh.admin.groups.modelTokenQuota)).toEqual(keyShape(en.admin.groups.modelTokenQuota))
    expect(keyShape(zh.admin.users.modelTokenQuota)).toEqual(keyShape(en.admin.users.modelTokenQuota))
  })

  it('contains translated labels rather than raw locale keys', () => {
    expect(zh.admin.groups.modelTokenQuota.title).not.toContain('admin.')
    expect(en.admin.groups.modelTokenQuota.title).not.toContain('admin.')
    expect(zh.admin.users.modelTokenQuota.title).not.toContain('admin.')
    expect(en.admin.users.modelTokenQuota.title).not.toContain('admin.')
    expect(zh.admin.groups.modelRouting.addCandidate).not.toContain('admin.')
    expect(en.admin.groups.modelRouting.addCandidate).not.toContain('admin.')
  })
})

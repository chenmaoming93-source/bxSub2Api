import { describe, expect, it } from 'vitest'
import { normalizeModelRouting, serializeModelRouting, validateModelRouting } from '../groupsModelRouting'
import type { ModelRoutingConfig, ModelRoutingRuleRow } from '@/types'

describe('groups model routing normalizer', () => {
  it('preserves legacy account order and serializes without model', () => {
    const rows = normalizeModelRouting({ 'claude-*': [9, 3, 7] })
    expect(rows).toEqual([{ alias: 'claude-*', candidates: [{ account_ids: [9, 3, 7], priority: 0 }] }])
    expect(serializeModelRouting(rows)).toEqual({
      'claude-*': [{ account_ids: [9, 3, 7], priority: 0 }]
    })
  })

  it('accepts historical model fields, drops them, and sorts priorities stably', () => {
    const historical = {
      coding: [
        { model: 'gpt-5', account_ids: [4, 2], priority: 20 },
        { model: 'claude-sonnet', account_ids: [8], priority: 10 },
        { model: 'gemini-pro', account_ids: [6], priority: 10 }
      ]
    } as unknown as ModelRoutingConfig
    const rows = normalizeModelRouting(historical)
    expect(serializeModelRouting(rows)).toEqual({
      coding: [
        { account_ids: [8], priority: 10 },
        { account_ids: [6], priority: 10 },
        { account_ids: [4, 2], priority: 20 }
      ]
    })
  })

  it('allows multiple candidates with the same priority and reports duplicate aliases', () => {
    const rows: ModelRoutingRuleRow[] = [
      { alias: 'coding', candidates: [{ account_ids: [1], priority: 1 }, { account_ids: [2], priority: 1 }] },
      { alias: 'coding', candidates: [{ account_ids: [3], priority: 2 }] }
    ]
    expect(validateModelRouting(rows).map(issue => issue.code)).toEqual(['duplicate_alias'])
  })

  it('keeps one account per candidate and rejects cross-priority reuse', () => {
    const rows: ModelRoutingRuleRow[] = [{
      alias: 'coding',
      candidates: [
        { account_ids: [1], priority: 1 },
        { account_ids: [1], priority: 2 }
      ]
    }]
    expect(validateModelRouting(rows).map(issue => issue.code)).toEqual(['account_priority_conflict'])
    expect(validateModelRouting([{ alias: 'coding', candidates: [{ account_ids: [1, 2], priority: 1 }] }]).map(issue => issue.code)).toEqual(['candidate_multiple_accounts'])
  })
})

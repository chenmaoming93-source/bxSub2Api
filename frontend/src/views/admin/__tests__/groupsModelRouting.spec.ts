import { describe, expect, it } from 'vitest'
import {
  normalizeModelRouting,
  serializeModelRouting,
  validateModelRouting
} from '../groupsModelRouting'
import type { ModelRoutingConfig, ModelRoutingRuleRow } from '@/types'

describe('groups model routing normalizer', () => {
  it('preserves legacy account order and serializes the normalized candidate format', () => {
    const rows = normalizeModelRouting({
      'claude-*': [9, 3, 7]
    })

    expect(rows).toEqual([
      {
        alias: 'claude-*',
        candidates: [
          {
            model: 'claude-*',
            account_ids: [9, 3, 7],
            priority: 0,
            daily_token_limit: null
          }
        ]
      }
    ])
    expect(serializeModelRouting(rows)['claude-*'][0].account_ids).toEqual([9, 3, 7])
  })

  it('round trips every new candidate field and sorts priorities stably', () => {
    const config: ModelRoutingConfig = {
      coding: [
        { model: 'gpt-5', account_ids: [4, 2], priority: 20, daily_token_limit: 1000 },
        { model: 'claude-sonnet', account_ids: [8], priority: 10, daily_token_limit: null },
        { model: 'gemini-pro', account_ids: [6], priority: 10, daily_token_limit: 0 }
      ]
    }

    const rows = normalizeModelRouting(config)
    expect(rows[0].candidates.map(candidate => candidate.model)).toEqual([
      'claude-sonnet',
      'gemini-pro',
      'gpt-5'
    ])
    expect(serializeModelRouting(rows)).toEqual({
      coding: [
        { model: 'claude-sonnet', account_ids: [8], priority: 10, daily_token_limit: null },
        { model: 'gemini-pro', account_ids: [6], priority: 10, daily_token_limit: 0 },
        { model: 'gpt-5', account_ids: [4, 2], priority: 20, daily_token_limit: 1000 }
      ]
    })
  })

  it('reports duplicate aliases, models, priorities, and negative limits', () => {
    const rows: ModelRoutingRuleRow[] = [
      {
        alias: 'coding',
        candidates: [
          { model: 'gpt-5', account_ids: [1], priority: 1, daily_token_limit: 100 },
          { model: 'gpt-5', account_ids: [2], priority: 1, daily_token_limit: -1 }
        ]
      },
      {
        alias: 'coding',
        candidates: [{ model: 'claude', account_ids: [3], priority: 2, daily_token_limit: null }]
      }
    ]

    expect(validateModelRouting(rows).map(issue => issue.code)).toEqual([
      'duplicate_model',
      'duplicate_priority',
      'invalid_daily_token_limit',
      'duplicate_alias'
    ])
  })
})

import { describe, expect, it } from 'vitest'
import { addRoutingCandidate, createEmptyRoutingCandidate, type RoutingEditorRule } from '../groupModelRoutingEditor'

describe('GroupModelRoutingEditor', () => {
  it('creates candidates with the same defaults as GroupsView', () => {
    expect(createEmptyRoutingCandidate()).toEqual({
      model: '', accounts: [], priority: 0, daily_token_limit: null
    })
  })

  it('appends a candidate with the next priority', () => {
    const rule: RoutingEditorRule = {
      alias: 'coding',
      candidates: [
        { model: 'first', accounts: [{ id: 1, name: 'one' }], priority: 4, daily_token_limit: null },
        { model: 'second', accounts: [{ id: 2, name: 'two' }], priority: 9, daily_token_limit: 100 }
      ]
    }
    addRoutingCandidate(rule)
    expect(rule.candidates.at(-1)).toEqual({
      model: '', accounts: [], priority: 10, daily_token_limit: null
    })
  })
})

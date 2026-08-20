import type {
  ModelRoutingCandidate,
  ModelRoutingConfig,
  ModelRoutingRuleRow
} from '@/types'

export type ModelRoutingValidationCode =
  | 'alias_required'
  | 'duplicate_alias'
  | 'candidate_required'
  | 'account_ids_required'
  | 'candidate_multiple_accounts'
  | 'invalid_account_id'
  | 'invalid_priority'
  | 'account_priority_conflict'

export interface ModelRoutingValidationIssue {
  code: ModelRoutingValidationCode
  ruleIndex: number
  candidateIndex?: number
  value?: string | number
}

function cloneCandidate(candidate: ModelRoutingCandidate): ModelRoutingCandidate {
  return {
    account_ids: [...candidate.account_ids],
    priority: candidate.priority
  }
}

export function normalizeModelRouting(config: ModelRoutingConfig | null | undefined): ModelRoutingRuleRow[] {
  if (!config) return []

  return Object.entries(config)
    .sort(([left], [right]) => left.localeCompare(right))
    .map(([alias, value]) => {
      if (value.length === 0 || typeof value[0] === 'number') {
        return {
          alias,
          candidates: [
            {
              account_ids: [...(value as number[])],
              priority: 0
            }
          ]
        }
      }

      const candidates = (value as ModelRoutingCandidate[])
        .map((candidate, index) => ({ candidate: cloneCandidate(candidate), index }))
        .sort((left, right) => left.candidate.priority - right.candidate.priority || left.index - right.index)
        .map(item => item.candidate)
      return { alias, candidates }
    })
}

export function serializeModelRouting(rows: ModelRoutingRuleRow[]): Record<string, ModelRoutingCandidate[]> {
  return [...rows]
    .map(row => ({ alias: row.alias.trim(), candidates: row.candidates }))
    .sort((left, right) => left.alias.localeCompare(right.alias))
    .reduce<Record<string, ModelRoutingCandidate[]>>((result, row) => {
      result[row.alias] = row.candidates
        .map((candidate, index) => ({ candidate: cloneCandidate(candidate), index }))
        .sort((left, right) => left.candidate.priority - right.candidate.priority || left.index - right.index)
        .map(item => item.candidate)
      return result
    }, {})
}

export function validateModelRouting(rows: ModelRoutingRuleRow[]): ModelRoutingValidationIssue[] {
  const issues: ModelRoutingValidationIssue[] = []
  const aliases = new Map<string, number>()

  rows.forEach((row, ruleIndex) => {
    const alias = row.alias.trim()
    if (!alias) {
      issues.push({ code: 'alias_required', ruleIndex })
    } else if (aliases.has(alias)) {
      issues.push({ code: 'duplicate_alias', ruleIndex, value: alias })
    } else {
      aliases.set(alias, ruleIndex)
    }
    if (row.candidates.length === 0) {
      issues.push({ code: 'candidate_required', ruleIndex })
      return
    }

    const accountPriorities = new Map<number, number>()
    row.candidates.forEach((candidate, candidateIndex) => {
      if (candidate.account_ids.length === 0) {
        issues.push({ code: 'account_ids_required', ruleIndex, candidateIndex })
      } else if (candidate.account_ids.length > 1) {
        issues.push({ code: 'candidate_multiple_accounts', ruleIndex, candidateIndex })
      } else if (candidate.account_ids.some(id => !Number.isInteger(id) || id <= 0)) {
        issues.push({ code: 'invalid_account_id', ruleIndex, candidateIndex })
      }

      if (!Number.isInteger(candidate.priority) || candidate.priority < 0) {
        issues.push({ code: 'invalid_priority', ruleIndex, candidateIndex, value: candidate.priority })
      }
      if (Number.isInteger(candidate.priority) && candidate.priority >= 0) {
        for (const accountID of candidate.account_ids) {
          const previousPriority = accountPriorities.get(accountID)
          if (previousPriority !== undefined && previousPriority !== candidate.priority) {
            issues.push({ code: 'account_priority_conflict', ruleIndex, candidateIndex, value: accountID })
          } else {
            accountPriorities.set(accountID, candidate.priority)
          }
        }
      }
    })
  })

  return issues
}

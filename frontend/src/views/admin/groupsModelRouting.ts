import type {
  ModelRoutingCandidate,
  ModelRoutingConfig,
  ModelRoutingRuleRow
} from '@/types'

export type ModelRoutingValidationCode =
  | 'alias_required'
  | 'duplicate_alias'
  | 'candidate_required'
  | 'model_required'
  | 'duplicate_model'
  | 'account_ids_required'
  | 'invalid_account_id'
  | 'invalid_priority'
  | 'duplicate_priority'
  | 'invalid_daily_token_limit'

export interface ModelRoutingValidationIssue {
  code: ModelRoutingValidationCode
  ruleIndex: number
  candidateIndex?: number
  value?: string | number
}

function cloneCandidate(candidate: ModelRoutingCandidate): ModelRoutingCandidate {
  return {
    model: candidate.model,
    account_ids: [...candidate.account_ids],
    priority: candidate.priority,
    daily_token_limit: candidate.daily_token_limit ?? null
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
              model: alias,
              account_ids: [...(value as number[])],
              priority: 0,
              daily_token_limit: null
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
        .map(item => ({
          ...item.candidate,
          model: item.candidate.model.trim()
        }))
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

    const models = new Set<string>()
    const priorities = new Set<number>()
    row.candidates.forEach((candidate, candidateIndex) => {
      const model = candidate.model.trim()
      if (!model) {
        issues.push({ code: 'model_required', ruleIndex, candidateIndex })
      } else if (models.has(model)) {
        issues.push({ code: 'duplicate_model', ruleIndex, candidateIndex, value: model })
      } else {
        models.add(model)
      }

      if (candidate.account_ids.length === 0) {
        issues.push({ code: 'account_ids_required', ruleIndex, candidateIndex })
      } else if (candidate.account_ids.some(id => !Number.isInteger(id) || id <= 0)) {
        issues.push({ code: 'invalid_account_id', ruleIndex, candidateIndex })
      }

      if (!Number.isInteger(candidate.priority) || candidate.priority < 0) {
        issues.push({ code: 'invalid_priority', ruleIndex, candidateIndex, value: candidate.priority })
      } else if (priorities.has(candidate.priority)) {
        issues.push({ code: 'duplicate_priority', ruleIndex, candidateIndex, value: candidate.priority })
      } else {
        priorities.add(candidate.priority)
      }

      const limit = candidate.daily_token_limit
      if (limit !== null && (!Number.isInteger(limit) || limit < 0)) {
        issues.push({
          code: 'invalid_daily_token_limit',
          ruleIndex,
          candidateIndex,
          value: limit
        })
      }
    })
  })

  return issues
}

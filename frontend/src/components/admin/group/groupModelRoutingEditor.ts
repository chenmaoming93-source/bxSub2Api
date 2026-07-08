export interface RoutingEditorAccount {
  id: number
  name: string
}

export interface RoutingEditorCandidate {
  model: string
  accounts: RoutingEditorAccount[]
  priority: number | string
  daily_token_limit: number | string | null
}

export interface RoutingEditorRule {
  alias: string
  candidates: RoutingEditorCandidate[]
}

export function createEmptyRoutingCandidate(): RoutingEditorCandidate {
  return { model: '', accounts: [], priority: 0, daily_token_limit: null }
}

export function addRoutingCandidate(rule: RoutingEditorRule): void {
  const candidate = createEmptyRoutingCandidate()
  candidate.priority = rule.candidates.length === 0
    ? 0
    : Math.max(...rule.candidates.map(item => Number(item.priority) || 0)) + 1
  rule.candidates.push(candidate)
}

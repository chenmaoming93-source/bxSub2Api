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

export interface RoutingEditorModel {
  id: string
  display_name?: string
}

export function intersectAccountModels(
  modelLists: ReadonlyArray<ReadonlyArray<RoutingEditorModel>>
): RoutingEditorModel[] {
  if (modelLists.length === 0 || modelLists.some(models => models.length === 0)) return []

  const normalizedLists = modelLists.map(models => {
    const byID = new Map<string, RoutingEditorModel>()
    for (const model of models) {
      const id = model.id.trim()
      if (!id) continue
      const existing = byID.get(id)
      if (!existing) {
        byID.set(id, model.display_name ? { id, display_name: model.display_name } : { id })
      } else if (!existing.display_name && model.display_name) {
        byID.set(id, { id, display_name: model.display_name })
      }
    }
    return byID
  })

  const [first, ...rest] = normalizedLists
  return [...first.values()]
    .filter(model => rest.every(models => models.has(model.id)))
    .map(model => {
      if (model.display_name) return model
      const displayName = rest.map(models => models.get(model.id)?.display_name).find(Boolean)
      return displayName ? { id: model.id, display_name: displayName } : model
    })
    .sort((left, right) => left.id.localeCompare(right.id))
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

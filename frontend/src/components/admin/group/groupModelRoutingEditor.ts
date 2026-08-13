export interface RoutingEditorAccount {
  id: number
  name: string
  concurrency?: number
  /** 该账号绑定的上游模型名（model_mapping 排序后第一个非空 key，与后端 FirstModelMappingKey 语义一致） */
  upstreamModel?: string
}

export interface RoutingEditorCandidate {
  accounts: RoutingEditorAccount[]
  priority: number | string
  maxConcurrency?: number | string | null
  accountConcurrency?: number
  allocatedConcurrency?: number
	otherAllocatedConcurrency?: number
}

export interface RoutingEditorRule {
  alias: string
  candidates: RoutingEditorCandidate[]
}

export function createEmptyRoutingCandidate(): RoutingEditorCandidate {
  return { accounts: [], priority: 0, maxConcurrency: null }
}

export function addRoutingCandidate(rule: RoutingEditorRule): void {
  const candidate = createEmptyRoutingCandidate()
  candidate.priority = rule.candidates.length === 0
    ? 0
    : Math.max(...rule.candidates.map(item => Number(item.priority) || 0)) + 1
  rule.candidates.push(candidate)
}

/**
 * 从账号 credentials 的 model_mapping 中提取路由用上游模型名。
 * 语义与后端 Account.FirstModelMappingKey 一致：取去空白后按字典序升序的第一个非空 key。
 * model_mapping 形如 { "请求模型/别名": "上游模型" }，路由绑定取 key 作为该账号的上游模型。
 */
export function extractUpstreamModel(credentials: Record<string, unknown> | undefined): string | undefined {
  const raw = credentials?.model_mapping
  if (!raw || typeof raw !== 'object') return undefined
  const keys: string[] = []
  for (const key of Object.keys(raw as Record<string, unknown>)) {
    const trimmed = key.trim()
    if (trimmed) keys.push(trimmed)
  }
  if (keys.length === 0) return undefined
  keys.sort()
  return keys[0]
}

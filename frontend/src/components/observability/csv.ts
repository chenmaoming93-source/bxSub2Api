import type { PathRow } from '@/api/admin/observability'

export function safeCSVCell(value: unknown): string {
  let text = value == null ? '' : String(value)
  if (/^[\s]*[=+\-@]/.test(text)) {
    text = `'${text}`
  }
  return `"${text.replace(/"/g, '""')}"`
}

export function pathsToCSV(rows: PathRow[]): string {
  const columns: Array<{ key: keyof PathRow; label: string }> = [
    { key: 'user_id', label: 'user_id' },
    { key: 'user_email', label: 'user_email' },
    { key: 'department', label: 'department' },
    { key: 'api_key_id', label: 'api_key_id' },
    { key: 'api_key_name', label: 'api_key_name' },
    { key: 'group_id', label: 'group_id' },
    { key: 'group_name', label: 'group_name' },
    { key: 'route_alias', label: 'route_alias' },
    { key: 'account_id', label: 'account_id' },
    { key: 'account_name', label: 'account_name' },
    { key: 'bound_model', label: 'bound_model' },
    { key: 'tokens', label: 'tokens' },
    { key: 'request_count', label: 'request_count' },
    { key: 'success_rate', label: 'success_rate' },
    { key: 'p95_first_token_ms', label: 'p95_first_token_ms' },
    { key: 'current', label: 'current_concurrency' },
    { key: 'limit', label: 'concurrency_limit' },
    { key: 'legacy_inferred', label: 'legacy_inferred' },
    { key: 'status', label: 'status' },
    { key: 'model_attribution', label: 'model_attribution' },
  ]
  const lines = [columns.map((column) => safeCSVCell(column.label)).join(',')]
  for (const row of rows) {
    lines.push(columns.map((column) => safeCSVCell(row[column.key])).join(','))
  }
  return `\uFEFF${lines.join('\r\n')}`
}

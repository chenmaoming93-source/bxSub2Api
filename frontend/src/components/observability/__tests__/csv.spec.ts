import { describe, expect, it } from 'vitest'
import type { PathRow } from '@/api/admin/observability'
import { pathsToCSV, safeCSVCell } from '../csv'

describe('observability CSV export', () => {
  it('neutralizes spreadsheet formula injection', () => {
    expect(safeCSVCell('=HYPERLINK("https://evil.example")')).toContain(`\"'=HYPERLINK`)
    expect(safeCSVCell('  +SUM(1,1)')).toContain(`\"'  +SUM`)
    expect(safeCSVCell('@cmd')).toContain(`\"'@cmd`)
    expect(safeCSVCell('-1')).toContain(`\"'-1`)
  })

  it('exports only current path rows with UTF-8 BOM and escaped values', () => {
    const row: PathRow = {
      user_id: 1,
      user_email: '=HYPERLINK("https://evil.example")',
      department: '研发部',
      api_key_id: 11,
      api_key_name: 'safe-key',
      group_id: 7,
      group_name: 'coding',
      route_alias: 'gpt-5',
      account_id: 101,
      account_name: 'claude-main',
      bound_model: 'claude-sonnet-4-5',
      tokens: 100,
      request_count: 2,
      success_rate: 1,
      p95_first_token_ms: 200,
      current: 1,
      limit: 10,
      legacy_inferred: false,
      status: 'healthy',
      model_attribution: 'derived',
    }
    const csv = pathsToCSV([row])
    expect(csv.startsWith('\uFEFF')).toBe(true)
    expect(csv).toContain(`\"'=HYPERLINK`)
    expect(csv.split('\r\n')).toHaveLength(2)
  })
})

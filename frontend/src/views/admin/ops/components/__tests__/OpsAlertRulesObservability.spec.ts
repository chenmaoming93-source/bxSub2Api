import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/ops/components/OpsAlertRulesCard.vue'),
  'utf8',
)

describe('Ops alert rule observability extensions', () => {
  it('offers model-path metrics without removing legacy metrics', () => {
    expect(source).toContain("'observability_client_failure_rate'")
    expect(source).toContain("'observability_business_limited_rate'")
    expect(source).toContain("'observability_sla_error_rate'")
    expect(source).toContain("'observability_ttft_p95_ms'")
    expect(source).toContain("'observability_concurrency_saturation'")
    expect(source).toContain("'account_error_count'")
  })

  it('requires a canonical object scope and exposes route/account filters', () => {
    expect(source).toContain("updateObservabilityFilter('route_alias'")
    expect(source).toContain("updateObservabilityFilter('account_id'")
    expect(source).toContain('observabilityObjectRequired')
  })
})

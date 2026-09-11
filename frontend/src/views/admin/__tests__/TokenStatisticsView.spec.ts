import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('configurable token statistics admin view', () => {
  const source = readFileSync(resolve(process.cwd(), 'src/views/admin/TokenStatisticsView.vue'), 'utf8')
  const selector = readFileSync(resolve(process.cwd(), 'src/components/admin/tokenstat/TokenQuotaDimensionValueSelector.vue'), 'utf8')

  it('exposes projection, quota and synchronization tabs with permission gates', () => {
    expect(source).toContain("id: 'projections'")
    expect(source).toContain("id: 'quotas'")
    expect(source).toContain("id: 'status'")
    expect(source).toContain("id: 'query'")
    expect(source).toContain("can('token_usage.manage')")
    expect(source).toContain("can('token_quota.update')")
  })

  it('renders dynamic filters, summary, trend, ranking, detail, completeness and CSV export', () => {
    expect(source).toContain('queryDimensions')
    expect(source).toContain('queryDraft.group_by')
    expect(source).toContain('真实累计汇总')
    expect(source).toContain('趋势')
    expect(source).toContain('排行榜')
    expect(source).toContain('数据仍在最终一致同步中')
    expect(source).toContain('重置快照')
    expect(source).toContain('重置后有效用量')
    expect(source).toContain('reset_snapshots_available')
    expect(source).toContain('row.reset_snapshot')
    expect(source).toContain('row.effective_value')
    expect(source).toContain('exportCSV')
    expect(source).toContain('重置快照')
    expect(source).toContain('重置后有效用量')
    expect(source).toContain('row.reset_snapshot')
    expect(source).toContain('row.effective_value')
    expect(source).toContain('reset_snapshots_available')
    expect(source).toContain("localStorage.setItem('token-stat-query-projection'")
  })

  it('explains collection and natural-period effective time without old API calls', () => {
    expect(source).toContain('历史数据不会回填')
    expect(source).toContain('立即作用于当前自然周期')
    expect(source).toContain('dynamicTokenStatisticsAPI')
    expect(source).not.toContain('modelTokenQuotas')
    expect(source).not.toContain('/admin/token-usage/')
  })

  it('offers reactivation for disabled projections', () => {
    expect(source).toContain("['PUBLISHED','DISABLED'].includes(item.status)")
    expect(source).toContain("item.status === 'DISABLED' ? '重新启用' : '启用'")
    expect(source).toContain("projectionAction(item.id, 'activate')")
  })

  it('provides a hot runtime switch for statistics and quotas', () => {
    expect(source).toContain('全局统计与限额')
    expect(source).toContain('toggleRuntime')
    expect(source).toContain('updateRuntime')
    expect(source).toContain('已有 Redis 数据仍会继续同步到 MySQL')
  })

  it('aligns natural ranges and enhances known dimensions with linked selectors', () => {
    expect(source).toContain('@change="alignNaturalQueryRange"')
    expect(source).toContain('自然周和自然月查询会自动将起止日期对齐到完整的自然周期边界')
    expect(source).toContain("['user_id', 'group_id', 'account_id', 'route_alias', 'upstream_model']")
    expect(source).toContain('usesEnhancedSelector(dimension.code)')
    expect(source).toContain("accountsAPI.getAvailableModels(accountID, {})")
    expect(source).toContain('Object.keys(group.model_routing || {})')
  })

  it('provides the same linked dimension selectors when creating quotas', () => {
    expect(source).toContain('onQuotaDimensionToggle(dimension.code)')
    expect(source).toContain('usesEnhancedSelector(dimension.code, quotaValues)')
    expect(source).toContain('enhancedDimensionOptions(dimension.code, quotaValues, quotaAccountModels)')
    expect(source).toContain('onQuotaValueChange(dimension.code)')
  })

  it('shares searchable dimension selectors and stores API key IDs', () => {
    expect(source).toContain("{ type: 'wildcard' }")
    expect(source).toContain('TokenQuotaDimensionValueSelector')
    expect(source).toContain('resetAccountModels')
    expect(selector).toContain('<Select')
    expect(selector).toContain('searchable')
    expect(selector).toContain('keyword.length < 2')
    expect(selector).toContain('}, 400)')
    expect(selector).toContain('apiKeySearchController?.abort()')
    expect(selector).toContain('version === apiKeySearchVersion')
    expect(selector).toContain("emit('update:modelValue', key.id)")
    expect(selector).toContain('key.masked_key')
  })

  it('provides a permission-gated, registry-driven quota reset form and confirmation', () => {
    expect(source).toContain('data-test="quota-reset-form"')
    expect(source).toContain("v-if=\"can('token_quota.update')\"")
    expect(source).toContain('v-for="dimension in dimensions"')
    expect(source).toContain('v-for="metric in quotaMetrics"')
    expect(source).toContain('metrics.value.filter(item => item.allow_quota)')
    expect(source).toContain("metric_code: 'total_tokens'")
    expect(source).toContain('window.confirm')
    expect(source).toContain('真实累计统计和报表不会被清零')
    expect(source).toContain('dynamicTokenStatisticsAPI.resetQuotaUsage')
  })

  it('shows all reset outcomes and encourages retry after partial success', () => {
    expect(source).toContain('限额用量已重置')
    expect(source).toContain('限额用量部分重置')
    expect(source).toContain('没有适用的已启用限额')
    expect(source).toContain('当前周期没有可重置用量')
    expect(source).toContain('可以再次点击重置')
    expect(source).toContain('resetResult.failed_count')
    expect(source).toContain('匹配到的限额规则')
    expect(source).toContain('命中的额度条目')
    expect(source).toContain('resetResult.matched_quotas')
    expect(source).toContain('resetResult.matched_entries')
    expect(source).toContain('matched_quota_ids')
  })

  it('shows the concrete dimension scope of every quota', () => {
    expect(source).toContain('<th>适用范围</th>')
    expect(source).toContain('quotaDimensionEntries(item)')
    expect(source).toContain('readableDimensionValue')
  })

  it('supports editing and safely deleting quota rules', () => {
    expect(source).toContain('@submit.prevent="createQuota"')
    expect(source).toContain('editQuota(item)')
    expect(source).toContain('<BaseDialog :show="Boolean(editingQuota)"')
    expect(source).toContain('@submit.prevent="saveQuotaEdit"')
    expect(source).toContain('dynamicTokenStatisticsAPI.updateQuota')
    expect(source).toContain('dynamicTokenStatisticsAPI.deleteQuota')
    expect(source).toContain('<ConfirmDialog')
    expect(source).toContain('周期和维度范围属于统计口径')
  })
})

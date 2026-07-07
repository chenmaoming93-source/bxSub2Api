import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('token usage admin views', () => {
  for (const view of ['ModelTokenUsageView.vue', 'RouteTokenUsageView.vue', 'UserModelTokenUsageView.vue']) {
    it(`${view} renders inside the shared admin layout`, () => {
      const source = readFileSync(resolve(process.cwd(), 'src/views/admin/token-usage', view), 'utf8')
      expect(source).toContain("import AppLayout from '@/components/layout/AppLayout.vue'")
      expect(source).toContain('<AppLayout>')
      expect(source).toContain('</AppLayout>')
      expect(source).toContain('useI18n')
      expect(source).toContain("import Pagination from '@/components/common/Pagination.vue'")
      expect(source).toContain('<Pagination')
      expect(source).toContain('@update:page=')
      expect(source).toContain('@update:page-size=')
      expect(source).not.toMatch(/title="(?:Global Model|Route Token|User Model)/)
      expect(source).not.toMatch(/>\s*Query\s*</)
    })
  }

  it('keeps non-date report filters independent and enabled', () => {
    const route = readFileSync(resolve(process.cwd(), 'src/views/admin/token-usage/RouteTokenUsageView.vue'), 'utf8')
    const user = readFileSync(resolve(process.cwd(), 'src/views/admin/token-usage/UserModelTokenUsageView.vue'), 'utf8')
    expect(route).not.toContain(':disabled="!selectedGroup"')
    expect(route).not.toContain(':disabled="!selectedRoute"')
    expect(route).toContain('params.route_alias = routeSearch.value.trim()')
    expect(user).not.toContain(':disabled="!report.targetId.value"')
    expect(user).toContain('params.model = modelSearch.value.trim()')
    expect(user).toContain('v-model="includeDeleted"')
    expect(user).toContain('include_deleted: includeDeleted.value')
  })

  it('sends selected sorting to the backend on every report', () => {
    for (const view of ['ModelTokenUsageView.vue', 'RouteTokenUsageView.vue', 'UserModelTokenUsageView.vue']) {
      const source = readFileSync(resolve(process.cwd(), 'src/views/admin/token-usage', view), 'utf8')
      expect(source).toContain('sort_by: sortBy.value')
      expect(source).toContain('sort_order: sortOrder.value')
    }
  })
})

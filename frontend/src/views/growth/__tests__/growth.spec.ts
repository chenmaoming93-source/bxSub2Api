import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const read = (path: string) => readFileSync(resolve(process.cwd(), path), 'utf8')

describe('growth pages', () => {
  it('registers admin and user routes with page permissions', () => {
    expect(read('src/router/index.ts')).toContain("path: '/admin/growth'")
    expect(read('src/router/index.ts')).toContain("path: '/growth'")
    expect(read('src/rbac/permissionMatrix.ts')).toContain("path: '/admin/growth', permission: 'growth.read'")
    expect(read('src/rbac/permissionMatrix.ts')).toContain("path: '/growth', permission: 'growth.self.read'")
    expect(read('src/router/index.ts')).toContain("to.path === '/admin/growth' || to.path === '/growth'")
    expect(read('src/views/growth/AdminGrowthView.vue')).toContain('v-if="!canRead"')
    expect(read('src/views/growth/UserGrowthView.vue')).toContain('v-if="!canRead"')
  })

  it('keeps Web check-in and completion actions absent', () => {
    const source = read('src/views/growth/UserGrowthView.vue')
    const checkin = read('src/components/growth/CheckinCard.vue')
    const locales = read('src/i18n/locales/growth.zh.ts')
    expect(source).not.toContain('/user/growth/checkins')
    expect(source).not.toContain('completion-requests')
    expect(checkin).not.toContain('<button')
    expect(checkin).not.toContain('checkins')
    expect(locales).toContain('签到请前往 AIWork')
  })

  it('keeps product and task admin actions wired to APIs', () => {
    expect(read('src/api/growth/admin.ts')).toContain("'/admin/growth/tasks'")
    expect(read('src/api/growth/admin.ts')).toContain("'/admin/growth/products'")
    expect(read('src/api/growth/admin.ts')).toContain("'/admin/growth/exchanges'")
  })

  it('uses formal dialogs instead of browser prompt flows in growth views', () => {
    const files = [
      'src/views/growth/admin/BudgetTab.vue',
      'src/views/growth/admin/TasksTab.vue',
      'src/views/growth/admin/ProductsTab.vue',
      'src/views/growth/admin/OrdersTab.vue',
      'src/views/growth/admin/UserPointsTab.vue',
      'src/views/growth/UserGrowthView.vue',
    ]
    for (const file of files) {
      const source = read(file)
      expect(source).not.toContain('window.prompt')
      expect(source).not.toContain('window.confirm')
      expect(source).not.toMatch(/\bconfirm\(/)
    }
  })

  it('keeps delete operations out of the task workbench', () => {
    const source = read('src/views/growth/admin/TasksTab.vue')
    expect(source).not.toContain("'delete'")
    expect(source).not.toMatch(/\.delete\(/)
    expect(source).not.toContain('DELETE')
  })
})

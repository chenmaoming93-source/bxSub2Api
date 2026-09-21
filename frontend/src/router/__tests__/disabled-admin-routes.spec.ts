// @vitest-environment jsdom

import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

import router from '@/router'

const disabledAdminPaths = [
  '/admin/default-group-routing',
  '/admin/channels',
  '/admin/channels/pricing',
  '/admin/channels/monitor',
  '/admin/subscriptions',
  '/admin/announcements',
  '/admin/proxies',
  '/admin/redeem',
  '/admin/promo-codes',
]

const sidebarSource = readFileSync(resolve(process.cwd(), 'src/components/layout/AppSidebar.vue'), 'utf8')
const permissionMatrixSource = readFileSync(resolve(process.cwd(), 'src/rbac/permissionMatrix.ts'), 'utf8')
const settingsSource = readFileSync(resolve(process.cwd(), 'src/views/admin/SettingsView.vue'), 'utf8')
const groupsSource = readFileSync(resolve(process.cwd(), 'src/views/admin/GroupsView.vue'), 'utf8')

describe('disabled legacy admin routes', () => {
  it.each(disabledAdminPaths)('%s resolves to the catch-all 404 route', (path) => {
    const resolved = router.resolve(path)

    expect(resolved.name).toBe('NotFound')
    expect(resolved.matched).toHaveLength(1)
    expect(resolved.matched[0]?.path).toBe('/:pathMatch(.*)*')
  })

  it('keeps active V2 routes registered', () => {
    expect(router.resolve('/admin/dashboard').name).toBe('AdminDashboard')
    expect(router.resolve('/admin/usage').name).toBe('AdminUsage')
    expect(router.resolve('/admin/ops').name).toBe('AdminOps')
    expect(router.resolve('/dashboard').name).toBe('Dashboard')
    expect(router.resolve('/usage').name).toBe('Usage')
  })

  it('removes disabled entries from the admin sidebar and permission matrix', () => {
    for (const path of disabledAdminPaths) {
      expect(sidebarSource).not.toContain(`path: '${path}'`)
      expect(permissionMatrixSource).not.toContain(`path: '${path}'`)
    }
  })

  it('removes settings links to disabled channel pages', () => {
    expect(settingsSource).not.toContain('to="/admin/channels/monitor"')
    expect(settingsSource).not.toContain('to="/admin/channels/pricing"')
  })

  it('keeps GroupModelRoutingEditor wired into group management', () => {
    expect(groupsSource).toContain("import GroupModelRoutingEditor from")
    expect(groupsSource).toContain('<GroupModelRoutingEditor')
  })
})

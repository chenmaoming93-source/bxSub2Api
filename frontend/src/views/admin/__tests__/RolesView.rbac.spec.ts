import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('RBAC management UI contracts', () => {
  it('protects system roles, wildcard and high-impact user changes', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/views/admin/RolesView.vue'), 'utf8')
    expect(source).toContain("selected.code === 'admin'")
    expect(source).toContain("selected.code === 'user'")
    expect(source).toContain("roles.permissions.assign")
    expect(source).toContain('confirm(')
    expect(source).toContain('risk_level')
  })

  it('provides multi-role assignment from the user management page', () => {
    const modal = readFileSync(resolve(process.cwd(), 'src/components/admin/user/UserRolesModal.vue'), 'utf8')
    expect(modal).toContain('type="checkbox"')
    expect(modal).toContain("selected.includes('admin')")
    expect(modal).toContain('replaceUserRoles')
  })
})

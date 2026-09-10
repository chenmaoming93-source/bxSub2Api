import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

describe('security check logs admin view', () => {
  const source = readFileSync(resolve(process.cwd(), 'src/views/admin/SecurityCheckLogsView.vue'), 'utf8')
  const sidebar = readFileSync(resolve(process.cwd(), 'src/components/layout/AppSidebar.vue'), 'utf8')
  const router = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
  const groupsView = readFileSync(resolve(process.cwd(), 'src/views/admin/GroupsView.vue'), 'utf8')

  it('renders a standalone page with status and decision columns', () => {
    expect(source).toContain('安全检查日志')
    expect(source).toContain('<BaseDialog')
    expect(source).toContain('detailOpen')
    expect(source).toContain('查看详情')
    expect(source).toContain('<th>检查状态</th>')
    expect(source).toContain('检查成功')
    expect(source).toContain('检查超时')
    expect(source).toContain('检查异常')
    expect(source).toContain('未执行')
  })

  it('shows stored exception details inside the detail dialog', () => {
    expect(source).toContain('异常信息')
    expect(source).toContain('日志保留与清理')
    expect(source).toContain('retention_days')
    expect(source).toContain('cleanup_time')
    expect(source).toContain('next_cleanup_at')
    expect(source).toContain('saveRetention')
    expect(source).toContain('detail.exception_type')
    expect(source).toContain('detail.exception_message')
  })

  it('registers the route above the token statistics menu item', () => {
    expect(router).toContain("path: '/admin/security-check-logs'")
    expect(sidebar.indexOf("path: '/admin/security-check-logs'")).toBeLessThan(sidebar.indexOf("path: '/admin/token-statistics'"))
  })

  it('removes the duplicate group-page security log entry', () => {
    expect(groupsView).not.toContain('SecurityCheckLogsModal')
    expect(groupsView).not.toContain('showSecurityLogsModal')
    expect(groupsView).not.toContain('data-test="security-check-logs"')
  })

  it('keeps risk dimensions readable while preserving protocol codes', () => {
    expect(source).toContain('危险操作与工具滥用')
    expect(source).toContain('恶意代码与网络攻击')
    expect(source).toContain('提示词注入与越狱')
    expect(source).toContain('敏感信息窃取')
    expect(source).toContain('Prompt_Injection_and_Jailbreak')
  })
})

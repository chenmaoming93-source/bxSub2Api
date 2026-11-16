import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const source = readFileSync(
  resolve(process.cwd(), 'src/views/admin/ModelObservabilityView.vue'),
  'utf8',
)
const routerSource = readFileSync(resolve(process.cwd(), 'src/router/index.ts'), 'utf8')
const heatmapSource = readFileSync(
  resolve(process.cwd(), 'src/components/observability/PressureHeatmap.vue'),
  'utf8',
)

describe('ModelObservabilityView foundation', () => {
  it('uses snapshot KPI and ranking data without client-side multipliers', () => {
    expect(source).toContain('snapshot.value?.metrics')
    expect(source).toContain('snapshot.value?.top_users')
    expect(source).not.toContain('ratioScale')
    expect(source).not.toContain('timeOfDayFactor')
  })

  it('shows accessible department and user labels and no credential fields', () => {
    expect(source).toContain('item.department')
    expect(source).toContain('item.email')
    expect(source).toContain('#${item.user_id}')
    expect(source).not.toContain('credentials')
    expect(source).not.toContain('request_body')
    expect(source).not.toContain('api_key_secret')
  })

  it('adds an independent route while preserving Dashboard and Ops', () => {
    expect(routerSource).toContain("path: '/admin/observability'")
    expect(routerSource).toContain("path: '/admin/dashboard'")
    expect(routerSource).toContain("path: '/admin/ops'")
  })

  it('provides responsive and dark-mode layout classes', () => {
    expect(source).toContain('<AppLayout>')
    expect(source).toContain("import AppLayout from '@/components/layout/AppLayout.vue'")
    expect(source).toContain('sm:grid-cols-2')
    expect(source).toContain('xl:grid-cols-7')
    expect(source).toContain('xl:grid-cols-[minmax(0,1.7fr)_minmax(300px,.9fr)]')
    expect(source).toContain('dark:bg-dark-800')
    expect(heatmapSource).toContain('overflow-x-auto')
  })

  it('renders the Demo-style shell, selector workbench and seven KPIs', () => {
    expect(source).toContain("t('admin.observability.heroEyebrow')")
    expect(source).toContain("t('admin.observability.workbenchTitle')")
    expect(source).toContain("t('admin.observability.scopeTitle')")
    expect(source).toContain("t('admin.observability.capacityAttention')")
    expect(source).toContain('autoRefresh.value')
    expect(source).toContain('avg_first_token_ms')
  })

  it('restores canonical object filters from alert deep links', () => {
    expect(source).toContain('applyRouteFilters')
    expect(source).toContain('route.query.group_id')
    expect(source).toContain("path: '/admin/observability'")
    expect(source).toContain('@open="openAlertObject"')
  })

  it('links heatmap selections to inline drilldown without changing global filters', () => {
    expect(source).toContain('@select="handleHeatmapSelection"')
    expect(source).toContain("getElementById('observability-drilldown')")
    expect(source).toContain(':focus="drilldownFocus"')
    expect(source).toContain("t('admin.observability.heatmapFocus'")
  })

  it('keeps custom time inputs visible and handles quick/custom ranges', () => {
    expect(source).toContain('type="datetime-local"')
    expect(source).toContain("t('admin.observability.customRange')")
    expect(source).toContain('selectQuickRange')
    expect(source).toContain('applyCustomTime')
    expect(source).toContain("t('admin.observability.endAfterStart')")
  })
})

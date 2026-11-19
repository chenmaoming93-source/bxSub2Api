import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import type { PathRow, PathsResponse, RelationsResponse } from '@/api/admin/observability'
import DrilldownPanel from '../DrilldownPanel.vue'

const pathRow: PathRow = {
  user_id: 1,
  user_email: 'alice@example.com',
  department: '研发部',
  api_key_id: 11,
  api_key_name: 'active-key',
  group_id: 7,
  group_name: 'coding',
  route_alias: 'gpt-5',
  account_id: 101,
  account_name: 'claude-main',
  bound_model: 'claude-sonnet-4-5',
  tokens: 500,
  request_count: 5,
  success_rate: 1,
  average_duration_ms: 0,
  average_first_token_ms: 0,
  duration_sample_count: 0,
  ttft_sample_count: 0,
  p95_first_token_ms: 0,
  error_stage: 'upstream',
  current: 2,
  limit: 10,
  legacy_inferred: true,
  status: 'healthy',
  model_attribution: 'derived',
}

const paths: PathsResponse = {
  meta: {
    as_of: '2026-11-11T00:00:00Z', window: '1h', bucket: '5m', timezone: 'Asia/Shanghai',
    sources: [], consistency: 'eventual', lag_seconds: 0, complete: false, warnings: ['partial_latency_histogram'],
  },
  total: 1,
  rows: [pathRow],
}

describe('DrilldownPanel', () => {
  it('opens a row without inventing missing TTFT samples', async () => {
    const wrapper = mount(DrilldownPanel, {
      props: {
        paths, relations: null, pathsLoading: false, relationsLoading: false,
        pathsError: '', relationsError: '', page: 1, pageSize: 20,
        sort: 'tokens', status: '', includeIdle: false, selectedPath: null,
        search: '', dimensions: [],
      },
    })
    await wrapper.get('tbody tr').trigger('click')
    expect(wrapper.emitted('openDetail')?.[0]).toEqual([pathRow])
    expect(wrapper.text()).not.toContain('0ms')
  })

  it('renders unavailable detail data and model attribution', () => {
    const wrapper = mount(DrilldownPanel, {
      props: {
        paths, relations: null, pathsLoading: false, relationsLoading: false,
        pathsError: '', relationsError: '', page: 1, pageSize: 20,
        sort: 'tokens', status: '', includeIdle: false, selectedPath: pathRow,
        search: '', dimensions: [],
      },
    })
    expect(wrapper.text()).toContain('admin.observability.legacyInferred')
    expect(wrapper.text()).toContain('admin.observability.unavailableNoSamples')
    expect(wrapper.text()).toContain('claude-sonnet-4-5')
    expect(wrapper.text()).toContain('upstream')
  })

  it('renders exactly four chain columns and applies heatmap focus without a global filter mutation', async () => {
    const relations: RelationsResponse = {
      meta: paths.meta,
      data_quality_errors: [],
      nodes: [
        { id: 'user:1', kind: 'user', label: 'alice', user_id: 1 },
        { id: 'group:7', kind: 'group', label: 'coding', group_id: 7 },
        { id: 'route:gpt-5', kind: 'route_alias', label: 'gpt-5', route_alias: 'gpt-5' },
        { id: 'account:101', kind: 'account', label: 'claude-main', account_id: 101, bound_model: 'claude-sonnet-4-5' },
      ],
      edges: [
        { from: 'user:1', to: 'group:7', kind: 'member', idle: false },
        { from: 'group:7', to: 'route:gpt-5', kind: 'route', idle: false },
        { from: 'route:gpt-5', to: 'account:101', kind: 'candidate', idle: false },
      ],
    }
    const wrapper = mount(DrilldownPanel, {
      props: {
        paths, relations, pathsLoading: false, relationsLoading: false,
        pathsError: '', relationsError: '', page: 1, pageSize: 20,
        sort: 'tokens', status: '', includeIdle: false, selectedPath: null,
        search: '', dimensions: [], focus: { kind: 'group', key: '7' }, focusContext: '热力图定位 · 7',
      },
    })
    await wrapper.findAll('[role="tab"]')[1].trigger('click')
    expect(wrapper.text()).toContain('热力图定位')
    expect(wrapper.text()).toContain('admin.observability.currentPathSummary')
    expect(wrapper.get('#drilldown-panel-chain').attributes('role')).toBe('tabpanel')
    expect(wrapper.text()).toContain('gpt-5')
    expect(wrapper.text()).toContain('claude-main')
    expect(wrapper.emitted('selectFilter')).toBeUndefined()
  })
})

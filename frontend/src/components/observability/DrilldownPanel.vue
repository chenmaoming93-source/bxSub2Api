<template>
  <section id="observability-drilldown" class="scroll-mt-4 rounded-2xl border border-gray-200 bg-white dark:border-dark-600 dark:bg-dark-800">
    <div class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-4 py-3 dark:border-dark-600">
      <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-700" role="tablist" :aria-label="t('admin.observability.drilldownTabs')">
        <button
          v-for="tab in tabs"
          :id="`drilldown-tab-${tab.value}`"
          :key="tab.value"
          type="button"
          role="tab"
          :aria-selected="activeTab === tab.value"
          :aria-controls="`drilldown-panel-${tab.value}`"
          :tabindex="activeTab === tab.value ? 0 : -1"
          class="rounded-md px-3 py-1.5 text-sm"
          :class="activeTab === tab.value ? 'bg-white shadow dark:bg-dark-600' : 'text-gray-500'"
          @click="activateTab(tab.value)"
          @keydown="handleTabKeydown($event, tab.value)"
        >
          {{ tab.label }}
        </button>
      </div>
      <div class="flex items-center gap-2 text-xs">
        <span v-if="focusContext" class="rounded-full bg-primary-50 px-2 py-1 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">{{ focusContext }}</span>
        <label class="inline-flex items-center gap-1">
          <input v-model="localIncludeIdle" type="checkbox" @change="$emit('update:includeIdle', localIncludeIdle)" />
          {{ t('admin.observability.includeIdle') }}
        </label>
        <button type="button" class="btn btn-secondary" :disabled="!paths?.rows.length" @click="exportCSV">
          {{ t('admin.observability.exportCurrentPage') }}
        </button>
      </div>
    </div>

    <div
      v-if="activeTab === 'matrix'"
      id="drilldown-panel-matrix"
      class="space-y-3 p-4"
      role="tabpanel"
      aria-labelledby="drilldown-tab-matrix"
      tabindex="0"
    >
      <div class="flex flex-wrap gap-2">
        <input :value="search" class="form-input max-w-xs dark:border-dark-600 dark:bg-dark-700 dark:text-white" :placeholder="t('admin.observability.searchPath')" @input="$emit('update:search', ($event.target as HTMLInputElement).value)" />
        <select :value="status" class="form-input dark:border-dark-600 dark:bg-dark-700 dark:text-white" @change="$emit('update:status', ($event.target as HTMLSelectElement).value)">
          <option value="">{{ t('admin.observability.allStatus') }}</option>
          <option value="healthy">{{ t('admin.observability.healthy') }}</option>
          <option value="degraded">{{ t('admin.observability.degraded') }}</option>
          <option value="error">{{ t('admin.observability.errorStatus') }}</option>
        </select>
        <select :value="sort" class="form-input dark:border-dark-600 dark:bg-dark-700 dark:text-white" @change="$emit('update:sort', ($event.target as HTMLSelectElement).value)">
          <option value="tokens">{{ t('admin.observability.tokens') }}</option>
          <option value="requests">{{ t('admin.observability.requests') }}</option>
          <option value="error_rate">{{ t('admin.observability.errorRate') }}</option>
          <option value="avg_duration">{{ t('admin.observability.avgDuration') }}</option>
        </select>
      </div>
      <div v-if="pathsError" class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300" role="alert">{{ pathsError }}</div>
      <div class="overflow-x-auto" :aria-busy="pathsLoading">
        <table class="min-w-full text-sm">
          <thead class="text-left text-xs text-gray-500">
            <tr>
              <th class="px-2 py-2">{{ t('admin.observability.user') }}</th>
              <th class="px-2 py-2">{{ t('admin.observability.apiKey') }}</th>
              <th class="px-2 py-2">{{ t('admin.observability.groupRoute') }}</th>
              <th class="px-2 py-2">{{ t('admin.observability.accountModel') }}</th>
              <th class="px-2 py-2 text-right">{{ t('admin.observability.tokens') }}</th>
              <th class="px-2 py-2 text-right">{{ t('admin.observability.requests') }}</th>
              <th class="px-2 py-2 text-right">{{ t('admin.observability.p95TTFT') }}</th>
              <th class="px-2 py-2">{{ t('admin.observability.status') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in paths?.rows || []"
              :key="`${row.user_id}-${row.api_key_id}-${row.group_id}-${row.route_alias}-${row.account_id}`"
              class="cursor-pointer border-t border-gray-100 hover:bg-gray-50 dark:border-dark-700 dark:hover:bg-dark-700"
              tabindex="0"
              @click="$emit('openDetail', row)"
              @keydown.enter="$emit('openDetail', row)"
              @keydown.space.prevent="$emit('openDetail', row)"
            >
              <td class="px-2 py-2">#{{ row.user_id }} {{ row.user_email }}<div class="text-xs text-gray-500">{{ row.department }}</div></td>
              <td class="px-2 py-2">#{{ row.api_key_id }} {{ row.api_key_name }}</td>
              <td class="px-2 py-2">{{ row.group_name || `#${row.group_id}` }}<div class="text-xs text-gray-500">{{ row.route_alias }}</div></td>
              <td class="px-2 py-2">{{ row.account_name || `#${row.account_id}` }}<div class="text-xs text-gray-500">{{ row.bound_model || t('admin.observability.unknownModel') }}</div></td>
              <td class="px-2 py-2 text-right font-mono">{{ row.tokens.toLocaleString() }}</td>
              <td class="px-2 py-2 text-right font-mono">{{ row.request_count.toLocaleString() }}</td>
              <td class="px-2 py-2 text-right font-mono">{{ row.p95_first_token_ms ? `${row.p95_first_token_ms}ms` : 'n/a' }}</td>
              <td class="px-2 py-2"><StatusBadge :status="row.status" /></td>
            </tr>
          </tbody>
        </table>
        <p v-if="pathsLoading && !paths?.rows.length" class="py-8 text-center text-sm text-gray-400">{{ t('admin.observability.loading') }}</p>
        <p v-else-if="!paths?.rows.length" class="py-8 text-center text-sm text-gray-400">{{ t('admin.observability.noMatchingPaths') }}</p>
      </div>
      <div class="flex items-center justify-between text-xs text-gray-500">
        <span>{{ t('admin.observability.pathsTotal', { count: paths?.total || 0 }) }}</span>
        <div class="flex items-center gap-2">
          <button class="btn btn-secondary" :disabled="page <= 1" @click="$emit('update:page', page - 1)">{{ t('admin.observability.previous') }}</button>
          <span>{{ t('admin.observability.pageNumber', { page }) }}</span>
          <button class="btn btn-secondary" :disabled="!paths || page * pageSize >= paths.total" @click="$emit('update:page', page + 1)">{{ t('admin.observability.next') }}</button>
        </div>
      </div>
    </div>

    <div
      v-else-if="activeTab === 'chain'"
      id="drilldown-panel-chain"
      class="space-y-4 p-4"
      role="tabpanel"
      aria-labelledby="drilldown-tab-chain"
      tabindex="0"
      :aria-busy="relationsLoading"
    >
      <div v-if="relationsError" class="rounded-lg bg-red-50 p-3 text-sm text-red-700 dark:bg-red-950/30 dark:text-red-300" role="alert">{{ relationsError }}</div>
      <div v-if="relations?.data_quality_errors.length" class="rounded-lg bg-amber-50 p-3 text-xs text-amber-800 dark:bg-amber-950/30 dark:text-amber-200">
        {{ relations.data_quality_errors.join(' · ') }}
      </div>
      <div class="rounded-xl border border-primary-200 bg-primary-50/60 px-3 py-2 dark:border-primary-900 dark:bg-primary-950/30">
        <div class="text-xs font-semibold uppercase text-primary-700 dark:text-primary-300">{{ t('admin.observability.currentPathSummary') }}</div>
        <div v-if="currentPathParts.length" class="mt-1 flex flex-wrap items-center gap-1 text-sm text-gray-800 dark:text-gray-100">
          <template v-for="(part, index) in currentPathParts" :key="`${part}-${index}`">
            <span class="rounded-md bg-white px-2 py-1 dark:bg-dark-700">{{ part }}</span>
            <span v-if="index < currentPathParts.length - 1" aria-hidden="true" class="text-primary-500">→</span>
          </template>
        </div>
        <p v-else class="mt-1 text-sm text-gray-500">{{ t('admin.observability.noCompletePath') }}</p>
      </div>
      <div class="grid gap-4 overflow-x-auto md:grid-cols-2 lg:grid-cols-4">
        <div v-for="kind in relationKinds" :key="kind" class="relative min-w-40 rounded-xl border border-gray-200 p-2 dark:border-dark-600">
          <span
            v-if="kind !== 'account'"
            class="absolute -right-3 top-1/2 hidden -translate-y-1/2 text-lg text-primary-500 lg:block"
            aria-hidden="true"
          >→</span>
          <span
            v-if="kind !== 'account'"
            class="mx-auto -mb-2 block text-center text-base text-primary-500 lg:hidden"
            aria-hidden="true"
          >↓</span>
          <div class="mb-2 text-xs font-semibold uppercase text-gray-500">{{ relationKindLabel(kind) }}</div>
          <button
            v-for="node in nodesByKind(kind)"
            :key="node.id"
            type="button"
            class="mb-1 block w-full rounded-lg border px-2 py-1.5 text-left text-xs"
            :class="selectedNodeIds[kind] === node.id ? 'border-primary-300 bg-primary-50 dark:bg-primary-900/20' : 'border-transparent hover:bg-gray-50 dark:hover:bg-dark-700'"
            @click="selectNode(node)"
          >
            <span class="block truncate">{{ node.label || node.id }}</span>
            <span v-if="node.department" class="block truncate text-gray-500">{{ node.department }}</span>
            <span v-if="node.bound_model" class="block truncate font-mono text-gray-500">{{ node.bound_model }}</span>
            <span v-if="node.route_pattern && node.route_pattern !== node.route_alias" class="block truncate text-gray-500">
              {{ t('admin.observability.pattern', { value: node.route_pattern }) }}
            </span>
          </button>
        </div>
      </div>
      <p class="text-xs text-gray-500">{{ t('admin.observability.directedEdges', { count: relations?.edges.length || 0 }) }}</p>
    </div>

    <div
      v-else
      id="drilldown-panel-dimensions"
      class="space-y-3 p-4"
      role="tabpanel"
      aria-labelledby="drilldown-tab-dimensions"
      tabindex="0"
    >
      <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-700" role="tablist" :aria-label="t('admin.observability.dimensions')">
        <button
          v-for="dimension in dimensions"
          :id="`dimension-tab-${dimension.key}`"
          :key="dimension.key"
          type="button"
          role="tab"
          :aria-selected="activeDimension === dimension.key"
          :aria-controls="`dimension-panel-${dimension.key}`"
          :tabindex="activeDimension === dimension.key ? 0 : -1"
          class="rounded-md px-3 py-1 text-xs"
          :class="activeDimension === dimension.key ? 'bg-white shadow dark:bg-dark-600' : 'text-gray-500'"
          @click="activateDimension(dimension.key)"
          @keydown="handleDimensionTabKeydown($event, dimension.key)"
        >
          {{ dimension.label }}
        </button>
      </div>
      <ol
        :id="`dimension-panel-${activeDimension}`"
        class="grid gap-2 md:grid-cols-2"
        role="tabpanel"
        :aria-labelledby="`dimension-tab-${activeDimension}`"
        tabindex="0"
      >
        <li
          v-for="row in activeDimensionRows"
          :key="row.key"
          class="flex cursor-pointer items-center justify-between rounded-lg bg-gray-50 px-3 py-2 dark:bg-dark-700"
          tabindex="0"
          @click="selectDimensionRow(row)"
          @keydown.enter="selectDimensionRow(row)"
        >
          <span class="truncate">{{ row.name }}</span>
          <span class="font-mono text-sm">{{ row.value.toLocaleString() }}</span>
        </li>
      </ol>
      <p v-if="activeDimensionRows.length === 0" class="py-8 text-center text-sm text-gray-400">{{ t('admin.observability.noDimensionData') }}</p>
    </div>

    <div
      v-if="selectedPath"
      ref="dialogRef"
      class="fixed inset-0 z-50 flex justify-end bg-black/30"
      role="dialog"
      aria-modal="true"
      aria-labelledby="path-detail-title"
      @click.self="$emit('closeDetail')"
      @keydown="trapDialogFocus"
    >
      <aside class="h-full w-full max-w-md overflow-y-auto bg-white p-5 shadow-2xl dark:bg-dark-800">
        <div class="flex items-start justify-between">
          <h2 id="path-detail-title" class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.observability.pathDetail') }}</h2>
          <button type="button" class="text-gray-500" :aria-label="t('admin.observability.close')" @click="$emit('closeDetail')">×</button>
        </div>
        <dl class="mt-4 grid grid-cols-[8rem_1fr] gap-y-3 text-sm">
          <dt class="text-gray-500">{{ t('admin.observability.user') }}</dt><dd>#{{ selectedPath.user_id }} · {{ selectedPath.user_email }} · {{ selectedPath.department }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.apiKey') }}</dt><dd>#{{ selectedPath.api_key_id }} · {{ selectedPath.api_key_name }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.group') }}</dt><dd>{{ selectedPath.group_name || `#${selectedPath.group_id}` }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.route') }}</dt><dd>{{ selectedPath.route_alias }} <span v-if="selectedPath.legacy_inferred" class="text-amber-600">{{ t('admin.observability.legacyInferred') }}</span></dd>
          <dt class="text-gray-500">{{ t('admin.observability.account') }}</dt><dd>{{ selectedPath.account_name || `#${selectedPath.account_id}` }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.boundModel') }}</dt><dd>{{ selectedPath.bound_model || t('admin.observability.unknown') }} ({{ selectedPath.model_attribution }})</dd>
          <dt class="text-gray-500">{{ t('admin.observability.tokensRequests') }}</dt><dd>{{ selectedPath.tokens.toLocaleString() }} / {{ selectedPath.request_count.toLocaleString() }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.concurrency') }}</dt><dd>{{ selectedPath.current }} / {{ selectedPath.limit || '∞' }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.avgRequestDuration') }}</dt><dd>{{ selectedPath.average_duration_ms ? `${Math.round(selectedPath.average_duration_ms)}ms` : t('admin.observability.noSampling') }} · {{ t('admin.observability.sampleCount', { count: selectedPath.duration_sample_count || 0 }) }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.avgFirstTokenSample') }}</dt><dd>{{ selectedPath.average_first_token_ms ? `${Math.round(selectedPath.average_first_token_ms)}ms` : t('admin.observability.noSampling') }} · {{ t('admin.observability.sampleCount', { count: selectedPath.ttft_sample_count || 0 }) }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.p95TTFT') }}</dt><dd>{{ selectedPath.p95_first_token_ms ? `${selectedPath.p95_first_token_ms}ms` : t('admin.observability.unavailableNoSamples') }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.previousWindowDelta') }}</dt><dd>{{ previousDeltaText(selectedPath) }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.errorStage') }}</dt><dd>{{ selectedPath.error_stage || (selectedPath.status === 'error' ? t('admin.observability.notCollected') : t('admin.observability.noError')) }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.dataTime') }}</dt><dd>{{ selectedPath.data_at ? formatTime(selectedPath.data_at) : t('admin.observability.notCollected') }}</dd>
          <dt class="text-gray-500">{{ t('admin.observability.status') }}</dt><dd><StatusBadge :status="selectedPath.status" /></dd>
        </dl>
        <div v-if="paths?.meta.warnings.length" class="mt-5 rounded-lg bg-amber-50 p-3 text-xs text-amber-800 dark:bg-amber-950/30 dark:text-amber-200">
          {{ t('admin.observability.dataQualityLabel') }}: {{ paths.meta.warnings.join(' · ') }}
        </div>
      </aside>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useObservabilityI18n } from './useObservabilityI18n'
import type {
  DimensionRow,
  PathRow,
  PathsResponse,
  RelationNode,
  RelationsResponse,
} from '@/api/admin/observability'
import { pathsToCSV } from './csv'
import StatusBadge from './StatusBadge.vue'
import { formatTime } from '@/utils/format'

const props = defineProps<{
  paths: PathsResponse | null
  relations: RelationsResponse | null
  pathsLoading: boolean
  relationsLoading: boolean
  pathsError: string
  relationsError: string
  page: number
  pageSize: number
  sort: string
  status: string
  includeIdle: boolean
  selectedPath: PathRow | null
  search: string
  dimensions: Array<{ key: string; label: string; rows: DimensionRow[] }>
  focus?: { kind: 'user' | 'group' | 'alias' | 'account'; key: string } | null
  focusContext?: string
}>()
const emit = defineEmits<{
  'update:page': [value: number]
  'update:sort': [value: string]
  'update:status': [value: string]
  'update:includeIdle': [value: boolean]
  'update:search': [value: string]
  openDetail: [row: PathRow]
  closeDetail: []
  selectFilter: [payload: { type: 'user' | 'group' | 'alias' | 'account'; id: number | string }]
}>()
const { t } = useObservabilityI18n()

type Tab = 'matrix' | 'chain' | 'dimensions'
const tabs = computed<Array<{ value: Tab; label: string }>>(() => [
  { value: 'matrix', label: t('admin.observability.pathMatrix') },
  { value: 'chain', label: t('admin.observability.callChain') },
  { value: 'dimensions', label: t('admin.observability.dimensions') },
])
const activeTab = ref<Tab>('matrix')
const activeDimension = ref('groups')
const dialogRef = ref<HTMLElement | null>(null)
const localIncludeIdle = ref(props.includeIdle)
watch(() => props.includeIdle, (value) => { localIncludeIdle.value = value })
const relationKinds = ['user', 'group', 'route_alias', 'account']
function relationKindLabel(kind: string) {
  if (kind === 'user') return t('admin.observability.user')
  if (kind === 'group') return t('admin.observability.group')
  if (kind === 'route_alias') return t('admin.observability.alias')
  return t('admin.observability.account')
}
const selectedNodeIds = ref<Record<string, string>>({})
const activeDimensionRows = computed(() =>
  props.dimensions.find((item) => item.key === activeDimension.value)?.rows || [],
)

function activateTab(tab: Tab) {
  activeTab.value = tab
}

function handleTabKeydown(event: KeyboardEvent, current: Tab) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const index = tabs.value.findIndex((tab) => tab.value === current)
  let next = index
  if (event.key === 'ArrowLeft') next = (index - 1 + tabs.value.length) % tabs.value.length
  if (event.key === 'ArrowRight') next = (index + 1) % tabs.value.length
  if (event.key === 'Home') next = 0
  if (event.key === 'End') next = tabs.value.length - 1
  activateTab(tabs.value[next].value)
  document.getElementById(`drilldown-tab-${tabs.value[next].value}`)?.focus()
}

function activateDimension(key: string) {
  activeDimension.value = key
}

function handleDimensionTabKeydown(event: KeyboardEvent, current: string) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const index = props.dimensions.findIndex((dimension) => dimension.key === current)
  let next = index
  if (event.key === 'ArrowLeft') next = (index - 1 + props.dimensions.length) % props.dimensions.length
  if (event.key === 'ArrowRight') next = (index + 1) % props.dimensions.length
  if (event.key === 'Home') next = 0
  if (event.key === 'End') next = props.dimensions.length - 1
  const dimension = props.dimensions[next]
  if (!dimension) return
  activateDimension(dimension.key)
  document.getElementById(`dimension-tab-${dimension.key}`)?.focus()
}

const currentPathParts = computed(() => {
  const parts: string[] = []
  for (const kind of relationKinds) {
    const selectedID = selectedNodeIds.value[kind]
    if (!selectedID) continue
    const node = props.relations?.nodes.find((item) => item.id === selectedID)
    if (node) parts.push(node.label || node.id)
  }
  const accountID = selectedNodeIds.value.account
  const account = props.relations?.nodes.find((item) => item.id === accountID)
  if (account?.bound_model && parts[parts.length - 1] !== account.bound_model) parts.push(account.bound_model)
  return parts
})

function nodesByKind(kind: string): RelationNode[] {
  const all = (props.relations?.nodes || []).filter((node) => node.kind === kind)
  const index = relationKinds.indexOf(kind)
  const upstreamSelected = relationKinds.slice(0, index).reverse().find((parentKind) => selectedNodeIds.value[parentKind])
  if (!upstreamSelected) return all
  const reachable = reachableNodeIDs(selectedNodeIds.value[upstreamSelected])
  return all.filter((node) => reachable.has(node.id))
}

function selectNode(node: RelationNode) {
  const index = relationKinds.indexOf(node.kind)
  if (index < 0) return
  selectedNodeIds.value[node.kind] = node.id
  for (const downstream of relationKinds.slice(index + 1)) delete selectedNodeIds.value[downstream]
}

function reachableNodeIDs(startID: string): Set<string> {
  const reachable = new Set<string>()
  const queue = [startID]
  while (queue.length) {
    const current = queue.shift()!
    for (const edge of props.relations?.edges || []) {
      if (edge.from !== current || reachable.has(edge.to)) continue
      reachable.add(edge.to)
      queue.push(edge.to)
    }
  }
  return reachable
}

function applyFocus() {
  selectedNodeIds.value = {}
  if (!props.focus) return
  activeTab.value = 'chain'
  const kind = props.focus.kind === 'alias' ? 'route_alias' : props.focus.kind
  const node = (props.relations?.nodes || []).find((item) => {
    if (item.kind !== kind) return false
    if (kind === 'user') return String(item.user_id) === props.focus?.key
    if (kind === 'group') return String(item.group_id) === props.focus?.key
    if (kind === 'route_alias') return (item.route_alias || item.route_pattern || item.label) === props.focus?.key
    return String(item.account_id) === props.focus?.key
  })
  if (node) selectedNodeIds.value[kind] = node.id
}

watch(() => props.focus, applyFocus, { immediate: true })
watch(() => props.relations, applyFocus)

function selectDimensionRow(row: DimensionRow) {
  if (activeDimension.value === 'users') emit('selectFilter', { type: 'user', id: Number(row.key) })
  if (activeDimension.value === 'groups') emit('selectFilter', { type: 'group', id: Number(row.key) })
  if (activeDimension.value === 'aliases') emit('selectFilter', { type: 'alias', id: row.key })
  if (activeDimension.value === 'accounts') emit('selectFilter', { type: 'account', id: Number(row.key) })
}

function trapDialogFocus(event: KeyboardEvent) {
  if (event.key !== 'Tab' || !dialogRef.value) return
  const focusable = Array.from(dialogRef.value.querySelectorAll<HTMLElement>(
    'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
  )).filter((element) => element.offsetParent !== null)
  if (focusable.length === 0) return
  const first = focusable[0]
  const last = focusable[focusable.length - 1]
  if (event.shiftKey && document.activeElement === first) {
    event.preventDefault()
    last.focus()
  } else if (!event.shiftKey && document.activeElement === last) {
    event.preventDefault()
    first.focus()
  }
}

function exportCSV() {
  const blob = new Blob([pathsToCSV(props.paths?.rows || [])], { type: 'text/csv;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const link = document.createElement('a')
  link.href = url
  link.download = `observability-paths-page-${props.page}.csv`
  link.click()
  URL.revokeObjectURL(url)
}

function previousDeltaText(row: PathRow) {
  if (row.previous_average_duration_ms == null && row.previous_average_first_token_ms == null) {
    return t('admin.observability.noComparison')
  }
  const duration = row.previous_average_duration_ms == null
    ? '-'
    : `${formatSigned(Math.round(row.average_duration_ms - row.previous_average_duration_ms))}ms`
  const ttft = row.previous_average_first_token_ms == null
    ? '-'
    : `${formatSigned(Math.round(row.average_first_token_ms - row.previous_average_first_token_ms))}ms`
  return `${t('admin.observability.avgDuration')} ${duration} · ${t('admin.observability.avgFirstTokenShort')} ${ttft}`
}

function formatSigned(value: number) {
  return value > 0 ? `+${value}` : String(value)
}
</script>

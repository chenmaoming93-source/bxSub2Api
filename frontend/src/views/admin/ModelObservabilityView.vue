<template>
  <AppLayout>
    <div class="space-y-5">
    <div
      v-if="!featureEnabled"
      class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-700/50 dark:bg-amber-950/30 dark:text-amber-200"
      role="status"
    >
      {{ t('admin.observability.disabled') }}
    </div>
    <section class="relative z-50 rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
      <div class="flex flex-col gap-3 xl:flex-row xl:items-end xl:justify-between">
        <div class="flex flex-wrap items-end gap-3">
          <div ref="timePickerRef" class="relative">
            <span class="mb-1 block text-xs font-medium text-gray-600 dark:text-gray-300">{{ t('admin.observability.window') }}</span>
            <button
              type="button"
              class="btn btn-secondary flex min-w-40 items-center justify-between gap-2"
              :aria-expanded="timeMenuOpen"
              aria-controls="observability-time-menu"
              @click="timeMenuOpen = !timeMenuOpen"
            >
              <span>{{ timeTriggerLabel }}</span>
              <svg class="h-4 w-4 transition-transform" :class="timeMenuOpen ? 'rotate-180' : ''" viewBox="0 0 20 20" fill="none" stroke="currentColor" stroke-width="1.8"><path d="m5 7.5 5 5 5-5"/></svg>
            </button>
            <div
              v-if="timeMenuOpen"
              id="observability-time-menu"
              class="absolute left-0 top-full z-40 mt-2 w-[330px] max-w-[calc(100vw-28px)] rounded-xl border border-gray-200 bg-white p-3 shadow-2xl dark:border-dark-600 dark:bg-dark-800"
            >
              <div class="text-xs text-gray-500">{{ t('admin.observability.quickTime') }}</div>
              <div class="mt-2 grid grid-cols-2 gap-2">
                <button
                  v-for="item in timeOptions"
                  :key="item.value"
                  type="button"
                  class="rounded-lg border px-3 py-2 text-left text-xs"
                  :class="window === item.value ? 'border-primary-200 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : 'border-gray-200 text-gray-600 dark:border-dark-600 dark:text-gray-300'"
                  :aria-pressed="window === item.value"
                  @click="selectQuickRange(item.value)"
                >{{ item.label }}</button>
              </div>
              <div class="mt-3 border-t border-gray-200 pt-3 dark:border-dark-600">
                <div class="text-xs font-medium text-gray-700 dark:text-gray-200">{{ t('admin.observability.customRange') }}</div>
                <div class="mt-2 grid grid-cols-2 gap-2">
                  <label class="field-label"><span>{{ t('admin.observability.startTime') }}</span><input v-model="customStart" type="datetime-local" class="form-input text-xs dark:border-dark-600 dark:bg-dark-700 dark:text-white" /></label>
                  <label class="field-label"><span>{{ t('admin.observability.endTime') }}</span><input v-model="customEnd" type="datetime-local" class="form-input text-xs dark:border-dark-600 dark:bg-dark-700 dark:text-white" /></label>
                </div>
                <p v-if="timeError" class="mt-2 text-xs text-red-600 dark:text-red-400" role="alert">{{ timeError }}</p>
                <div class="mt-2 flex justify-end gap-2"><button type="button" class="btn btn-secondary" @click="timeMenuOpen = false">{{ t('admin.observability.cancel') }}</button><button type="button" class="btn btn-primary" @click="applyCustomTime">{{ t('admin.observability.apply') }}</button></div>
              </div>
            </div>
          </div>
          <label class="field-label">
            <span>{{ t('admin.observability.groupBy') }}</span>
            <select v-model="groupBy" class="form-input min-w-36 dark:border-dark-600 dark:bg-dark-700 dark:text-white">
              <option value="user_id">{{ t('admin.observability.user') }}</option>
              <option value="group_id">{{ t('admin.observability.group') }}</option>
              <option value="route_alias">{{ t('admin.observability.alias') }}</option>
              <option value="account_id">{{ t('admin.observability.account') }}</option>
            </select>
          </label>
          <button type="button" class="btn btn-secondary" :aria-pressed="autoRefresh" @click="toggleAutoRefresh">
            {{ autoRefresh ? `● ${t('admin.observability.autoRefresh', { seconds: 5 })}` : `○ ${t('admin.observability.refreshPaused')}` }}
          </button>
          <button type="button" class="btn btn-secondary" @click="refreshAll">{{ t('admin.observability.refresh') }}</button>
        </div>
        <span class="text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.observability.lastRefresh') }}:
          {{ lastFreshAt ? lastFreshAt.toLocaleTimeString() : t('admin.observability.notLoaded') }}
        </span>
      </div>
    </section>

    <section class="overflow-hidden rounded-2xl border border-primary-200 bg-gradient-to-r from-primary-50 via-white to-indigo-50 p-5 shadow-sm dark:border-primary-900 dark:from-primary-950 dark:via-dark-800 dark:to-dark-800">
      <div class="flex flex-col gap-5 xl:flex-row xl:items-start xl:justify-between">
        <div class="max-w-3xl">
          <div class="flex items-center gap-2 text-sm font-medium text-primary-700 dark:text-primary-300">
            <span class="h-2 w-2 rounded-full bg-emerald-500"></span>{{ t('admin.observability.heroEyebrow') }}
          </div>
          <h1 class="mt-2 text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">{{ t('admin.observability.heroTitle') }}</h1>
          <p class="mt-2 text-sm text-gray-600 dark:text-gray-300">
            {{ t('admin.observability.heroDescription') }}
          </p>
        </div>
        <div class="grid grid-cols-2 gap-2 sm:grid-cols-4">
          <div v-for="stat in heroStats" :key="stat.label" class="min-w-28 rounded-xl border border-gray-200 bg-white px-3 py-2 dark:border-dark-600 dark:bg-dark-800">
            <span class="text-xs text-gray-500">{{ stat.label }}</span>
            <strong class="mt-1 block text-lg" :class="stat.className">{{ stat.value }}</strong>
          </div>
        </div>
      </div>
    </section>

    <section class="rounded-2xl border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-600 dark:bg-dark-800">
      <div class="mb-3">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.observability.workbenchTitle') }}</h2>
        <p class="mt-1 text-xs text-gray-500">{{ t('admin.observability.workbenchHint') }}</p>
      </div>
      <div>
        <div class="min-w-0">
          <div class="flex flex-wrap gap-2" role="tablist" :aria-label="t('admin.observability.workbenchTitle')">
            <button
              v-for="selector in selectors"
              :id="`object-dimension-tab-${selector.key}`"
              :key="selector.key"
              type="button"
              role="tab"
              :aria-selected="activeDimension === selector.key"
              :aria-controls="`object-dimension-panel-${selector.key}`"
              :tabindex="activeDimension === selector.key ? 0 : -1"
              class="rounded-lg border px-3 py-1.5 text-sm"
              :class="activeDimension === selector.key ? 'border-primary-200 bg-primary-50 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300' : 'border-gray-300 text-gray-600 dark:border-dark-600 dark:text-gray-300'"
              @click="activateDimensionSelector(selector.key)"
              @keydown="handleDimensionSelectorKeydown($event, selector.key)"
            >
              {{ selector.label }} <span class="ml-1 rounded-full bg-white/70 px-1.5 text-xs dark:bg-dark-700">{{ selector.options.length }}</span>
            </button>
          </div>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <input v-model="dimensionSearch" type="search" class="form-input min-w-56 flex-1 dark:border-dark-600 dark:bg-dark-700 dark:text-white" :placeholder="t('admin.observability.searchDimension')" />
            <button type="button" class="btn btn-secondary" @click="selectAllActive">{{ t('admin.observability.selectAllScope') }}</button>
            <button type="button" class="btn btn-secondary" @click="clearActiveDimension">{{ t('admin.observability.clearDimension') }}</button>
            <label class="inline-flex items-center gap-2 text-xs text-gray-600 dark:text-gray-300">
              <input v-model="highPressureOnly" type="checkbox" class="rounded border-gray-300 text-primary-600" />{{ t('admin.observability.highPressureOnlyLabel') }}
            </label>
            <button type="button" class="btn btn-primary" :disabled="!hasPendingSelectionChanges" @click="confirmObjectSelection">{{ t('admin.observability.applySelection') }}</button>
          </div>
          <div
            :id="`object-dimension-panel-${activeDimension}`"
            class="mt-3 grid max-h-48 gap-2 overflow-y-auto sm:grid-cols-2 2xl:grid-cols-3"
            role="tabpanel"
            :aria-labelledby="`object-dimension-tab-${activeDimension}`"
            tabindex="0"
          >
            <label
              v-for="option in filteredActiveOptions"
              :key="option.value"
              class="flex cursor-pointer items-center gap-2 rounded-xl border px-3 py-2 text-sm"
              :class="activeSelector?.selected(option.value) ? 'border-primary-200 bg-primary-50 dark:bg-primary-900/20' : 'border-gray-200 dark:border-dark-600'"
            >
              <input type="checkbox" class="rounded border-gray-300 text-primary-600" :checked="activeSelector?.selected(option.value)" @change="activeSelector?.toggle(option.value)" />
              <span class="min-w-0 flex-1 truncate text-gray-700 dark:text-gray-200">{{ option.label }}</span>
              <span v-if="option.detail" class="max-w-28 truncate text-xs text-gray-400">{{ option.detail }}</span>
            </label>
            <p v-if="filteredActiveOptions.length === 0" class="col-span-full py-6 text-center text-sm text-gray-400">{{ t('admin.observability.noMatchingObjects') }}</p>
          </div>
          <div class="mt-3 flex flex-wrap items-center gap-2">
            <button v-for="chip in activeFilterChips" :key="chip.key" type="button" class="rounded-full bg-primary-50 px-3 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300" @click="removeAppliedFilter(chip.type, chip.value)">
              {{ chip.label }} ×
            </button>
            <button type="button" class="text-xs text-gray-500 underline" @click="clearAppliedFilters">{{ t('admin.observability.clearAll') }}</button>
          </div>
        </div>
      </div>
    </section>

    <section v-if="capacityAttention" class="rounded-xl border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-700/50 dark:bg-amber-950/30 dark:text-amber-200">
      <b>{{ t('admin.observability.capacityAttention') }}</b>
      <span class="ml-2">{{ capacityAttention }}</span>
    </section>

    <section>
      <div v-if="snapshotError" class="mb-3 rounded-xl border border-red-200 bg-red-50 p-3 text-sm text-red-700 dark:border-red-800 dark:bg-red-950/30 dark:text-red-300" role="alert">{{ snapshotError }}</div>
      <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-4 xl:grid-cols-8" :aria-busy="snapshotLoading">
        <article v-for="metric in kpis" :key="metric.label" class="rounded-2xl border border-gray-200 bg-white p-3 shadow-sm dark:border-dark-600 dark:bg-dark-800">
          <div class="flex items-center gap-2 text-xs text-gray-500"><span class="h-2 w-2 rounded-full" :class="metric.dot"></span>{{ metric.label }}</div>
          <div class="mt-2 text-2xl font-semibold text-gray-900 dark:text-white">{{ metric.value }}</div>
          <Sparkline class="mt-1" :points="metric.points" :color="metric.color" />
          <div class="mt-1 truncate text-xs text-gray-500">{{ metric.detail }}</div>
        </article>
        <article v-if="snapshotLoading && !snapshot" class="h-28 animate-pulse rounded-2xl bg-gray-100 dark:bg-dark-700">
          <span class="sr-only">{{ t('admin.observability.loading') }}</span>
        </article>
      </div>
    </section>

    <section class="grid min-w-0 gap-4 xl:grid-cols-[minmax(0,1.7fr)_minmax(320px,.8fr)]">
      <div class="min-w-0 space-y-4">
        <div class="space-y-2">
          <div class="flex flex-wrap items-center justify-end gap-2">
            <div class="inline-flex rounded-lg bg-gray-100 p-1 text-xs dark:bg-dark-700">
              <button
                v-for="item in tokenMetricOptions"
                :key="item.value"
                type="button"
                class="rounded-md px-3 py-1"
                :class="tokenMetric === item.value ? 'bg-white shadow dark:bg-dark-600' : 'text-gray-500'"
                :aria-pressed="tokenMetric === item.value"
                @click="tokenMetric = item.value"
              >{{ item.label }}</button>
            </div>
          </div>
          <TokenTrendChart
            :labels="chartLabels"
            :response="trend"
            :concurrency="concurrencyOverlay"
            :token-metric="tokenMetric"
            :group-by="groupBy"
            :dimension-labels="trendDimensionLabels"
            :loading="trendLoading"
            :error="trendError"
          />
        </div>
        <div class="space-y-2">
          <div class="flex flex-wrap items-center justify-end gap-2">
            <div class="inline-flex rounded-lg bg-gray-100 p-1 text-xs dark:bg-dark-700">
              <button
                v-for="item in qualityModeOptions"
                :key="item.value"
                type="button"
                class="rounded-md px-3 py-1"
                :class="qualityMode === item.value ? 'bg-white shadow dark:bg-dark-600' : 'text-gray-500'"
                :aria-pressed="qualityMode === item.value"
                @click="qualityMode = item.value"
              >{{ item.label }}</button>
            </div>
          </div>
          <QualityTrendChart :labels="chartLabels" :response="latency" :mode="qualityMode" :loading="trendLoading" :error="latencyError" />
        </div>
      </div>
      <div class="min-w-0 space-y-4">
        <PressureRanking :accounts="visiblePressureAccounts" :candidates="visiblePressureCandidates" @select="selectAccount" />
        <ConcurrencyCapacityCard :capacity="snapshot?.capacity" />
        <QualityTopPeriods :response="latency" :loading="trendLoading" :error="latencyError" />
      </div>
    </section>

    <section class="min-w-0">
      <div class="min-w-0 space-y-2">
        <div class="flex flex-wrap items-center justify-end gap-2">
          <div class="inline-flex rounded-lg bg-gray-100 p-1 text-xs dark:bg-dark-700">
            <button
              v-for="item in heatmapDimensionOptions"
              :key="item.value"
              type="button"
              class="rounded-md px-3 py-1"
              :class="heatmapGroupBy === item.value ? 'bg-white shadow dark:bg-dark-600' : 'text-gray-500'"
              :aria-pressed="heatmapGroupBy === item.value"
              @click="heatmapGroupBy = item.value"
            >{{ item.label }}</button>
          </div>
          <label class="field-label">
            <span>{{ t('admin.observability.heatmapValue') }}</span>
            <select v-model="heatmapValueKind" class="form-input min-w-44 dark:border-dark-600 dark:bg-dark-700 dark:text-white">
              <option value="sampled_concurrency">{{ t('admin.observability.sampledConcurrency') }}</option>
              <option value="token_rate">{{ t('admin.observability.tokenRate') }}</option>
              <option value="error_rate">{{ t('admin.observability.errorRate') }}</option>
            </select>
          </label>
        </div>
        <PressureHeatmap
          :response="heatmap"
          :value-kind="heatmapValueKind"
          :single-account-filtered="filters.accountIds.length === 1"
          :row-labels="heatmapRowLabels"
          :value-kind-help="heatmapValueKindHelp"
          :error="heatmapError"
          @select="handleHeatmapSelection"
        />
      </div>
    </section>

    <DrilldownPanel
      :paths="drilldown.paths.value"
      :relations="drilldown.relations.value"
      :paths-loading="drilldown.pathsLoading.value"
      :relations-loading="drilldown.relationsLoading.value"
      :paths-error="drilldown.pathsError.value"
      :relations-error="drilldown.relationsError.value"
      :page="drilldown.page.value"
      :page-size="drilldown.pageSize.value"
      :sort="drilldown.sort.value"
      :status="drilldown.status.value"
      :include-idle="drilldown.includeIdle.value"
      :selected-path="drilldown.selectedPath.value"
      :search="search"
      :dimensions="dimensionTabs"
      :focus="drilldownFocus"
      :focus-context="drilldownFocusContext"
      @update:page="drilldown.page.value = $event"
      @update:sort="drilldown.sort.value = $event"
      @update:status="drilldown.status.value = $event"
      @update:includeIdle="drilldown.includeIdle.value = $event"
      @update:search="search = $event"
      @open-detail="drilldown.openDetail"
      @close-detail="drilldown.closeDetail"
      @select-filter="applyDrilldownFilter"
    />
    <section
      v-if="snapshot?.meta.warnings.length"
      class="rounded-2xl border border-amber-200 bg-amber-50 p-4 text-sm text-amber-900 dark:border-amber-700/50 dark:bg-amber-950/30 dark:text-amber-200"
      role="status"
      data-testid="data-quality-banner"
    >
      <div class="font-medium">{{ t('admin.observability.dataQuality') }}</div>
      <div class="mt-1 flex flex-wrap gap-2">
        <span
          v-for="warning in snapshot.meta.warnings"
          :key="warning"
          class="rounded-full bg-amber-100 px-2 py-0.5 text-xs dark:bg-amber-900/40"
          :title="warning"
        >{{ warningLabel(warning) }}</span>
      </div>
      <div class="mt-2 text-xs">{{ t('admin.observability.lag') }}: {{ snapshot.meta.lag_seconds }}s · {{ snapshot.meta.complete ? t('admin.observability.complete') : t('admin.observability.partial') }}</div>
    </section>
    <section
      v-if="snapshot"
      class="rounded-2xl border border-gray-200 bg-white px-4 py-3 text-xs text-gray-600 dark:border-dark-600 dark:bg-dark-800 dark:text-gray-300"
      data-testid="observability-data-contract"
    >
      <div class="font-medium text-gray-800 dark:text-gray-100">{{ t('admin.observability.dataSourceAndScope') }}</div>
      <dl class="mt-2 grid gap-2 sm:grid-cols-2 xl:grid-cols-4">
        <div><dt class="text-gray-400">{{ t('admin.observability.timezoneLabel') }}</dt><dd class="mt-0.5 font-mono">{{ snapshot.meta.timezone }}</dd></div>
        <div><dt class="text-gray-400">{{ t('admin.observability.sourcesLabel') }}</dt><dd class="mt-0.5 font-mono">{{ snapshot.meta.sources.join(', ') || '-' }}</dd></div>
        <div><dt class="text-gray-400">{{ t('admin.observability.consistencyLabel') }}</dt><dd class="mt-0.5">{{ snapshot.meta.consistency === 'eventual' ? t('admin.observability.consistencyEventual') : snapshot.meta.consistency }}</dd></div>
        <div><dt class="text-gray-400">{{ t('admin.observability.coverageLabel') }}</dt><dd class="mt-0.5">{{ snapshot.meta.coverage === 'full' ? t('admin.observability.coverageFull') : snapshot.meta.coverage || '-' }}</dd></div>
      </dl>
    </section>
    <ObservabilityAlertSummary @open="openAlertObject" />
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import AppLayout from '@/components/layout/AppLayout.vue'
import { useObservabilityI18n } from '@/components/observability/useObservabilityI18n'
import type {
  NamedFilterOption,
  AccountFilterOption,
  UserFilterOption,
  ObservabilityFilters,
  ObservabilityGroupBy,
  ObservabilityWindow,
  SnapshotFilterOptions,
} from '@/api/admin/observability'
import { useObservability } from '@/composables/useObservability'
import TokenTrendChart from '@/components/observability/TokenTrendChart.vue'
import QualityTrendChart from '@/components/observability/QualityTrendChart.vue'
import PressureRanking from '@/components/observability/PressureRanking.vue'
import PressureHeatmap from '@/components/observability/PressureHeatmap.vue'
import DrilldownPanel from '@/components/observability/DrilldownPanel.vue'
import { useObservabilityDrilldown } from '@/composables/useObservabilityDrilldown'
import ObservabilityAlertSummary from '@/components/observability/ObservabilityAlertSummary.vue'
import QualityTopPeriods from '@/components/observability/QualityTopPeriods.vue'
import ConcurrencyCapacityCard from '@/components/observability/ConcurrencyCapacityCard.vue'
import Sparkline from '@/components/observability/Sparkline.vue'
import { isHighPressure, statusFor, OBSERVABILITY_STATUS_THRESHOLDS } from '@/components/observability/status'

const { t } = useObservabilityI18n()
const route = useRoute()
const router = useRouter()
const {
  filters,
  window,
  rangeStart,
  rangeEnd,
  bucket,
  groupBy,
  search,
  highPressureOnly,
  autoRefresh,
  snapshot,
  trend,
  latency,
  heatmap,
  concurrencyHeatmap,
  heatmapValueKind,
  heatmapGroupBy,
  featureEnabled,
  snapshotLoading,
  trendLoading,
  snapshotError,
  trendError,
  latencyError,
  heatmapError,
  lastFreshAt,
  visiblePressureAccounts,
  visiblePressureCandidates,
  activeFilterChips,
  concurrencyOverlay,
  refreshAll,
  clearFilters,
  removeFilter,
} = useObservability()

interface SelectorOption {
  value: number | string
  label: string
  detail?: string
}

type SelectorKey = 'users' | 'groups' | 'aliases' | 'accounts'
interface PendingSelections {
  users: number[]
  groups: number[]
  aliases: string[]
  accounts: number[]
}
const activeDimension = ref<SelectorKey>('groups')
const dimensionSearch = ref('')
const pendingSelections = reactive<PendingSelections>({
  users: [],
  groups: [],
  aliases: [],
  accounts: [],
})
const knownFilterOptions = reactive<SnapshotFilterOptions>({
  users: [],
  groups: [],
  aliases: [],
  accounts: [],
})
const tokenMetric = ref<'total' | 'input' | 'output' | 'cache'>('total')
const qualityMode = ref<'all' | 'duration' | 'ttft' | 'failure'>('all')
const timeMenuOpen = ref(false)
const timeError = ref('')
const customStart = ref('')
const customEnd = ref('')
const timeOptions = computed(() => [
  { value: 'today' as const, label: t('admin.observability.quickToday') },
  { value: 'yesterday' as const, label: t('admin.observability.quickYesterday') },
  { value: '1h' as const, label: t('admin.observability.quick1h') },
  { value: '6h' as const, label: t('admin.observability.quick6h') },
  { value: '24h' as const, label: t('admin.observability.quick24h') },
  { value: '7d' as const, label: t('admin.observability.quick7d') },
])
const timeTriggerLabel = computed(() => {
  if (window.value === 'custom' && rangeStart.value && rangeEnd.value) {
    return `${rangeStart.value.replace('T', ' ')} ${t('admin.observability.rangeTo')} ${rangeEnd.value.replace('T', ' ')}`
  }
  return timeOptions.value.find((item) => item.value === window.value)?.label || t('admin.observability.customRangeFallback')
})
const heatmapFocus = ref<null | {
  kind: 'user' | 'group' | 'alias' | 'account'
  key: string
  column: string
  actual?: number
  limit?: number
}>(null)
const tokenMetricOptions = computed(() => [
  { value: 'total' as const, label: t('admin.observability.totalTokenMetric') },
  { value: 'input' as const, label: t('admin.observability.input') },
  { value: 'output' as const, label: t('admin.observability.output') },
  { value: 'cache' as const, label: t('admin.observability.cache') },
])
const qualityModeOptions = computed(() => [
  { value: 'all' as const, label: t('admin.observability.allMetrics') },
  { value: 'duration' as const, label: t('admin.observability.avgDuration') },
  { value: 'ttft' as const, label: t('admin.observability.avgFirstTokenShort') },
  { value: 'failure' as const, label: t('admin.observability.failureRateShort') },
])
const heatmapDimensionOptions = computed(() => [
  { value: 'user_id' as const, label: t('admin.observability.user') },
  { value: 'group_id' as const, label: t('admin.observability.group') },
  { value: 'route_alias' as const, label: t('admin.observability.alias') },
  { value: 'account_id' as const, label: t('admin.observability.account') },
])
const chartLabels = computed(() => trend.value?.labels?.length ? trend.value.labels : latency.value?.labels || [])

function mergeFilterOptions<T>(
  existing: T[],
  incoming: T[],
  keyOf: (item: T) => string | number,
): T[] {
  const merged = [...existing]
  for (const item of incoming) {
    const index = merged.findIndex((candidate) => keyOf(candidate) === keyOf(item))
    if (index >= 0) merged[index] = item
    else merged.push(item)
  }
  return merged
}

watch(snapshot, (current) => {
  if (!current) return
  const options = current.filter_options
  const optionUsers = Array.isArray(options?.users) ? options.users : []
  const optionGroups = Array.isArray(options?.groups) ? options.groups : []
  const optionAliases = Array.isArray(options?.aliases) ? options.aliases : []
  const optionAccounts = Array.isArray(options?.accounts) ? options.accounts : []
  const topUsers = Array.isArray(current.top_users) ? current.top_users : []
  const topGroups = Array.isArray(current.top_groups) ? current.top_groups : []
  const topAliases = Array.isArray(current.top_aliases) ? current.top_aliases : []
  const topAccounts = Array.isArray(current.top_accounts) ? current.top_accounts : []
  const pressureAccounts = Array.isArray(current.pressure?.accounts) ? current.pressure.accounts : []
  const pressureCandidates = Array.isArray(current.pressure?.candidates) ? current.pressure.candidates : []
  const pressureRows = [...pressureAccounts, ...pressureCandidates]
  const fallbackUsers: UserFilterOption[] = topUsers
    .map((row) => Number(row.key))
    .filter((id) => Number.isFinite(id) && id > 0)
    .map((id) => {
      const row = topUsers.find((candidate) => Number(candidate.key) === id)
      return { user_id: id, email: row?.name || `#${id}`, department: '' }
    })
  const fallbackGroups: NamedFilterOption[] = [
    ...topGroups
      .map((row) => ({ id: Number(row.key), name: row.name || `#${row.key}` }))
      .filter((item) => Number.isFinite(item.id) && item.id > 0),
    ...pressureRows
      .filter((row) => (row.group_id || 0) > 0 && Boolean(row.group_name))
      .map((row) => ({ id: Number(row.group_id), name: String(row.group_name) })),
  ]
  const fallbackAliases: NamedFilterOption[] = [
    ...topAliases.map((row) => ({ id: 0, name: row.name || row.key })),
    ...pressureRows
      .filter((row) => Boolean(row.route_alias))
      .map((row) => ({ id: 0, name: String(row.route_alias) })),
  ]
  const fallbackAccounts: AccountFilterOption[] = [
    ...topAccounts
      .map((row) => ({
        account_id: Number(row.key),
        name: row.name || `#${row.key}`,
        bound_model: '',
      }))
      .filter((item) => Number.isFinite(item.account_id) && item.account_id > 0),
    ...pressureAccounts
      .map((row) => ({
        account_id: Number(row.account_id),
        name: row.account_name || `#${row.account_id}`,
        bound_model: '',
      }))
      .filter((item) => Number.isFinite(item.account_id) && item.account_id > 0),
  ]
  knownFilterOptions.users = mergeFilterOptions(knownFilterOptions.users, [...optionUsers, ...fallbackUsers], (item) => item.user_id)
  knownFilterOptions.groups = mergeFilterOptions(knownFilterOptions.groups, [...optionGroups, ...fallbackGroups], (item) => item.id)
  knownFilterOptions.aliases = mergeFilterOptions(knownFilterOptions.aliases, [...optionAliases, ...fallbackAliases], (item) => item.name)
  knownFilterOptions.accounts = mergeFilterOptions(knownFilterOptions.accounts, [...optionAccounts, ...fallbackAccounts], (item) => item.account_id)
}, { immediate: true })

const selectors = computed<Array<{
  key: SelectorKey
  label: string
  options: SelectorOption[]
  selected: (value: number | string) => boolean
  toggle: (value: number | string) => void
}>>(() => {
  const options = knownFilterOptions
  return [
    {
      key: 'users',
      label: t('admin.observability.user'),
      options: (options?.users || []).map((item: UserFilterOption): SelectorOption => ({
        value: item.user_id,
        label: `#${item.user_id} · ${item.email || t('admin.observability.notLoaded')}`,
        detail: item.department || t('admin.observability.noDepartment'),
      })),
      selected: (value: number | string) => pendingSelections.users.includes(Number(value)),
      toggle: (value: number | string) => togglePendingSelection('users', Number(value)),
    },
    {
      key: 'groups',
      label: t('admin.observability.group'),
      options: (options?.groups || []).map((item: NamedFilterOption): SelectorOption => ({
        value: item.id,
        label: item.name,
      })),
      selected: (value: number | string) => pendingSelections.groups.includes(Number(value)),
      toggle: (value: number | string) => togglePendingSelection('groups', Number(value)),
    },
    {
      key: 'aliases',
      label: t('admin.observability.alias'),
      options: (options?.aliases || []).map((item: NamedFilterOption): SelectorOption => ({
        value: item.name,
        label: item.name,
      })),
      selected: (value: number | string) => pendingSelections.aliases.includes(String(value)),
      toggle: (value: number | string) => togglePendingSelection('aliases', String(value)),
    },
    {
      key: 'accounts',
      label: t('admin.observability.account'),
      options: (options?.accounts || []).map((item: AccountFilterOption): SelectorOption => ({
        value: item.account_id,
        label: item.name || `#${item.account_id}`,
        detail: item.bound_model || t('admin.observability.unknownModel'),
      })),
      selected: (value: number | string) => pendingSelections.accounts.includes(Number(value)),
      toggle: (value: number | string) => togglePendingSelection('accounts', Number(value)),
    },
  ]
})

const activeSelector = computed(() => selectors.value.find((selector) => selector.key === activeDimension.value))

function togglePendingSelection(key: SelectorKey, value: number | string) {
  switch (key) {
  case 'users':
    pendingSelections.users = toggleValue(pendingSelections.users, Number(value))
    break
  case 'groups':
    pendingSelections.groups = toggleValue(pendingSelections.groups, Number(value))
    break
  case 'aliases':
    pendingSelections.aliases = toggleValue(pendingSelections.aliases, String(value))
    break
  case 'accounts':
    pendingSelections.accounts = toggleValue(pendingSelections.accounts, Number(value))
    break
  }
}

function toggleValue<T>(values: T[], value: T): T[] {
  return values.includes(value)
    ? values.filter((item) => item !== value)
    : [...values, value]
}

function selectionSignature(values: Array<number | string>) {
  return values.map(String).sort().join('\u0000')
}

const hasPendingSelectionChanges = computed(() =>
  selectionSignature(pendingSelections.users) !== selectionSignature(filters.userIds)
  || selectionSignature(pendingSelections.groups) !== selectionSignature(filters.groupIds)
  || selectionSignature(pendingSelections.aliases) !== selectionSignature(filters.routeAliases)
  || selectionSignature(pendingSelections.accounts) !== selectionSignature(filters.accountIds),
)

function syncPendingSelections() {
  pendingSelections.users = [...filters.userIds]
  pendingSelections.groups = [...filters.groupIds]
  pendingSelections.aliases = [...filters.routeAliases]
  pendingSelections.accounts = [...filters.accountIds]
}

function confirmObjectSelection() {
  if (!hasPendingSelectionChanges.value) return
  filters.userIds = [...pendingSelections.users]
  filters.groupIds = [...pendingSelections.groups]
  filters.routeAliases = [...pendingSelections.aliases]
  filters.accountIds = [...pendingSelections.accounts]
}

function removeAppliedFilter(type: keyof ObservabilityFilters, value: number | string) {
  removeFilter(type, value)
  syncPendingSelections()
}

function clearAppliedFilters() {
  clearFilters()
  syncPendingSelections()
}

function activateDimensionSelector(key: SelectorKey) {
  activeDimension.value = key
  dimensionSearch.value = ''
}

function handleDimensionSelectorKeydown(event: KeyboardEvent, current: SelectorKey) {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const index = selectors.value.findIndex((selector) => selector.key === current)
  let next = index
  if (event.key === 'ArrowLeft') next = (index - 1 + selectors.value.length) % selectors.value.length
  if (event.key === 'ArrowRight') next = (index + 1) % selectors.value.length
  if (event.key === 'Home') next = 0
  if (event.key === 'End') next = selectors.value.length - 1
  const selector = selectors.value[next]
  if (!selector) return
  activateDimensionSelector(selector.key)
  document.getElementById(`object-dimension-tab-${selector.key}`)?.focus()
}

const filteredActiveOptions = computed(() => {
  const selector = activeSelector.value
  if (!selector) return []
  const query = dimensionSearch.value.trim().toLowerCase()
  return selector.options.filter((option) => !query || `${option.label} ${option.detail || ''}`.toLowerCase().includes(query))
})

function selectAllActive() {
  const selector = activeSelector.value
  if (!selector) return
  const values = selector.options.map((option) => option.value)
  if (activeDimension.value === 'users') pendingSelections.users = values.map(Number)
  else if (activeDimension.value === 'groups') pendingSelections.groups = values.map(Number)
  else if (activeDimension.value === 'aliases') pendingSelections.aliases = values.map(String)
  else pendingSelections.accounts = values.map(Number)
}

function clearActiveDimension() {
  if (activeDimension.value === 'users') pendingSelections.users = []
  else if (activeDimension.value === 'groups') pendingSelections.groups = []
  else if (activeDimension.value === 'aliases') pendingSelections.aliases = []
  else pendingSelections.accounts = []
}

const drilldown = useObservabilityDrilldown({ filters, window, bucket, rangeStart, rangeEnd, search })

applyRouteFilters()
syncPendingSelections()
watch(() => route.query, () => {
  applyRouteFilters()
  syncPendingSelections()
})

const dimensionTabs = computed(() => [
  { key: 'users', label: t('admin.observability.user'), rows: snapshot.value?.top_users || [] },
  { key: 'groups', label: t('admin.observability.group'), rows: snapshot.value?.top_groups || [] },
  { key: 'aliases', label: t('admin.observability.alias'), rows: snapshot.value?.top_aliases || [] },
  { key: 'accounts', label: t('admin.observability.account'), rows: snapshot.value?.top_accounts || [] },
])

const pressureAccounts = computed(() => snapshot.value?.pressure.accounts || [])
const maxPressure = computed(() => Math.max(0, ...pressureAccounts.value.map((row) => row.utilization)))
const overallStatus = computed(() => {
  const status = statusFor(maxPressure.value)
  return { value: t(status.labelKey), className: status.textClass }
})
const heroStats = computed(() => [
  { label: t('admin.observability.overallLoad'), value: overallStatus.value.value, className: overallStatus.value.className },
  { label: t('admin.observability.activeModelAccounts'), value: `${pressureAccounts.value.length}`, className: 'text-gray-900 dark:text-white' },
  { label: t('admin.observability.highPressureObjects'), value: `${pressureAccounts.value.filter((row) => isHighPressure(row.utilization)).length}`, className: 'text-red-600 dark:text-red-400' },
  { label: t('admin.observability.dataLag'), value: `${snapshot.value?.meta.lag_seconds || 0}s`, className: 'text-gray-900 dark:text-white' },
])

function metricValue(code: string, fallback = 0) {
  return snapshot.value?.metrics.find((metric) => metric.code === code)?.value ?? fallback
}

const capacityAttention = computed(() => {
  const row = [...pressureAccounts.value].sort((a, b) => b.utilization - a.utilization)[0]
  if (!row || row.utilization < OBSERVABILITY_STATUS_THRESHOLDS.attention) return ''
  return t('admin.observability.accountCapacityWarning', {
    id: row.account_id,
    current: row.current,
    limit: row.limit || '∞',
    waiting: row.waiting,
    percent: Math.round(row.utilization * 100),
  })
})

const kpis = computed(() => {
  const requestCount = metricValue('request_count')
  const failureRate = metricValue('client_failure_rate')
  const global = snapshot.value?.capacity?.global
  const topAccount = pressureAccounts.value[0]
  const tokenRates = seriesPoints('total_tokens').map((value) => value / bucketSeconds.value)
  const requestPoints = seriesPoints('total_requests')
  const failurePoints = seriesPointsFrom(latency.value, 'client_failure_rate')
  return [
    { label: t('admin.observability.tokenUsage'), value: formatCompact(metricValue('total_tokens')), detail: t('admin.observability.totalTokens'), dot: 'bg-emerald-500', color: '#10b981', points: seriesPoints('total_tokens') },
    { label: t('admin.observability.currentTokenRate'), value: `${metricValue('token_rate').toFixed(1)}/s`, detail: t('admin.observability.tokenRateDetail'), dot: 'bg-teal-500', color: '#14b8a6', points: tokenRates },
    { label: t('admin.observability.currentConcurrencyLimit'), value: `${global?.current || 0}/${global?.limit || 0}`, detail: t('admin.observability.waterLevel', { percent: Math.round((global?.utilization || 0) * 100) }), dot: 'bg-blue-500', color: '#3b82f6', points: concurrencyOverlay.value },
    { label: t('admin.observability.activeModelAccounts'), value: String(pressureAccounts.value.length), detail: t('admin.observability.availableAccounts', { count: snapshot.value?.filter_options?.accounts?.length || 0 }), dot: 'bg-indigo-500', color: '#6366f1', points: activeAccountPoints.value },
    { label: t('admin.observability.maxScenarioPressure'), value: `${Math.round(maxPressure.value * 100)}%`, detail: topAccount?.account_name || t('admin.observability.noPressureSample'), dot: 'bg-amber-500', color: '#f59e0b', points: maxPressurePoints.value },
    { label: t('admin.observability.rateLimitedFailedRequests'), value: String(Math.round(requestCount * failureRate)), detail: t('admin.observability.failureRate', { percent: (failureRate * 100).toFixed(2) }), dot: 'bg-sky-500', color: '#0ea5e9', points: requestPoints.map((value, index) => value * (failurePoints[index] || 0)) },
    { label: t('admin.observability.avgFirstToken'), value: `${Math.round(metricValue('avg_first_token_ms'))}ms`, detail: t('admin.observability.weightedBySamples'), dot: 'bg-emerald-500', color: '#14b8a6', points: seriesPointsFrom(latency.value, 'avg_first_token_ms') },
    { label: t('admin.observability.clientFailureRate'), value: `${(failureRate * 100).toFixed(2)}%`, detail: t('admin.observability.finalClientRequests'), dot: 'bg-red-500', color: '#ef4444', points: failurePoints },
  ]
})

const bucketSeconds = computed(() => {
  switch (bucket.value) {
    case '5m': return 300
    case '15m': return 900
    case '1h': return 3600
    case '6h': return 21600
    default: return 300
  }
})

function seriesPointsFrom(response: typeof trend.value, key: string) {
  return response?.series.find((series) => series.key === key)?.points || []
}

function seriesPoints(key: string) {
  return seriesPointsFrom(trend.value, key)
}

const activeAccountPoints = computed(() => {
  const columns = concurrencyHeatmap.value?.columns || []
  return columns.map((column) => new Set(
    (concurrencyHeatmap.value?.cells || [])
      .filter((cell) => cell.column === column && cell.available && (cell.actual_concurrency || 0) > 0)
      .map((cell) => cell.row),
  ).size)
})

function dimensionLabelsFor(groupBy: ObservabilityGroupBy) {
  const options = knownFilterOptions
  const labels: Record<string, string> = {}
  if (groupBy === 'account_id') {
    for (const account of options?.accounts || []) {
      labels[String(account.account_id)] = account.name || t('admin.observability.accountRef', { id: account.account_id })
    }
    for (const row of snapshot.value?.top_accounts || []) {
      if (!labels[row.key]) labels[row.key] = row.name || t('admin.observability.accountRef', { id: row.key })
    }
  } else if (groupBy === 'group_id') {
    for (const group of options?.groups || []) {
      labels[String(group.id)] = group.name || t('admin.observability.groupRef', { id: group.id })
    }
    for (const row of snapshot.value?.top_groups || []) {
      if (!labels[row.key]) labels[row.key] = row.name || t('admin.observability.groupRef', { id: row.key })
    }
  } else if (groupBy === 'user_id') {
    for (const user of options?.users || []) {
      labels[String(user.user_id)] = user.email || `${t('admin.observability.user')} #${user.user_id}`
    }
    for (const row of snapshot.value?.top_users || []) {
      if (!labels[row.key]) labels[row.key] = row.name || `${t('admin.observability.user')} #${row.key}`
    }
  } else if (groupBy === 'route_alias') {
    for (const row of snapshot.value?.top_aliases || []) {
      labels[row.key] = row.name || row.key
    }
  }
  return labels
}

const heatmapRowLabels = computed(() => dimensionLabelsFor(heatmapGroupBy.value))
const trendDimensionLabels = computed(() => dimensionLabelsFor(groupBy.value))
const heatmapValueKindHelp = computed(() => {
  if (heatmapValueKind.value === 'error_rate') return t('admin.observability.heatmapErrorRateHelp')
  return ''
})

const maxPressurePoints = computed(() => {
  const columns = concurrencyHeatmap.value?.columns || []
  return columns.map((column) => Math.max(
    0,
    ...(concurrencyHeatmap.value?.cells || [])
      .filter((cell) => cell.column === column && cell.available)
      .map((cell) => cell.value),
  ))
})

function formatCompact(value: number) {
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(2)}M`
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`
  return Math.round(value).toLocaleString()
}

const warningLabelKeys: Record<string, string> = {
  request_rollup_unavailable: 'admin.observability.warning_request_rollup_unavailable',
  wait_duration_unavailable: 'admin.observability.warning_wait_duration_unavailable',
  concurrency_snapshot_unavailable: 'admin.observability.warning_concurrency_snapshot_unavailable',
  today_rate_limit_unavailable: 'admin.observability.warning_today_rate_limit_unavailable',
  tokenstat_projection_unavailable: 'admin.observability.warning_tokenstat_projection_unavailable',
  tokenstat_projection_lagging: 'admin.observability.warning_tokenstat_projection_lagging',
  tokenstat_rollup_reconciliation_difference: 'admin.observability.warning_tokenstat_rollup_reconciliation_difference',
  partial_latency_histogram: 'admin.observability.warning_partial_latency_histogram',
  request_rollup_lagging: 'admin.observability.warning_request_rollup_lagging',
  concurrency_samples_unavailable: 'admin.observability.warning_concurrency_samples_unavailable',
  user_concurrency_samples_unavailable: 'admin.observability.warning_user_concurrency_samples_unavailable',
  missing_request_bucket: 'admin.observability.warning_missing_request_bucket',
  bucket_window_invalid: 'admin.observability.warning_bucket_window_invalid',
}

function warningLabel(warning: string) {
  const key = warningLabelKeys[warning]
  return key ? t(key) : warning
}

function selectQuickRange(value: ObservabilityWindow) {
  timeError.value = ''
  window.value = value
  fillQuickRange(value)
}

function fillQuickRange(value: ObservabilityWindow) {
  const end = new Date()
  const start = new Date(end)
  if (value === 'today') {
    start.setHours(0, 0, 0, 0)
  } else if (value === 'yesterday') {
    start.setDate(start.getDate() - 1)
    start.setHours(0, 0, 0, 0)
    end.setDate(end.getDate() - 1)
    end.setHours(23, 59, 0, 0)
  } else if (value === '1h') {
    start.setHours(start.getHours() - 1)
  } else if (value === '6h') {
    start.setHours(start.getHours() - 6)
  } else if (value === '24h') {
    start.setDate(start.getDate() - 1)
  } else if (value === '7d') {
    start.setDate(start.getDate() - 7)
  }
  customStart.value = toDatetimeLocal(start)
  customEnd.value = toDatetimeLocal(end)
}

function applyCustomTime() {
  timeError.value = ''
  if (!customStart.value || !customEnd.value) {
    timeError.value = t('admin.observability.selectStartEnd')
    return
  }
  if (new Date(customStart.value) >= new Date(customEnd.value)) {
    timeError.value = t('admin.observability.endAfterStart')
    return
  }
  rangeStart.value = customStart.value
  rangeEnd.value = customEnd.value
  window.value = 'custom'
  timeMenuOpen.value = false
}

function toDatetimeLocal(value: Date) {
  const pad = (part: number) => String(part).padStart(2, '0')
  return `${value.getFullYear()}-${pad(value.getMonth() + 1)}-${pad(value.getDate())}T${pad(value.getHours())}:${pad(value.getMinutes())}`
}

function toggleAutoRefresh() {
  autoRefresh.value = !autoRefresh.value
}

fillQuickRange('1h')

function selectAccount(accountId: number) {
  filters.accountIds = filters.accountIds.includes(accountId) ? [] : [accountId]
  syncPendingSelections()
}

const drilldownFocus = computed(() => heatmapFocus.value ? { kind: heatmapFocus.value.kind, key: heatmapFocus.value.key } : null)
const drilldownFocusContext = computed(() => {
  const focus = heatmapFocus.value
  if (!focus) return ''
  const concurrency = focus.actual == null ? '' : ` · ${focus.actual}/${focus.limit ?? 0}`
  return t('admin.observability.heatmapFocus', { key: focus.key, column: focus.column, concurrency })
})

function handleHeatmapSelection(selection: {
  row: string
  column: string
  cell: { actual_concurrency?: number; concurrency_limit?: number }
}) {
  const kind = heatmapGroupBy.value === 'user_id'
    ? 'user'
    : heatmapGroupBy.value === 'account_id'
      ? 'account'
      : heatmapGroupBy.value === 'route_alias' ? 'alias' : 'group'
  heatmapFocus.value = {
    kind,
    key: selection.row,
    column: selection.column,
    actual: selection.cell.actual_concurrency,
    limit: selection.cell.concurrency_limit,
  }
  setTimeout(() => elDrilldown()?.scrollIntoView({ behavior: 'smooth', block: 'start' }), 40)
}

function elDrilldown() {
  return document.getElementById('observability-drilldown')
}

function applyDrilldownFilter(payload: { type: 'user' | 'group' | 'alias' | 'account'; id: number | string }) {
  const id = payload.id
  if (payload.type === 'user') filters.userIds = filters.userIds.includes(Number(id)) ? [] : [Number(id)]
  if (payload.type === 'group') filters.groupIds = filters.groupIds.includes(Number(id)) ? [] : [Number(id)]
  if (payload.type === 'alias') filters.routeAliases = filters.routeAliases.includes(String(id)) ? [] : [String(id)]
  if (payload.type === 'account') filters.accountIds = filters.accountIds.includes(Number(id)) ? [] : [Number(id)]
}

function openAlertObject(payload: { groupId?: number; routeAlias?: string; accountId?: number }) {
  filters.groupIds = payload.groupId ? [payload.groupId] : []
  filters.routeAliases = payload.routeAlias ? [payload.routeAlias] : []
  filters.accountIds = payload.accountId ? [payload.accountId] : []
  router.replace({
    path: '/admin/observability',
    query: {
      group_id: payload.groupId ? String(payload.groupId) : undefined,
      route_alias: payload.routeAlias,
      account_id: payload.accountId ? String(payload.accountId) : undefined,
    },
  })
  setTimeout(() => elDrilldown()?.scrollIntoView({ behavior: 'smooth', block: 'start' }), 40)
}

function applyRouteFilters() {
  filters.userIds = queryNumbers(route.query.user_id)
  filters.groupIds = queryNumbers(route.query.group_id)
  filters.accountIds = queryNumbers(route.query.account_id)
  filters.routeAliases = queryStrings(route.query.route_alias)
  filters.department = queryStrings(route.query.department)[0] || ''
}

function queryStrings(value: unknown): string[] {
  if (Array.isArray(value)) return value.flatMap(queryStrings)
  return typeof value === 'string' && value.trim() ? [value.trim()] : []
}

function queryNumbers(value: unknown): number[] {
  return queryStrings(value)
    .map((item) => Number(item))
    .filter((item) => Number.isFinite(item) && item > 0)
}
</script>

<style scoped>
.field-label {
  @apply flex flex-col gap-1 text-xs font-medium text-gray-600 dark:text-gray-300;
}
</style>

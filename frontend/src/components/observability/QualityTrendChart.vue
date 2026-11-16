<template>
  <article class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
    <div class="mb-3 flex items-center justify-between">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.observability.latencyQuality') }}</h2>
      <span v-if="partialTTFT" class="text-xs text-amber-600 dark:text-amber-400">{{ t('admin.observability.ttftUnavailable') }}</span>
    </div>
    <div v-if="error" class="py-8 text-center text-sm text-red-600 dark:text-red-400" role="alert">{{ error }}</div>
    <div v-else-if="loading && !datasets.length" class="h-64 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-700" aria-busy="true">
      <span class="sr-only">{{ t('admin.observability.loading') }}</span>
    </div>
    <div v-else-if="datasets.length">
      <div class="h-64">
        <Line
          :data="{ labels, datasets: chartDatasets }"
          :options="options"
          role="img"
          :aria-label="t('admin.observability.chartTextSummary', { title: t('admin.observability.latencyQuality') })"
          aria-describedby="quality-trend-summary"
        />
      </div>
      <p id="quality-trend-summary" class="sr-only">{{ chartSummary }}</p>
      <div class="mt-2 flex flex-wrap gap-x-4 gap-y-2 text-xs text-gray-600 dark:text-gray-300">
        <button
          v-for="dataset in datasets"
          :key="dataset.key || dataset.label"
          type="button"
          class="inline-flex cursor-help items-center gap-1.5 rounded px-1 py-0.5 focus:outline-none focus:ring-2 focus:ring-primary-500"
          :class="{ 'opacity-45 line-through': hiddenDatasetKeys.includes(dataset.key || dataset.label) }"
          :title="legendDescription(dataset)"
          :aria-label="legendDescription(dataset)"
          :aria-pressed="!hiddenDatasetKeys.includes(dataset.key || dataset.label)"
          @click="toggleDataset(dataset.key || dataset.label)"
        >
          <span class="h-2.5 w-2.5 rounded-full" :style="{ backgroundColor: dataset.borderColor }"></span>
          {{ dataset.label }}
        </button>
      </div>
    </div>
    <div v-else class="flex h-64 items-center justify-center text-sm text-gray-400">{{ t('admin.observability.noLatencyData') }}</div>
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import {
  CategoryScale,
  Chart as ChartJS,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
} from 'chart.js'
import { Line } from 'vue-chartjs'
import { useObservabilityI18n } from './useObservabilityI18n'
import type { TimeSeriesResponse } from '@/api/admin/observability'
import { qualityChartDatasets } from './chartData'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend)
const { t } = useObservabilityI18n()

const props = defineProps<{
  labels: string[]
  response: TimeSeriesResponse | null
  mode?: 'all' | 'duration' | 'ttft' | 'failure'
  loading?: boolean
  error?: string
}>()

const datasets = computed(() => qualityChartDatasets(
  props.labels,
  props.response,
  props.mode || 'all',
  (key, fallback) => {
    const labels: Record<string, string> = {
      avg_duration_ms: t('admin.observability.avgDuration'),
      avg_first_token_ms: t('admin.observability.avgFirstTokenShort'),
      client_failure_rate: t('admin.observability.clientFailureRate'),
      business_limited_rate: t('admin.observability.businessLimitedRate'),
      sla_error_rate: t('admin.observability.slaErrorRate'),
      slo_2000ms: t('admin.observability.sloThreshold'),
    }
    return labels[key] || fallback
  },
))
const hiddenDatasetKeys = ref<string[]>([])
const chartDatasets = computed(() => datasets.value.map((dataset) => ({
  ...dataset,
  hidden: hiddenDatasetKeys.value.includes(dataset.key || dataset.label),
})))
const partialTTFT = computed(() => props.response?.meta.warnings.includes('partial_latency_histogram') ?? false)
const options = {
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: {
    legend: { display: false },
  },
  scales: {
    yLatency: { beginAtZero: true, position: 'left' as const },
    yRate: { beginAtZero: true, max: 100, position: 'right' as const, grid: { drawOnChartArea: false } },
  },
}
function toggleDataset(key: string) {
  hiddenDatasetKeys.value = hiddenDatasetKeys.value.includes(key)
    ? hiddenDatasetKeys.value.filter((value) => value !== key)
    : [...hiddenDatasetKeys.value, key]
}

function legendDescription(dataset: { key?: string; label: string }) {
  if (dataset.key === 'client_failure_rate') return t('admin.observability.clientFailureRateHelp')
  if (dataset.key === 'business_limited_rate') return t('admin.observability.businessLimitedRateHelp')
  if (dataset.key === 'sla_error_rate') return t('admin.observability.slaErrorRateHelp')
  return dataset.label
}

const chartSummary = computed(() => {
  const latestIndex = props.labels.length - 1
  return datasets.value
    .map((dataset) => t('admin.observability.latestValue', {
      label: dataset.label,
      value: latestIndex >= 0 ? dataset.data[latestIndex] : '-',
    }))
    .join('; ')
})
</script>

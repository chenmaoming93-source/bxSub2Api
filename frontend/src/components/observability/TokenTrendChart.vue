<template>
  <article class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
    <div class="mb-3 flex items-center justify-between">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.observability.tokenRequestTrend') }}</h2>
        <p class="mt-1 text-xs text-gray-500">{{ t('admin.observability.cacheHitRateHelp') }}</p>
      </div>
      <span class="text-xs text-gray-500">{{ t('admin.observability.buckets', { count: labels.length }) }}</span>
    </div>
    <div v-if="error" class="py-8 text-center text-sm text-red-600 dark:text-red-400" role="alert">{{ error }}</div>
    <div v-else-if="loading && !datasets.length" class="h-64 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-700" aria-busy="true">
      <span class="sr-only">{{ t('admin.observability.loading') }}</span>
    </div>
    <div v-else-if="datasets.length" class="h-64">
      <Line
        :data="{ labels, datasets }"
        :options="options"
        role="img"
        :aria-label="t('admin.observability.chartTextSummary', { title: t('admin.observability.tokenRequestTrend') })"
        aria-describedby="token-trend-summary"
      />
      <p id="token-trend-summary" class="sr-only">{{ chartSummary }}</p>
    </div>
    <div v-else class="flex h-64 items-center justify-center text-sm text-gray-400">{{ t('admin.observability.noTrendData') }}</div>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import {
  CategoryScale,
  Chart as ChartJS,
  Filler,
  Legend,
  LinearScale,
  LineElement,
  PointElement,
  Tooltip,
} from 'chart.js'
import type { ChartOptions, TooltipItem } from 'chart.js'
import { Line } from 'vue-chartjs'
import { useObservabilityI18n } from './useObservabilityI18n'
import type { ObservabilityGroupBy, TimeSeriesResponse } from '@/api/admin/observability'
import { tokenChartDatasets } from './chartData'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend, Filler)
const { t } = useObservabilityI18n()

const props = defineProps<{
  labels: string[]
  response: TimeSeriesResponse | null
  concurrency: number[]
  tokenMetric?: 'total' | 'input' | 'output' | 'cache'
  groupBy?: ObservabilityGroupBy
  dimensionLabels?: Record<string, string>
  loading?: boolean
  error?: string
}>()

const datasets = computed(() => tokenChartDatasets(
  props.labels,
  props.response,
  props.concurrency,
  props.tokenMetric || 'total',
  (key, fallback) => {
    const labels: Record<string, string> = {
      total_tokens: t('admin.observability.totalTokens'),
      input_tokens: t('admin.observability.input'),
      output_tokens: t('admin.observability.output'),
      cache_tokens: t('admin.observability.cache'),
      cache_hit_rate: t('admin.observability.cacheHitRate'),
      total_requests: t('admin.observability.requests'),
      sampled_concurrency: t('admin.observability.sampledConcurrency'),
      dimension_other: t('admin.observability.dimensionOther'),
      overall_token: t('admin.observability.overallToken'),
    }
    return labels[key] || fallback
  },
  props.groupBy,
  props.dimensionLabels || {},
))
const options = computed<ChartOptions<'line'>>(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: {
    legend: { labels: { usePointStyle: true, boxWidth: 8 } },
    tooltip: {
      callbacks: {
        label: (context: TooltipItem<'line'>) => {
          const label = context.dataset.label || ''
          const value = context.parsed.y
          if (value === null) return `${label}: -`
          if (context.dataset.yAxisID === 'yRate') {
            return `${label}: ${(value * 100).toFixed(1)}%`
          }
          return `${label}: ${value.toLocaleString()}`
        },
      },
    },
  },
  scales: {
    yTokens: { type: 'linear' as const, position: 'left' as const, beginAtZero: true },
    yRequests: { type: 'linear' as const, position: 'right' as const, beginAtZero: true, grid: { drawOnChartArea: false } },
    yConcurrency: { type: 'linear' as const, position: 'right' as const, beginAtZero: true, display: props.concurrency.length > 0, grid: { drawOnChartArea: false } },
    yRate: {
      type: 'linear' as const,
      position: 'right' as const,
      beginAtZero: true,
      offset: true,
      grid: { drawOnChartArea: false },
      ticks: { callback: (value: string | number) => `${(Number(value) * 100).toFixed(0)}%` },
    },
  },
}))
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

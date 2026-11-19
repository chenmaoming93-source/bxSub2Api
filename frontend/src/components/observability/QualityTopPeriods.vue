<template>
  <article class="min-w-0 rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
    <div class="mb-3 flex items-center justify-between gap-2">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.observability.qualityPeriodTitle') }}</h2>
        <p class="mt-1 text-xs text-gray-500">{{ t('admin.observability.qualityPeriodHint') }}</p>
      </div>
      <span class="rounded-full bg-amber-50 px-2 py-1 text-xs text-amber-700 dark:bg-amber-900/30 dark:text-amber-300">TOP 8</span>
    </div>
    <div v-if="error" class="py-6 text-center text-sm text-red-600 dark:text-red-400" role="alert">{{ error }}</div>
    <div v-else-if="loading && !rows.length" class="h-32 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-700" aria-busy="true">
      <span class="sr-only">{{ t('admin.observability.loading') }}</span>
    </div>
    <div v-else-if="rows.length" class="overflow-x-auto">
      <table class="min-w-full text-xs">
        <thead class="text-left text-gray-500">
          <tr><th class="py-2">{{ t('admin.observability.period') }}</th><th class="py-2 text-right">{{ t('admin.observability.avgDuration') }}</th><th class="py-2 text-right">{{ t('admin.observability.avgFirstToken') }}</th></tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.label" class="border-t border-gray-100 dark:border-dark-700">
            <td class="py-2 text-gray-700 dark:text-gray-200">{{ row.label }}</td>
            <td class="py-2 text-right font-mono text-gray-900 dark:text-white">{{ row.duration }}</td>
            <td class="py-2 text-right font-mono text-gray-900 dark:text-white">{{ row.ttft }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="flex h-32 items-center justify-center text-sm text-gray-400">{{ t('admin.observability.noQualityPeriodData') }}</div>
    <p class="mt-2 text-xs text-amber-600 dark:text-amber-400">{{ t('admin.observability.missingBucketsNotZero') }}</p>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useObservabilityI18n } from './useObservabilityI18n'
import type { TimeSeriesResponse } from '@/api/admin/observability'

const props = defineProps<{
  response: TimeSeriesResponse | null
  loading?: boolean
  error?: string
}>()
const { t } = useObservabilityI18n()

const rows = computed(() => {
  const response = props.response
  if (!response) return []
  const duration = response.series.find((item) => item.key === 'avg_duration_ms')
  const ttft = response.series.find((item) => item.key === 'avg_first_token_ms')
  if (!duration) return []
  return response.labels
    .map((label, index) => ({
      label,
      durationValue: duration.points[index],
      ttftValue: ttft?.points[index],
    }))
    .filter((row) => Number.isFinite(row.durationValue) && (row.durationValue || 0) > 0)
    .sort((left, right) => right.durationValue - left.durationValue)
    .slice(0, 8)
    .map((row) => ({
      label: row.label,
      duration: `${Math.round(row.durationValue)}ms`,
      ttft: row.ttftValue && row.ttftValue > 0 ? `${Math.round(row.ttftValue)}ms` : t('admin.observability.noSampling'),
    }))
})
</script>

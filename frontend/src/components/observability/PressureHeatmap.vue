<template>
  <article class="min-w-0 rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.observability.pressureHeatmap') }}</h2>
      <span
        class="inline-flex cursor-help items-center gap-1 text-xs text-gray-500"
        :title="valueKindHelp || valueKindLabel"
        :aria-label="valueKindHelp || valueKindLabel"
      >
        {{ valueKindLabel }}
        <span v-if="valueKindHelp" aria-hidden="true" class="text-[10px]">ⓘ</span>
      </span>
    </div>
    <div v-if="error" class="py-8 text-center text-sm text-red-600 dark:text-red-400" role="alert">{{ error }}</div>
    <div v-else-if="!response?.available" class="rounded-xl bg-gray-50 py-10 text-center text-sm text-gray-500 dark:bg-dark-700">
      {{ t('admin.observability.unavailableNoSampledData') }}
    </div>
    <div v-else-if="matrix.rows.length" class="relative overflow-x-auto">
      <div v-if="hovered" class="pointer-events-none absolute right-2 top-1 z-20 w-52 rounded-xl border border-gray-200 bg-white p-3 text-xs shadow-xl dark:border-dark-600 dark:bg-dark-700">
        <div class="flex items-center justify-between gap-2">
          <b class="truncate text-gray-900 dark:text-white">{{ rowLabel(hovered.row) }}</b>
          <span class="shrink-0 text-gray-500">{{ hovered.column }}</span>
        </div>
        <div v-if="valueKind === 'sampled_concurrency'" class="mt-2">
          <strong class="block text-lg text-gray-900 dark:text-white">{{ hovered.cell.actual_concurrency ?? 0 }} / {{ hovered.cell.concurrency_limit ?? 0 }}</strong>
          <span class="mt-1 block text-gray-500">{{ valueKind === 'sampled_concurrency' ? concurrencyDenominatorLabel : formatValue(hovered.cell.value) }}</span>
          <span v-if="hovered.cell.sample_at" class="mt-1 block text-gray-400">{{ t('admin.observability.samplePeakTime', { time: formatTime(hovered.cell.sample_at) }) }}</span>
        </div>
        <div v-else class="mt-2 font-mono text-base text-gray-900 dark:text-white">{{ formatValue(hovered.cell.value) }}</div>
      </div>
      <table class="min-w-full border-separate border-spacing-1 text-xs">
        <thead>
          <tr>
            <th class="sticky left-0 bg-white px-2 py-1 text-left dark:bg-dark-800">{{ t('admin.observability.heatmapObject') }}</th>
            <th v-for="column in matrix.columns" :key="column" class="px-2 py-1 text-center font-normal text-gray-500">{{ column }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, rowIndex) in matrix.rows" :key="row">
            <th class="sticky left-0 bg-white px-2 py-1 text-left font-medium dark:bg-dark-800">{{ rowLabel(row) }}</th>
            <td v-for="(cell, columnIndex) in matrix.cells[rowIndex]" :key="`${row}-${columnIndex}`">
              <button
                v-if="cell.available"
                type="button"
                class="flex h-10 w-full min-w-16 items-center justify-center rounded-md font-mono text-gray-900 ring-1 ring-inset ring-black/5 focus:outline-none focus:ring-2 focus:ring-primary-500"
                :style="{ backgroundColor: heatColor(cell) }"
                :title="tooltipTitle(row, columnIndex, cell)"
                :aria-label="tooltipTitle(row, columnIndex, cell)"
                @mouseenter="hovered = { row, column: matrix.columns[columnIndex], cell }"
                @focus="hovered = { row, column: matrix.columns[columnIndex], cell }"
                @mouseleave="hovered = null"
                @blur="hovered = null"
                @click="$emit('select', { row, column: matrix.columns[columnIndex], cell })"
              >
                {{ formatValue(cell.value) }}
              </button>
              <div v-else class="flex h-10 w-full min-w-16 items-center justify-center rounded-md bg-gray-100 text-gray-400 dark:bg-dark-700">—</div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="py-10 text-center text-sm text-gray-400">{{ t('admin.observability.noHeatmapRows') }}</div>
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useObservabilityI18n } from './useObservabilityI18n'
import type { HeatmapResponse } from '@/api/admin/observability'
import { heatmapMatrix, type HeatmapMatrix } from './chartData'
import { statusFor } from './status'

const props = defineProps<{
  response: HeatmapResponse | null
  valueKind: string
  singleAccountFiltered?: boolean
  rowLabels?: Record<string, string>
  valueKindHelp?: string
  error?: string
}>()
export interface HeatmapSelection {
  row: string
  column: string
  cell: HeatmapMatrix['cells'][number][number]
}
defineEmits<{ select: [selection: HeatmapSelection] }>()

const matrix = computed(() => {
  const raw = heatmapMatrix(props.response)
  const rowIndexes = raw.rows
    .map((_row, index) => index)
    .filter((index) => raw.cells[index]?.some((cell) => cell.available))
  return {
    rows: rowIndexes.map((index) => raw.rows[index]),
    columns: raw.columns,
    cells: rowIndexes.map((index) => raw.cells[index]),
  }
})
const { t } = useObservabilityI18n()
const hovered = ref<null | { row: string; column: string; cell: HeatmapMatrix['cells'][number][number] }>(null)
const concurrencyDenominatorLabel = computed(() =>
  props.singleAccountFiltered
    ? t('admin.observability.accountConcurrencyDenominator')
    : t('admin.observability.rangeConcurrencyDenominator'),
)
const valueKindLabel = computed(() => {
  if (props.valueKind === 'sampled_concurrency') return t('admin.observability.sampledConcurrency')
  if (props.valueKind === 'token_rate') return t('admin.observability.tokenRate')
  if (props.valueKind === 'error_rate') return t('admin.observability.errorRate')
  return props.valueKind
})
const maximum = computed(() => Math.max(1, ...matrix.value.cells.flat().filter((cell) => cell.available).map((cell) => cell.value)))

function heatColor(cell: HeatmapMatrix['cells'][number][number]) {
  let ratio = cell.value
  if (props.valueKind === 'sampled_concurrency') {
    ratio = cell.concurrency_limit && cell.concurrency_limit > 0
      ? (cell.actual_concurrency || 0) / cell.concurrency_limit
      : 0
  } else if (props.valueKind === 'token_rate') {
    ratio = cell.value / maximum.value
  }
  return statusFor(Math.min(1, Math.max(0, ratio))).backgroundColor
}

function formatValue(value: number) {
  if (props.valueKind.endsWith('_rate')) return `${(value * 100).toFixed(1)}%`
  return value.toFixed(props.valueKind === 'token_rate' ? 1 : 0)
}

function rowLabel(row: string) {
  return props.rowLabels?.[row] || row
}

function tooltipTitle(row: string, columnIndex: number, cell: HeatmapMatrix['cells'][number][number]) {
  const label = rowLabel(row)
  if (props.valueKind === 'sampled_concurrency') {
    return `${label} · ${matrix.value.columns[columnIndex]} · ${cell.actual_concurrency ?? 0}/${cell.concurrency_limit ?? 0} · ${cell.sample_at || t('admin.observability.noSampleTime')}`
  }
  return `${label} · ${matrix.value.columns[columnIndex]} · ${formatValue(cell.value)}`
}

function formatTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}
</script>

<template>
  <div class="max-h-80 w-full overflow-y-auto">
    <div v-if="displayRows.length > 0" class="min-h-[320px]" :style="{ height: `${Math.max(displayRows.length, 10) * 32}px` }">
      <Bar :data="chartData" :options="chartOptions" />
    </div>
    <div v-else class="flex h-80 items-center justify-center text-sm text-gray-500 dark:text-gray-400">
      {{ t('admin.usage.departmentUsage.noUserData') }}
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Bar } from 'vue-chartjs'
import { BarElement, CategoryScale, Chart as ChartJS, LinearScale, Tooltip } from 'chart.js'
import type { DepartmentUserUsageRow } from '@/api/admin/usage'

ChartJS.register(CategoryScale, LinearScale, BarElement, Tooltip)

const { t } = useI18n()
const props = defineProps<{ rows: DepartmentUserUsageRow[] }>()
const displayRows = computed(() => [...(props.rows || [])].sort((a, b) => b.total_tokens - a.total_tokens).slice(0, 20))
const label = (row: DepartmentUserUsageRow) => row.username || row.email || String(row.user_id)
const barColors = ['#4f46e5', '#818cf8']
const chartData = computed(() => ({
  labels: displayRows.value.map(label),
  datasets: [{
    label: t('admin.usage.departmentUsage.tokens'),
    data: displayRows.value.map((row) => row.total_tokens),
    backgroundColor: displayRows.value.map((_, index) => barColors[index % barColors.length]),
    borderRadius: 4,
    barThickness: 16
  }]
}))
const chartOptions = computed(() => ({
  indexAxis: 'y' as const,
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: {
      callbacks: {
        label: (context: any) => `${Number(context.raw).toLocaleString()} ${t('admin.usage.departmentUsage.tokens')}`
      }
    }
  },
  scales: {
    // Keep the numeric axis visible as soon as the chart opens; the rows scroll below it.
    x: { beginAtZero: true, position: 'top' as const },
    y: { ticks: { autoSkip: false } }
  }
}))
</script>

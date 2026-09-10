<template>
  <div class="flex flex-col gap-6 md:flex-row md:items-center">
    <div class="mx-auto h-52 w-52 shrink-0 sm:h-56 sm:w-56">
      <Doughnut :data="chartData" :options="chartOptions" />
    </div>
    <div class="max-h-64 min-w-0 flex-1 overflow-y-auto">
      <table class="w-full text-xs">
        <thead>
          <tr class="text-gray-500 dark:text-gray-400">
            <th class="pb-2 text-left">{{ t('admin.usage.departmentUsage.department') }}</th>
            <th class="pb-2 text-right">{{ t('admin.usage.departmentUsage.tokens') }}</th>
            <th class="pb-2 text-right">{{ t('admin.usage.departmentUsage.percent') }}</th>
            <th class="pb-2 text-right">{{ t('admin.usage.departmentUsage.userCount') }}</th>
            <th class="pb-2 text-right">{{ t('admin.usage.departmentUsage.averageTokens') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="row in displayRows"
            :key="row.department"
            class="cursor-pointer border-t border-gray-100 transition-colors hover:bg-gray-50 dark:border-gray-700 dark:hover:bg-dark-700/40"
            @click="emit('select', row.department || t('admin.usage.departmentUsage.unset'))"
          >
            <td class="max-w-[180px] truncate py-2 font-medium text-gray-900 dark:text-white" :title="row.department">
              {{ row.department || t('admin.usage.departmentUsage.unset') }}
            </td>
            <td class="py-2 text-right text-gray-600 dark:text-gray-400">{{ row.total_tokens.toLocaleString() }}</td>
            <td class="py-2 text-right text-gray-500 dark:text-gray-400">{{ row.percentage.toFixed(1) }}%</td>
            <td class="py-2 text-right text-gray-500 dark:text-gray-400">{{ row.user_count.toLocaleString() }}</td>
            <td class="py-2 text-right text-gray-500 dark:text-gray-400">{{ Math.round(row.average_tokens).toLocaleString() }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Chart as ChartJS, ArcElement, Tooltip, Legend } from 'chart.js'
import { Doughnut } from 'vue-chartjs'
import type { DepartmentUsageRow } from '@/api/admin/usage'

ChartJS.register(ArcElement, Tooltip, Legend)

const { t } = useI18n()
const props = defineProps<{ rows: DepartmentUsageRow[] }>()
const emit = defineEmits<{ select: [department: string] }>()

const colors = ['#3b82f6', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#14b8a6', '#f97316', '#6366f1', '#84cc16']
const displayRows = computed(() => [...(props.rows || [])].sort((a, b) => b.total_tokens - a.total_tokens))
const total = computed(() => displayRows.value.reduce((sum, row) => sum + row.total_tokens, 0))
const percentage = (value: number) => total.value > 0 ? ((value / total.value) * 100).toFixed(1) : '0.0'
const chartData = computed(() => ({
  labels: displayRows.value.map((row) => row.department || t('admin.usage.departmentUsage.unset')),
  datasets: [{ data: displayRows.value.map((row) => row.total_tokens), backgroundColor: displayRows.value.map((_, index) => colors[index % colors.length]), borderWidth: 0 }]
}))
const chartOptions = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  plugins: {
    legend: { display: false },
    tooltip: { callbacks: { label: (context: any) => `${context.label}: ${Number(context.raw).toLocaleString()} (${percentage(Number(context.raw))}%)` } }
  }
}))
</script>

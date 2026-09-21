<script setup lang="ts">
import { computed, ref } from 'vue'
import { Line } from 'vue-chartjs'
import UiSearchInput from '@/components/ui-v2/forms/UiSearchInput.vue'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend } from 'chart.js'
import type { UserUsageTrendPoint } from '@/types'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend)

const props = withDefaults(defineProps<{ points?: UserUsageTrendPoint[]; loading?: boolean }>(), { points: () => [], loading: false })
const query = ref('')
const scope = ref<'top12' | 'all'>('top12')
const colors = ['#2563eb', '#10b981', '#f59e0b', '#ef4444', '#8b5cf6', '#ec4899', '#0ea5e9', '#f97316', '#14b8a6', '#6366f1', '#84cc16', '#a855f7']

const chartData = computed(() => {
  const users = new Map<number, { label: string; values: Map<string, number>; total: number }>()
  for (const point of props.points) {
    const label = point.username?.trim() || point.email?.trim() || `用户 ${point.user_id}`
    const user = users.get(point.user_id) ?? { label, values: new Map<string, number>(), total: 0 }
    user.values.set(point.date, point.tokens)
    user.total += point.tokens
    users.set(point.user_id, user)
  }
  const keyword = query.value.trim().toLowerCase()
  const selectedUsers = [...users.values()]
    .filter((user) => !keyword || user.label.toLowerCase().includes(keyword))
    .sort((a, b) => b.total - a.total)
  const visibleUsers = scope.value === 'top12' ? selectedUsers.slice(0, 12) : selectedUsers
  const labels = [...new Set(visibleUsers.flatMap((user) => [...user.values.keys()]))].sort()
  return {
    labels,
    datasets: visibleUsers.map((user, index) => ({
      label: user.label,
      data: labels.map((date) => user.values.get(date) ?? 0),
      borderColor: colors[index % colors.length],
      backgroundColor: `${colors[index % colors.length]}22`,
      borderWidth: 2,
      pointRadius: 2,
      tension: .3,
      fill: false
    }))
  }
})

const options = computed(() => ({
  responsive: true,
  maintainAspectRatio: false,
  interaction: { intersect: false, mode: 'index' as const },
  plugins: { legend: { position: 'bottom' as const, labels: { usePointStyle: true, color: 'var(--ui-v2-muted)' } } },
  scales: {
    x: { grid: { display: false }, ticks: { color: '#64748b', maxRotation: 0 } },
    y: { beginAtZero: true, grid: { color: '#e2e8f0' }, ticks: { color: '#64748b' } }
  }
}))
</script>

<template>
  <div v-if="loading" class="ui-v2-chart-state">加载中…</div>
  <template v-else>
    <div class="admin-dashboard-v2-trend-toolbar">
      <UiSearchInput v-model="query" placeholder="搜索用户名称或邮箱" />
      <div class="ui-v2-ranking__scope" role="group" aria-label="用户趋势范围">
        <button type="button" :class="{ 'ui-v2-ranking__scope-button--active': scope === 'top12' }" @click="scope = 'top12'">Top 12</button>
        <button type="button" :class="{ 'ui-v2-ranking__scope-button--active': scope === 'all' }" @click="scope = 'all'">全部</button>
      </div>
    </div>
    <div v-if="!chartData.datasets.length" class="ui-v2-chart-state">暂无匹配的用户 Token 趋势数据</div>
    <div v-else class="admin-dashboard-v2-trend"><Line :data="chartData" :options="options" /></div>
  </template>
</template>

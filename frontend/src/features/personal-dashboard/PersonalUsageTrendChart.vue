<script setup lang="ts">
import { computed } from 'vue'
import { Line } from 'vue-chartjs'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend } from 'chart.js'
import type { TrendDataPoint } from '@/types'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend)
const props = withDefaults(defineProps<{ points?: TrendDataPoint[]; loading?: boolean }>(), { points: () => [], loading: false })
const chartData = computed(() => ({
  labels: props.points.map((point) => point.date),
  datasets: [{ label: 'Token 消耗', data: props.points.map((point) => point.total_tokens), borderColor: '#2563eb', backgroundColor: '#2563eb22', borderWidth: 2, pointRadius: 2, tension: .3, fill: true }]
}))
const options = { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { x: { grid: { display: false }, ticks: { color: '#64748b' } }, y: { beginAtZero: true, grid: { color: '#e2e8f0' }, ticks: { color: '#64748b' } } } }
</script>

<template>
  <div v-if="loading" class="ui-v2-chart-state">加载中…</div>
  <div v-else-if="!points.length" class="ui-v2-chart-state">暂无趋势数据</div>
  <div v-else class="personal-dashboard-v2-trend"><Line :data="chartData" :options="options" /></div>
</template>

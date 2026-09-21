<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { Line } from 'vue-chartjs'
import { Chart as ChartJS, CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend } from 'chart.js'
import { opsAPI, type OpsModelLatencyPercentile, type OpsModelLatencyTrendPoint, type OpsRequestOptions, type OpsQueryMode } from '@/api/admin/ops'
import type { OpsRequestDetailsPreset } from './OpsRequestDetailsModal.vue'

ChartJS.register(CategoryScale, LinearScale, PointElement, LineElement, Tooltip, Legend)
const props = defineProps<{ timeRange: string; platform: string; groupId: number | null; queryMode: OpsQueryMode }>()
const emit = defineEmits<{ (event: 'open-details', preset: OpsRequestDetailsPreset): void }>()
const loading = ref(false); const error = ref(''); const models = ref<OpsModelLatencyPercentile[]>([]); const points = ref<OpsModelLatencyTrendPoint[]>([]); const selectedModel = ref('')
const topModels = computed(() => models.value.slice(0, 5))
const chartData = computed(() => ({ labels: points.value.map((p) => new Date(p.bucket_start).toLocaleString()), datasets: [{ label: '平均耗时(ms)', data: points.value.map((p) => p.avg_ms ?? 0), borderColor: '#2563eb', backgroundColor: '#2563eb22', fill: true, tension: .25, pointRadius: 2 }] }))
const chartOptions = { responsive: true, maintainAspectRatio: false, plugins: { legend: { display: false } }, scales: { y: { beginAtZero: true, ticks: { color: '#64748b' } }, x: { ticks: { color: '#64748b', maxTicksLimit: 8 } } } }
function params() { return { time_range: props.timeRange as any, platform: props.platform || undefined, group_id: props.groupId || undefined, model: selectedModel.value || undefined, mode: props.queryMode } }
async function load() { loading.value = true; error.value = ''; try { const options: OpsRequestOptions = {}; const [percentiles, trend] = await Promise.all([opsAPI.getDashboardModelLatencyPercentiles(params(), options), opsAPI.getDashboardModelLatencyTrend(params(), options)]); models.value = percentiles.models ?? []; points.value = trend.points ?? [] } catch (e: any) { error.value = e?.message || '模型耗时数据加载失败' } finally { loading.value = false } }
function selectModel(model: string) { selectedModel.value = model; void load() }
watch(() => [props.timeRange, props.platform, props.groupId, props.queryMode], () => void load())
onMounted(() => void load())
</script>

<template>
  <section class="ops-model-latency card p-5">
    <div class="mb-4 flex flex-wrap items-center justify-between gap-3"><div><h3 class="text-base font-semibold text-gray-900 dark:text-white">按模型耗时分析</h3><p class="mt-1 text-xs text-gray-500 dark:text-gray-400">P50 / P90 / P95 / P99 与平均耗时趋势</p></div><button type="button" class="btn btn-secondary btn-sm" @click="emit('open-details', { title: '模型耗时请求详情', kind: 'all', sort: 'duration_desc' })">查看请求</button></div>
    <div v-if="error" class="mb-3 rounded-lg bg-rose-50 p-3 text-sm text-rose-600">{{ error }}</div>
    <div v-if="loading" class="py-10 text-center text-sm text-gray-500">加载中…</div>
    <template v-else>
      <div class="mb-5 flex flex-wrap gap-2"><button type="button" class="rounded-lg px-3 py-1.5 text-xs" :class="!selectedModel ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'" @click="selectModel('')">全部模型</button><button v-for="model in topModels" :key="model.model" type="button" class="rounded-lg px-3 py-1.5 text-xs" :class="selectedModel === model.model ? 'bg-blue-600 text-white' : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'" @click="selectModel(model.model)">{{ model.model }}</button></div>
      <div class="overflow-auto"><table class="min-w-full text-left text-sm"><thead class="text-xs text-gray-500"><tr><th class="pb-2 pr-5">模型</th><th class="pb-2 pr-5">请求数</th><th class="pb-2 pr-5">平均</th><th class="pb-2 pr-5">P50</th><th class="pb-2 pr-5">P90</th><th class="pb-2 pr-5">P95</th><th class="pb-2">P99</th></tr></thead><tbody><tr v-for="model in topModels" :key="model.model" class="border-t border-gray-100 dark:border-dark-700"><td class="py-2 pr-5 font-medium">{{ model.model }}</td><td class="py-2 pr-5">{{ model.request_count.toLocaleString() }}</td><td class="py-2 pr-5">{{ model.avg_ms ?? '-' }}ms</td><td class="py-2 pr-5">{{ model.p50_ms ?? '-' }}ms</td><td class="py-2 pr-5">{{ model.p90_ms ?? '-' }}ms</td><td class="py-2 pr-5">{{ model.p95_ms ?? '-' }}ms</td><td class="py-2">{{ model.p99_ms ?? '-' }}ms</td></tr></tbody></table></div>
      <div v-if="points.length" class="mt-5 h-64 cursor-pointer" title="点击查看请求详情" @click="emit('open-details', { title: '模型耗时请求详情', kind: 'all', sort: 'duration_desc' })"><Line :data="chartData" :options="chartOptions" /></div><div v-else class="py-8 text-center text-sm text-gray-500">暂无趋势数据</div>
    </template>
  </section>
</template>

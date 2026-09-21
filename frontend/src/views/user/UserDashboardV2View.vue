<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useAuthStore } from '@/stores/auth'
import { usageAPI, type UserDashboardStats } from '@/api/usage'
import AppLayout from '@/components/layout/AppLayout.vue'
import UiPageHeader from '@/components/ui-v2/layout/UiPageHeader.vue'
import UiStatCard from '@/components/ui-v2/display/UiStatCard.vue'
import UiCard from '@/components/ui-v2/display/UiCard.vue'
import UiDateRangePicker from '@/components/ui-v2/forms/UiDateRangePicker.vue'
import UiSelect from '@/components/ui-v2/forms/UiSelect.vue'
import UiButton from '@/components/ui-v2/primitives/UiButton.vue'
import UiHorizontalRanking, { type UiRankingItem } from '@/components/ui-v2/data/UiHorizontalRanking.vue'
import PersonalUsageTrendChart from '@/features/personal-dashboard/PersonalUsageTrendChart.vue'
import type { ModelStat, TrendDataPoint } from '@/types'

const authStore = useAuthStore()
const stats = ref<UserDashboardStats | null>(null)
const rangeStats = ref<{ total_requests: number; total_tokens: number; total_actual_cost: number; average_duration_ms: number } | null>(null)
const trend = ref<TrendDataPoint[]>([])
const models = ref<ModelStat[]>([])
const loading = ref(false)
const loadingCharts = ref(false)
function localDate(date: Date) { return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}` }
const end = new Date(); const start = new Date(end.getTime() - 6 * 86400000)
const startDate = ref(localDate(start)); const endDate = ref(localDate(end)); const granularity = ref<'day' | 'hour'>('day')
function formatTokens(value: number | undefined | null) { const n = value ?? 0; if (n >= 100_000_000) return `${(n / 100_000_000).toFixed(2)}亿`; if (n >= 10_000) return `${(n / 10_000).toFixed(2)}万`; return n.toLocaleString() }
function formatCost(value: number | undefined | null) { return `$${(value ?? 0).toFixed(2)}` }
const modelRanking = computed<UiRankingItem[]>(() => { const total = models.value.reduce((sum, row) => sum + row.total_tokens, 0); return [...models.value].sort((a, b) => b.total_tokens - a.total_tokens).map((row) => ({ id: row.model, label: row.model, value: row.total_tokens, displayValue: formatTokens(row.total_tokens), percentage: total ? row.total_tokens / total * 100 : 0 })) })
async function load() {
  loading.value = true; loadingCharts.value = true
  try {
    await authStore.refreshUser()
    const [userStats, periodStats, trendResponse, modelResponse] = await Promise.all([
      usageAPI.getDashboardStats(), usageAPI.getStatsByDateRange(startDate.value, endDate.value),
      usageAPI.getDashboardTrend({ start_date: startDate.value, end_date: endDate.value, granularity: granularity.value }),
      usageAPI.getDashboardModels({ start_date: startDate.value, end_date: endDate.value })
    ])
    stats.value = userStats
    rangeStats.value = { total_requests: periodStats.total_requests, total_tokens: periodStats.total_tokens, total_actual_cost: periodStats.total_actual_cost, average_duration_ms: periodStats.average_duration_ms }
    trend.value = trendResponse.trend ?? []; models.value = modelResponse.models ?? []
  } finally { loading.value = false; loadingCharts.value = false }
}
function applyRange() { void load() }
onMounted(() => { void load() })
</script>

<template>
  <AppLayout>
    <div class="personal-dashboard-v2">
      <UiPageHeader title="个人仪表盘" description="查看当前用户的 API Key、Token 和费用使用情况">
        <UiButton variant="secondary" :loading="loading" @click="load">刷新数据</UiButton>
      </UiPageHeader>
      <div class="personal-dashboard-v2__toolbar">
        <UiDateRangePicker v-model:start-date="startDate" v-model:end-date="endDate" />
        <UiSelect v-model="granularity" :options="[{ label: '按小时', value: 'hour' }, { label: '按天', value: 'day' }]" />
        <UiButton size="sm" @click="applyRange">应用范围</UiButton>
      </div>
      <div class="personal-dashboard-v2__stats">
        <UiStatCard label="当前用户 API Key 数" :value="(stats?.total_api_keys ?? 0).toLocaleString()" />
        <UiStatCard label="时间范围内请求数" :value="(rangeStats?.total_requests ?? 0).toLocaleString()" />
        <UiStatCard label="时间范围内 Token 消耗" :value="formatTokens(rangeStats?.total_tokens)" />
        <UiStatCard label="当前用户累计 Token" :value="formatTokens(stats?.total_tokens)" />
        <UiStatCard label="时间范围内平均响应耗时" :value="`${Math.round(rangeStats?.average_duration_ms ?? 0)}ms`" tone="warning" />
        <UiStatCard label="当前用户累计费用" :value="formatCost(stats?.total_actual_cost)" tone="warning" />
      </div>
      <div class="personal-dashboard-v2__charts">
        <UiCard title="模型 Token 用量"><UiHorizontalRanking :items="modelRanking" search-placeholder="搜索模型名称" :show-action="false" /></UiCard>
        <UiCard title="Token 消耗趋势"><PersonalUsageTrendChart :points="trend" :loading="loadingCharts" /></UiCard>
      </div>
    </div>
  </AppLayout>
</template>

<style scoped>
.personal-dashboard-v2 { min-width: 0; color: var(--ui-v2-text); }
.personal-dashboard-v2__toolbar { display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: .65rem; margin-bottom: 1rem; }
.personal-dashboard-v2__stats { display: grid; grid-template-columns: repeat(6, minmax(0, 1fr)); gap: 1rem; margin-bottom: 1rem; }
.personal-dashboard-v2__charts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; }
.personal-dashboard-v2__charts > * { min-width: 0; }
:deep(.personal-dashboard-v2-trend) { height: 21rem; }
@media (max-width: 1200px) { .personal-dashboard-v2__stats { grid-template-columns: repeat(3, minmax(0, 1fr)); } }
@media (max-width: 900px) { .personal-dashboard-v2__charts { grid-template-columns: 1fr; } }
@media (max-width: 640px) { .personal-dashboard-v2__stats { grid-template-columns: 1fr; } }
</style>

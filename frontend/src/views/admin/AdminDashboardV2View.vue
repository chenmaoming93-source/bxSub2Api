<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import AppLayout from '@/components/layout/AppLayout.vue'
import UiCard from '@/components/ui-v2/display/UiCard.vue'
import UiStatCard from '@/components/ui-v2/display/UiStatCard.vue'
import UiDateRangePicker from '@/components/ui-v2/forms/UiDateRangePicker.vue'
import UiSelect from '@/components/ui-v2/forms/UiSelect.vue'
import UiButton from '@/components/ui-v2/primitives/UiButton.vue'
import UiPageHeader from '@/components/ui-v2/layout/UiPageHeader.vue'
import UiResponsiveGrid from '@/components/ui-v2/layout/UiResponsiveGrid.vue'
import UiChartCard from '@/components/ui-v2/charts/UiChartCard.vue'
import UiHorizontalRanking, { type UiRankingItem } from '@/components/ui-v2/data/UiHorizontalRanking.vue'
import UserTokenTrendChart from '@/features/admin-dashboard/UserTokenTrendChart.vue'
import SceneAccountTokenDialog from '@/features/admin-dashboard/SceneAccountTokenDialog.vue'
import DepartmentUserTokenDialog from '@/features/admin-dashboard/DepartmentUserTokenDialog.vue'
import { useAdminDashboardData, type AdminDashboardDateRange, type SceneRankingRow } from '@/features/admin-dashboard/adminDashboardData'
import type { ModelStat } from '@/types'

function localDate(date: Date) {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}-${String(date.getDate()).padStart(2, '0')}`
}
const end = new Date()
const start = new Date(end.getTime() - 24 * 60 * 60 * 1000)
const initialRange: AdminDashboardDateRange = { startDate: localDate(start), endDate: localDate(end), granularity: 'hour' }
const { range, data, loading, load } = useAdminDashboardData(initialRange)
const selectedScene = ref<SceneRankingRow | null>(null)
const selectedDepartment = ref<string | null>(null)

const stats = computed(() => data.value.snapshot?.stats)
const models = computed<UiRankingItem[]>(() => {
  const rows = data.value.snapshot?.models ?? []
  const total = rows.reduce((sum, row) => sum + row.total_tokens, 0)
  return [...rows].sort((a, b) => b.total_tokens - a.total_tokens).map((row: ModelStat) => ({
    id: row.model,
    label: row.model,
    value: row.total_tokens,
    displayValue: formatTokens(row.total_tokens),
    percentage: total ? row.total_tokens / total * 100 : 0
  }))
})
const scenes = computed<UiRankingItem[]>(() => data.value.scenes.map((row) => ({
  id: row.id, label: row.label, value: row.totalTokens, displayValue: formatTokens(row.totalTokens), percentage: row.percentage
})))
const departments = computed<UiRankingItem[]>(() => data.value.departments.map((row) => ({
  id: row.department, label: row.department, value: row.total_tokens, displayValue: formatTokens(row.total_tokens), percentage: row.percentage
})))

function formatNumber(value: number | undefined | null) { return (value ?? 0).toLocaleString() }
function formatTokens(value: number | undefined | null) {
  const number = value ?? 0
  if (number >= 100_000_000) return `${(number / 100_000_000).toFixed(2)}亿`
  if (number >= 10_000) return `${(number / 10_000).toFixed(2)}万`
  return number.toLocaleString()
}
function formatDuration(value: number | undefined | null) { return `${Math.round(value ?? 0)}ms` }
function formatCost(value: number | undefined | null) { return `$${(value ?? 0).toFixed(2)}` }
const rangeMinutes = computed(() => Math.max(1, (new Date(range.value.endDate).getTime() - new Date(range.value.startDate).getTime()) / 60000))
const rangeRpm = computed(() => (data.value.usageStats?.total_requests ?? 0) / rangeMinutes.value)
const rangeTpm = computed(() => (data.value.usageStats?.total_tokens ?? 0) / rangeMinutes.value)
function updateRange() {
  const nextRange: AdminDashboardDateRange = { ...range.value }
  const days = Math.ceil((new Date(nextRange.endDate).getTime() - new Date(nextRange.startDate).getTime()) / 86400000)
  nextRange.granularity = days <= 1 ? 'hour' : 'day'
  void load(nextRange)
}
function openScene(item: UiRankingItem) { selectedScene.value = data.value.scenes.find((row) => row.id === item.id) ?? null }
function openDepartment(item: UiRankingItem) { selectedDepartment.value = item.label }

onMounted(() => { void load() })
</script>

<template>
  <AppLayout>
    <div class="admin-dashboard-v2">
      <UiPageHeader title="管理员仪表盘" description="统一查看平台 Token、请求和费用运行概况">
        <UiButton variant="secondary" :loading="loading" @click="load()">刷新数据</UiButton>
      </UiPageHeader>

      <div class="admin-dashboard-v2__toolbar">
        <UiDateRangePicker v-model:start-date="range.startDate" v-model:end-date="range.endDate" @change="updateRange" />
        <UiSelect v-model="range.granularity" :options="[{ label: '按小时', value: 'hour' }, { label: '按天', value: 'day' }]" placeholder="粒度" />
        <UiButton size="sm" @click="updateRange">应用范围</UiButton>
      </div>

      <UiResponsiveGrid :columns="5" class="admin-dashboard-v2__stats">
        <UiStatCard label="系统总用户数" :value="formatNumber(stats?.total_users)" />
        <UiStatCard label="模型账号数" :value="formatNumber(stats?.total_accounts)" tone="success" />
        <UiStatCard label="API Key 数" :value="formatNumber(stats?.total_api_keys)" tone="success" />
        <UiStatCard label="范围内请求" :value="formatNumber(data.usageStats?.total_requests)" />
        <UiStatCard label="范围内 Token" :value="formatTokens(data.usageStats?.total_tokens)" />
        <UiStatCard label="累计系统 Token" :value="formatTokens(stats?.total_tokens)" tone="success" />
        <UiStatCard label="范围性能指标" :value="`RPM ${rangeRpm.toFixed(1)} / TPM ${formatTokens(rangeTpm)}`" hint="当前日期范围平均" tone="success" />
        <UiStatCard label="平均响应耗时" :value="formatDuration(data.usageStats?.average_duration_ms)" tone="warning" />
        <UiStatCard label="累计费用消耗" :value="formatCost(stats?.total_actual_cost)" tone="warning" />
      </UiResponsiveGrid>

      <div class="admin-dashboard-v2__charts">
        <UiChartCard title="用户 Token 趋势 Top 12" description="按当前时间范围显示用户 Token 消耗">
          <UserTokenTrendChart :points="data.usersTrend" :loading="loading" />
        </UiChartCard>
        <UiCard title="模型 Token 排行"><UiHorizontalRanking :items="models" value-label="Token" :show-action="false" /></UiCard>
        <UiCard title="场景（分组）Token 排行"><UiHorizontalRanking :items="scenes" value-label="Token" @select="openScene" /></UiCard>
        <UiCard title="部门 Token 排行"><UiHorizontalRanking :items="departments" value-label="Token" @select="openDepartment" /></UiCard>
      </div>

      <SceneAccountTokenDialog
        v-if="selectedScene"
        :model-value="Boolean(selectedScene)"
        :scene="selectedScene"
        :start-date="range.startDate"
        :end-date="range.endDate"
        @update:model-value="selectedScene = $event ? selectedScene : null"
      />
      <DepartmentUserTokenDialog
        v-if="selectedDepartment"
        :model-value="Boolean(selectedDepartment)"
        :department="selectedDepartment"
        :start-date="range.startDate"
        :end-date="range.endDate"
        @update:model-value="selectedDepartment = $event ? selectedDepartment : null"
      />
    </div>
  </AppLayout>
</template>

<style scoped>
.admin-dashboard-v2 { min-width: 0; color: var(--ui-v2-text); }
.admin-dashboard-v2__toolbar { display: flex; flex-wrap: wrap; align-items: center; justify-content: flex-end; gap: .65rem; margin-bottom: 1rem; }
.admin-dashboard-v2__toolbar .ui-v2-select { width: 7rem; }
.admin-dashboard-v2__stats { margin-bottom: 1rem; }
.admin-dashboard-v2__charts { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 1rem; }
.admin-dashboard-v2__charts > * { min-width: 0; }
.admin-dashboard-v2__selection-note { color: var(--ui-v2-muted); font-size: .75rem; }
:deep(.ui-v2-chart-state) { display: grid; min-height: 15rem; place-items: center; color: var(--ui-v2-muted); font-size: .8125rem; }
:deep(.admin-dashboard-v2-trend) { height: 15rem; }
@media (max-width: 900px) { .admin-dashboard-v2__charts { grid-template-columns: 1fr; } }
</style>

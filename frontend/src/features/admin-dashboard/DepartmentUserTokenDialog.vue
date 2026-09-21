<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import UiDetailDialog from '@/components/ui-v2/feedback/UiDetailDialog.vue'
import UiSearchInput from '@/components/ui-v2/forms/UiSearchInput.vue'
import UiButton from '@/components/ui-v2/primitives/UiButton.vue'
import UiPagination from '@/components/ui-v2/data/UiPagination.vue'
import { adminAPI } from '@/api/admin'
import type { DepartmentUserUsageResponse } from '@/api/admin/usage'

interface UserRow { id: number; user: string; email: string; totalTokens: number; percentage: number }
const props = withDefaults(defineProps<{ modelValue?: boolean; department: string | null; startDate: string; endDate: string }>(), { modelValue: false, department: null })
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const loading = ref(false)
const error = ref('')
const response = ref<DepartmentUserUsageResponse | null>(null)
const query = ref('')
const page = ref(1)
const pageSize = ref(20)
const departmentTokens = computed(() => response.value?.department_total_tokens ?? 0)
const rows = computed<UserRow[]>(() => {
  const keyword = query.value.trim().toLowerCase()
  const total = departmentTokens.value
  return (response.value?.rows ?? [])
    .map((row) => ({ id: row.user_id, user: row.username || row.email, email: row.email, totalTokens: row.total_tokens, percentage: total > 0 ? row.total_tokens / total * 100 : row.percentage }))
    .filter((row) => !keyword || `${row.user} ${row.email}`.toLowerCase().includes(keyword))
    .sort((a, b) => b.totalTokens - a.totalTokens)
})
const visibleRows = computed(() => rows.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
function formatTokens(value: number) {
  if (value >= 100_000_000) return `${(value / 100_000_000).toFixed(2)}亿`
  if (value >= 10_000) return `${(value / 10_000).toFixed(2)}万`
  return value.toLocaleString()
}

async function load() {
  if (!props.modelValue || !props.department) return
  loading.value = true
  error.value = ''
  page.value = 1
  try {
    response.value = await adminAPI.usage.queryDepartmentUsers({ department: props.department, start_date: props.startDate, end_date: props.endDate, page: 1, page_size: 100 })
  } catch (caught) {
    response.value = null
    error.value = caught instanceof Error ? caught.message : '部门详情加载失败'
  } finally {
    loading.value = false
  }
}
function close() { emit('update:modelValue', false) }
watch(() => [props.modelValue, props.department, props.startDate, props.endDate], () => { void load() }, { immediate: true })
</script>

<template>
  <UiDetailDialog width="full" :model-value="modelValue" :title="`${department ?? '部门'} · 用户 Token 明细`" :loading="loading" @update:model-value="emit('update:modelValue', $event)">
    <div class="ui-v2-detail-subtitle">{{ startDate }} 至 {{ endDate }} · 部门用户明细</div>
    <div class="ui-v2-detail-toolbar"><div><strong>{{ formatTokens(departmentTokens) }}</strong><span> Token · {{ rows.length }} 位用户</span></div><UiSearchInput v-model="query" placeholder="搜索用户或邮箱" /></div>
    <p v-if="error" class="ui-v2-detail-error">{{ error }} <UiButton size="sm" variant="secondary" @click="load">重试</UiButton></p>
    <div v-else-if="!visibleRows.length" class="ui-v2-detail-empty">当前部门暂无用户数据</div>
    <div v-else class="ui-v2-token-ranking">
      <div class="ui-v2-token-ranking__head"><span>#</span><span>用户</span><span>邮箱</span><span>Token 用量</span><span>占比</span></div>
      <div v-for="(row, index) in visibleRows" :key="row.id" class="ui-v2-token-ranking__row">
        <span>{{ (page - 1) * pageSize + index + 1 }}</span><strong>{{ row.user }}</strong><span>{{ row.email }}</span>
        <div class="ui-v2-token-ranking__metric"><b>{{ formatTokens(row.totalTokens) }}</b><span class="ui-v2-token-ranking__track"><i :style="{ width: `${Math.min(row.percentage, 100)}%` }" /></span></div>
        <span>{{ row.percentage.toFixed(1) }}%</span>
      </div>
    </div>
    <UiPagination v-if="rows.length" v-model:page="page" v-model:page-size="pageSize" :total="rows.length" />
    <template #footer><UiButton variant="secondary" @click="close">关闭</UiButton></template>
  </UiDetailDialog>
</template>

<style scoped>
.ui-v2-detail-subtitle { margin: -.5rem 0 1rem; color: var(--ui-v2-muted); font-size: .875rem; }
.ui-v2-detail-toolbar { display: flex; align-items: center; justify-content: space-between; gap: 1rem; margin-bottom: 1rem; color: var(--ui-v2-muted); font-size: .75rem; }
.ui-v2-detail-toolbar strong { color: var(--ui-v2-text); font-size: 1.5rem; font-weight: 750; }
.ui-v2-detail-toolbar .ui-v2-search-input { width: 20rem; }
.ui-v2-detail-error { display: flex; align-items: center; gap: .75rem; color: var(--ui-v2-danger); }
.ui-v2-detail-empty { padding: 3rem; color: var(--ui-v2-muted); text-align: center; }
.ui-v2-token-ranking { overflow-x: auto; border: 1px solid var(--ui-v2-border); border-radius: 10px; }
.ui-v2-token-ranking__head, .ui-v2-token-ranking__row { display: grid; grid-template-columns: 2rem minmax(13rem, 1.2fr) minmax(16rem, 1fr) minmax(18rem, 1.5fr) 5rem; align-items: center; gap: 1rem; min-width: 58rem; padding: .55rem 1rem; }
.ui-v2-token-ranking__head { color: var(--ui-v2-muted); background: var(--ui-v2-page); font-size: .75rem; font-weight: 700; }
.ui-v2-token-ranking__row { border-top: 1px solid var(--ui-v2-border); color: var(--ui-v2-text); font-size: .875rem; }
.ui-v2-token-ranking__row strong { font-weight: 750; }
.ui-v2-token-ranking__metric { display: grid; grid-template-columns: 5rem minmax(8rem, 1fr); align-items: center; gap: .75rem; font-variant-numeric: tabular-nums; }
.ui-v2-token-ranking__track { height: 1rem; overflow: hidden; border-radius: 5px; background: color-mix(in srgb, var(--ui-v2-primary) 12%, var(--ui-v2-border)); }
.ui-v2-token-ranking__track i { display: block; height: 100%; border-radius: 5px; background: var(--ui-v2-primary); }
</style>

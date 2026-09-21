<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import UiDetailDialog from '@/components/ui-v2/feedback/UiDetailDialog.vue'
import UiSearchInput from '@/components/ui-v2/forms/UiSearchInput.vue'
import UiButton from '@/components/ui-v2/primitives/UiButton.vue'
import UiPagination from '@/components/ui-v2/data/UiPagination.vue'
import { adminAPI } from '@/api/admin'
import type { SceneAccountDailyUsageResponse } from '@/api/admin/usage'
import type { SceneRankingRow } from './adminDashboardData'

interface AccountRow { id: string; account: string; model: string; totalTokens: number; percentage: number }
const props = withDefaults(defineProps<{ modelValue?: boolean; scene: SceneRankingRow | null; startDate: string; endDate: string }>(), { modelValue: false, scene: null })
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const loading = ref(false)
const error = ref('')
const response = ref<SceneAccountDailyUsageResponse | null>(null)
const query = ref('')
const page = ref(1)
const pageSize = ref(20)

const rows = computed<AccountRow[]>(() => {
  const grouped = new Map<string, AccountRow>()
  for (const day of response.value?.days ?? []) {
    for (const scene of day.scenes) {
      if (props.scene && scene.group_id !== props.scene.id) continue
      for (const account of scene.accounts) {
        const id = `${account.account_id}:${account.upstream_model}`
        const current = grouped.get(id) ?? { id, account: account.account_name, model: account.upstream_model, totalTokens: 0, percentage: 0 }
        current.totalTokens += account.total_tokens
        grouped.set(id, current)
      }
    }
  }
  const keyword = query.value.trim().toLowerCase()
  const mainTotal = props.scene?.totalTokens ?? 0
  return [...grouped.values()]
    .filter((row) => !keyword || `${row.account} ${row.model}`.toLowerCase().includes(keyword))
    .map((row) => ({ ...row, percentage: mainTotal > 0 ? row.totalTokens / mainTotal * 100 : 0 }))
    .sort((a, b) => b.totalTokens - a.totalTokens)
})
const detailTotalTokens = computed(() => rows.value.reduce((sum, row) => sum + row.totalTokens, 0))
const totalTokens = computed(() => props.scene?.totalTokens ?? detailTotalTokens.value)
const visibleRows = computed(() => rows.value.slice((page.value - 1) * pageSize.value, page.value * pageSize.value))
function formatTokens(value: number) {
  if (value >= 100_000_000) return `${(value / 100_000_000).toFixed(2)}亿`
  if (value >= 10_000) return `${(value / 10_000).toFixed(2)}万`
  return value.toLocaleString()
}

async function load() {
  if (!props.modelValue || !props.scene) return
  loading.value = true
  error.value = ''
  page.value = 1
  try {
    response.value = await adminAPI.usage.querySceneAccountDaily({ start_date: props.startDate, end_date: props.endDate, group_name: props.scene.label })
  } catch (caught) {
    response.value = null
    error.value = caught instanceof Error ? caught.message : '场景详情加载失败'
  } finally {
    loading.value = false
  }
}
function close() { emit('update:modelValue', false) }
watch(() => [props.modelValue, props.scene?.id, props.startDate, props.endDate], () => { void load() }, { immediate: true })
</script>

<template>
  <UiDetailDialog width="full" :model-value="modelValue" :title="`${scene?.label ?? '场景'} · 账号 Token 明细`" :loading="loading" @update:model-value="emit('update:modelValue', $event)">
    <div class="ui-v2-detail-subtitle">{{ startDate }} 至 {{ endDate }} · 场景（分组）明细</div>
    <div class="ui-v2-detail-toolbar"><div><strong>{{ formatTokens(totalTokens) }}</strong><span> Token · {{ rows.length }} 个账号/模型组合</span></div><UiSearchInput v-model="query" placeholder="搜索账号或模型" /></div>
    <p v-if="error" class="ui-v2-detail-error">{{ error }} <UiButton size="sm" variant="secondary" @click="load">重试</UiButton></p>
    <div v-else-if="!visibleRows.length" class="ui-v2-detail-empty">当前场景暂无账号数据</div>
    <div v-else class="ui-v2-token-ranking">
      <div class="ui-v2-token-ranking__head"><span>#</span><span>模型账号</span><span>上游模型</span><span>Token 用量</span><span>占比</span></div>
      <div v-for="(row, index) in visibleRows" :key="row.id" class="ui-v2-token-ranking__row">
        <span>{{ (page - 1) * pageSize + index + 1 }}</span><strong>{{ row.account }}</strong><span>{{ row.model }}</span>
        <div class="ui-v2-token-ranking__metric"><b>{{ formatTokens(row.totalTokens) }}</b><span class="ui-v2-token-ranking__track"><i :style="{ width: `${Math.min(row.percentage, 100)}%` }" /></span></div>
        <span>{{ row.percentage.toFixed(1) }}%</span>
      </div>
    </div>
    <UiPagination v-if="rows.length" v-model:page="page" v-model:page-size="pageSize" :total="rows.length" :show-page-size-selector="true" />
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
.ui-v2-token-ranking__head, .ui-v2-token-ranking__row { display: grid; grid-template-columns: 2rem minmax(13rem, 1.2fr) minmax(12rem, 1fr) minmax(18rem, 1.5fr) 5rem; align-items: center; gap: 1rem; min-width: 58rem; padding: .55rem 1rem; }
.ui-v2-token-ranking__head { color: var(--ui-v2-muted); background: var(--ui-v2-page); font-size: .75rem; font-weight: 700; }
.ui-v2-token-ranking__row { border-top: 1px solid var(--ui-v2-border); color: var(--ui-v2-text); font-size: .875rem; }
.ui-v2-token-ranking__row strong { font-weight: 750; }
.ui-v2-token-ranking__metric { display: grid; grid-template-columns: 5rem minmax(8rem, 1fr); align-items: center; gap: .75rem; font-variant-numeric: tabular-nums; }
.ui-v2-token-ranking__track { height: 1rem; overflow: hidden; border-radius: 5px; background: color-mix(in srgb, var(--ui-v2-primary) 12%, var(--ui-v2-border)); }
.ui-v2-token-ranking__track i { display: block; height: 100%; border-radius: 5px; background: var(--ui-v2-primary); }
</style>

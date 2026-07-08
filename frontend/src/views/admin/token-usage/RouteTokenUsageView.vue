<template>
  <AppLayout>
  <TokenUsageReportLayout
    :title="t('tokenUsageReport.routeTitle')"
    :loading="report.loading.value"
    :error="report.error.value"
    :has-data="items.length > 0"
    @refresh="search"
    @retry="search"
  >
    <template #filters>
      <div class="flex flex-wrap items-end gap-3">
        <!-- Group selector -->
        <div class="flex-1 min-w-[200px]">
          <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.group') }}</label>
          <input
            v-model="groupSearch"
            type="text"
            class="input w-full text-sm"
            :placeholder="t('tokenUsageReport.searchGroups')"
            @input="onGroupSearch"
          />
          <div v-if="groupOptions.length > 0 && groupSearch" class="absolute z-10 mt-1 max-h-40 w-full overflow-auto rounded border bg-white dark:border-dark-600 dark:bg-dark-800">
            <div
              v-for="g in groupOptions"
              :key="g.id"
              class="cursor-pointer px-3 py-1.5 text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
              @click="selectGroup(g)"
            >
              {{ g.label }}
            </div>
          </div>
        </div>
        <!-- Route alias selector -->
        <div class="flex-1 min-w-[200px]">
          <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.routeAlias') }}</label>
          <input
            v-model="routeSearch"
            type="text"
            class="input w-full text-sm"
            :placeholder="t('tokenUsageReport.searchRoutes')"
            @input="onRouteSearch"
          />
          <div v-if="routeOptions.length > 0 && routeSearch" class="absolute z-10 mt-1 max-h-40 w-full overflow-auto rounded border bg-white dark:border-dark-600 dark:bg-dark-800">
            <div
              v-for="r in routeOptions"
              :key="r.route_alias"
              class="cursor-pointer px-3 py-1.5 text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
              @click="selectRoute(r)"
            >
              {{ r.label }}
            </div>
          </div>
        </div>
        <!-- Upstream model selector (optional) -->
        <div class="flex-1 min-w-[200px]">
          <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.upstreamModel') }}</label>
          <input
            v-model="modelSearch"
            type="text"
            class="input w-full text-sm"
            :placeholder="t('tokenUsageReport.optionalModelPlaceholder')"
            @input="onRouteModelSearch"
          />
          <div v-if="modelOptions.length > 0 && modelSearch" class="absolute z-10 mt-1 max-h-40 w-full overflow-auto rounded border bg-white dark:border-dark-600 dark:bg-dark-800">
            <div v-for="m in modelOptions" :key="m.model" class="cursor-pointer px-3 py-1.5 text-sm hover:bg-gray-100 dark:hover:bg-dark-700" @mousedown.prevent="selectRouteModel(m)">{{ m.label }}</div>
          </div>
        </div>
        <!-- Date range -->
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.start') }}</label>
          <input v-model="startDateModel" type="date" class="input text-sm" @change="onDateChange" />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.end') }}</label>
          <input v-model="endDateModel" type="date" class="input text-sm" @change="onDateChange" />
        </div>
        <TokenUsageSortEditor v-model="sortRules" :options="sortOptions" />
        <div>
          <button class="btn btn-primary btn-sm" :disabled="!canQuery" @click="search">
            {{ t('tokenUsageReport.query') }}
          </button>
        </div>
      </div>
    </template>

    <template #summary>
      <TokenUsageSummary
        v-if="reportResult"
        :used-tokens="reportResult.summary.used_tokens"
        :daily-limit-tokens="null"
      />
    </template>

    <template #table>
      <RouteTokenUsageTable :items="items" />
    </template>

    <template #pagination>
      <Pagination
        v-if="reportResult && reportResult.pagination.total > 0"
        :total="reportResult.pagination.total"
        :page="report.page.value"
        :page-size="report.pageSize.value"
        @update:page="onPageChange"
        @update:page-size="onPageSizeChange"
      />
    </template>
  </TokenUsageReportLayout>
  </AppLayout>
</template>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import { useI18n } from 'vue-i18n'
import { ref, computed, onMounted, watch } from 'vue'
import TokenUsageReportLayout from '@/components/admin/token-usage/TokenUsageReportLayout.vue'
import TokenUsageSummary from '@/components/admin/token-usage/TokenUsageSummary.vue'
import RouteTokenUsageTable from '@/components/admin/token-usage/RouteTokenUsageTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import TokenUsageSortEditor from '@/components/admin/token-usage/TokenUsageSortEditor.vue'
import { useTokenUsageReport } from '@/composables/useTokenUsageReport'
import { tokenUsageAPI } from '@/api/admin/tokenUsage'
import type { TokenUsageReport, RouteTokenUsageItem, TokenUsageOption, TokenUsageSortField, TokenUsageSortRule } from '@/api/admin/tokenUsage'

const report = useTokenUsageReport({ paramPrefix: 'route' })
const { t } = useI18n()

const items = ref<RouteTokenUsageItem[]>([])
const reportResult = ref<TokenUsageReport<RouteTokenUsageItem> | null>(null)
const sortRules = ref<TokenUsageSortRule[]>([{ field: 'usage_date', order: 'desc' }])
const sortOptions = computed<Array<{ value: TokenUsageSortField; label: string }>>(() => [
  { value: 'usage_date', label: t('tokenUsageReport.date') },
  { value: 'group', label: t('tokenUsageReport.group') },
  { value: 'route_alias', label: t('tokenUsageReport.routeAlias') },
  { value: 'upstream_model', label: t('tokenUsageReport.upstreamModelName') },
  { value: 'priority', label: t('tokenUsageReport.priority') },
  { value: 'used_tokens', label: t('tokenUsageReport.usedTokens') },
  { value: 'daily_limit_tokens', label: t('tokenUsageReport.dailyLimit') },
  { value: 'usage_rate', label: t('tokenUsageReport.usageRate') },
  { value: 'status', label: t('tokenUsageReport.status') }
])

// 鈹€鈹€ Group search 鈹€鈹€
const groupSearch = ref(report.targetLabel.value || '')
const groupOptions = ref<TokenUsageOption[]>([])
const selectedGroup = ref<TokenUsageOption | null>(null)
let groupTimer: ReturnType<typeof setTimeout> | undefined

function onGroupSearch() {
  if (selectedGroup.value && groupSearch.value !== selectedGroup.value.label) {
    selectedGroup.value = null
  }
  clearTimeout(groupTimer)
  groupTimer = setTimeout(async () => {
    if (!groupSearch.value) { groupOptions.value = []; return }
    try {
      const res = await tokenUsageAPI.searchGroups(groupSearch.value)
      groupOptions.value = res.items
    } catch { /* ignore */ }
  }, 300)
}

function selectGroup(g: TokenUsageOption) {
  selectedGroup.value = g
  groupSearch.value = g.label
  groupOptions.value = []
}

// 鈹€鈹€ Route search 鈹€鈹€
const routeSearch = ref('')
const routeOptions = ref<TokenUsageOption[]>([])
const selectedRoute = ref<TokenUsageOption | null>(null)
let routeTimer: ReturnType<typeof setTimeout> | undefined

function onRouteSearch() {
  if (selectedRoute.value && routeSearch.value !== selectedRoute.value.label) {
    selectedRoute.value = null
  }
  clearTimeout(routeTimer)
  routeTimer = setTimeout(async () => {
    if (!routeSearch.value) { routeOptions.value = []; return }
    if (!selectedGroup.value) { routeOptions.value = []; return }
    try {
      const res = await tokenUsageAPI.searchRoutes(selectedGroup.value!.id, routeSearch.value)
      routeOptions.value = res.items
    } catch { /* ignore */ }
  }, 300)
}

function selectRoute(r: TokenUsageOption) {
  selectedRoute.value = r
  routeSearch.value = r.label
  routeOptions.value = []
}

// 鈹€鈹€ Model search (optional) 鈹€鈹€
const modelSearch = ref('')
const modelOptions = ref<TokenUsageOption[]>([])
let modelTimer: ReturnType<typeof setTimeout> | undefined

function onRouteModelSearch() {
  clearTimeout(modelTimer)
  modelTimer = setTimeout(async () => {
    try {
      modelOptions.value = selectedGroup.value && routeSearch.value.trim()
        ? (await tokenUsageAPI.searchRouteModels(selectedGroup.value.id, routeSearch.value.trim(), modelSearch.value)).items
        : (await tokenUsageAPI.searchModels(modelSearch.value)).items
    } catch { modelOptions.value = [] }
  }, 300)
}

function selectRouteModel(model: TokenUsageOption) { modelSearch.value = model.model || model.label; modelOptions.value = [] }

// 鈹€鈹€ Date models for binding 鈹€鈹€
const startDateModel = ref(report.startDate.value)
const endDateModel = ref(report.endDate.value)

function onDateChange() {
  report.setDates(startDateModel.value, endDateModel.value)
}

// 鈹€鈹€ Can query 鈹€鈹€
const canQuery = computed(() => Boolean(startDateModel.value && endDateModel.value) && !report.loading.value)

// 鈹€鈹€ Search 鈹€鈹€
async function search() {
  if (!canQuery.value) return
  report.error.value = null
  const { seq } = report.nextSignal()
  report.loading.value = true
  try {
    const params: any = {
      start_date: startDateModel.value,
      end_date: endDateModel.value,
      page: report.page.value,
      page_size: report.pageSize.value,
      sort_by: sortRules.value.map(rule => rule.field).join(','),
      sort_order: sortRules.value.map(rule => rule.order).join(',')
    }
    if (selectedGroup.value) params.group_id = selectedGroup.value.id
    if (routeSearch.value.trim()) params.route_alias = routeSearch.value.trim()
    if (modelSearch.value.trim()) {
      params.upstream_model = modelSearch.value.trim()
    }
    const res = await tokenUsageAPI.getRouteReport(params)
    if (!report.isCurrent(seq)) return
    reportResult.value = res
    items.value = res.items
  } catch (e: any) {
    if (report.isAbortError(e)) return
    if (!report.isCurrent(seq)) return
    report.error.value = e?.message || t('tokenUsageReport.loadFailed')
  } finally {
    if (report.isCurrent(seq)) report.loading.value = false
  }
}

// 鈹€鈹€ Pagination 鈹€鈹€
async function onPageChange(p: number) {
  await report.setPage(p)
  await search()
}

async function onPageSizeChange(ps: number) {
  await report.setPageSize(ps)
  await search()
}

watch(sortRules, async () => {
  await report.setPage(1)
  await search()
}, { deep: true })

// 鈹€鈹€ Init 鈹€鈹€
onMounted(async () => {
  // If URL has target, restore group
  if (report.targetId.value && report.targetLabel.value) {
    selectedGroup.value = {
      id: Number(report.targetId.value.split(':')[0]),
      label: report.targetLabel.value.split(':')[0],
      group_id: Number(report.targetId.value.split(':')[0])
    }
    report.resetPage()
    
    if (selectedGroup.value) {
      groupSearch.value = selectedGroup.value.label
    }
    // If URL has a route alias too, restore it
    const parts = report.targetId.value.split(':')
    if (parts.length > 1) {
      selectedRoute.value = {
        id: 0,
        label: parts[1],
        route_alias: parts[1],
        group_id: selectedGroup.value!.id
      }
      routeSearch.value = parts[1]
    }
  }

  await search()
})
</script>

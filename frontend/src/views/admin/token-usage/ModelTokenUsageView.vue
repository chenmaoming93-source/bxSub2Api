<template>
  <AppLayout>
  <TokenUsageReportLayout
    :title="t('tokenUsageReport.modelTitle')"
    :loading="report.loading.value"
    :error="report.error.value"
    :has-data="items.length > 0"
    @refresh="doQuery"
    @retry="doQuery"
  >
    <!-- Filters -->
    <template #filters>
      <div class="flex flex-wrap items-end gap-3">
        <!-- Model search -->
        <div class="flex-1 min-w-[200px]">
          <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.model') }}</label>
          <div class="relative">
            <input
              v-model="modelSearch"
              type="text"
              class="input w-full"
              :placeholder="t('tokenUsageReport.searchModel')"
              @input="onModelSearchInput"
              @focus="showModelDropdown = true"
              @blur="hideModelDropdown"
            />
            <div
              v-if="showModelDropdown && modelOptions.length > 0"
              class="absolute z-10 mt-1 max-h-48 w-full overflow-auto rounded-md border border-gray-200 bg-white shadow-lg dark:border-dark-600 dark:bg-dark-700"
            >
              <div
                v-for="opt in modelOptions"
                :key="opt.model ?? opt.label"
                class="cursor-pointer px-3 py-2 text-sm hover:bg-gray-100 dark:hover:bg-dark-600"
                :class="{ 'bg-primary-50 dark:bg-primary-900/30': opt.model === report.targetId.value }"
                @mousedown.prevent="selectModel(opt)"
              >
                {{ opt.label }}
              </div>
            </div>
          </div>
        </div>

        <!-- Date range -->
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.startDate') }}</label>
          <input
            :value="report.startDate.value"
            type="date"
            class="input"
            @change="onStartDateChange"
          />
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.endDate') }}</label>
          <input
            :value="report.endDate.value"
            type="date"
            class="input"
            @change="onEndDateChange"
          />
        </div>

        <!-- Sort -->
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.sortBy') }}</label>
          <select v-model="sortBy" class="input">
            <option value="usage_date">{{ t('tokenUsageReport.date') }}</option>
            <option value="used_tokens">{{ t('tokenUsageReport.tokens') }}</option>
            <option value="model">{{ t('tokenUsageReport.model') }}</option>
            <option value="daily_limit_tokens">{{ t('tokenUsageReport.dailyLimit') }}</option>
            <option value="usage_rate">{{ t('tokenUsageReport.usageRate') }}</option>
            <option value="status">{{ t('tokenUsageReport.status') }}</option>
          </select>
        </div>
        <div>
          <label class="mb-1 block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.order') }}</label>
          <select v-model="sortOrder" class="input">
            <option value="desc">{{ t('tokenUsageReport.descending') }}</option>
            <option value="asc">{{ t('tokenUsageReport.ascending') }}</option>
          </select>
        </div>

        <!-- Query button -->
        <div>
          <button
            class="btn btn-primary"
            :disabled="!datesValid || report.loading.value"
            @click="doQuery"
          >
            {{ t('tokenUsageReport.query') }}
          </button>
        </div>
      </div>
    </template>

    <!-- Summary -->
    <template #summary>
      <TokenUsageSummary
        :used-tokens="summaryUsed"
        :daily-limit-tokens="summaryDailyLimit"
      />
    </template>

    <!-- Table -->
    <template #table>
      <ModelTokenUsageTable :items="items" />
    </template>

    <!-- Pagination -->
    <template #pagination>
      <Pagination
        v-if="total > 0"
        :page="report.page.value"
        :page-size="report.pageSize.value"
        :total="total"
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
import { computed, ref, onMounted, watch } from 'vue'
import TokenUsageReportLayout from '@/components/admin/token-usage/TokenUsageReportLayout.vue'
import TokenUsageSummary from '@/components/admin/token-usage/TokenUsageSummary.vue'
import ModelTokenUsageTable from '@/components/admin/token-usage/ModelTokenUsageTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import { useTokenUsageReport } from '@/composables/useTokenUsageReport'
import { searchModelOptions, getModelTokenUsageReport, type ModelTokenUsageItem, type TokenUsageOption, type TokenUsageSortField } from '@/api/admin/tokenUsage'

const report = useTokenUsageReport({ paramPrefix: 'model' })
const { t } = useI18n()
const items = ref<ModelTokenUsageItem[]>([])
const summaryUsed = ref(0)
const summaryDailyLimit = ref<number | null>(null)
const total = ref(0)
const datesValid = computed(() => Boolean(report.startDate.value && report.endDate.value))

// Model search
const modelSearch = ref(report.targetLabel.value)
const modelOptions = ref<TokenUsageOption[]>([])
const showModelDropdown = ref(false)
let modelSearchTimer: ReturnType<typeof setTimeout> | null = null

// Sort
const sortBy = ref<TokenUsageSortField>('usage_date')
const sortOrder = ref<'asc' | 'desc'>('desc')

// 鈹€鈹€ Model search 鈹€鈹€

async function onModelSearchInput() {
  if (modelSearch.value !== report.targetLabel.value) report.setTarget('', '')
  if (modelSearchTimer) clearTimeout(modelSearchTimer)
  modelSearchTimer = setTimeout(async () => {
    const q = modelSearch.value.trim()
    if (!q) {
      modelOptions.value = []
      return
    }
    try {
      const res = await searchModelOptions(q, 20)
      modelOptions.value = res.items
      showModelDropdown.value = true
    } catch {
      // ignore
    }
  }, 300)
}

function selectModel(opt: TokenUsageOption) {
  modelSearch.value = opt.label
  modelOptions.value = []
  showModelDropdown.value = false
  report.setTarget(opt.model ?? opt.label, opt.label)
}

function hideModelDropdown() {
  setTimeout(() => { showModelDropdown.value = false }, 150)
}

// 鈹€鈹€ Query 鈹€鈹€

async function doQuery() {
  if (!datesValid.value) return
  const { seq } = report.nextSignal()
  report.error.value = null
  report.loading.value = true
  try {
    const res = await getModelTokenUsageReport({
      ...(report.targetId.value ? { model: report.targetId.value } : {}),
      start_date: report.startDate.value,
      end_date: report.endDate.value,
      page: report.page.value,
      page_size: report.pageSize.value,
      sort_by: sortBy.value,
      sort_order: sortOrder.value
    })
    if (!report.isCurrent(seq)) return
    items.value = res.items
    summaryUsed.value = res.summary.used_tokens
    total.value = res.pagination.total
    // Find first item with a limit to display summary rate
    const limitedItem = res.items.find(i => i.daily_limit_tokens && i.daily_limit_tokens > 0)
    summaryDailyLimit.value = limitedItem?.daily_limit_tokens ?? null
  } catch (e: unknown) {
    if (!report.isAbortError(e) && report.isCurrent(seq)) {
      report.error.value = (e as any)?.message || t('tokenUsageReport.queryFailed')
    }
  } finally {
    if (report.isCurrent(seq)) report.loading.value = false
  }
}

// 鈹€鈹€ Date / Page / Sort handlers 鈹€鈹€

function onStartDateChange(e: Event) {
  const val = (e.target as HTMLInputElement).value
  report.setDates(val, report.endDate.value)
}

function onEndDateChange(e: Event) {
  const val = (e.target as HTMLInputElement).value
  report.setDates(report.startDate.value, val)
}

async function onPageChange(page: number) {
  await report.setPage(page)
  doQuery()
}

async function onPageSizeChange(size: number) {
  await report.setPageSize(size)
  doQuery()
}

watch([sortBy, sortOrder], () => {
  report.resetPage()
  doQuery()
})

// 鈹€鈹€ Init 鈹€鈹€

onMounted(() => {
  // Restore label from URL if available
  if (report.targetLabel.value) {
    modelSearch.value = report.targetLabel.value
  }
  doQuery()
})
</script>

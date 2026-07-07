<template>
  <AppLayout>
  <TokenUsageReportLayout
    :title="t('tokenUsageReport.userTitle')"
    :loading="report.loading.value"
    :error="report.error.value"
    :has-data="items.length > 0"
    @refresh="doQuery"
    @retry="doQuery"
  >
    <template #filters>
      <div class="flex flex-wrap items-end gap-3">
        <div class="flex-1 min-w-[200px]">
          <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{{ t('tokenUsageReport.user') }}</label>
          <input
            v-model="userSearch"
            type="text"
            class="input w-full"
            :placeholder="t('tokenUsageReport.searchUsers')"
            @input="onUserSearchInput"
            @focus="showUserDropdown = userOptions.length > 0"
            @blur="hideUserDropdown"
          />
          <div
            v-if="showUserDropdown && userOptions.length > 0"
            class="absolute z-10 mt-1 w-[calc(100%-1.5rem)] rounded-md border border-gray-200 bg-white shadow-lg dark:border-dark-700 dark:bg-dark-800 max-h-48 overflow-auto"
          >
            <button
              v-for="opt in userOptions"
              :key="opt.id"
              class="block w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
              @mousedown.prevent="selectUser(opt)"
            >
              {{ opt.label }}
            </button>
          </div>
        </div>
        <div class="flex-1 min-w-[150px]">
          <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{{ t('tokenUsageReport.optionalModel') }}</label>
          <input
            v-model="modelSearch"
            type="text"
            class="input w-full"
            :placeholder="t('tokenUsageReport.searchModel')"
            @input="onModelSearchInput"
            @focus="showModelDropdown = modelOptions.length > 0"
            @blur="hideModelDropdown"
          />
          <div
            v-if="showModelDropdown && modelOptions.length > 0"
            class="absolute z-10 mt-1 w-[calc(100%-1.5rem)] rounded-md border border-gray-200 bg-white shadow-lg dark:border-dark-700 dark:bg-dark-800 max-h-48 overflow-auto"
          >
            <button
              v-for="opt in modelOptions"
              :key="opt.id"
              class="block w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700"
              @mousedown.prevent="selectModel(opt)"
            >
              {{ opt.label }}
            </button>
          </div>
        </div>
        <div class="flex gap-2">
          <div>
            <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{{ t('tokenUsageReport.start') }}</label>
            <input
              v-model="startDateModel"
              type="date"
              class="input text-sm"
              :max="endDateModel"
            />
          </div>
          <div>
            <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{{ t('tokenUsageReport.end') }}</label>
            <input
              v-model="endDateModel"
              type="date"
              class="input text-sm"
              :min="startDateModel"
            />
          </div>
        </div>
        <div class="min-w-[170px]">
          <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{{ t('tokenUsageReport.deletedUsers') }}</label>
          <select v-model="includeDeleted" class="input w-full text-sm" @change="onIncludeDeletedChange">
            <option :value="false">{{ t('tokenUsageReport.excludeDeletedUsers') }}</option>
            <option :value="true">{{ t('tokenUsageReport.includeDeletedUsers') }}</option>
          </select>
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{{ t('tokenUsageReport.sortBy') }}</label>
          <select v-model="sortBy" class="input text-sm">
            <option value="usage_date">{{ t('tokenUsageReport.date') }}</option>
            <option value="user">{{ t('tokenUsageReport.user') }}</option>
            <option value="user_deleted">{{ t('tokenUsageReport.deletedUsers') }}</option>
            <option value="model">{{ t('tokenUsageReport.model') }}</option>
            <option value="used_tokens">{{ t('tokenUsageReport.usedTokens') }}</option>
            <option value="daily_limit_tokens">{{ t('tokenUsageReport.dailyLimit') }}</option>
            <option value="usage_rate">{{ t('tokenUsageReport.usageRate') }}</option>
            <option value="status">{{ t('tokenUsageReport.status') }}</option>
          </select>
        </div>
        <div>
          <label class="block text-xs font-medium text-gray-500 dark:text-gray-400 mb-1">{{ t('tokenUsageReport.order') }}</label>
          <select v-model="sortOrder" class="input text-sm">
            <option value="desc">{{ t('tokenUsageReport.descending') }}</option>
            <option value="asc">{{ t('tokenUsageReport.ascending') }}</option>
          </select>
        </div>
        <div>
          <button
            class="btn btn-primary btn-sm"
            :disabled="!canQuery"
            @click="doQuery"
          >
            {{ t('tokenUsageReport.query') }}
          </button>
        </div>
      </div>
    </template>

    <template #summary>
      <TokenUsageSummary
        :used-tokens="summaryUsed"
        :daily-limit-tokens="summaryLimit"
      />
    </template>

    <template #table>
      <UserModelTokenUsageTable :items="items" />
    </template>

    <template #pagination>
      <Pagination
        v-if="pagination.total > 0"
        :total="pagination.total"
        :page="pagination.page"
        :page-size="pagination.page_size"
        :show-page-size-selector="true"
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
import UserModelTokenUsageTable from '@/components/admin/token-usage/UserModelTokenUsageTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import { useTokenUsageReport } from '@/composables/useTokenUsageReport'
import { tokenUsageAPI } from '@/api/admin/tokenUsage'
import type { TokenUsageOption, UserModelTokenUsageItem, UserReportParams, TokenUsageSortField } from '@/api/admin/tokenUsage'

const report = useTokenUsageReport({ paramPrefix: 'user' })
const { t } = useI18n()

const items = ref<UserModelTokenUsageItem[]>([])
const summaryUsed = ref(0)
const summaryLimit = ref<number | null>(null)
const pagination = ref({ page: 1, page_size: 20, total: 0 })

const userSearch = ref('')
const userOptions = ref<TokenUsageOption[]>([])
const showUserDropdown = ref(false)
const modelSearch = ref('')
const modelOptions = ref<TokenUsageOption[]>([])
const showModelDropdown = ref(false)
const includeDeleted = ref(false)
const sortBy = ref<TokenUsageSortField>('usage_date')
const sortOrder = ref<'asc' | 'desc'>('desc')

let userDebounceTimer: ReturnType<typeof setTimeout> | null = null
let modelDebounceTimer: ReturnType<typeof setTimeout> | null = null

// Sync local refs with composable
const startDateModel = computed({
  get: () => report.startDate.value,
  set: (v) => report.setDates(v, report.endDate.value)
})
const endDateModel = computed({
  get: () => report.endDate.value,
  set: (v) => report.setDates(report.startDate.value, v)
})

const canQuery = computed(() => Boolean(report.startDate.value && report.endDate.value) && !report.loading.value)

function onUserSearchInput() {
  if (userSearch.value !== report.targetLabel.value) report.setTarget('', '')
  if (userDebounceTimer) clearTimeout(userDebounceTimer)
  userDebounceTimer = setTimeout(async () => {
    try {
      const res = await tokenUsageAPI.searchUsers(userSearch.value, 20)
      userOptions.value = includeDeleted.value ? res.items : res.items.filter(item => !item.label.endsWith(' (deleted)'))
      showUserDropdown.value = userOptions.value.length > 0
    } catch { /* ignore */ }
  }, 300)
}

function onModelSearchInput() {
  if (modelDebounceTimer) clearTimeout(modelDebounceTimer)
  modelDebounceTimer = setTimeout(async () => {
    try {
      const uid = Number(report.targetId.value)
      const res = uid
        ? await tokenUsageAPI.searchUserModels(uid, modelSearch.value, 20)
        : await tokenUsageAPI.searchModels(modelSearch.value, 20)
      modelOptions.value = res.items
      showModelDropdown.value = res.items.length > 0
    } catch { /* ignore */ }
  }, 300)
}

function selectUser(opt: TokenUsageOption) {
  userSearch.value = opt.label
  report.setTarget(String(opt.id), opt.label)
  userOptions.value = []
  showUserDropdown.value = false
  items.value = []
}

function selectModel(opt: TokenUsageOption) {
  modelSearch.value = opt.label
  modelOptions.value = []
  showModelDropdown.value = false
}

function hideUserDropdown() { setTimeout(() => { showUserDropdown.value = false }, 200) }
function hideModelDropdown() { setTimeout(() => { showModelDropdown.value = false }, 200) }

function onIncludeDeletedChange() {
  userOptions.value = []
  showUserDropdown.value = false
  report.resetPage()
  if (!includeDeleted.value && report.targetLabel.value.endsWith(' (deleted)')) {
    userSearch.value = ''
    report.setTarget('', '')
  }
}

async function doQuery() {
  if (!canQuery.value) return
  const { seq } = report.nextSignal()
  report.loading.value = true
  report.error.value = null
  try {
    const params: UserReportParams = {
      start_date: report.startDate.value,
      end_date: report.endDate.value,
      page: report.page.value,
      page_size: report.pageSize.value,
      sort_by: sortBy.value,
      sort_order: sortOrder.value,
      include_deleted: includeDeleted.value
    }
    if (report.targetId.value) params.user_id = Number(report.targetId.value)
    if (modelSearch.value.trim()) params.model = modelSearch.value.trim()
    const res = await tokenUsageAPI.getUserReport(params)
    if (!report.isCurrent(seq)) return
    items.value = res.items
    summaryUsed.value = res.summary.used_tokens
    pagination.value = res.pagination
  } catch (e: any) {
    if (report.isAbortError(e)) return
    if (!report.isCurrent(seq)) return
    report.error.value = e?.message || t('tokenUsageReport.loadFailed')
  } finally {
    if (report.isCurrent(seq)) report.loading.value = false
  }
}

function onPageChange(page: number) { report.setPage(page); doQuery() }
function onPageSizeChange(size: number) { report.setPageSize(size); doQuery() }

watch([sortBy, sortOrder], async () => {
  await report.setPage(1)
  await doQuery()
})

onMounted(() => {
  doQuery()
})
</script>

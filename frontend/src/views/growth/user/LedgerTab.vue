<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LedgerItem } from '@/api/growth'
import EmptyState from '@/components/common/EmptyState.vue'
import GrowthStatusBadge from '@/components/growth/GrowthStatusBadge.vue'
import PointsAmount from '@/components/growth/PointsAmount.vue'

const props = defineProps<{
  items: LedgerItem[]
  hasMore: boolean
  loadingMore: boolean
}>()

const emit = defineEmits<{
  typeChange: [type: string]
  loadMore: []
  overview: []
}>()

const { t } = useI18n()
const typeFilter = ref('')
const timeRange = ref('all')
const expanded = ref<number>()
const visibleItems = computed(() => {
  const now = Date.now()
  const maxAge = timeRange.value === '7d' ? 7 : timeRange.value === '30d' ? 30 : 0
  if (!maxAge) return props.items
  return props.items.filter(item => now - new Date(item.created_at).getTime() <= maxAge * 24 * 60 * 60 * 1000)
})

function changeType(value: string) {
  typeFilter.value = value
  emit('typeChange', value)
}

const dateTime = (value: string) => new Date(value).toLocaleString()
</script>

<template>
  <div class="space-y-4">
    <div class="flex flex-col gap-3 sm:flex-row sm:items-center">
      <select :value="typeFilter" class="input sm:w-44" @change="changeType(($event.target as HTMLSelectElement).value)">
        <option value="">{{ t('growth.user.ledger.filterType') }}</option>
        <option value="EARN">{{ t('growth.status.EARN') }}</option>
        <option value="REDEEM">{{ t('growth.status.REDEEM') }}</option>
        <option value="REFUND">{{ t('growth.status.REFUND') }}</option>
        <option value="ADJUST">{{ t('growth.status.ADJUST') }}</option>
      </select>
      <select v-model="timeRange" class="input sm:w-44">
        <option value="all">{{ t('growth.user.ledger.allTime') }}</option>
        <option value="7d">{{ t('growth.user.ledger.last7Days') }}</option>
        <option value="30d">{{ t('growth.user.ledger.last30Days') }}</option>
      </select>
      <div class="flex-1" />
      <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.user.ledger.loaded', { count: items.length }) }}</span>
    </div>

    <div v-if="!visibleItems.length" class="card">
      <EmptyState :title="t('growth.user.ledger.empty')" :description="t('growth.user.ledger.emptyHint')" :action-text="t('growth.user.ledger.goOverview')" @action="emit('overview')" />
    </div>
    <div v-else class="card overflow-hidden">
      <div class="hidden overflow-x-auto sm:block">
        <table class="w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
          <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            <tr>
              <th class="px-4 py-3">{{ t('growth.user.ledger.time') }}</th>
              <th class="px-4 py-3">{{ t('growth.user.ledger.type') }}</th>
              <th class="px-4 py-3">{{ t('growth.user.ledger.description') }}</th>
              <th class="px-4 py-3 text-right">{{ t('growth.user.ledger.change') }}</th>
              <th class="px-4 py-3 text-right">{{ t('growth.user.ledger.balance') }}</th>
              <th class="px-4 py-3">{{ t('growth.user.ledger.reference') }}</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
            <tr v-for="entry in visibleItems" :key="entry.entry_id" class="cursor-pointer hover:bg-gray-50 dark:hover:bg-dark-800/60" @click="expanded = expanded === entry.entry_id ? undefined : entry.entry_id">
              <td class="px-4 py-3 text-xs text-gray-500 dark:text-dark-400">{{ dateTime(entry.created_at) }}</td>
              <td class="px-4 py-3"><GrowthStatusBadge :status="entry.type" /></td>
              <td class="px-4 py-3">{{ entry.title }}</td>
              <td class="px-4 py-3 text-right"><PointsAmount :value="entry.points" :direction="entry.direction" signed /></td>
              <td class="px-4 py-3 text-right tabular-nums text-gray-700 dark:text-dark-200">{{ entry.balance_after ?? '-' }}</td>
              <td class="px-4 py-3 font-mono text-xs text-gray-500 dark:text-dark-400">{{ entry.biz_type }}#{{ entry.transfer_id }}</td>
            </tr>
          </tbody>
        </table>
      </div>
      <div class="divide-y divide-gray-100 dark:divide-dark-700 sm:hidden">
        <button v-for="entry in visibleItems" :key="entry.entry_id" class="block w-full p-4 text-left" @click="expanded = expanded === entry.entry_id ? undefined : entry.entry_id">
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0">
              <div class="font-medium text-gray-900 dark:text-white">{{ entry.title }}</div>
              <div class="mt-1 text-xs text-gray-500 dark:text-dark-400">{{ dateTime(entry.created_at) }}</div>
            </div>
            <PointsAmount :value="entry.points" :direction="entry.direction" signed />
          </div>
          <div v-if="expanded === entry.entry_id" class="mt-3 rounded-lg bg-gray-50 p-2 text-xs text-gray-500 dark:bg-dark-800 dark:text-dark-400">
            {{ t('growth.user.ledger.balance') }}: {{ entry.balance_after ?? '-' }} · {{ entry.biz_type }}#{{ entry.transfer_id }}
          </div>
        </button>
      </div>
    </div>
    <div v-if="hasMore" class="text-center">
      <button class="btn btn-secondary" :disabled="loadingMore" @click="emit('loadMore')">{{ loadingMore ? t('growth.common.loading') : t('growth.user.ledger.loadMore') }}</button>
    </div>
  </div>
</template>

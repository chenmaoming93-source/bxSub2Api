<template>
  <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
    <thead class="bg-gray-50 dark:bg-dark-800">
      <tr>
        <th class="w-[7.5rem] px-3 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
          {{ t('tokenUsageReport.date') }}
        </th>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
          {{ t('tokenUsageReport.user') }}
        </th>
        <th class="px-4 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
          {{ t('tokenUsageReport.model') }}
        </th>
        <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
          {{ t('tokenUsageReport.usedTokens') }}
        </th>
        <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
          {{ t('tokenUsageReport.dailyLimit') }}
        </th>
        <th class="px-4 py-3 text-right text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
          {{ t('tokenUsageReport.usageRate') }}
        </th>
        <th class="px-4 py-3 text-center text-xs font-medium uppercase text-gray-500 dark:text-gray-400">
          {{ t('tokenUsageReport.status') }}
        </th>
      </tr>
    </thead>
    <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
      <tr v-for="item in items" :key="`${item.usage_date}-${item.user_id}-${item.model}`">
        <td class="w-[7.5rem] whitespace-nowrap px-3 py-3 text-sm text-gray-900 dark:text-white">
          {{ item.usage_date }}
        </td>
        <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
          <template v-if="item.email">{{ item.email }}</template>
          <template v-else-if="item.username">{{ item.username }}</template>
          <template v-else>{{ t('tokenUsageReport.userId', { id: item.user_id }) }}</template>
          <span v-if="item.user_deleted" class="ml-1 text-xs text-red-500">({{ t('tokenUsageReport.deleted') }})</span>
        </td>
        <td class="whitespace-nowrap px-4 py-3 text-sm text-gray-700 dark:text-gray-300">
          {{ item.model }}
        </td>
        <td class="whitespace-nowrap px-4 py-3 text-right text-sm text-gray-900 dark:text-white">
          {{ formatNumber(item.used_tokens) }}
        </td>
        <td class="whitespace-nowrap px-4 py-3 text-right text-sm text-gray-500 dark:text-gray-400">
          <template v-if="item.daily_limit_tokens !== null && item.daily_limit_tokens !== undefined">
            {{ formatNumber(item.daily_limit_tokens) }}
          </template>
          <span v-else class="text-gray-400">—</span>
        </td>
        <td class="whitespace-nowrap px-4 py-3 text-right text-sm">
          <UsageRateCell :rate="item.usage_rate" />
        </td>
        <td class="whitespace-nowrap px-4 py-3 text-center text-sm">
          <UsageStatusBadge :status="item.status" />
        </td>
      </tr>
    </tbody>
  </table>
</template>

<script setup lang="ts">
import UsageStatusBadge from '@/components/admin/token-usage/UsageStatusBadge.vue'
import UsageRateCell from '@/components/admin/token-usage/UsageRateCell.vue'
import { useI18n } from 'vue-i18n'
import type { UserModelTokenUsageItem } from '@/api/admin/tokenUsage'
const { t } = useI18n()

interface Props {
  items: UserModelTokenUsageItem[]
}

defineProps<Props>()

function formatNumber(n: number): string {
  return n.toLocaleString()
}
</script>

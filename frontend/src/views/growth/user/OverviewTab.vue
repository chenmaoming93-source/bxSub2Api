<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { LedgerItem, UserOverview, UserProduct, UserTask } from '@/api/growth'
import EmptyState from '@/components/common/EmptyState.vue'
import ProductCard from '@/components/growth/ProductCard.vue'
import PointsAmount from '@/components/growth/PointsAmount.vue'

const props = defineProps<{
  overview: UserOverview
  tasks: UserTask[]
  products: UserProduct[]
  ledger: LedgerItem[]
}>()

const emit = defineEmits<{
  tab: [key: string]
  product: [product: UserProduct]
  aiwork: []
}>()

const { t } = useI18n()
const previewProducts = computed(() => props.products.slice(0, 4))
const dateTime = (value: string) => new Date(value).toLocaleString()
</script>

<template>
  <div class="grid gap-5 lg:grid-cols-2">
    <section class="card">
      <div class="card-header flex items-center justify-between">
        <h2 class="card-title">{{ t('growth.user.overview.tasks') }}</h2>
        <span class="text-xs text-gray-500 dark:text-dark-400">{{ t('growth.user.overview.taskGuide') }}</span>
      </div>
      <EmptyState v-if="!tasks.length" :title="t('growth.user.overview.noTasks')" />
      <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
        <div v-for="task in tasks" :key="task.task_id" class="flex flex-wrap items-center gap-3 p-4">
          <div class="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-amber-50 text-lg text-amber-600 dark:bg-amber-500/10 dark:text-amber-300">✦</div>
          <div class="min-w-0 flex-1">
            <div class="font-medium text-gray-900 dark:text-white">{{ task.title }}</div>
            <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ task.description }}</div>
            <div v-if="task.completed_today" class="mt-1 text-xs text-teal-600 dark:text-teal-300">{{ t('growth.user.overview.completedToday', { count: task.completed_today }) }}</div>
          </div>
          <PointsAmount :value="task.points" direction="IN" signed />
          <button class="btn btn-sm btn-secondary" @click="emit('aiwork')">{{ t('growth.user.overview.goComplete') }}</button>
        </div>
      </div>
    </section>

    <section class="card">
      <div class="card-header flex items-center justify-between">
        <h2 class="card-title">{{ t('growth.user.overview.recentLedger') }}</h2>
        <button class="text-sm font-medium text-primary-600 dark:text-primary-400" @click="emit('tab', 'ledger')">{{ t('growth.user.overview.viewAll') }}</button>
      </div>
      <EmptyState v-if="!ledger.length" :title="t('growth.user.ledger.empty')" :description="t('growth.user.ledger.emptyHint')" />
      <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
        <div v-for="entry in ledger.slice(0, 5)" :key="entry.entry_id" class="grid grid-cols-[1fr_auto] gap-2 p-4 text-sm">
          <div>
            <div class="font-medium text-gray-800 dark:text-dark-100">{{ entry.title }}</div>
            <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ dateTime(entry.created_at) }}</div>
          </div>
          <div class="text-right">
            <PointsAmount :value="entry.points" :direction="entry.direction" signed />
            <div class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">{{ entry.balance_after ?? '-' }}</div>
          </div>
        </div>
      </div>
    </section>

    <section class="lg:col-span-2">
      <div class="mb-3 flex items-center justify-between">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('growth.user.overview.storePreview') }}</h2>
        <button class="text-sm font-medium text-primary-600 dark:text-primary-400" @click="emit('tab', 'store')">{{ t('growth.user.overview.viewAll') }}</button>
      </div>
      <EmptyState v-if="!previewProducts.length" :title="t('growth.user.store.empty')" :description="t('growth.user.store.emptyHint')" />
      <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <ProductCard v-for="product in previewProducts" :key="product.product_id" :product="product" :balance="overview.available_points" @open="emit('product', $event)" />
      </div>
    </section>
  </div>
</template>

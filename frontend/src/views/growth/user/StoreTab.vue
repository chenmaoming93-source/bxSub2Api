<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserExchange, UserProduct } from '@/api/growth'
import EmptyState from '@/components/common/EmptyState.vue'
import GrowthStatusBadge from '@/components/growth/GrowthStatusBadge.vue'
import PointsAmount from '@/components/growth/PointsAmount.vue'
import ProductCard from '@/components/growth/ProductCard.vue'

const props = defineProps<{
  products: UserProduct[]
  exchanges: UserExchange[]
  balance: number
}>()

const emit = defineEmits<{
  product: [product: UserProduct]
}>()

const { t } = useI18n()
const search = ref('')
const sort = ref('recommended')
const visibleProducts = computed(() => {
  const filtered = props.products.filter(product => {
    const needle = search.value.trim().toLowerCase()
    return !needle || product.name.toLowerCase().includes(needle) || product.description.toLowerCase().includes(needle)
  })
  if (sort.value === 'points_asc') return [...filtered].sort((a, b) => a.points_cost - b.points_cost)
  if (sort.value === 'points_desc') return [...filtered].sort((a, b) => b.points_cost - a.points_cost)
  return filtered
})
</script>

<template>
  <div class="space-y-5">
    <div class="flex flex-col gap-3 sm:flex-row">
      <input v-model="search" class="input min-w-0 flex-1" :placeholder="t('growth.user.store.searchPlaceholder')">
      <select v-model="sort" class="input sm:w-48">
        <option value="recommended">{{ t('growth.user.store.recommended') }}</option>
        <option value="points_asc">{{ t('growth.user.store.pointsAsc') }}</option>
        <option value="points_desc">{{ t('growth.user.store.pointsDesc') }}</option>
      </select>
    </div>
    <EmptyState v-if="!visibleProducts.length" :title="t('growth.user.store.empty')" :description="t('growth.user.store.emptyHint')" />
    <div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
      <ProductCard v-for="product in visibleProducts" :key="product.product_id" :product="product" :balance="balance" @open="emit('product', $event)" />
    </div>

    <section class="card">
      <div class="card-header">
        <h2 class="card-title">{{ t('growth.user.store.exchanges') }}</h2>
      </div>
      <EmptyState v-if="!exchanges.length" :title="t('growth.user.store.noExchanges')" />
      <div v-else class="divide-y divide-gray-100 dark:divide-dark-700">
        <div v-for="exchange in exchanges" :key="exchange.order_id" class="flex flex-wrap items-center justify-between gap-2 p-4 text-sm">
          <div>
            <div class="font-medium text-gray-900 dark:text-white">{{ exchange.product_title }}</div>
            <div class="mt-0.5 font-mono text-xs text-gray-500 dark:text-dark-400">{{ exchange.order_no }} · v{{ exchange.product_version }}</div>
          </div>
          <div class="flex items-center gap-3">
            <PointsAmount :value="exchange.points_cost" direction="OUT" />
            <GrowthStatusBadge :status="exchange.status" />
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

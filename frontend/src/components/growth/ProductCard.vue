<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { UserProduct } from '@/api/growth'
import PointsAmount from './PointsAmount.vue'
import { productEligibilityText } from './growthRules'

const props = defineProps<{
  product: UserProduct
  balance: number
}>()

const emit = defineEmits<{
  open: [product: UserProduct]
}>()

const { t } = useI18n()
const eligibility = computed(() => productEligibilityText(props.product, props.product.reason, props.balance))
const label = computed(() => t(`growth.user.store.${eligibility.value.key}`, eligibility.value.params ?? {}))
const stockText = computed(() => props.product.stock < 0
  ? t('growth.user.store.unlimited')
  : t('growth.user.store.stock', { count: props.product.stock }))
</script>

<template>
  <button
    type="button"
    class="card group flex min-h-52 flex-col p-5 text-left transition hover:-translate-y-0.5 hover:border-teal-300 hover:shadow-md dark:hover:border-teal-700"
    @click="emit('open', product)"
  >
    <div class="flex items-start justify-between gap-3">
      <div class="flex h-11 w-11 items-center justify-center rounded-xl bg-primary-50 text-xl text-primary-600 transition group-hover:bg-teal-50 group-hover:text-teal-600 dark:bg-primary-500/10 dark:text-primary-300 dark:group-hover:bg-teal-500/10 dark:group-hover:text-teal-300">◇</div>
      <span class="rounded-full px-2 py-1 text-xs font-medium" :class="product.can_exchange
        ? 'bg-teal-50 text-teal-700 dark:bg-teal-500/10 dark:text-teal-300'
        : 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-dark-300'">
        {{ label }}
      </span>
    </div>
    <h3 class="mt-4 font-semibold text-gray-900 dark:text-white">{{ product.name }}</h3>
    <p class="mt-1 flex-1 text-sm leading-6 text-gray-500 dark:text-dark-400">{{ product.description }}</p>
    <div class="mt-4 flex items-end justify-between gap-2">
      <PointsAmount :value="product.points_cost" unit="PT" />
      <span class="text-xs text-gray-500 dark:text-dark-400">{{ stockText }}</span>
    </div>
    <div v-if="product.per_user_limit > 0" class="mt-1 text-right text-xs text-gray-400 dark:text-dark-500">{{ t('growth.user.store.limit', { count: product.per_user_limit }) }}</div>
  </button>
</template>

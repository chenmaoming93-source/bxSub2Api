<template>
  <span class="rounded-full px-2 py-0.5 text-xs" :class="className">{{ statusLabel }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useObservabilityI18n } from './useObservabilityI18n'

const props = defineProps<{ status: string }>()
const { t } = useObservabilityI18n()
const statusLabel = computed(() => {
  if (props.status === 'healthy') return t('admin.observability.healthy')
  if (props.status === 'degraded') return t('admin.observability.degraded')
  if (props.status === 'error') return t('admin.observability.errorStatus')
  return props.status || t('admin.observability.unknown')
})
const className = computed(() => {
  if (props.status === 'healthy') return 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/40 dark:text-emerald-300'
  if (props.status === 'degraded') return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
  if (props.status === 'error') return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
  return 'bg-gray-100 text-gray-600 dark:bg-dark-700 dark:text-gray-300'
})
</script>

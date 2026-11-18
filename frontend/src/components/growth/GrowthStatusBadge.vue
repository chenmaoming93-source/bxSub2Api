<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ status: string }>()
const { t, te } = useI18n()
const label = computed(() => te(`growth.status.${props.status}`) ? t(`growth.status.${props.status}`) : props.status)
const cls = computed(() => {
  if (['ACTIVE', 'SUCCEEDED', 'REWARDED'].includes(props.status)) return 'bg-teal-50 text-teal-700 ring-teal-600/20 dark:bg-teal-500/10 dark:text-teal-300'
  if (['FAILED', 'OFFLINE'].includes(props.status)) return 'bg-rose-50 text-rose-700 ring-rose-600/20 dark:bg-rose-500/10 dark:text-rose-300'
  if (['DRAFT', 'FULFILLING', 'PENDING', 'PROCESSING', 'IN_PROGRESS'].includes(props.status)) return 'bg-amber-50 text-amber-700 ring-amber-600/20 dark:bg-amber-500/10 dark:text-amber-300'
  if (['REFUNDED', 'CREATED'].includes(props.status)) return 'bg-sky-50 text-sky-700 ring-sky-600/20 dark:bg-sky-500/10 dark:text-sky-300'
  return 'bg-gray-100 text-gray-700 ring-gray-500/20 dark:bg-dark-700 dark:text-dark-200'
})
</script>

<template>
  <span class="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset" :class="cls">{{ label }}</span>
</template>

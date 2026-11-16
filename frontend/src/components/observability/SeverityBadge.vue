<template>
  <span class="rounded-full px-2 py-0.5 text-xs font-medium" :class="className">
    {{ severityLabel }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useObservabilityI18n } from './useObservabilityI18n'

const props = defineProps<{ severity: string }>()
const { t } = useObservabilityI18n()
const severityLabel = computed(() => {
  if (props.severity === 'critical') return t('admin.observability.critical')
  if (props.severity === 'warning') return t('admin.observability.warning')
  if (props.severity === 'info') return t('admin.observability.info')
  return props.severity || t('admin.observability.unknownSeverity')
})
const className = computed(() => {
  if (['P0', 'P1', 'critical'].includes(props.severity)) return 'bg-red-100 text-red-700 dark:bg-red-900/40 dark:text-red-300'
  if (['P2', 'warning'].includes(props.severity)) return 'bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300'
  return 'bg-blue-100 text-blue-700 dark:bg-blue-900/40 dark:text-blue-300'
})
</script>

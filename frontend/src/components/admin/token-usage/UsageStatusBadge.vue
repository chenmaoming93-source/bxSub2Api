<template>
  <span
    :class="badgeClass"
    class="inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium"
  >
    <span
      :class="dotClass"
      class="inline-block h-1.5 w-1.5 rounded-full"
    ></span>
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

interface Props {
  status: string
}

const props = defineProps<Props>()

const badgeClass = computed(() => {
  switch (props.status) {
    case 'exceeded':
      return 'bg-red-100 text-red-800 dark:bg-red-900/30 dark:text-red-400'
    case 'warning':
      return 'bg-yellow-100 text-yellow-800 dark:bg-yellow-900/30 dark:text-yellow-400'
    case 'normal':
      return 'bg-green-100 text-green-800 dark:bg-green-900/30 dark:text-green-400'
    case 'unlimited':
    default:
      return 'bg-gray-100 text-gray-600 dark:bg-gray-800 dark:text-gray-400'
  }
})

const dotClass = computed(() => {
  switch (props.status) {
    case 'exceeded':
      return 'bg-red-500'
    case 'warning':
      return 'bg-yellow-500'
    case 'normal':
      return 'bg-green-500'
    case 'unlimited':
    default:
      return 'bg-gray-400'
  }
})

const label = computed(() => {
  switch (props.status) {
    case 'exceeded':
      return t('tokenUsageReport.exceeded')
    case 'warning':
      return t('tokenUsageReport.warning')
    case 'normal':
      return t('tokenUsageReport.normal')
    case 'unlimited':
      return t('tokenUsageReport.unlimited')
    default:
      return props.status
  }
})
</script>

<template>
  <div
    class="flex flex-wrap gap-4 rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800"
  >
    <div class="flex items-center gap-2">
      <span class="text-sm text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.totalTokens') }}</span>
      <span class="text-lg font-semibold text-gray-900 dark:text-white">
        {{ formattedTokens }}
      </span>
    </div>
    <div v-if="usageRate !== null" class="flex items-center gap-2">
      <span class="text-sm text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.usageRate') }}</span>
      <div class="flex items-center gap-1.5">
        <div class="h-2 w-24 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
          <div
            :class="rateBarClass"
            :style="{ width: `${Math.min(usageRate * 100, 100)}%` }"
            class="h-full rounded-full transition-all"
          ></div>
        </div>
        <span class="text-sm font-medium text-gray-700 dark:text-gray-300">
          {{ (usageRate * 100).toFixed(1) }}%
        </span>
      </div>
    </div>
    <slot name="extra" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
const { t } = useI18n()

interface Props {
  usedTokens: number
  dailyLimitTokens?: number | null
}

const props = withDefaults(defineProps<Props>(), {
  dailyLimitTokens: null
})

const formattedTokens = computed(() => {
  if (props.usedTokens >= 1_000_000_000) {
    return `${(props.usedTokens / 1_000_000_000).toFixed(1)}B`
  }
  if (props.usedTokens >= 1_000_000) {
    return `${(props.usedTokens / 1_000_000).toFixed(1)}M`
  }
  if (props.usedTokens >= 1_000) {
    return `${(props.usedTokens / 1_000).toFixed(1)}K`
  }
  return String(props.usedTokens)
})

const usageRate = computed<number | null>(() => {
  if (!props.dailyLimitTokens || props.dailyLimitTokens <= 0) return null
  return props.usedTokens / props.dailyLimitTokens
})

const rateBarClass = computed(() => {
  const rate = usageRate.value
  if (rate === null) return 'bg-gray-400'
  if (rate >= 1) return 'bg-red-500'
  if (rate >= 0.8) return 'bg-yellow-500'
  return 'bg-green-500'
})
</script>

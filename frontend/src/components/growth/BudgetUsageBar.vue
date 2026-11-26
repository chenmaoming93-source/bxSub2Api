<script setup lang="ts">
import { computed } from 'vue'
import { quotaTone } from './growthRules'

const props = defineProps<{
  usageRate: number
  remaining: number
}>()

const tone = computed(() => quotaTone(props.remaining, props.usageRate))
const width = computed(() => `${Math.min(100, Math.max(0, props.usageRate * 100))}%`)
const barClass = computed(() => ({
  healthy: 'bg-teal-500',
  warning: 'bg-amber-500',
  tight: 'bg-rose-500',
  exhausted: 'bg-rose-500',
})[tone.value])
</script>

<template>
  <div class="flex min-w-32 items-center gap-2">
    <div class="h-2 flex-1 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
      <div class="h-full rounded-full transition-all" :class="barClass" :style="{ width }" />
    </div>
    <span class="w-12 text-right text-xs tabular-nums text-gray-600 dark:text-dark-300">{{ (usageRate * 100).toFixed(1) }}%</span>
  </div>
</template>

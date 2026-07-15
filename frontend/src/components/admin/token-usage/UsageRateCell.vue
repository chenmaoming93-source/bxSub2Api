<template>
  <div v-if="rate !== null" class="flex min-w-[8rem] items-center justify-end gap-2">
    <div class="h-2 w-20 overflow-hidden rounded-full bg-gray-200 dark:bg-dark-600">
      <div
        class="h-full rounded-full transition-all"
        :class="barClass"
        :style="{ width: `${Math.min(rate * 100, 100)}%` }"
      />
    </div>
    <span class="w-14 text-right">{{ (rate * 100).toFixed(1) }}%</span>
  </div>
  <span v-else class="text-gray-400">—</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{ rate: number | null }>()

const barClass = computed(() => {
  if (props.rate === null) return 'bg-gray-400'
  if (props.rate >= 1) return 'bg-red-500'
  if (props.rate >= 0.8) return 'bg-yellow-500'
  return 'bg-green-500'
})
</script>

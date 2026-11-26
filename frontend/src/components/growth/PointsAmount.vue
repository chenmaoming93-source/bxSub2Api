<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  value: number
  direction?: 'IN' | 'OUT'
  signed?: boolean
  unit?: string
}>(), {
  signed: false,
  unit: ''
})

const formatted = computed(() => {
  const absolute = Math.abs(props.value).toLocaleString()
  const sign = props.signed || props.direction
    ? props.direction === 'OUT' || (!props.direction && props.value < 0) ? '−' : '+'
    : ''
  return `${sign}${absolute}${props.unit ? ` ${props.unit}` : ''}`
})

const tone = computed(() => {
  if (!props.direction && !props.signed) return 'text-gray-900 dark:text-white'
  return props.direction === 'OUT' || (!props.direction && props.value < 0)
    ? 'text-rose-600 dark:text-rose-400'
    : 'text-teal-600 dark:text-teal-400'
})
</script>

<template>
  <span class="tabular-nums font-semibold" :class="tone">{{ formatted }}</span>
</template>

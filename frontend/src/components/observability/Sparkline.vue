<template>
  <svg
    v-if="points.length > 1"
    class="h-6 w-full"
    viewBox="0 0 100 24"
    preserveAspectRatio="none"
    aria-hidden="true"
  >
    <polyline
      :points="polylinePoints"
      fill="none"
      :stroke="color"
      stroke-width="2"
      stroke-linecap="round"
      stroke-linejoin="round"
      vector-effect="non-scaling-stroke"
    />
  </svg>
  <div v-else class="h-6" aria-hidden="true"></div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{
  points: number[]
  color?: string
}>(), {
  color: '#14b8a6',
})

const polylinePoints = computed(() => {
  if (props.points.length < 2) return ''
  const min = Math.min(...props.points)
  const max = Math.max(...props.points)
  const span = max - min || 1
  return props.points.map((value, index) => {
    const x = props.points.length === 1 ? 0 : (index / (props.points.length - 1)) * 100
    const y = 22 - ((value - min) / span) * 20
    return `${x.toFixed(2)},${y.toFixed(2)}`
  }).join(' ')
})
</script>

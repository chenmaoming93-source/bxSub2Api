<script setup lang="ts">
defineProps<{
  tone?: 'error' | 'warning' | 'info' | 'success'
}>()
</script>

<template>
  <div
    class="mb-4 rounded-xl border px-4 py-3 text-sm"
    :class="{
      'border-rose-200 bg-rose-50 text-rose-800 dark:border-rose-900/60 dark:bg-rose-950/30 dark:text-rose-200': tone === 'error',
      'border-amber-200 bg-amber-50 text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/30 dark:text-amber-200': tone === 'warning' || !tone,
      'border-sky-200 bg-sky-50 text-sky-800 dark:border-sky-900/60 dark:bg-sky-950/30 dark:text-sky-200': tone === 'info',
      'border-teal-200 bg-teal-50 text-teal-800 dark:border-teal-900/60 dark:bg-teal-950/30 dark:text-teal-200': tone === 'success',
    }"
    role="status"
  >
    <slot />
  </div>
</template>

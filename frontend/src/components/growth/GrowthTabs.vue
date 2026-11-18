<script setup lang="ts">
interface TabItem {
  key: string
  label: string
}

defineProps<{
  tabs: readonly TabItem[]
  active: string
}>()

const emit = defineEmits<{
  select: [key: string]
}>()
</script>

<template>
  <div class="mb-5 overflow-x-auto border-b border-gray-200 dark:border-dark-700">
    <nav class="-mb-px flex min-w-max gap-1" aria-label="Growth page tabs">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        type="button"
        class="border-b-2 px-3 py-2.5 text-sm font-medium transition-colors sm:px-4"
        :class="active === tab.key
          ? 'border-primary-500 text-primary-600 dark:text-primary-400'
          : 'border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-800 dark:text-dark-400 dark:hover:text-gray-200'"
        :aria-current="active === tab.key ? 'page' : undefined"
        @click="emit('select', tab.key)"
      >
        {{ tab.label }}
      </button>
    </nav>
  </div>
</template>

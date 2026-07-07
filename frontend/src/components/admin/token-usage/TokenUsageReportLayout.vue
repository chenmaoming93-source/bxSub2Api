<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex flex-wrap items-center justify-between gap-4">
      <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
        {{ title }}
      </h1>
      <div class="flex items-center gap-2">
        <button
          class="btn btn-ghost btn-sm"
          :disabled="loading"
          @click="$emit('refresh')"
        >
          <svg
            class="h-4 w-4"
            :class="{ 'animate-spin': loading }"
            fill="none"
            stroke="currentColor"
            viewBox="0 0 24 24"
          >
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
            />
          </svg>
        </button>
      </div>
    </div>

    <!-- Filter card -->
    <div
      class="rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800"
    >
      <slot name="filters" />
    </div>

    <!-- Loading skeleton -->
    <div v-if="loading && !hasData" class="space-y-3">
      <div
        v-for="i in 5"
        :key="i"
        class="h-10 animate-pulse rounded bg-gray-200 dark:bg-dark-700"
      ></div>
    </div>

    <!-- Error state -->
    <div
      v-else-if="error && !hasData"
      class="flex flex-col items-center justify-center rounded-lg border border-red-200 bg-red-50 p-8 dark:border-red-900 dark:bg-red-900/20"
    >
      <p class="mb-4 text-sm text-red-600 dark:text-red-400">
        {{ error }}
      </p>
      <button class="btn btn-primary btn-sm" @click="$emit('retry')">{{ t('tokenUsageReport.retry') }}</button>
    </div>

    <!-- Empty state -->
    <div
      v-else-if="!loading && !hasData"
      class="flex flex-col items-center justify-center rounded-lg border border-gray-200 bg-white p-12 dark:border-dark-700 dark:bg-dark-800"
    >
      <svg
        class="mb-4 h-12 w-12 text-gray-300 dark:text-gray-600"
        fill="none"
        stroke="currentColor"
        viewBox="0 0 24 24"
      >
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          stroke-width="1.5"
          d="M9 19v-6a2 2 0 00-2-2H5a2 2 0 00-2 2v6a2 2 0 002 2h2a2 2 0 002-2zm0 0V9a2 2 0 012-2h2a2 2 0 012 2v10m-6 0a2 2 0 002 2h2a2 2 0 002-2m0 0V5a2 2 0 012-2h2a2 2 0 012 2v14a2 2 0 01-2 2h-2a2 2 0 01-2-2z"
        />
      </svg>
      <p class="text-sm text-gray-500 dark:text-gray-400">
        {{ emptyMessage || t('tokenUsageReport.empty') }}
      </p>
    </div>

    <!-- Data content -->
    <template v-if="hasData">
      <slot name="summary" />

      <div class="overflow-x-auto rounded-lg border border-gray-200 dark:border-dark-700">
        <slot name="table" />
      </div>

      <slot name="pagination" />
    </template>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
const { t } = useI18n()
interface Props {
  title: string
  loading?: boolean
  error?: string | null
  hasData?: boolean
  emptyMessage?: string
}

withDefaults(defineProps<Props>(), {
  loading: false,
  error: null,
  hasData: false,
  emptyMessage: ''
})

defineEmits<{
  refresh: []
  retry: []
}>()
</script>

<template>
  <article class="min-w-0 rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
    <div class="mb-3 flex items-center justify-between gap-2">
      <div>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.observability.capacityTitle') }}</h2>
        <p class="mt-1 text-xs text-gray-500">{{ t('admin.observability.capacitySubtitle') }}</p>
      </div>
      <span class="rounded-full bg-primary-50 px-2 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">● {{ t('admin.observability.realtime') }}</span>
    </div>
    <div v-if="!capacity" class="flex h-52 items-center justify-center text-sm text-gray-400">{{ t('admin.observability.noCapacityData') }}</div>
    <template v-else>
      <div class="space-y-3">
        <div v-for="level in levels" :key="level.key">
          <div class="flex items-center justify-between gap-2 text-xs">
            <div class="min-w-0">
              <b class="block truncate text-gray-800 dark:text-gray-100">{{ level.data.name || level.label }}</b>
              <small class="text-gray-500">{{ level.label }}</small>
            </div>
            <span class="shrink-0 font-mono text-gray-700 dark:text-gray-200">
              {{ level.data.available ? `${level.data.current} / ${level.data.limit} · ${Math.round(level.data.utilization * 100)}% · ${t(statusFor(level.data.utilization).labelKey)}` : t('admin.observability.unavailable') }}
            </span>
          </div>
          <div class="mt-1 h-2 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-700">
            <div class="h-full rounded-full" :class="barClass(level.data.utilization)" :style="{ width: `${Math.min(100, level.data.utilization * 100)}%` }"></div>
          </div>
        </div>
      </div>
      <div class="mt-4 grid grid-cols-3 gap-2 border-t border-gray-200 pt-4 dark:border-dark-600">
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-700">
          <span class="text-xs text-gray-500">{{ t('admin.observability.queueCurrent') }}</span>
          <strong class="mt-1 block text-lg text-gray-900 dark:text-white">{{ capacity.queue_current }}</strong>
          <small class="text-gray-400">{{ t('admin.observability.queueDepth') }}</small>
        </div>
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-700">
          <span class="text-xs text-gray-500">{{ t('admin.observability.averageWait') }}</span>
          <strong class="mt-1 block text-lg text-gray-900 dark:text-white">
            {{ capacity.average_wait_ms == null ? t('admin.observability.noSampling') : `${Math.round(capacity.average_wait_ms)} ms` }}
          </strong>
          <small class="text-gray-400">{{ t('admin.observability.fiveMinuteSamples') }}</small>
        </div>
        <div class="rounded-lg border border-gray-200 bg-gray-50 p-2 dark:border-dark-600 dark:bg-dark-700">
          <span class="text-xs text-gray-500">{{ t('admin.observability.todayRateLimits') }}</span>
          <strong class="mt-1 block text-lg text-gray-900 dark:text-white">{{ capacity.rate_limit_available ? capacity.today_rate_limited_count : t('admin.observability.unavailable') }}</strong>
          <small class="text-gray-400">{{ t('admin.observability.calendarDayTotal') }}</small>
        </div>
      </div>
    </template>
  </article>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useObservabilityI18n } from './useObservabilityI18n'
import type { CapacitySummary } from '@/api/admin/observability'
import { statusFor } from './status'

const props = defineProps<{ capacity?: CapacitySummary | null }>()
const { t } = useObservabilityI18n()

const levels = computed(() => props.capacity ? [
  { key: 'global', label: t('admin.observability.globalSlots'), data: props.capacity.global },
  { key: 'user', label: t('admin.observability.groupUserLevel'), data: props.capacity.user },
  { key: 'account', label: t('admin.observability.accountLevel'), data: props.capacity.account },
  { key: 'route', label: t('admin.observability.routeLevel'), data: props.capacity.route },
] : [])

function barClass(utilization: number) {
  return statusFor(utilization).barClass
}
</script>

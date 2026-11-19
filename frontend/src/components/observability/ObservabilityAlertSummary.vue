<template>
  <section class="rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
    <div class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <span class="flex h-6 w-6 items-center justify-center rounded-full bg-amber-100 text-amber-700 dark:bg-amber-900/40 dark:text-amber-300" aria-hidden="true">!</span>
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.observability.pendingAlerts') }}</h2>
      </div>
      <span class="text-xs text-gray-500">{{ events.length }}</span>
    </div>
    <div v-if="error" class="mt-3 text-sm text-red-600 dark:text-red-400" role="alert">{{ error }}</div>
    <div v-else-if="loading && !events.length" class="mt-3 h-16 animate-pulse rounded-xl bg-gray-100 dark:bg-dark-700" aria-busy="true">
      <span class="sr-only">{{ t('admin.observability.loading') }}</span>
    </div>
    <ul v-else-if="events.length" class="mt-3 grid gap-2 lg:grid-cols-2">
      <li v-for="event in events" :key="event.id">
        <button
          type="button"
          class="w-full rounded-xl border border-gray-200 px-3 py-2 text-left hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-primary-500 dark:border-dark-600 dark:hover:bg-dark-700"
          @click="openEvent(event)"
        >
          <div class="flex items-start justify-between gap-2">
            <span class="text-sm font-medium text-gray-900 dark:text-white">{{ event.title || t('admin.observability.alertFallback', { id: event.id }) }}</span>
            <SeverityBadge :severity="event.severity" />
          </div>
          <div class="mt-1 text-xs text-gray-500">{{ dimensionText(event) }}</div>
          <div class="mt-1 flex flex-wrap gap-2 text-xs text-gray-500">
            <span>{{ formatTime(event.fired_at) }}</span>
            <span v-if="event.metric_value != null">{{ formatMetric(event.metric_value) }}</span>
            <span v-if="dimension(event, 'sample_count') != null">{{ t('admin.observability.samples', { count: dimension(event, 'sample_count') }) }}</span>
            <span v-if="dimension(event, 'data_at')">{{ t('admin.observability.dataAt', { time: formatTime(String(dimension(event, 'data_at'))) }) }}</span>
          </div>
        </button>
      </li>
    </ul>
    <p v-else class="mt-3 text-sm text-gray-400">{{ t('admin.observability.noPendingAlerts') }}</p>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useObservabilityI18n } from './useObservabilityI18n'
import { opsAPI, type AlertEvent } from '@/api/admin/ops'
import SeverityBadge from './SeverityBadge.vue'

const emit = defineEmits<{
  open: [filters: { groupId?: number; routeAlias?: string; accountId?: number }]
}>()
const { t } = useObservabilityI18n()
const events = ref<AlertEvent[]>([])
const loading = ref(false)
const error = ref('')

onMounted(load)

async function load() {
  loading.value = true
  try {
    events.value = (await opsAPI.listAlertEvents({ status: 'firing', limit: 30 }))
      .filter((event) => Boolean(dimension(event, 'group_id') || dimension(event, 'route_alias') || dimension(event, 'account_id')))
      .slice(0, 12)
  } catch (err) {
    error.value = err instanceof Error ? err.message : t('admin.observability.unableToLoadAlerts')
  } finally {
    loading.value = false
  }
}

function dimension(event: AlertEvent, key: string): unknown {
  return event.dimensions?.[key]
}

function dimensionText(event: AlertEvent) {
  const parts = [
    dimension(event, 'group_id') ? t('admin.observability.groupRef', { id: dimension(event, 'group_id') }) : '',
    dimension(event, 'route_alias') ? t('admin.observability.routeRef', { name: dimension(event, 'route_alias') }) : '',
    dimension(event, 'account_id') ? t('admin.observability.accountRef', { id: dimension(event, 'account_id') }) : '',
  ].filter(Boolean)
  return parts.join(' · ') || t('admin.observability.noObjectDimensions')
}

function openEvent(event: AlertEvent) {
  const groupId = Number(dimension(event, 'group_id'))
  const accountId = Number(dimension(event, 'account_id'))
  emit('open', {
    groupId: Number.isFinite(groupId) && groupId > 0 ? groupId : undefined,
    routeAlias: dimension(event, 'route_alias') ? String(dimension(event, 'route_alias')) : undefined,
    accountId: Number.isFinite(accountId) && accountId > 0 ? accountId : undefined,
  })
}

function formatTime(value: string) {
  const date = new Date(value)
  return Number.isNaN(date.getTime()) ? value : date.toLocaleString()
}

function formatMetric(value: number) {
  return Number.isInteger(value) ? value.toLocaleString() : value.toFixed(2)
}
</script>

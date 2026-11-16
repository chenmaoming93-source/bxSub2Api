<template>
  <article class="min-w-0 rounded-2xl border border-gray-200 bg-white p-4 dark:border-dark-600 dark:bg-dark-800">
    <div class="mb-3 flex flex-wrap items-center justify-between gap-2">
      <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('admin.observability.pressureRanking') }}</h2>
      <div class="inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-700" role="tablist" :aria-label="t('admin.observability.pressureRanking')">
        <button
          v-for="tab in tabs"
          :id="`pressure-tab-${tab.value}`"
          :key="tab.value"
          type="button"
          class="rounded-md px-3 py-1 text-xs"
          :class="kind === tab.value ? 'bg-white shadow dark:bg-dark-600' : 'text-gray-500'"
          role="tab"
          :aria-selected="kind === tab.value"
          :aria-controls="`pressure-panel-${tab.value}`"
          :tabindex="kind === tab.value ? 0 : -1"
          @click="activateTab(tab.value)"
          @keydown="handleTabKeydown($event, tab.value)"
        >
          {{ tab.label }}
        </button>
      </div>
    </div>
    <ol
      :id="`pressure-panel-${kind}`"
      class="space-y-2"
      role="tabpanel"
      :aria-labelledby="`pressure-tab-${kind}`"
      tabindex="0"
    >
      <li
        v-for="row in rows"
        :key="`${row.group_id}-${row.route_alias}-${row.account_id}`"
        class="cursor-pointer rounded-xl px-3 py-2 hover:bg-gray-50 focus-within:bg-gray-50 dark:hover:bg-dark-700"
        tabindex="0"
        role="button"
        @click="select(row.account_id)"
        @keydown.enter="select(row.account_id)"
        @keydown.space.prevent="select(row.account_id)"
      >
        <div class="flex items-center justify-between gap-3">
          <div class="min-w-0">
            <div class="truncate text-sm font-medium text-gray-800 dark:text-gray-100">
              {{ row.account_name || `#${row.account_id}` }}
            </div>
            <div class="truncate text-xs text-gray-500">
              {{ row.group_name || (row.group_id ? `#${row.group_id}` : '') }} {{ row.route_alias ? `/ ${row.route_alias}` : '' }}
            </div>
          </div>
          <div class="text-right text-xs">
            <div class="font-mono text-gray-900 dark:text-white">{{ row.current }} / {{ row.limit || '∞' }}</div>
            <div class="text-gray-500">{{ Math.round(row.utilization * 100) }}% · {{ t(statusFor(row.utilization).labelKey) }} · {{ t('admin.observability.wait') }} {{ row.waiting }}</div>
          </div>
        </div>
        <div class="mt-2 h-1.5 overflow-hidden rounded-full bg-gray-100 dark:bg-dark-600">
          <div
            class="h-full rounded-full"
            :class="statusFor(row.utilization).barClass"
            :style="{ width: `${Math.min(100, Math.max(0, row.utilization * 100))}%` }"
          ></div>
        </div>
      </li>
    </ol>
    <p v-if="rows.length === 0" class="py-8 text-center text-sm text-gray-400">{{ t('admin.observability.noPressureSamples') }}</p>
  </article>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useObservabilityI18n } from './useObservabilityI18n'
import { statusFor } from './status'
import type { PressureRow } from '@/api/admin/observability'

const props = defineProps<{
  accounts: PressureRow[]
  candidates: PressureRow[]
}>()
const emit = defineEmits<{ select: [accountId: number] }>()
const { t } = useObservabilityI18n()
const kind = ref<'accounts' | 'candidates'>('accounts')
const tabs = computed(() => [
  { value: 'accounts' as const, label: t('admin.observability.accounts') },
  { value: 'candidates' as const, label: t('admin.observability.candidates') },
])
const rows = computed(() => kind.value === 'accounts' ? props.accounts : props.candidates)
function activateTab(value: 'accounts' | 'candidates') {
  kind.value = value
}

function handleTabKeydown(event: KeyboardEvent, current: 'accounts' | 'candidates') {
  if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return
  event.preventDefault()
  const index = tabs.value.findIndex((tab) => tab.value === current)
  let next = index
  if (event.key === 'ArrowLeft') next = (index - 1 + tabs.value.length) % tabs.value.length
  if (event.key === 'ArrowRight') next = (index + 1) % tabs.value.length
  if (event.key === 'Home') next = 0
  if (event.key === 'End') next = tabs.value.length - 1
  activateTab(tabs.value[next].value)
  document.getElementById(`pressure-tab-${tabs.value[next].value}`)?.focus()
}

function select(accountId: number) {
  emit('select', accountId)
}
</script>

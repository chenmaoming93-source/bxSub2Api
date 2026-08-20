<template>
  <BaseDialog :show="show" :title="t('admin.groups.concurrencyView.title')" width="wide" @close="emit('close')">
    <div class="space-y-4">
      <div v-if="loading" class="flex items-center justify-center gap-2 py-8 text-sm text-gray-500"><Icon name="refresh" size="sm" class="animate-spin" /><span>{{ t('common.loading') }}</span></div>
      <div v-else-if="!rows.length" class="py-8 text-center text-sm text-gray-500">{{ t('admin.groups.concurrencyView.empty') }}</div>
      <div v-else class="max-h-[60vh] space-y-3 overflow-y-auto">
        <section v-for="route in rows" :key="route.alias" class="rounded-lg border border-gray-200 dark:border-dark-600">
          <div class="border-b border-gray-200 bg-gray-50 px-4 py-2.5 font-medium dark:border-dark-600 dark:bg-dark-700/50">{{ route.alias }}</div>
          <div class="divide-y divide-gray-100 dark:divide-dark-700">
            <div v-for="candidate in route.candidates" :key="candidate.key" class="px-4 py-3">
              <div class="mb-2 text-xs font-medium text-gray-500">{{ t('admin.groups.concurrencyView.candidate') }} {{ candidate.priority }}</div>
              <div class="space-y-1.5">
                <div v-for="account in candidate.accounts" :key="account.id" class="flex items-center justify-between gap-3 text-sm">
                  <span class="min-w-0 truncate text-gray-700 dark:text-gray-200">{{ account.name }} <span class="text-xs text-gray-400">#{{ account.id }}</span></span>
                  <span class="shrink-0 font-mono">{{ account.current === null ? '—' : account.current }} / {{ candidate.max === null ? '∞' : candidate.max }}</span>
                </div>
              </div>
            </div>
          </div>
        </section>
      </div>
      <p v-if="error" class="text-xs text-red-500">{{ t('admin.groups.concurrencyView.loadFailed') }}</p>
    </div>
    <template #footer><div class="flex justify-end"><button type="button" class="btn btn-secondary" :disabled="loading" @click="load"><Icon name="refresh" size="sm" class="mr-1.5" :class="loading ? 'animate-spin' : ''" />{{ t('common.refresh') }}</button></div></template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { adminAPI } from '@/api/admin'
import type { AdminGroup } from '@/types'
import { normalizeModelRouting } from '@/views/admin/groupsModelRouting'
import { useAutoRefresh } from '@/composables/useAutoRefresh'

const props = defineProps<{ show: boolean; group: AdminGroup | null }>()
const emit = defineEmits<{ (event: 'close'): void }>()
const { t } = useI18n()
interface AccountRow { id: number; name: string; current: number | null }
interface CandidateRow { key: string; priority: number; max: number | null; accounts: AccountRow[] }
interface RouteRow { alias: string; candidates: CandidateRow[] }
const rows = ref<RouteRow[]>([])
const loading = ref(false)
const error = ref(false)

const load = async () => {
  if (!props.group) return
  loading.value = true
  error.value = false
  try {
    const rules = normalizeModelRouting(props.group.model_routing)
    const references = await adminAPI.groups.listModelRouteConcurrency(props.group.id)
    const accountIDs = [...new Set(rules.flatMap(rule => rule.candidates.flatMap(candidate => candidate.account_ids)))]
    const accounts = await Promise.all(accountIDs.map(async id => {
      try {
        const account = await adminAPI.accounts.getById(id)
        return { id, name: account.name, current: account.current_concurrency ?? 0 }
      } catch { return { id, name: `#${id}`, current: null } }
    }))
    const byID = new Map(accounts.map(account => [account.id, account]))
    rows.value = rules.map((rule, ruleIndex) => ({ alias: rule.alias, candidates: rule.candidates.map((candidate, candidateIndex) => ({
      key: `${rule.alias}-${ruleIndex}-${candidateIndex}`,
      priority: candidate.priority,
      max: (() => {
        const match = references.find(reference => reference.route_alias === rule.alias && candidate.account_ids.includes(reference.account_id))
        // New backends return the materialized limit for the current minute;
        // fall back to the legacy default while mixed-version deployments roll out.
        const effectiveMax = match?.effective_max_concurrency !== undefined
          ? match.effective_max_concurrency
          : match?.max_concurrency
        return effectiveMax == null ? null : Number(effectiveMax)
      })(),
      accounts: candidate.account_ids.map(id => {
        const account = byID.get(id) ?? { id, name: `#${id}`, current: null }
        const reference = references.find(item => item.route_alias === rule.alias && item.account_id === id)
        return { ...account, current: reference?.current_concurrency ?? account.current }
      }),
    })) }))
  } catch { error.value = true } finally { loading.value = false }
}

const autoRefresh = useAutoRefresh({
  storageKey: 'group-concurrency-view-auto-refresh',
  intervals: [5, 10, 15, 30] as const,
  defaultInterval: 30,
  onRefresh: load,
  shouldPause: () => !props.show || document.hidden,
})

watch(() => [props.show, props.group?.id], ([show]) => {
  if (show) {
    void load()
    autoRefresh.setEnabled(true)
  } else {
    autoRefresh.setEnabled(false)
  }
}, { immediate: true })
</script>

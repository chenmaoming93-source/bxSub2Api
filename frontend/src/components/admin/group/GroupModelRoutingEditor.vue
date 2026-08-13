<template>
  <div class="border-t pt-4">
    <div class="mb-1.5 flex items-center gap-1">
      <label class="text-sm font-medium text-gray-700 dark:text-gray-300">{{ t('admin.groups.modelRouting.title') }}</label>
      <div class="group relative inline-flex">
        <Icon name="questionCircle" size="sm" :stroke-width="2" class="cursor-help text-gray-400 hover:text-primary-500" />
        <div class="pointer-events-none absolute bottom-full left-0 z-50 mb-2 w-80 opacity-0 transition-all group-hover:opacity-100">
          <div class="rounded-lg bg-gray-900 p-3 text-white shadow-lg"><p class="text-xs text-gray-300">{{ t('admin.groups.modelRouting.tooltip') }}</p></div>
        </div>
      </div>
    </div>
    <div class="mb-3 flex items-center gap-3">
      <button type="button" :class="['relative inline-flex h-6 w-11 items-center rounded-full transition-colors', enabled ? 'bg-primary-500' : 'bg-gray-300 dark:bg-dark-600']" @click="enabled = !enabled">
        <span :class="['inline-block h-4 w-4 rounded-full bg-white shadow transition-transform', enabled ? 'translate-x-6' : 'translate-x-1']" />
      </button>
      <span class="text-sm text-gray-500 dark:text-gray-400">{{ enabled ? t('admin.groups.modelRouting.enabled') : t('admin.groups.modelRouting.disabled') }}</span>
    </div>
    <div class="mb-3 flex items-center justify-between gap-3">
      <p class="text-xs text-gray-500 dark:text-gray-400">{{ t(enabled ? 'admin.groups.modelRouting.noRulesHint' : 'admin.groups.modelRouting.disabledHint') }}</p>
    </div>
    <div v-if="enabled" class="space-y-3">
      <div v-for="rule in rules" :key="ruleKey(rule)" class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
        <div class="flex items-start gap-3">
          <div class="min-w-0 flex-1 space-y-3">
            <div>
              <label class="input-label text-xs">{{ t('admin.groups.modelRouting.routeAlias', 'Route alias') }}</label>
              <input v-model="rule.alias" type="text" class="input text-sm" :placeholder="t('admin.groups.modelRouting.routeAliasPlaceholder', 'e.g. coding-fast')" />
            </div>
            <div v-for="candidate in rule.candidates" :key="candidateKey(candidate)" class="relative w-full min-w-0 space-y-2 rounded-md bg-gray-50 p-3 dark:bg-dark-700/50">
              <div v-if="candidate.accounts.length" class="flex flex-wrap gap-1.5">
                <span v-for="account in candidate.accounts" :key="account.id" class="inline-flex items-center gap-1 rounded-full bg-primary-100 px-2.5 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
                  <span class="max-w-[160px] truncate">{{ account.name }}</span>
                  <span v-if="account.upstreamModel" class="rounded bg-primary-200/60 px-1.5 py-0.5 text-[10px] font-medium text-primary-800 dark:bg-primary-800/40 dark:text-primary-200">{{ account.upstreamModel }}</span>
                  <button type="button" data-test="remove-account" :data-account-id="account.id" @click="removeAccount(candidate, account.id)"><Icon name="x" size="xs" /></button>
                </span>
              </div>
              <div class="relative account-search-container">
                <input v-model="keywords[candidateKey(candidate)]" type="text" class="input text-sm" :placeholder="t('admin.groups.modelRouting.searchAccountPlaceholder')" @input="search(candidate)" @focus="focus(candidate)" @pointerdown="toggleDropdown(candidate, $event)" @keydown.esc.prevent="closeAllDropdowns" />
                <div v-if="dropdowns[candidateKey(candidate)]" class="absolute z-50 mt-1 max-h-48 w-full overflow-auto rounded-lg border bg-white shadow-lg dark:border-dark-600 dark:bg-dark-800">
                  <div v-if="loading[candidateKey(candidate)]" class="flex items-center justify-center gap-2 px-3 py-4 text-sm text-gray-500 dark:text-gray-400">
                    <Icon name="refresh" size="sm" class="animate-spin" />
                    <span>加载中...</span>
                  </div>
                  <button v-else v-for="account in results[candidateKey(candidate)] || []" :key="account.id" type="button" class="flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700" :disabled="candidate.accounts.some(item => item.id === account.id)" @click="select(candidate, account)">
                    <span class="truncate">{{ account.name }} <span class="text-xs text-gray-400">#{{ account.id }}</span></span>
                    <span v-if="account.upstreamModel" class="shrink-0 rounded bg-gray-100 px-1.5 py-0.5 text-[10px] text-gray-500 dark:bg-dark-600 dark:text-gray-400">{{ account.upstreamModel }}</span>
                  </button>
                  <div v-if="!loading[candidateKey(candidate)] && !results[candidateKey(candidate)]?.length" class="px-3 py-4 text-center text-sm text-gray-500 dark:text-gray-400">暂无匹配账号</div>
                </div>
              </div>
			  <p v-if="selectionErrors[candidateKey(candidate)]" class="text-xs text-red-500">{{ selectionErrors[candidateKey(candidate)] }}</p>
              <div class="grid min-w-0 gap-2 md:grid-cols-3">
                <div><label class="input-label text-xs">{{ t('admin.groups.modelRouting.priority', 'Priority') }}</label><input v-model.number="candidate.priority" type="number" min="0" step="1" class="input text-sm" /></div>
                <div>
                  <label class="input-label text-xs">最大并发</label>
                  <input v-model.number="candidate.maxConcurrency" @change="markConcurrencyDirty" type="number" min="1" step="1" class="input text-sm" placeholder="不限" />
                  <p v-if="candidate.maxConcurrency == null" class="mt-1 text-[10px] text-gray-500">无限制，不参与账号额度分配</p>
                  <p v-else-if="accountConcurrency(candidate) !== null" class="mt-1 text-[10px] text-gray-500">
                    其他分组已占用 {{ candidate.allocatedConcurrency ?? 0 }} / {{ accountConcurrency(candidate) }}，本分组配置 {{ groupAccountConcurrency(candidate) }}
                    （{{ candidateConcurrencyPercentage(candidate) }}%）
                  </p>
                </div>
              </div>
              <p v-if="invalid(candidate)" class="text-xs text-red-500">{{ t('admin.groups.modelRouting.candidateValidation', 'Account and non-negative integer priority are required') }}</p>
              <button type="button" class="absolute right-2 top-2 text-gray-400 hover:text-red-500" @click="removeCandidate(rule, candidate)"><Icon name="x" size="xs" /></button>
            </div>
            <button type="button" class="flex items-center gap-1 text-xs text-primary-600" @click="addRoutingCandidate(rule)"><Icon name="plus" size="xs" />{{ t('admin.groups.modelRouting.addCandidate', 'Add candidate') }}</button>
          </div>
          <button type="button" class="mt-5 text-gray-400 hover:text-red-500" @click="removeRule(rule)"><Icon name="trash" size="sm" /></button>
        </div>
      </div>
      <button type="button" class="mt-3 flex items-center gap-1.5 text-sm text-primary-600" @click="addRule"><Icon name="plus" size="sm" />{{ t('admin.groups.modelRouting.addRule') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import Icon from '@/components/icons/Icon.vue'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import { useKeyedDebouncedSearch } from '@/composables/useKeyedDebouncedSearch'
import { addRoutingCandidate, createEmptyRoutingCandidate, extractUpstreamModel, type RoutingEditorAccount, type RoutingEditorCandidate, type RoutingEditorRule } from './groupModelRoutingEditor'

const props = defineProps<{ platform: string; groupId?: number }>()
const enabled = defineModel<boolean>('enabled', { required: true })
const rules = defineModel<RoutingEditorRule[]>('rules', { required: true })
const { t } = useI18n()
const resolveRuleKey = createStableObjectKeyResolver<RoutingEditorRule>('routing-rule')
const resolveCandidateKey = createStableObjectKeyResolver<RoutingEditorCandidate>('routing-candidate')
const ruleKey = (rule: RoutingEditorRule) => resolveRuleKey(rule)
const candidateKey = (candidate: RoutingEditorCandidate) => resolveCandidateKey(candidate)
const keywords = ref<Record<string, string>>({})
const results = ref<Record<string, RoutingEditorAccount[]>>({})
const loading = ref<Record<string, boolean>>({})
const dropdowns = ref<Record<string, boolean>>({})
const selectionErrors = ref<Record<string, string>>({})
const concurrencyDirty = ref(false)
const runner = useKeyedDebouncedSearch<RoutingEditorAccount[]>({ delay: 300, search: async (keyword, { signal }) => {
  const response = await adminAPI.accounts.list(1, 20, { search: keyword, platform: props.platform as never }, { signal })
  return response.items.map(account => ({ id: account.id, name: account.name, concurrency: account.concurrency, upstreamModel: extractUpstreamModel(account.credentials) }))
}, onSuccess: (key, value) => { results.value[key] = value; loading.value[key] = false }, onError: key => { results.value[key] = []; loading.value[key] = false } })
const search = (candidate: RoutingEditorCandidate) => {
  const key = candidateKey(candidate)
  loading.value[key] = true
  runner.trigger(key, keywords.value[key] || '')
}
const focus = (candidate: RoutingEditorCandidate) => { dropdowns.value[candidateKey(candidate)] = true; if (!results.value[candidateKey(candidate)]?.length) search(candidate) }
const closeAllDropdowns = () => { dropdowns.value = {} }
const toggleDropdown = (candidate: RoutingEditorCandidate, event: PointerEvent) => {
  const key = candidateKey(candidate)
  if (!dropdowns.value[key]) return
  // Keep focus from immediately reopening the dropdown after closing it.
  event.preventDefault()
  dropdowns.value[key] = false
}
const loadAccountAllocation = async (candidate: RoutingEditorCandidate, account: RoutingEditorAccount) => {
  try {
    const refs = await adminAPI.accounts.getModelRouteReferences(account.id)
    const reference = refs[0]
    candidate.accountConcurrency = reference?.account_concurrency ?? account.concurrency
    candidate.allocatedConcurrency = props.groupId
      ? refs
          .filter(item => item.group_id !== props.groupId && item.max_concurrency != null)
          .reduce((total, item) => total + Number(item.max_concurrency), 0)
      : reference?.allocated_concurrency ?? 0
  } catch {
    // Keep the input usable when the reference lookup is temporarily unavailable.
  }
}
const select = (candidate: RoutingEditorCandidate, account: RoutingEditorAccount) => {
	if (candidate.accounts.length > 0 && !candidate.accounts.some(item => item.id === account.id)) {
	  selectionErrors.value[candidateKey(candidate)] = '每个候选只能选择一个模型账号'
	  return
	}
	selectionErrors.value[candidateKey(candidate)] = ''
  if (!candidate.accounts.some(item => item.id === account.id)) {
    candidate.accounts.push(account)
    void loadAccountAllocation(candidate, account)
  }
  keywords.value[candidateKey(candidate)] = ''
  dropdowns.value[candidateKey(candidate)] = false
}
const removeAccount = (candidate: RoutingEditorCandidate, id: number) => {
  candidate.accounts = candidate.accounts.filter(item => item.id !== id)
}
const removeCandidate = (rule: RoutingEditorRule, candidate: RoutingEditorCandidate) => {
  const key = candidateKey(candidate)
  rule.candidates.splice(rule.candidates.indexOf(candidate), 1)
  runner.clearKey(key)
}
const addRule = () => rules.value.push({ alias: '', candidates: [createEmptyRoutingCandidate()] })
const removeRule = (rule: RoutingEditorRule) => { rules.value.splice(rules.value.indexOf(rule), 1) }
const invalid = (candidate: RoutingEditorCandidate) => {
  return !candidate.accounts.length || !Number.isInteger(Number(candidate.priority)) || Number(candidate.priority) < 0
}
const accountConcurrency = (candidate: RoutingEditorCandidate): number | null => candidate.accountConcurrency ?? candidate.accounts[0]?.concurrency ?? null
const groupAccountConcurrency = (candidate: RoutingEditorCandidate): number => {
  const accountIDs = new Set(candidate.accounts.map(account => account.id))
  let total = 0
  for (const rule of rules.value) {
    for (const item of rule.candidates) {
      if (!item.accounts.some(account => accountIDs.has(account.id))) continue
      const max = item.maxConcurrency === '' || item.maxConcurrency == null ? null : Number(item.maxConcurrency)
      if (max !== null && Number.isFinite(max)) total += max
    }
  }
  return total
}
const candidateConcurrencyPercentage = (candidate: RoutingEditorCandidate): number | string => {
  const total = accountConcurrency(candidate)
  return total !== null && total > 0 ? Math.round(groupAccountConcurrency(candidate) * 100 / total) : '不适用'
}
const markConcurrencyDirty = () => { concurrencyDirty.value = true }
const getConcurrencyUpdates = (): Array<{ route_alias: string; account_id: number; max_concurrency: number | null }> => {
  const updates: Array<{ route_alias: string; account_id: number; max_concurrency: number | null }> = []
  for (const rule of rules.value) for (const candidate of rule.candidates) {
    const max = candidate.maxConcurrency === '' || candidate.maxConcurrency == null ? null : Number(candidate.maxConcurrency)
    if (max !== null && (!Number.isInteger(max) || max <= 0)) throw new Error('最大并发数必须是正整数或留空')
    for (const account of candidate.accounts) updates.push({ route_alias: rule.alias.trim(), account_id: account.id, max_concurrency: max })
  }
  return updates
}
watch(() => props.groupId, async (groupId) => {
  if (!groupId) return
  try {
    const refs = await adminAPI.groups.listModelRouteReferences(groupId)
    for (const rule of rules.value) {
      for (const candidate of rule.candidates) {
        const account = candidate.accounts[0]
        if (!account) continue
        const match = refs.find(item => item.route_alias === rule.alias && item.account_id === account.id)
        if (match) {
          candidate.maxConcurrency = match.max_concurrency
          candidate.accountConcurrency = match.account_concurrency
          candidate.allocatedConcurrency = match.allocated_concurrency
        }
      }
    }
  } catch { /* the editor still works when the projection is not initialized */ }
  concurrencyDirty.value = false
}, { immediate: true })
const isValid = () => !enabled.value || (rules.value.length > 0 && rules.value.every(rule => rule.alias.trim() && rule.candidates.length > 0 && rule.candidates.every(candidate => !invalid(candidate) && candidate.accounts.length === 1)))
defineExpose({ isValid, getConcurrencyUpdates })
const closeDropdowns = (event: PointerEvent) => {
  if (!(event.target as HTMLElement).closest('.account-search-container')) closeAllDropdowns()
}
onMounted(() => {
  document.addEventListener('pointerdown', closeDropdowns, true)
})
onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', closeDropdowns, true)
  runner.clearAll()
})
</script>

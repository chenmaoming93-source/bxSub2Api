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
    <p class="mb-3 text-xs text-gray-500 dark:text-gray-400">{{ t(enabled ? 'admin.groups.modelRouting.noRulesHint' : 'admin.groups.modelRouting.disabledHint') }}</p>
    <div v-if="enabled" class="space-y-3">
      <div v-for="rule in rules" :key="ruleKey(rule)" class="rounded-lg border border-gray-200 p-3 dark:border-dark-600">
        <div class="flex items-start gap-3">
          <div class="min-w-0 flex-1 space-y-3">
            <div>
              <label class="input-label text-xs">{{ t('admin.groups.modelRouting.routeAlias', 'Route alias') }}</label>
              <input v-model="rule.alias" type="text" class="input text-sm" :placeholder="t('admin.groups.modelRouting.routeAliasPlaceholder', 'e.g. coding-fast')" />
            </div>
            <div v-for="candidate in rule.candidates" :key="candidateKey(candidate)" class="relative w-full min-w-0 space-y-2 rounded-md bg-gray-50 p-3 dark:bg-dark-700/50">
              <div class="grid min-w-0 gap-2 md:grid-cols-3">
                <div><label class="input-label text-xs">{{ t('admin.groups.modelRouting.candidateModel', 'Upstream model') }}</label><input v-model="candidate.model" type="text" class="input text-sm" /></div>
                <div><label class="input-label text-xs">{{ t('admin.groups.modelRouting.priority', 'Priority') }}</label><input v-model.number="candidate.priority" type="number" min="0" step="1" class="input text-sm" /></div>
                <div><label class="input-label text-xs">{{ t('admin.groups.modelRouting.dailyTokenLimit', 'Daily token limit') }}</label><input v-model.number="candidate.daily_token_limit" type="number" min="0" step="1" class="input text-sm" :placeholder="t('admin.groups.modelRouting.unlimited')" /></div>
              </div>
              <div v-if="candidate.accounts.length" class="flex flex-wrap gap-1.5">
                <span v-for="account in candidate.accounts" :key="account.id" class="inline-flex items-center gap-1 rounded-full bg-primary-100 px-2.5 py-1 text-xs text-primary-700 dark:bg-primary-900/30 dark:text-primary-300">
                  {{ account.name }}<button type="button" @click="removeAccount(candidate, account.id)"><Icon name="x" size="xs" /></button>
                </span>
              </div>
              <div class="relative account-search-container">
                <input v-model="keywords[candidateKey(candidate)]" type="text" class="input text-sm" :placeholder="t('admin.groups.modelRouting.searchAccountPlaceholder')" @input="search(candidate)" @focus="focus(candidate)" />
                <div v-if="dropdowns[candidateKey(candidate)] && results[candidateKey(candidate)]?.length" class="absolute z-50 mt-1 max-h-48 w-full overflow-auto rounded-lg border bg-white shadow-lg dark:border-dark-600 dark:bg-dark-800">
                  <button v-for="account in results[candidateKey(candidate)]" :key="account.id" type="button" class="w-full px-3 py-2 text-left text-sm hover:bg-gray-100 dark:hover:bg-dark-700" :disabled="candidate.accounts.some(item => item.id === account.id)" @click="select(candidate, account)">{{ account.name }} <span class="text-xs text-gray-400">#{{ account.id }}</span></button>
                </div>
              </div>
              <p v-if="invalid(candidate)" class="text-xs text-red-500">{{ t('admin.groups.modelRouting.candidateValidation', 'Model, account, and non-negative integer values are required') }}</p>
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
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import Icon from '@/components/icons/Icon.vue'
import { createStableObjectKeyResolver } from '@/utils/stableObjectKey'
import { useKeyedDebouncedSearch } from '@/composables/useKeyedDebouncedSearch'
import { addRoutingCandidate, createEmptyRoutingCandidate, type RoutingEditorAccount, type RoutingEditorCandidate, type RoutingEditorRule } from './groupModelRoutingEditor'

const props = defineProps<{ platform: string }>()
const enabled = defineModel<boolean>('enabled', { required: true })
const rules = defineModel<RoutingEditorRule[]>('rules', { required: true })
const { t } = useI18n()
const resolveRuleKey = createStableObjectKeyResolver<RoutingEditorRule>('routing-rule')
const resolveCandidateKey = createStableObjectKeyResolver<RoutingEditorCandidate>('routing-candidate')
const ruleKey = (rule: RoutingEditorRule) => resolveRuleKey(rule)
const candidateKey = (candidate: RoutingEditorCandidate) => resolveCandidateKey(candidate)
const keywords = ref<Record<string, string>>({})
const results = ref<Record<string, RoutingEditorAccount[]>>({})
const dropdowns = ref<Record<string, boolean>>({})
const runner = useKeyedDebouncedSearch<RoutingEditorAccount[]>({ delay: 300, search: async (keyword, { signal }) => {
  const response = await adminAPI.accounts.list(1, 20, { search: keyword, platform: props.platform as never }, { signal })
  return response.items.map(account => ({ id: account.id, name: account.name }))
}, onSuccess: (key, value) => { results.value[key] = value }, onError: key => { results.value[key] = [] } })
const search = (candidate: RoutingEditorCandidate) => runner.trigger(candidateKey(candidate), keywords.value[candidateKey(candidate)] || '')
const focus = (candidate: RoutingEditorCandidate) => { dropdowns.value[candidateKey(candidate)] = true; if (!results.value[candidateKey(candidate)]?.length) search(candidate) }
const select = (candidate: RoutingEditorCandidate, account: RoutingEditorAccount) => { if (!candidate.accounts.some(item => item.id === account.id)) candidate.accounts.push(account); keywords.value[candidateKey(candidate)] = ''; dropdowns.value[candidateKey(candidate)] = false }
const removeAccount = (candidate: RoutingEditorCandidate, id: number) => { candidate.accounts = candidate.accounts.filter(item => item.id !== id) }
const removeCandidate = (rule: RoutingEditorRule, candidate: RoutingEditorCandidate) => { rule.candidates.splice(rule.candidates.indexOf(candidate), 1); runner.clearKey(candidateKey(candidate)) }
const addRule = () => rules.value.push({ alias: '', candidates: [createEmptyRoutingCandidate()] })
const removeRule = (rule: RoutingEditorRule) => { rules.value.splice(rules.value.indexOf(rule), 1) }
const invalid = (candidate: RoutingEditorCandidate) => !candidate.model.trim() || !candidate.accounts.length || !Number.isInteger(Number(candidate.priority)) || Number(candidate.priority) < 0 || (candidate.daily_token_limit !== null && candidate.daily_token_limit !== '' && (!Number.isInteger(Number(candidate.daily_token_limit)) || Number(candidate.daily_token_limit) < 0))
const closeDropdowns = (event: MouseEvent) => { if (!(event.target as HTMLElement).closest('.account-search-container')) dropdowns.value = {} }
onMounted(() => document.addEventListener('click', closeDropdowns))
onBeforeUnmount(() => { document.removeEventListener('click', closeDropdowns); runner.clearAll() })
</script>

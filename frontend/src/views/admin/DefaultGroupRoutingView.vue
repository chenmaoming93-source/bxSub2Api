<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6 p-6">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('admin.defaultGroupRouting.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500">{{ t('admin.defaultGroupRouting.description') }}</p>
      </div>
      <div v-if="loading" class="card p-6 text-sm text-gray-500">{{ t('common.loading') }}</div>
      <div v-else-if="!status?.configured" class="card p-6">
        <p class="text-gray-700 dark:text-gray-300">{{ t('admin.defaultGroupRouting.unconfigured') }}</p>
        <RouterLink class="btn btn-primary mt-4 inline-flex" to="/admin/settings">{{ t('admin.defaultGroupRouting.goToSettings') }}</RouterLink>
      </div>
      <div v-else-if="!status.exists" class="card p-6">
        <p class="font-medium text-amber-600">{{ t('admin.defaultGroupRouting.missing', { name: status.name }) }}</p>
        <div class="mt-4 grid gap-3 sm:grid-cols-[1fr_220px_auto]">
          <input :value="status.name" class="input" disabled data-test="locked-default-group-name" />
          <select v-model="createPlatform" class="input">
            <option value="openai">OpenAI</option><option value="anthropic">Anthropic</option><option value="gemini">Gemini</option><option value="antigravity">Antigravity</option>
          </select>
          <button class="btn btn-primary" :disabled="creating" @click="createMissingGroup">{{ t('admin.defaultGroupRouting.create') }}</button>
        </div>
      </div>
      <div v-else-if="group" class="card p-6">
        <p class="mb-4 text-sm text-gray-500">{{ group.name }}</p>
        <GroupModelRoutingEditor ref="routingEditor" v-model:enabled="enabled" v-model:rules="rules" :platform="group.platform" />
        <div class="mt-6 flex justify-end"><button class="btn btn-primary" :disabled="saving" @click="save">{{ t('common.save') }}</button></div>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import GroupModelRoutingEditor from '@/components/admin/group/GroupModelRoutingEditor.vue'
import type { RoutingEditorRule } from '@/components/admin/group/groupModelRoutingEditor'
import { settingsAPI, type DefaultGroupStatus } from '@/api/admin/settings'
import { adminAPI } from '@/api/admin'
import type { AdminGroup, ModelRoutingCandidate, ModelRoutingRuleRow } from '@/types'
import { normalizeModelRouting, serializeModelRouting } from './groupsModelRouting'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const loading = ref(true)
const saving = ref(false)
const creating = ref(false)
const createPlatform = ref<'openai' | 'anthropic' | 'gemini' | 'antigravity'>('openai')
const status = ref<DefaultGroupStatus | null>(null)
const group = ref<AdminGroup | null>(null)
const enabled = ref(false)
const rules = ref<RoutingEditorRule[]>([])
const routingEditor = ref<{ isValid: () => boolean } | null>(null)

async function load() {
  loading.value = true
  try {
    status.value = await settingsAPI.getDefaultGroup()
    if (status.value.exists && status.value.group) {
      group.value = await adminAPI.groups.getById(status.value.group.id)
      enabled.value = group.value.model_routing_enabled === true
      rules.value = await Promise.all(normalizeModelRouting(group.value.model_routing).map(async row => ({
        alias: row.alias,
        candidates: await Promise.all(row.candidates.map(async candidate => ({
          model: candidate.model, priority: candidate.priority, daily_token_limit: candidate.daily_token_limit,
          accounts: await Promise.all(candidate.account_ids.map(async id => {
            try { const account = await adminAPI.accounts.getById(id); return { id: account.id, name: account.name } }
            catch { return { id, name: `#${id}` } }
          }))
        })))
      })))
    }
  } catch (error) { appStore.showError(extractApiErrorMessage(error, t('common.error'))) }
  finally { loading.value = false }
}

async function save() {
  if (!group.value) return
  if (routingEditor.value && !routingEditor.value.isValid()) {
    appStore.showError(t('admin.groups.modelRouting.candidateValidation'))
    return
  }
  saving.value = true
  try {
    const rows: ModelRoutingRuleRow[] = rules.value.map(rule => ({ alias: rule.alias, candidates: rule.candidates.map((candidate): ModelRoutingCandidate => ({
      model: candidate.model, account_ids: candidate.accounts.map(account => account.id), priority: Number(candidate.priority),
      daily_token_limit: candidate.daily_token_limit === null || candidate.daily_token_limit === '' ? null : Number(candidate.daily_token_limit)
    })) }))
    group.value = await adminAPI.groups.update(group.value.id, { model_routing_enabled: enabled.value, model_routing: rows.length ? serializeModelRouting(rows) : null })
    appStore.showSuccess(t('admin.defaultGroupRouting.saved'))
  } catch (error) { appStore.showError(extractApiErrorMessage(error, t('common.error'))) }
  finally { saving.value = false }
}

async function createMissingGroup() {
  if (!status.value?.configured || status.value.exists) return
  creating.value = true
  try {
    await adminAPI.groups.create({ name: status.value.name, platform: createPlatform.value, rate_multiplier: 1, subscription_type: 'standard', model_routing_enabled: false })
    await load()
  } catch (error) {
    await load()
    if (!status.value?.exists) appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally { creating.value = false }
}

onMounted(load)
</script>

<template>
  <BaseDialog :show="show" :title="t('admin.users.defaultQuota.title', 'Default model daily token quotas for new users')" width="wide" @close="$emit('close')">
    <div class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('admin.users.defaultQuota.description', 'New users will automatically inherit these model token quota settings') }}</p>
      <div v-if="loading" data-test="default-quota-loading" class="py-10 text-center text-gray-500">{{ t('common.loading') }}</div>
      <div v-else class="space-y-3">
        <div v-for="(row, index) in rows" :key="row.key" class="grid items-end gap-2 rounded-lg border border-gray-200 p-3 dark:border-dark-700 md:grid-cols-[1fr_180px_auto]">
          <label class="block text-xs text-gray-500">
            {{ t('admin.users.defaultQuota.model', 'Upstream model') }}
            <input v-model="row.model" data-test="default-quota-model" type="text" class="input mt-1 text-sm" />
          </label>
          <label class="block text-xs text-gray-500">
            {{ t('admin.users.defaultQuota.limit', 'Daily token limit') }}
            <input v-model.number="row.daily_limit_tokens" data-test="default-quota-limit" type="number" min="0" step="1" class="input mt-1 text-sm" :placeholder="t('admin.users.defaultQuota.unlimited', 'Unlimited')" />
          </label>
          <button type="button" class="p-2 text-gray-400 hover:text-red-500" :title="t('common.delete')" @click="removeRow(index)"><Icon name="trash" size="sm" /></button>
        </div>
        <button data-test="default-quota-add" type="button" class="flex items-center gap-1 text-sm text-primary-600" @click="addRow"><Icon name="plus" size="sm" />{{ t('admin.users.defaultQuota.add', 'Add model') }}</button>
        <p v-if="errorMessage" data-test="default-quota-error" class="text-sm text-red-500">{{ errorMessage }}</p>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="$emit('close')">{{ t('common.cancel') }}</button>
        <button data-test="default-quota-save" type="button" class="btn btn-primary" :disabled="loading || submitting" @click="onSave">{{ submitting ? t('common.saving') : t('common.save') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { DefaultModelTokenQuotaItem } from '@/api/admin/modelTokenQuotas'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits(['close', 'success'])
const { t } = useI18n()
const appStore = useAppStore()

interface QuotaRow {
  key: number
  model: string
  daily_limit_tokens: number | string | null
}

let nextKey = 1
const loading = ref(false)
const submitting = ref(false)
const errorMessage = ref('')
const rows = ref<QuotaRow[]>([])

function toRows(items: DefaultModelTokenQuotaItem[]): QuotaRow[] {
  return items.map(item => ({ ...item, key: nextKey++ }))
}

async function load() {
  loading.value = true
  errorMessage.value = ''
  try {
    const response = await adminAPI.modelTokenQuotas.getDefault()
    rows.value = toRows(response.quotas || [])
  } catch (error: any) {
    errorMessage.value = error?.response?.data?.message || t('admin.users.defaultQuota.loadFailed', 'Failed to load default quotas')
    rows.value = []
  } finally {
    loading.value = false
  }
}

watch(() => props.show, show => { if (show) load() }, { immediate: true })

function addRow() {
  rows.value.push({ key: nextKey++, model: '', daily_limit_tokens: null })
}

function removeRow(index: number) {
  rows.value.splice(index, 1)
}

function buildPayload(): DefaultModelTokenQuotaItem[] | null {
  const seen = new Set<string>()
  const payload: DefaultModelTokenQuotaItem[] = []
  for (const row of rows.value) {
    const model = row.model.trim()
    const rawLimit = row.daily_limit_tokens
    const limit = rawLimit === null || rawLimit === '' ? null : Number(rawLimit)
    if (!model || seen.has(model) || (limit !== null && (!Number.isInteger(limit) || limit < 0))) return null
    seen.add(model)
    payload.push({ model, daily_limit_tokens: limit })
  }
  return payload
}

async function onSave() {
  const payload = buildPayload()
  if (!payload) {
    errorMessage.value = t('admin.users.defaultQuota.invalid', 'Models must be unique and limits must be non-negative integers')
    return
  }
  submitting.value = true
  errorMessage.value = ''
  try {
    await adminAPI.modelTokenQuotas.updateDefault(payload)
    appStore.showSuccess(t('admin.users.defaultQuota.saveSuccess', 'Default quotas updated'))
    emit('success')
  } catch (error: any) {
    errorMessage.value = error?.response?.data?.message || t('admin.users.defaultQuota.updateFailed', 'Failed to save default quotas')
  } finally {
    submitting.value = false
  }
}
</script>

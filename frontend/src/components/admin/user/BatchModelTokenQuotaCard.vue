<template>
  <BaseDialog :show="show" :title="t('admin.users.batchQuota.title', 'Batch manage user model token quotas')" width="wide" @close="$emit('close')">
    <div class="space-y-4">
      <p class="text-sm text-gray-600 dark:text-gray-400">{{ t('admin.users.batchQuota.description') }}</p>
      <div class="space-y-3">
        <div v-for="(row, index) in rows" :key="row.key" class="grid items-end gap-2 rounded-lg border border-gray-200 p-3 dark:border-dark-700 md:grid-cols-[140px_1fr_180px_auto]">
          <label class="block text-xs text-gray-500">
            {{ t('admin.users.batchQuota.action', 'Action') }}
            <select v-model="row.action" data-test="batch-quota-action" class="input mt-1 text-sm">
              <option value="create">{{ t('admin.users.batchQuota.create', 'Create') }}</option>
              <option value="update">{{ t('admin.users.batchQuota.update', 'Update') }}</option>
              <option value="delete">{{ t('admin.users.batchQuota.delete', 'Delete') }}</option>
            </select>
          </label>
          <label class="block text-xs text-gray-500">
            {{ t('admin.users.batchQuota.model', 'Model') }}
            <input v-model="row.model" data-test="batch-quota-model" type="text" class="input mt-1 text-sm" />
          </label>
          <label class="block text-xs text-gray-500">
            {{ t('admin.users.batchQuota.limit', 'Daily token limit') }}
            <input v-model.number="row.daily_limit_tokens" data-test="batch-quota-limit" type="number" min="0" step="1" class="input mt-1 text-sm" :placeholder="t('admin.users.batchQuota.unlimited', 'Unlimited')" :disabled="row.action === 'delete'" />
            <p v-if="row.action === 'delete'" class="mt-1 text-xs text-gray-400">{{ t('admin.users.batchQuota.deleteHint', 'No token limit needed for delete') }}</p>
          </label>
          <button type="button" class="p-2 text-gray-400 hover:text-red-500" :title="t('common.delete')" @click="removeRow(index)"><Icon name="trash" size="sm" /></button>
        </div>
        <button data-test="batch-quota-add" type="button" class="flex items-center gap-1 text-sm text-primary-600" @click="addRow"><Icon name="plus" size="sm" />{{ t('admin.users.batchQuota.addRow', 'Add row') }}</button>
        <p v-if="errorMessage" data-test="batch-quota-error" class="text-sm text-red-500">{{ errorMessage }}</p>
      </div>
    </div>
    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="$emit('close')">{{ t('common.cancel') }}</button>
        <button data-test="batch-quota-submit" type="button" class="btn btn-primary" :disabled="submitting" @click="onSubmit">{{ submitting ? t('admin.users.batchQuota.submitting', 'Executing...') : t('admin.users.batchQuota.submit', 'Execute') }}</button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { BatchModelTokenQuotaOperation } from '@/api/admin/modelTokenQuotas'
import { useAppStore } from '@/stores/app'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'

defineProps<{ show: boolean }>()
const emit = defineEmits(['close', 'success'])
const { t } = useI18n()
const appStore = useAppStore()

interface BatchRow {
  key: number
  action: 'create' | 'update' | 'delete'
  model: string
  daily_limit_tokens: number | string | null
}

let nextKey = 1
const submitting = ref(false)
const errorMessage = ref('')
const rows = ref<BatchRow[]>([{ key: nextKey++, action: 'create', model: '', daily_limit_tokens: null }])

function addRow() {
  rows.value.push({ key: nextKey++, action: 'create', model: '', daily_limit_tokens: null })
}

function removeRow(index: number) {
  if (rows.value.length <= 1) return
  rows.value.splice(index, 1)
}

function buildPayload(): BatchModelTokenQuotaOperation[] | null {
  const seen = new Set<string>()
  const payload: BatchModelTokenQuotaOperation[] = []
  for (const row of rows.value) {
    const model = row.model.trim()
    if (!model || seen.has(model)) return null
    seen.add(model)
    const op: BatchModelTokenQuotaOperation = { action: row.action, model }
    if (row.action !== 'delete') {
      const rawLimit = row.daily_limit_tokens
      const limit = rawLimit === null || rawLimit === '' ? null : Number(rawLimit)
      if (limit !== null && (!Number.isInteger(limit) || limit < 0)) return null
      op.daily_limit_tokens = limit
    }
    payload.push(op)
  }
  return payload
}

async function onSubmit() {
  const payload = buildPayload()
  if (!payload) {
    errorMessage.value = t('admin.users.batchQuota.invalid', 'Models must be unique and non-empty')
    return
  }

  const ok = confirm(t('admin.users.batchQuota.confirmMessage', { ops: payload.length }))
  if (!ok) return

  submitting.value = true
  errorMessage.value = ''
  try {
    const result = await adminAPI.modelTokenQuotas.batchApply(payload)
    appStore.showSuccess(t('admin.users.batchQuota.success', { count: result.affected_users }))
    if (result.errors && result.errors.length > 0) {
      appStore.showError(result.errors.slice(0, 3).join('; '))
    }
    emit('success', result)
  } catch (error: any) {
    errorMessage.value = error?.response?.data?.message || t('admin.users.batchQuota.failed', 'Batch operation failed')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="card">
    <div class="border-b border-gray-100 px-6 py-4 dark:border-dark-700">
      <h2 class="text-lg font-semibold text-gray-900 dark:text-white">{{ t('admin.settings.defaultGroup.title') }}</h2>
      <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.settings.defaultGroup.description') }}</p>
    </div>
    <div class="space-y-3 p-6">
      <label class="input-label">{{ t('admin.settings.defaultGroup.name') }}</label>
      <div class="flex flex-col gap-3 sm:flex-row">
        <input v-model="name" class="input flex-1" maxlength="100" :placeholder="t('admin.settings.defaultGroup.placeholder')" />
        <button type="button" class="btn btn-primary" :disabled="loading || !name.trim()" @click="save">{{ t('common.save') }}</button>
      </div>
      <p v-if="status?.configured" :class="status.exists ? 'text-green-600' : 'text-amber-600'" class="text-sm">
        {{ t(status.exists ? 'admin.settings.defaultGroup.exists' : 'admin.settings.defaultGroup.missing') }}
      </p>
      <p v-else class="text-sm text-gray-500">{{ t('admin.settings.defaultGroup.unconfigured') }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { settingsAPI, type DefaultGroupStatus } from '@/api/admin/settings'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'

const { t } = useI18n()
const appStore = useAppStore()
const name = ref('')
const status = ref<DefaultGroupStatus | null>(null)
const loading = ref(false)

async function load() {
  loading.value = true
  try {
    status.value = await settingsAPI.getDefaultGroup()
    name.value = status.value.name
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loading.value = false
  }
}

async function save() {
  const normalized = name.value.trim()
  if (!normalized) return
  loading.value = true
  try {
    status.value = await settingsAPI.updateDefaultGroup(normalized)
    name.value = status.value.name
    appStore.showSuccess(t('admin.settings.defaultGroup.saved'))
  } catch (error) {
    appStore.showError(extractApiErrorMessage(error, t('common.error')))
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

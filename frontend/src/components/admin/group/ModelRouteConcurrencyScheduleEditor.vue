<template>
  <div v-if="ready" class="mt-3 rounded-md border border-dashed border-gray-300 p-3 dark:border-dark-500" data-test="schedule-editor">
    <div class="mb-2 flex items-center justify-between gap-2">
      <div>
        <p class="text-xs font-medium text-gray-700 dark:text-gray-300">{{ t('admin.groups.modelRouting.scheduleTitle') }}</p>
        <p class="text-[10px] text-gray-500 dark:text-gray-400">{{ t('admin.groups.modelRouting.scheduleHint') }}</p>
      </div>
      <button type="button" class="text-xs text-primary-600" data-test="add-schedule" @click="addRow">+ {{ t('admin.groups.modelRouting.addSchedule') }}</button>
    </div>

    <p v-if="loading" class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.groups.modelRouting.scheduleLoading') }}</p>
    <template v-else>
      <p v-if="!rows.length" class="mb-2 text-xs text-gray-500 dark:text-gray-400" data-test="schedule-default-hint">{{ t('admin.groups.modelRouting.scheduleDefaultHint') }}</p>
      <div v-for="(row, index) in rows" :key="row.id ?? `new-${index}`" class="mb-2 grid items-end gap-2 md:grid-cols-[1fr_1fr_1fr_auto]" data-test="schedule-row">
        <label class="text-xs text-gray-600 dark:text-gray-400">
          {{ t('admin.groups.modelRouting.scheduleStart') }}
          <MinuteTimePicker v-model="row.start" :label="t('admin.groups.modelRouting.scheduleStart')" data-test="schedule-start" />
        </label>
        <label class="text-xs text-gray-600 dark:text-gray-400">
          {{ t('admin.groups.modelRouting.scheduleEnd') }}
          <MinuteTimePicker v-model="row.end" :label="t('admin.groups.modelRouting.scheduleEnd')" :allow-end-of-day="true" data-test="schedule-end" />
        </label>
        <label class="text-xs text-gray-600 dark:text-gray-400">
          {{ t('admin.groups.modelRouting.scheduleMaxConcurrency') }}
          <input v-model.number="row.maxConcurrency" type="number" min="1" step="1" class="input mt-1 text-sm" :placeholder="t('admin.groups.modelRouting.scheduleUnlimited')" data-test="schedule-limit" />
        </label>
        <button type="button" class="mb-1 text-xs text-red-500" data-test="remove-schedule" @click="removeRow(index)">{{ t('admin.groups.modelRouting.removeSchedule') }}</button>
      </div>
      <p v-if="error" class="mb-2 text-xs text-red-500" data-test="schedule-error">{{ error }}</p>
      <p v-if="saved" class="mb-2 text-xs text-green-600" data-test="schedule-saved">{{ t('admin.groups.modelRouting.scheduleSaved') }}</p>
      <button type="button" class="rounded bg-primary-600 px-3 py-1.5 text-xs text-white disabled:opacity-50" data-test="save-schedules" :disabled="saving" @click="save">
        {{ saving ? t('admin.groups.modelRouting.scheduleSaving') : t('admin.groups.modelRouting.saveSchedules') }}
      </button>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import { extractApiErrorMessage } from '@/utils/apiError'
import { parseScheduleMinute, toSchedulePayload, validateScheduleDrafts, type ScheduleDraft } from './modelRouteConcurrencySchedule'
import MinuteTimePicker from './MinuteTimePicker.vue'

const props = defineProps<{ groupId?: number; routeAlias: string; accountId?: number; modelValue?: ScheduleDraft[] }>()
const emit = defineEmits<{ (event: 'update:modelValue', value: ScheduleDraft[]): void }>()
const { t } = useI18n()
const rows = ref<ScheduleDraft[]>([])
const loading = ref(false)
const saving = ref(false)
const saved = ref(false)
const error = ref('')
const draftMode = computed(() => !props.groupId)
const ready = computed(() => Boolean(props.accountId && props.routeAlias.trim()))

const emptyRow = (): ScheduleDraft => ({ start: '00:00', end: '24:00', maxConcurrency: null })

const load = async () => {
  if (!ready.value) {
    rows.value = []
    return
  }
  if (draftMode.value) {
    rows.value = (props.modelValue ?? []).map(row => ({ ...row }))
    loading.value = false
    return
  }
  loading.value = true
  error.value = ''
  saved.value = false
  try {
    const schedules = await adminAPI.groups.listModelRouteConcurrencySchedules(props.groupId!, props.routeAlias.trim(), props.accountId!)
    rows.value = schedules.map(item => ({
      id: item.id,
      start: item.start,
      end: item.end,
      maxConcurrency: item.max_concurrency,
    }))
  } catch (err) {
    error.value = extractApiErrorMessage(err, t('admin.groups.modelRouting.scheduleLoadFailed'))
  } finally {
    loading.value = false
  }
}

const addRow = () => {
  rows.value.push(emptyRow())
  saved.value = false
}
const removeRow = (index: number) => {
  rows.value.splice(index, 1)
  saved.value = false
}
watch(rows, value => {
  if (draftMode.value) emit('update:modelValue', value.map(row => ({ ...row })))
}, { deep: true })
const save = async () => {
  const validation = validateScheduleDrafts(rows.value)
  if (validation) {
    error.value = validation
    saved.value = false
    return
  }
  // Keep the conversion explicit here so a future UI input cannot bypass the minute contract.
  if (rows.value.some(row => parseScheduleMinute(row.start, false) == null || parseScheduleMinute(row.end, true) == null)) return
  if (draftMode.value) {
    saved.value = true
    return
  }
  saving.value = true
  error.value = ''
  saved.value = false
  try {
    await adminAPI.groups.replaceModelRouteConcurrencySchedules(props.groupId!, {
      route_alias: props.routeAlias.trim(),
      account_id: props.accountId!,
      schedules: toSchedulePayload(rows.value)
    })
    saved.value = true
  } catch (err) {
    error.value = extractApiErrorMessage(err, t('admin.groups.modelRouting.scheduleSaveFailed'))
  } finally {
    saving.value = false
  }
}

watch(() => [props.groupId, props.routeAlias, props.accountId], () => { void load() }, { immediate: true })
</script>

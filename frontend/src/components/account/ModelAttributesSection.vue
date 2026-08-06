<template>
  <div class="border-t border-gray-200 pt-4 dark:border-dark-600">
    <div class="mb-3 flex items-center justify-between gap-2">
      <div>
        <label class="input-label mb-0">{{ t('admin.accounts.modelAttributes.title') }}</label>
        <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">
          {{ t('admin.accounts.modelAttributes.hint') }}
        </p>
      </div>
      <button
        type="button"
        data-testid="add-attribute-button"
        @click="addCustom"
        class="flex items-center gap-1 rounded-lg border border-gray-300 px-3 py-1.5 text-xs font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-500 dark:text-gray-300 dark:hover:bg-dark-600"
      >
        <svg class="h-3.5 w-3.5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
          <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4" />
        </svg>
        {{ t('admin.accounts.modelAttributes.add') }}
      </button>
    </div>

    <p v-if="rows.length === 0" class="text-xs text-gray-400 dark:text-gray-500">
      {{ t('admin.accounts.modelAttributes.empty') }}
    </p>

    <div
      v-for="(row, index) in rows"
      :key="row.uid"
      class="mb-2 rounded-lg border p-3"
      :class="
        duplicateKeys.has(row.key.trim())
          ? 'border-red-300 dark:border-red-900'
          : 'border-gray-200 dark:border-dark-600'
      "
      data-testid="attribute-row"
    >
      <div class="flex items-center gap-2">
        <input
          :value="row.key"
          type="text"
          class="input flex-1 font-mono text-xs"
          :placeholder="t('admin.accounts.modelAttributes.namePlaceholder')"
          data-testid="row-key"
          @input="onRowChange(row, 'key', ($event.target as HTMLInputElement).value)"
        />
        <input
          :value="row.description"
          type="text"
          class="input flex-1 text-xs"
          :placeholder="t('admin.accounts.modelAttributes.descriptionPlaceholder')"
          data-testid="row-description"
          @input="onRowChange(row, 'description', ($event.target as HTMLInputElement).value)"
        />
        <input
          :value="row.valueText"
          type="text"
          class="input flex-1 text-xs"
          :placeholder="t('admin.accounts.modelAttributes.valuePlaceholder')"
          data-testid="row-value"
          @input="onRowChange(row, 'valueText', ($event.target as HTMLInputElement).value)"
        />
        <button
          type="button"
          :title="t('admin.accounts.modelAttributes.remove')"
          data-testid="row-remove"
          @click="removeRow(index)"
          class="flex-shrink-0 rounded-lg p-2 text-gray-400 transition-colors hover:bg-red-50 hover:text-red-500 dark:hover:bg-red-900/20"
        >
          <svg class="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path
              stroke-linecap="round"
              stroke-linejoin="round"
              stroke-width="2"
              d="M6 18L18 6M6 6l12 12"
            />
          </svg>
        </button>
      </div>
      <p
        v-if="duplicateKeys.has(row.key.trim())"
        class="mt-1 text-xs text-red-500"
        data-testid="duplicate-hint"
      >
        {{ t('admin.accounts.modelAttributes.duplicateKey', { key: row.key.trim() }) }}
      </p>
    </div>

    <!-- 预置快捷标签：点击直接添加一行预填的属性 -->
    <div class="mt-3">
      <p class="mb-1.5 text-xs text-gray-500 dark:text-gray-400">
        {{ t('admin.accounts.modelAttributes.presetLabel') }}
      </p>
      <div class="flex flex-wrap gap-1.5">
        <button
          v-for="preset in MODEL_ATTRIBUTE_PRESETS"
          :key="preset.key"
          type="button"
          data-testid="preset-chip"
          :data-preset-key="preset.key"
          :title="t(preset.descriptionKey)"
          :disabled="isAdded(preset.key)"
          @click="addPreset(preset)"
          class="rounded-lg border border-gray-200 px-2.5 py-1 font-mono text-xs text-gray-600 transition-colors hover:border-primary-300 hover:text-primary-600 disabled:cursor-not-allowed disabled:opacity-40 dark:border-dark-500 dark:text-gray-400 dark:hover:border-primary-700 dark:hover:text-primary-400"
        >
          {{ preset.key }}
        </button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ModelAttributes } from '@/types'
import { MODEL_ATTRIBUTE_PRESETS, type ModelAttributePreset } from './modelAttributePresets'
import {
  buildModelAttributes,
  rowsFromModelAttributes,
  type ModelAttributeRow
} from './modelAttributeUtils'

const props = defineProps<{
  modelValue?: ModelAttributes | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ModelAttributes]
}>()

const { t } = useI18n()

let uidCounter = 0
const makeUid = () => `ma-${Date.now()}-${uidCounter++}`

const rows = ref<ModelAttributeRow[]>([])
let syncingFromProps = false

// 回显：外部 map 变化 → 重建行。若父组件回写的内容与当前行构建结果一致
// （即本次变化源于本组件自己的 emit），则跳过重建，避免“添加行 → emit →
// 父回写 → watch 重建 → 行被清掉”的循环。
watch(
  () => props.modelValue,
  (value) => {
    if (JSON.stringify(value) === JSON.stringify(buildModelAttributes(rows.value))) {
      return
    }
    syncingFromProps = true
    rows.value = rowsFromModelAttributes(value, makeUid)
    void nextTick(() => {
      syncingFromProps = false
    })
  },
  { immediate: true }
)

function onRowChange(row: ModelAttributeRow, field: 'key' | 'description' | 'valueText', value: string) {
  row[field] = value
  syncEmit()
}

function syncEmit() {
  if (syncingFromProps) {
    return
  }
  emit('update:modelValue', buildModelAttributes(rows.value))
}

function addPreset(preset: ModelAttributePreset) {
  if (isAdded(preset.key)) {
    return
  }
  rows.value.push({
    uid: makeUid(),
    key: preset.key,
    description: t(preset.descriptionKey),
    valueText: ''
  })
  syncEmit()
}

function addCustom() {
  rows.value.push({ uid: makeUid(), key: '', description: '', valueText: '' })
  syncEmit()
}

function removeRow(index: number) {
  rows.value.splice(index, 1)
  syncEmit()
}

const isAdded = (key: string) => rows.value.some((row) => row.key.trim() === key)

// 重复 key（去空白后）集合，仅用于行内提示；构建时保留第一份
const duplicateKeys = computed(() => {
  const seen = new Set<string>()
  const dups = new Set<string>()
  for (const row of rows.value) {
    const key = row.key.trim()
    if (key === '') {
      continue
    }
    if (seen.has(key)) {
      dups.add(key)
    }
    seen.add(key)
  }
  return dups
})
</script>

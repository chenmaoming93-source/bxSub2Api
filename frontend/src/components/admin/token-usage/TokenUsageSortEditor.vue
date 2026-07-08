<template>
  <div class="min-w-[20rem] space-y-2">
    <label class="block text-xs font-medium text-gray-500 dark:text-gray-400">{{ t('tokenUsageReport.sortRules') }}</label>
    <div v-for="(rule, index) in modelValue" :key="index" class="flex items-center gap-2">
      <span class="w-5 shrink-0 text-center text-xs text-gray-400">{{ index + 1 }}</span>
      <select :value="rule.field" class="input min-w-0 flex-1 text-sm" @change="updateField(index, $event)">
        <option v-for="option in availableOptions(index)" :key="option.value" :value="option.value">{{ option.label }}</option>
      </select>
      <select :value="rule.order" class="input w-24 shrink-0 text-sm" @change="updateOrder(index, $event)">
        <option value="asc">{{ t('tokenUsageReport.ascending') }}</option>
        <option value="desc">{{ t('tokenUsageReport.descending') }}</option>
      </select>
      <button
        type="button"
        class="btn btn-ghost btn-sm shrink-0"
        :disabled="modelValue.length === 1"
        :title="t('tokenUsageReport.removeSort')"
        @click="removeRule(index)"
      >
        ×
      </button>
    </div>
    <button
      v-if="modelValue.length < options.length"
      type="button"
      class="text-xs text-primary-600 hover:text-primary-700 dark:text-primary-400"
      @click="addRule"
    >
      + {{ t('tokenUsageReport.addSort') }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { TokenUsageSortField, TokenUsageSortRule } from '@/api/admin/tokenUsage'

const props = defineProps<{
  modelValue: TokenUsageSortRule[]
  options: Array<{ value: TokenUsageSortField; label: string }>
}>()
const emit = defineEmits<{ (event: 'update:modelValue', value: TokenUsageSortRule[]): void }>()
const { t } = useI18n()

function availableOptions(index: number) {
  const selected = new Set(props.modelValue.filter((_, i) => i !== index).map(rule => rule.field))
  return props.options.filter(option => !selected.has(option.value))
}

function updateField(index: number, event: Event) {
  const rules = props.modelValue.map(rule => ({ ...rule }))
  rules[index].field = (event.target as HTMLSelectElement).value as TokenUsageSortField
  emit('update:modelValue', rules)
}

function updateOrder(index: number, event: Event) {
  const rules = props.modelValue.map(rule => ({ ...rule }))
  rules[index].order = (event.target as HTMLSelectElement).value as 'asc' | 'desc'
  emit('update:modelValue', rules)
}

function addRule() {
  const selected = new Set(props.modelValue.map(rule => rule.field))
  const next = props.options.find(option => !selected.has(option.value))
  if (next) emit('update:modelValue', [...props.modelValue, { field: next.value, order: 'asc' }])
}

function removeRule(index: number) {
  if (props.modelValue.length > 1) emit('update:modelValue', props.modelValue.filter((_, i) => i !== index))
}
</script>

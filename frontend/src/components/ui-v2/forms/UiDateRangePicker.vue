<script setup lang="ts">
import UiDatePicker from './UiDatePicker.vue'

withDefaults(defineProps<{
  startDate?: string
  endDate?: string
  disabled?: boolean
  invalid?: boolean
}>(), {
  startDate: '',
  endDate: '',
  disabled: false,
  invalid: false
})

const emit = defineEmits<{
  'update:startDate': [value: string]
  'update:endDate': [value: string]
  change: [range: { startDate: string; endDate: string }]
}>()

function updateStart(value: string) {
  emit('update:startDate', value)
  emit('change', { startDate: value, endDate: '' })
}

function updateEnd(value: string) {
  emit('update:endDate', value)
  emit('change', { startDate: '', endDate: value })
}
</script>

<template>
  <div class="ui-v2-date-range-picker" :class="{ 'ui-v2-date-range-picker--invalid': invalid }">
    <UiDatePicker :model-value="startDate" :disabled="disabled" @update:model-value="updateStart" />
    <span class="ui-v2-date-range-picker__separator">至</span>
    <UiDatePicker :model-value="endDate" :disabled="disabled" @update:model-value="updateEnd" />
  </div>
</template>

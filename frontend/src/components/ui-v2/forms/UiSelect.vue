<script setup lang="ts">
export interface UiSelectOption {
  label: string
  value: string | number
  disabled?: boolean
}

withDefaults(defineProps<{
  modelValue?: string | number
  options?: UiSelectOption[]
  placeholder?: string
  disabled?: boolean
  invalid?: boolean
}>(), {
  modelValue: '',
  options: () => [],
  placeholder: '请选择',
  disabled: false,
  invalid: false
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  change: [value: string]
}>()
</script>

<template>
  <select
    :value="modelValue"
    :disabled="disabled"
    class="ui-v2-select"
    :class="{ 'ui-v2-input--invalid': invalid }"
    @change="emit('update:modelValue', ($event.target as HTMLSelectElement).value); emit('change', ($event.target as HTMLSelectElement).value)"
  >
    <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
    <option v-for="option in options" :key="String(option.value)" :value="option.value" :disabled="option.disabled">
      {{ option.label }}
    </option>
  </select>
</template>

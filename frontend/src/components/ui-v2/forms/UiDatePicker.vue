<script setup lang="ts">
withDefaults(defineProps<{
  modelValue?: string
  min?: string
  max?: string
  disabled?: boolean
  invalid?: boolean
}>(), {
  modelValue: '',
  min: undefined,
  max: undefined,
  disabled: false,
  invalid: false
})

const emit = defineEmits<{ 'update:modelValue': [value: string]; change: [value: string] }>()
</script>

<template>
  <input
    type="date"
    :value="modelValue"
    :min="min"
    :max="max"
    :disabled="disabled"
    class="ui-v2-input ui-v2-date-picker"
    :class="{ 'ui-v2-input--invalid': invalid }"
    @input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
    @change="emit('change', ($event.target as HTMLInputElement).value)"
  >
</template>

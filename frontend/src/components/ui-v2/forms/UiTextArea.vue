<script setup lang="ts">
withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  rows?: number
  disabled?: boolean
  readonly?: boolean
  invalid?: boolean
}>(), {
  modelValue: '',
  placeholder: '',
  rows: 4,
  disabled: false,
  readonly: false,
  invalid: false
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<template>
  <textarea
    :value="modelValue"
    :rows="rows"
    :placeholder="placeholder"
    :disabled="disabled"
    :readonly="readonly"
    class="ui-v2-textarea"
    :class="{ 'ui-v2-input--invalid': invalid }"
    @input="emit('update:modelValue', ($event.target as HTMLTextAreaElement).value)"
  />
</template>

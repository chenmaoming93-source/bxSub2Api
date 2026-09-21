<script setup lang="ts">
import UiInput from './UiInput.vue'

withDefaults(defineProps<{
  modelValue?: string
  placeholder?: string
  disabled?: boolean
}>(), {
  modelValue: '',
  placeholder: '搜索…',
  disabled: false
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  search: [value: string]
}>()
</script>

<template>
  <label class="ui-v2-search-input">
    <span class="ui-v2-search-input__icon" aria-hidden="true">⌕</span>
    <UiInput
      :model-value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      @update:model-value="emit('update:modelValue', $event)"
      @keydown.enter="emit('search', modelValue)"
    />
  </label>
</template>

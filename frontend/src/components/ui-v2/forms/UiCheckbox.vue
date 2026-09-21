<script setup lang="ts">
withDefaults(defineProps<{
  modelValue?: boolean
  label?: string
  disabled?: boolean
}>(), {
  modelValue: false,
  label: '',
  disabled: false
})

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>

<template>
  <label class="ui-v2-check-control" :class="{ 'ui-v2-check-control--disabled': disabled }">
    <input
      type="checkbox"
      :checked="modelValue"
      :disabled="disabled"
      @change="emit('update:modelValue', ($event.target as HTMLInputElement).checked)"
    >
    <span class="ui-v2-check-control__box" aria-hidden="true">✓</span>
    <span v-if="label">{{ label }}</span>
    <slot />
  </label>
</template>

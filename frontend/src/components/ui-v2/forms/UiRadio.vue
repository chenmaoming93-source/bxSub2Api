<script setup lang="ts">
withDefaults(defineProps<{
  modelValue?: string | number
  value: string | number
  label?: string
  name?: string
  disabled?: boolean
}>(), {
  modelValue: '',
  label: '',
  name: undefined,
  disabled: false
})

const emit = defineEmits<{ 'update:modelValue': [value: string | number] }>()
</script>

<template>
  <label class="ui-v2-check-control" :class="{ 'ui-v2-check-control--disabled': disabled }">
    <input
      type="radio"
      :name="name"
      :value="value"
      :checked="modelValue === value"
      :disabled="disabled"
      @change="emit('update:modelValue', value)"
    >
    <span class="ui-v2-check-control__radio" aria-hidden="true" />
    <span v-if="label">{{ label }}</span>
    <slot />
  </label>
</template>

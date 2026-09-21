<script setup lang="ts">
withDefaults(defineProps<{
  modelValue?: boolean
  disabled?: boolean
  label?: string
}>(), {
  modelValue: false,
  disabled: false,
  label: ''
})

const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>

<template>
  <label class="ui-v2-switch-control" :class="{ 'ui-v2-switch-control--disabled': disabled }">
    <button
      type="button"
      role="switch"
      :aria-checked="modelValue"
      :disabled="disabled"
      class="ui-v2-switch"
      :class="{ 'ui-v2-switch--on': modelValue }"
      @click="emit('update:modelValue', !modelValue)"
    >
      <span class="ui-v2-switch__thumb" />
    </button>
    <span v-if="label">{{ label }}</span>
    <slot />
  </label>
</template>

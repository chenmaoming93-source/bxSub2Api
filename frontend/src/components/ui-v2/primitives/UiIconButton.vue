<script setup lang="ts">
withDefaults(defineProps<{
  label: string
  size?: 'sm' | 'md' | 'lg'
  variant?: 'default' | 'primary' | 'ghost'
  disabled?: boolean
}>(), {
  size: 'md',
  variant: 'default',
  disabled: false
})
</script>

<template>
  <button
    type="button"
    class="ui-v2-icon-button"
    :class="[`ui-v2-icon-button--${size}`, `ui-v2-icon-button--${variant}`]"
    :aria-label="label"
    :title="label"
    :disabled="disabled"
  >
    <slot />
  </button>
</template>

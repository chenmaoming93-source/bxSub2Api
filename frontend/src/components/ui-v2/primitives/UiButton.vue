<script setup lang="ts">
withDefaults(defineProps<{
  variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
  size?: 'sm' | 'md' | 'lg'
  loading?: boolean
  block?: boolean
  type?: 'button' | 'submit' | 'reset'
}>(), {
  variant: 'primary',
  size: 'md',
  loading: false,
  block: false,
  type: 'button'
})
</script>

<template>
  <button
    :type="type"
    class="ui-v2-button"
    :class="[`ui-v2-button--${variant}`, `ui-v2-button--${size}`, { 'ui-v2-button--block': block }]"
    :disabled="loading"
  >
    <span v-if="loading" class="ui-v2-spinner ui-v2-spinner--small" aria-hidden="true" />
    <slot />
  </button>
</template>

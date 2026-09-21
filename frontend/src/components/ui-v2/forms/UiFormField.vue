<script setup lang="ts">
defineProps<{
  label?: string
  hint?: string
  error?: string
  required?: boolean
}>()
</script>

<template>
  <label class="ui-v2-form-field">
    <span v-if="label" class="ui-v2-form-field__label">
      {{ label }}<span v-if="required" class="ui-v2-form-field__required"> *</span>
    </span>
    <slot />
    <span v-if="error" class="ui-v2-form-field__error">{{ error }}</span>
    <span v-else-if="hint" class="ui-v2-form-field__hint">{{ hint }}</span>
  </label>
</template>

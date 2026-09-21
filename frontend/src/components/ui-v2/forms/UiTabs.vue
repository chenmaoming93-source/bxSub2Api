<script setup lang="ts">
export interface UiTab {
  label: string
  value: string
  disabled?: boolean
}

withDefaults(defineProps<{
  modelValue?: string
  tabs?: UiTab[]
}>(), {
  modelValue: '',
  tabs: () => []
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<template>
  <div class="ui-v2-tabs" role="tablist">
    <button
      v-for="tab in tabs"
      :key="tab.value"
      type="button"
      role="tab"
      :aria-selected="modelValue === tab.value"
      :disabled="tab.disabled"
      class="ui-v2-tab"
      :class="{ 'ui-v2-tab--active': modelValue === tab.value }"
      @click="emit('update:modelValue', tab.value)"
    >
      {{ tab.label }}
    </button>
  </div>
</template>

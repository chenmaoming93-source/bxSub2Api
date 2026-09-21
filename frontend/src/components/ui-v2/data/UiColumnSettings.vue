<script setup lang="ts">
export interface UiColumnOption { key: string; label: string }
const props = withDefaults(defineProps<{ modelValue?: string[]; columns?: UiColumnOption[] }>(), { modelValue: () => [], columns: () => [] })
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()
function toggle(key: string, checked: boolean) {
  const current = new Set(props.modelValue)
  if (checked) current.add(key)
  else current.delete(key)
  emit('update:modelValue', [...current])
}
</script>

<template>
  <details class="ui-v2-column-settings"><summary>列设置</summary><label v-for="column in columns" :key="column.key"><input type="checkbox" :checked="modelValue?.includes(column.key)" @change="toggle(column.key, ($event.target as HTMLInputElement).checked)">{{ column.label }}</label></details>
</template>

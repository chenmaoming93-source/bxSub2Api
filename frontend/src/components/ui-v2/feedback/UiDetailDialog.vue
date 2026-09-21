<script setup lang="ts">
import UiDialog from './UiDialog.vue'

withDefaults(defineProps<{
  modelValue?: boolean
  title?: string
  width?: 'narrow' | 'normal' | 'wide' | 'extra-wide' | 'full'
  loading?: boolean
}>(), { modelValue: false, title: '详情', width: 'extra-wide', loading: false })

const emit = defineEmits<{ 'update:modelValue': [value: boolean]; close: [] }>()
</script>

<template>
  <UiDialog :model-value="modelValue" :title="title" :width="width" @update:model-value="emit('update:modelValue', $event)" @close="emit('close')">
    <div v-if="loading" class="ui-v2-detail-loading">加载中…</div>
    <slot v-else />
    <template v-if="$slots.footer" #footer><slot name="footer" /></template>
  </UiDialog>
</template>

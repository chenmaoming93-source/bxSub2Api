<script setup lang="ts">
import UiButton from '../primitives/UiButton.vue'
import UiDialog from './UiDialog.vue'

withDefaults(defineProps<{ modelValue?: boolean; title?: string; message?: string; loading?: boolean; confirmText?: string; cancelText?: string }>(), {
  modelValue: false, title: '请确认', message: '', loading: false, confirmText: '确认', cancelText: '取消'
})
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; confirm: []; cancel: [] }>()
function cancel() { emit('update:modelValue', false); emit('cancel') }
</script>

<template>
  <UiDialog :model-value="modelValue" :title="title" width="narrow" @update:model-value="emit('update:modelValue', $event)">
    <p class="ui-v2-confirm-message">{{ message }}<slot /></p>
    <template #footer>
      <div class="ui-v2-dialog-actions">
        <UiButton variant="secondary" @click="cancel">{{ cancelText }}</UiButton>
        <UiButton :loading="loading" @click="emit('confirm')">{{ confirmText }}</UiButton>
      </div>
    </template>
  </UiDialog>
</template>

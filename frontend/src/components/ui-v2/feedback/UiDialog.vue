<script setup lang="ts">
import { nextTick, onUnmounted, ref, watch } from 'vue'

type DialogWidth = 'narrow' | 'normal' | 'wide' | 'extra-wide' | 'full'

const props = withDefaults(defineProps<{
  modelValue?: boolean
  show?: boolean
  title?: string
  width?: DialogWidth
  closeOnEscape?: boolean
  closeOnClickOutside?: boolean
  showCloseButton?: boolean
}>(), {
  modelValue: false,
  show: undefined,
  title: '',
  width: 'normal',
  closeOnEscape: true,
  closeOnClickOutside: false,
  showCloseButton: true
})

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  close: []
}>()

const dialogRef = ref<HTMLElement | null>(null)
let previousActiveElement: HTMLElement | null = null

const isOpen = () => props.show === undefined ? props.modelValue : props.show
function close() {
  emit('update:modelValue', false)
  emit('close')
}
function handleKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape' && isOpen() && props.closeOnEscape) close()
}
function handleBackdrop() {
  if (props.closeOnClickOutside) close()
}
watch(isOpen, async (open) => {
  if (open) {
    previousActiveElement = document.activeElement instanceof HTMLElement ? document.activeElement : null
    document.body.classList.add('ui-v2-dialog-open')
    await nextTick()
    dialogRef.value?.querySelector<HTMLElement>('button, input, select, textarea, [tabindex]:not([tabindex="-1"])')?.focus()
  } else {
    document.body.classList.remove('ui-v2-dialog-open')
    previousActiveElement?.focus()
    previousActiveElement = null
  }
}, { immediate: true })

window.addEventListener('keydown', handleKeydown)
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  document.body.classList.remove('ui-v2-dialog-open')
})
</script>

<template>
  <Teleport to="body">
    <Transition name="ui-v2-dialog">
      <div v-if="isOpen()" class="ui-v2-dialog-backdrop" @click.self="handleBackdrop">
        <section ref="dialogRef" class="ui-v2-dialog" :class="`ui-v2-dialog--${width}`" role="dialog" aria-modal="true" :aria-label="title">
          <header class="ui-v2-dialog__header">
            <h2 class="ui-v2-dialog__title">{{ title }}</h2>
            <button v-if="showCloseButton" type="button" class="ui-v2-dialog__close" aria-label="关闭" @click="close">×</button>
          </header>
          <div class="ui-v2-dialog__body"><slot /></div>
          <footer v-if="$slots.footer" class="ui-v2-dialog__footer"><slot name="footer" /></footer>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

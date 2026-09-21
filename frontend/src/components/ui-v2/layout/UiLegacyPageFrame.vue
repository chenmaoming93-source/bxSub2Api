<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import UiPageContainer from './UiPageContainer.vue'
import UiPageHeader from './UiPageHeader.vue'

withDefaults(defineProps<{
  title?: string
  description?: string
  showHeader?: boolean
}>(), {
  title: '',
  description: '',
  showHeader: true
})
</script>

<template>
  <AppLayout>
    <UiPageContainer class="ui-v2-legacy-frame">
      <UiPageHeader v-if="showHeader" :title="title" :description="description">
        <slot name="actions" />
      </UiPageHeader>
      <div class="ui-v2-legacy-frame__content">
        <slot />
      </div>
    </UiPageContainer>
  </AppLayout>
</template>

<style scoped>
.ui-v2-legacy-frame {
  min-width: 0;
  color: var(--ui-v2-text);
}

.ui-v2-legacy-frame__content {
  min-width: 0;
}

.ui-v2-legacy-frame__content :deep(.ui-v2-embedded-page .table-page-layout) {
  height: calc(100vh - 64px - 4rem - 4.5rem);
}

.ui-v2-legacy-frame__content :deep(.card) {
  border-color: var(--ui-v2-border);
  border-radius: 12px;
  box-shadow: var(--ui-v2-shadow);
}

.ui-v2-legacy-frame__content :deep(.btn) {
  border-radius: var(--ui-v2-radius);
}

.ui-v2-legacy-frame__content :deep(.input) {
  border-color: var(--ui-v2-border);
  border-radius: var(--ui-v2-radius);
}
</style>

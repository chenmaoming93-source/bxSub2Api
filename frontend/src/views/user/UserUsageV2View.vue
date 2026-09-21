<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import UiPageHeader from '@/components/ui-v2/layout/UiPageHeader.vue'
import UsageView from './UsageView.vue'
</script>

<template>
  <AppLayout>
    <div class="user-usage-v2">
      <UiPageHeader title="个人使用记录" description="查询当前用户的模型调用明细和错误请求" />
      <UsageView compact embedded />
    </div>
  </AppLayout>
</template>

<style scoped>
.user-usage-v2 { min-width: 0; color: var(--ui-v2-text); }
.user-usage-v2 :deep(.table-page-layout) { height: calc(100vh - 64px - 9rem); gap: 1rem; }
.user-usage-v2 :deep(.table-scroll-container), .user-usage-v2 :deep(.card) { border-color: var(--ui-v2-border); border-radius: 12px; }
</style>

<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import UiPageHeader from '@/components/ui-v2/layout/UiPageHeader.vue'
import UsageView from './UsageView.vue'
</script>

<template>
  <AppLayout>
    <div class="admin-usage-v2">
      <UiPageHeader title="使用记录" description="查询模型调用明细与错误请求" />
      <UsageView compact embedded />
    </div>
  </AppLayout>
</template>

<style scoped>
.admin-usage-v2 { min-width: 0; color: var(--ui-v2-text); }
.admin-usage-v2 :deep(.space-y-6) { gap: 1rem; }
.admin-usage-v2 :deep(.card) { border-color: var(--ui-v2-border); border-radius: 12px; }
.admin-usage-v2 :deep(.usage-filter-dropdown), .admin-usage-v2 :deep(.input-label) { color: var(--ui-v2-text); }
</style>

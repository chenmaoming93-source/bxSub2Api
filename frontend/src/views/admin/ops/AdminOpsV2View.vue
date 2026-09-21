<script setup lang="ts">
import AppLayout from '@/components/layout/AppLayout.vue'
import UiPageHeader from '@/components/ui-v2/layout/UiPageHeader.vue'
import OpsDashboard from './OpsDashboard.vue'
</script>

<template>
  <AppLayout>
    <div class="admin-ops-v2">
      <UiPageHeader title="运维监控" description="实时查看吞吐、延迟、错误和系统日志" />
      <OpsDashboard embedded />
    </div>
  </AppLayout>
</template>

<style scoped>
.admin-ops-v2 { min-width: 0; color: var(--ui-v2-text); }
.admin-ops-v2 :deep(.space-y-6) { gap: 1rem; }
.admin-ops-v2 :deep(.card) { border-color: var(--ui-v2-border); border-radius: 12px; }
</style>

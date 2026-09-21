<script setup lang="ts">
import UiLegacyPageFrame from '@/components/ui-v2/layout/UiLegacyPageFrame.vue'
import AccountsView from './AccountsView.vue'
</script>

<template>
  <UiLegacyPageFrame title="账号管理" description="管理上游模型账号、状态、配额与批量数据操作">
    <AccountsView embedded />
  </UiLegacyPageFrame>
</template>

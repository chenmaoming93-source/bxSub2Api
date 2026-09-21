<script setup lang="ts">
import UiLegacyPageFrame from '@/components/ui-v2/layout/UiLegacyPageFrame.vue'
import GroupsView from './GroupsView.vue'
</script>

<template>
  <UiLegacyPageFrame title="分组管理" description="配置业务分组、默认路由、模型范围与调度策略">
    <GroupsView embedded />
  </UiLegacyPageFrame>
</template>

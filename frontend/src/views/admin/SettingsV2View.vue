<script setup lang="ts">
import UiLegacyPageFrame from '@/components/ui-v2/layout/UiLegacyPageFrame.vue'
import SettingsView from './SettingsView.vue'
</script>

<template>
  <UiLegacyPageFrame title="系统设置" description="集中管理安全、登录、计费、通知及平台运行配置">
    <SettingsView embedded />
  </UiLegacyPageFrame>
</template>

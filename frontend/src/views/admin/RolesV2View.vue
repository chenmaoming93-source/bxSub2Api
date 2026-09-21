<script setup lang="ts">
import UiLegacyPageFrame from '@/components/ui-v2/layout/UiLegacyPageFrame.vue'
import RolesView from './RolesView.vue'
</script>

<template>
  <UiLegacyPageFrame :show-header="false">
    <RolesView embedded />
  </UiLegacyPageFrame>
</template>

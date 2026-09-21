<script setup lang="ts">
import UiLegacyPageFrame from '@/components/ui-v2/layout/UiLegacyPageFrame.vue'
import TokenStatisticsView from './TokenStatisticsView.vue'
</script>

<template>
  <UiLegacyPageFrame :show-header="false">
    <TokenStatisticsView embedded />
  </UiLegacyPageFrame>
</template>

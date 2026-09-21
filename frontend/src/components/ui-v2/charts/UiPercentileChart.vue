<script setup lang="ts">
withDefaults(defineProps<{ height?: number }>(), { height: 240 })
</script>

<template><div class="ui-v2-chart-placeholder" :style="{ minHeight: `${height}px` }"><slot><span>百分位图</span></slot></div></template>

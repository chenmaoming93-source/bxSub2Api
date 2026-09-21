<script setup lang="ts">
withDefaults(defineProps<{ height?: number }>(), { height: 240 })
</script>

<template><div class="ui-v2-chart-placeholder" :style="{ minHeight: `${height}px` }"><slot><span>折线图</span></slot></div></template>

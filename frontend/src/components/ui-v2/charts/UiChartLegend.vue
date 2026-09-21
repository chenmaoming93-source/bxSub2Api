<script setup lang="ts">
export interface UiLegendItem { label: string; color: string }
withDefaults(defineProps<{ items?: UiLegendItem[] }>(), { items: () => [] })
</script>

<template><div class="ui-v2-chart-legend"><span v-for="item in items" :key="item.label"><i :style="{ backgroundColor: item.color }" />{{ item.label }}</span></div></template>

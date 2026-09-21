<script setup lang="ts">
withDefaults(defineProps<{ label: string; value: string | number; hint?: string; tone?: 'primary' | 'success' | 'warning' | 'danger' }>(), { hint: '', tone: 'primary' })
</script>

<template><section class="ui-v2-stat-card" :class="`ui-v2-stat-card--${tone}`"><div class="ui-v2-stat-card__accent" /><div class="ui-v2-stat-card__content"><span class="ui-v2-stat-card__label">{{ label }}</span><strong>{{ value }}</strong><small v-if="hint">{{ hint }}</small></div></section></template>
